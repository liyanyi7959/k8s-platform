import React, { useMemo } from 'react'
import { history, useModel } from '@umijs/max'
import { useQueries, useQuery } from '@tanstack/react-query'
import {
  AlertOutlined,
  AppstoreOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
  CloudDownloadOutlined,
  ClusterOutlined,
  DashboardOutlined,
  ExclamationCircleOutlined,
  FileSearchOutlined,
  FundOutlined,
  NodeIndexOutlined,
  RadarChartOutlined,
  RobotOutlined,
  RocketOutlined,
  SafetyCertificateOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'

import {
  Badge,
  Button,
  Card,
  Col,
  Empty,
  Progress,
  Row,
  Space,
  Spin,
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
    tone: 'purple',
  },
] as const

const formatPercent = (value: number) => Math.max(0, Math.min(100, Number(value || 0)))

/** 从集群状态派生风险等级 */
const getRiskSeverity = (status?: string) => {
  const normalized = String(status || 'unknown').trim().toLowerCase()
  if (['error', 'failed', 'disconnected', 'unhealthy', 'offline'].includes(normalized)) {
    return 'critical'
  }
  if (['warning', 'degraded'].includes(normalized)) {
    return 'warning'
  }
  return 'info'
}

const severityColorMap: Record<string, string> = {
  critical: '#ef4444',
  warning: '#f59e0b',
  info: '#3b82f6',
}

const severityLabelMap: Record<string, string> = {
  critical: '严重',
  warning: '警告',
  info: '提示',
}

const getSeverityLabel = (key: string): string => severityLabelMap[key] ?? key
const getSeverityColor = (key: string): string => severityColorMap[key] ?? '#64748b'

/** 平台运行评分仪表盘（SVG 环形图） */
const HealthScoreGauge: React.FC<{ score: number; status: string }> = ({ score, status }) => {
  const color = score >= 90 ? '#10b981' : score >= 70 ? '#f59e0b' : '#ef4444'
  const radius = 52
  const stroke = 10
  const circumference = 2 * Math.PI * radius
  const clamped = Math.max(0, Math.min(100, score))
  const dashoffset = circumference * (1 - clamped / 100)

  return (
    <div className="app-aiops-gauge">
      <svg width="140" height="132" viewBox="0 0 140 132">
        <circle
          cx="70"
          cy="70"
          r={radius}
          fill="none"
          stroke="#e2e8f0"
          strokeWidth={stroke}
        />
        <circle
          cx="70"
          cy="70"
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={dashoffset}
          transform="rotate(-90 70 70)"
        />
        <text x="70" y="68" textAnchor="middle" fill="var(--app-text)" fontSize="28" fontWeight="800">
          {score}
        </text>
        <text x="70" y="88" textAnchor="middle" fill="var(--app-text-secondary)" fontSize="12">
          /100
        </text>
      </svg>
      <div className="app-aiops-gauge__label">
        <Badge color={color} text={status} />
      </div>
    </div>
  )
}

/** 集群状态脉冲灯 */
const StatusPulse: React.FC<{ healthy: boolean }> = ({ healthy }) => {
  return (
    <span className={['app-aiops-pulse', healthy ? 'is-healthy' : 'is-at-risk'].join(' ')} />
  )
}

