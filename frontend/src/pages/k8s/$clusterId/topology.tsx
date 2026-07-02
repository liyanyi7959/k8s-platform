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
import { AppPage } from '@/components'
import { useQuery } from '@tanstack/react-query'
import {
  getNodes,
  getNamespaces,
  getDeployments,
  getPods,
  getK8sServices,
  getConfigMaps,
  getResourceYaml,
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
}

// ═══════════════════════════════════════
// 自动布局 - 分层布局
// ═══════════════════════════════════════
function autoLayout(nodes: TopoNode[], _edges: TopoEdge[]): TopoNode[] {
  const layers: Record<string, string[]> = {
    node: [],
    namespace: [],
    deployment: [],
    pod: [],
    service: [],
    configmap: [],
  }
  nodes.forEach((n) => {
    if (layers[n.type]) layers[n.type]!.push(n.id)
  })

  const layerOrder = ['node', 'namespace', 'deployment', 'service', 'pod', 'configmap']
  const nodeMap = new Map(nodes.map((n) => [n.id, { ...n }]))
  const layerWidth = 280
  const nodeHeight = 100
  const gapX = 60
  const gapY = 40

  let currentX = 60
  layerOrder.forEach((layerKey) => {
    const ids = layers[layerKey] || []
    if (ids.length === 0) return
    ids.forEach((id, idx) => {
      const node = nodeMap.get(id)
      if (node) {
        node.x = currentX
        node.y = 60 + idx * (nodeHeight + gapY)
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
  const svgRef = useRef<SVGSVGElement>(null)
  const [svgSize, setSvgSize] = useState({ w: 1200, h: 700 })

  // ═══ Load resources ═══
  const { data: k8sNodes, isLoading: loadingNodes } = useQuery({
    queryKey: ['k8s-nodes', clusterId],
    queryFn: ({ signal }) => getNodes(clusterId, signal),
  })
  const { data: k8sNamespaces, isLoading: loadingNS } = useQuery({
    queryKey: ['k8s-namespaces', clusterId],
    queryFn: ({ signal }) => getNamespaces(clusterId, signal),
  })
  const { data: k8sDeployments, isLoading: loadingDeploy } = useQuery({
    queryKey: ['k8s-deployments', clusterId, ''],
    queryFn: ({ signal }) => getDeployments(clusterId, undefined, signal),
  })
  const { data: k8sPods, isLoading: loadingPods } = useQuery({
    queryKey: ['k8s-pods', clusterId, ''],
    queryFn: ({ signal }) => getPods(clusterId, undefined, signal),
  })
  const { data: k8sServices, isLoading: loadingSvc } = useQuery({
    queryKey: ['k8s-services', clusterId, ''],
    queryFn: ({ signal }) => getK8sServices(clusterId, undefined, signal),
  })
  const { data: k8sConfigMaps, isLoading: loadingCM } = useQuery({
    queryKey: ['k8s-configmaps', clusterId, ''],
    queryFn: ({ signal }) => getConfigMaps(clusterId, undefined, signal),
  })

  const isLoading =
    loadingNodes || loadingNS || loadingDeploy || loadingPods || loadingSvc || loadingCM

  // ═══ Build topology ═══
  useEffect(() => {
    if (isLoading) return
    const topoNodes: TopoNode[] = []
    const topoEdges: TopoEdge[] = []

    k8sNodes?.items?.forEach((n) => {
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

    k8sNamespaces?.items?.forEach((ns) => {
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

    k8sDeployments?.items?.forEach((d) => {
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
        topoEdges.push({ id: `ns:${d.namespace}->${id}`, source: `ns:${d.namespace}`, target: id })
    })

    k8sPods?.items?.forEach((p) => {
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
        const ownerId = `deploy:${p.namespace}/${p.ownerName}`
        topoEdges.push({ id: `${ownerId}->${id}`, source: ownerId, target: id, label: 'owns' })
      }
    })

    k8sServices?.items?.forEach((s) => {
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
      topoEdges.push({ id: `ns:${s.namespace}->${id}`, source: `ns:${s.namespace}`, target: id })
    })

    k8sConfigMaps?.items?.forEach((cm) => {
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
    })

    const laid = autoLayout(topoNodes, topoEdges)
    setNodes(laid)
    setEdges(topoEdges)
  }, [k8sNodes, k8sNamespaces, k8sDeployments, k8sPods, k8sServices, k8sConfigMaps, isLoading])

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

  // ═══ Zoom ═══
  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    setZoom((z) => Math.min(2, Math.max(0.3, z + (e.deltaY < 0 ? 0.1 : -0.1))))
  }, [])

  // ═══ Select node ═══
  const handleNodeClick = useCallback(
    async (node: TopoNode) => {
      setSelectedNode(node)
      setYamlContent('')
      if (
        node.type === 'node' ||
        node.type === 'deployment' ||
        node.type === 'pod' ||
        node.type === 'service' ||
        node.type === 'configmap'
      ) {
        const resourceType =
          node.type === 'node'
            ? 'nodes'
            : node.type === 'deployment'
              ? 'deployments'
              : node.type === 'pod'
                ? 'pods'
                : node.type === 'service'
                  ? 'services'
                  : 'configmaps'
        const ns = node.namespace || 'default'
        setYamlLoading(true)
        try {
          const res = await getResourceYaml(clusterId, resourceType, ns, node.name)
          setYamlContent(typeof res === 'string' ? res : JSON.stringify(res, null, 2))
        } catch {
          setYamlContent('获取 YAML 失败')
        } finally {
          setYamlLoading(false)
        }
      }
    },
    [clusterId],
  )

  // ═══ Search & Filter ═══
  const filteredNodes = useMemo(() => {
    return nodes.filter((n) => {
      if (!activeTypes.includes(n.type)) return false
      if (
        search &&
        !n.name.toLowerCase().includes(search.toLowerCase()) &&
        !(n.namespace || '').toLowerCase().includes(search.toLowerCase())
      )
        return false
      return true
    })
  }, [nodes, search, activeTypes])

  const filteredEdges = useMemo(() => {
    const nodeIds = new Set(filteredNodes.map((n) => n.id))
    return edges.filter((e) => nodeIds.has(e.source) && nodeIds.has(e.target))
  }, [edges, filteredNodes])

  const highlightedIds = useMemo(() => {
    if (!selectedNode) return new Set<string>()
    const ids = new Set<string>([selectedNode.id])
    edges.forEach((e) => {
      if (e.source === selectedNode.id) ids.add(e.target)
      if (e.target === selectedNode.id) ids.add(e.source)
    })
    return ids
  }, [selectedNode, edges])

  // ═══ Stats ═══
  const stats = useMemo(
    () => ({
      nodes: nodes.filter((n) => n.type === 'node').length,
      namespaces: nodes.filter((n) => n.type === 'namespace').length,
      deployments: nodes.filter((n) => n.type === 'deployment').length,
      pods: nodes.filter((n) => n.type === 'pod').length,
      services: nodes.filter((n) => n.type === 'service').length,
      configmaps: nodes.filter((n) => n.type === 'configmap').length,
    }),
    [nodes],
  )

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />

  return (
    <AppPage>
      {/* ═══ Toolbar ═══ */}
      <Card size="small" style={{ marginBottom: 12 }}>
        <Row justify="space-between" align="middle">
          <Col>
            <Space size={12} wrap>
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

      {/* ═══ Canvas + Detail ═══ */}
      <Row gutter={12}>
        <Col flex="auto">
          <Card bodyStyle={{ padding: 0 }} style={{ overflow: 'hidden' }}>
            <svg
              ref={svgRef}
              width={svgSize.w}
              height={svgSize.h}
              style={{ background: '#f8fafc', display: 'block' }}
              onMouseDown={handleSvgMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
              onWheel={handleWheel}
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
                  const isHighlighted =
                    selectedNode &&
                    (edge.source === selectedNode.id || edge.target === selectedNode.id)
                  return (
                    <g key={edge.id}>
                      <path
                        d={`M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}`}
                        fill="none"
                        stroke={isHighlighted ? '#2563eb' : '#cbd5e1'}
                        strokeWidth={isHighlighted ? 2.5 : 1.5}
                        strokeDasharray={isHighlighted ? undefined : '6,4'}
                        markerEnd="url(#arrowhead)"
                      />
                      {edge.label && (
                        <text
                          x={mx}
                          y={(y1 + y2) / 2 - 6}
                          textAnchor="middle"
                          fontSize={9}
                          fill="#94a3b8"
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
        </Col>

        {/* ═══ Detail Panel ═══ */}
        <Col flex="360px">
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
            size="small"
            style={{ height: svgSize.h + 2, overflow: 'auto' }}
            bodyStyle={{ padding: 12 }}
          >
            {selectedNode ? (
              <>
                <Descriptions column={1} size="small" labelStyle={{ width: 80 }}>
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
                    <Descriptions column={1} size="small" labelStyle={{ width: 80 }}>
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
                    <Descriptions column={1} size="small" labelStyle={{ width: 80 }}>
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
                    <Descriptions column={1} size="small" labelStyle={{ width: 80 }}>
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
                    <Descriptions column={1} size="small" labelStyle={{ width: 80 }}>
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

                {/* YAML Preview */}
                <Divider style={{ margin: '12px 0' }} />
                <Text strong style={{ fontSize: 12 }}>
                  YAML 预览
                </Text>
                {yamlLoading ? (
                  <Spin size="small" style={{ display: 'block', marginTop: 8 }} />
                ) : (
                  <pre
                    style={{
                      background: '#1e1e1e',
                      color: '#d4d4d4',
                      padding: 12,
                      borderRadius: 8,
                      fontSize: 11,
                      lineHeight: 1.5,
                      maxHeight: 300,
                      overflow: 'auto',
                      marginTop: 8,
                      fontFamily: 'Consolas, Monaco, monospace',
                    }}
                  >
                    {yamlContent || '选择资源后加载 YAML'}
                  </pre>
                )}
              </>
            ) : (
              <Empty description="点击节点查看详情" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </Col>
      </Row>

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
