<template>
  <WorkloadDetailDrawerShell
    v-model="visible"
    :title="detailTitle"
    :loading="refreshing"
    :ns="detailNamespace"
    :name="detailName"
    @refresh="refreshDetail"
  >
    <template #actions>
      <el-tooltip content="打开资源关系图" placement="top">
        <el-button size="small" text @click="emit('open-topology', { mode: 'service', namespace: detailNamespace, name: detailName })">关系图</el-button>
      </el-tooltip>
    </template>
    <template #summary>
      <div class="k8s-summary-grid">
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">名称:</div><div class="k8s-v">{{ detailName }}</div></div>
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">命名空间:</div><div class="k8s-v">{{ detailNamespace }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Type:</div><div class="k8s-v"><span class="k8s-link">{{ typeText }}</span></div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Ports:</div><div class="k8s-v">{{ portsText }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="命名空间">{{ detailNamespace }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ createdAtText }}</el-descriptions-item>
              <el-descriptions-item label="type">{{ typeText }}</el-descriptions-item>
              <el-descriptions-item label="clusterIP">{{ clusterIpText }}</el-descriptions-item>
              <el-descriptions-item label="ports">{{ portsText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Selector</div></template>
            <CodeMirrorViewer :text="selectorViewText" language="json" height="180px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Ports</div></template>
            <CodeMirrorViewer :text="portsViewText" language="json" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Labels</div></template>
            <CodeMirrorViewer :text="labelsViewText" language="json" height="200px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="关联资源" name="related">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Pods（匹配 Selector）</div>
                <div class="k8s-section-actions">
                  <el-space :size="8">
                    <el-tag v-if="relatedLoading" size="small" type="info" effect="light">加载中</el-tag>
                    <el-tag v-else size="small" type="info" effect="light">共 {{ relatedPods.length }} 条</el-tag>
                    <el-tooltip content="刷新" placement="top">
                      <el-button size="small" :icon="RefreshRight" circle :loading="relatedLoading" @click="loadRelated" />
                    </el-tooltip>
                  </el-space>
                </div>
              </div>
            </template>
            <el-table :data="relatedPods" stripe size="small" class="k8s-detail-table" @row-dblclick="(r: any) => emit('open-related-pod', r)">
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
              <el-table-column prop="phase" label="Phase" width="140" show-overflow-tooltip />
              <el-table-column prop="ready" label="Ready" width="100" align="center" header-align="center" />
              <el-table-column prop="restarts" label="Restarts" width="100" align="center" header-align="center" />
              <el-table-column prop="node" label="Node" min-width="200" show-overflow-tooltip />
              <el-table-column prop="ownersText" label="Owners" min-width="240" show-overflow-tooltip />
            </el-table>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">资源（从 Pods ownerReferences + 工作负载模板聚合）</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ relatedControllers.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="relatedControllers" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="kind" label="Kind" width="180" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
              <el-table-column prop="name" label="名称" min-width="260" show-overflow-tooltip />
            </el-table>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Ingresses（引用该 Service）</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ relatedIngresses.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="relatedIngresses" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
              <el-table-column prop="hostsText" label="Hosts" min-width="220" show-overflow-tooltip />
              <el-table-column prop="rulesText" label="Rules" min-width="320" show-overflow-tooltip />
              </el-table>
            </el-card>
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header>
                <div class="k8s-section-title-row">
                  <div class="k8s-section-title">Endpoints（传统关联）</div>
                  <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ relatedEndpoints.length }} 条</el-tag></div>
                </div>
              </template>
              <el-table :data="relatedEndpoints" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
                <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
                <el-table-column prop="ready" label="Ready" width="100" align="center" header-align="center" />
                <el-table-column prop="notReady" label="NotReady" width="110" align="center" header-align="center" />
                <el-table-column prop="ports" label="Ports" width="100" align="center" header-align="center" />
              </el-table>
            </el-card>
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header>
                <div class="k8s-section-title-row">
                  <div class="k8s-section-title">EndpointSlices（关联 Service）</div>
                  <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ relatedEndpointSlices.length }} 条</el-tag></div>
                </div>
              </template>
              <el-table :data="relatedEndpointSlices" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
                <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
                <el-table-column prop="addressType" label="AddressType" width="140" show-overflow-tooltip />
                <el-table-column prop="endpoints" label="Endpoints" width="120" align="center" header-align="center" />
                <el-table-column prop="ports" label="Ports" width="100" align="center" header-align="center" />
              </el-table>
            </el-card>
          </div>
        </el-tab-pane>

      <el-tab-pane label="事件日志" name="events">
        <div class="k8s-tab-pane">
          <div class="k8s-pane-toolbar">
            <el-space :size="8">
              <el-tooltip content="刷新" placement="top">
                <el-button size="small" :icon="RefreshRight" circle :loading="eventsLoading" @click="loadEvents" />
              </el-tooltip>
              <el-tag v-if="eventsLoading" size="small" type="info" effect="light">加载中</el-tag>
              <el-tag v-else size="small" type="info" effect="light">共 {{ events.length }} 条</el-tag>
            </el-space>
          </div>
          <el-table :data="events" stripe size="small" class="k8s-detail-table">
            <el-table-column prop="type" label="Type" width="110" />
            <el-table-column prop="reason" label="Reason" width="200" show-overflow-tooltip />
            <el-table-column prop="message" label="Message" min-width="360" show-overflow-tooltip />
            <el-table-column prop="count" label="Count" width="90" align="center" header-align="center" />
            <el-table-column prop="lastSeen" label="LastSeen" width="140" align="center" header-align="center" />
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="YAML配置" name="yaml">
        <div class="k8s-tab-pane">
          <K8sYamlPanel
            :meta="`cluster=${props.clusterId}  ${detailNamespace}/${detailName}`"
            :text="yamlViewText"
            :loading="yamlLoading"
            height="100%"
            @refresh="loadYaml"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
  </WorkloadDetailDrawerShell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { CopyDocument, Expand, Fold, Moon, RefreshRight, Search, Sunny } from '@element-plus/icons-vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import WorkloadDetailDrawerShell from '@/features/k8s/components/WorkloadDetailDrawerShell.vue'
