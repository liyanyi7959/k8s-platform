import React, { useState, useRef, useCallback, useEffect, useMemo } from 'react'
import {
  Card,
  Input,
  Space,
  Button,
  Tooltip,
  Tag,
  Spin,
  Empty,
  Row,
  Col,
  Descriptions,
  Typography,
  Badge,
  Divider,
  Select,
  Radio,
  message,
} from 'antd'
import {
  SearchOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  CompressOutlined,
  ReloadOutlined,
  ClusterOutlined,
  CloudServerOutlined,
  AppstoreOutlined,
  NodeIndexOutlined,
  HddOutlined,
  BlockOutlined,
  ApartmentOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons'
import { AppPage, YamlEditor } from '@/components'
import { useQuery } from '@tanstack/react-query'
import {
  getNodes,
  getNamespaces,
  getDeployments,
  getPods,
  getK8sServices,
  getConfigMaps,
  getResourceYaml,
  listSecrets,
  listIngresses,
  listGenericResources,
} from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'

const { Text } = Typography

// ═══════════════════════════════════════
// 资源类型配置
// ═══════════════════════════════════════
interface ResourceConfig {
  key: string
  label: string
  icon: React.ReactNode
  color: string
  bg: string
}

const RESOURCE_CONFIGS: ResourceConfig[] = [
  { key: 'node', label: '节点', icon: <ClusterOutlined />, color: '#0891b2', bg: '#ecfeff' },
  {
    key: 'namespace',
    label: '命名空间',
    icon: <AppstoreOutlined />,
    color: '#7c3aed',
    bg: '#f5f3ff',
  },
  {
    key: 'deployment',
    label: 'Deployment',
    icon: <CloudServerOutlined />,
    color: '#2563eb',
    bg: '#eff6ff',
  },
  { key: 'pod', label: 'Pod', icon: <HddOutlined />, color: '#059669', bg: '#ecfdf5' },
  {
    key: 'service',
    label: 'Service',
    icon: <NodeIndexOutlined />,
    color: '#d97706',
    bg: '#fffbeb',
  },
  {
    key: 'configmap',
    label: 'ConfigMap',
    icon: <BlockOutlined />,
    color: '#6366f1',
    bg: '#eef2ff',
  },
  { key: 'statefulset', label: 'StatefulSet', icon: <CloudServerOutlined />, color: '#0d9488', bg: '#f0fdfa' },
  { key: 'daemonset', label: 'DaemonSet', icon: <CloudServerOutlined />, color: '#65a30d', bg: '#f7fee7' },
  { key: 'secret', label: 'Secret', icon: <BlockOutlined />, color: '#be185d', bg: '#fdf2f8' },
  { key: 'pvc', label: 'PVC', icon: <HddOutlined />, color: '#0369a1', bg: '#f0f9ff' },
  { key: 'ingress', label: 'Ingress', icon: <ApartmentOutlined />, color: '#c2410c', bg: '#fff7ed' },
]

function getConfig(type: string): ResourceConfig {
  return RESOURCE_CONFIGS.find((c) => c.key === type) || RESOURCE_CONFIGS[0]!
}

// ═══════════════════════════════════════
// 拓扑节点数据
// ═══════════════════════════════════════
interface TopoNode {
  id: string
  type: string
  name: string
  namespace?: string
  status?: string
  x: number
  y: number
  meta?: Record<string, any>
}

interface TopoEdge {
  id: string
  source: string
  target: string
  label?: string
  type?: 'owns' | 'contains' | 'routes' | 'uses' // 关系类型
}

// 连线样式分类
const edgeStyleMap: Record<string, { stroke: string; dasharray?: string; width: number }> = {
  owns: { stroke: '#2563eb', dasharray: undefined, width: 2 },
  contains: { stroke: '#cbd5e1', dasharray: '6,4', width: 1.5 },
  routes: { stroke: '#c2410c', dasharray: '8,3', width: 2 },
  default: { stroke: '#94a3b8', dasharray: '6,4', width: 1.5 },
}

// 节点类型 → K8s 资源类型映射（用于加载 YAML）
const NODE_TYPE_TO_RESOURCE: Record<string, string> = {
  node: 'nodes',
  namespace: 'namespaces',
  deployment: 'deployments',
  statefulset: 'statefulsets',
  daemonset: 'daemonsets',
  pod: 'pods',
  service: 'services',
  configmap: 'configmaps',
  secret: 'secrets',
  pvc: 'pvcs',
  ingress: 'ingresses',
}

// ═══════════════════════════════════════
// BFS 查找全链路关联节点
// ═══════════════════════════════════════
function findRelatedNodeIds(startId: string, allEdges: TopoEdge[]): Set<string> {
  const ids = new Set<string>([startId])
  const queue = [startId]
  while (queue.length > 0) {
    const current = queue.shift()!
    allEdges.forEach((e) => {
      if (e.source === current && !ids.has(e.target)) {
        ids.add(e.target)
        queue.push(e.target)
      }
      if (e.target === current && !ids.has(e.source)) {
        ids.add(e.source)
        queue.push(e.source)
      }
    })
  }
  return ids
}

// ═══════════════════════════════════════
// 自动布局 - 分层布局
// ═══════════════════════════════════════
function autoLayout(nodes: TopoNode[], _edges: TopoEdge[]): TopoNode[] {
  // 层级顺序：node → namespace → ingress → deployment/statefulset/daemonset → service → pod → configmap/secret/pvc
  const layerOrder = [
    'node',
    'namespace',
    'ingress',
    'deployment',
    'statefulset',
    'daemonset',
    'service',
    'pod',
    'configmap',
    'secret',
    'pvc',
  ]
  const layers: Record<string, TopoNode[]> = {}
  layerOrder.forEach((k) => (layers[k] = []))
  nodes.forEach((n) => {
    if (layers[n.type]) layers[n.type]!.push(n)
  })
  // 每层内按名称排序，减少重叠
  Object.values(layers).forEach((arr) => arr.sort((a, b) => a.name.localeCompare(b.name)))

  const nodeMap = new Map(nodes.map((n) => [n.id, { ...n }]))
  const layerWidth = 300
  const nodeHeight = 90
  const gapX = 60
  const gapY = 50

  let currentX = 60
  layerOrder.forEach((layerKey) => {
    const arr = layers[layerKey] || []
    if (arr.length === 0) return
    arr.forEach((node, idx) => {
      const n = nodeMap.get(node.id)
      if (n) {
        n.x = currentX
        n.y = 60 + idx * (nodeHeight + gapY)
      }
    })
    currentX += layerWidth + gapX
  })

  return Array.from(nodeMap.values())
}

// ═══════════════════════════════════════
// 节点组件
// ═══════════════════════════════════════
interface NodeCardProps {
  node: TopoNode
  selected: boolean
  highlighted: boolean
  onMouseDown: (e: React.MouseEvent) => void
  onClick: () => void
}

const NodeCard: React.FC<NodeCardProps> = React.memo(
  ({ node, selected, highlighted, onMouseDown, onClick }) => {
    const cfg = getConfig(node.type)
    const w = 220
    const h = 80

    const statusColor = useMemo(() => {
      if (!node.status) return cfg.color
      const s = node.status.toLowerCase()
      if (s === 'ready' || s === 'running' || s === 'active') return '#059669'
      if (s === 'pending' || s === 'terminating') return '#d97706'
      if (s === 'failed' || s === 'error' || s === 'notready') return '#dc2626'
      return cfg.color
    }, [node.status, cfg.color])

    return (
      <g
        transform={`translate(${node.x}, ${node.y})`}
        onMouseDown={onMouseDown}
        onClick={onClick}
        style={{ cursor: 'grab' }}
      >
        {/* Shadow */}
        <rect x={3} y={3} width={w} height={h} rx={12} fill="rgba(0,0,0,0.06)" />
        {/* Card */}
        <rect
          width={w}
          height={h}
          rx={12}
          fill="white"
          stroke={selected ? '#2563eb' : highlighted ? '#93c5fd' : '#e5e7eb'}
          strokeWidth={selected ? 3 : highlighted ? 2 : 1}
        />
        {/* Left color bar */}
        <rect x={0} y={0} width={6} height={h} rx={3} fill={cfg.color} />
        {/* Status dot */}
        <circle cx={w - 20} cy={20} r={6} fill={statusColor} />
        {/* Icon placeholder */}
        <rect
          x={20}
          y={18}
          width={36}
          height={36}
          rx={8}
          fill={cfg.bg}
          stroke={cfg.color}
          strokeWidth={1}
        />
        <text x={38} y={42} textAnchor="middle" fontSize={16} fill={cfg.color}>
          ⬡
        </text>
        {/* Name */}
        <text x={66} y={34} fontSize={13} fontWeight={700} fill="#1e293b" style={{ maxWidth: 130 }}>
          {node.name.length > 16 ? node.name.slice(0, 16) + '…' : node.name}
        </text>
        {/* Type + Status */}
        <text x={66} y={52} fontSize={11} fill="#64748b">
          {cfg.label}
        </text>
        {node.namespace && (
          <text x={66} y={66} fontSize={10} fill="#94a3b8">
            {node.namespace}
          </text>
        )}
        {node.status && (
          <text
            x={w - 30}
            y={66}
            textAnchor="end"
            fontSize={10}
            fill={statusColor}
            fontWeight={600}
          >
            {node.status}
          </text>
        )}
      </g>
    )
  },
)

// ═══════════════════════════════════════
// 主组件
// ═══════════════════════════════════════
const K8sTopologyPage: React.FC = () => {
  const clusterId = useClusterId()
  const [nodes, setNodes] = useState<TopoNode[]>([])
  const [edges, setEdges] = useState<TopoEdge[]>([])
  const [selectedNode, setSelectedNode] = useState<TopoNode | null>(null)
  const [yamlContent, setYamlContent] = useState<string>('')
  const [yamlLoading, setYamlLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [zoom, setZoom] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [dragging, setDragging] = useState<{
    nodeId: string
    offsetX: number
    offsetY: number
  } | null>(null)
  const [panning, setPanning] = useState<{
    startX: number
    startY: number
    panX: number
    panY: number
  } | null>(null)
  const [activeTypes, setActiveTypes] = useState<string[]>(RESOURCE_CONFIGS.map((c) => c.key))
  // 筛选模式：all=全量 / namespace=命名空间 / pod=Pod链路
  const [filterMode, setFilterMode] = useState<'all' | 'namespace' | 'pod'>('all')
  const [selectedNamespaces, setSelectedNamespaces] = useState<string[]>([])
  const [selectedPodId, setSelectedPodId] = useState<string | undefined>(undefined)
  const svgRef = useRef<SVGSVGElement>(null)
  const [svgSize, setSvgSize] = useState({ w: 1200, h: 700 })

  // ═══ Load resources ═══
  const topoQueryOpts = { staleTime: 120_000 } as const
  const { data: k8sNodes, isLoading: loadingNodes } = useQuery({
    queryKey: ['k8s-nodes', clusterId],
    queryFn: ({ signal }) => getNodes(clusterId, signal),
    ...topoQueryOpts,
  })
  const { data: k8sNamespaces, isLoading: loadingNS } = useQuery({
    queryKey: ['k8s-topology-namespaces', clusterId],
    queryFn: ({ signal }) => getNamespaces(clusterId, signal),
    ...topoQueryOpts,
  })
  const { data: k8sDeployments, isLoading: loadingDeploy } = useQuery({
    queryKey: ['k8s-deployments', clusterId, ''],
    queryFn: ({ signal }) => getDeployments(clusterId, undefined, signal),
    ...topoQueryOpts,
  })
  const { data: k8sPods, isLoading: loadingPods } = useQuery({
    queryKey: ['k8s-pods', clusterId, ''],
    queryFn: ({ signal }) => getPods(clusterId, undefined, signal),
    ...topoQueryOpts,
  })
  const { data: k8sServices, isLoading: loadingSvc } = useQuery({
    queryKey: ['k8s-services', clusterId, ''],
    queryFn: ({ signal }) => getK8sServices(clusterId, undefined, signal),
    ...topoQueryOpts,
  })
  const { data: k8sConfigMaps, isLoading: loadingCM } = useQuery({
    queryKey: ['k8s-configmaps', clusterId, ''],
    queryFn: ({ signal }) => getConfigMaps(clusterId, undefined, signal),
    ...topoQueryOpts,
  })
  // StatefulSet
  const { data: k8sStatefulSets, isLoading: loadingSts } = useQuery({
    queryKey: ['k8s-statefulsets', clusterId, ''],
    queryFn: ({ signal }) => listGenericResources(Number(clusterId), 'statefulsets', undefined, signal),
    enabled: !!clusterId,
    ...topoQueryOpts,
  })
  // DaemonSet
  const { data: k8sDaemonSets, isLoading: loadingDS } = useQuery({
    queryKey: ['k8s-daemonsets', clusterId, ''],
    queryFn: ({ signal }) => listGenericResources(Number(clusterId), 'daemonsets', undefined, signal),
    enabled: !!clusterId,
    ...topoQueryOpts,
  })
  // Secret
  const { data: k8sSecrets, isLoading: loadingSecret } = useQuery({
    queryKey: ['k8s-secrets-topo', clusterId, ''],
    queryFn: ({ signal }) => listSecrets(Number(clusterId), undefined, signal),
    enabled: !!clusterId,
    ...topoQueryOpts,
  })
  // PVC
  const { data: k8sPVCs, isLoading: loadingPVC } = useQuery({
    queryKey: ['k8s-pvcs-topo', clusterId, ''],
    queryFn: ({ signal }) => listGenericResources(Number(clusterId), 'persistentvolumeclaims', undefined, signal),
    enabled: !!clusterId,
    ...topoQueryOpts,
  })
  // Ingress
  const { data: k8sIngresses, isLoading: loadingIngress } = useQuery({
    queryKey: ['k8s-ingresses-topo', clusterId, ''],
    queryFn: ({ signal }) => listIngresses(Number(clusterId), undefined, signal),
    enabled: !!clusterId,
    ...topoQueryOpts,
  })

  const isLoading =
    loadingNodes ||
    loadingNS ||
    loadingDeploy ||
    loadingPods ||
    loadingSvc ||
    loadingCM ||
    loadingSts ||
    loadingDS ||
    loadingSecret ||
    loadingPVC ||
    loadingIngress

  // ═══ Build topology ═══
  // 限制各类型最大展示数量，避免大集群节点过多导致卡顿
  const MAX_PER_TYPE = 50
  useEffect(() => {
    if (isLoading) return
    const topoNodes: TopoNode[] = []
    const topoEdges: TopoEdge[] = []

    k8sNodes?.items?.slice(0, MAX_PER_TYPE).forEach((n) => {
      topoNodes.push({
        id: `node:${n.name}`,
        type: 'node',
        name: n.name,
        status: n.status,
        x: 0,
        y: 0,
        meta: n,
      })
    })

    k8sNamespaces?.items?.slice(0, MAX_PER_TYPE).forEach((ns) => {
      topoNodes.push({
        id: `ns:${ns.name}`,
        type: 'namespace',
        name: ns.name,
        status: ns.status,
        x: 0,
        y: 0,
        meta: ns,
      })
    })

    k8sDeployments?.items?.slice(0, MAX_PER_TYPE).forEach((d) => {
      const id = `deploy:${d.namespace}/${d.name}`
      topoNodes.push({
        id,
        type: 'deployment',
        name: d.name,
        namespace: d.namespace,
        status: `${d.readyReplicas || 0}/${d.replicas}`,
        x: 0,
        y: 0,
        meta: d,
      })
      const nsEdge = topoEdges.find((e) => e.source === `ns:${d.namespace}`)
      if (!nsEdge)
        topoEdges.push({ id: `ns:${d.namespace}->${id}`, source: `ns:${d.namespace}`, target: id, type: 'contains' })
    })

    k8sPods?.items?.slice(0, MAX_PER_TYPE).forEach((p) => {
      const id = `pod:${p.namespace}/${p.name}`
      topoNodes.push({
        id,
        type: 'pod',
        name: p.name,
        namespace: p.namespace,
        status: p.status,
        x: 0,
        y: 0,
        meta: p,
      })
      if (p.ownerName) {
        // 根据 ownerKind 判断前缀：StatefulSet→sts / DaemonSet→ds / 其他→deploy
        const ownerKindLower = (p.ownerKind || '').toLowerCase()
        const ownerPrefix =
          ownerKindLower === 'statefulset' ? 'sts' : ownerKindLower === 'daemonset' ? 'ds' : 'deploy'
        const ownerId = `${ownerPrefix}:${p.namespace}/${p.ownerName}`
        topoEdges.push({ id: `${ownerId}->${id}`, source: ownerId, target: id, label: 'owns', type: 'owns' })
      }
      // 通过 Pod volumes 建立 ConfigMap/Secret/PVC → Pod 的连线
      if (Array.isArray(p.volumes)) {
        for (const vol of p.volumes) {
          if (!vol.source) continue
          if (vol.type === 'ConfigMap') {
            const cmId = `cm:${p.namespace}/${vol.source}`
            topoEdges.push({ id: `${cmId}->${id}`, source: cmId, target: id, label: 'mounts', type: 'uses' })
          } else if (vol.type === 'Secret') {
            const secretId = `secret:${p.namespace}/${vol.source}`
            topoEdges.push({ id: `${secretId}->${id}`, source: secretId, target: id, label: 'mounts', type: 'uses' })
          } else if (vol.type === 'PVC') {
            const pvcId = `pvc:${p.namespace}/${vol.source}`
            topoEdges.push({ id: `${pvcId}->${id}`, source: pvcId, target: id, label: 'mounts', type: 'uses' })
          }
        }
      }
    })

    k8sServices?.items?.slice(0, MAX_PER_TYPE).forEach((s) => {
      const id = `svc:${s.namespace}/${s.name}`
      topoNodes.push({
        id,
        type: 'service',
        name: s.name,
        namespace: s.namespace,
        status: s.type,
        x: 0,
        y: 0,
        meta: s,
      })
      topoEdges.push({ id: `ns:${s.namespace}->${id}`, source: `ns:${s.namespace}`, target: id, type: 'contains' })
      // 通过 selector 匹配 Pod labels，建立 Service → Pod 连线
      if (s.selector && typeof s.selector === 'object') {
        const selectorEntries = Object.entries(s.selector)
        if (selectorEntries.length > 0) {
          k8sPods?.items?.slice(0, MAX_PER_TYPE).forEach((p) => {
            if (p.namespace !== s.namespace) return
            const podLabels = p.labels || {}
            const match = selectorEntries.every(([k, v]) => podLabels[k] === v)
            if (match) {
              const podId = `pod:${p.namespace}/${p.name}`
              topoEdges.push({ id: `${id}->${podId}`, source: id, target: podId, label: 'selects', type: 'routes' })
            }
          })
        }
      }
    })

    k8sConfigMaps?.items?.slice(0, MAX_PER_TYPE).forEach((cm) => {
      const id = `cm:${cm.namespace}/${cm.name}`
      topoNodes.push({
        id,
        type: 'configmap',
        name: cm.name,
        namespace: cm.namespace,
        x: 0,
        y: 0,
        meta: cm,
      })
      topoEdges.push({ id: `ns:${cm.namespace}->${id}`, source: `ns:${cm.namespace}`, target: id, type: 'contains' })
    })

    // StatefulSet 节点
    k8sStatefulSets?.items?.slice(0, MAX_PER_TYPE).forEach((st) => {
      const id = `sts:${st.namespace}/${st.name}`
      topoNodes.push({
        id,
        type: 'statefulset',
        name: st.name,
        namespace: st.namespace,
        status: `${st.raw?.status?.readyReplicas || 0}/${st.raw?.status?.replicas || 0}`,
        x: 0,
        y: 0,
        meta: st,
      })
      // 关联到 namespace
      topoEdges.push({
        id: `ns:${st.namespace}->${id}`,
        source: `ns:${st.namespace}`,
        target: id,
        label: 'contains',
        type: 'contains',
      })
    })

    // DaemonSet 节点
    k8sDaemonSets?.items?.slice(0, MAX_PER_TYPE).forEach((ds) => {
      const id = `ds:${ds.namespace}/${ds.name}`
      topoNodes.push({
        id,
        type: 'daemonset',
        name: ds.name,
        namespace: ds.namespace,
        status: `${ds.raw?.status?.numberReady || 0}/${ds.raw?.status?.desiredNumberScheduled || 0}`,
        x: 0,
        y: 0,
        meta: ds,
      })
      // 关联到 namespace
      topoEdges.push({
        id: `ns:${ds.namespace}->${id}`,
        source: `ns:${ds.namespace}`,
        target: id,
        label: 'contains',
        type: 'contains',
      })
    })

    // Secret 节点
    k8sSecrets?.items?.slice(0, MAX_PER_TYPE).forEach((s) => {
      const id = `secret:${s.namespace}/${s.name}`
      topoNodes.push({
        id,
        type: 'secret',
        name: s.name,
        namespace: s.namespace,
        status: s.type,
        x: 0,
        y: 0,
        meta: s,
      })
      topoEdges.push({ id: `ns:${s.namespace}->${id}`, source: `ns:${s.namespace}`, target: id, type: 'contains' })
    })

    // PVC 节点
    k8sPVCs?.items?.slice(0, MAX_PER_TYPE).forEach((pvc) => {
      const id = `pvc:${pvc.namespace}/${pvc.name}`
      topoNodes.push({
        id,
        type: 'pvc',
        name: pvc.name,
        namespace: pvc.namespace,
        status: pvc.raw?.status?.phase,
        x: 0,
        y: 0,
        meta: pvc,
      })
      topoEdges.push({ id: `ns:${pvc.namespace}->${id}`, source: `ns:${pvc.namespace}`, target: id, type: 'contains' })
    })

    // Ingress 节点 + 关联到 Service
    k8sIngresses?.items?.slice(0, MAX_PER_TYPE).forEach((ing) => {
      const id = `ingress:${ing.namespace}/${ing.name}`
      topoNodes.push({
        id,
        type: 'ingress',
        name: ing.name,
        namespace: ing.namespace,
        x: 0,
        y: 0,
        meta: ing,
      })
      // 关联到 Service（Ingress.rules 直接包含 spec.rules）
      const rules = ing.rules || []
      rules.forEach((rule: any) => {
        const paths = rule.http?.paths || []
        paths.forEach((p: any) => {
          const svcName = p.backend?.service?.name || p.backend?.serviceName
          if (svcName) {
            const svcId = `svc:${ing.namespace}/${svcName}`
            topoEdges.push({
              id: `${id}->${svcId}`,
              source: id,
              target: svcId,
              label: 'routes',
              type: 'routes',
            })
          }
        })
      })
    })

    const laid = autoLayout(topoNodes, topoEdges)
    setNodes(laid)
    setEdges(topoEdges)
  }, [
    k8sNodes,
    k8sNamespaces,
    k8sDeployments,
    k8sPods,
    k8sServices,
    k8sConfigMaps,
    k8sStatefulSets,
    k8sDaemonSets,
    k8sSecrets,
    k8sPVCs,
    k8sIngresses,
    isLoading,
  ])

  // ═══ Responsive SVG size ═══
  useEffect(() => {
    const onResize = () => {
      const container = svgRef.current?.parentElement
      if (container)
        setSvgSize({ w: container.clientWidth, h: Math.max(600, window.innerHeight - 240) })
    }
    onResize()
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])

  // ═══ Drag handlers ═══
  const handleNodeMouseDown = useCallback(
    (nodeId: string, e: React.MouseEvent) => {
      e.stopPropagation()
      const node = nodes.find((n) => n.id === nodeId)
      if (!node) return
      const svgRect = svgRef.current?.getBoundingClientRect()
      if (!svgRect) return
      const mouseX = (e.clientX - svgRect.left - pan.x) / zoom
      const mouseY = (e.clientY - svgRect.top - pan.y) / zoom
      setDragging({ nodeId, offsetX: mouseX - node.x, offsetY: mouseY - node.y })
    },
    [nodes, pan, zoom],
  )

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (dragging) {
        const svgRect = svgRef.current?.getBoundingClientRect()
        if (!svgRect) return
        const mouseX = (e.clientX - svgRect.left - pan.x) / zoom
        const mouseY = (e.clientY - svgRect.top - pan.y) / zoom
        setNodes((prev) =>
          prev.map((n) =>
            n.id === dragging.nodeId
              ? { ...n, x: mouseX - dragging.offsetX, y: mouseY - dragging.offsetY }
              : n,
          ),
        )
      } else if (panning) {
        setPan({
          x: panning.panX + (e.clientX - panning.startX),
          y: panning.panY + (e.clientY - panning.startY),
        })
      }
    },
    [dragging, panning, pan, zoom],
  )

  const handleMouseUp = useCallback(() => {
    setDragging(null)
    setPanning(null)
  }, [])

  const handleSvgMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (
        e.target === svgRef.current ||
        ((e.target as SVGElement).tagName === 'rect' &&
          (e.target as SVGElement).getAttribute('data-bg'))
      ) {
        setPanning({ startX: e.clientX, startY: e.clientY, panX: pan.x, panY: pan.y })
        setSelectedNode(null)
      }
    },
    [pan],
  )

  // ═══ Select node ═══
  const handleNodeClick = useCallback(
    async (node: TopoNode) => {
      setSelectedNode(node)
      setYamlContent('')
      const resourceType = NODE_TYPE_TO_RESOURCE[node.type]
      if (resourceType) {
        const ns = node.namespace || 'default'
        setYamlLoading(true)
        try {
          const res = await getResourceYaml(Number(clusterId), resourceType, ns, node.name)
          // 替换 YAML 内容中 JSON 转义的 \n 字面量为真实换行，确保 pre 正确显示
          setYamlContent((res.yaml || '').replace(/\\n/g, '\n'))
        } catch (e: any) {
          const errMsg = e?.message || e?.response?.data?.message || String(e)
          setYamlContent(`# 获取 YAML 失败\n# 错误: ${errMsg}\n# 资源: ${resourceType}/${ns}/${node.name}`)
        } finally {
          setYamlLoading(false)
        }
      }
    },
    [clusterId],
  )

  // ═══ Search & Filter ═══
  const filteredNodes = useMemo(() => {
    let result = nodes.filter((n) => {
      if (!activeTypes.includes(n.type)) return false
      if (
        search &&
        !n.name.toLowerCase().includes(search.toLowerCase()) &&
        !(n.namespace || '').toLowerCase().includes(search.toLowerCase())
      )
        return false
      return true
    })

    // 命名空间模式：仅展示选中命名空间下的资源
    if (filterMode === 'namespace' && selectedNamespaces.length > 0) {
      const nsSet = new Set(selectedNamespaces)
      result = result.filter((n) => {
        // 节点（node）为集群级资源，保留展示
        if (n.type === 'node') return true
        // 命名空间节点按名称匹配
        if (n.type === 'namespace') return nsSet.has(n.name)
        return !!n.namespace && nsSet.has(n.namespace)
      })
    }

    // Pod 链路模式：仅展示选中 Pod 的全链路关联节点
    if (filterMode === 'pod' && selectedPodId) {
      const relatedIds = findRelatedNodeIds(selectedPodId, edges)
      result = result.filter((n) => relatedIds.has(n.id))
    }

    return result
  }, [nodes, search, activeTypes, filterMode, selectedNamespaces, selectedPodId, edges])

  const filteredEdges = useMemo(() => {
    const nodeIds = new Set(filteredNodes.map((n) => n.id))
    return edges.filter((e) => nodeIds.has(e.source) && nodeIds.has(e.target))
  }, [edges, filteredNodes])

  // 全链路高亮：BFS 递归查找所有关联节点
  const highlightedIds = useMemo(() => {
    if (!selectedNode) return new Set<string>()
    return findRelatedNodeIds(selectedNode.id, edges)
  }, [selectedNode, edges])

  // ═══ Stats ═══
  const stats = useMemo(
    () => ({
      nodes: nodes.filter((n) => n.type === 'node').length,
      namespaces: nodes.filter((n) => n.type === 'namespace').length,
      deployments: nodes.filter((n) => n.type === 'deployment').length,
      statefulsets: nodes.filter((n) => n.type === 'statefulset').length,
      daemonsets: nodes.filter((n) => n.type === 'daemonset').length,
      pods: nodes.filter((n) => n.type === 'pod').length,
      services: nodes.filter((n) => n.type === 'service').length,
      configmaps: nodes.filter((n) => n.type === 'configmap').length,
      secrets: nodes.filter((n) => n.type === 'secret').length,
      pvcs: nodes.filter((n) => n.type === 'pvc').length,
      ingresses: nodes.filter((n) => n.type === 'ingress').length,
    }),
    [nodes],
  )

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />

  // 重置所有筛选条件
  const handleResetFilter = () => {
    setSearch('')
    setActiveTypes(RESOURCE_CONFIGS.map((c) => c.key))
    setFilterMode('all')
    setSelectedNamespaces([])
    setSelectedPodId(undefined)
  }

  // Pod 选项（用于 Pod 链路模式选择）
  const podOptions = (k8sPods?.items || []).map((p) => ({
    value: `pod:${p.namespace}/${p.name}`,
    label: `${p.namespace}/${p.name}`,
  }))

  return (
    <AppPage>
      {/* ═══ Toolbar ═══ */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Row justify="space-between" align="middle">
          <Col>
            <Space size={12} wrap>
              <Radio.Group
                value={filterMode}
                onChange={(e) => setFilterMode(e.target.value)}
                optionType="button"
                buttonStyle="solid"
                size="small"
              >
                <Radio.Button value="all">全量模式</Radio.Button>
                <Radio.Button value="namespace">命名空间模式</Radio.Button>
                <Radio.Button value="pod">Pod链路模式</Radio.Button>
              </Radio.Group>
              <Button size="small" onClick={handleResetFilter}>
                重置筛选
              </Button>
              <Input
                placeholder="搜索资源名称或命名空间"
                prefix={<SearchOutlined />}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                style={{ width: 240 }}
                allowClear
              />
              <Select
                mode="multiple"
                placeholder="资源类型"
                value={activeTypes}
                onChange={setActiveTypes}
                style={{ minWidth: 280 }}
                maxTagCount={3}
                options={RESOURCE_CONFIGS.map((c) => ({
                  value: c.key,
                  label: (
                    <Space>
                      {c.icon}
                      {c.label} ({(stats as any)[c.key + 's'] || (stats as any)[c.key] || 0})
                    </Space>
                  ),
                }))}
              />
              {/* 命名空间模式：命名空间多选 */}
              {filterMode === 'namespace' && (
                <Select
                  mode="multiple"
                  placeholder="选择命名空间"
                  value={selectedNamespaces}
                  onChange={setSelectedNamespaces}
                  style={{ minWidth: 240 }}
                  maxTagCount={3}
                  options={(k8sNamespaces?.items || []).map((ns) => ({
                    value: ns.name,
                    label: ns.name,
                  }))}
                />
              )}
              {/* Pod 链路模式：选择目标 Pod */}
              {filterMode === 'pod' && (
                <Select
                  showSearch
                  placeholder="选择目标 Pod"
                  value={selectedPodId}
                  onChange={setSelectedPodId}
                  style={{ minWidth: 260 }}
                  allowClear
                  filterOption={(input, option) =>
                    (option?.label as string).toLowerCase().includes(input.toLowerCase())
                  }
                  options={podOptions}
                />
              )}
            </Space>
          </Col>
          <Col>
            <Space>
              <Tooltip title="放大">
                <Button
                  icon={<ZoomInOutlined />}
                  onClick={() => setZoom((z) => Math.min(2, z + 0.2))}
                />
              </Tooltip>
              <Tooltip title="缩小">
                <Button
                  icon={<ZoomOutOutlined />}
                  onClick={() => setZoom((z) => Math.max(0.3, z - 0.2))}
                />
              </Tooltip>
              <Tooltip title="重置视图">
                <Button
                  icon={<CompressOutlined />}
                  onClick={() => {
                    setZoom(1)
                    setPan({ x: 0, y: 0 })
                  }}
                />
              </Tooltip>
              <Tooltip title="自动布局">
                <Button
                  icon={<ApartmentOutlined />}
                  onClick={() => setNodes((prev) => autoLayout(prev, edges))}
                />
              </Tooltip>
              <Tooltip title="刷新">
                <Button icon={<ReloadOutlined />} onClick={() => window.location.reload()} />
              </Tooltip>
              <Text type="secondary" style={{ fontSize: 11 }}>
                {Math.round(zoom * 100)}%
              </Text>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* ═══ Legend ═══ */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space size={16} wrap>
          <Text type="secondary" style={{ fontSize: 11 }}>
            图例：
          </Text>
          {RESOURCE_CONFIGS.filter((c) => activeTypes.includes(c.key)).map((cfg) => (
            <Space key={cfg.key} size={4}>
              <div
                style={{
                  width: 14,
                  height: 14,
                  borderRadius: 4,
                  background: cfg.color,
                  opacity: 0.85,
                }}
              />
              <Text style={{ fontSize: 11 }}>{cfg.label}</Text>
            </Space>
          ))}
          <Divider type="vertical" />
          <Space size={4}>
            <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#059669' }} />
            <Text style={{ fontSize: 11 }}>Ready/Running</Text>
          </Space>
          <Space size={4}>
            <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#d97706' }} />
            <Text style={{ fontSize: 11 }}>Pending</Text>
          </Space>
          <Space size={4}>
            <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#dc2626' }} />
            <Text style={{ fontSize: 11 }}>Failed</Text>
          </Space>
        </Space>
      </Card>

      {/* ═══ Canvas ═══ */}
      <Card bodyStyle={{ padding: 0 }} style={{ overflow: 'hidden', marginBottom: 12 }}>
        <svg
              ref={svgRef}
              width={svgSize.w}
              height={svgSize.h}
              style={{ background: '#f8fafc', display: 'block' }}
              onMouseDown={handleSvgMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
            >
              <defs>
                <marker
                  id="arrowhead"
                  markerWidth={8}
                  markerHeight={6}
                  refX={8}
                  refY={3}
                  orient="auto"
                >
                  <polygon points="0 0, 8 3, 0 6" fill="#94a3b8" />
                </marker>
                <pattern id="grid" width={20} height={20} patternUnits="userSpaceOnUse">
                  <circle cx={1} cy={1} r={0.5} fill="#e2e8f0" />
                </pattern>
              </defs>
              <rect data-bg="true" width="100%" height="100%" fill="url(#grid)" />
              <g transform={`translate(${pan.x}, ${pan.y}) scale(${zoom})`}>
                {/* Edges */}
                {filteredEdges.map((edge) => {
                  const src = filteredNodes.find((n) => n.id === edge.source)
                  const tgt = filteredNodes.find((n) => n.id === edge.target)
                  if (!src || !tgt) return null
                  const x1 = src.x + 220
                  const y1 = src.y + 40
                  const x2 = tgt.x
                  const y2 = tgt.y + 40
                  const mx = (x1 + x2) / 2
                  // 高亮：两端节点均在全链路高亮集合中
                  const isHighlighted =
                    !!selectedNode &&
                    highlightedIds.has(edge.source) &&
                    highlightedIds.has(edge.target)
                  const style = edgeStyleMap[edge.type || 'default'] || edgeStyleMap.default!
                  return (
                    <g
                      key={edge.id}
                      style={{ cursor: 'pointer' }}
                      onClick={() => {
                        const srcLabel = getConfig(src.type).label
                        const tgtLabel = getConfig(tgt.type).label
                        const rel = edge.label || '关联'
                        message.info(`${srcLabel} ${src.name} ${rel} ${tgtLabel} ${tgt.name}`)
                      }}
                    >
                      <path
                        d={`M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}`}
                        fill="none"
                        stroke={style.stroke}
                        strokeWidth={isHighlighted ? style.width + 1 : style.width}
                        strokeDasharray={isHighlighted ? undefined : style.dasharray}
                        markerEnd="url(#arrowhead)"
                      />
                      {edge.label && (
                        <text
                          x={mx}
                          y={(y1 + y2) / 2 - 6}
                          textAnchor="middle"
                          fontSize={9}
                          fill={isHighlighted ? style.stroke : '#94a3b8'}
                        >
                          {edge.label}
                        </text>
                      )}
                    </g>
                  )
                })}
                {/* Nodes */}
                {filteredNodes.map((node) => (
                  <NodeCard
                    key={node.id}
                    node={node}
                    selected={selectedNode?.id === node.id}
                    highlighted={highlightedIds.has(node.id)}
                    onMouseDown={(e) => handleNodeMouseDown(node.id, e)}
                    onClick={() => handleNodeClick(node)}
                  />
                ))}
              </g>
            </svg>
          </Card>

          {/* ═══ Detail Panel ═══ */}
          <Card
            title={
              selectedNode ? (
                <Space>
                  {getConfig(selectedNode.type).icon}
                  {selectedNode.name}
                </Space>
              ) : (
                <Space>
                  <InfoCircleOutlined />
                  资源详情
                </Space>
              )
            }
            bodyStyle={{ padding: 16 }}
          >
            {selectedNode ? (
              <Row gutter={24}>
                <Col xs={24} lg={8}>
              <div style={{ background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: 8, padding: 16 }}>
                <Descriptions column={1} labelStyle={{ width: 100 }}>
                  <Descriptions.Item label="类型">
                    <Tag color={getConfig(selectedNode.type).color}>
                      {getConfig(selectedNode.type).label}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="名称">{selectedNode.name}</Descriptions.Item>
                  {selectedNode.namespace && (
                    <Descriptions.Item label="命名空间">{selectedNode.namespace}</Descriptions.Item>
                  )}
                  {selectedNode.status && (
                    <Descriptions.Item label="状态">
                      <Badge
                        status={
                          selectedNode.status === 'Running' || selectedNode.status === 'Ready'
                            ? 'success'
                            : selectedNode.status === 'Pending'
                              ? 'warning'
                              : 'error'
                        }
                        text={selectedNode.status}
                      />
                    </Descriptions.Item>
                  )}
                </Descriptions>

                {/* Resource-specific details */}
                {selectedNode.type === 'pod' && selectedNode.meta && (
                  <>
                    <Divider style={{ margin: '12px 0' }} />
                    <Descriptions column={1} labelStyle={{ width: 100 }}>
                      <Descriptions.Item label="Pod IP">
                        {selectedNode.meta.podIP || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="节点">
                        {selectedNode.meta.nodeName || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="QoS">
                        {selectedNode.meta.qosClass || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="重启次数">
                        {selectedNode.meta.restartCount ?? 0}
                      </Descriptions.Item>
                      <Descriptions.Item label="所属">
                        {selectedNode.meta.ownerKind}/{selectedNode.meta.ownerName}
                      </Descriptions.Item>
                    </Descriptions>
                  </>
                )}

                {selectedNode.type === 'deployment' && selectedNode.meta && (
                  <>
                    <Divider style={{ margin: '12px 0' }} />
                    <Descriptions column={1} labelStyle={{ width: 100 }}>
                      <Descriptions.Item label="副本数">
                        {selectedNode.meta.readyReplicas || 0}/{selectedNode.meta.replicas}
                      </Descriptions.Item>
                      <Descriptions.Item label="策略">
                        {selectedNode.meta.strategy || 'RollingUpdate'}
                      </Descriptions.Item>
                      <Descriptions.Item label="创建时间">
                        {selectedNode.meta.creationTimestamp || '-'}
                      </Descriptions.Item>
                    </Descriptions>
                  </>
                )}

                {selectedNode.type === 'node' && selectedNode.meta && (
                  <>
                    <Divider style={{ margin: '12px 0' }} />
                    <Descriptions column={1} labelStyle={{ width: 100 }}>
                      <Descriptions.Item label="IP">
                        {selectedNode.meta.ip || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="版本">
                        {selectedNode.meta.kubeletVersion || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="OS">
                        {selectedNode.meta.osImage || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="CPU">
                        {selectedNode.meta.cpu || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="内存">
                        {selectedNode.meta.memory || '-'}
                      </Descriptions.Item>
                    </Descriptions>
                  </>
                )}

                {selectedNode.type === 'service' && selectedNode.meta && (
                  <>
                    <Divider style={{ margin: '12px 0' }} />
                    <Descriptions column={1} labelStyle={{ width: 100 }}>
                      <Descriptions.Item label="类型">{selectedNode.meta.type}</Descriptions.Item>
                      <Descriptions.Item label="ClusterIP">
                        {selectedNode.meta.clusterIP || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="端口">
                        {selectedNode.meta.ports
                          ?.map((p: any) => `${p.port}:${p.targetPort}/${p.protocol}`)
                          .join(', ') || '-'}
                      </Descriptions.Item>
                    </Descriptions>
                  </>
                )}

                </div>
                </Col>
                <Col xs={24} lg={16}>
                <Divider style={{ margin: '0 0 16px 0' }} />
                <Text strong style={{ fontSize: 13, display: 'block', marginBottom: 8 }}>
                  YAML 预览
                </Text>
                {yamlLoading ? (
                  <Spin size="small" style={{ display: 'block', marginTop: 8 }} />
                ) : (
                  <YamlEditor
                    value={yamlContent || '# 选择资源后加载 YAML'}
                    readOnly
                    height={400}
                  />
                )}
                </Col>
              </Row>
            ) : (
              <Empty description="点击节点查看详情" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>

      {/* ═══ Stats Bar ═══ */}
      <Card size="small" style={{ marginTop: 12 }}>
        <Space size={24} wrap>
          <Text type="secondary" style={{ fontSize: 11 }}>
            资源统计：
          </Text>
          {RESOURCE_CONFIGS.map((cfg) => {
            const count = (stats as any)[cfg.key + 's'] || (stats as any)[cfg.key] || 0
            return (
              <Space key={cfg.key} size={4}>
                <div style={{ width: 10, height: 10, borderRadius: 3, background: cfg.color }} />
                <Text style={{ fontSize: 11 }}>
                  {cfg.label}: {count}
                </Text>
              </Space>
            )
          })}
          <Divider type="vertical" />
          <Text type="secondary" style={{ fontSize: 11 }}>
            连接: {filteredEdges.length}
          </Text>
          <Text type="secondary" style={{ fontSize: 11 }}>
            缩放: {Math.round(zoom * 100)}%
          </Text>
        </Space>
      </Card>
    </AppPage>
  )
}

export default K8sTopologyPage