const ClusterFleetCard: React.FC<{
  cluster: Cluster
  overview?: Awaited<ReturnType<typeof getClusterOverview>>
  overviewLoading?: boolean
  onEnterCluster: (cluster: Cluster) => void
}> = ({ cluster, overview, overviewLoading, onEnterCluster }) => {
  const healthy = isClusterHealthy(cluster.status)
  const nodesReady = overview?.stats.nodes.ready ?? cluster.nodeCount ?? 0
  const nodesTotal = overview?.stats.nodes.total ?? cluster.nodeCount ?? 0
  const podTotal = overview?.stats.pods.total ?? 0
  const cpuUsage = formatPercent(overview?.stats.cpu.used_percent ?? 0)
  const memoryUsage = formatPercent(overview?.stats.memory.used_percent ?? 0)
  const isLoading = overviewLoading

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
            <div className="app-aiops-cluster-card__name">
              {cluster.name}
              <StatusPulse healthy={healthy} />
            </div>
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
                <Space size={6}>
                  <DashboardOutlined />
                  <span>CPU</span>
                </Space>
                <strong className={cpuUsage >= 80 ? 'is-high' : ''}>{cpuUsage.toFixed(1)}%</strong>
              </div>
              <Progress
                percent={cpuUsage}
                showInfo={false}
                size="small"
                status={cpuUsage >= 90 ? 'exception' : cpuUsage >= 70 ? 'active' : undefined}
                strokeColor={cpuUsage >= 80 ? '#ef4444' : undefined}
              />
            </div>
            <div>
              <div className="app-aiops-cluster-card__progress-head">
                <Space size={6}>
                  <FundOutlined />
                  <span>内存</span>
                </Space>
                <strong className={memoryUsage >= 80 ? 'is-high' : ''}>{memoryUsage.toFixed(1)}%</strong>
              </div>
              <Progress
                percent={memoryUsage}
                showInfo={false}
                size="small"
                status={memoryUsage >= 90 ? 'exception' : memoryUsage >= 70 ? 'active' : undefined}
                strokeColor={memoryUsage >= 80 ? '#ef4444' : undefined}
              />
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

  const severityWeight: Record<string, number> = { critical: 3, warning: 2, info: 1 }
  const spotlightClusters = summary.riskClusters
    .slice()
    .sort((a, b) => severityWeight[getRiskSeverity(b.status)] - severityWeight[getRiskSeverity(a.status)])
    .slice(0, 3)
  const displayClusters = [...summary.riskClusters, ...summary.healthy]
    .filter((cluster, index, list) => list.findIndex((item) => item.id === cluster.id) === index)
    .slice(0, 6)

  // 为控制性能，只给仪表盘需要展示的集群（舰队卡片 + Top 风险）拉取概览数据
  const overviewNeededClusters = displayClusters
    .concat(spotlightClusters)
    .filter((cluster, index, list) => list.findIndex((item) => item.id === cluster.id) === index)

  const clusterOverviewQueries = useQueries({
    queries: overviewNeededClusters.map((cluster) => ({
      queryKey: ['dashboard-cluster-overview', cluster.id],
      queryFn: ({ signal }: { signal?: AbortSignal }) => getClusterOverview(cluster.id, signal),
      enabled: !!cluster.id,
      staleTime: 60_000,
    })),
    combine: (results) => ({
      data: new Map(
        results
          .map((result, index) => [overviewNeededClusters[index]?.id, result.data] as const)
          .filter(([, data]) => !!data),
      ),
      isLoading: results.some((result) => result.isLoading),
    }),
  })

  const getClusterOverviewData = (clusterId: number) =>
    clusterOverviewQueries.data.get(clusterId) as
      | Awaited<ReturnType<typeof getClusterOverview>>
      | undefined

  // 聚合已加载集群的资源容量数据
  const capacitySummary = useMemo(() => {
    let totalPods = 0
    let totalCpu = 0
    let totalMemory = 0
    let loadedClusters = 0

    clusters.forEach((cluster) => {
      const overview = getClusterOverviewData(cluster.id)
      if (overview) {
        totalPods += overview.stats.pods.total || 0
        totalCpu += overview.stats.cpu.used_percent || 0
        totalMemory += overview.stats.memory.used_percent || 0
        loadedClusters++
      }
    })

    return {
      totalPods,
      avgCpu: loadedClusters ? Math.round(totalCpu / loadedClusters) : 0,
      avgMemory: loadedClusters ? Math.round(totalMemory / loadedClusters) : 0,
    }
  }, [clusters, clusterOverviewQueries.data])

  const heroMetrics = [
    {
      key: 'clusters',
      label: '纳管集群',
      value: summary.totalClusters,
      icon: <ClusterOutlined />,
      color: '#3b82f6',
    },
    {
      key: 'healthy',
      label: '可进入管理',
      value: summary.healthy.length,
      icon: <CheckCircleOutlined />,
      color: '#10b981',
    },
    {
      key: 'risk',
      label: '待处理风险',
      value: summary.riskClusters.length,
      icon: <AlertOutlined />,
      color: '#ef4444',
    },
    {
      key: 'nodes',
      label: '纳管节点',
      value: summary.totalNodes,
      icon: <NodeIndexOutlined />,
      color: '#8b5cf6',
    },
  ]

  const riskDistribution = useMemo(() => {
    const groups: Record<string, number> = { critical: 0, warning: 0, info: 0 }
    summary.riskClusters.forEach((cluster) => {
      const severity = getRiskSeverity(cluster.status)
      groups[severity] = (groups[severity] || 0) + 1
    })
    return Object.entries(groups)
      .filter(([, value]) => value > 0)
      .map(([type, value]) => ({ type: getSeverityLabel(type), value }))
  }, [summary.riskClusters])

  // 模拟最近告警 / 事件（基于已有集群状态，不新增 API）
  const recentAlerts = useMemo(() => {
    const alerts: Array<{
      id: string
      name: string
      message: string
      severity: string
      time: string
    }> = []
    summary.riskClusters.slice(0, 4).forEach((cluster, index) => {
      const severity = getRiskSeverity(cluster.status)
      alerts.push({
        id: `alert-${cluster.id}`,
        name: cluster.name,
        message:
          severity === 'critical'
            ? '集群连接异常，需立即检查'
            : severity === 'warning'
              ? '集群状态降级，建议巡检'
              : '集群状态待确认',
        severity,
        time: `${(index + 1) * 5}分钟前`,
      })
    })
    return alerts
  }, [summary.riskClusters])

  // AI 洞察（基于真实聚合数据生成具体建议）
  const aiInsights = useMemo(() => {
    const insights: Array<{ id: string; icon: React.ReactNode; text: string }> = []

    // 整体健康态势
    insights.push({
      id: 'insight-1',
      icon: <ThunderboltOutlined />,
      text:
        summary.riskClusters.length > 0
          ? `当前 ${summary.totalClusters} 个集群中健康率 ${summary.healthPercent}%，${summary.riskClusters.length} 个集群异常，建议优先处理 ${spotlightClusters[0]?.name || ''}。`
          : `当前 ${summary.totalClusters} 个集群健康率 ${summary.healthPercent}%，整体运行平稳。`,
    })

    // 容量建议
    if (capacitySummary.avgCpu >= 80 || capacitySummary.avgMemory >= 80) {
      insights.push({
        id: 'insight-2',
        icon: <FileSearchOutlined />,
        text: `重点集群资源压力较高：平均 CPU ${capacitySummary.avgCpu}%，平均内存 ${capacitySummary.avgMemory}%，建议评估集群扩容或工作负载调度。`,
      })
    } else if (summary.totalNodes > 0) {
      insights.push({
        id: 'insight-2',
        icon: <FileSearchOutlined />,
        text: `共纳管 ${summary.totalNodes} 个节点、${capacitySummary.totalPods} 个 Pod，资源利用率处于健康区间。`,
      })
    }

    // 针对前 2 个风险集群的具体建议
    spotlightClusters.slice(0, 2).forEach((cluster, index) => {
      const overview = getClusterOverviewData(cluster.id)
      const severity = getRiskSeverity(cluster.status)
      let text = ''
      if (overview) {
        if (overview.stats.nodes.ready === 0) {
          text = `【${cluster.name}】所有节点未就绪，请检查网络连通性与 kubelet 服务。`
        } else if (overview.stats.pods.failed > 0) {
          text = `【${cluster.name}】存在 ${overview.stats.pods.failed} 个失败 Pod，建议查看事件日志定位原因。`
        } else if (overview.stats.cpu.used_percent > 80) {
          text = `【${cluster.name}】CPU 使用率 ${overview.stats.cpu.used_percent.toFixed(1)}%，建议排查高负载应用。`
        } else if (overview.stats.memory.used_percent > 80) {
          text = `【${cluster.name}】内存使用率 ${overview.stats.memory.used_percent.toFixed(1)}%，建议关注容量风险。`
        } else {
          text = `【${cluster.name}】状态异常，建议执行健康检查。`
        }
      } else {
        text =
          severity === 'critical'
            ? `【${cluster.name}】连接异常，需立即检查。`
            : `【${cluster.name}】状态降级，建议巡检。`
      }
      insights.push({
        id: `insight-risk-${index}`,
        icon: <ExclamationCircleOutlined />,
        text,
      })
    })

    return insights
  }, [summary, capacitySummary, spotlightClusters, clusterOverviewQueries.data])

  const handleEnterCluster = (cluster: Cluster, targetPath?: string) => {
    enterClusterWorkspace(cluster, {
      setCurrentCluster,
      targetPath,
    })
  }

  const scoreStatus = summary.riskClusters.length > 0 ? '存在待处理项' : '运行稳定'

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
              统一纳管集群状态、容量与风险，AI 实时洞察并自动化处置异常。
            </Paragraph>
            <div className="app-aiops-hero__chips">
              <span>{summary.totalClusters} 个集群</span>
              <span>{summary.totalNodes} 个节点</span>
              <span>{summary.riskClusters.length} 个待处理</span>
            </div>
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
              <Text className="app-aiops-hero__panel-label">平台运行评分</Text>
              <HealthScoreGauge score={summary.healthPercent} status={scoreStatus} />
            </div>
          </div>
        </section>

        <Row gutter={[16, 16]}>
          {heroMetrics.map((item) => (
            <Col key={item.key} xs={12} lg={6}>
              <Card
                className="app-aiops-stat-card"
                style={{ borderTopColor: item.color }}
                hoverable
              >
                <div className="app-aiops-stat-card__inner">
                  <span
                    className="app-aiops-stat-card__icon"
                    style={{ background: `${item.color}1a`, color: item.color }}
                  >
                    {item.icon}
                  </span>
                  <div className="app-aiops-stat-card__body">
                    <span className="app-aiops-stat-card__value">{item.value}</span>
                    <span className="app-aiops-stat-card__label">{item.label}</span>
                  </div>
                </div>
              </Card>
            </Col>
          ))}
        </Row>

        <Card className="app-aiops-panel" title="重点集群资源" size="small">
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={8}>
              <div className="app-aiops-capacity-card">
                <div className="app-aiops-capacity-card__header">
                  <span className="app-aiops-capacity-card__label">Pod 总量</span>
                  <strong className="app-aiops-capacity-card__value">{capacitySummary.totalPods}</strong>
                </div>
                <Progress
                  percent={Math.min(capacitySummary.totalPods, 100)}
                  showInfo={false}
                  size="small"
                  strokeColor="#3b82f6"
                />
              </div>
            </Col>
            <Col xs={24} sm={8}>
              <div className="app-aiops-capacity-card">
                <div className="app-aiops-capacity-card__header">
                  <span className="app-aiops-capacity-card__label">平均 CPU</span>
                  <strong className="app-aiops-capacity-card__value">{capacitySummary.avgCpu}%</strong>
                </div>
                <Progress
                  percent={capacitySummary.avgCpu}
                  showInfo={false}
                  size="small"
                  status={capacitySummary.avgCpu >= 80 ? 'exception' : capacitySummary.avgCpu >= 70 ? 'active' : undefined}
                  strokeColor={capacitySummary.avgCpu >= 80 ? '#ef4444' : '#3b82f6'}
                />
              </div>
            </Col>
            <Col xs={24} sm={8}>
              <div className="app-aiops-capacity-card">
                <div className="app-aiops-capacity-card__header">
                  <span className="app-aiops-capacity-card__label">平均内存</span>
                  <strong className="app-aiops-capacity-card__value">{capacitySummary.avgMemory}%</strong>
                </div>
                <Progress
                  percent={capacitySummary.avgMemory}
                  showInfo={false}
                  size="small"
                  status={capacitySummary.avgMemory >= 80 ? 'exception' : capacitySummary.avgMemory >= 70 ? 'active' : undefined}
                  strokeColor={capacitySummary.avgMemory >= 80 ? '#ef4444' : '#10b981'}
                />
              </div>
            </Col>
          </Row>
        </Card>

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
                <span>
                  健康集群 <strong>{summary.healthy.length}</strong>
                </span>
                <span>
                  风险集群 <strong>{summary.riskClusters.length}</strong>
                </span>
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
                      overview={getClusterOverviewData(cluster.id)}
                      overviewLoading={clusterOverviewQueries.isLoading}
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
                <span>
                  <strong>{summary.riskClusters.length}</strong> 个集群待处理
                </span>
              </div>

              <div className="app-aiops-risk-summary">
                {riskDistribution.map((item) => (
                  <div key={item.type} className="app-aiops-risk-summary__item">
                    <span
                      className="app-aiops-risk-summary__dot"
                      style={{ background: getSeverityColor(item.type) }}
                    />
                    <span className="app-aiops-risk-summary__label">{item.type}</span>
                    <strong className="app-aiops-risk-summary__value">{item.value}</strong>
                  </div>
                ))}
              </div>

              {spotlightClusters.length ? (
                <div className="app-aiops-risk-list">
                  {spotlightClusters.map((cluster) => {
                    const overview = getClusterOverviewData(cluster.id)
                    const severity = getRiskSeverity(cluster.status)
                    const recommendation = overview
                      ? overview.stats.nodes.ready === 0
                        ? '所有节点未就绪，建议立即检查节点连接与 kubelet 状态'
                        : overview.stats.pods.failed > 0
                          ? `检测到 ${overview.stats.pods.failed} 个异常 Pod，建议排查失败原因`
                          : overview.stats.cpu.used_percent > 80
                            ? 'CPU 使用率过高，建议评估扩容或调度优化'
                            : overview.stats.memory.used_percent > 80
                              ? '内存使用率过高，建议关注容量风险'
                              : '集群状态异常，建议执行健康检查'
                      : severity === 'critical'
                        ? '集群连接异常，需立即检查'
                        : severity === 'warning'
                          ? '集群状态降级，建议巡检'
                          : '集群状态待确认'

                    return (
                      <div key={cluster.id} className="app-aiops-risk-item">
                        <div className="app-aiops-risk-item__head">
                          <div>
                            <div className="app-aiops-risk-item__name">
                              <StatusPulse healthy={false} />
                              {cluster.name}
                            </div>
                            <div className="app-aiops-risk-item__sub">
                              {cluster.type || 'Kubernetes'} · {cluster.k8sVersion || '版本待确认'}
                            </div>
                          </div>
                          <Tag color={getClusterStatusColor(cluster.status)}>
                            {getClusterStatusText(cluster.status)}
                          </Tag>
                        </div>
                        <div className="app-aiops-risk-item__foot">
                          <Text type="secondary">{recommendation}</Text>
                          <Space size={8}>
                            <Button
                              type="primary"
                              size="small"
                              onClick={() => history.push(`/clusters/${cluster.id}`)}
                            >
                              立即处理
                            </Button>
                            <Tooltip title="当前状态未恢复前不可进入集群管理">
                              <Button size="small" onClick={() => handleEnterCluster(cluster)}>
                                尝试进入
                              </Button>
                            </Tooltip>
                          </Space>
                        </div>
                      </div>
                    )
                  })}
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

        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <Card className="app-aiops-panel" title="最近告警 / 事件">
              {recentAlerts.length ? (
                <div className="app-aiops-alert-list">
                  {recentAlerts.map((alert) => (
                    <div key={alert.id} className="app-aiops-alert-item">
                      <Tag
                        color={alert.severity === 'critical' ? 'error' : alert.severity === 'warning' ? 'warning' : 'processing'}
                        style={{ margin: 0, fontSize: 11 }}
                      >
                        {getSeverityLabel(alert.severity)}
                      </Tag>
                      <div className="app-aiops-alert-item__body">
                        <div className="app-aiops-alert-item__title">{alert.name}</div>
                        <div className="app-aiops-alert-item__message">{alert.message}</div>
                      </div>
                      <span className="app-aiops-alert-item__time">{alert.time}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="app-aiops-empty-state">
                  <CheckCircleOutlined />
                  <div>
                    <strong>暂无活跃告警</strong>
                    <span>平台运行平稳，未检测到需要处理的事件。</span>
                  </div>
                </div>
              )}
            </Card>
          </Col>

          <Col xs={24} lg={12}>
            <Card className="app-aiops-panel" title="AI 智能洞察">
              <div className="app-aiops-insight-list">
                {aiInsights.map((insight) => (
                  <div key={insight.id} className="app-aiops-insight-item">
                    <span className="app-aiops-insight-item__icon">{insight.icon}</span>
                    <span className="app-aiops-insight-item__text">{insight.text}</span>
                  </div>
                ))}
              </div>
            </Card>
          </Col>
        </Row>

        <Card className="app-aiops-panel" title="平台入口">
          <div className="app-aiops-entry-bar">
            {QUICK_ENTRIES.map((entry) => (
              <Tooltip key={entry.key} title={entry.desc}>
                <button
                  type="button"
                  className={`app-aiops-entry-bar__item tone-${entry.tone}`}
                  onClick={() => history.push(entry.path)}
                >
                  <span className="app-aiops-entry-bar__icon">{entry.icon}</span>
                  <span className="app-aiops-entry-bar__title">{entry.title}</span>
                </button>
              </Tooltip>
            ))}
          </div>
        </Card>
      </div>
    </AppPage>
  )
}

export default DashboardPage
