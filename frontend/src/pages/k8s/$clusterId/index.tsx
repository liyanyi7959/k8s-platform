import React, { useMemo, useState } from 'react'
import {
  Row,
  Col,
  Card,
  Progress,
  Tag,
  Space,
  Typography,
  Badge,
  Tooltip,
  Button,
  Switch,
  Select,
  Spin,
  Empty,
  Table,
  Alert,
} from 'antd'
import {
  ClusterOutlined,
  CloudServerOutlined,
  AppstoreOutlined,
  NodeIndexOutlined,
  ReloadOutlined,
  WarningOutlined,
  LockOutlined,
  CloseCircleFilled,
  DashboardOutlined,
} from '@ant-design/icons'
import { Line, Pie, Column, Gauge } from '@ant-design/charts'
import { useQuery } from '@tanstack/react-query'
import { getClusterOverview } from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'
import { AppPage } from '@/components'
import { history } from '@umijs/max'

const { Text, Title } = Typography

/** 健康评分等级 */
function getHealthTier(score: number): 'good' | 'warn' | 'bad' {
  return score >= 80 ? 'good' : score >= 60 ? 'warn' : 'bad'
}

function getHealthHint(score: number): string {
  return score >= 90 ? '运行优秀' : score >= 80 ? '运行良好' : score >= 60 ? '需要关注' : '存在风险'
}

function getHealthColor(tier: 'good' | 'warn' | 'bad'): string {
  return tier === 'good' ? '#52c41a' : tier === 'warn' ? '#faad14' : '#ff4d4f'
}

/** 证书状态标签 */
function certStatusLabel(s: string): string {
  if (s === 'critical') return '紧急'
  if (s === 'warn') return '预警'
  if (s === 'ok') return '正常'
  return '未知'
}

function certStatusColor(s: string): string {
  if (s === 'critical') return '#ff4d4f'
  if (s === 'warn') return '#faad14'
  if (s === 'ok') return '#52c41a'
  return '#d9d9d9'
}

