<template>
  <div :class="['topology-page', props.embedded ? 'topology-page--embedded' : '']">
    <section class="topology-stage">
      <section class="topology-control-card">
        <div class="topology-console-head">
          <div class="topology-console-head__main">
            <div class="topology-control-head__title">关系分析</div>
            <div class="topology-toolbar__meta topology-toolbar__meta--compact">
              <el-tag type="info" effect="light">{{ isNamespaceScope ? '命名空间视图' : '单资源视图' }}</el-tag>
              <el-tag type="info" effect="light">{{ activeModeLabel }}</el-tag>
              <el-tag effect="plain">{{ graphViewModeLabel }}</el-tag>
              <span class="topology-toolbar__summary">节点 {{ visibleGraphMeta.nodes }} / 边 {{ visibleGraphMeta.edges }}</span>
              <span class="topology-toolbar__summary">异常 {{ anomalyCount }}</span>
              <span class="topology-toolbar__summary">{{ activeClusterName }}</span>
              <span v-if="requiresNamespace" class="topology-toolbar__summary">{{ namespace || '-' }}</span>
            </div>
          </div>
          <div class="topology-console-head__actions">
            <el-button size="small" :icon="Document" @click="syntaxVisible = true">Mermaid</el-button>
            <el-button size="small" :icon="RefreshRight" :loading="loading" @click="loadTopology">刷新</el-button>
            <el-button size="small" @click="resetQuery">重置</el-button>
            <el-button size="small" type="primary" :loading="loading" @click="loadTopology">生成关系图</el-button>
          </div>
        </div>

        <div class="topology-filter-combo">
          <div class="topology-filter-combo__selects">
            <el-select v-if="!props.fixedClusterId" v-model="clusterId" placeholder="集群" class="topology-combo-select" filterable>
              <el-option v-for="item in clusters" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
            <el-select v-model="mode" placeholder="模板" class="topology-combo-select">
              <el-option v-for="item in modeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select v-if="requiresNamespace" v-model="namespace" placeholder="命名空间" class="topology-combo-select" filterable clearable>
              <el-option v-for="item in namespaces" :key="item" :label="item" :value="item" />
            </el-select>
            <el-select v-if="supportsNamespaceScope" v-model="scope" placeholder="视角" class="topology-combo-select">
              <el-option label="单资源" value="resource" />
              <el-option label="命名空间" value="namespace" />
            </el-select>
            <el-select
              v-if="requiresResourceSelection"
              v-model="resourceName"
              :placeholder="resourceSelectPlaceholder"
              class="topology-combo-select topology-combo-select--resource"
              filterable
              clearable
              default-first-option
              :loading="resourceOptionsLoading"
              no-data-text="暂无可选资源"
            >
              <el-option v-for="item in resourceOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select v-model="graphViewMode" placeholder="图谱模式" class="topology-combo-select">
              <el-option v-for="item in graphViewModeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </div>
          <div class="topology-filter-combo__prefs">
            <div class="topology-toggle-item">
              <span class="topology-toggle-item__text">关键资源</span>
              <el-switch v-model="onlyKeyResources" inline-prompt active-text="开" inactive-text="关" />
            </div>
            <div class="topology-toggle-item">
              <span class="topology-toggle-item__text">小地图</span>
              <el-switch v-model="minimapVisible" inline-prompt active-text="开" inactive-text="关" />
            </div>
            <el-radio-group v-model="layoutDensity" size="small">
              <el-radio-button v-for="item in layoutDensityOptions" :key="item.value" :label="item.value">{{ item.label }}</el-radio-button>
            </el-radio-group>
          </div>
        </div>
      </section>

      <section ref="canvasPanelRef" :style="canvasPanelStyle" class="topology-panel topology-panel--canvas topology-panel--main">
        <div class="topology-overlay-toolbar">
          <el-button size="small" :disabled="visibleGraphMeta.nodes === 0" @click="zoomOut">缩小</el-button>
          <el-button size="small" :disabled="visibleGraphMeta.nodes === 0" @click="fitView">适应视图</el-button>
          <el-button size="small" :disabled="visibleGraphMeta.nodes === 0" @click="zoomIn">放大</el-button>
          <el-button size="small" :disabled="visibleGraphMeta.nodes === 0" @click="relayoutGraph">一键美化</el-button>
          <el-button size="small" :icon="Download" :disabled="visibleGraphMeta.nodes === 0" @click="exportPng">导出 PNG</el-button>
          <el-button size="small" :icon="FullScreen" :disabled="visibleGraphMeta.nodes === 0" @click="toggleFullscreen">全屏</el-button>
        </div>

        <EmptyState v-if="!loading && visibleGraphMeta.nodes === 0" type="no-data" description="当前没有可展示的关系数据；可直接生成命名空间视图，或切换单资源后搜索并选择资源" />
        <TopologyFlowCanvas
          v-else
          ref="graphCanvasRef"
          :graph="visibleGraph"
          :dark="isDark()"
          :view-mode="graphViewMode"
          :minimap-visible="minimapVisible"
          @open-node="handleTopologyNodeOpen"
          @positions-change="onGraphPositionsChange"
        />
      </section>
    </section>

    <el-dialog v-model="syntaxVisible" width="58%" title="Mermaid 视图" append-to-body>
      <div class="topology-syntax-actions">
        <el-button :icon="CopyDocument" :disabled="visibleGraphMeta.nodes === 0" @click="copyMermaid">复制</el-button>
      </div>
      <CodeMirrorViewer :text="mermaidPreviewText" language="text" height="78vh" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { CopyDocument, Document, Download, FullScreen, RefreshRight } from '@element-plus/icons-vue'
import * as clustersApi from '@/features/clusters/api/clusters'
import * as k8sApi from '@/features/k8s/api/k8s'
import type TopologyFlowCanvasComponent from '@/features/k8s/components/topology/TopologyFlowCanvas.vue'
import TopologyFlowCanvas from '@/features/k8s/components/topology/TopologyFlowCanvas.vue'
import { applyTopologyAutoLayout } from '@/features/k8s/components/topology/topologyLayout'
import type { TopologyLayoutDensity, TopologyLayoutStrategy } from '@/features/k8s/components/topology/topologyLayout'
import { toPng } from 'html-to-image'
import EmptyState from '@/shared/components/EmptyState.vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import { matchLabels, podUsesPvc } from '@/features/k8s/pages/ClusterManageView.utils'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'

type TopologyMode = 'service' | 'pvc' | 'pod' | 'node' | 'ingress' | 'config' | 'pv' | 'serviceaccount' | 'deployment' | 'node-storage' | 'networkpolicy' | 'namespace'
type TopologyScope = 'resource' | 'namespace'
type GraphViewMode = 'architecture' | 'analysis' | 'anomaly'
type GraphMeta = { nodes: number; edges: number }
type TopologySeverity = 'normal' | 'warning' | 'error'
type TopologyEmphasis = 'core' | 'primary' | 'secondary'
type TopologyGroup = 'core' | 'control' | 'identity' | 'network' | 'storage' | 'config' | 'runtime'
type TopologyNodeData = {
  badge: string
  title: string
  description: string
  refText: string
  kind: string
  targetKind?: string
  targetNamespace?: string
  targetName?: string
  group?: TopologyGroup
  severity?: TopologySeverity
  statusText?: string
  emphasis?: TopologyEmphasis
  cardWidth?: number
  cardHeight?: number
  tooltip?: string
}
type TopologyNode = {
  id: string
  position: { x: number; y: number }
  data: TopologyNodeData
}
type TopologyEdge = { id: string; source: string; target: string; label?: string; relation?: string }
type TopologyLane = { key: string; label: string; x: number; y: number; width: number; height: number }
type TopologyGraph = { nodes: TopologyNode[]; edges: TopologyEdge[]; mermaid: string; lanes?: TopologyLane[] }
type ResourceOption = { label: string; value: string }
type ConfigResourceKind = 'ConfigMap' | 'Secret'

const props = withDefaults(defineProps<{ fixedClusterId?: number; embedded?: boolean }>(), {
  fixedClusterId: undefined,
  embedded: false
})

const route = useRoute()
const router = useRouter()

const graphCanvasRef = ref<InstanceType<typeof TopologyFlowCanvasComponent> | null>(null)
const canvasPanelRef = ref<HTMLDivElement | null>(null)

function defaultScopeForMode(mode: TopologyMode): TopologyScope {
  return mode === 'namespace' ? 'namespace' : 'resource'
}
const loading = ref(false)
const clusters = ref<clustersApi.ClusterItem[]>([])
const namespaces = ref<string[]>([])
const clusterId = ref<number | undefined>(undefined)
const mode = ref<TopologyMode>('service')
const scope = ref<TopologyScope>(defaultScopeForMode('service'))
const graphViewMode = ref<GraphViewMode>('analysis')
const layoutDensity = ref<TopologyLayoutDensity>('balanced')
const minimapVisible = ref(true)
const namespace = ref('default')
const resourceName = ref('')
const resourceOptions = ref<ResourceOption[]>([])
const resourceOptionsLoading = ref(false)
const currentGraph = ref<TopologyGraph>({ nodes: [], edges: [], mermaid: '' })
const syntaxVisible = ref(false)
watch(syntaxVisible, (val) => {
  if (val) {
    const { nodes, edges } = visibleGraph.value
    mermaidPreviewText.value = buildMermaid(nodes, edges) || 'graph LR\n  A[等待生成关系图]'
  }
})
const onlyKeyResources = ref(false)
let resourceOptionsRequestId = 0
const resourceOptionsCache = new Map<string, ResourceOption[]>()

const modeOptions = [
  { label: 'Service', value: 'service' },
  { label: 'PVC', value: 'pvc' },
  { label: 'Pod', value: 'pod' },
  { label: 'Node', value: 'node' },
  { label: 'Ingress', value: 'ingress' },
  { label: 'Config', value: 'config' },
  { label: 'PV', value: 'pv' },
  { label: 'ServiceAccount', value: 'serviceaccount' },
  { label: 'Deployment', value: 'deployment' },
  { label: 'Node-Storage', value: 'node-storage' },
  { label: 'NetworkPolicy', value: 'networkpolicy' },
  { label: 'Namespace', value: 'namespace' }
]

const graphViewModeOptions = [
  { label: '架构视图', value: 'architecture' },
  { label: '分析视图', value: 'analysis' },
  { label: '异常视图', value: 'anomaly' }
] as const

const layoutDensityOptions = [
  { label: '紧凑布局', value: 'compact' },
  { label: '均衡布局', value: 'balanced' },
  { label: '舒展布局', value: 'spacious' }
] as const

const layoutMap: Record<string, number[]> = {
  service: [0, 360, 760, 1180],
  pvc: [0, 320, 680, 1040, 1400],
  pod: [0, 340, 700],
  node: [0, 380, 820],
  ingress: [0, 360, 760, 1180],
  config: [0, 360, 780],
  pv: [0, 360, 820],
  serviceaccount: [0, 340, 760, 1140],
  deployment: [0, 340, 760, 1140],
  'node-storage': [0, 340, 720, 1080],
  networkpolicy: [0, 340, 760, 1120],
  namespace: [0, 320, 700, 1080]
}

const nameLabel = computed(() => {
  if (mode.value === 'service') return 'Service 名称'
  if (mode.value === 'pvc') return 'PVC 名称'
  if (mode.value === 'pod') return 'Pod 名称'
  if (mode.value === 'node') return 'Node 名称'
  if (mode.value === 'ingress') return 'Ingress 名称'
  if (mode.value === 'config') return 'ConfigMap / Secret 名称'
  if (mode.value === 'pv') return 'PV 名称'
  if (mode.value === 'serviceaccount') return 'ServiceAccount 名称'
  if (mode.value === 'deployment') return 'Deployment 名称'
  if (mode.value === 'node-storage') return 'Node 名称'
  if (mode.value === 'networkpolicy') return 'NetworkPolicy 名称'
  return '命名空间综合视图'
})
const requiresNamespace = computed(() => !['node', 'pv', 'node-storage'].includes(mode.value))
const supportsNamespaceScope = computed(() => ['service', 'pvc', 'pod', 'ingress', 'config', 'serviceaccount', 'deployment', 'networkpolicy', 'namespace'].includes(mode.value))
const isNamespaceScope = computed(() => requiresNamespace.value && supportsNamespaceScope.value && scope.value === 'namespace')
const requiresResourceSelection = computed(() => !isNamespaceScope.value && mode.value !== 'namespace')
const resourceSelectPlaceholder = computed(() => `搜索并选择${nameLabel.value}`)
const activeClusterName = computed(() => clusters.value.find((it) => it.id === clusterId.value)?.name || '-')
const activeModeLabel = computed(() => modeOptions.find((it) => it.value === mode.value)?.label || '-')
const graphViewModeLabel = computed(() => graphViewModeOptions.find((it) => it.value === graphViewMode.value)?.label || '分析视图')
const layoutStrategy = computed<TopologyLayoutStrategy>(() => {
  if (isNamespaceScope.value) return 'overview'
  if (mode.value === 'pod' || mode.value === 'deployment') return 'pod-centric'
  if (mode.value === 'service' || mode.value === 'ingress') return 'service-centric'
  if (mode.value === 'pvc' || mode.value === 'pv' || mode.value === 'node-storage' || mode.value === 'node') return 'storage-centric'
  if (mode.value === 'serviceaccount' || mode.value === 'networkpolicy') return 'rbac-centric'
  return 'overview'
})
const anomalyCount = computed(() => currentGraph.value.nodes.filter((node) => node.data.severity && node.data.severity !== 'normal').length)
function isTopologyAnomaly(node: TopologyNode) {
  return !!node.data.severity && node.data.severity !== 'normal'
}

function buildFocusedNodeIds(nodes: TopologyNode[], edges: TopologyEdge[]) {
  const ids = new Set(nodes.map((node) => node.id))
  const keepIds = new Set(
    nodes
      .filter((node) => node.data.emphasis === 'core' || node.data.emphasis === 'primary' || isTopologyAnomaly(node))
      .map((node) => node.id)
  )

  if (keepIds.size === 0) return ids

  edges.forEach((edge) => {
    if (keepIds.has(edge.source)) keepIds.add(edge.target)
    if (keepIds.has(edge.target)) keepIds.add(edge.source)
  })

  return keepIds
}

const visibleGraph = computed<TopologyGraph>(() => {
  let nodes = currentGraph.value.nodes.filter((node) => {
    const kind = String(node.data.kind ?? 'infra')
    if (graphViewMode.value === 'architecture' && kind === 'event') return false
    return true
  })
  let ids = new Set(nodes.map((n) => n.id))
  let edges = currentGraph.value.edges.filter((edge) => ids.has(edge.source) && ids.has(edge.target))

  if (graphViewMode.value === 'anomaly') {
    const anomalyIds = new Set(nodes.filter((node) => isTopologyAnomaly(node)).map((node) => node.id))
    if (anomalyIds.size > 0) {
      const keepIds = new Set<string>(anomalyIds)
      edges.forEach((edge) => {
        if (anomalyIds.has(edge.source) || anomalyIds.has(edge.target)) {
          keepIds.add(edge.source)
          keepIds.add(edge.target)
        }
      })
      nodes.filter((node) => node.data.emphasis === 'core').forEach((node) => keepIds.add(node.id))
      nodes = nodes.filter((node) => keepIds.has(node.id))
    }
  }

  if (onlyKeyResources.value && graphViewMode.value !== 'anomaly') {
    const keepIds = buildFocusedNodeIds(nodes, edges)
    nodes = nodes.filter((node) => keepIds.has(node.id))
  }

  ids = new Set(nodes.map((n) => n.id))
  edges = edges.filter((edge) => ids.has(edge.source) && ids.has(edge.target))

  if (graphViewMode.value === 'anomaly') {
    // 用 Map 索引替代 O(n²) 的 find 查找
    const nodeMap = new Map(nodes.map((n) => [n.id, n]))
    edges = edges.filter((edge) => {
      const source = nodeMap.get(edge.source)
      const target = nodeMap.get(edge.target)
      return !!source && !!target && ((source.data.severity && source.data.severity !== 'normal') || (target.data.severity && target.data.severity !== 'normal') || source.data.emphasis === 'core' || target.data.emphasis === 'core')
    })
  }
  const lanes = (currentGraph.value.lanes || []).filter((lane) => nodes.some((node) => node.data.group === lane.key))
  return { nodes, edges, mermaid: buildMermaid(nodes, edges), lanes }
})
const visibleGraphMeta = computed<GraphMeta>(() => ({ nodes: visibleGraph.value.nodes.length, edges: visibleGraph.value.edges.length }))
const canvasPanelStyle = computed<Record<string, string> | undefined>(() => {
  if (props.embedded) return undefined
  const nodes = visibleGraphMeta.value.nodes

  if (nodes > 0 && nodes <= 4) {
    return { '--topology-panel-height': 'clamp(280px, 36vh, 400px)' }
  }

  if (nodes <= 8) {
    return { '--topology-panel-height': 'clamp(380px, 50vh, 560px)' }
  }

  if (nodes <= 16) {
    return { '--topology-panel-height': 'clamp(480px, 64vh, 720px)' }
  }

  return { '--topology-panel-height': 'clamp(560px, calc(100vh - 200px), 860px)' }
})
// mermaid 文本惰性生成：只在弹窗打开时才计算
const mermaidPreviewText = ref('graph LR\n  A[等待生成关系图]')

function isDark() {
  return document.documentElement.classList.contains('dark')
}

