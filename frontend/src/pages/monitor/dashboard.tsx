import React, { useMemo } from 'react'
import { history } from '@umijs/max'
import { useQueries, useQuery } from '@tanstack/react-query'
import { AlertOutlined, CheckCircleOutlined, ClockCircleOutlined, ClusterOutlined, DashboardOutlined, WarningOutlined } from '@ant-design/icons'
import { Badge, Button, Card, Col, Empty, Progress, Row, Spin, Tag, Typography } from 'antd'
import dayjs from 'dayjs'

import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'
import { getClusterOverview } from '@/services/k8s'
import { listIncidents } from '@/services/monitor'
import { isClusterHealthy } from '@/utils'
import { DESIGN_COLORS } from '@/theme/designTokens'

const { Text } = Typography

const severityLabel = { critical: '严重', warning: '警告', info: '提示' }

const MonitorDashboardPage: React.FC = () => {
  const clustersQuery = useQuery({
    queryKey: ['monitor-clusters'],
    queryFn: ({ signal }) => listClusters({ page: 1, pageSize: 100 }, signal),
  })
  const incidentsQuery = useQuery({
    queryKey: ['monitor-incidents-overview'],
    queryFn: ({ signal }) => listIncidents({ page: 1, pageSize: 100 }, signal),
    refetchInterval: 30_000,
  })
  const clusters = clustersQuery.data?.items || []
  const monitoredClusters = clusters.slice(0, 12)
  const overviewQueries = useQueries({
    queries: monitoredClusters.map((cluster) => ({
      queryKey: ['monitor-cluster-overview', cluster.id],
      queryFn: ({ signal }: { signal?: AbortSignal }) => getClusterOverview(cluster.id, signal),
      enabled: isClusterHealthy(cluster.status),
      staleTime: 60_000,
    })),
  })

  const activeIncidents = useMemo(
    () => (incidentsQuery.data?.items || []).filter((item) => item.status !== 'resolved'),
    [incidentsQuery.data],
  )
  const criticalCount = activeIncidents.filter((item) => item.severity === 'critical').length
  const healthyCount = clusters.filter((cluster) => isClusterHealthy(cluster.status)).length
  const totalNodes = clusters.reduce((sum, cluster) => sum + (cluster.nodeCount || 0), 0)
  const metricsCoverage = overviewQueries.filter(
    (query) => query.data && query.data.meta?.metrics_available !== false,
  ).length
  const lastUpdated = Math.max(clustersQuery.dataUpdatedAt || 0, incidentsQuery.dataUpdatedAt || 0)

  return (
    <AppPage keepHeaderTitle title="监控概览">
      <div className="app-page-shell app-monitor-overview">
        <div className="app-data-provenance">
          <span><Badge status={criticalCount ? 'error' : 'success'} />实时监控状态</span>
          <span>事件来源：Alertmanager</span>
          <span>资源来源：Kubernetes API / metrics.k8s.io</span>
          <span><ClockCircleOutlined /> {lastUpdated ? `更新于 ${dayjs(lastUpdated).format('HH:mm:ss')}` : '正在同步'}</span>
        </div>

        <Row gutter={[12, 12]}>
          <Col xs={12} md={6}><div className="app-monitor-fact"><ClusterOutlined /><span>纳管集群</span><strong>{clusters.length}</strong></div></Col>
          <Col xs={12} md={6}><div className="app-monitor-fact"><CheckCircleOutlined /><span>健康集群</span><strong>{healthyCount}</strong></div></Col>
          <Col xs={12} md={6}><div className="app-monitor-fact"><DashboardOutlined /><span>指标覆盖</span><strong>{metricsCoverage}/{monitoredClusters.length}</strong></div></Col>
          <Col xs={12} md={6}><div className={`app-monitor-fact ${criticalCount ? 'is-critical' : ''}`}><AlertOutlined /><span>未恢复事件</span><strong>{activeIncidents.length}</strong></div></Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={16}>
            <Card className="app-ops-panel" title="集群实时指标" extra={<Text type="secondary">{totalNodes} 个节点 · 未采集不估算</Text>}>
              {clustersQuery.isLoading ? <div className="app-ops-loading"><Spin /></div> : (
                <div className="app-monitor-cluster-grid">
                  {monitoredClusters.map((cluster, index) => {
                    const overview = overviewQueries[index]?.data
                    const available = Boolean(overview) && overview?.meta?.metrics_available !== false
                    return (
                      <article key={cluster.id} className="app-monitor-cluster-card">
                        <div className="app-monitor-cluster-card__head">
                          <div><strong>{cluster.name}</strong><span>{cluster.k8sVersion || '版本待确认'}</span></div>
                          <Badge status={isClusterHealthy(cluster.status) ? 'success' : 'error'} text={isClusterHealthy(cluster.status) ? '连接正常' : '连接异常'} />
                        </div>
                        <div className="app-monitor-cluster-card__nodes">
                          <span>Ready 节点</span>
                          <strong>{overview ? `${overview.stats.nodes.ready}/${overview.stats.nodes.total}` : cluster.nodeCount || '—'}</strong>
                          <span>Pod</span>
                          <strong>{overview?.stats.pods.total ?? '—'}</strong>
                        </div>
                        <div className="app-monitor-cluster-card__metric">
                          <div><span>CPU</span><strong>{available ? `${overview!.stats.cpu.used_percent}%` : '未采集'}</strong></div>
                          <Progress percent={available ? overview!.stats.cpu.used_percent : 0} showInfo={false} status={available ? undefined : 'normal'} />
                        </div>
                        <div className="app-monitor-cluster-card__metric">
                          <div><span>内存</span><strong>{available ? `${overview!.stats.memory.used_percent}%` : '未采集'}</strong></div>
                          <Progress percent={available ? overview!.stats.memory.used_percent : 0} showInfo={false} strokeColor={DESIGN_COLORS.dataSecondary} />
                        </div>
                        <div className="app-monitor-cluster-card__source">
                          {overview?.meta?.updated_at
                            ? `${overview.meta.metrics_source || overview.meta.source} · ${dayjs(overview.meta.updated_at).format('HH:mm:ss')}`
                            : '尚未取得真实指标'}
                        </div>
                      </article>
                    )
                  })}
                </div>
              )}
            </Card>
          </Col>
          <Col xs={24} xl={8}>
            <Card
              className="app-ops-panel"
              title="未恢复事件"
              extra={<Button type="link" onClick={() => history.push('/monitor/events')}>进入处置</Button>}
            >
              {incidentsQuery.isLoading ? <div className="app-ops-loading"><Spin /></div> : activeIncidents.length ? (
                <div className="app-monitor-event-list">
                  {activeIncidents.slice(0, 6).map((incident) => (
                    <button type="button" key={incident.id} onClick={() => history.push(`/monitor/events?incident=${incident.id}`)}>
                      <Tag color={incident.severity === 'critical' ? 'error' : incident.severity === 'warning' ? 'warning' : 'processing'}>
                        {severityLabel[incident.severity]}
                      </Tag>
                      <div><strong>{incident.alertName}</strong><span>{incident.clusterName || `集群 ${incident.clusterId}`} · {dayjs(incident.startedAt).format('MM-DD HH:mm')}</span></div>
                    </button>
                  ))}
                </div>
              ) : (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无未恢复事件" />
              )}
              {criticalCount > 0 ? <div className="app-inline-alert"><WarningOutlined /><span>{criticalCount} 个严重事件需要优先处理</span></div> : null}
            </Card>
          </Col>
        </Row>
      </div>
    </AppPage>
  )
}

export default MonitorDashboardPage
