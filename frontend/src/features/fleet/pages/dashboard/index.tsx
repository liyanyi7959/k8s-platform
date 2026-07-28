import React, { useMemo } from 'react'
import { history, useModel } from '@umijs/max'
import { useQueries, useQuery } from '@tanstack/react-query'
import {
  ArrowRightOutlined,
  ClockCircleOutlined,
  ClusterOutlined,
  DashboardOutlined,
  NodeIndexOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { Badge, Button, Card, Col, Empty, Progress, Row, Space, Spin, Tag, Tooltip, Typography } from 'antd'
import dayjs from 'dayjs'

import { AppPage } from '@/components'
import { listClusters } from '@/features/fleet/api/clusters'
import { getClusterOverview } from '@/features/kops/api/k8s'
import { listIncidents } from '@/features/incident/api'
import { enterClusterWorkspace, getClusterStatusColor, getClusterStatusText, isClusterHealthy } from '@/utils'
import type { IncidentStatus, MonitorIncident } from '@/shared/types'

const { Text, Title } = Typography

const severityMeta = {
  critical: { label: '严重', color: 'error' as const, weight: 3 },
  warning: { label: '警告', color: 'warning' as const, weight: 2 },
  info: { label: '提示', color: 'processing' as const, weight: 1 },
}

const actionByStatus: Record<IncidentStatus, string> = {
  open: '先认领事件并确认影响范围',
  acknowledged: '启动诊断，收集事件与日志证据',
  diagnosing: '查看诊断证据并确认处置方案',
  awaiting_approval: '审核变更范围与回滚方案',
  executing: '跟踪执行日志，避免扩大影响',
  verifying: '验证指标、事件和业务恢复情况',
  resolved: '复盘事件并沉淀运行手册',
}

const incidentImpact = (incident: MonitorIncident) =>
  [
    incident.clusterName || `集群 ${incident.clusterId}`,
    incident.namespace,
    incident.resourceName ? `${incident.resourceKind || '资源'} / ${incident.resourceName}` : '',
  ].filter(Boolean)

const DashboardPage: React.FC = () => {
  const { setCurrentCluster } = useModel('cluster')
  const clustersQuery = useQuery({
    queryKey: ['dashboard-clusters'],
    queryFn: ({ signal }) => listClusters({ page: 1, pageSize: 100 }, signal),
  })
  const incidentsQuery = useQuery({
    queryKey: ['dashboard-incidents'],
    queryFn: ({ signal }) => listIncidents({ page: 1, pageSize: 100 }, signal),
    refetchInterval: 30_000,
  })

  const clusters = clustersQuery.data?.items || []
  const observedClusters = clusters.slice(0, 12)
  const overviewQueries = useQueries({
    queries: observedClusters.map((cluster) => ({
      queryKey: ['dashboard-cluster-overview', cluster.id],
      queryFn: ({ signal }: { signal?: AbortSignal }) => getClusterOverview(cluster.id, signal),
      enabled: Boolean(cluster.id) && isClusterHealthy(cluster.status),
      staleTime: 60_000,
    })),
  })

  const overviewByCluster = useMemo(() => {
    const result = new Map<number, Awaited<ReturnType<typeof getClusterOverview>>>()
    overviewQueries.forEach((query, index) => {
      const cluster = observedClusters[index]
      if (cluster && query.data) result.set(cluster.id, query.data)
    })
    return result
  }, [observedClusters, overviewQueries])

  const activeIncidents = useMemo(
    () =>
      (incidentsQuery.data?.items || [])
        .filter((item) => item.status !== 'resolved')
        .sort((a, b) => {
          const severityDiff = severityMeta[b.severity].weight - severityMeta[a.severity].weight
          return severityDiff || new Date(a.startedAt).getTime() - new Date(b.startedAt).getTime()
        }),
    [incidentsQuery.data],
  )

  const criticalCount = activeIncidents.filter((item) => item.severity === 'critical').length
  const warningCount = activeIncidents.filter((item) => item.severity === 'warning').length
  const unassignedCount = activeIncidents.filter((item) => !item.assigneeName).length
  const affectedClusters = new Set(activeIncidents.map((item) => item.clusterId)).size
  const healthyClusters = clusters.filter((cluster) => isClusterHealthy(cluster.status)).length
  const totalNodes = clusters.reduce((sum, cluster) => sum + (cluster.nodeCount || 0), 0)
  const metricsCoverage = observedClusters.filter(
    (cluster) => overviewByCluster.get(cluster.id)?.meta?.metrics_available !== false && overviewByCluster.has(cluster.id),
  ).length
  const updatedAt = incidentsQuery.dataUpdatedAt || clustersQuery.dataUpdatedAt
  const isLoading = clustersQuery.isLoading || incidentsQuery.isLoading

  const headline = criticalCount
    ? `${criticalCount} 个严重事件需要立即处理`
    : activeIncidents.length
      ? `${activeIncidents.length} 个事件正在处置`
      : '当前没有未恢复事件'

  return (
    <AppPage keepHeaderTitle title="全局运维态势">
      <div className="app-page-shell app-ops-dashboard">
        <section className={`app-ops-briefing ${criticalCount ? 'is-critical' : activeIncidents.length ? 'is-warning' : 'is-stable'}`}>
          <div className="app-ops-briefing__source">
            <Badge status={activeIncidents.length ? 'processing' : 'success'} />
            <span>Alertmanager + Kubernetes API</span>
            <span className="app-ops-briefing__divider" />
            <ClockCircleOutlined />
            <span>{updatedAt ? `更新于 ${dayjs(updatedAt).format('HH:mm:ss')}` : '正在确认数据时间'}</span>
          </div>
          <div className="app-ops-briefing__body">
            <div className="app-ops-briefing__message">
              <span className="app-ops-kicker">当前风险</span>
              <Title level={2}>{headline}</Title>
              <Text>
                {activeIncidents.length
                  ? `影响 ${affectedClusters} 个集群，其中 ${unassignedCount} 个事件尚未认领。先处理严重且持续时间最长的事件。`
                  : '所有接入事件均已恢复。继续关注集群指标采集覆盖与健康状态。'}
              </Text>
              <Space wrap>
                <Button type="primary" danger={criticalCount > 0} onClick={() => history.push('/monitor/events')}>
                  {activeIncidents.length ? '进入事件处置' : '查看事件记录'}
                </Button>
                <Button onClick={() => history.push('/clusters')}>查看集群运行面</Button>
              </Space>
            </div>
            <div className="app-ops-briefing__metrics" aria-label="当前风险摘要">
              <div><span>严重</span><strong className={criticalCount > 0 ? 'is-critical' : undefined}>{criticalCount}</strong></div>
              <div><span>警告</span><strong className={warningCount > 0 ? 'is-warning' : undefined}>{warningCount}</strong></div>
              <div><span>影响集群</span><strong>{affectedClusters}</strong></div>
              <div><span>未认领</span><strong>{unassignedCount}</strong></div>
            </div>
          </div>
        </section>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={16}>
            <Card
              className="app-ops-panel app-ops-priority-panel"
              title="优先处置队列"
              extra={<Button type="link" onClick={() => history.push('/monitor/events')}>查看全部 <ArrowRightOutlined /></Button>}
            >
              {incidentsQuery.isLoading ? (
                <div className="app-ops-loading"><Spin /></div>
              ) : activeIncidents.length ? (
                <div className="app-ops-incident-list">
                  {activeIncidents.slice(0, 5).map((incident) => (
                    <article key={incident.id} className={`app-ops-incident severity-${incident.severity}`}>
                      <div className="app-ops-incident__severity">
                        <Tag color={severityMeta[incident.severity].color}>{severityMeta[incident.severity].label}</Tag>
                      </div>
                      <div className="app-ops-incident__body">
                        <div className="app-ops-incident__title">{incident.alertName}</div>
                        <div className="app-ops-incident__summary">{incident.summary || 'Alertmanager 未提供事件摘要'}</div>
                        <Space size={[6, 6]} wrap className="app-ops-incident__scope">
                          {incidentImpact(incident).map((item) => <span key={item}>{item}</span>)}
                          <span>{incident.assigneeName ? `负责人 ${incident.assigneeName}` : '尚未认领'}</span>
                        </Space>
                      </div>
                      <div className="app-ops-incident__action">
                        <Text>{actionByStatus[incident.status]}</Text>
                        <Button size="small" type="primary" onClick={() => history.push(`/monitor/events?incident=${incident.id}`)}>
                          处置
                        </Button>
                      </div>
                    </article>
                  ))}
                </div>
              ) : (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无未恢复事件">
                  <Button onClick={() => history.push('/monitor/events')}>查看历史事件</Button>
                </Empty>
              )}
            </Card>
          </Col>

          <Col xs={24} xl={8}>
            <Card className="app-ops-panel" title="运行面摘要">
              <div className="app-ops-facts">
                <div><ClusterOutlined /><span>纳管集群</span><strong>{clusters.length}</strong></div>
                <div><SafetyCertificateOutlined /><span>健康集群</span><strong>{healthyClusters}</strong></div>
                <div><NodeIndexOutlined /><span>节点总数</span><strong>{totalNodes}</strong></div>
                <div><DashboardOutlined /><span>指标覆盖</span><strong>{metricsCoverage}/{observedClusters.length}</strong></div>
              </div>
              <div className="app-ops-data-note">
                <DashboardOutlined />
                <span>容量数据只展示 Kubernetes API 与 metrics.k8s.io 的真实采样；未采集时显示“未采集”，不再估算。</span>
              </div>
            </Card>
          </Col>
        </Row>

        <Card
          className="app-ops-panel"
          title="集群运行面"
          extra={<Text type="secondary">最多展示 12 个集群 · 指标按各集群实际更新时间</Text>}
        >
          {isLoading ? (
            <div className="app-ops-loading"><Spin /></div>
          ) : clusters.length === 0 ? (
            <Empty description="尚未接入集群"><Button type="primary" onClick={() => history.push('/clusters/import')}>导入集群</Button></Empty>
          ) : (
            <div className="app-ops-cluster-list">
              <div className="app-ops-cluster-list__head" aria-hidden="true">
                <span>集群</span>
                <span>Ready 节点</span>
                <span>CPU</span>
                <span>内存</span>
                <span>更新</span>
              </div>
              {observedClusters.map((cluster) => {
                const overview = overviewByCluster.get(cluster.id)
                const metricsAvailable = overview?.meta?.metrics_available !== false && Boolean(overview)
                return (
                  <button
                    type="button"
                    key={cluster.id}
                    className="app-ops-cluster-row"
                    onClick={() => enterClusterWorkspace(cluster, { setCurrentCluster })}
                  >
                    <div className="app-ops-cluster-row__identity">
                      <span className="app-ops-cluster-row__icon"><ClusterOutlined /></span>
                      <div>
                        <div className="app-ops-cluster-row__name">
                          <strong>{cluster.name}</strong>
                          <Tag color={getClusterStatusColor(cluster.status)}>{getClusterStatusText(cluster.status)}</Tag>
                        </div>
                        <span>{cluster.k8sVersion || '版本待确认'}</span>
                      </div>
                    </div>
                    <div className="app-ops-cluster-row__nodes">
                      <span>Ready 节点</span>
                      <strong>{overview ? `${overview.stats.nodes.ready}/${overview.stats.nodes.total}` : `${cluster.nodeCount || '—'}`}</strong>
                    </div>
                    <div className="app-ops-cluster-row__metric">
                      <span>CPU</span>
                      {metricsAvailable ? <Progress percent={overview!.stats.cpu.used_percent} size="small" /> : <Text type="secondary">未采集</Text>}
                    </div>
                    <div className="app-ops-cluster-row__metric">
                      <span>内存</span>
                      {metricsAvailable ? <Progress percent={overview!.stats.memory.used_percent} size="small" /> : <Text type="secondary">未采集</Text>}
                    </div>
                    <Tooltip title={overview?.meta ? `来源 ${overview.meta.metrics_source || overview.meta.source}；更新 ${dayjs(overview.meta.updated_at).format('YYYY-MM-DD HH:mm:ss')}` : '尚未取得集群概览'}>
                      <span className="app-ops-cluster-row__freshness">
                        {overview?.meta?.updated_at ? dayjs(overview.meta.updated_at).format('HH:mm:ss') : '待更新'}
                      </span>
                    </Tooltip>
                  </button>
                )
              })}
            </div>
          )}
        </Card>
      </div>
    </AppPage>
  )
}

export default DashboardPage