import K8sYamlPanel from '@/features/k8s/components/K8sYamlPanel.vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import { notifyError } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import {
  type ControllerRow,
  type EventRow,
  collectControllersFromPodsRaw,
  formatAgeMs,
  formatPorts,
  formatRules,
  formatTs,
  getEventTimeMs,
  getHosts,
  getRowNamespace,
  ingressUsesService,
  matchLabels,
  mergeControllers,
  normalizeLabelRecord,
  normalizeMultilineText,
  toRelatedPodVmFromPod
} from '../../ClusterManageView.utils'

const props = defineProps<{
  clusterId: number
  editorTheme: 'auto' | 'light' | 'dark'
  editorThemeEffectiveDark: boolean
  list: any[]
}>()

const emit = defineEmits<{
  (e: 'toggle-editor-theme'): void
  (e: 'open-related-pod', row: any): void
  (e: 'open-topology', payload: { mode: 'service'; namespace: string; name: string }): void
  (e: 'refresh-list'): void
}>()

function copyText(text: string) {
  if (!text) return
  navigator.clipboard.writeText(text).catch(() => notifyError('复制失败'))
}

function findRow(ns: string, name: string): any | null {
  for (const it of props.list ?? []) {
    if (getRowNamespace(it) === ns && String(it?.metadata?.name ?? '') === name) return it
  }
  return null
}

// ── state ──
const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'related' | 'events' | 'yaml'>('overview')
const row = ref<any>(null)

