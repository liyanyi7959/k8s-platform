import React, { useMemo } from 'react'
import { history, useModel } from '@umijs/max'
import { useQuery } from '@tanstack/react-query'
import {
  AlertOutlined,
  ApartmentOutlined,
  AppstoreOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
  CloudDownloadOutlined,
  ClusterOutlined,
  NodeIndexOutlined,
  RadarChartOutlined,
  RobotOutlined,
  RocketOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import {
  Button,
  Card,
  Col,
  Empty,
  Progress,
  Row,
  Space,
  Spin,
  Statistic,
  Tag,
  Tooltip,
  Typography,
} from 'antd'
import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'
import { getClusterOverview } from '@/services/k8s'
import {
  enterClusterWorkspace,
  getClusterStatusColor,
  getClusterStatusText,
  isClusterHealthy,
  needsClusterAttention,
} from '@/utils'
import type { Cluster } from '@/types'

const { Paragraph, Text, Title } = Typography

const QUICK_ENTRIES = [
  {
    key: 'clusters',
    title: '集群列表',
    desc: '查看接入与状态',
    path: '/clusters',
    icon: <ClusterOutlined />,
    tone: 'blue',
  },
  {
    key: 'topology',
    title: '资源拓扑',
    desc: '查看资源关系',
    path: '/topology',
    icon: <ApartmentOutlined />,
    tone: 'cyan',
  },
  {
    key: 'projects',
    title: '项目空间',
    desc: '管理业务边界',
    path: '/projects',
    icon: <RocketOutlined />,
    tone: 'green',
  },
  {
    key: 'app-store',
    title: '应用商店',
    desc: '快速交付应用',
    path: '/app-store',
    icon: <AppstoreOutlined />,
    tone: 'orange',
  },
  {
    key: 'helm',
    title: 'Helm 管理',
    desc: '进入发布管理',
    path: '/clusters',
    icon: <CloudDownloadOutlined />,
    tone: 'slate',
  },
  {
    key: 'ai',
    title: 'AI 运维',
    desc: '模型与诊断入口',
    path: '/ai/settings',
    icon: <RobotOutlined />,
    tone: 'red',
  },
] as const

const formatPercent = (value: number) => Math.max(0, Math.min(100, Number(value || 0)))

const ClusterFleetCard: React.FC<{
  cluster: Cluster
  onEnterCluster: (cluster: Cluster) => void
}> = ({ cluster, onEnterCluster }) => {
  const { data: overview, isLoading } = useQuery({
    queryKey: ['dashboard-cluster-overview', cluster.id],
    queryFn: ({ signal }) => getClusterOverview(cluster.id, signal),
    enabled: !!cluster.id,
    staleTime: 60_000,
  })

  const healthy = isClusterHealthy(cluster.status)
  const nodesReady = overview?.stats.nodes.ready ?? cluster.nodeCount ?? 0
  const nodesTotal = overview?.stats.nodes.total ?? cluster.nodeCount ?? 0
  const podTotal = overview?.stats.pods.total ?? 0
  const cpuUsage = formatPercent(overview?.stats.cpu.used_percent ?? 0)
  const memoryUsage = formatPercent(overview?.stats.memory.used_percent ?? 0)

  return (
    <Card
      hoverable
      className={['app-aiops-cluster-card', healthy ? '' : 'is-blocked'].filter(Boolean).join(' ')}
      onClick={() => onEnterCluster(cluster)}
    >
      <div className="app-aiops-cluster-card__top">
        <div className="app-aiops-cluster-card__identity">
          <span className="app-aiops-cluster-card__icon">
            <ClusterOutlined />
          </span>
          <div>
            <div className="app-aiops-cluster-card__name">{cluster.name}</div>
            <div className="app-aiops-cluster-card__meta">
              <span>{cluster.type || 'Kubernetes'}</span>
              <span>{cluster.k8sVersion || '版本待确认'}</span>
            </div>
          </div>
        </div>
        <Tag color={getClusterStatusColor(cluster.status)}>
          {getClusterStatusText(cluster.status)}
        </Tag>
      </div>

      {isLoading ? (
        <div className="app-aiops-cluster-card__loading">
          <Spin size="small" />
        </div>
      ) : (
        <>
          <div className="app-aiops-cluster-card__stats">
            <div className="app-aiops-cluster-card__stat">
              <span className="app-aiops-cluster-card__label">节点 Ready</span>
              <strong>
                {nodesReady} / {nodesTotal}
              </strong>
            </div>
            <div className="app-aiops-cluster-card__stat">
              <span className="app-aiops-cluster-card__label">Pod 总量</span>
              <strong>{podTotal}</strong>
            </div>
          </div>

          <div className="app-aiops-cluster-card__progress">
            <div>
              <div className="app-aiops-cluster-card__progress-head">
                <span>CPU</span>
                <strong>{cpuUsage.toFixed(1)}%</strong>
              </div>
              <Progress percent={cpuUsage} showInfo={false} size="small" />
            </div>
            <div>
              <div className="app-aiops-cluster-card__progress-head">
                <span>内存</span>
                <strong>{memoryUsage.toFixed(1)}%</strong>
              </div>
              <Progress percent={memoryUsage} showInfo={false} size="small" />
            </div>
          </div>
        </>
      )}

      <div className="app-aiops-cluster-card__footer">
        <Text type={healthy ? 'secondary' : 'warning'}>
          {healthy ? '可进入管理台' : '需先恢复健康状态'}
        </Text>
        <Button
          type={healthy ? 'primary' : 'default'}
          size="small"
          onClick={(event) => {
            event.stopPropagation()
            onEnterCluster(cluster)
          }}
        >
          {healthy ? '进入管理台' : '查看状态'}
        </Button>
      </div>
    </Card>
  )
}

const DashboardPage: React.FC = () => {
  const { setCurrentCluster } = useModel('cluster')
  const { data, isLoading } = useQuery({
    queryKey: ['clusters-dashboard-overview'],
    queryFn: () => listClusters({ pageSize: 100 }),
  })

  const clusters = data?.items || []

  const summary = useMemo(() => {
    const healthy = clusters.filter((cluster) => isClusterHealthy(cluster.status))
    const attention = clusters.filter((cluster) => needsClusterAttention(cluster.status))
    const totalNodes = clusters.reduce((sum, cluster) => sum + (cluster.nodeCount || 0), 0)
    const healthPercent = clusters.length
      ? Math.round((healthy.length / clusters.length) * 100)
      : 0

    return {
      healthy,
      totalNodes,
      healthPercent,
      totalClusters: clusters.length,
      riskClusters: attention.filter((cluster) => !isClusterHealthy(cluster.status)),
    }
  }, [clusters])

  const heroMetrics = [
    {
      key: 'clusters',
      label: '纳管集群',
      value: summary.totalClusters,
      icon: <ClusterOutlined />,
    },
    {
      key: 'healthy',
      label: '可进入管理',
      value: summary.healthy.length,
      icon: <CheckCircleOutlined />,
    },
    {
      key: 'risk',
      label: '待处理风险',
      value: summary.riskClusters.length,
      icon: <AlertOutlined />,
    },
    {
      key: 'nodes',
      label: '纳管节点',
      value: summary.totalNodes,
      icon: <NodeIndexOutlined />,
    },
  ]

  const spotlightClusters = summary.riskClusters.slice(0, 4)
  const displayClusters = [...summary.riskClusters, ...summary.healthy]
    .filter((cluster, index, list) => list.findIndex((item) => item.id === cluster.id) === index)
    .slice(0, 8)

  const handleEnterCluster = (cluster: Cluster, targetPath?: string) => {
    enterClusterWorkspace(cluster, {
      setCurrentCluster,
      targetPath,
    })
  }

  return (
    <AppPage breadcrumbRender={false}>
      <div className="app-page-shell app-aiops-overview">
        <section className="app-aiops-hero">
          <div className="app-aiops-hero__content">
            <span className="app-aiops-hero__eyebrow">
              <RadarChartOutlined />
              AIOPS GLOBAL OVERVIEW
            </span>
            <Title level={2} className="app-aiops-hero__title">
              AIOPS 全局态势
            </Title>
            <Paragraph className="app-aiops-hero__desc">
              统一查看集群状态、容量与风险。
            </Paragraph>
            <Space wrap className="app-aiops-hero__actions">
              <Button type="primary" size="large" onClick={() => history.push('/clusters')}>
                查看集群
              </Button>
              <Button size="large" onClick={() => history.push('/ai/settings')}>
                模型配置
              </Button>
            </Space>
          </div>

          <div className="app-aiops-hero__panel">
            <div className="app-aiops-hero__panel-header">
              <div>
                <Text className="app-aiops-hero__panel-label">平台运行评分</Text>
                <div className="app-aiops-hero__panel-score">
                  {summary.healthPercent}
                  <span>/100</span>
                </div>
              </div>
              <Tag color={summary.riskClusters.length > 0 ? 'warning' : 'success'}>
                {summary.riskClusters.length > 0 ? '存在待处理项' : '运行稳定'}
              </Tag>
            </div>
            <Progress percent={summary.healthPercent} showInfo={false} strokeColor="#2563eb" />
            <div className="app-aiops-hero__metric-grid">
              {heroMetrics.map((item) => (
                <div key={item.key} className="app-aiops-hero__metric">
                  <span className="app-aiops-hero__metric-icon">{item.icon}</span>
                  <div>
                    <strong>{item.value}</strong>
                    <span>{item.label}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <Row gutter={[16, 16]}>
          {heroMetrics.map((item) => (
            <Col key={item.key} xs={12} lg={6}>
              <Card className="app-aiops-stat-card">
                <Statistic title={item.label} value={item.value} prefix={item.icon} />
              </Card>
            </Col>
          ))}
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={16}>
            <Card
              className="app-aiops-panel"
              title="集群舰队"
              extra={
                <Button type="link" onClick={() => history.push('/clusters')}>
                  查看全部 <ArrowRightOutlined />
                </Button>
              }
            >
              <div className="app-aiops-panel__meta">
                <span>健康集群 {summary.healthy.length}</span>
                <span>风险集群 {summary.riskClusters.length}</span>
              </div>

              {isLoading ? (
                <div className="app-aiops-panel__loading">
                  <Spin size="large" />
                </div>
              ) : displayClusters.length === 0 ? (
                <Empty
                  description="当前还没有接入任何集群"
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                >
                  <Button type="primary" onClick={() => history.push('/clusters/import')}>
                    导入集群
                  </Button>
                </Empty>
              ) : (
                <div className="app-aiops-cluster-grid">
                  {displayClusters.map((cluster) => (
                    <ClusterFleetCard
                      key={cluster.id}
                      cluster={cluster}
                      onEnterCluster={handleEnterCluster}
                    />
                  ))}
                </div>
              )}
            </Card>
          </Col>

          <Col xs={24} xl={8}>
            <Card className="app-aiops-panel" title="风险处置">
              <div className="app-aiops-panel__meta">
                <span>{summary.riskClusters.length} 个集群待处理</span>
              </div>

              {spotlightClusters.length ? (
                <div className="app-aiops-risk-list">
                  {spotlightClusters.map((cluster) => (
                    <div key={cluster.id} className="app-aiops-risk-item">
                      <div className="app-aiops-risk-item__head">
                        <div>
                          <div className="app-aiops-risk-item__name">{cluster.name}</div>
                          <div className="app-aiops-risk-item__sub">
                            {cluster.type || 'Kubernetes'} · {cluster.k8sVersion || '版本待确认'}
                          </div>
                        </div>
                        <Tag color={getClusterStatusColor(cluster.status)}>
                          {getClusterStatusText(cluster.status)}
                        </Tag>
                      </div>
                      <div className="app-aiops-risk-item__foot">
                        <Text type="secondary">建议先检查连接和节点状态。</Text>
                        <Space size={8}>
                          <Button size="small" onClick={() => history.push(`/clusters/${cluster.id}`)}>
                            查看信息
                          </Button>
                          <Tooltip title="当前状态未恢复前不可进入集群管理">
                            <Button size="small" onClick={() => handleEnterCluster(cluster)}>
                              尝试进入
                            </Button>
                          </Tooltip>
                        </Space>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="app-aiops-empty-state">
                  <SafetyCertificateOutlined />
                  <div>
                    <strong>当前无风险集群</strong>
                    <span>可直接进入健康集群管理台。</span>
                  </div>
                </div>
              )}
            </Card>
          </Col>
        </Row>

        <Card className="app-aiops-panel" title="平台入口">
          <div className="app-aiops-entry-grid">
            {QUICK_ENTRIES.map((entry) => (
              <button
                key={entry.key}
                type="button"
                className={`app-aiops-entry-card tone-${entry.tone}`}
                onClick={() => history.push(entry.path)}
              >
                <span className="app-aiops-entry-card__icon">{entry.icon}</span>
                <span className="app-aiops-entry-card__title">{entry.title}</span>
                <span className="app-aiops-entry-card__desc">{entry.desc}</span>
              </button>
            ))}
          </div>
        </Card>
      </div>
    </AppPage>
  )
}

export default DashboardPage