function buildMermaid(nodes: TopologyNode[], edges: TopologyEdge[]) {
  const lines = ['graph LR']
  for (const node of nodes) {
    const label = String((node.data as any)?.title ?? node.id).replace(/"/g, '\\"')
    lines.push(`  ${node.id}["${label}"]`)
  }
  for (const edge of edges) {
    const label = String((edge.label ?? '')).trim()
    lines.push(label ? `  ${edge.source} -->|${label}| ${edge.target}` : `  ${edge.source} --> ${edge.target}`)
  }
  return lines.join('\n')
}

function podUsesConfigMap(pod: any, configMapName: string) {
  const spec = pod?.spec ?? {}
  const volumes: any[] = Array.isArray(spec?.volumes) ? spec.volumes : []
  if (volumes.some((it) => String(it?.configMap?.name ?? '') === configMapName)) return true
  const containers = [...(Array.isArray(spec?.containers) ? spec.containers : []), ...(Array.isArray(spec?.initContainers) ? spec.initContainers : [])]
  return containers.some((container: any) => {
    const env: any[] = Array.isArray(container?.env) ? container.env : []
    const envFrom: any[] = Array.isArray(container?.envFrom) ? container.envFrom : []
    return env.some((it) => String(it?.valueFrom?.configMapKeyRef?.name ?? '') === configMapName) || envFrom.some((it) => String(it?.configMapRef?.name ?? '') === configMapName)
  })
}

function podUsesSecret(pod: any, secretName: string) {
  const spec = pod?.spec ?? {}
  const volumes: any[] = Array.isArray(spec?.volumes) ? spec.volumes : []
  if (volumes.some((it) => String(it?.secret?.secretName ?? '') === secretName)) return true
  const imagePullSecrets: any[] = Array.isArray(spec?.imagePullSecrets) ? spec.imagePullSecrets : []
  if (imagePullSecrets.some((it) => String(it?.name ?? '') === secretName)) return true
  const containers = [...(Array.isArray(spec?.containers) ? spec.containers : []), ...(Array.isArray(spec?.initContainers) ? spec.initContainers : [])]
  return containers.some((container: any) => {
    const env: any[] = Array.isArray(container?.env) ? container.env : []
    const envFrom: any[] = Array.isArray(container?.envFrom) ? container.envFrom : []
    return env.some((it) => String(it?.valueFrom?.secretKeyRef?.name ?? '') === secretName) || envFrom.some((it) => String(it?.secretRef?.name ?? '') === secretName)
  })
}

function endpointSliceTargetsPod(slice: any, pod: any) {
  const podName = String(pod?.metadata?.name ?? '')
  const podNamespace = String(pod?.metadata?.namespace ?? '')
  const podIP = String(pod?.status?.podIP ?? '')
  const endpoints: any[] = Array.isArray(slice?.endpoints) ? slice.endpoints : []

  return endpoints.some((endpoint) => {
    const targetName = String(endpoint?.targetRef?.name ?? '')
    const targetNamespace = String(endpoint?.targetRef?.namespace ?? '')
    const addresses: string[] = Array.isArray(endpoint?.addresses) ? endpoint.addresses.map((item: any) => String(item)) : []
    return (targetName === podName && (!targetNamespace || targetNamespace === podNamespace)) || (!!podIP && addresses.includes(podIP))
  })
}

function legacyEndpointsTargetPod(endpoint: any, pod: any) {
  const podName = String(pod?.metadata?.name ?? '')
  const podNamespace = String(pod?.metadata?.namespace ?? '')
  const podIP = String(pod?.status?.podIP ?? '')
  const subsets: any[] = Array.isArray(endpoint?.subsets) ? endpoint.subsets : []

  return subsets.some((subset) => {
    const addresses: any[] = [
      ...(Array.isArray(subset?.addresses) ? subset.addresses : []),
      ...(Array.isArray(subset?.notReadyAddresses) ? subset.notReadyAddresses : [])
    ]
    return addresses.some((address) => {
      const targetName = String(address?.targetRef?.name ?? '')
      const targetNamespace = String(address?.targetRef?.namespace ?? '')
      return (targetName === podName && (!targetNamespace || targetNamespace === podNamespace)) || (!!podIP && String(address?.ip ?? '') === podIP)
    })
  })
}

type PodServiceLink = {
  service: any
  endpointSlices: any[]
  legacyEndpoint?: any
  backendPods: any[]
  matchedBySelector: boolean
  matchedByEndpoint: boolean
}

type PodLaneLayout = {
  controlX: number
  identityX: number
  podX: number
  runtimeX: number
  networkX: number
  endpointX: number
  ingressX: number
  storageX: number
  pvX: number
  attachmentX: number
  configX: number
  secretX: number
  eventX: number
}

function podLaneLayout(): PodLaneLayout {
  return {
    controlX: 60,
    identityX: 420,
    podX: 880,
    runtimeX: 1220,
    networkX: 1220,
    endpointX: 1580,
    ingressX: 1940,
    storageX: 1220,
    pvX: 1580,
    attachmentX: 1940,
    configX: 420,
    secretX: 780,
    eventX: 1580
  }
}

function collectPodServiceLinks(services: any[], endpoints: any[], endpointSlices: any[], pods: any[], pod: any): PodServiceLink[] {
  return services.reduce<PodServiceLink[]>((result, service) => {
    const serviceName = String(service?.metadata?.name ?? '').trim()
    if (!serviceName) return result
    const selector = service?.spec?.selector ?? {}
    const backendPods = Object.keys(selector).length > 0 ? pods.filter((item) => matchLabels(item?.metadata?.labels, selector)) : []
    const matchedBySelector = backendPods.some((item) => String(item?.metadata?.name ?? '') === String(pod?.metadata?.name ?? ''))
    const matchedEndpointSlices = endpointSlices.filter((item) => String(item?.metadata?.labels?.['kubernetes.io/service-name'] ?? '') === serviceName)
    const endpointBackedSlices = matchedEndpointSlices.filter((item) => endpointSliceTargetsPod(item, pod))
    const legacyEndpoint = endpoints.find((item) => String(item?.metadata?.name ?? '') === serviceName)
    const matchedByEndpoint = endpointBackedSlices.length > 0 || (!!legacyEndpoint && legacyEndpointsTargetPod(legacyEndpoint, pod))

    if (!matchedBySelector && !matchedByEndpoint) return result

    result.push({
      service,
      endpointSlices: matchedEndpointSlices,
      legacyEndpoint,
      backendPods,
      matchedBySelector,
      matchedByEndpoint
    })
    return result
  }, [])
}

const nodeWidth = 236
const nodeHeight = 104

function normalizeRelation(relation?: string) {
  return String(relation ?? '').trim().toLowerCase() || undefined
}

const relationLabelI18nMap: Record<string, string> = {
  attach: '附着关系',
  binds: '绑定关系',
  bound: '绑定关系',
  claim: '声明',
  class: '存储类',
  config: '配置',
  contains: '包含',
  controls: '控制关系',
  'consumed by': '被使用',
  endpoint: '端点',
  entry: '入口',
  env: '环境变量',
  egress: '出站',
  'envfrom': '批量环境源',
  identity: '身份关联',
  'imagepullsecret': '镜像拉取凭据',
  ingress: '入站',
  'ingress+egress': '入站+出站',
  legacy: '旧版',
  limits: '限制',
  mount: '挂载',
  mounts: '挂载关系',
  'mount/env': '挂载或环境变量',
  'mounted on': '挂载到',
  'namespace all pods': '命名空间全部 Pod',
  owns: '归属控制',
  policy: '策略',
  quota: '配额',
  ref: '引用关系',
  references: '引用关系',
  reports: '上报',
  routes: '路由',
  runs: '运行',
  scheduled: '调度',
  'scheduled on': '调度',
  secret: '密钥',
  selector: '选择器',
  'selector+endpoint': '选择器+端点',
  selects: '选择',
  slice: '分片',
  storage: '存储',
  'storage capability': '存储能力',
  targets: '指向',
  volume: '卷'
}

function formatRelationLabel(label?: string) {
  const english = String(label ?? '').trim()
  if (!english) return undefined
  const translated = relationLabelI18nMap[normalizeRelation(english) ?? '']
  return translated ? `${translated}(${english})` : english
}

function getPodRestartCount(pod: any) {
  const statuses: any[] = Array.isArray(pod?.status?.containerStatuses) ? pod.status.containerStatuses : []
  return statuses.reduce((sum, status) => sum + Number(status?.restartCount ?? 0), 0)
}

function getPodReadySummary(pod: any) {
  const statuses: any[] = Array.isArray(pod?.status?.containerStatuses) ? pod.status.containerStatuses : []
  const total = statuses.length
  const ready = statuses.filter((status) => status?.ready === true).length
  return `${ready}/${total || 1} Ready`
}

function getPodAbnormalReason(pod: any) {
  const statuses: any[] = [
    ...(Array.isArray(pod?.status?.containerStatuses) ? pod.status.containerStatuses : []),
    ...(Array.isArray(pod?.status?.initContainerStatuses) ? pod.status.initContainerStatuses : [])
  ]

  for (const status of statuses) {
    const waitingReason = String(status?.state?.waiting?.reason ?? '').trim()
    if (waitingReason) return waitingReason
    const terminatedReason = String(status?.lastState?.terminated?.reason ?? status?.state?.terminated?.reason ?? '').trim()
    if (terminatedReason) return terminatedReason
  }

  return String(pod?.status?.reason ?? '').trim()
}

function getPodSeverity(pod: any): Pick<TopologyNodeData, 'severity' | 'statusText'> {
  const phase = String(pod?.status?.phase ?? '').trim()
  const reason = getPodAbnormalReason(pod)
  if (['CrashLoopBackOff', 'ImagePullBackOff', 'ErrImagePull', 'OOMKilled', 'CreateContainerError'].includes(reason)) {
    return { severity: 'error', statusText: reason }
  }
  if (phase === 'Failed') return { severity: 'error', statusText: 'Failed' }
  if (phase === 'Pending' || reason === 'ContainerCreating') return { severity: 'warning', statusText: reason || phase }
  return { severity: 'normal', statusText: phase || 'Running' }
}

function getPVCSeverity(pvc: any): Pick<TopologyNodeData, 'severity' | 'statusText'> {
  const phase = String(pvc?.status?.phase ?? '').trim()
  const pvName = String(pvc?.spec?.volumeName ?? '').trim()
  if (!pvName || phase === 'Pending') return { severity: 'warning', statusText: phase || '未绑定 PV' }
  return { severity: 'normal', statusText: phase || 'Bound' }
}

function getServiceSeverity(service: any, backendCount: number, sliceCount: number, hasLegacyEndpoints: boolean): Pick<TopologyNodeData, 'severity' | 'statusText'> {
  if (backendCount === 0 && sliceCount === 0 && !hasLegacyEndpoints) {
    return { severity: 'warning', statusText: '无后端实例' }
  }
  if (backendCount === 0 && (sliceCount > 0 || hasLegacyEndpoints)) {
    return { severity: 'warning', statusText: '存在端点但未命中当前 Pod' }
  }
  if (backendCount === 0) {
    return { severity: 'warning', statusText: '无 healthy backend' }
  }
  return { severity: 'normal', statusText: `${backendCount} 后端` }
}

function getServiceRelationSummary(link: PodServiceLink) {
  const parts = [
    `selector=${link.matchedBySelector ? 'yes' : 'no'}`,
    `endpoint=${link.matchedByEndpoint ? 'yes' : 'no'}`,
    `slice=${link.endpointSlices.length}`
  ]
  return parts.join(' | ')
}

function podServiceRelationText(link: PodServiceLink) {
  if (link.matchedBySelector && link.matchedByEndpoint) return 'selector+endpoint'
  if (link.matchedByEndpoint) return 'endpoint'
  return 'selector'
}

function getVolumeAttachmentSeverity(attachment: any): Pick<TopologyNodeData, 'severity' | 'statusText'> {
  return attachment?.status?.attached === true
    ? { severity: 'normal', statusText: 'attached' }
    : { severity: 'warning', statusText: 'unattached' }
}

function getServiceAccountSeverity(localBindings: any[], globalBindings: any[]): Pick<TopologyNodeData, 'severity' | 'statusText'> {
  const roleRefs = [...localBindings, ...globalBindings].map((item) => String(item?.roleRef?.name ?? '').trim().toLowerCase())
  if (roleRefs.includes('cluster-admin')) return { severity: 'error', statusText: 'cluster-admin' }
  if (roleRefs.some((item) => item.includes('admin'))) return { severity: 'warning', statusText: '高权限绑定' }
  return { severity: 'normal', statusText: `绑定 ${roleRefs.length}` }
}

function buildNodeTooltip(data: TopologyNodeData) {
  return [data.title, data.description, data.refText, data.statusText].filter(Boolean).join('\n')
}

function finalizeGraph(graph: TopologyGraph) {
  return {
    ...graph,
    nodes: graph.nodes.map((node, index) => {
      const isPrimaryNode = !isNamespaceScope.value && index === 0
      const emphasis: TopologyEmphasis = isPrimaryNode || node.data.group === 'core'
        ? 'core'
        : node.data.emphasis ?? 'secondary'
      const group = isPrimaryNode ? 'core' : node.data.group
      const decoratedData: TopologyNodeData = {
        ...node.data,
        group,
        emphasis,
        severity: node.data.severity ?? 'normal',
        statusText: node.data.statusText ?? ''
      }

      return {
        ...node,
        data: {
          ...decoratedData,
          tooltip: buildNodeTooltip(decoratedData)
        }
      }
    })
  }
}

function buildPodLanes(graph: TopologyGraph): TopologyLane[] {
  const laneLabels: Record<string, string> = {
    control: '控制',
    identity: '权限',
    core: 'Pod',
    network: '网络',
    storage: '存储',
    config: '配置',
    runtime: '事件 / 运行'
  }
  const groups = Object.keys(laneLabels)
  const lanes: TopologyLane[] = []
  for (const group of groups) {
    const items = graph.nodes.filter((node) => node.data.group === group)
    if (items.length === 0) continue
    const minX = Math.min(...items.map((node) => node.position.x)) - 36
    const minY = Math.min(...items.map((node) => node.position.y)) - 48
    const maxX = Math.max(...items.map((node) => node.position.x + Number(node.data.cardWidth ?? 260))) + 36
    const maxY = Math.max(...items.map((node) => node.position.y + Number(node.data.cardHeight ?? 118))) + 36
    lanes.push({ key: group, label: laneLabels[group], x: minX, y: minY, width: maxX - minX, height: maxY - minY })
  }
  return lanes
}

function createNode(
  id: string,
  x: number,
  y: number,
  badge: string,
  title: string,
  description: string,
  refText: string,
  kind: string,
  targetKind?: string,
  targetNamespace?: string,
  targetName?: string,
  group?: TopologyGroup,
  meta: Partial<TopologyNodeData> = {}
): TopologyNode {
  const data: TopologyNodeData = {
    badge,
    title,
    description,
    refText,
    kind,
    targetKind,
    targetNamespace,
    targetName,
    group,
    severity: 'normal',
    statusText: '',
    emphasis: group === 'core' ? 'core' : 'secondary',
    ...meta
  }
  data.tooltip = meta.tooltip ?? buildNodeTooltip(data)
  return { id, position: { x, y }, data }
}

function createEdge(id: string, source: string, target: string, label?: string, relation?: string): TopologyEdge {
  return {
    id,
    source,
    target,
    label: formatRelationLabel(label),
    relation: normalizeRelation(relation ?? label)
  }
}

function safeGraphIdPart(value: string) {
  const normalized = String(value || 'item').trim()
  return normalized.replace(/[^A-Za-z0-9_-]+/g, '_') || 'item'
}

function stableGraphId(prefix: string, ...parts: string[]) {
  return [prefix, ...parts.map(safeGraphIdPart)].join('-')
}

function appendNodeOnce(nodes: TopologyNode[], ids: Set<string>, node: TopologyNode) {
  if (ids.has(node.id)) return
  nodes.push(node)
  ids.add(node.id)
}

function appendEdgeOnce(edges: TopologyEdge[], ids: Set<string>, edge: TopologyEdge) {
  if (ids.has(edge.id)) return
  edges.push(edge)
  ids.add(edge.id)
}

function distributeVertical<T>(items: T[], startY: number, gap = 130) {
  return items.map((item, index) => ({ item, y: startY + index * gap }))
}

function takeByDepth<T>(items: T[]) {
  return items
}

function applyGraphLayout(graph: TopologyGraph) {
  return applyTopologyAutoLayout(graph, {
    mode: mode.value,
    namespaceScope: isNamespaceScope.value,
    density: layoutDensity.value,
    strategy: layoutStrategy.value,
    viewMode: graphViewMode.value,
    width: nodeWidth,
    height: nodeHeight
  })
}

function fitView() {
  graphCanvasRef.value?.fitView()
}

function openNodeTarget(node: TopologyNode) {
  const { targetKind, targetNamespace, targetName } = node.data
  if (!clusterId.value || !targetKind || !targetName) return
  router.push({
    name: 'K8sClusterManage',
    params: { clusterId: String(clusterId.value) },
    query: {
      targetKind,
      targetNamespace,
      targetName
    }
  })
}

function handleTopologyNodeOpen(node: { data: { targetKind?: string; targetNamespace?: string; targetName?: string } }) {
  openNodeTarget(node as TopologyNode)
}

function zoomIn() {
  graphCanvasRef.value?.zoomIn()
}

function zoomOut() {
  graphCanvasRef.value?.zoomOut()
}

async function relayoutGraph(showToast = true) {
  if (currentGraph.value.nodes.length === 0) return
  if (showToast) {
    const nodeCount = currentGraph.value.nodes.length
    const nextDensity: TopologyLayoutDensity =
      nodeCount > 88 ? 'compact' : nodeCount > 28 ? 'balanced' : 'spacious'
    const nextOnlyKeyResources =
      nodeCount > 120 || ((mode.value === 'node' || mode.value === 'pv' || mode.value === 'node-storage') && nodeCount > 36)

    if (layoutDensity.value !== nextDensity) layoutDensity.value = nextDensity
    if (onlyKeyResources.value !== nextOnlyKeyResources) onlyKeyResources.value = nextOnlyKeyResources
    if (!minimapVisible.value) minimapVisible.value = true
  }

  currentGraph.value = applyGraphLayout(currentGraph.value)
  await nextTick()
  fitView()
  if (showToast) notifySuccess('已完成一键美化布局')
}

function onGraphPositionsChange(payload: Array<{ id: string; position: { x: number; y: number } }>) {
  if (!payload.length) return
  const positionMap = new Map(payload.map((item) => [item.id, item.position]))
  currentGraph.value = {
    ...currentGraph.value,
    nodes: currentGraph.value.nodes.map((node) => {
      const nextPosition = positionMap.get(node.id)
      if (!nextPosition) return node
      return {
        ...node,
        position: { ...nextPosition }
      }
    })
  }
}

async function buildServiceTopology(cluster: number, ns: string, name: string): Promise<TopologyGraph> {
  const [services, endpoints, endpointSlices, pods] = await Promise.all([
    k8sApi.listServices(cluster, { namespace: ns }),
    k8sApi.listEndpoints(cluster, { namespace: ns }),
    k8sApi.listEndpointSlices(cluster, { namespace: ns }),
    k8sApi.listPods(cluster, { namespace: ns })
  ])
  const svc = (services.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (!svc) throw new Error(`未找到 Service ${ns}/${name}`)
  const nodes: TopologyNode[] = []
  const edges: TopologyEdge[] = []
  const selector = svc?.spec?.selector ?? {}
  nodes.push(createNode('svc', 0, 80, 'Service', `${ns}/${name}`, `type=${String(svc?.spec?.type ?? '-')}`, `selector=${Object.keys(selector).length}`, 'service', 'Service', ns, name))
  const ep = (endpoints.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (ep) {
    const subsets: any[] = Array.isArray(ep?.subsets) ? ep.subsets : []
    const ready = subsets.reduce((sum, s) => sum + (Array.isArray(s?.addresses) ? s.addresses.length : 0), 0)
    const notReady = subsets.reduce((sum, s) => sum + (Array.isArray(s?.notReadyAddresses) ? s.notReadyAddresses.length : 0), 0)
    nodes.push(createNode('ep', 320, 0, 'Endpoints', name, `ready=${ready} notReady=${notReady}`, `${ns}/${name}`, 'network', 'Endpoints', ns, name))
    edges.push(createEdge('svc-ep', 'svc', 'ep', 'legacy'))
  }
  const slices = (endpointSlices.list || []).filter((it: any) => String(it?.metadata?.labels?.['kubernetes.io/service-name'] ?? '') === name)
  slices.forEach((it: any, idx: number) => {
    const sliceId = `slice-${idx}`
    nodes.push(createNode(sliceId, 320, 160 + idx * 150, 'EndpointSlice', String(it?.metadata?.name ?? '-'), `endpoints=${Array.isArray(it?.endpoints) ? it.endpoints.length : 0}`, `type=${String(it?.addressType ?? '-')}`, 'network', 'EndpointSlice', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`svc-${sliceId}`, 'svc', sliceId, 'slice'))
  })
  const matchedPods = (pods.list || []).filter((it: any) => matchLabels(it?.metadata?.labels, selector))
  matchedPods.forEach((it: any, idx: number) => {
    const podId = `pod-${idx}`
    const podName = String(it?.metadata?.name ?? '-')
    nodes.push(createNode(podId, 680, idx * 140, 'Pod', podName, `phase=${String(it?.status?.phase ?? '-')}`, `${String(it?.spec?.nodeName ?? '-')}`, 'workload', 'Pod', ns, podName))
    edges.push(createEdge(`svc-${podId}`, 'svc', podId, 'selector'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNamespaceServiceTopology(cluster: number, ns: string): Promise<TopologyGraph> {
  const [services, endpointSlices, pods] = await Promise.all([
    k8sApi.listServices(cluster, { namespace: ns }),
    k8sApi.listEndpointSlices(cluster, { namespace: ns }),
    k8sApi.listPods(cluster, { namespace: ns })
  ])
  const nodes: TopologyNode[] = [createNode('ns', 0, 180, 'Namespace', ns, 'service topology overview', 'scope=namespace', 'infra')]
  const edges: TopologyEdge[] = []
  const nodeIds = new Set(nodes.map((node) => node.id))
  const edgeIds = new Set<string>()
  const svcList = takeByDepth(services.list || [])
  distributeVertical(svcList, 40, 180).forEach(({ item: svc, y }) => {
    const svcName = String(svc?.metadata?.name ?? '-')
    const svcId = stableGraphId('service', svcName)
    appendNodeOnce(nodes, nodeIds, createNode(svcId, layoutMap.service[1], y, 'Service', svcName, `type=${String(svc?.spec?.type ?? '-')}`, `selector=${Object.keys(svc?.spec?.selector ?? {}).length}`, 'service', 'Service', ns, svcName))
    appendEdgeOnce(edges, edgeIds, createEdge(`ns-${svcId}`, 'ns', svcId, 'contains'))
    const selector = svc?.spec?.selector ?? {}
    const slices = takeByDepth((endpointSlices.list || []).filter((it: any) => String(it?.metadata?.labels?.['kubernetes.io/service-name'] ?? '') === svcName))
    distributeVertical(slices, y, 92).forEach(({ item: it, y: sliceY }) => {
      const sliceName = String(it?.metadata?.name ?? '-')
      const sliceId = stableGraphId('endpointslice', sliceName)
      appendNodeOnce(nodes, nodeIds, createNode(sliceId, layoutMap.service[2], sliceY, 'EndpointSlice', sliceName, `endpoints=${Array.isArray(it?.endpoints) ? it.endpoints.length : 0}`, `type=${String(it?.addressType ?? '-')}`, 'network', 'EndpointSlice', ns, sliceName))
      appendEdgeOnce(edges, edgeIds, createEdge(`${svcId}-${sliceId}`, svcId, sliceId, 'slice'))
    })
    const matched = takeByDepth((pods.list || []).filter((it: any) => matchLabels(it?.metadata?.labels, selector)))
    distributeVertical(matched, y, 86).forEach(({ item: it, y: podY }, podIdx) => {
      const podName = String(it?.metadata?.name ?? '-')
      const podId = stableGraphId('pod', podName)
      appendNodeOnce(nodes, nodeIds, createNode(podId, layoutMap.service[3], podY + podIdx * 6, 'Pod', podName, `phase=${String(it?.status?.phase ?? '-')}`, `${String(it?.spec?.nodeName ?? '-')}`, 'workload', 'Pod', ns, podName, undefined, getPodSeverity(it)))
      appendEdgeOnce(edges, edgeIds, createEdge(`${svcId}-${podId}`, svcId, podId, 'selector', 'routes'))
    })
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildPVCTopology(cluster: number, ns: string, name: string): Promise<TopologyGraph> {
  const [pvcs, pvs, storageClasses, volumeAttachments, pods] = await Promise.all([
    k8sApi.listPVCs(cluster, { namespace: ns }),
    k8sApi.listPVs(cluster, {}),
    k8sApi.listStorageClasses(cluster, {}),
    k8sApi.listVolumeAttachments(cluster, {}),
    k8sApi.listPods(cluster, { namespace: ns })
  ])
  const pvc = (pvcs.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (!pvc) throw new Error(`未找到 PVC ${ns}/${name}`)
  const nodes: TopologyNode[] = []
  const edges: TopologyEdge[] = []
  nodes.push(createNode('pvc', 0, 120, 'PVC', `${ns}/${name}`, `phase=${String(pvc?.status?.phase ?? '-')}`, `request=${String(pvc?.spec?.resources?.requests?.storage ?? '-')}`, 'storage', 'PVC', ns, name))
  const pvName = String(pvc?.spec?.volumeName ?? '')
  const storageClassName = String(pvc?.spec?.storageClassName ?? '')
  if (pvName) {
    const pv = (pvs.list || []).find((it: any) => String(it?.metadata?.name ?? '') === pvName)
    if (pv) {
      nodes.push(createNode('pv', 290, 40, 'PV', pvName, `phase=${String(pv?.status?.phase ?? '-')}`, `capacity=${String(pv?.spec?.capacity?.storage ?? '-')}`, 'storage', 'PV', undefined, pvName))
      edges.push(createEdge('pvc-pv', 'pvc', 'pv', 'bound'))
    }
    const attachments = (volumeAttachments.list || []).filter((it: any) => String(it?.spec?.source?.persistentVolumeName ?? '') === pvName)
    attachments.forEach((it: any, idx: number) => {
      const vaId = `va-${idx}`
      const nodeName = String(it?.spec?.nodeName ?? '-')
      nodes.push(createNode(vaId, 610, idx * 170, 'VolumeAttachment', String(it?.metadata?.name ?? '-'), `attached=${it?.status?.attached === true ? 'yes' : 'no'}`, `node=${nodeName}`, 'storage', 'VolumeAttachment', undefined, String(it?.metadata?.name ?? '-')))
      edges.push(createEdge(`pv-${vaId}`, 'pv', vaId, 'attach'))
      if (nodeName && !nodes.some((n) => n.id === `node-${nodeName}`)) {
        nodes.push(createNode(`node-${nodeName}`, 940, idx * 170, 'Node', nodeName, 'attached node', '', 'infra', 'Node', undefined, nodeName))
      }
      edges.push(createEdge(`${vaId}-node-${idx}`, vaId, `node-${nodeName}`, 'mounted on'))
    })
  }
  if (storageClassName) {
    const sc = (storageClasses.list || []).find((it: any) => String(it?.metadata?.name ?? '') === storageClassName)
    if (sc) {
      nodes.push(createNode('sc', 290, 260, 'StorageClass', storageClassName, `mode=${String(sc?.volumeBindingMode ?? '-')}`, `reclaim=${String(sc?.reclaimPolicy ?? '-')}`, 'storage', 'StorageClass', undefined, storageClassName))
      edges.push(createEdge('pvc-sc', 'pvc', 'sc', 'class'))
    }
  }
  const relatedPods = (pods.list || []).filter((it: any) => podUsesPvc(it, name))
  relatedPods.forEach((it: any, idx: number) => {
    const podId = `pod-${idx}`
    nodes.push(createNode(podId, 610, 320 + idx * 140, 'Pod', String(it?.metadata?.name ?? '-'), `phase=${String(it?.status?.phase ?? '-')}`, `${String(it?.spec?.nodeName ?? '-')}`, 'workload', 'Pod', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`pvc-pod-${idx}`, 'pvc', podId, 'consumed by'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNamespacePVCTopology(cluster: number, ns: string): Promise<TopologyGraph> {
  const [pvcs, _pvs, storageClasses, volumeAttachments] = await Promise.all([
    k8sApi.listPVCs(cluster, { namespace: ns }),
    k8sApi.listPVs(cluster, {}),
    k8sApi.listStorageClasses(cluster, {}),
    k8sApi.listVolumeAttachments(cluster, {})
  ])
  const nodes: TopologyNode[] = [createNode('ns', 0, 180, 'Namespace', ns, 'pvc topology overview', 'scope=namespace', 'infra')]
  const edges: TopologyEdge[] = []
  const addedScIds = new Set<string>()
  const pvcList = takeByDepth(pvcs.list || [])
  distributeVertical(pvcList, 40, 190).forEach(({ item: pvc, y }, idx) => {
    const pvcId = `pvc-${idx}`
    const pvcName = String(pvc?.metadata?.name ?? '-')
    nodes.push(createNode(pvcId, layoutMap.pvc[1], y, 'PVC', pvcName, `phase=${String(pvc?.status?.phase ?? '-')}`, `request=${String(pvc?.spec?.resources?.requests?.storage ?? '-')}`, 'storage', 'PVC', ns, pvcName))
    edges.push(createEdge(`ns-pvc-${idx}`, 'ns', pvcId, 'claim'))
    const pvName = String(pvc?.spec?.volumeName ?? '')
    const scName = String(pvc?.spec?.storageClassName ?? '')
    if (pvName) {
      const pvId = `${pvcId}-pv`
      nodes.push(createNode(pvId, layoutMap.pvc[2], y, 'PV', pvName, 'bound pv', '', 'storage', 'PV', undefined, pvName))
      edges.push(createEdge(`${pvcId}-pv-edge`, pvcId, pvId, 'bound'))
      const vas = takeByDepth((volumeAttachments.list || []).filter((it: any) => String(it?.spec?.source?.persistentVolumeName ?? '') === pvName))
      distributeVertical(vas, y, 92).forEach(({ item: it, y: vaY }, vaIdx) => {
        const vaId = `${pvcId}-va-${vaIdx}`
        const nodeName = String(it?.spec?.nodeName ?? '-')
        nodes.push(createNode(vaId, layoutMap.pvc[4], vaY, 'VolumeAttachment', String(it?.metadata?.name ?? '-'), `attached=${it?.status?.attached === true ? 'yes' : 'no'}`, `node=${nodeName}`, 'storage', 'VolumeAttachment', undefined, String(it?.metadata?.name ?? '-')))
        edges.push(createEdge(`${pvId}-va-edge-${vaIdx}`, pvId, vaId, 'attach'))
      })
    }
    if (scName) {
      const scId = `sc-${scName}`
      if (!addedScIds.has(scId)) {
        const sc = (storageClasses.list || []).find((it: any) => String(it?.metadata?.name ?? '') === scName)
        nodes.push(createNode(scId, layoutMap.pvc[3], y, 'StorageClass', scName, `mode=${String(sc?.volumeBindingMode ?? '-')}`, `reclaim=${String(sc?.reclaimPolicy ?? '-')}`, 'storage', 'StorageClass', undefined, scName))
        addedScIds.add(scId)
      }
      edges.push(createEdge(`${pvcId}-sc-edge`, pvcId, scId, 'class'))
    }
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildPodTopology(cluster: number, ns: string, name: string): Promise<TopologyGraph> {
  const [pods, replicaSets, workloads, services, endpoints, endpointSlices, ingresses, events, networkPolicies, pvcs, pvs, storageClasses, volumeAttachments, configMaps, secrets, serviceAccounts, roleBindings, clusterRoleBindings, roles, clusterRoles] = await Promise.all([
    k8sApi.listPods(cluster, { namespace: ns }),
    k8sApi.listReplicaSets(cluster, { namespace: ns }),
    k8sApi.listWorkloads(cluster, { namespace: ns }),
    k8sApi.listServices(cluster, { namespace: ns }),
    k8sApi.listEndpoints(cluster, { namespace: ns }),
    k8sApi.listEndpointSlices(cluster, { namespace: ns }),
    k8sApi.listIngresses(cluster, { namespace: ns }),
    k8sApi.listEvents(cluster, { namespace: ns }),
    k8sApi.listNetworkPolicies(cluster, { namespace: ns }),
    k8sApi.listPVCs(cluster, { namespace: ns }),
    k8sApi.listPVs(cluster, {}),
    k8sApi.listStorageClasses(cluster, {}),
    k8sApi.listVolumeAttachments(cluster, {}),
    k8sApi.listConfigMaps(cluster, { namespace: ns }),
    k8sApi.listSecrets(cluster, { namespace: ns }),
    k8sApi.listServiceAccounts(cluster, { namespace: ns }),
    k8sApi.listRoleBindings(cluster, { namespace: ns }),
    k8sApi.listClusterRoleBindings(cluster, {}),
    k8sApi.listRoles(cluster, { namespace: ns }),
    k8sApi.listClusterRoles(cluster, {})
  ])
  const pod = (pods.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (!pod) throw new Error(`未找到 Pod ${ns}/${name}`)
  const nodes: TopologyNode[] = []
  const edges: TopologyEdge[] = []
  const lane = podLaneLayout()
  const podStatus = getPodSeverity(pod)
  const podRestarts = getPodRestartCount(pod)
  const podReason = getPodAbnormalReason(pod)
  nodes.push(createNode(
    'pod',
    lane.podX,
    620,
    'Pod',
    `${ns}/${name}`,
    `${getPodReadySummary(pod)} · restarts=${podRestarts}`,
    `node=${String(pod?.spec?.nodeName ?? '-')}`,
    'workload',
    'Pod',
    ns,
    name,
    'core',
    {
      ...podStatus,
      emphasis: 'core',
      tooltip: [`${ns}/${name}`, getPodReadySummary(pod), `phase=${String(pod?.status?.phase ?? '-')}`, `restart=${podRestarts}`, podReason || '', `node=${String(pod?.spec?.nodeName ?? '-')}`].filter(Boolean).join('\n')
    }
  ))
  const owner = (Array.isArray(pod?.metadata?.ownerReferences) ? pod.metadata.ownerReferences : [])[0]
  if (owner && String(owner?.kind ?? '') === 'ReplicaSet') {
    const rsName = String(owner?.name ?? '')
    const rs = (replicaSets.list || []).find((it: any) => String(it?.metadata?.name ?? '') === rsName)
    if (rs) {
      nodes.push(createNode('rs', lane.controlX + 320, 600, 'ReplicaSet', rsName, `ready=${String(rs?.status?.readyReplicas ?? 0)}/${String(rs?.status?.replicas ?? 0)}`, `namespace=${ns}`, 'workload', 'ReplicaSet', ns, rsName, 'control', { emphasis: 'primary' }))
      edges.push(createEdge('rs-pod', 'rs', 'pod', 'controls', 'controls'))
      const rsOwner = (Array.isArray(rs?.metadata?.ownerReferences) ? rs.metadata.ownerReferences : [])[0]
      if (rsOwner && String(rsOwner?.kind ?? '') === 'Deployment') {
        const depName = String(rsOwner?.name ?? '')
        const dep = (workloads.list || []).find((it: any) => String(it?.kind ?? '') === 'Deployment' && String(it?.metadata?.name ?? '') === depName)
        nodes.push(createNode('dep', lane.controlX, 600, 'Deployment', depName, dep ? `ready=${String(dep?.status?.readyReplicas ?? 0)}/${String(dep?.status?.replicas ?? 0)}` : 'controller', `namespace=${ns}`, 'workload', 'Deployment', ns, depName, 'control', { emphasis: 'primary' }))
        edges.push(createEdge('dep-rs', 'dep', 'rs', 'owns', 'controls'))
      }
    }
  } else if (owner && ['StatefulSet', 'DaemonSet', 'Job'].includes(String(owner?.kind ?? ''))) {
    const kind = String(owner?.kind ?? '')
    const ownerName = String(owner?.name ?? '')
    const workload = (workloads.list || []).find((it: any) => String(it?.kind ?? '') === kind && String(it?.metadata?.name ?? '') === ownerName)
    nodes.push(createNode('owner', lane.controlX, 600, kind, ownerName, workload ? `ready=${String(workload?.status?.readyReplicas ?? workload?.status?.replicas ?? 0)}` : 'controller', `namespace=${ns}`, 'workload', kind, ns, ownerName, 'control', { emphasis: 'primary' }))
    edges.push(createEdge('owner-pod', 'owner', 'pod', 'controls', 'controls'))
  } else if (owner && String(owner?.kind ?? '') === 'CronJob') {
    const ownerName = String(owner?.name ?? '')
    nodes.push(createNode('cronjob', lane.controlX, 600, 'CronJob', ownerName, 'schedule controller', `namespace=${ns}`, 'workload', 'CronJob', ns, ownerName, 'control', { emphasis: 'primary' }))
    edges.push(createEdge('cronjob-pod', 'cronjob', 'pod', 'controls', 'controls'))
  }

  const nodeName = String(pod?.spec?.nodeName ?? '').trim()
  if (nodeName) {
    nodes.push(createNode('node', lane.runtimeX, 420, 'Node', nodeName, 'scheduled node', '', 'infra', 'Node', undefined, nodeName, 'runtime', { emphasis: 'primary' }))
    edges.push(createEdge('pod-node', 'pod', 'node', 'scheduled on', 'references'))
  }

  const serviceAccountName = String(pod?.spec?.serviceAccountName ?? 'default').trim()
  const serviceAccount = (serviceAccounts.list || []).find((it: any) => String(it?.metadata?.name ?? '') === serviceAccountName)
  if (serviceAccount) {
    const localBindings = takeByDepth((roleBindings.list || []).filter((it: any) => Array.isArray(it?.subjects) && it.subjects.some((s: any) => String(s?.kind ?? '') === 'ServiceAccount' && String(s?.name ?? '') === serviceAccountName && String(s?.namespace ?? ns) === ns)))
    const globalBindings = takeByDepth((clusterRoleBindings.list || []).filter((it: any) => Array.isArray(it?.subjects) && it.subjects.some((s: any) => String(s?.kind ?? '') === 'ServiceAccount' && String(s?.name ?? '') === serviceAccountName && String(s?.namespace ?? '') === ns)))
    nodes.push(createNode('sa', lane.identityX, 620, 'ServiceAccount', serviceAccountName, `bindings=${localBindings.length + globalBindings.length}`, ns, 'network', 'ServiceAccount', ns, serviceAccountName, 'identity', {
      ...getServiceAccountSeverity(localBindings, globalBindings),
      emphasis: 'primary'
    }))
    edges.push(createEdge('sa-pod', 'sa', 'pod', 'identity', 'binds'))
    localBindings.forEach((it: any, idx: number) => {
      const rbId = `rb-${idx}`
      const roleRefName = String(it?.roleRef?.name ?? '-')
        nodes.push(createNode(rbId, lane.identityX - 320, 920 + idx * 120, 'RoleBinding', String(it?.metadata?.name ?? '-'), `roleRef=${roleRefName}`, ns, 'network', 'RoleBinding', ns, String(it?.metadata?.name ?? '-'), 'identity'))
      edges.push(createEdge(`sa-rb-${idx}`, 'sa', rbId, 'bound', 'binds'))
      const role = (roles.list || []).find((r: any) => String(r?.metadata?.name ?? '') === roleRefName)
      if (role) {
        const roleId = `role-${idx}`
          nodes.push(createNode(roleId, lane.identityX, 920 + idx * 120, 'Role', roleRefName, `rules=${Array.isArray(role?.rules) ? role.rules.length : 0}`, ns, 'network', 'Role', ns, roleRefName, 'identity'))
        edges.push(createEdge(`rb-role-${idx}`, rbId, roleId, 'ref', 'references'))
      }
    })
    globalBindings.forEach((it: any, idx: number) => {
      const crbId = `crb-${idx}`
      const roleRefName = String(it?.roleRef?.name ?? '-')
        nodes.push(createNode(crbId, lane.identityX - 320, 1280 + idx * 120, 'ClusterRoleBinding', String(it?.metadata?.name ?? '-'), `roleRef=${roleRefName}`, 'cluster scope', 'network', 'ClusterRoleBinding', undefined, String(it?.metadata?.name ?? '-'), 'identity', {
        severity: roleRefName.toLowerCase() === 'cluster-admin' ? 'error' : 'normal',
        statusText: roleRefName.toLowerCase() === 'cluster-admin' ? 'cluster-admin' : 'cluster role'
      }))
      edges.push(createEdge(`sa-crb-${idx}`, 'sa', crbId, 'bound', 'binds'))
      const clusterRole = (clusterRoles.list || []).find((r: any) => String(r?.metadata?.name ?? '') === roleRefName)
      if (clusterRole) {
        const crId = `cr-${idx}`
          nodes.push(createNode(crId, lane.identityX, 1280 + idx * 120, 'ClusterRole', roleRefName, `rules=${Array.isArray(clusterRole?.rules) ? clusterRole.rules.length : 0}`, 'cluster scope', 'network', 'ClusterRole', undefined, roleRefName, 'identity', {
          severity: roleRefName.toLowerCase() === 'cluster-admin' ? 'error' : 'normal',
          statusText: roleRefName.toLowerCase() === 'cluster-admin' ? '全局高权限' : 'cluster role'
        }))
        edges.push(createEdge(`crb-cr-${idx}`, crbId, crId, 'ref', 'references'))
      }
    })
  }

  const podServiceLinks = takeByDepth(collectPodServiceLinks(services.list || [], endpoints.list || [], endpointSlices.list || [], pods.list || [], pod))
  podServiceLinks.forEach((link, idx: number) => {
    const svcName = String(link.service?.metadata?.name ?? '-')
    const svcId = `svc-${idx}`
      nodes.push(createNode(svcId, lane.networkX, 180 + idx * 220, 'Service', svcName, `type=${String(link.service?.spec?.type ?? '-')} · backend=${link.backendPods.length}`, `${getServiceRelationSummary(link)}${link.backendPods.length === 0 ? ' | no healthy backend' : ''}`, 'service', 'Service', ns, svcName, 'network', {
        ...getServiceSeverity(link.service, link.backendPods.length, link.endpointSlices.length, Boolean(link.legacyEndpoint)),
        emphasis: 'primary'
      }))
    edges.push(createEdge(`svc-pod-${idx}`, svcId, 'pod', podServiceRelationText(link), 'routes'))
    link.endpointSlices.forEach((it: any, sliceIdx: number) => {
      const sliceId = `${svcId}-slice-${sliceIdx}`
      const targetsCurrentPod = endpointSliceTargetsPod(it, pod)
        nodes.push(createNode(sliceId, lane.endpointX, 120 + idx * 220 + sliceIdx * 112, 'EndpointSlice', String(it?.metadata?.name ?? '-'), `endpoints=${Array.isArray(it?.endpoints) ? it.endpoints.length : 0}`, `type=${String(it?.addressType ?? '-')}`, 'network', 'EndpointSlice', ns, String(it?.metadata?.name ?? '-'), 'network', {
        severity: targetsCurrentPod ? 'normal' : 'warning',
        statusText: targetsCurrentPod ? '命中当前 Pod' : '未命中当前 Pod'
      }))
      edges.push(createEdge(`${svcId}-slice-${sliceIdx}`, svcId, sliceId, 'slice', 'references'))
      if (targetsCurrentPod) {
        edges.push(createEdge(`${sliceId}-pod`, sliceId, 'pod', 'targets', 'routes'))
      }
    })

    if (link.legacyEndpoint) {
      const legacyId = `ep-${idx}`
      const subsets: any[] = Array.isArray(link.legacyEndpoint?.subsets) ? link.legacyEndpoint.subsets : []
      const ready = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.addresses) ? subset.addresses.length : 0), 0)
      nodes.push(createNode(legacyId, lane.endpointX, 210 + idx * 220, 'Endpoints', svcName, `ready=${ready}`, ns, 'network', 'Endpoints', ns, svcName, 'network', {
        severity: ready > 0 ? 'normal' : 'warning',
        statusText: ready > 0 ? `${ready} ready` : '无 ready 地址'
      }))
      edges.push(createEdge(`svc-ep-${idx}`, svcId, legacyId, 'legacy', 'references'))
      if (legacyEndpointsTargetPod(link.legacyEndpoint, pod)) {
        edges.push(createEdge(`${legacyId}-pod`, legacyId, 'pod', 'targets', 'routes'))
      }
    }

    const ingRefs = takeByDepth((ingresses.list || []).filter((ing: any) => ingressUsesServiceName(ing, svcName)))
    ingRefs.forEach((ing: any, ingIdx: number) => {
      const id = `ing-${idx}-${ingIdx}`
      nodes.push(createNode(id, lane.ingressX, 120 + idx * 220 + ingIdx * 98, 'Ingress', String(ing?.metadata?.name ?? '-'), `class=${String(ing?.spec?.ingressClassName ?? '-')}`, ns, 'network', 'Ingress', ns, String(ing?.metadata?.name ?? '-'), 'network'))
      edges.push(createEdge(`${id}-svc-${idx}`, id, svcId, 'routes', 'routes'))
    })
  })

  const selectedPolicies = takeByDepth((networkPolicies.list || []).filter((it: any) => matchLabels(pod?.metadata?.labels, it?.spec?.podSelector?.matchLabels ?? {})))
  selectedPolicies.forEach((policy: any, idx: number) => {
    const id = `np-${idx}`
    nodes.push(createNode(id, lane.networkX, 1120 + idx * 120, 'NetworkPolicy', String(policy?.metadata?.name ?? '-'), `types=${Array.isArray(policy?.spec?.policyTypes) ? policy.spec.policyTypes.join(',') : '-'}`, `scope=${networkPolicyScopeText(policy)}`, 'network', 'NetworkPolicy', ns, String(policy?.metadata?.name ?? '-'), 'network', {
      statusText: networkPolicyDirectionBadge(policy),
      severity: Object.keys(policy?.spec?.podSelector?.matchLabels ?? {}).length === 0 ? 'warning' : 'normal'
    }))
    edges.push(createEdge(`np-pod-${idx}`, id, 'pod', networkPolicyDirectionText(policy), 'references'))
  })

  const podEvents = takeByDepth((events.list || []).filter((it: any) => String(it?.involvedObject?.kind ?? '') === 'Pod' && String(it?.involvedObject?.name ?? '') === name))
  podEvents.forEach((event: any, idx: number) => {
    const id = `event-${idx}`
    const reason = String(event?.reason ?? '-')
    const evtType = String(event?.type ?? '-')
    nodes.push(createNode(id, lane.eventX, 20 + idx * 118, 'Event', reason, `type=${evtType}`, String(event?.message ?? '-').slice(0, 52), 'event', undefined, undefined, undefined, 'runtime', {
      severity: evtType === 'Warning' ? 'warning' : 'normal',
      statusText: evtType
    }))
    edges.push(createEdge(`event-pod-${idx}`, id, 'pod', 'reports', 'reports'))
  })

  takeByDepth(extractPodPVCs(pod)).forEach((claimRef, idx) => {
    const claim = claimRef.name
    const pvc = (pvcs.list || []).find((it: any) => String(it?.metadata?.name ?? '') === claim && String(it?.metadata?.namespace ?? ns) === ns)
    const pvcId = `pvc-${idx}`
    nodes.push(createNode(pvcId, lane.storageX, 1520 + idx * 150, 'PVC', claim, `phase=${String(pvc?.status?.phase ?? '-')}`, `via=${claimRef.source}`, 'storage', 'PVC', ns, claim, 'storage', getPVCSeverity(pvc)))
    edges.push(createEdge(`pod-pvc-${idx}`, 'pod', pvcId, claimRef.source, 'mounts'))
    const pvName = String(pvc?.spec?.volumeName ?? '')
    const storageClassName = String(pvc?.spec?.storageClassName ?? '')
    const pv = (pvs.list || []).find((it: any) => String(it?.metadata?.name ?? '') === pvName)
    if (pv) {
      const pvId = `${pvcId}-pv`
      nodes.push(createNode(pvId, lane.pvX, 1500 + idx * 150, 'PV', pvName, `phase=${String(pv?.status?.phase ?? '-')}`, `capacity=${String(pv?.spec?.capacity?.storage ?? '-')}`, 'storage', 'PV', undefined, pvName, 'storage', {
        severity: String(pv?.status?.phase ?? '') === 'Available' ? 'warning' : 'normal',
        statusText: String(pv?.status?.phase ?? 'Bound')
      }))
      edges.push(createEdge(`${pvcId}-pv`, pvcId, pvId, 'bound', 'mounts'))
      const attachments = takeByDepth((volumeAttachments.list || []).filter((it: any) => String(it?.spec?.source?.persistentVolumeName ?? '') === pvName))
      attachments.forEach((it: any, aIdx: number) => {
        const vaId = `${pvId}-va-${aIdx}`
        nodes.push(createNode(vaId, lane.attachmentX, 1500 + idx * 150 + aIdx * 98, 'VolumeAttachment', String(it?.metadata?.name ?? '-'), `attached=${it?.status?.attached === true ? 'yes' : 'no'}`, `node=${String(it?.spec?.nodeName ?? '-')}`, 'storage', 'VolumeAttachment', undefined, String(it?.metadata?.name ?? '-'), 'storage', getVolumeAttachmentSeverity(it)))
        edges.push(createEdge(`${pvId}-va-edge-${aIdx}`, pvId, vaId, 'attach', 'mounts'))
        const attachedNodeName = String(it?.spec?.nodeName ?? '').trim()
        if (attachedNodeName) {
          const attachedNodeId = attachedNodeName === nodeName ? 'node' : `attached-node-${aIdx}`
          if (!nodes.some((item) => item.id === attachedNodeId)) {
            nodes.push(createNode(attachedNodeId, lane.attachmentX + 340, 1500 + idx * 150 + aIdx * 98, 'Node', attachedNodeName, 'volume attached node', '', 'infra', 'Node', undefined, attachedNodeName, 'runtime'))
          }
          edges.push(createEdge(`${vaId}-${attachedNodeId}`, vaId, attachedNodeId, 'mounted on', 'mounts'))
        }
      })
    }
    if (storageClassName) {
      const sc = (storageClasses.list || []).find((it: any) => String(it?.metadata?.name ?? '') === storageClassName)
      const scId = `${pvcId}-sc`
      nodes.push(createNode(scId, lane.pvX, 1600 + idx * 150, 'StorageClass', storageClassName, `mode=${String(sc?.volumeBindingMode ?? '-')}`, `reclaim=${String(sc?.reclaimPolicy ?? '-')}`, 'storage', 'StorageClass', undefined, storageClassName, 'storage'))
      edges.push(createEdge(`${pvcId}-sc`, pvcId, scId, 'class', 'references'))
    }
  })

  takeByDepth(extractPodConfigMaps(pod)).forEach((configRef, idx) => {
    const cmName = configRef.name
    const cm = (configMaps.list || []).find((it: any) => String(it?.metadata?.name ?? '') === cmName)
    const id = `cm-${idx}`
    nodes.push(createNode(id, lane.configX, 1520 + idx * 120, 'ConfigMap', cmName, `data=${Object.keys(cm?.data ?? {}).length}`, `via=${configRef.source}`, 'service', 'ConfigMap', ns, cmName, 'config'))
    edges.push(createEdge(`pod-cm-${idx}`, 'pod', id, configRef.source, 'references'))
  })

  takeByDepth(extractPodSecrets(pod)).forEach((secretRef, idx) => {
    const secretName = secretRef.name
    const secret = (secrets.list || []).find((it: any) => String(it?.metadata?.name ?? '') === secretName)
    const id = `secret-${idx}`
    nodes.push(createNode(id, lane.secretX, 1520 + idx * 120, 'Secret', secretName, `type=${String(secret?.type ?? '-')}`, `via=${secretRef.source}`, 'storage', 'Secret', ns, secretName, 'config'))
    edges.push(createEdge(`pod-secret-${idx}`, 'pod', id, secretRef.source, 'references'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNamespacePodTopology(cluster: number, ns: string): Promise<TopologyGraph> {
  const [pods, replicaSets, workloads, services, networkPolicies, pvcs, configMaps, secrets, serviceAccounts] = await Promise.all([
    k8sApi.listPods(cluster, { namespace: ns }),
    k8sApi.listReplicaSets(cluster, { namespace: ns }),
    k8sApi.listWorkloads(cluster, { namespace: ns }),
    k8sApi.listServices(cluster, { namespace: ns }),
    k8sApi.listNetworkPolicies(cluster, { namespace: ns }),
    k8sApi.listPVCs(cluster, { namespace: ns }),
    k8sApi.listConfigMaps(cluster, { namespace: ns }),
    k8sApi.listSecrets(cluster, { namespace: ns }),
    k8sApi.listServiceAccounts(cluster, { namespace: ns })
  ])
  const nodes: TopologyNode[] = [createNode('ns', 0, 200, 'Namespace', ns, 'pod control topology', 'scope=namespace', 'infra', 'Namespace', undefined, ns, 'core', { emphasis: 'core' })]
  const edges: TopologyEdge[] = []
  const nodeIds = new Set(nodes.map((node) => node.id))
  const edgeIds = new Set<string>()
  const podList = takeByDepth(pods.list || [])
  const replicaSetByName = new Map((replicaSets.list || []).map((item: any) => [String(item?.metadata?.name ?? ''), item]))
  const workloadByKey = new Map((workloads.list || []).map((item: any) => [`${String(item?.kind ?? '')}:${String(item?.metadata?.name ?? '')}`, item]))
  const serviceAccountByName = new Map((serviceAccounts.list || []).map((item: any) => [String(item?.metadata?.name ?? ''), item]))
  const pvcByName = new Map((pvcs.list || []).map((item: any) => [String(item?.metadata?.name ?? ''), item]))
  const configMapByName = new Map((configMaps.list || []).map((item: any) => [String(item?.metadata?.name ?? ''), item]))
  const secretByName = new Map((secrets.list || []).map((item: any) => [String(item?.metadata?.name ?? ''), item]))

  type ControllerInfo = { kind: string; name: string; item?: any; standalone?: boolean }
  type PodGroup = { key: string; controller: ControllerInfo; pods: any[]; podGroupId: string }

  function resolvePodController(pod: any): ControllerInfo {
    const owner = (Array.isArray(pod?.metadata?.ownerReferences) ? pod.metadata.ownerReferences : [])[0]
    if (!owner) return { kind: 'Pod', name: '独立 Pods', standalone: true }

    const ownerKind = String(owner?.kind ?? '')
    const ownerName = String(owner?.name ?? '')
    if (ownerKind === 'ReplicaSet') {
      const rs = replicaSetByName.get(ownerName)
      const rsOwner = (Array.isArray(rs?.metadata?.ownerReferences) ? rs.metadata.ownerReferences : [])[0]
      if (rsOwner && String(rsOwner?.kind ?? '') === 'Deployment') {
        const depName = String(rsOwner?.name ?? '')
        return { kind: 'Deployment', name: depName, item: workloadByKey.get(`Deployment:${depName}`) }
      }
      return { kind: 'ReplicaSet', name: ownerName, item: rs }
    }

    if (['Deployment', 'StatefulSet', 'DaemonSet', 'Job', 'CronJob'].includes(ownerKind)) {
      return { kind: ownerKind, name: ownerName, item: workloadByKey.get(`${ownerKind}:${ownerName}`) }
    }

    return { kind: ownerKind || 'Owner', name: ownerName || 'unknown', item: workloadByKey.get(`${ownerKind}:${ownerName}`) }
  }

  function podIsReady(pod: any) {
    const statuses: any[] = Array.isArray(pod?.status?.containerStatuses) ? pod.status.containerStatuses : []
    return String(pod?.status?.phase ?? '') === 'Running' && statuses.length > 0 && statuses.every((status) => status?.ready === true)
  }

  const podGroups = new Map<string, PodGroup>()
  podList.forEach((pod: any) => {
    const controller = resolvePodController(pod)
    const podName = String(pod?.metadata?.name ?? '-')
    const key = controller.standalone ? 'Pod:standalone' : `${controller.kind}:${controller.name}`
    const group = podGroups.get(key) ?? {
      key,
      controller,
      pods: [],
      podGroupId: stableGraphId('pods', controller.kind, controller.standalone && podList.length === 1 ? podName : controller.name)
    }
    group.pods.push(pod)
    podGroups.set(key, group)
  })

  const podGroupByPodName = new Map<string, string>()
  Array.from(podGroups.values()).forEach((group, index) => {
    const groupPods = group.pods
    const total = groupPods.length
    const ready = groupPods.filter((pod) => podIsReady(pod)).length
    const restarts = groupPods.reduce((sum, pod) => sum + getPodRestartCount(pod), 0)
    const severities = groupPods.map((pod) => getPodSeverity(pod))
    const errorCount = severities.filter((item) => item.severity === 'error').length
    const warningCount = severities.filter((item) => item.severity === 'warning').length
    const abnormalNames = groupPods
      .filter((pod, podIndex) => severities[podIndex]?.severity !== 'normal')
      .map((pod) => String(pod?.metadata?.name ?? '-'))
      .slice(0, 3)
    const nodeCount = new Set(groupPods.map((pod) => String(pod?.spec?.nodeName ?? '')).filter(Boolean)).size
    const groupSeverity: TopologySeverity = errorCount > 0 ? 'error' : warningCount > 0 ? 'warning' : 'normal'
    const statusText = errorCount > 0 ? `${errorCount} 异常` : warningCount > 0 ? `${warningCount} 待处理` : `${ready}/${total} Ready`
    const isSingleStandalone = group.controller.standalone && total === 1
    const singlePodName = isSingleStandalone ? String(groupPods[0]?.metadata?.name ?? '-') : ''

    if (!group.controller.standalone) {
      const workloadId = stableGraphId('workload', group.controller.kind, group.controller.name)
      const readyReplicas = String(group.controller.item?.status?.readyReplicas ?? group.controller.item?.status?.succeeded ?? ready)
      const replicas = String(group.controller.item?.status?.replicas ?? group.controller.item?.status?.active ?? total)
      appendNodeOnce(nodes, nodeIds, createNode(workloadId, 320, index * 170, group.controller.kind, group.controller.name, `ready=${readyReplicas}/${replicas}`, `pods=${total}`, 'workload', group.controller.kind, ns, group.controller.name, 'control', { emphasis: 'primary' }))
      appendEdgeOnce(edges, edgeIds, createEdge(`ns-${workloadId}`, 'ns', workloadId, 'runs', 'controls'))
      appendEdgeOnce(edges, edgeIds, createEdge(`${workloadId}-${group.podGroupId}`, workloadId, group.podGroupId, 'controls', 'controls'))
    } else {
      appendEdgeOnce(edges, edgeIds, createEdge(`ns-${group.podGroupId}`, 'ns', group.podGroupId, 'contains', 'controls'))
    }

    appendNodeOnce(
      nodes,
      nodeIds,
      createNode(
        group.podGroupId,
        680,
        index * 170,
        isSingleStandalone ? 'Pod' : 'Pods',
        isSingleStandalone ? singlePodName : `${group.controller.name} · Pods`,
        `${ready}/${total} Ready · restarts=${restarts}`,
        [`nodes=${nodeCount || '-'}`, abnormalNames.length ? `异常=${abnormalNames.join(', ')}` : ''].filter(Boolean).join(' · '),
        'workload',
        isSingleStandalone ? 'Pod' : undefined,
        isSingleStandalone ? ns : undefined,
        isSingleStandalone ? singlePodName : undefined,
        'core',
        { severity: groupSeverity, statusText, emphasis: 'primary' }
      )
    )

    groupPods.forEach((pod) => {
      podGroupByPodName.set(String(pod?.metadata?.name ?? ''), group.podGroupId)
    })
  })

  ;(services.list || []).forEach((svc: any) => {
    const selector = svc?.spec?.selector ?? {}
    const matchedGroupIds = new Set(
      podList
        .filter((pod: any) => matchLabels(pod?.metadata?.labels, selector))
        .map((pod: any) => podGroupByPodName.get(String(pod?.metadata?.name ?? '')))
        .filter(Boolean) as string[]
    )
    if (matchedGroupIds.size === 0) return

    const svcName = String(svc?.metadata?.name ?? '-')
    const svcId = stableGraphId('service', svcName)
    appendNodeOnce(nodes, nodeIds, createNode(svcId, 1040, 0, 'Service', svcName, `type=${String(svc?.spec?.type ?? '-')}`, `selector=${Object.keys(selector).length}`, 'service', 'Service', ns, svcName, 'network', { emphasis: 'primary' }))
    matchedGroupIds.forEach((groupId) => {
      appendEdgeOnce(edges, edgeIds, createEdge(`${svcId}-${groupId}`, svcId, groupId, 'selects', 'routes'))
    })
  })

  Array.from(podGroups.values()).forEach((group) => {
    const podGroupId = group.podGroupId
    const serviceAccountNames = new Set(group.pods.map((pod) => String(pod?.spec?.serviceAccountName ?? 'default')).filter(Boolean))
    serviceAccountNames.forEach((saName) => {
      const serviceAccount = serviceAccountByName.get(saName)
      if (!serviceAccount) return
      const saId = stableGraphId('serviceaccount', saName)
      appendNodeOnce(nodes, nodeIds, createNode(saId, 1040, 0, 'ServiceAccount', saName, `secrets=${Array.isArray(serviceAccount?.secrets) ? serviceAccount.secrets.length : 0}`, ns, 'network', 'ServiceAccount', ns, saName, 'identity'))
      appendEdgeOnce(edges, edgeIds, createEdge(`${saId}-${podGroupId}`, saId, podGroupId, 'identity', 'binds'))
    })

    const pvcRefs = new Map<string, PodPVCRef>()
    const configRefs = new Map<string, PodConfigRef>()
    const secretRefs = new Map<string, PodSecretRef>()
    group.pods.forEach((pod) => {
      extractPodPVCs(pod).forEach((ref) => pvcRefs.set(ref.name, ref))
      extractPodConfigMaps(pod).forEach((ref) => configRefs.set(ref.name, ref))
      extractPodSecrets(pod).forEach((ref) => secretRefs.set(ref.name, ref))
    })

    pvcRefs.forEach((ref, pvcName) => {
      const pvc = pvcByName.get(pvcName)
      const pvcId = stableGraphId('pvc', pvcName)
      appendNodeOnce(nodes, nodeIds, createNode(pvcId, 1380, 0, 'PVC', pvcName, `phase=${String(pvc?.status?.phase ?? '-')}`, `via=${ref.source}`, 'storage', 'PVC', ns, pvcName, 'storage', getPVCSeverity(pvc)))
      appendEdgeOnce(edges, edgeIds, createEdge(`${podGroupId}-${pvcId}`, podGroupId, pvcId, ref.source, 'mounts'))
    })

    configRefs.forEach((ref, cmName) => {
      const configMap = configMapByName.get(cmName)
      const cmId = stableGraphId('configmap', cmName)
      appendNodeOnce(nodes, nodeIds, createNode(cmId, 1380, 0, 'ConfigMap', cmName, `data=${Object.keys(configMap?.data ?? {}).length}`, `via=${ref.source}`, 'service', 'ConfigMap', ns, cmName, 'config'))
      appendEdgeOnce(edges, edgeIds, createEdge(`${podGroupId}-${cmId}`, podGroupId, cmId, ref.source, 'references'))
    })

    secretRefs.forEach((ref, secretName) => {
      const secret = secretByName.get(secretName)
      const secretId = stableGraphId('secret', secretName)
      appendNodeOnce(nodes, nodeIds, createNode(secretId, 1380, 0, 'Secret', secretName, `type=${String(secret?.type ?? '-')}`, `via=${ref.source}`, 'storage', 'Secret', ns, secretName, 'config'))
      appendEdgeOnce(edges, edgeIds, createEdge(`${podGroupId}-${secretId}`, podGroupId, secretId, ref.source, 'references'))
    })
  })

  ;(networkPolicies.list || []).forEach((policy: any) => {
    const matchedGroupIds = new Set(
      podList
        .filter((pod: any) => matchLabels(pod?.metadata?.labels, policy?.spec?.podSelector?.matchLabels ?? {}))
        .map((pod: any) => podGroupByPodName.get(String(pod?.metadata?.name ?? '')))
        .filter(Boolean) as string[]
    )
    if (matchedGroupIds.size === 0) return

    const policyName = String(policy?.metadata?.name ?? '-')
    const policyId = stableGraphId('networkpolicy', policyName)
    appendNodeOnce(nodes, nodeIds, createNode(policyId, 1040, 0, 'NetworkPolicy', policyName, `types=${Array.isArray(policy?.spec?.policyTypes) ? policy.spec.policyTypes.join(',') : '-'}`, `scope=${networkPolicyScopeText(policy)}`, 'network', 'NetworkPolicy', ns, policyName, 'network', {
      severity: Object.keys(policy?.spec?.podSelector?.matchLabels ?? {}).length === 0 ? 'warning' : 'normal',
      statusText: networkPolicyDirectionBadge(policy)
    }))
    matchedGroupIds.forEach((groupId) => {
      appendEdgeOnce(edges, edgeIds, createEdge(`${policyId}-${groupId}`, policyId, groupId, networkPolicyDirectionText(policy), 'references'))
    })
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNodeTopology(cluster: number, _ns: string, name: string): Promise<TopologyGraph> {
  const [nodesResp, csiNodes, volumeAttachments, pods] = await Promise.all([
    k8sApi.listNodes(cluster, {}),
    k8sApi.listCSINodes(cluster, {}),
    k8sApi.listVolumeAttachments(cluster, {}),
    k8sApi.listPods(cluster, {})
  ])
  const node = (nodesResp.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (!node) throw new Error(`未找到 Node ${name}`)
  const nodes: TopologyNode[] = []
  const edges: TopologyEdge[] = []
  nodes.push(createNode('node', 0, 120, 'Node', name, `roles=${Object.keys(node?.metadata?.labels ?? {}).filter((k) => k.startsWith('node-role.kubernetes.io/')).length}`, `schedulable=${node?.spec?.unschedulable === true ? 'no' : 'yes'}`, 'infra', 'Node', undefined, name))
  const csi = (csiNodes.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (csi) {
    const drivers: any[] = Array.isArray(csi?.spec?.drivers) ? csi.spec.drivers : []
    nodes.push(createNode('csinode', 320, 20, 'CSINode', name, `drivers=${drivers.length}`, `first=${String(drivers[0]?.name ?? '-')}`, 'storage', 'CSINode', undefined, name))
    edges.push(createEdge('node-csinode', 'node', 'csinode', 'storage capability'))
  }
  const attachments = (volumeAttachments.list || []).filter((it: any) => String(it?.spec?.nodeName ?? '') === name)
  attachments.forEach((it: any, idx: number) => {
    const id = `va-${idx}`
    nodes.push(createNode(id, 320, 180 + idx * 160, 'VolumeAttachment', String(it?.metadata?.name ?? '-'), `attached=${it?.status?.attached === true ? 'yes' : 'no'}`, `pv=${String(it?.spec?.source?.persistentVolumeName ?? '-')}`, 'storage', 'VolumeAttachment', undefined, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`node-va-${idx}`, 'node', id, 'mount'))
  })
  const nodePods = takeByDepth((pods.list || []).filter((it: any) => String(it?.spec?.nodeName ?? '') === name))
  nodePods.forEach((it: any, idx: number) => {
    const id = `pod-${idx}`
    nodes.push(createNode(id, 700, idx * 130, 'Pod', String(it?.metadata?.name ?? '-'), `phase=${String(it?.status?.phase ?? '-')}`, `${String(it?.metadata?.namespace ?? '-')}`, 'workload', 'Pod', String(it?.metadata?.namespace ?? ''), String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`node-pod-${idx}`, 'node', id, 'scheduled'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildIngressTopology(cluster: number, ns: string, name: string): Promise<TopologyGraph> {
  const [ingresses, services, endpointSlices, pods] = await Promise.all([
    k8sApi.listIngresses(cluster, { namespace: ns }),
    k8sApi.listServices(cluster, { namespace: ns }),
    k8sApi.listEndpointSlices(cluster, { namespace: ns }),
    k8sApi.listPods(cluster, { namespace: ns })
  ])
  const ingress = (ingresses.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (!ingress) throw new Error(`未找到 Ingress ${ns}/${name}`)
  const nodes: TopologyNode[] = []
  const edges: TopologyEdge[] = []
  nodes.push(createNode('ing', 0, 160, 'Ingress', `${ns}/${name}`, `class=${String(ingress?.spec?.ingressClassName ?? '-')}`, `rules=${Array.isArray(ingress?.spec?.rules) ? ingress.spec.rules.length : 0}`, 'network', 'Ingress', ns, name))
  const svcNames = new Set<string>()
  const rules: any[] = Array.isArray(ingress?.spec?.rules) ? ingress.spec.rules : []
  for (const rule of rules) {
    const paths: any[] = Array.isArray(rule?.http?.paths) ? rule.http.paths : []
    for (const p of paths) {
      const svcName = String(p?.backend?.service?.name ?? '').trim()
      if (svcName) svcNames.add(svcName)
    }
  }
  Array.from(svcNames).forEach((svcName, idx) => {
    const svc = (services.list || []).find((it: any) => String(it?.metadata?.name ?? '') === svcName)
    const svcId = `svc-${idx}`
    nodes.push(createNode(svcId, 320, idx * 200, 'Service', svcName, `type=${String(svc?.spec?.type ?? '-')}`, `selector=${Object.keys(svc?.spec?.selector ?? {}).length}`, 'service', 'Service', ns, svcName))
    edges.push(createEdge(`ing-svc-${idx}`, 'ing', svcId, 'routes'))
    const selector = svc?.spec?.selector ?? {}
    const slices = (endpointSlices.list || []).filter((it: any) => String(it?.metadata?.labels?.['kubernetes.io/service-name'] ?? '') === svcName)
    slices.forEach((it: any, sliceIdx: number) => {
      const sliceId = `${svcId}-slice-${sliceIdx}`
      nodes.push(createNode(sliceId, 640, idx * 200 + sliceIdx * 120, 'EndpointSlice', String(it?.metadata?.name ?? '-'), `endpoints=${Array.isArray(it?.endpoints) ? it.endpoints.length : 0}`, `type=${String(it?.addressType ?? '-')}`, 'network', 'EndpointSlice', ns, String(it?.metadata?.name ?? '-')))
      edges.push(createEdge(`${svcId}-edge-slice-${sliceIdx}`, svcId, sliceId, 'slice'))
    })
    const matchedPods = (pods.list || []).filter((it: any) => matchLabels(it?.metadata?.labels, selector))
    matchedPods.forEach((it: any, podIdx: number) => {
      const podId = `${svcId}-pod-${podIdx}`
      nodes.push(createNode(podId, 980, idx * 200 + podIdx * 110, 'Pod', String(it?.metadata?.name ?? '-'), `phase=${String(it?.status?.phase ?? '-')}`, `${String(it?.spec?.nodeName ?? '-')}`, 'workload', 'Pod', ns, String(it?.metadata?.name ?? '-')))
      edges.push(createEdge(`${svcId}-edge-pod-${podIdx}`, svcId, podId, 'selector'))
    })
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNamespaceIngressTopology(cluster: number, ns: string): Promise<TopologyGraph> {
  const [ingresses, services] = await Promise.all([
    k8sApi.listIngresses(cluster, { namespace: ns }),
    k8sApi.listServices(cluster, { namespace: ns })
  ])
  const nodes: TopologyNode[] = [createNode('ns', 0, 200, 'Namespace', ns, 'ingress traffic overview', 'scope=namespace', 'infra')]
  const edges: TopologyEdge[] = []
  const ingressList = takeByDepth(ingresses.list || [])
  ingressList.forEach((ing: any, idx: number) => {
    const ingId = `ing-${idx}`
    nodes.push(createNode(ingId, 260, idx * 170, 'Ingress', String(ing?.metadata?.name ?? '-'), `class=${String(ing?.spec?.ingressClassName ?? '-')}`, `rules=${Array.isArray(ing?.spec?.rules) ? ing.spec.rules.length : 0}`, 'network', 'Ingress', ns, String(ing?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-ing-${idx}`, 'ns', ingId, 'entry'))
    const svcNames = new Set<string>()
    const rules: any[] = Array.isArray(ing?.spec?.rules) ? ing.spec.rules : []
    for (const rule of rules) {
      const paths: any[] = Array.isArray(rule?.http?.paths) ? rule.http.paths : []
      for (const p of paths) {
        const svcName = String(p?.backend?.service?.name ?? '').trim()
        if (svcName) svcNames.add(svcName)
      }
    }
    Array.from(svcNames).forEach((svcName, svcIdx) => {
      const svcId = `svc-${svcName}`
      const svc = (services.list || []).find((it: any) => String(it?.metadata?.name ?? '') === svcName)
      if (!nodes.some((n) => n.id === svcId)) {
        nodes.push(createNode(svcId, 620, idx * 170 + svcIdx * 90, 'Service', svcName, `type=${String(svc?.spec?.type ?? '-')}`, `selector=${Object.keys(svc?.spec?.selector ?? {}).length}`, 'service', 'Service', ns, svcName))
      }
      edges.push(createEdge(`${ingId}-${svcId}`, ingId, svcId, 'routes'))
    })
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildConfigTopology(cluster: number, ns: string, selection = ''): Promise<TopologyGraph> {
  const [configMaps, secrets, pods] = await Promise.all([
    k8sApi.listConfigMaps(cluster, { namespace: ns }),
    k8sApi.listSecrets(cluster, { namespace: ns }),
    k8sApi.listPods(cluster, { namespace: ns })
  ])
  const nodes: TopologyNode[] = [createNode('ns', 0, 200, 'Namespace', ns, 'config dependency overview', 'scope=namespace', 'infra')]
  const edges: TopologyEdge[] = []
  const podList = pods.list || []
  const cmList = configMaps.list || []
  const secretList = secrets.list || []
  const selectedConfig = parseConfigResourceSelection(selection)

  if (selectedConfig.name) {
    const entries: Array<{ kind: ConfigResourceKind; item: any }> = []
    if (!selectedConfig.kind || selectedConfig.kind === 'ConfigMap') {
      const configMap = cmList.find((item: any) => String(item?.metadata?.name ?? '') === selectedConfig.name)
      if (configMap) entries.push({ kind: 'ConfigMap', item: configMap })
    }
    if (!selectedConfig.kind || selectedConfig.kind === 'Secret') {
      const secret = secretList.find((item: any) => String(item?.metadata?.name ?? '') === selectedConfig.name)
      if (secret) entries.push({ kind: 'Secret', item: secret })
    }
    if (entries.length === 0) throw new Error(`未找到配置资源 ${ns}/${selectedConfig.name}`)

    const podNodeMap = new Map<string, string>()
    entries.forEach((entry, idx) => {
      const itemName = String(entry.item?.metadata?.name ?? '-')
      const nodeId = `${entry.kind.toLowerCase()}-${idx}`
      const y = 120 + idx * 260

      if (entry.kind === 'ConfigMap') {
        nodes.push(createNode(nodeId, 280, y, 'ConfigMap', itemName, `data=${Object.keys(entry.item?.data ?? {}).length}`, ns, 'service', 'ConfigMap', ns, itemName))
        edges.push(createEdge(`ns-${nodeId}`, 'ns', nodeId, 'config'))
      } else {
        nodes.push(createNode(nodeId, 280, y, 'Secret', itemName, `type=${String(entry.item?.type ?? '-')}`, ns, 'storage', 'Secret', ns, itemName))
        edges.push(createEdge(`ns-${nodeId}`, 'ns', nodeId, 'secret'))
      }

      const relatedPods = podList.filter((pod: any) => (entry.kind === 'ConfigMap' ? podUsesConfigMap(pod, itemName) : podUsesSecret(pod, itemName)))
      relatedPods.forEach((pod: any, podIdx: number) => {
        const podKey = `${String(pod?.metadata?.namespace ?? '')}/${String(pod?.metadata?.name ?? '')}`
        let podId = podNodeMap.get(podKey)
        if (!podId) {
          podId = `pod-${podNodeMap.size}`
          podNodeMap.set(podKey, podId)
          nodes.push(
            createNode(
              podId,
              650,
              y + podIdx * 96,
              'Pod',
              String(pod?.metadata?.name ?? '-'),
              `phase=${String(pod?.status?.phase ?? '-')}`,
              `${String(pod?.spec?.nodeName ?? '-')}`,
              'workload',
              'Pod',
              ns,
              String(pod?.metadata?.name ?? '-')
            )
          )
        }
        edges.push(createEdge(`${nodeId}-${podId}`, nodeId, podId, 'mount/env'))
      })
    })

    return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
  }

  const nodeIds = new Set(nodes.map((node) => node.id))
  const edgeIds = new Set<string>()
  const podNodeByKey = new Map<string, string>()
  const ensurePodNode = (pod: any, x: number, y: number) => {
    const podNamespace = String(pod?.metadata?.namespace ?? ns)
    const podName = String(pod?.metadata?.name ?? '-')
    const podKey = `${podNamespace}/${podName}`
    const existing = podNodeByKey.get(podKey)
    if (existing) return existing

    const podId = stableGraphId('pod', podNamespace, podName)
    podNodeByKey.set(podKey, podId)
    appendNodeOnce(nodes, nodeIds, createNode(podId, x, y, 'Pod', podName, `phase=${String(pod?.status?.phase ?? '-')}`, `${String(pod?.spec?.nodeName ?? '-')}`, 'workload', 'Pod', podNamespace, podName, undefined, getPodSeverity(pod)))
    return podId
  }

  cmList.forEach((item: any, idx: number) => {
    const id = stableGraphId('configmap', String(item?.metadata?.name ?? '-'))
    const name = String(item?.metadata?.name ?? '-')
    appendNodeOnce(nodes, nodeIds, createNode(id, 280, idx * 110, 'ConfigMap', name, `data=${Object.keys(item?.data ?? {}).length}`, ns, 'service', 'ConfigMap', ns, name))
    appendEdgeOnce(edges, edgeIds, createEdge(`ns-${id}`, 'ns', id, 'config'))
    podList.filter((pod: any) => podUsesConfigMap(pod, name)).forEach((pod: any, podIdx: number) => {
      const podId = ensurePodNode(pod, 650, idx * 110 + podIdx * 86)
      appendEdgeOnce(edges, edgeIds, createEdge(`${id}-${podId}`, id, podId, 'mount/env'))
    })
  })
  secretList.forEach((item: any, idx: number) => {
    const name = String(item?.metadata?.name ?? '-')
    const id = stableGraphId('secret', name)
    appendNodeOnce(nodes, nodeIds, createNode(id, 280, 420 + idx * 110, 'Secret', name, `type=${String(item?.type ?? '-')}`, ns, 'storage', 'Secret', ns, name))
    appendEdgeOnce(edges, edgeIds, createEdge(`ns-${id}`, 'ns', id, 'secret'))
    podList.filter((pod: any) => podUsesSecret(pod, name)).forEach((pod: any, podIdx: number) => {
      const podId = ensurePodNode(pod, 650, 420 + idx * 110 + podIdx * 86)
      appendEdgeOnce(edges, edgeIds, createEdge(`${id}-${podId}`, id, podId, 'mount/env'))
    })
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildPVTopology(cluster: number, _ns: string, name: string): Promise<TopologyGraph> {
  const [pvs, storageClasses, volumeAttachments, pvcs, pods] = await Promise.all([
    k8sApi.listPVs(cluster, {}),
    k8sApi.listStorageClasses(cluster, {}),
    k8sApi.listVolumeAttachments(cluster, {}),
    k8sApi.listPVCs(cluster, {}),
    k8sApi.listPods(cluster, {})
  ])

  const pv = (pvs.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (!pv) throw new Error(`未找到 PV ${name}`)

  const nodes: TopologyNode[] = [
    createNode(
      'pv',
      0,
      220,
      'PV',
      name,
      `phase=${String(pv?.status?.phase ?? '-')}`,
      `capacity=${String(pv?.spec?.capacity?.storage ?? '-')}`,
      'storage',
      'PV',
      undefined,
      name,
      'core',
      {
        severity: String(pv?.status?.phase ?? '') === 'Available' ? 'warning' : 'normal',
        statusText: String(pv?.status?.phase ?? 'Bound'),
        emphasis: 'core'
      }
    )
  ]
  const edges: TopologyEdge[] = []
  const nodeIds = new Set(nodes.map((node) => node.id))
  const edgeIds = new Set<string>()
  const pvName = String(pv?.metadata?.name ?? '-')
  const scName = String(pv?.spec?.storageClassName ?? '').trim()
  const claimNamespace = String(pv?.spec?.claimRef?.namespace ?? '').trim()
  const claimName = String(pv?.spec?.claimRef?.name ?? '').trim()

  if (scName) {
    const sc = (storageClasses.list || []).find((it: any) => String(it?.metadata?.name ?? '') === scName)
    appendNodeOnce(nodes, nodeIds, createNode('sc', 320, 40, 'StorageClass', scName, `mode=${String(sc?.volumeBindingMode ?? '-')}`, `reclaim=${String(sc?.reclaimPolicy ?? '-')}`, 'storage', 'StorageClass', undefined, scName, 'storage'))
    appendEdgeOnce(edges, edgeIds, createEdge('pv-sc', 'pv', 'sc', 'class'))
  }

  const pvc = (pvcs.list || []).find((it: any) => {
    const itemNamespace = String(it?.metadata?.namespace ?? '').trim()
    const itemName = String(it?.metadata?.name ?? '').trim()
    const volumeName = String(it?.spec?.volumeName ?? '').trim()
    if (claimNamespace && claimName) return itemNamespace === claimNamespace && itemName === claimName
    return volumeName === pvName
  })

  if (pvc) {
    const pvcNamespace = String(pvc?.metadata?.namespace ?? '-')
    const pvcName = String(pvc?.metadata?.name ?? '-')
    appendNodeOnce(nodes, nodeIds, createNode('pvc', 320, 260, 'PVC', `${pvcNamespace}/${pvcName}`, `phase=${String(pvc?.status?.phase ?? '-')}`, `request=${String(pvc?.spec?.resources?.requests?.storage ?? '-')}`, 'storage', 'PVC', pvcNamespace, pvcName, 'storage', getPVCSeverity(pvc)))
    appendEdgeOnce(edges, edgeIds, createEdge('pvc-pv', 'pvc', 'pv', 'bound'))

    const relatedPods = takeByDepth((pods.list || []).filter((it: any) => String(it?.metadata?.namespace ?? '') === pvcNamespace && podUsesPvc(it, pvcName)))
    relatedPods.forEach((pod: any, idx: number) => {
      const podId = `pod-${idx}`
      appendNodeOnce(nodes, nodeIds, createNode(podId, 700, 220 + idx * 130, 'Pod', String(pod?.metadata?.name ?? '-'), `phase=${String(pod?.status?.phase ?? '-')}`, `${String(pod?.spec?.nodeName ?? '-')}`, 'workload', 'Pod', pvcNamespace, String(pod?.metadata?.name ?? '-'), 'runtime', getPodSeverity(pod)))
      appendEdgeOnce(edges, edgeIds, createEdge(`pod-pvc-${idx}`, podId, 'pvc', 'volume'))
    })
  }

  const vas = takeByDepth((volumeAttachments.list || []).filter((it: any) => String(it?.spec?.source?.persistentVolumeName ?? '') === pvName))
  vas.forEach((it: any, idx: number) => {
    const vaId = `va-${idx}`
    const nodeName = String(it?.spec?.nodeName ?? '-')
    appendNodeOnce(nodes, nodeIds, createNode(vaId, 700, 40 + idx * 120, 'VolumeAttachment', String(it?.metadata?.name ?? '-'), `attached=${it?.status?.attached === true ? 'yes' : 'no'}`, `node=${nodeName}`, 'storage', 'VolumeAttachment', undefined, String(it?.metadata?.name ?? '-'), 'storage', getVolumeAttachmentSeverity(it)))
    appendEdgeOnce(edges, edgeIds, createEdge(`pv-va-${idx}`, 'pv', vaId, 'attach'))
    if (nodeName) {
      const nodeId = `node-${nodeName}`
      appendNodeOnce(nodes, nodeIds, createNode(nodeId, 1040, 40 + idx * 120, 'Node', nodeName, 'attached node', '', 'infra', 'Node', undefined, nodeName, 'runtime'))
      appendEdgeOnce(edges, edgeIds, createEdge(`va-node-${idx}`, vaId, nodeId, 'mounted on'))
    }
  })

  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildServiceAccountTopology(cluster: number, ns: string, name: string): Promise<TopologyGraph> {
  const [serviceAccounts, roleBindings, clusterRoleBindings, roles, clusterRoles] = await Promise.all([
    k8sApi.listServiceAccounts(cluster, { namespace: ns }),
    k8sApi.listRoleBindings(cluster, { namespace: ns }),
    k8sApi.listClusterRoleBindings(cluster, {}),
    k8sApi.listRoles(cluster, { namespace: ns }),
    k8sApi.listClusterRoles(cluster, {})
  ])
  const sa = (serviceAccounts.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (!sa) throw new Error(`未找到 ServiceAccount ${ns}/${name}`)
  const nodes: TopologyNode[] = []
  const edges: TopologyEdge[] = []
  nodes.push(createNode('sa', 0, 200, 'ServiceAccount', `${ns}/${name}`, `secrets=${Array.isArray(sa?.secrets) ? sa.secrets.length : 0}`, ns, 'service', 'ServiceAccount', ns, name))
  const localBindings = (roleBindings.list || []).filter((it: any) => Array.isArray(it?.subjects) && it.subjects.some((s: any) => String(s?.kind ?? '') === 'ServiceAccount' && String(s?.name ?? '') === name && String(s?.namespace ?? ns) === ns))
  localBindings.forEach((it: any, idx: number) => {
    const rbId = `rb-${idx}`
    const roleRefName = String(it?.roleRef?.name ?? '-')
    nodes.push(createNode(rbId, layoutMap.serviceaccount[1], idx * 160, 'RoleBinding', String(it?.metadata?.name ?? '-'), `roleRef=${roleRefName}`, ns, 'network', 'RoleBinding', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`sa-rb-${idx}`, 'sa', rbId, 'bound'))
    const role = (roles.list || []).find((r: any) => String(r?.metadata?.name ?? '') === roleRefName)
    if (role) {
      const roleId = `role-${idx}`
      nodes.push(createNode(roleId, layoutMap.serviceaccount[2], idx * 160, 'Role', roleRefName, `rules=${Array.isArray(role?.rules) ? role.rules.length : 0}`, ns, 'network', 'Role', ns, roleRefName))
      edges.push(createEdge(`rb-role-${idx}`, rbId, roleId, 'ref'))
    }
  })
  const globalBindings = (clusterRoleBindings.list || []).filter((it: any) => Array.isArray(it?.subjects) && it.subjects.some((s: any) => String(s?.kind ?? '') === 'ServiceAccount' && String(s?.name ?? '') === name && String(s?.namespace ?? '') === ns))
  globalBindings.forEach((it: any, idx: number) => {
    const crbId = `crb-${idx}`
    const roleRefName = String(it?.roleRef?.name ?? '-')
    const y = 420 + idx * 160
    nodes.push(createNode(crbId, layoutMap.serviceaccount[1], y, 'ClusterRoleBinding', String(it?.metadata?.name ?? '-'), `roleRef=${roleRefName}`, 'cluster scope', 'network', 'ClusterRoleBinding', undefined, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`sa-crb-${idx}`, 'sa', crbId, 'bound'))
    const role = (clusterRoles.list || []).find((r: any) => String(r?.metadata?.name ?? '') === roleRefName)
    if (role) {
      const crId = `cr-${idx}`
      nodes.push(createNode(crId, layoutMap.serviceaccount[2], y, 'ClusterRole', roleRefName, `rules=${Array.isArray(role?.rules) ? role.rules.length : 0}`, 'cluster scope', 'network', 'ClusterRole', undefined, roleRefName))
      edges.push(createEdge(`crb-cr-${idx}`, crbId, crId, 'ref'))
    }
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNamespaceServiceAccountTopology(cluster: number, ns: string): Promise<TopologyGraph> {
  const serviceAccounts = await k8sApi.listServiceAccounts(cluster, { namespace: ns })
  const nodes: TopologyNode[] = [createNode('ns', 0, 200, 'Namespace', ns, 'service account authorization overview', 'scope=namespace', 'infra')]
  const edges: TopologyEdge[] = []
  ;(serviceAccounts.list || []).forEach((it: any, idx: number) => {
    const saId = `sa-${idx}`
    nodes.push(createNode(saId, layoutMap.serviceaccount[1], idx * 120, 'ServiceAccount', String(it?.metadata?.name ?? '-'), `secrets=${Array.isArray(it?.secrets) ? it.secrets.length : 0}`, ns, 'service', 'ServiceAccount', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-sa-${idx}`, 'ns', saId, 'contains'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildDeploymentTopology(cluster: number, ns: string, name: string): Promise<TopologyGraph> {
  const [workloads, replicaSets, pods, configMaps, secrets, pvcs, services, ingresses] = await Promise.all([
    k8sApi.listWorkloads(cluster, { namespace: ns }),
    k8sApi.listReplicaSets(cluster, { namespace: ns }),
    k8sApi.listPods(cluster, { namespace: ns }),
    k8sApi.listConfigMaps(cluster, { namespace: ns }),
    k8sApi.listSecrets(cluster, { namespace: ns }),
    k8sApi.listPVCs(cluster, { namespace: ns }),
    k8sApi.listServices(cluster, { namespace: ns }),
    k8sApi.listIngresses(cluster, { namespace: ns })
  ])
  const deployment = (workloads.list || []).find((it: any) => String(it?.kind ?? '') === 'Deployment' && String(it?.metadata?.name ?? '') === name)
  if (!deployment) throw new Error(`未找到 Deployment ${ns}/${name}`)
  const nodes: TopologyNode[] = []
  const edges: TopologyEdge[] = []
  nodes.push(createNode('dep', 0, 220, 'Deployment', `${ns}/${name}`, `ready=${String(deployment?.status?.readyReplicas ?? 0)}/${String(deployment?.status?.replicas ?? 0)}`, ns, 'workload', 'Deployment', ns, name))
  const relatedReplicaSets = (replicaSets.list || []).filter((it: any) => Array.isArray(it?.metadata?.ownerReferences) && it.metadata.ownerReferences.some((owner: any) => String(owner?.kind ?? '') === 'Deployment' && String(owner?.name ?? '') === name))
  relatedReplicaSets.forEach((it: any, idx: number) => {
    const rsId = `rs-${idx}`
    const y = idx * 150
    nodes.push(createNode(rsId, layoutMap.deployment[1], y, 'ReplicaSet', String(it?.metadata?.name ?? '-'), `ready=${String(it?.status?.readyReplicas ?? 0)}/${String(it?.status?.replicas ?? 0)}`, ns, 'workload', 'ReplicaSet', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`dep-rs-${idx}`, 'dep', rsId, 'owns'))
  })
  const selector = deployment?.spec?.selector?.matchLabels ?? {}
  const relatedPods = (pods.list || []).filter((it: any) => matchLabels(it?.metadata?.labels, selector))
  relatedPods.forEach((it: any, idx: number) => {
    const podId = `pod-${idx}`
    nodes.push(createNode(podId, layoutMap.deployment[2], idx * 120, 'Pod', String(it?.metadata?.name ?? '-'), `phase=${String(it?.status?.phase ?? '-')}`, `${String(it?.spec?.nodeName ?? '-')}`, 'workload', 'Pod', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`dep-pod-${idx}`, 'dep', podId, 'selector'))
  })
  const templateSpec = deployment?.spec?.template?.spec ?? {}
  const depConfigMaps = new Set<string>()
  const depSecrets = new Set<string>()
  const depPVCs = new Set<string>()
  ;(Array.isArray(templateSpec?.volumes) ? templateSpec.volumes : []).forEach((volume: any) => {
    const cm = String(volume?.configMap?.name ?? '').trim()
    const secret = String(volume?.secret?.secretName ?? '').trim()
    const pvc = String(volume?.persistentVolumeClaim?.claimName ?? '').trim()
    if (cm) depConfigMaps.add(cm)
    if (secret) depSecrets.add(secret)
    if (pvc) depPVCs.add(pvc)
  })
  const containers = [...(Array.isArray(templateSpec?.containers) ? templateSpec.containers : []), ...(Array.isArray(templateSpec?.initContainers) ? templateSpec.initContainers : [])]
  containers.forEach((container: any) => {
    const envList: any[] = Array.isArray(container?.env) ? container.env : []
    const envFromList: any[] = Array.isArray(container?.envFrom) ? container.envFrom : []
    envList.forEach((env: any) => {
      const cm = String(env?.valueFrom?.configMapKeyRef?.name ?? '').trim()
      const secret = String(env?.valueFrom?.secretKeyRef?.name ?? '').trim()
      if (cm) depConfigMaps.add(cm)
      if (secret) depSecrets.add(secret)
    })
    envFromList.forEach((envFrom: any) => {
      const cm = String(envFrom?.configMapRef?.name ?? '').trim()
      const secret = String(envFrom?.secretRef?.name ?? '').trim()
      if (cm) depConfigMaps.add(cm)
      if (secret) depSecrets.add(secret)
    })
  })
  Array.from(depConfigMaps).forEach((cmName, idx) => {
    const cm = (configMaps.list || []).find((it: any) => String(it?.metadata?.name ?? '') === cmName)
    const id = `cm-${idx}`
    nodes.push(createNode(id, layoutMap.deployment[3], idx * 120, 'ConfigMap', cmName, `data=${Object.keys(cm?.data ?? {}).length}`, ns, 'service', 'ConfigMap', ns, cmName))
    edges.push(createEdge(`dep-cm-${idx}`, 'dep', id, 'config'))
  })
  Array.from(depSecrets).forEach((secretName, idx) => {
    const secret = (secrets.list || []).find((it: any) => String(it?.metadata?.name ?? '') === secretName)
    const id = `secret-${idx}`
    nodes.push(createNode(id, layoutMap.deployment[3], 320 + idx * 120, 'Secret', secretName, `type=${String(secret?.type ?? '-')}`, ns, 'storage', 'Secret', ns, secretName))
    edges.push(createEdge(`dep-secret-${idx}`, 'dep', id, 'secret'))
  })
  Array.from(depPVCs).forEach((pvcName, idx) => {
    const pvc = (pvcs.list || []).find((it: any) => String(it?.metadata?.name ?? '') === pvcName)
    const id = `pvc-${idx}`
    nodes.push(createNode(id, layoutMap.deployment[3], 620 + idx * 120, 'PVC', pvcName, `phase=${String(pvc?.status?.phase ?? '-')}`, ns, 'storage', 'PVC', ns, pvcName))
    edges.push(createEdge(`dep-pvc-${idx}`, 'dep', id, 'storage'))
  })
  const relatedServices = takeByDepth((services.list || []).filter((svc: any) => {
    const serviceSelector = svc?.spec?.selector ?? {}
    return Object.keys(serviceSelector).length > 0 && relatedPods.some((pod: any) => matchLabels(pod?.metadata?.labels, serviceSelector))
  }))
  relatedServices.forEach((svc: any, idx: number) => {
    const svcId = `svc-${idx}`
    const svcName = String(svc?.metadata?.name ?? '-')
    nodes.push(createNode(svcId, layoutMap.deployment[3], 920 + idx * 120, 'Service', svcName, `type=${String(svc?.spec?.type ?? '-')}`, `selector=${Object.keys(svc?.spec?.selector ?? {}).length}`, 'service', 'Service', ns, svcName, 'network'))
    edges.push(createEdge(`svc-dep-${idx}`, svcId, 'dep', 'selects', 'routes'))

    takeByDepth((ingresses.list || []).filter((ing: any) => ingressUsesServiceName(ing, svcName))).forEach((ing: any, ingIdx: number) => {
      const ingId = `ing-${idx}-${ingIdx}`
      nodes.push(createNode(ingId, layoutMap.deployment[2], 920 + idx * 120 + ingIdx * 92, 'Ingress', String(ing?.metadata?.name ?? '-'), `class=${String(ing?.spec?.ingressClassName ?? '-')}`, ns, 'network', 'Ingress', ns, String(ing?.metadata?.name ?? '-'), 'network'))
      edges.push(createEdge(`ing-svc-${idx}-${ingIdx}`, ingId, svcId, 'routes', 'routes'))
    })
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNamespaceDeploymentTopology(cluster: number, ns: string): Promise<TopologyGraph> {
  const workloads = await k8sApi.listWorkloads(cluster, { namespace: ns })
  const nodes: TopologyNode[] = [createNode('ns', 0, 200, 'Namespace', ns, 'deployment overview', 'scope=namespace', 'infra')]
  const edges: TopologyEdge[] = []
  ;(workloads.list || []).filter((it: any) => String(it?.kind ?? '') === 'Deployment').forEach((it: any, idx: number) => {
    const id = `dep-${idx}`
    nodes.push(createNode(id, layoutMap.deployment[1], idx * 140, 'Deployment', String(it?.metadata?.name ?? '-'), `ready=${String(it?.status?.readyReplicas ?? 0)}/${String(it?.status?.replicas ?? 0)}`, ns, 'workload', 'Deployment', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-dep-${idx}`, 'ns', id, 'contains'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNodeStorageTopology(cluster: number, _ns: string, name: string): Promise<TopologyGraph> {
  const [pods, pvcs, pvs] = await Promise.all([
    k8sApi.listPods(cluster, {}),
    k8sApi.listPVCs(cluster, {}),
    k8sApi.listPVs(cluster, {})
  ])
  const nodes: TopologyNode[] = [createNode('node', 0, 220, 'Node', name, 'storage usage view', '', 'infra', 'Node', undefined, name)]
  const edges: TopologyEdge[] = []
  const nodePods = takeByDepth((pods.list || []).filter((it: any) => String(it?.spec?.nodeName ?? '') === name))
  nodePods.forEach((pod: any, idx: number) => {
    const podId = `pod-${idx}`
    const ns = String(pod?.metadata?.namespace ?? '')
    nodes.push(createNode(podId, layoutMap['node-storage'][1], idx * 150, 'Pod', String(pod?.metadata?.name ?? '-'), `phase=${String(pod?.status?.phase ?? '-')}`, ns, 'workload', 'Pod', ns, String(pod?.metadata?.name ?? '-')))
    edges.push(createEdge(`node-pod-${idx}`, 'node', podId, 'scheduled'))
    const claims = extractPodPVCs(pod)
    claims.forEach((claimRef, pvcIdx) => {
      const claim = claimRef.name
      const pvc = (pvcs.list || []).find((it: any) => String(it?.metadata?.name ?? '') === claim && String(it?.metadata?.namespace ?? '') === ns)
      if (!pvc) return
      const pvcId = `${podId}-pvc-${pvcIdx}`
      nodes.push(createNode(pvcId, layoutMap['node-storage'][2], idx * 150 + pvcIdx * 90, 'PVC', claim, `phase=${String(pvc?.status?.phase ?? '-')}`, `source=${claimRef.source}`, 'storage', 'PVC', ns, claim))
      edges.push(createEdge(`${podId}-${pvcId}`, podId, pvcId, 'mount'))
      const pvName = String(pvc?.spec?.volumeName ?? '')
      const pv = (pvs.list || []).find((it: any) => String(it?.metadata?.name ?? '') === pvName)
      if (pv) {
        const pvId = `${pvcId}-pv`
        nodes.push(createNode(pvId, layoutMap['node-storage'][3], idx * 150 + pvcIdx * 90, 'PV', pvName, `phase=${String(pv?.status?.phase ?? '-')}`, `capacity=${String(pv?.spec?.capacity?.storage ?? '-')}`, 'storage', 'PV', undefined, pvName))
        edges.push(createEdge(`${pvcId}-${pvId}`, pvcId, pvId, 'bound'))
      }
    })
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNetworkPolicyTopology(cluster: number, ns: string, name: string): Promise<TopologyGraph> {
  const [policies, pods] = await Promise.all([
    k8sApi.listNetworkPolicies(cluster, { namespace: ns }),
    k8sApi.listPods(cluster, { namespace: ns })
  ])
  const policy = (policies.list || []).find((it: any) => String(it?.metadata?.name ?? '') === name)
  if (!policy) throw new Error(`未找到 NetworkPolicy ${ns}/${name}`)
  const nodes: TopologyNode[] = [createNode('ns', 0, 220, 'Namespace', ns, 'network policy scope', '', 'infra'), createNode('np', layoutMap.networkpolicy[1], 220, 'NetworkPolicy', name, `types=${Array.isArray(policy?.spec?.policyTypes) ? policy.spec.policyTypes.join(',') : '-'}`, ns, 'network', 'NetworkPolicy', ns, name)]
  const edges: TopologyEdge[] = [createEdge('ns-np', 'ns', 'np', 'contains')]
  const selector = policy?.spec?.podSelector?.matchLabels ?? {}
  const matchedPods = takeByDepth((pods.list || []).filter((it: any) => matchLabels(it?.metadata?.labels, selector)))
  matchedPods.forEach((pod: any, idx: number) => {
    const id = `pod-${idx}`
    nodes.push(createNode(id, layoutMap.networkpolicy[2], idx * 120, 'Pod', String(pod?.metadata?.name ?? '-'), `phase=${String(pod?.status?.phase ?? '-')}`, ns, 'workload', 'Pod', ns, String(pod?.metadata?.name ?? '-')))
    edges.push(createEdge(`np-pod-${idx}`, 'np', id, 'selects'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNamespaceOverviewTopology(cluster: number, ns: string): Promise<TopologyGraph> {
  const [workloads, resourceQuotas, limitRanges, services, ingresses] = await Promise.all([
    k8sApi.listWorkloads(cluster, { namespace: ns }),
    k8sApi.listResourceQuotas(cluster, { namespace: ns }),
    k8sApi.listLimitRanges(cluster, { namespace: ns }),
    k8sApi.listServices(cluster, { namespace: ns }),
    k8sApi.listIngresses(cluster, { namespace: ns })
  ])
  const nodes: TopologyNode[] = [createNode('ns', 0, 220, 'Namespace', ns, 'governance overview', 'scope=namespace', 'infra')]
  const edges: TopologyEdge[] = []
  ;(resourceQuotas.list || []).forEach((it: any, idx: number) => {
    const id = `rq-${idx}`
    nodes.push(createNode(id, layoutMap.namespace[1], idx * 120, 'ResourceQuota', String(it?.metadata?.name ?? '-'), `hard=${Object.keys(it?.spec?.hard ?? {}).length}`, ns, 'storage', 'ResourceQuota', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-rq-${idx}`, 'ns', id, 'quota'))
  })
  ;(limitRanges.list || []).forEach((it: any, idx: number) => {
    const id = `lr-${idx}`
    nodes.push(createNode(id, layoutMap.namespace[1], 360 + idx * 120, 'LimitRange', String(it?.metadata?.name ?? '-'), `limits=${Array.isArray(it?.spec?.limits) ? it.spec.limits.length : 0}`, ns, 'storage', 'LimitRange', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-lr-${idx}`, 'ns', id, 'limits'))
  })
  takeByDepth((workloads.list || []).filter((it: any) => ['Deployment', 'StatefulSet', 'DaemonSet'].includes(String(it?.kind ?? '')))).forEach((it: any, idx: number) => {
    const id = `wl-${idx}`
    nodes.push(createNode(id, layoutMap.namespace[2], idx * 140, String(it?.kind ?? 'Workload'), String(it?.metadata?.name ?? '-'), `ready=${String(it?.status?.readyReplicas ?? it?.status?.replicas ?? 0)}`, ns, 'workload', String(it?.kind ?? 'Workload'), ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-wl-${idx}`, 'ns', id, 'runs'))
  })
  takeByDepth(services.list || []).forEach((it: any, idx: number) => {
    const id = `svc-${idx}`
    nodes.push(createNode(id, layoutMap.namespace[3], idx * 120, 'Service', String(it?.metadata?.name ?? '-'), `type=${String(it?.spec?.type ?? '-')}`, ns, 'service', 'Service', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-svc-${idx}`, 'ns', id, 'service'))
  })
  takeByDepth(ingresses.list || []).forEach((it: any, idx: number) => {
    const id = `ing-${idx}`
    nodes.push(createNode(id, layoutMap.namespace[3], 340 + idx * 120, 'Ingress', String(it?.metadata?.name ?? '-'), `rules=${Array.isArray(it?.spec?.rules) ? it.spec.rules.length : 0}`, ns, 'network', 'Ingress', ns, String(it?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-ing-${idx}`, 'ns', id, 'entry'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

async function buildNamespaceNetworkPolicyTopology(cluster: number, ns: string): Promise<TopologyGraph> {
  const policies = await k8sApi.listNetworkPolicies(cluster, { namespace: ns })
  const nodes: TopologyNode[] = [createNode('ns', 0, 220, 'Namespace', ns, 'network policy overview', '', 'infra')]
  const edges: TopologyEdge[] = []
  takeByDepth(policies.list || []).forEach((policy: any, idx: number) => {
    const id = `np-${idx}`
    nodes.push(createNode(id, layoutMap.networkpolicy[1], idx * 130, 'NetworkPolicy', String(policy?.metadata?.name ?? '-'), `types=${Array.isArray(policy?.spec?.policyTypes) ? policy.spec.policyTypes.join(',') : '-'}`, ns, 'network', 'NetworkPolicy', ns, String(policy?.metadata?.name ?? '-')))
    edges.push(createEdge(`ns-np-${idx}`, 'ns', id, 'contains'))
  })
  return { nodes, edges, mermaid: buildMermaid(nodes, edges) }
}

type PodConfigRef = { name: string; source: 'volume' | 'env' | 'envFrom' }
type PodSecretRef = { name: string; source: 'volume' | 'env' | 'envFrom' | 'imagePullSecret' }
type PodPVCRef = { name: string; source: 'volume' }

function extractPodPVCs(pod: any): PodPVCRef[] {
  const volumes: any[] = Array.isArray(pod?.spec?.volumes) ? pod.spec.volumes : []
  return volumes
    .map((v) => String(v?.persistentVolumeClaim?.claimName ?? '').trim())
    .filter(Boolean)
    .map((name) => ({ name, source: 'volume' as const }))
}

function extractPodConfigMaps(pod: any): PodConfigRef[] {
  const items = new Map<string, PodConfigRef>()
  const spec = pod?.spec ?? {}
  ;(Array.isArray(spec?.volumes) ? spec.volumes : []).forEach((volume: any) => {
    const name = String(volume?.configMap?.name ?? '').trim()
    if (name) items.set(`volume:${name}`, { name, source: 'volume' })
  })
  const containers = [...(Array.isArray(spec?.containers) ? spec.containers : []), ...(Array.isArray(spec?.initContainers) ? spec.initContainers : [])]
  containers.forEach((container: any) => {
    const envList: any[] = Array.isArray(container?.env) ? container.env : []
    const envFromList: any[] = Array.isArray(container?.envFrom) ? container.envFrom : []
    envList.forEach((env: any) => {
      const name = String(env?.valueFrom?.configMapKeyRef?.name ?? '').trim()
      if (name) items.set(`env:${name}`, { name, source: 'env' })
    })
    envFromList.forEach((envFrom: any) => {
      const name = String(envFrom?.configMapRef?.name ?? '').trim()
      if (name) items.set(`envFrom:${name}`, { name, source: 'envFrom' })
    })
  })
  return Array.from(items.values())
}

function extractPodSecrets(pod: any): PodSecretRef[] {
  const items = new Map<string, PodSecretRef>()
  const spec = pod?.spec ?? {}
  ;(Array.isArray(spec?.volumes) ? spec.volumes : []).forEach((volume: any) => {
    const name = String(volume?.secret?.secretName ?? '').trim()
    if (name) items.set(`volume:${name}`, { name, source: 'volume' })
  })
  ;(Array.isArray(spec?.imagePullSecrets) ? spec.imagePullSecrets : []).forEach((it: any) => {
    const name = String(it?.name ?? '').trim()
    if (name) items.set(`imagePullSecret:${name}`, { name, source: 'imagePullSecret' })
  })
  const containers = [...(Array.isArray(spec?.containers) ? spec.containers : []), ...(Array.isArray(spec?.initContainers) ? spec.initContainers : [])]
  containers.forEach((container: any) => {
    const envList: any[] = Array.isArray(container?.env) ? container.env : []
    const envFromList: any[] = Array.isArray(container?.envFrom) ? container.envFrom : []
    envList.forEach((env: any) => {
      const name = String(env?.valueFrom?.secretKeyRef?.name ?? '').trim()
      if (name) items.set(`env:${name}`, { name, source: 'env' })
    })
    envFromList.forEach((envFrom: any) => {
      const name = String(envFrom?.secretRef?.name ?? '').trim()
      if (name) items.set(`envFrom:${name}`, { name, source: 'envFrom' })
    })
  })
  return Array.from(items.values())
}

function ingressUsesServiceName(ingress: any, serviceName: string) {
  const rules: any[] = Array.isArray(ingress?.spec?.rules) ? ingress.spec.rules : []
  for (const rule of rules) {
    const paths: any[] = Array.isArray(rule?.http?.paths) ? rule.http.paths : []
    for (const path of paths) {
      if (String(path?.backend?.service?.name ?? '') === serviceName) return true
    }
  }
  const defaultBackend = ingress?.spec?.defaultBackend?.service?.name
  return String(defaultBackend ?? '') === serviceName
}

function networkPolicyDirectionText(policy: any) {
  const types: string[] = Array.isArray(policy?.spec?.policyTypes) ? policy.spec.policyTypes : []
  if (types.length === 0) return 'policy'
  return types.join('/')
}

function networkPolicyDirectionBadge(policy: any) {
  const types: string[] = Array.isArray(policy?.spec?.policyTypes) ? policy.spec.policyTypes : []
  const hasIngress = types.includes('Ingress')
  const hasEgress = types.includes('Egress')
  if (hasIngress && hasEgress) return 'ingress+egress'
  if (hasIngress) return 'ingress'
  if (hasEgress) return 'egress'
  return 'policy'
}

function networkPolicyScopeText(policy: any) {
  const selector = policy?.spec?.podSelector?.matchLabels ?? {}
  return Object.keys(selector).length === 0 ? 'namespace all pods' : `selector=${Object.keys(selector).length}`
}

function sortResourceOptions(items: ResourceOption[]) {
  return [...items].sort((a, b) => a.label.localeCompare(b.label, 'zh-CN'))
}

function toResourceOptions(items: any[], labelPrefix?: string): ResourceOption[] {
  return sortResourceOptions(
    items
      .map((item) => String(item?.metadata?.name ?? '').trim())
      .filter(Boolean)
      .map((name) => ({
        label: labelPrefix ? `${labelPrefix} / ${name}` : name,
        value: labelPrefix ? `${labelPrefix.toLowerCase()}:${name}` : name
      }))
  )
}

function parseConfigResourceSelection(value: string): { kind?: ConfigResourceKind; name: string } {
  const raw = String(value ?? '').trim()
  if (raw.startsWith('configmap:')) return { kind: 'ConfigMap', name: raw.slice('configmap:'.length) }
  if (raw.startsWith('secret:')) return { kind: 'Secret', name: raw.slice('secret:'.length) }
  return { name: raw }
}

async function loadResourceOptions() {
  const requestId = ++resourceOptionsRequestId
  if (!clusterId.value || !requiresResourceSelection.value || (requiresNamespace.value && !namespace.value)) {
    resourceOptions.value = []
    resourceOptionsLoading.value = false
    if (!requiresResourceSelection.value) resourceName.value = ''
    return
  }

  const cluster = clusterId.value
  const ns = namespace.value
  const cacheKey = `${cluster}:${mode.value}:${ns}`
  const cached = resourceOptionsCache.get(cacheKey)
  if (cached) {
    resourceOptions.value = cached
    if (resourceName.value && !cached.some((item) => item.value === resourceName.value)) resourceName.value = ''
    return
  }

  resourceOptionsLoading.value = true
  try {
    let nextOptions: ResourceOption[] = []

    if (mode.value === 'service') {
      nextOptions = toResourceOptions((await k8sApi.listServices(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' })).list || [])
    } else if (mode.value === 'pvc') {
      nextOptions = toResourceOptions((await k8sApi.listPVCs(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' })).list || [])
    } else if (mode.value === 'pod') {
      nextOptions = toResourceOptions((await k8sApi.listPods(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' })).list || [])
    } else if (mode.value === 'node' || mode.value === 'node-storage') {
      nextOptions = toResourceOptions((await k8sApi.listNodes(cluster, { sort_by: 'metadata.name', order: 'asc' })).list || [])
    } else if (mode.value === 'ingress') {
      nextOptions = toResourceOptions((await k8sApi.listIngresses(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' })).list || [])
    } else if (mode.value === 'config') {
      const [configMaps, secrets] = await Promise.all([
        k8sApi.listConfigMaps(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' }),
        k8sApi.listSecrets(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' })
      ])
      nextOptions = sortResourceOptions([
        ...toResourceOptions(configMaps.list || [], 'ConfigMap'),
        ...toResourceOptions(secrets.list || [], 'Secret')
      ])
    } else if (mode.value === 'pv') {
      nextOptions = toResourceOptions((await k8sApi.listPVs(cluster, { sort_by: 'metadata.name', order: 'asc' })).list || [])
    } else if (mode.value === 'serviceaccount') {
      nextOptions = toResourceOptions((await k8sApi.listServiceAccounts(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' })).list || [])
    } else if (mode.value === 'deployment') {
      const workloads = await k8sApi.listWorkloads(cluster, { namespace: ns, kind: 'Deployment', sort_by: 'metadata.name', order: 'asc' })
      nextOptions = toResourceOptions((workloads.list || []).filter((item: any) => {
        const kind = String(item?.kind ?? '').trim()
        return !kind || kind === 'Deployment'
      }))
      if (nextOptions.length === 0) {
        const fallback = await k8sApi.listWorkloads(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' })
        nextOptions = toResourceOptions((fallback.list || []).filter((item: any) => String(item?.kind ?? '') === 'Deployment'))
      }
    } else if (mode.value === 'networkpolicy') {
      nextOptions = toResourceOptions((await k8sApi.listNetworkPolicies(cluster, { namespace: ns, sort_by: 'metadata.name', order: 'asc' })).list || [])
    }

    if (requestId !== resourceOptionsRequestId) return
    resourceOptionsCache.set(cacheKey, nextOptions)
    resourceOptions.value = nextOptions
    if (resourceName.value && !nextOptions.some((item) => item.value === resourceName.value)) {
      resourceName.value = ''
    }
  } catch (e) {
    if (requestId !== resourceOptionsRequestId) return
    resourceOptions.value = []
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    if (requestId === resourceOptionsRequestId) resourceOptionsLoading.value = false
  }
}

async function loadClusters() {
  try {
    const data = await clustersApi.listClusters({ page: 1, page_size: 200 })
    clusters.value = data.list ?? []
    if (!clusterId.value && clusters.value.length > 0) clusterId.value = clusters.value[0].id
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  }
}

async function loadNamespaces() {
  if (!clusterId.value) return
  try {
    const data = await k8sApi.listNamespaces(clusterId.value, { sort_by: 'metadata.name', order: 'asc' })
    namespaces.value = (data.list ?? []).map((it: any) => String(it?.metadata?.name ?? '')).filter(Boolean)
    if (!namespace.value && namespaces.value.length > 0) namespace.value = namespaces.value[0]
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  }
}

function syncQuery() {
  if (props.embedded) return
  router.replace({
    query: {
      clusterId: clusterId.value ? String(clusterId.value) : undefined,
      mode: mode.value,
      scope: supportsNamespaceScope.value ? scope.value : undefined,
      namespace: requiresNamespace.value ? (namespace.value || undefined) : undefined,
      name: resourceName.value || undefined
    }
  })
}

async function loadTopology() {
  if (!clusterId.value || (requiresNamespace.value && !namespace.value)) {
    notifyError('请先选择集群与命名空间')
    return
  }
  if (requiresResourceSelection.value && !resourceName.value.trim()) {
    if (supportsNamespaceScope.value) {
      scope.value = defaultScopeForMode(mode.value)
      syncQuery()
    } else {
      notifyError('请先选择资源后再生成关系图')
      return
    }
  }
  loading.value = true
  try {
    type BuilderFn = (cluster: number, ns: string, name: string) => Promise<TopologyGraph>
    const builderMap: Record<string, BuilderFn | [BuilderFn, BuilderFn]> = {
      service:       [buildServiceTopology,              buildNamespaceServiceTopology],
      pvc:           [buildPVCTopology,                  buildNamespacePVCTopology],
      pod:           [buildPodTopology,                  buildNamespacePodTopology],
      ingress:       [buildIngressTopology,              buildNamespaceIngressTopology],
      serviceaccount:[buildServiceAccountTopology,       buildNamespaceServiceAccountTopology],
      deployment:    [buildDeploymentTopology,           buildNamespaceDeploymentTopology],
      networkpolicy: [buildNetworkPolicyTopology,        buildNamespaceNetworkPolicyTopology],
      node:          buildNodeTopology,
      config:        buildConfigTopology,
      pv:            buildPVTopology,
      'node-storage': buildNodeStorageTopology,
      namespace:     buildNamespaceOverviewTopology,
    }
    const entry = builderMap[mode.value] ?? buildNamespaceOverviewTopology
    const builder: BuilderFn = Array.isArray(entry) ? (isNamespaceScope.value ? entry[1] : entry[0]) : entry
    const baseGraph = applyGraphLayout(finalizeGraph(await builder(clusterId.value, namespace.value, resourceName.value.trim())))
    currentGraph.value = {
      ...baseGraph,
      lanes: mode.value === 'pod' ? buildPodLanes(baseGraph) : undefined
    }
    syncQuery()
                  await nextTick()
    fitView()
    notifySuccess('关系图已生成')
  } catch (e) {
    const err = e as ApiError
    notifyError(err?.message || String((e as Error)?.message || e))
    currentGraph.value = { nodes: [], edges: [], mermaid: '' }
  } finally {
    loading.value = false
  }
}

function resetQuery() {
  mode.value = 'service'
  scope.value = defaultScopeForMode('service')
  namespace.value = namespaces.value.includes('default') ? 'default' : (namespaces.value[0] ?? '')
  resourceName.value = ''
  currentGraph.value = { nodes: [], edges: [], mermaid: '' }
  syncQuery()
}

function exportPng() {
  void exportRenderedPng()
}

async function exportRenderedPng() {
  const shell = canvasPanelRef.value?.querySelector('.topology-vue-flow') as HTMLElement | null
  if (!shell || visibleGraph.value.nodes.length === 0) return

  // 1. 保存当前 viewport，导出后恢复
  const savedViewport = graphCanvasRef.value?.getViewport?.() ?? null

  try {
    // 2. fitView 让所有节点收入可见区域，等待动画结束
    graphCanvasRef.value?.fitView()
    await new Promise<void>((resolve) => setTimeout(resolve, 340))

    // 3. 计算安全像素比：浏览器 Canvas 上限约 268M 像素（≈ 16384²）
    //    留额度，限制单边最大为 8000px
    const MAX_SIDE = 8000
    const w = shell.clientWidth || shell.offsetWidth
    const h = shell.clientHeight || shell.offsetHeight
    const ratio = Math.min(2, MAX_SIDE / Math.max(w, h, 1))

    // 4. 修补 SVG fill 属性（防止黑色填充）
    const svgPaths = shell.querySelectorAll<SVGElement>(
      '.vue-flow__edges path, .vue-flow__edges polyline, .vue-flow__edges polygon, .vue-flow__edge-path, .vue-flow__connection-path, marker path, marker polygon'
    )
    const patchedFills: Array<{ el: SVGElement; was: string | null }> = []
    svgPaths.forEach((el) => {
      const was = el.getAttribute('fill')
      if (!was || was === '' || was === 'inherit') {
        el.setAttribute('fill', 'none')
        patchedFills.push({ el, was })
      }
    })
    const markerPaths = shell.querySelectorAll<SVGElement>('marker path, marker polygon')
    markerPaths.forEach((el) => {
      const stroke = el.getAttribute('stroke') || el.closest('.vue-flow__edge')?.querySelector('.vue-flow__edge-path')?.getAttribute('stroke')
      if (stroke) el.setAttribute('fill', stroke)
    })

    let dataUrl: string
    try {
      dataUrl = await toPng(shell, {
        backgroundColor: isDark() ? '#0f172a' : '#f8fafc',
        pixelRatio: ratio,
        cacheBust: true,
        filter: (node) => {
          if (node.classList && (
            node.classList.contains('vue-flow__controls') ||
            node.classList.contains('vue-flow__minimap') ||
            node.classList.contains('topology-overlay-toolbar')
          )) return false
          return true
        }
      })
    } finally {
      patchedFills.forEach(({ el, was }) => {
        if (was === null) el.removeAttribute('fill')
        else el.setAttribute('fill', was)
      })
    }

    const link = document.createElement('a')
    link.href = dataUrl
    link.download = `k8s-topology-${mode.value}-${resourceName.value || 'graph'}.png`
    link.click()
    notifySuccess(`关系图 PNG 已导出（${visibleGraph.value.nodes.length} 节点，像素比 ${ratio.toFixed(2)}×）`)
  } catch (error) {
    notifyError(`导出失败：${String((error as Error)?.message || error)}`)
  } finally {
    // 5. 恢复导出前的 viewport
    if (savedViewport) {
      await new Promise<void>((resolve) => setTimeout(resolve, 60))
      graphCanvasRef.value?.setViewport?.(savedViewport)
    }
  }
}

async function toggleFullscreen() {
  const el = canvasPanelRef.value
  if (!el) return
  if (document.fullscreenElement) {
    await document.exitFullscreen()
    return
  }
  await el.requestFullscreen()
}

async function copyMermaid() {
  try {
    await navigator.clipboard.writeText(visibleGraph.value.mermaid)
    notifySuccess('Mermaid 文本已复制')
  } catch {
    notifyError('复制失败')
  }
}

watch(clusterId, async () => {
  if (props.fixedClusterId && clusterId.value !== props.fixedClusterId) {
    clusterId.value = props.fixedClusterId
    return
  }
  // 切换集群时清空资源选项缓存，避免显示旧集群的资源
  resourceOptionsCache.clear()
  await loadNamespaces()
  await loadResourceOptions()
  syncQuery()
})

watch(mode, async (value) => {
  scope.value = defaultScopeForMode(value)
  await loadResourceOptions()
  syncQuery()
})

watch(scope, async () => {
  await loadResourceOptions()
  syncQuery()
})

watch(namespace, async () => {
  await loadResourceOptions()
  syncQuery()
})

watch(resourceName, () => syncQuery())

watch(layoutDensity, async (value, previous) => {
  if (value === previous || currentGraph.value.nodes.length === 0) return
  await relayoutGraph(false)
})

onMounted(async () => {
  const q = route.query
  if (q.mode === 'service' || q.mode === 'pvc' || q.mode === 'pod' || q.mode === 'node' || q.mode === 'ingress' || q.mode === 'config' || q.mode === 'pv' || q.mode === 'serviceaccount' || q.mode === 'deployment' || q.mode === 'node-storage' || q.mode === 'networkpolicy' || q.mode === 'namespace') mode.value = q.mode
  scope.value = defaultScopeForMode(mode.value)
  if (q.scope === 'resource' || q.scope === 'namespace') scope.value = q.scope
  if (typeof q.namespace === 'string') namespace.value = q.namespace
  if (typeof q.name === 'string') resourceName.value = q.name
  if (props.fixedClusterId) {
    clusterId.value = props.fixedClusterId
  } else if (typeof q.clusterId === 'string' && q.clusterId.trim()) {
    clusterId.value = Number(q.clusterId)
  }
  await loadClusters()
  await loadNamespaces()
  await loadResourceOptions()
  if (clusterId.value && ((requiresNamespace.value && namespace.value) || !requiresNamespace.value) && (isNamespaceScope.value || resourceName.value.trim())) await loadTopology()
})
</script>

<style scoped>
.topology-page {
  --topology-page-bg: linear-gradient(180deg, rgba(248, 250, 252, 0.88) 0%, rgba(241, 245, 249, 0.98) 100%);
  --topology-surface-bg: rgba(255, 255, 255, 0.9);
  --topology-surface-border: rgba(148, 163, 184, 0.18);
  --topology-surface-shadow: 0 14px 30px rgba(15, 23, 42, 0.05);
  --topology-panel-bg: rgba(255, 255, 255, 0.86);
  --topology-panel-shadow: 0 24px 60px rgba(15, 23, 42, 0.08);
  --topology-toolbar-bg: rgba(255, 255, 255, 0.85);
  --topology-toolbar-border: rgba(148, 163, 184, 0.2);
  --topology-toolbar-shadow: 0 8px 24px rgba(15, 23, 42, 0.06);
  --topology-text-primary: var(--color-text-primary);
  --topology-text-secondary: var(--color-text-secondary);
  min-height: 100%;
  padding: 16px;
  background: var(--topology-page-bg);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

:global(html.dark) .topology-page {
  --topology-page-bg: linear-gradient(180deg, rgba(2, 6, 23, 0.34) 0%, rgba(15, 23, 42, 0.74) 100%);
  --topology-surface-bg: rgba(15, 23, 42, 0.82);
  --topology-surface-border: rgba(148, 163, 184, 0.18);
  --topology-surface-shadow: 0 20px 40px rgba(2, 6, 23, 0.24), inset 0 1px 0 rgba(148, 163, 184, 0.06);
  --topology-panel-bg: rgba(15, 23, 42, 0.88);
  --topology-panel-shadow: 0 24px 60px rgba(2, 6, 23, 0.3);
  --topology-toolbar-bg: rgba(15, 23, 42, 0.88);
  --topology-toolbar-border: rgba(148, 163, 184, 0.18);
  --topology-toolbar-shadow: 0 14px 36px rgba(2, 6, 23, 0.28);
}

.topology-page--embedded {
  padding: 0;
  background: transparent;
  min-height: 0;
  height: 100%;
  flex: 1 1 auto;
}

.topology-stage {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-height: 0;
  min-width: 0;
}

.topology-page--embedded .topology-stage {
  flex: 1 1 auto;
  min-height: 0;
}

.topology-toolbar__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 10px;
  min-width: 0;
}

.topology-toolbar__meta--compact {
  gap: 4px 8px;
}

.topology-toolbar__summary {
  color: var(--topology-text-secondary);
  font-size: 11px;
  font-weight: 600;
  line-height: 1.2;
}

.topology-control-card {
  border: 1px solid var(--topology-surface-border);
  border-radius: 8px;
  background: var(--topology-surface-bg);
  box-shadow: var(--topology-surface-shadow);
  padding: 8px 12px;
}

.topology-console-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 6px;
}

.topology-console-head__main {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-width: 0;
  flex: 1 1 0%;
}

.topology-console-head__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
  flex-shrink: 0;
}

.topology-control-head__title {
  color: var(--topology-text-primary);
  font-size: 13px;
  font-weight: 800;
}

.topology-filter-combo {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.topology-filter-combo__selects {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 0%;
  min-width: 0;
  flex-wrap: wrap;
}

.topology-combo-select {
  min-width: 100px;
  flex: 1 1 0%;
  max-width: 180px;
}

.topology-combo-select--resource {
  min-width: 220px;
  max-width: 320px;
}

.topology-filter-combo__prefs {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.topology-toggle-item {
  display: flex;
  align-items: center;
  gap: 10px;
  white-space: nowrap;
}

.topology-toggle-item__text {
  color: var(--topology-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.topology-panel {
  border: 1px solid var(--topology-surface-border);
  border-radius: 8px;
  background: var(--topology-panel-bg);
  box-shadow: var(--topology-panel-shadow);
  backdrop-filter: blur(14px);
  overflow: hidden;
}

.topology-panel--canvas {
  display: flex;
  flex-direction: column;
  position: relative;
  min-height: 0;
  width: 100%;
  min-width: 0;
}

.topology-overlay-toolbar {
  position: absolute;
  top: 14px;
  right: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--topology-toolbar-bg);
  backdrop-filter: blur(10px);
  border: 1px solid var(--topology-toolbar-border);
  border-radius: 8px;
  box-shadow: var(--topology-toolbar-shadow);
  z-index: 10;
}

.topology-panel--main {
  width: 100%;
  min-width: 0;
  flex: 1 1 auto;
  min-height: var(--topology-panel-height, clamp(420px, calc(100vh - 260px), 680px));
  height: var(--topology-panel-height, clamp(420px, calc(100vh - 260px), 680px));
}

.topology-page--embedded .topology-panel--main {
  min-height: 0;
  height: auto;
  flex: 1 1 auto;
}

.topology-syntax-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

@media (max-width: 980px) {
  .topology-panel--main {
    min-height: clamp(380px, calc(100vh - 280px), 640px);
    height: clamp(380px, calc(100vh - 280px), 640px);
  }

  .topology-console-head {
    flex-direction: column;
    align-items: stretch;
  }

  .topology-filter-combo {
    flex-direction: column;
    align-items: stretch;
  }

  .topology-filter-combo__selects,
  .topology-filter-combo__prefs {
    flex-wrap: wrap;
  }

  .topology-combo-select {
    max-width: 100%;
  }
}

@media (max-width: 640px) {
  .topology-panel--main {
    min-height: 360px;
    height: min(60vh, 520px);
  }

  .topology-toggle-item {
    flex-wrap: wrap;
  }

  .topology-combo-select {
    max-width: 100%;
    flex: 1 1 120px;
  }

  .topology-overlay-toolbar {
    top: auto;
    bottom: 14px;
    right: 50%;
    transform: translateX(50%);
    width: auto;
    justify-content: center;
    flex-wrap: wrap;
  }
}

/* Dark mode direct overrides — bypasses CSS variable cascade issue */
:global(html.dark) .topology-page {
  background: linear-gradient(180deg, rgba(2, 6, 23, 0.34) 0%, rgba(15, 23, 42, 0.74) 100%);
}

:global(html.dark) .topology-panel {
  background: rgba(15, 23, 42, 0.88);
  border-color: rgba(148, 163, 184, 0.18);
  box-shadow: 0 24px 60px rgba(2, 6, 23, 0.3);
}

:global(html.dark) .topology-control-card {
  background: rgba(15, 23, 42, 0.82);
  border-color: rgba(148, 163, 184, 0.18);
  box-shadow: 0 20px 40px rgba(2, 6, 23, 0.24), inset 0 1px 0 rgba(148, 163, 184, 0.06);
}

:global(html.dark) .topology-overlay-toolbar {
  background: rgba(15, 23, 42, 0.88);
  border-color: rgba(148, 163, 184, 0.18);
  box-shadow: 0 14px 36px rgba(2, 6, 23, 0.28);
}
</style>
