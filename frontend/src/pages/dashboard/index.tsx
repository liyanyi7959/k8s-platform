/**
 * AIOps 智能运维平台 - 全局概览
 * 统一管理多集群 Kubernetes 资源、应用部署与智能运维
 */
import React from 'react'
import { history } from '@umijs/max'
import { Card, Row, Col, Spin, Tag, Progress, Typography, Empty, Button, Statistic } from 'antd'
import {
  ClusterOutlined,
  HddOutlined,
  CloudServerOutlined,
  AppstoreOutlined,
  ApartmentOutlined,
  RobotOutlined,
  ArrowRightOutlined,
  CloudDownloadOutlined,
  SafetyOutlined,
  RocketOutlined,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'
import { getClusterOverview } from '@/services/k8s'

const { Text, Title } = Typography

/** 功能入口配置 */
const ENTRIES = [
  { key: 'clusters', title: '集群管理', desc: '导入和管理 K8s 集群', icon: <ClusterOutlined />, color: '#2563eb', path: '/clusters' },
  { key: 'topology', title: '资源视图', desc: '全局资源概览', icon: <ApartmentOutlined />, color: '#0891b2', path: '/topology' },
  { key: 'appstore', title: '应用商店', desc: 'YAML 模板和 Helm Chart', icon: <AppstoreOutlined />, color: '#059669', path: '/app-store' },
  { key: 'projects', title: '项目管理', desc: '命名空间分组管理', icon: <RocketOutlined />, color: '#7c3aed', path: '/projects' },
  { key: 'helm', title: 'Helm 管理', desc: 'Release 安装和管理', icon: <CloudDownloadOutlined />, color: '#c2410c', path: '/clusters' },
  { key: 'ai', title: 'AI 运维', desc: '智能运维助手', icon: <RobotOutlined />, color: '#dc2626', path: '/ai/chat' },
] as const

/** 集群概览卡片 */
function ClusterOverviewCard({ cluster }: { cluster: any }) {
  const { data: overview, isLoading } = useQuery({
    queryKey: ['cluster-overview-dashboard', cluster.id],
    queryFn: ({ signal }) => getClusterOverview(cluster.id, signal),
    enabled: !!cluster.id,
    staleTime: 60_000,
  })

  const statusColor = cluster.status === 'healthy' ? 'success' : cluster.status === 'unhealthy' ? 'error' : 'default'
  const statusText = cluster.status === 'healthy' ? '健康' : cluster.status === 'unhealthy' ? '异常' : '未知'

  // 防御性取值
  const nodeCount = overview?.nodeCount ?? overview?.nodes?.total ?? cluster.nodeCount ?? 0
  const podCount = overview?.podCount ?? overview?.pods?.total ?? 0
  const cpuUsage = overview?.cpuUsagePercent ?? overview?.cpu?.percent ?? 0
  const memUsage = overview?.memoryUsagePercent ?? overview?.memory?.percent ?? 0

  return (
    <Card
      hoverable
      size="small"
      style={{ borderRadius: 12, cursor: 'pointer', transition: 'all 0.3s' }}
      onClick={() => history.push(`/clusters/${cluster.id}`)}
    >
      {/* 标题行 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ClusterOutlined style={{ color: '#2563eb', fontSize: 18 }} />
          <Text strong style={{ fontSize: 15 }}>{cluster.name}</Text>
        </div>
        <Tag color={statusColor}>{statusText}</Tag>
      </div>

      {/* 指标 */}
      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 16 }}><Spin size="small" /></div>
      ) : (
        <>
          <Row gutter={16}>
            <Col span={12}>
              <Statistic title="节点" value={nodeCount} valueStyle={{ fontSize: 18 }} />
            </Col>
            <Col span={12}>
              <Statistic title="Pod" value={podCount} valueStyle={{ fontSize: 18 }} />
            </Col>
          </Row>

          {/* CPU/内存使用率 */}
          <div style={{ marginTop: 12 }}>
            <div style={{ marginBottom: 8 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                <Text type="secondary" style={{ fontSize: 11 }}>CPU</Text>
                <Text style={{ fontSize: 11, fontWeight: 600 }}>{cpuUsage.toFixed(1)}%</Text>
              </div>
              <Progress
                percent={cpuUsage}
                size="small"
                strokeColor={cpuUsage > 80 ? '#ff4d4f' : cpuUsage > 60 ? '#faad14' : '#52c41a'}
                showInfo={false}
              />
            </div>
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                <Text type="secondary" style={{ fontSize: 11 }}>内存</Text>
                <Text style={{ fontSize: 11, fontWeight: 600 }}>{memUsage.toFixed(1)}%</Text>
              </div>
              <Progress
                percent={memUsage}
                size="small"
                strokeColor={memUsage > 80 ? '#ff4d4f' : memUsage > 60 ? '#faad14' : '#52c41a'}
                showInfo={false}
              />
            </div>
          </div>

          {/* 进入集群 */}
          <Button
            type="link"
            size="small"
            style={{ padding: 0, marginTop: 8 }}
            onClick={(e) => {
              e.stopPropagation()
              history.push(`/k8s/${cluster.id}/dashboard`)
            }}
          >
            进入集群 <ArrowRightOutlined />
          </Button>
        </>
      )}
    </Card>
  )
}