const detailNamespace = computed(() => getRowNamespace(row.value) || '')
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `Service 详情：${detailName.value}` : 'Service 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const typeText = computed(() => String(row.value?.spec?.type ?? ''))
const clusterIpText = computed(() => String(row.value?.spec?.clusterIP ?? '-'))
const portsText = computed(() => formatPorts(row.value?.spec?.ports ?? []))
const portsViewText = computed(() => JSON.stringify(row.value?.spec?.ports ?? [], null, 2))
const selectorViewText = computed(() => JSON.stringify(row.value?.spec?.selector ?? {}, null, 2))
const labelsViewText = computed(() => JSON.stringify(row.value?.metadata?.labels ?? {}, null, 2))

// ── related ──
const relatedPods = ref<any[]>([])
const relatedControllers = ref<ControllerRow[]>([])
const relatedIngresses = ref<any[]>([])
const relatedEndpoints = ref<any[]>([])
const relatedEndpointSlices = ref<any[]>([])
const relatedLoading = ref(false)

// ── events ──
const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

// ── yaml ──
const yamlLoading = ref(false)
const yamlText = ref('')
const yamlWrap = ref(true)
const yamlLineNumbers = ref(true)
const yamlViewerRef = ref<{ openSearch: () => void; foldAll: () => void; unfoldAll: () => void } | null>(null)
const yamlViewText = computed(() => normalizeMultilineText(yamlText.value))

async function loadYaml() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value; const name = detailName.value
  if (!ns || !name) return
  yamlLoading.value = true
  try { yamlText.value = (await k8sApi.getServiceYaml(props.clusterId, ns, name)).text ?? '' }
  catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { yamlLoading.value = false }
}

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value; const name = detailName.value
  const uid = String(row.value?.metadata?.uid ?? '').trim()
  if (!ns || !name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, { namespace: ns })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const filtered = list.filter((ev) => {
      const inv = ev?.involvedObject ?? ev?.regarding ?? {}
      if (String(inv?.kind ?? '') && String(inv?.kind ?? '') !== 'Service') return false
      if (String(inv?.namespace ?? '') && String(inv?.namespace ?? '') !== ns) return false
      if (uid && String(inv?.uid ?? '')) return String(inv?.uid ?? '') === uid
      return String(inv?.name ?? '') === name
    })
    const now = Date.now()
    events.value = filtered
      .map((ev) => { const t = getEventTimeMs(ev); return { tMs: t ?? -1, type: String(ev?.type ?? '') || '-', reason: String(ev?.reason ?? '') || '-', message: String(ev?.message ?? '') || '-', count: Number(ev?.count ?? ev?.series?.count ?? 1) || 1, lastSeen: t != null ? formatAgeMs(Math.max(0, now - t)) : '-' } })
      .sort((a, b) => b.tMs - a.tMs)
      .map(({ tMs: _, ...rest }) => rest)
  } catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { eventsLoading.value = false }
}

