/**
 * 全局资源视图
 * 展示所有集群的资源概览（节点/Pod/工作负载数量 + CPU/内存使用率）
 */
import React from 'react'
import { history } from '@umijs/max'
import { Card, Row, Col, Spin, Tag, Progress, Typography, Empty, Button, Space, Statistic } from 'antd'
import {
  ClusterOutlined,
  ReloadOutlined,
  CloudServerOutlined,
  HddOutlined,
  ArrowRightOutlined,
  CloudServerOutlined as NodeIcon,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'
import { getClusterOverview } from '@/services/k8s'
import { formatDate } from '@/utils'

const { Text, Title } = Typography

/** 集群概览卡片 */
function ClusterCard({ cluster }: { cluster: any }) {
  const { data: overview, isLoading } = useQuery({
    queryKey: ['cluster-overview-topo', cluster.id],
    queryFn: ({ signal }) => getClusterOverview(cluster.id, signal),
    enabled: !!cluster.id,
    staleTime: 60_000,
  })

  // 防御性取值（overview 字段名可能因后端版本不同）
  const nodeCount = overview?.nodeCount ?? overview?.nodes?.total ?? 0
  const readyNodes = overview?.readyNodes ?? overview?.nodes?.ready ?? 0
  const podCount = overview?.podCount ?? overview?.pods?.total ?? 0
  const runningPods = overview?.runningPods ?? overview?.pods?.running ?? 0
  const deployCount = overview?.deploymentCount ?? overview?.deployments?.total ?? 0
  const svcCount = overview?.serviceCount ?? overview?.services?.total ?? 0
  const cpuUsage = overview?.cpuUsagePercent ?? overview?.cpu?.percent ?? 0
  const memUsage = overview?.memoryUsagePercent ?? overview?.memory?.percent ?? 0

  const statusColor = cluster.status === 'healthy' ? 'success' : cluster.status === 'unhealthy' ? 'error' : 'default'

  return (
    <Card
      hoverable
      size="small"
      style={{ borderRadius: 12 }}
      onClick={() => history.push(`/clusters/${cluster.id}`)}
      title={
        <Space>
          <ClusterOutlined style={{ color: '#2563eb' }} />
          <Text strong>{cluster.name}</Text>
          <Tag color={statusColor}>{cluster.status || '未知'}</Tag>
        </Space>
      }
      extra={
        <Button
          type="text"
          size="small"
          icon={<ArrowRightOutlined />}
          onClick={(e) => {
            e.stopPropagation()
            history.push(`/k8s/${cluster.id}/dashboard`)
          }}
        />
      }
    >
      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 24 }}>
          <Spin />
        </div>
      ) : (
        <>
          {/* 资源统计网格 */}
          <Row gutter={[8, 8]}>
            <Col span={6}>
              <Statistic title="节点" value={nodeCount} prefix={<ClusterOutlined />} valueStyle={{ fontSize: 18 }} />
              <Text type="secondary" style={{ fontSize: 11 }}>就绪 {readyNodes}</Text>
            </Col>
            <Col span={6}>
              <Statistic title="Pod" value={podCount} prefix={<HddOutlined />} valueStyle={{ fontSize: 18 }} />
              <Text type="secondary" style={{ fontSize: 11 }}>运行 {runningPods}</Text>
            </Col>
            <Col span={6}>
              <Statistic title="工作负载" value={deployCount} prefix={<CloudServerOutlined />} valueStyle={{ fontSize: 18 }} />
            </Col>
            <Col span={6}>
              <Statistic title="Service" value={svcCount} prefix={<NodeIcon />} valueStyle={{ fontSize: 18 }} />
            </Col>
          </Row>

          {/* CPU/内存使用率 */}
          <div style={{ marginTop: 16 }}>
            <div style={{ marginBottom: 8 }}>
              <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                <Text type="secondary" style={{ fontSize: 12 }}>CPU 使用率</Text>
                <Text style={{ fontSize: 12, fontWeight: 600 }}>{cpuUsage.toFixed(1)}%</Text>
              </Space>
              <Progress
                percent={cpuUsage}
                size="small"
                strokeColor={cpuUsage > 80 ? '#ff4d4f' : cpuUsage > 60 ? '#faad14' : '#52c41a'}
              />
            </div>
            <div>
              <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                <Text type="secondary" style={{ fontSize: 12 }}>内存使用率</Text>
                <Text style={{ fontSize: 12, fontWeight: 600 }}>{memUsage.toFixed(1)}%</Text>
              </Space>
              <Progress
                percent={memUsage}
                size="small"
                strokeColor={memUsage > 80 ? '#ff4d4f' : memUsage > 60 ? '#faad14' : '#52c41a'}
              />
            </div>
          </div>

          {/* 快捷入口 */}
          <Space style={{ marginTop: 12 }}>
            <Button
              size="small"
              onClick={(e) => {
                e.stopPropagation()
                history.push(`/k8s/${cluster.id}/topology`)
              }}
            >
              资源关系图
            </Button>
            <Button
              size="small"
              onClick={(e) => {
                e.stopPropagation()
                history.push(`/k8s/${cluster.id}/helm-releases`)
              }}
            >
              Helm
            </Button>
            <Button
              size="small"
              onClick={(e) => {
                e.stopPropagation()
                history.push(`/k8s/${cluster.id}/resource-metrics`)
              }}
            >
              监控
            </Button>
          </Space>
        </>
      )}
    </Card>
  )
}

/** 全局资源视图页面 */
const ResourceOverviewPage: React.FC = () => {
  const { data, isLoading } = useQuery({
    queryKey: ['clusters-for-resource-overview'],
    queryFn: () => listClusters({ pageSize: 100 }),
  })

  const clusters = data?.items || []

  // 全局统计
  const totalNodes = clusters.length
  const healthyClusters = clusters.filter((c: any) => c.status === 'healthy').length

  return (
    <AppPage breadcrumbRender={false}>
      <div className="app-data-console">
        {/* 全局统计 */}
        <section className="app-data-console__statgrid">
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">集群总数</span>
            <strong className="app-data-console__stat-value">{totalNodes}</strong>
            <span className="app-data-console__stat-hint">健康 {healthyClusters} 个</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">集群状态</span>
            <strong className="app-data-console__stat-value">
              {healthyClusters}/{totalNodes}
            </strong>
            <span className="app-data-console__stat-hint">点击卡片进入集群详情</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">快捷操作</span>
            <strong className="app-data-console__stat-value" style={{ fontSize: 14 }}>
              <Button
                type="link"
                size="small"
                icon={<ReloadOutlined />}
                onClick={() => window.location.reload()}
              >
                刷新
              </Button>
            </strong>
            <span className="app-data-console__stat-hint">数据每 60 秒自动刷新</span>
          </div>
        </section>

        {/* 集群卡片网格 */}
        <section>
          {isLoading ? (
            <div style={{ textAlign: 'center', padding: 60 }}>
              <Spin size="large" />
            </div>
          ) : clusters.length === 0 ? (
            <Empty description="暂无集群，请先导入集群" image={Empty.PRESENTED_IMAGE_SIMPLE}>
              <Button type="primary" onClick={() => history.push('/clusters/import')}>
                导入集群
              </Button>
            </Empty>
          ) : (
            <Row gutter={[16, 16]}>
              {clusters.map((cluster: any) => (
                <Col key={cluster.id} xs={24} sm={12} lg={8} xl={6}>
                  <ClusterCard cluster={cluster} />
                </Col>
              ))}
            </Row>
          )}
        </section>
      </div>
    </AppPage>
  )
}

export default ResourceOverviewPage