/** 全局概览页面 */
const DashboardPage: React.FC = () => {
  const { data, isLoading } = useQuery({
    queryKey: ['clusters-dashboard'],
    queryFn: () => listClusters({ pageSize: 100 }),
  })

  const clusters = data?.items || []
  const healthyCount = clusters.filter((c: any) => c.status === 'healthy').length
  const unhealthyCount = clusters.length - healthyCount
  const totalNodes = clusters.reduce((sum: number, c: any) => sum + (c.nodeCount || 0), 0)

  return (
    <AppPage breadcrumbRender={false}>
      <div style={{ maxWidth: 1400, margin: '0 auto' }}>
        {/* ========== 欢迎区 ========== */}
        <div style={{ marginBottom: 28 }}>
          <Title level={3} style={{ marginBottom: 4 }}>AIOps 智能运维平台</Title>
          <Text type="secondary">统一管理多集群 Kubernetes 资源、应用部署与智能运维</Text>
        </div>

        {/* ========== 全局指标 ========== */}
        <Row gutter={[16, 16]} style={{ marginBottom: 28 }}>
          <Col xs={12} sm={6}>
            <Card bordered={false} style={{ borderRadius: 12, background: 'linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%)' }}>
              <Statistic
                title="集群总数"
                value={clusters.length}
                prefix={<ClusterOutlined />}
                valueStyle={{ color: '#2563eb' }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card bordered={false} style={{ borderRadius: 12, background: 'linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%)' }}>
              <Statistic
                title="健康集群"
                value={healthyCount}
                prefix={<SafetyOutlined />}
                valueStyle={{ color: '#059669' }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card bordered={false} style={{ borderRadius: 12, background: 'linear-gradient(135deg, #f0fdfa 0%, #ccfbf1 100%)' }}>
              <Statistic
                title="节点总数"
                value={totalNodes}
                prefix={<HddOutlined />}
                valueStyle={{ color: '#0d9488' }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card bordered={false} style={{ borderRadius: 12, background: 'linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%)' }}>
              <Statistic
                title="异常集群"
                value={unhealthyCount}
                prefix={<CloudServerOutlined />}
                valueStyle={{ color: '#dc2626' }}
              />
            </Card>
          </Col>
        </Row>

        {/* ========== 集群概览 ========== */}
        <div style={{ marginBottom: 28 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <Title level={5} style={{ margin: 0 }}>集群概览</Title>
            {clusters.length > 0 && (
              <Button type="link" onClick={() => history.push('/clusters')}>
                查看全部 <ArrowRightOutlined />
              </Button>
            )}
          </div>
          {isLoading ? (
            <div style={{ textAlign: 'center', padding: 60 }}>
              <Spin size="large" />
            </div>
          ) : clusters.length === 0 ? (
            <Card bordered={false} style={{ borderRadius: 12, textAlign: 'center', padding: 40 }}>
              <Empty description="暂无纳管集群，请先导入集群">
                <Button type="primary" onClick={() => history.push('/clusters/import')}>
                  导入集群
                </Button>
              </Empty>
            </Card>
          ) : (
            <Row gutter={[16, 16]}>
              {clusters.slice(0, 8).map((cluster: any) => (
                <Col key={cluster.id} xs={24} sm={12} lg={6}>
                  <ClusterOverviewCard cluster={cluster} />
                </Col>
              ))}
            </Row>
          )}
        </div>

        {/* ========== 功能入口 ========== */}
        <div>
          <Title level={5} style={{ marginBottom: 12 }}>功能入口</Title>
          <Row gutter={[16, 16]}>
            {ENTRIES.map((entry) => (
              <Col key={entry.key} xs={12} sm={8} lg={4}>
                <Card
                  hoverable
                  bordered={false}
                  style={{
                    borderRadius: 12,
                    textAlign: 'center',
                    cursor: 'pointer',
                    transition: 'all 0.3s',
                    padding: '8px 0',
                  }}
                  onClick={() => history.push(entry.path)}
                >
                  <div style={{ fontSize: 32, color: entry.color, marginBottom: 8 }}>
                    {entry.icon}
                  </div>
                  <Text strong>{entry.title}</Text>
                  <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>{entry.desc}</Text>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </div>
      </div>
    </AppPage>
  )
}

export default DashboardPage
