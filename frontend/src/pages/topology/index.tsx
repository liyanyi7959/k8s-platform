import React, { useEffect, useMemo, useState } from 'react'
import { history, useModel } from '@umijs/max'
import {
  Alert,
  Button,
  Empty,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd'
import {
  ClusterOutlined,
  HddOutlined,
  NodeIndexOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import {
  Background,
  Controls,
  MarkerType,
  MiniMap,
  ReactFlow,
  type Edge,
  type Node,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { ProCard } from '@ant-design/pro-components'
import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'
import { getTopology } from '@/services/k8s'

const { Text } = Typography

type TopologyResource = {
  id: string
  type: string
  data?: {
    label?: string
    namespace?: string
    status?: string
    clusterIP?: string
  }
}

const RESOURCE_THEME: Record<string, { color: string; fill: string }> = {
  node: { color: '#0284c7', fill: '#f0f9ff' },
  pod: { color: '#059669', fill: '#ecfdf5' },
}

function buildFlowNodes(items: TopologyResource[]): Node[] {
  const graphItems = items.filter((item) => item.type !== 'service')
  const grouped = graphItems.reduce<Record<string, TopologyResource[]>>((acc, item) => {
    acc[item.type] = acc[item.type] || []
    acc[item.type]!.push(item)
    return acc
  }, {})

  const order = ['node', 'pod']
  const columnX: Record<string, number> = { node: 80, pod: 420 }

  return order.flatMap((type) => {
    const itemsOfType = grouped[type] || []
    return itemsOfType.map((item, index) => {
      const theme = RESOURCE_THEME[type] || RESOURCE_THEME.node || { color: '#2563eb', fill: '#eff6ff' }
      const label = item.data?.label || item.id
      return {
        id: item.id,
        position: { x: columnX[type] || 80, y: 70 + index * 108 },
        data: { label },
        draggable: false,
        selectable: false,
        style: {
          width: 220,
          borderRadius: 14,
          border: `1px solid ${theme.color}33`,
          background: theme.fill,
          color: '#0f172a',
          fontWeight: 700,
          boxShadow: '0 12px 24px rgba(15, 23, 42, 0.08)',
          padding: '18px 20px',
        },
      }
    })
  })
}

function buildFlowEdges(edges: Edge[]): Edge[] {
  return edges.map((edge) => ({
    ...edge,
    type: 'smoothstep',
    animated: true,
    selectable: false,
    markerEnd: { type: MarkerType.ArrowClosed, color: '#94a3b8' },
    style: { stroke: '#94a3b8', strokeWidth: 1.5 },
  }))
}

const TopologyPage: React.FC = () => {
  const { currentCluster } = useModel('cluster')
  const [selectedClusterId, setSelectedClusterId] = useState<string | undefined>(undefined)

  const { data: clustersData, isLoading: clustersLoading } = useQuery({
    queryKey: ['topology-clusters'],
    queryFn: () => listClusters(),
  })

  const clusters = clustersData?.items || []

  useEffect(() => {
    if (selectedClusterId || clusters.length === 0) {
      return
    }
    const preferredCluster = currentCluster?.id && clusters.some((item) => String(item.id) === currentCluster.id)
      ? currentCluster.id
      : String(clusters[0]!.id)
    setSelectedClusterId(preferredCluster)
  }, [clusters, currentCluster?.id, selectedClusterId])

  const { data: topologyData, isLoading, isRefetching, refetch } = useQuery({
    queryKey: ['global-topology-overview', selectedClusterId],
    enabled: Boolean(selectedClusterId),
    queryFn: () => getTopology(Number(selectedClusterId)),
  })

  const selectedCluster = clusters.find((item) => String(item.id) === selectedClusterId)
  const resources = (topologyData?.nodes || []) as TopologyResource[]
  const serviceResources = resources.filter((item) => item.type === 'service')
  const graphNodes = useMemo(() => buildFlowNodes(resources), [resources])
  const graphEdges = useMemo(() => buildFlowEdges((topologyData?.edges || []) as Edge[]), [topologyData?.edges])

  const summary = useMemo(() => {
    const nodeCount = resources.filter((item) => item.type === 'node').length
    const podCount = resources.filter((item) => item.type === 'pod').length
    const serviceCount = serviceResources.length
    const readyNodes = resources.filter((item) => item.type === 'node' && item.data?.status === 'Ready').length
    return {
      nodeCount,
      podCount,
      serviceCount,
      edgeCount: graphEdges.length,
      readyNodes,
    }
  }, [graphEdges.length, resources, serviceResources.length])

  return (
    <AppPage breadcrumbRender={false}>
      <div className="app-data-console">
        <section className="app-data-console__statgrid">
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">节点数量</span>
            <strong className="app-data-console__stat-value">{summary.nodeCount}</strong>
            <span className="app-data-console__stat-hint">就绪节点 {summary.readyNodes} 台</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">Pod 数量</span>
            <strong className="app-data-console__stat-value">{summary.podCount}</strong>
            <span className="app-data-console__stat-hint">按节点归属构建承载关系</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">Service 数量</span>
            <strong className="app-data-console__stat-value">{summary.serviceCount}</strong>
            <span className="app-data-console__stat-hint">结合右侧资源清单辅助排查</span>
          </div>
        </section>

        <section className="app-data-console__filters">
          <div className="app-data-console__filters-left">
            <Select
              style={{ width: 260 }}
              placeholder="选择要查看的集群"
              loading={clustersLoading}
              value={selectedClusterId}
              onChange={setSelectedClusterId}
              options={clusters.map((cluster) => ({
                label: `${cluster.name}${cluster.k8sVersion ? ` · ${cluster.k8sVersion}` : ''}`,
                value: String(cluster.id),
              }))}
            />
            <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
              刷新
            </Button>
          </div>
          <div className="app-data-console__filters-right">
            <span className="app-data-console__meta">
              连接关系 <strong>{summary.edgeCount}</strong>
            </span>
            {selectedClusterId ? (
              <Button onClick={() => history.push(`/k8s/${selectedClusterId}/topology`)}>
                进入集群关系图
              </Button>
            ) : null}
          </div>
        </section>

        <Alert
          type="info"
          showIcon
          message="资源视图用于快速回答“这个集群当前由哪些节点承载 Pod、有哪些 Service 正在对外提供能力”"
          description="当前图谱基于节点、Pod 与 Service 实时构建。Service 与 Pod 的 selector 细粒度映射仍建议在集群关系图中继续深入查看。"
        />

        <ProCard bordered split="vertical" className="app-topology-page">
          <ProCard colSpan="70%" bodyStyle={{ padding: 0 }}>
            {!selectedClusterId ? (
              <div className="app-topology-overview__empty">
                <Empty description="请选择一个集群后查看资源视图" />
              </div>
            ) : isLoading ? (
              <div className="app-topology-overview__empty">
                <Spin />
              </div>
            ) : graphNodes.length === 0 ? (
              <div className="app-topology-overview__empty">
                <Empty description="当前集群暂无可展示的节点或 Pod 关系" />
              </div>
            ) : (
              <ReactFlow nodes={graphNodes} edges={graphEdges} fitView fitViewOptions={{ padding: 0.16 }}>
                <Background gap={20} size={1} />
                <MiniMap pannable zoomable />
                <Controls position="bottom-right" />
              </ReactFlow>
            )}
          </ProCard>
          <ProCard colSpan="30%" className="app-topology-overview__side">
            <Space direction="vertical" size={14} style={{ width: '100%' }}>
              <div>
                <Text strong style={{ fontSize: 16 }}>
                  {selectedCluster?.name || '未选择集群'}
                </Text>
                <div className="app-topology-overview__hint">
                  {selectedCluster?.k8sVersion || '未识别版本'}
                </div>
              </div>

              <div>
                <Text strong>视图说明</Text>
                <div className="app-topology-overview__legend">
                  <Tag color="blue" icon={<ClusterOutlined />}>节点承载层</Tag>
                  <Tag color="green" icon={<HddOutlined />}>Pod 运行层</Tag>
                  <Tag color="gold" icon={<NodeIndexOutlined />}>Service 清单</Tag>
                </div>
              </div>

              <div>
                <Text strong>Service 清单</Text>
                <div className="app-topology-overview__service-list">
                  {serviceResources.length > 0 ? serviceResources.map((service) => (
                    <div key={service.id} className="app-topology-overview__service-item">
                      <div>
                        <Text strong>{service.data?.label || service.id}</Text>
                        <div className="app-topology-overview__hint">{service.data?.namespace || 'default'}</div>
                      </div>
                      <Tag color="gold">{service.data?.clusterIP || 'ClusterIP'}</Tag>
                    </div>
                  )) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无 Service" />}
                </div>
              </div>

              <div>
                <Text strong>推荐动作</Text>
                <div className="app-topology-overview__legend">
                  <Button block onClick={() => history.push('/clusters')}>返回集群列表</Button>
                  {selectedClusterId ? (
                    <Button block type="primary" onClick={() => history.push(`/k8s/${selectedClusterId}/pods`)}>
                      查看集群 Pod 列表
                    </Button>
                  ) : null}
                </div>
              </div>
            </Space>
          </ProCard>
        </ProCard>
      </div>
    </AppPage>
  )
}

export default TopologyPage