async function loadRelated() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value; const name = detailName.value
  if (!ns || !name) return
  const selector = normalizeLabelRecord(row.value?.spec?.selector)
  relatedLoading.value = true
  try {
    relatedPods.value = []; relatedControllers.value = []; relatedIngresses.value = []; relatedEndpoints.value = []; relatedEndpointSlices.value = []
    if (Object.keys(selector).length > 0) {
      const podsResp = await k8sApi.listPods(props.clusterId, { namespace: ns })
      const pods = (Array.isArray(podsResp.list) ? podsResp.list : [])
        .filter((p: any) => matchLabels(p?.metadata?.labels, selector))
        .map((p: any) => toRelatedPodVmFromPod(p))
        .sort((a: any, b: any) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`))
      relatedPods.value = pods
      const cFromPods = collectControllersFromPodsRaw(pods.map((p: any) => p.rawPod ?? p))
      const w = await k8sApi.listWorkloads(props.clusterId, { namespace: ns })
      const cFromWl: ControllerRow[] = (Array.isArray(w.list) ? w.list : [])
        .filter((it: any) => matchLabels(it?.spec?.template?.metadata?.labels, selector))
        .map((it: any) => ({ kind: String(it?.kind ?? '').trim(), namespace: String(it?.metadata?.namespace ?? ns).trim(), name: String(it?.metadata?.name ?? '').trim() }))
        .filter((c: ControllerRow) => c.kind && c.namespace && c.name)
      relatedControllers.value = mergeControllers(cFromPods, cFromWl)
    }
    const ing = await k8sApi.listIngresses(props.clusterId, { namespace: ns })
    relatedIngresses.value = (Array.isArray(ing.list) ? ing.list : [])
      .filter((it: any) => ingressUsesService(it, name))
      .map((it: any) => ({ namespace: String(it?.metadata?.namespace ?? ns).trim(), name: String(it?.metadata?.name ?? '').trim(), hostsText: getHosts(it).join(', ') || '-', rulesText: formatRules(it) || '-' }))
      .filter((it: any) => it.name)
      .sort((a: any, b: any) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`))
    const endpoints = await k8sApi.listEndpoints(props.clusterId, { namespace: ns })
    relatedEndpoints.value = (Array.isArray(endpoints.list) ? endpoints.list : [])
      .filter((it: any) => String(it?.metadata?.name ?? '') === name)
      .map((it: any) => {
        const subsets: any[] = Array.isArray(it?.subsets) ? it.subsets : []
        const ready = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.addresses) ? subset.addresses.length : 0), 0)
        const notReady = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.notReadyAddresses) ? subset.notReadyAddresses.length : 0), 0)
        const ports = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.ports) ? subset.ports.length : 0), 0)
        return { namespace: String(it?.metadata?.namespace ?? ns).trim(), name: String(it?.metadata?.name ?? '').trim(), ready, notReady, ports }
      })
      .filter((it: any) => it.name)
    const eps = await k8sApi.listEndpointSlices(props.clusterId, { namespace: ns })
    relatedEndpointSlices.value = (Array.isArray(eps.list) ? eps.list : [])
      .filter((it: any) => String(it?.metadata?.labels?.['kubernetes.io/service-name'] ?? '') === name)
      .map((it: any) => ({
        namespace: String(it?.metadata?.namespace ?? ns).trim(),
        name: String(it?.metadata?.name ?? '').trim(),
        addressType: String(it?.addressType ?? '-'),
        endpoints: Array.isArray(it?.endpoints) ? it.endpoints.length : 0,
        ports: Array.isArray(it?.ports) ? it.ports.length : 0
      }))
      .filter((it: any) => it.name)
      .sort((a: any, b: any) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`))
  } catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { relatedLoading.value = false }
}

async function refreshDetail() {
  if (!visible.value) return
  try {
    refreshing.value = true
    emit('refresh-list')
    await new Promise(r => setTimeout(r, 300))
    const next = findRow(detailNamespace.value, detailName.value)
    if (next) row.value = next
    if (tab.value === 'events') await loadEvents()
    if (tab.value === 'related') await loadRelated()
    if (tab.value === 'yaml') await loadYaml()
  } finally { refreshing.value = false }
}

watch(() => visible.value, (v) => {
  if (v) return
  tab.value = 'overview'; row.value = null
  relatedPods.value = []; relatedControllers.value = []; relatedIngresses.value = []; relatedEndpoints.value = []; relatedEndpointSlices.value = []
  events.value = []; yamlText.value = ''
})

watch(
  () => [visible.value, tab.value, detailNamespace.value, detailName.value] as const,
  ([v, t]) => {
    if (!v) return
    if (t === 'events' && events.value.length === 0) void loadEvents()
    if (t === 'related' && relatedPods.value.length === 0 && relatedControllers.value.length === 0 && relatedIngresses.value.length === 0 && relatedEndpoints.value.length === 0 && relatedEndpointSlices.value.length === 0) void loadRelated()
    if (t === 'yaml' && !yamlText.value) void loadYaml()
  }
)

function open(r: any) { row.value = r; tab.value = 'overview'; visible.value = true }
function close() { visible.value = false }
defineExpose({ open, close })
</script>