/** 时间格式化 */
function timeAgo(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime()
  if (diff < 0 || !Number.isFinite(diff)) return ''
  const m = Math.floor(diff / 60000)
  if (m < 1) return '刚刚'
  if (m < 60) return `${m}分钟前`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}小时前`
  return `${Math.floor(h / 24)}天前`
}

const K8sDashboardPage: React.FC = () => {
  const clusterId = useClusterId()
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState(30)
  const [expandedAlert, setExpandedAlert] = useState<number>(-1)

  const {
    data: overview,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ['k8s-overview', clusterId],
    queryFn: ({ signal }) => getClusterOverview(clusterId, signal),
    refetchInterval: autoRefresh ? refreshInterval * 1000 : false,
  })

  /** 健康计算 */
  const healthScore = useMemo(() => {
    if (!overview) return 0
    const s = overview.stats
    const nodeScore = s.nodes.total > 0 ? Math.round((s.nodes.ready / s.nodes.total) * 100) : 0
    const podTotal = Math.max(1, s.pods.total)
    const podScore = Math.max(0, Math.min(100, Math.round((s.pods.running / podTotal) * 100)))
    const cpuScore = Math.max(0, 100 - s.cpu.used_percent)
    const memScore = Math.max(0, 100 - s.memory.used_percent)
    const pressureScore = Math.round((cpuScore + memScore) / 2)
    return Math.round(nodeScore * 0.3 + podScore * 0.3 + pressureScore * 0.2 + 80 * 0.2)
  }, [overview])

  const healthBreakdown = useMemo(() => {
    if (!overview) return []
    const s = overview.stats
    const nodeScore = s.nodes.total > 0 ? Math.round((s.nodes.ready / s.nodes.total) * 100) : 0
    const podTotal = Math.max(1, s.pods.total)
    const podScore = Math.max(0, Math.min(100, Math.round((s.pods.running / podTotal) * 100)))
    const pressureScore = Math.max(
      0,
      Math.round(100 - (s.cpu.used_percent + s.memory.used_percent) / 2),
    )
    return [
      {
        key: 'nodes',
        label: '节点可用性',
        score: nodeScore,
        hint: `${s.nodes.ready}/${s.nodes.total} Ready`,
      },
      {
        key: 'pods',
        label: 'Pod 运行率',
        score: podScore,
        hint: `${s.pods.running}/${podTotal} Running`,
      },
      {
        key: 'pressure',
        label: '资源压力',
        score: pressureScore,
        hint: `CPU ${s.cpu.used_percent}% / 内存 ${s.memory.used_percent}%`,
      },
    ].map((item) => ({ ...item, tier: getHealthTier(item.score) }))
  }, [overview])

  const warningEvents = useMemo(() => {
    if (!overview?.events) return []
    const warnings = overview.events.filter((e) => e.type === 'Warning')
    return (warnings.length ? warnings : overview.events).slice(0, 5)
  }, [overview])

  const alertCounts = useMemo(() => {
    const counts = { critical: 0, warning: 0, info: 0 }
    if (!overview?.events) return counts
    overview.events.forEach((e) => {
      if (e.type === 'Warning') counts.warning++
      else counts.info++
    })
    return counts
  }, [overview])

  const workloadsTotal = useMemo(() => {
    if (!overview) return 0
    const w = overview.stats.workloads
    return w.deployments + w.statefulsets + w.daemonsets
  }, [overview])

  const failedPods = overview?.anomalies?.failed_pods || []

  if (isLoading) {
    return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />
  }

  if (!overview) {
    return (
      <AppPage>
        <Empty description="暂无集群概览数据">
          <Button type="primary" icon={<ReloadOutlined />} onClick={() => refetch()}>
            重新加载
          </Button>
        </Empty>
      </AppPage>
    )
  }

  const tier = getHealthTier(healthScore)
  const color = getHealthColor(tier)

  return (
    <AppPage>
      {/* ═══ Header ═══ */}
      <Card style={{ marginBottom: 16 }}>
        <Row justify="space-between" align="middle">
          <Col>
            <Space direction="vertical" size={8}>
              <Space align="center">
                <Title level={4} style={{ margin: 0 }}>
                  {overview.cluster.name}
                </Title>
                <Tag color={overview.cluster.status === 'Active' ? 'success' : 'warning'}>
                  {overview.cluster.status}
                </Tag>
              </Space>
              <Space size={16} wrap>
                <Text type="secondary">
                  <ClusterOutlined /> {overview.stats.namespaces ?? '-'} 个命名空间
                </Text>
                <Text type="secondary">
                  <NodeIndexOutlined /> Ready {overview.stats.nodes.ready}/
                  {overview.stats.nodes.total}
                </Text>
                <Text type="secondary">
                  <CloudServerOutlined /> Pending Pods {overview.stats.pods.pending}
                </Text>
                <Text type="secondary">
                  <WarningOutlined /> 告警 {warningEvents.length}
                </Text>
              </Space>
            </Space>
          </Col>
          <Col>
            <Space size={16} align="center">
              <Space>
                <Switch
                  checked={autoRefresh}
                  onChange={setAutoRefresh}
                  checkedChildren="自动刷新"
                  unCheckedChildren="手动"
                />
                <Select
                  value={refreshInterval}
                  onChange={setRefreshInterval}
                  size="small"
                  style={{ width: 90 }}
                  disabled={!autoRefresh}
                  options={[15, 30, 60, 120].map((v) => ({ value: v, label: `${v} 秒` }))}
                />
                <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
                  立即刷新
                </Button>
              </Space>
              <div style={{ textAlign: 'center' }}>
                <Progress
                  type="dashboard"
                  percent={healthScore}
                  size={80}
                  strokeColor={color}
                  format={(p) => <span style={{ fontSize: 22, fontWeight: 900, color }}>{p}</span>}
                />
                <div>
                  <Text type="secondary" style={{ fontSize: 11 }}>
                    集群健康
                  </Text>
                  <br />
                  <Text strong style={{ fontSize: 12, color }}>
                    {getHealthHint(healthScore)}
                  </Text>
                </div>
              </div>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* ═══ Health Breakdown + Focus ═══ */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} md={14}>
          <Card title="健康拆解" size="small">
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              {healthBreakdown.map((item) => (
                <div key={item.key}>
                  <Row justify="space-between" style={{ marginBottom: 4 }}>
                    <Text>{item.label}</Text>
                    <Text strong style={{ color: getHealthColor(item.tier) }}>
                      {item.score}
                    </Text>
                  </Row>
                  <Progress
                    percent={item.score}
                    showInfo={false}
                    strokeColor={getHealthColor(item.tier)}
                    size="small"
                  />
                  <Text type="secondary" style={{ fontSize: 11 }}>
                    {item.hint}
                  </Text>
                </div>
              ))}
            </Space>
          </Card>
        </Col>
        <Col xs={24} md={10}>
          <Card title="当前关注项" size="small">
            <Row gutter={[12, 12]}>
              {[
                {
                  label: '未就绪节点',
                  value: Math.max(0, overview.stats.nodes.total - overview.stats.nodes.ready),
                  path: `/k8s/${clusterId}/nodes`,
                },
                {
                  label: 'Pending Pods',
                  value: overview.stats.pods.pending,
                  path: `/k8s/${clusterId}/pods`,
                },
                {
                  label: '失败 Pods',
                  value: overview.stats.pods.failed,
                  path: `/k8s/${clusterId}/pods`,
                },
                {
                  label: 'Warning 事件',
                  value: warningEvents.length,
                  path: `/k8s/${clusterId}/pods`,
                },
                {
                  label: '工作负载总量',
                  value: workloadsTotal,
                  path: `/k8s/${clusterId}/workloads`,
                },
                {
                  label: '证书预警',
                  value: overview.risks?.certificates?.filter((c) => c.status !== 'ok').length ?? 0,
                  path: '#',
                },
              ].map((item) => (
                <Col span={8} key={item.label}>
                  <Card
                    hoverable
                    size="small"
                    style={{ textAlign: 'center', cursor: 'pointer' }}
                    onClick={() => item.path !== '#' && history.push(item.path)}
                  >
                    <Text type="secondary" style={{ fontSize: 11 }}>
                      {item.label}
                    </Text>
                    <div style={{ fontSize: 24, fontWeight: 900, lineHeight: 1.2 }}>
                      {item.value}
                    </div>
                  </Card>
                </Col>
              ))}
            </Row>
          </Card>
        </Col>
      </Row>

      {/* ═══ KPI Cards ═══ */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        {[
          {
            label: '节点',
            value: overview.stats.nodes.total,
            sub: `Ready ${overview.stats.nodes.ready}/${overview.stats.nodes.total}`,
            icon: <ClusterOutlined />,
            color: '#0891b2',
            path: `/k8s/${clusterId}/nodes`,
          },
          {
            label: 'Pods',
            value: overview.stats.pods.total,
            sub: `Running ${overview.stats.pods.running}`,
            icon: <CloudServerOutlined />,
            color: '#2563eb',
            path: `/k8s/${clusterId}/pods`,
          },
          {
            label: 'CPU',
            value: `${overview.stats.cpu.used_percent}%`,
            sub: '使用率',
            icon: <DashboardOutlined />,
            color: '#7c3aed',
            path: `/k8s/${clusterId}/nodes`,
          },
          {
            label: '内存',
            value: `${overview.stats.memory.used_percent}%`,
            sub: '使用率',
            icon: <AppstoreOutlined />,
            color: '#d97706',
            path: `/k8s/${clusterId}/nodes`,
          },
        ].map((kpi) => (
          <Col xs={12} sm={6} key={kpi.label}>
            <Card
              hoverable
              style={{ borderLeft: `3px solid ${kpi.color}`, cursor: 'pointer' }}
              onClick={() => history.push(kpi.path)}
            >
              <Space>
                <div
                  style={{
                    width: 40,
                    height: 40,
                    borderRadius: 8,
                    background: `${kpi.color}15`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: kpi.color,
                    fontSize: 18,
                  }}
                >
                  {kpi.icon}
                </div>
                <div>
                  <Text type="secondary" style={{ fontSize: 11 }}>
                    {kpi.label}
                  </Text>
                  <div style={{ fontSize: 22, fontWeight: 900, lineHeight: 1 }}>{kpi.value}</div>
                  <Text type="secondary" style={{ fontSize: 10 }}>
                    {kpi.sub}
                  </Text>
                </div>
              </Space>
            </Card>
          </Col>
        ))}
      </Row>

      {/* ═══ Charts Grid ═══ */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        {/* CPU / Memory 24h Trend */}
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space>
                <span style={{ color: '#0891b2' }}>CPU / 内存 24h 趋势</span>
                <Tag>近 24 小时</Tag>
              </Space>
            }
            size="small"
            bodyStyle={{ padding: '12px 16px' }}
          >
            <Line
              data={overview.charts.cpu_memory_24h.labels.flatMap((label, i) => [
                { time: label, type: 'CPU', value: overview.charts.cpu_memory_24h.cpu[i] },
                { time: label, type: '内存', value: overview.charts.cpu_memory_24h.memory[i] },
              ])}
              xField="time"
              yField="value"
              colorField="type"
              color={['#0891b2', '#7c3aed']}
              smooth
              height={220}
              point={false}
              xAxis={{ label: { autoRotate: false } }}
              yAxis={{ min: 0, max: 100 }}
              annotations={[
                {
                  type: 'line',
                  yField: 80,
                  style: { stroke: '#d97706', lineDash: [4, 4] },
                  label: {
                    text: '警告 80%',
                    position: 'end',
                    style: { fill: '#d97706', fontSize: 10 },
                  },
                },
                {
                  type: 'line',
                  yField: 90,
                  style: { stroke: '#dc2626', lineDash: [4, 4] },
                  label: {
                    text: '危险 90%',
                    position: 'end',
                    style: { fill: '#dc2626', fontSize: 10 },
                  },
                },
              ]}
            />
          </Card>
        </Col>

        {/* Pod Phase */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <span style={{ color: '#2563eb' }}>Pod 状态分布</span>
                <Tag>{overview.stats.pods.total} 总计</Tag>
              </Space>
            }
            size="small"
            bodyStyle={{ padding: '12px 16px' }}
          >
            <Pie
              data={[
                { type: 'Running', value: overview.charts.pod_phase.running },
                { type: 'Pending', value: overview.charts.pod_phase.pending },
                { type: 'Failed', value: overview.charts.pod_phase.failed },
                { type: 'Succeeded', value: overview.charts.pod_phase.succeeded },
              ].filter((d) => d.value > 0)}
              angleField="value"
              colorField="type"
              color={['#10b981', '#f59e0b', '#ef4444', '#94a3b8']}
              radius={0.85}
              innerRadius={0.6}
              height={220}
              label={{ text: 'type', position: 'outside' }}
              legend={{ position: 'bottom' }}
              statistic={{
                title: { content: 'Pods' },
                content: { content: String(overview.stats.pods.total) },
              }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        {/* Workload Distribution */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <span style={{ color: '#7c3aed' }}>工作负载分布</span>
                <Tag>{workloadsTotal} 总计</Tag>
              </Space>
            }
            size="small"
            bodyStyle={{ padding: '12px 16px' }}
          >
            <Pie
              data={[
                { type: 'Deployment', value: overview.stats.workloads.deployments },
                { type: 'StatefulSet', value: overview.stats.workloads.statefulsets },
                { type: 'DaemonSet', value: overview.stats.workloads.daemonsets },
              ].filter((d) => d.value > 0)}
              angleField="value"
              colorField="type"
              color={['#3b82f6', '#8b5cf6', '#f97316']}
              radius={0.85}
              innerRadius={0.6}
              height={220}
              label={{ text: 'type', position: 'outside' }}
              legend={{ position: 'bottom' }}
              statistic={{
                title: { content: '负载' },
                content: { content: String(workloadsTotal) },
              }}
            />
          </Card>
        </Col>

        {/* Namespace Pod Top */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <span style={{ color: '#d97706' }}>Namespace Pod 排行</span>
                <Tag>Top {overview.charts.namespace_pods_top.length}</Tag>
              </Space>
            }
            size="small"
            bodyStyle={{ padding: '12px 16px' }}
          >
            <Column
              data={[...overview.charts.namespace_pods_top].sort((a, b) => a.pods - b.pods)}
              xField="namespace"
              yField="pods"
              color="#d97706"
              height={220}
              label={{ position: 'top', style: { fontWeight: 700, fontSize: 11 } }}
              xAxis={{ label: { autoRotate: true } }}
              style={{ radiusTopLeft: 4, radiusTopRight: 4 }}
            />
          </Card>
        </Col>

        {/* Node Ready Gauge */}
        <Col xs={24} lg={8}>
          <Card
            title={
              <Space>
                <span style={{ color: '#059669' }}>节点就绪率</span>
                <Tag>
                  {overview.charts.node_ready.ready}/{overview.charts.node_ready.total}
                </Tag>
              </Space>
            }
            size="small"
            bodyStyle={{ padding: '12px 16px' }}
          >
            <Gauge
              percent={
                overview.charts.node_ready.total > 0
                  ? overview.charts.node_ready.ready / overview.charts.node_ready.total
                  : 0
              }
              height={220}
              innerRadius={0.75}
              range={{
                color:
                  overview.charts.node_ready.ready /
                    Math.max(1, overview.charts.node_ready.total) >=
                  0.9
                    ? '#059669'
                    : '#d97706',
              }}
              indicator={false}
              statistic={{
                title: {
                  content: `${overview.charts.node_ready.ready} / ${overview.charts.node_ready.total} Ready`,
                  style: { fontSize: 12, fontWeight: 600 },
                },
                content: {
                  content: `${Math.round((overview.charts.node_ready.ready / Math.max(1, overview.charts.node_ready.total)) * 100)}%`,
                  style: { fontSize: 28, fontWeight: 900 },
                },
              }}
            />
          </Card>
        </Col>
      </Row>

      {/* ═══ Failed Pods ═══ */}
      {failedPods.length > 0 && (
        <Card
          title={
            <Space>
              <CloseCircleFilled style={{ color: '#ff4d4f' }} />
              异常 Pod
              <Badge count={failedPods.length} style={{ backgroundColor: '#ff4d4f' }} />
            </Space>
          }
          size="small"
          style={{ marginBottom: 16 }}
        >
          <Table
            dataSource={failedPods}
            rowKey={(r) => `${r.namespace}/${r.name}`}
            pagination={false}
            size="small"
            columns={[
              { title: 'Namespace', dataIndex: 'namespace', width: 120 },
              { title: '名称', dataIndex: 'name' },
              {
                title: '原因',
                dataIndex: 'reason',
                width: 160,
                render: (v: string) => <Tag color="error">{v}</Tag>,
              },
            ]}
            onRow={(_record) => ({
              style: { cursor: 'pointer' },
              onClick: () => history.push(`/k8s/${clusterId}/pods`),
            })}
          />
        </Card>
      )}

      {/* ═══ Alert Events ═══ */}
      {warningEvents.length > 0 && (
        <Card
          title={
            <Space>
              <WarningOutlined style={{ color: '#ff4d4f' }} />
              告警事件
              <Badge count={warningEvents.length} style={{ backgroundColor: '#ff4d4f' }} />
              {alertCounts.critical > 0 && <Tag color="error">{alertCounts.critical} 严重</Tag>}
              {alertCounts.warning > 0 && <Tag color="warning">{alertCounts.warning} 警告</Tag>}
            </Space>
          }
          size="small"
          style={{ marginBottom: 16 }}
        >
          {warningEvents.map((e, idx) => (
            <Alert
              key={idx}
              type={
                e.reason === 'Unhealthy' || e.reason === 'CrashLoopBackOff' ? 'error' : 'warning'
              }
              showIcon
              style={{ marginBottom: 8, cursor: 'pointer' }}
              message={
                <Space>
                  <Tag color={e.reason === 'Unhealthy' ? 'error' : 'warning'}>{e.reason}</Tag>
                  <Text>{e.message}</Text>
                  <Text type="secondary" style={{ fontSize: 11, marginLeft: 'auto' }}>
                    {e.lastTimestamp ? timeAgo(e.lastTimestamp) : ''}
                  </Text>
                </Space>
              }
              description={
                expandedAlert === idx ? (
                  <div style={{ marginTop: 8 }}>
                    <Text type="secondary" style={{ fontSize: 11 }}>
                      完整信息
                    </Text>
                    <pre
                      style={{
                        background: '#0f172a',
                        color: '#cbd5e1',
                        padding: 12,
                        borderRadius: 8,
                        fontSize: 12,
                        marginTop: 4,
                        whiteSpace: 'pre-wrap',
                      }}
                    >
                      {e.message}
                    </pre>
                    <Space size={8}>
                      {e.involvedObject && (
                        <Tag>
                          资源: {e.namespace}/{e.involvedObject}
                        </Tag>
                      )}
                      {e.count && <Tag>次数: {e.count}</Tag>}
                    </Space>
                  </div>
                ) : undefined
              }
              onClick={() => setExpandedAlert(expandedAlert === idx ? -1 : idx)}
            />
          ))}
        </Card>
      )}

      {/* ═══ Certificate Risk ═══ */}
      {overview.risks && overview.risks.certificates.length > 0 && (
        <Card
          title={
            <Space>
              <LockOutlined />
              证书风险
              <Badge count={overview.risks.certificates.length} />
              {overview.risks.certificates.filter((c) => c.status === 'critical').length > 0 && (
                <Tag color="error">
                  {overview.risks.certificates.filter((c) => c.status === 'critical').length} 紧急
                </Tag>
              )}
              {overview.risks.certificates.filter((c) => c.status === 'warn').length > 0 && (
                <Tag color="warning">
                  {overview.risks.certificates.filter((c) => c.status === 'warn').length} 预警
                </Tag>
              )}
            </Space>
          }
          size="small"
          style={{ marginBottom: 16 }}
        >
          <Row gutter={[12, 12]}>
            {overview.risks.certificates.map((cert) => (
              <Col xs={24} sm={12} md={8} key={cert.key}>
                <Tooltip
                  title={`${cert.component} - ${cert.purpose}${cert.not_after ? ` | 过期: ${cert.not_after}` : ''}`}
                >
                  <Card
                    size="small"
                    style={{ borderLeft: `4px solid ${certStatusColor(cert.status)}` }}
                    bodyStyle={{ padding: '12px 16px' }}
                  >
                    <Row justify="space-between" align="middle">
                      <div>
                        <Text strong>{cert.name}</Text>
                        <br />
                        <Text type="secondary" style={{ fontSize: 11 }}>
                          {cert.component}
                        </Text>
                      </div>
                      <div style={{ textAlign: 'right' }}>
                        {cert.days_left != null ? (
                          <>
                            <div
                              style={{
                                fontSize: 24,
                                fontWeight: 900,
                                color: certStatusColor(cert.status),
                                lineHeight: 1,
                              }}
                            >
                              {cert.days_left}
                            </div>
                            <Text type="secondary" style={{ fontSize: 11 }}>
                              天
                            </Text>
                          </>
                        ) : (
                          <Text type="secondary">--</Text>
                        )}
                      </div>
                    </Row>
                    <div style={{ marginTop: 8 }}>
                      <Tag
                        color={
                          cert.status === 'ok'
                            ? 'success'
                            : cert.status === 'warn'
                              ? 'warning'
                              : 'error'
                        }
                      >
                        {certStatusLabel(cert.status)}
                      </Tag>
                    </div>
                  </Card>
                </Tooltip>
              </Col>
            ))}
          </Row>
        </Card>
      )}
    </AppPage>
  )
}

export default K8sDashboardPage
