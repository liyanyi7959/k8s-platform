import React, { useEffect, useMemo, useState, lazy, Suspense } from 'react'
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
  List,
} from 'antd'
import {
  CloudServerOutlined,
  AppstoreOutlined,
  NodeIndexOutlined,
  ReloadOutlined,
  WarningOutlined,
  LockOutlined,
  CloseCircleFilled,
  DashboardOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { getClusterOverview } from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'
import { AppPage, EllipsisText } from '@/components'
import { ResourceTrendChart } from '@/components/ResourceTrendChart'
import { history } from '@umijs/max'
import dayjs from 'dayjs'

const LazyLine: React.FC<{
  data: Array<{ time: string; sampledAt?: string; type: string; value: number | null }>
  [key: string]: unknown
}> = ({ data }) => {
  const labels = Array.from(new Set(data.map((point) => point.time)))
  const points = labels.map((time) => ({
    time,
    sampledAt: data.find((point) => point.time === time)?.sampledAt,
    cpu: data.find((point) => point.time === time && point.type === 'CPU')?.value ?? null,
    memory: data.find((point) => point.time === time && point.type !== 'CPU')?.value ?? null,
  }))
  return (
    <ResourceTrendChart
      points={points}
      empty={points.every((point) => point.cpu == null && point.memory == null)}
      height={240}
    />
  )
}

// 懒加载重型图表组件，减少首屏 JS 体积
const LazyPie = lazy(() => import('@ant-design/charts').then((m) => ({ default: m.Pie })))
const LazyColumn = lazy(() => import('@ant-design/charts').then((m) => ({ default: m.Column })))

const { Text } = Typography

const COLOR = {
  healthy: '#52c41a',
  warning: '#faad14',
  critical: '#ff4d4f',
  idle: '#bfbfbf',
  primary: '#1890ff',
  purple: '#7c3aed',
  cyan: '#0891b2',
} as const

function getHealthTier(score: number): 'good' | 'warn' | 'bad' {
  return score >= 80 ? 'good' : score >= 60 ? 'warn' : 'bad'
}

function getHealthColor(tier: 'good' | 'warn' | 'bad'): string {
  return tier === 'good' ? COLOR.healthy : tier === 'warn' ? COLOR.warning : COLOR.critical
}

function timeAgo(ts: string): string {
  if (!ts) return ''
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
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [refreshInterval, setRefreshInterval] = useState(30)
  const [selectedMetricNode, setSelectedMetricNode] = useState('__cluster__')

  useEffect(() => {
    setSelectedMetricNode('__cluster__')
  }, [clusterId])

  const {
    data: overview,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ['k8s-overview', clusterId],
    queryFn: ({ signal }) => getClusterOverview(clusterId, signal),
    refetchInterval: autoRefresh ? refreshInterval * 1000 : false,
  })

  const healthScore = useMemo(() => {
    if (!overview) return 0
    const s = overview.stats
    const nodeScore = s.nodes.total > 0 ? Math.round((s.nodes.ready / s.nodes.total) * 100) : 0
    const podTotal = Math.max(1, s.pods.total)
    const podScore = Math.max(0, Math.min(100, Math.round((s.pods.running / podTotal) * 100)))
    const pressureScore =
      overview.meta?.metrics_available === false
        ? 80
        : Math.round(
            (Math.max(0, 100 - s.cpu.used_percent) + Math.max(0, 100 - s.memory.used_percent)) / 2,
          )
    return Math.round(nodeScore * 0.3 + podScore * 0.3 + pressureScore * 0.2 + 80 * 0.2)
  }, [overview])

  const healthBreakdown = useMemo(() => {
    if (!overview) return []
    const s = overview.stats
    const nodeScore = s.nodes.total > 0 ? Math.round((s.nodes.ready / s.nodes.total) * 100) : 0
    const podTotal = Math.max(1, s.pods.total)
    const podScore = Math.max(0, Math.min(100, Math.round((s.pods.running / podTotal) * 100)))
    const pressureScore =
      overview.meta?.metrics_available === false
        ? 80
        : Math.max(0, Math.round(100 - (s.cpu.used_percent + s.memory.used_percent) / 2))
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
        hint:
          overview.meta?.metrics_available === false
            ? '指标服务不可用，暂按中性分计算'
            : `CPU ${s.cpu.used_percent}% / Mem ${s.memory.used_percent}%`,
      },
    ].map((item) => ({ ...item, tier: getHealthTier(item.score) }))
  }, [overview])

  const warningEvents = useMemo(() => {
    if (!overview?.events) return []
    return overview.events.slice(0, 10)
  }, [overview])

  const failedPods = overview?.anomalies?.failed_pods || []
  const unscheduledPods = overview?.anomalies?.unscheduled_pods || []
  const topWorkloads = overview?.top_workloads || []
  const certRisks = (overview?.risks?.certificates || []).filter((cert) => cert.status !== 'ok')

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

  const s = overview.stats
  const nodeTrends = overview.charts.node_cpu_memory_24h || []
  const selectedNodeTrend = nodeTrends.find((trend) => trend.name === selectedMetricNode)
  const activeTrend = selectedNodeTrend || overview.charts.cpu_memory_24h
  const activeMetricNode = selectedNodeTrend?.name
  const trendLabels = activeTrend.labels
  const cpuMemData = trendLabels.flatMap((label, i) => [
    {
      time: label,
      sampledAt: activeTrend.timestamps?.[i],
      type: 'CPU',
      value: activeTrend.cpu[i] ?? null,
    },
    {
      time: label,
      sampledAt: activeTrend.timestamps?.[i],
      type: '内存',
      value: activeTrend.memory[i] ?? null,
    },
  ])
  const metricNodeOptions = [
    { label: '全部节点（聚合）', value: '__cluster__' },
    ...nodeTrends.map((trend) => ({
      label: trend.ip ? `${trend.name}（${trend.ip}）` : trend.name,
      value: trend.name,
    })),
  ]
  const podPhaseData = [
    { type: '运行中', value: s.pods.running },
    { type: '等待中', value: s.pods.pending },
    { type: '失败', value: s.pods.failed },
    { type: '完成', value: s.pods.succeeded },
  ].filter((d) => d.value > 0)
  const nodeReadyPercent =
    overview.charts.node_ready.total > 0
      ? Math.round((overview.charts.node_ready.ready / overview.charts.node_ready.total) * 100)
      : 0

  return (
    <AppPage>
      {/* ═══ Zone 1: 核心集群总览 ═══ */}
      <Card
        size="small"
        style={{ marginBottom: 12 }}
        title={
          <Space>
            <DashboardOutlined style={{ color: COLOR.primary }} />
            <span>集群总览</span>
            <Tag color={overview.cluster.api_ok ? 'success' : 'error'}>
              {overview.cluster.api_ok ? 'API 正常' : 'API 异常'}
            </Tag>
            {overview.cluster.k8s_version && <Tag>{overview.cluster.k8s_version}</Tag>}
          </Space>
        }
        extra={
          <Space>
            {overview.meta?.updated_at && (
              <Tooltip
                title={`来源: ${overview.meta.source || 'k8s-api'} | 更新: ${dayjs(overview.meta.updated_at).format('HH:mm:ss')}`}
              >
                <Text type="secondary" style={{ fontSize: 11 }}>
                  <ClockCircleOutlined /> {timeAgo(overview.meta.updated_at)}
                </Text>
              </Tooltip>
            )}
            <Switch
              checkedChildren="自动"
              unCheckedChildren="手动"
              checked={autoRefresh}
              onChange={setAutoRefresh}
              size="small"
            />
            {autoRefresh && (
              <Select
                size="small"
                value={refreshInterval}
                onChange={setRefreshInterval}
                style={{ width: 65 }}
                options={[
                  { label: '10s', value: 10 },
                  { label: '30s', value: 30 },
                  { label: '60s', value: 60 },
                ]}
              />
            )}
            <Button size="small" icon={<ReloadOutlined />} onClick={() => refetch()} />
          </Space>
        }
      >
        <Row gutter={[12, 12]}>
          {[
            {
              label: '节点',
              value: `${s.nodes.ready}/${s.nodes.total}`,
              sub: 'Ready / Total',
              icon: <NodeIndexOutlined />,
              color: COLOR.primary,
              path: `/k8s/${clusterId}/nodes`,
            },
            {
              label: 'Pod',
              value: s.pods.total,
              sub: `${s.pods.running} 运行 / ${s.pods.pending} 等待 / ${s.pods.failed} 异常`,
              icon: <AppstoreOutlined />,
              color: s.pods.failed > 0 ? COLOR.critical : COLOR.healthy,
              path: `/k8s/${clusterId}/pods`,
            },
            {
              label: '工作负载',
              value: s.workloads.deployments + s.workloads.statefulsets + s.workloads.daemonsets,
              sub: `Deploy ${s.workloads.deployments} / STS ${s.workloads.statefulsets} / DS ${s.workloads.daemonsets}`,
              icon: <CloudServerOutlined />,
              color: COLOR.purple,
              path: `/k8s/${clusterId}/deployments`,
            },
            {
              label: 'CPU 使用率',
              value: overview.meta?.metrics_available === false ? '--' : `${s.cpu.used_percent}%`,
              sub:
                overview.meta?.metrics_available === false
                  ? 'metrics-server 未就绪'
                  : `内存 ${s.memory.used_percent}%`,
              icon: <DashboardOutlined />,
              color:
                overview.meta?.metrics_available === false
                  ? COLOR.idle
                  : s.cpu.used_percent >= 80
                    ? COLOR.critical
                    : s.cpu.used_percent >= 70
                      ? COLOR.warning
                      : COLOR.healthy,
              path: `/k8s/${clusterId}/resource-metrics`,
            },
          ].map((kpi) => (
            <Col xs={12} sm={6} key={kpi.label}>
              <Card
                hoverable
                size="small"
                style={{
                  borderLeft: `3px solid ${kpi.color}`,
                  cursor: kpi.path ? 'pointer' : 'default',
                }}
                onClick={() => kpi.path && history.push(kpi.path)}
              >
                <Space>
                  <div
                    style={{
                      width: 38,
                      height: 38,
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
                    <div style={{ fontSize: 22, fontWeight: 900, lineHeight: 1.1 }}>
                      {kpi.value}
                    </div>
                    <Text type="secondary" style={{ fontSize: 10 }}>
                      {kpi.sub}
                    </Text>
                  </div>
                </Space>
              </Card>
            </Col>
          ))}
        </Row>
      </Card>

      {/* ═══ Zone 2: 健康度 + Pod 分布 ═══ */}
      <Row gutter={12} style={{ marginBottom: 12 }}>
        <Col xs={24} md={10}>
          <Card
            title={
              <Space>
                <DashboardOutlined style={{ color: COLOR.primary }} /> 集群健康度
              </Space>
            }
            size="small"
            style={{ height: '100%' }}
          >
            <Row align="middle" gutter={16}>
              <Col span={10} style={{ textAlign: 'center' }}>
                <div style={{ position: 'relative', display: 'inline-block' }}>
                  <Progress
                    type="dashboard"
                    percent={healthScore}
                    size={140}
                    strokeColor={getHealthColor(getHealthTier(healthScore))}
                    format={() => (
                      <div>
                        <div style={{ fontSize: 28, fontWeight: 900 }}>{healthScore}</div>
                        <div style={{ fontSize: 10 }}>健康分</div>
                      </div>
                    )}
                  />
                </div>
              </Col>
              <Col span={14}>
                {healthBreakdown.map((item) => (
                  <div key={item.key} style={{ marginBottom: 10 }}>
                    <Row justify="space-between" style={{ marginBottom: 2 }}>
                      <Text strong style={{ fontSize: 12 }}>
                        {item.label}
                      </Text>
                      <Text
                        style={{ color: getHealthColor(item.tier), fontSize: 12, fontWeight: 600 }}
                      >
                        {item.score}分
                      </Text>
                    </Row>
                    <Progress
                      percent={item.score}
                      size="small"
                      strokeColor={getHealthColor(item.tier)}
                      showInfo={false}
                    />
                    <Text type="secondary" style={{ fontSize: 10 }}>
                      {item.hint}
                    </Text>
                  </div>
                ))}
              </Col>
            </Row>
          </Card>
        </Col>
        <Col xs={24} md={7}>
          <Card
            title={
              <Space>
                <AppstoreOutlined style={{ color: COLOR.purple }} /> Pod 状态分布
              </Space>
            }
            size="small"
            style={{ height: '100%' }}
          >
            {podPhaseData.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无 Pod" />
            ) : (
              <>
                <Suspense fallback={<Spin style={{ display: 'block', margin: '50px auto' }} />}>
                  <LazyPie
                    height={140}
                    data={podPhaseData}
                    angleField="value"
                    colorField="type"
                    scale={{
                      color: {
                        domain: ['运行中', '等待中', '失败', '完成'],
                        range: [COLOR.healthy, COLOR.warning, COLOR.critical, COLOR.idle],
                      },
                    }}
                    innerRadius={0.55}
                    legend={false}
                    label={{ text: 'value', position: 'outside', style: { fontSize: 11 } }}
                    tooltip={{ title: 'type', items: [{ channel: 'y', name: 'Pod 数量' }] }}
                  />
                </Suspense>
                <Row justify="center" gutter={16}>
                  {[
                    { label: '运行', value: s.pods.running, color: COLOR.healthy },
                    { label: '等待', value: s.pods.pending, color: COLOR.warning },
                    { label: '失败', value: s.pods.failed, color: COLOR.critical },
                    { label: '完成', value: s.pods.succeeded, color: COLOR.idle },
                  ].map((p) => (
                    <Space key={p.label} size={4}>
                      <span
                        style={{
                          width: 8,
                          height: 8,
                          borderRadius: '50%',
                          background: p.color,
                          display: 'inline-block',
                        }}
                      />
                      <Text style={{ fontSize: 11 }}>
                        {p.label} {p.value}
                      </Text>
                    </Space>
                  ))}
                </Row>
              </>
            )}
          </Card>
        </Col>
        <Col xs={24} md={7}>
          <Card
            title={
              <Space>
                <NodeIndexOutlined style={{ color: COLOR.primary }} /> 节点就绪
              </Space>
            }
            size="small"
            style={{ height: '100%' }}
          >
            <div style={{ textAlign: 'center' }}>
              <Progress
                type="circle"
                percent={nodeReadyPercent}
                size={140}
                strokeColor={
                  nodeReadyPercent === 100
                    ? COLOR.healthy
                    : nodeReadyPercent >= 50
                      ? COLOR.warning
                      : COLOR.critical
                }
                format={() => (
                  <div>
                    <div style={{ fontSize: 28, fontWeight: 900 }}>
                      {overview.charts.node_ready.ready}/{overview.charts.node_ready.total}
                    </div>
                    <div style={{ fontSize: 10 }}>Ready 节点</div>
                  </div>
                )}
              />
            </div>
          </Card>
        </Col>
      </Row>

      {/* ═══ Zone 3: 实时风险预警 + 证书 ═══ */}
      <Row gutter={12} style={{ marginBottom: 12 }}>
        <Col xs={24} md={8}>
          <Card
            title={
              <Space>
                <CloseCircleFilled style={{ color: COLOR.critical }} /> 异常 Pod{' '}
                <Badge count={failedPods.length} style={{ backgroundColor: COLOR.critical }} />
              </Space>
            }
            size="small"
            style={{ height: '100%' }}
            bodyStyle={{ maxHeight: 200, overflow: 'auto' }}
          >
            {failedPods.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无异常" />
            ) : (
              <List
                size="small"
                dataSource={failedPods}
                renderItem={(item) => (
                  <List.Item
                    style={{ padding: '4px 0', cursor: 'pointer' }}
                    onClick={() => history.push(`/k8s/${clusterId}/pods`)}
                  >
                    <Space direction="vertical" size={0} style={{ width: '100%' }}>
                      <Row justify="space-between">
                        <EllipsisText text={item.name} />
                        <Tag color="error" style={{ fontSize: 10 }}>
                          {item.reason}
                        </Tag>
                      </Row>
                      <Text type="secondary" style={{ fontSize: 10 }}>
                        {item.namespace}
                      </Text>
                    </Space>
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card
            title={
              <Space>
                <WarningOutlined style={{ color: COLOR.warning }} /> 未调度 Pod{' '}
                <Badge count={unscheduledPods.length} style={{ backgroundColor: COLOR.warning }} />
              </Space>
            }
            size="small"
            style={{ height: '100%' }}
            bodyStyle={{ maxHeight: 200, overflow: 'auto' }}
          >
            {unscheduledPods.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无未调度" />
            ) : (
              <List
                size="small"
                dataSource={unscheduledPods}
                renderItem={(item) => (
                  <List.Item
                    style={{ padding: '4px 0', cursor: 'pointer' }}
                    onClick={() => history.push(`/k8s/${clusterId}/pods`)}
                  >
                    <Space direction="vertical" size={0} style={{ width: '100%' }}>
                      <EllipsisText text={item.name} />
                      <Text type="secondary" style={{ fontSize: 10 }}>
                        {item.namespace}
                      </Text>
                    </Space>
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card
            title={
              <Space>
                <LockOutlined
                  style={{ color: certRisks.length > 0 ? COLOR.warning : COLOR.healthy }}
                />{' '}
                证书过期风险{' '}
                <Badge count={certRisks.length} style={{ backgroundColor: COLOR.warning }} />
              </Space>
            }
            size="small"
            style={{ height: '100%' }}
            bodyStyle={{ maxHeight: 200, overflow: 'auto' }}
          >
            {certRisks.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无证书风险" />
            ) : (
              <List
                size="small"
                dataSource={certRisks}
                renderItem={(cert) => (
                  <List.Item style={{ padding: '6px 0' }}>
                    <div style={{ width: '100%' }}>
                      <Row justify="space-between" align="middle">
                        <Text strong style={{ fontSize: 12 }}>
                          {cert.name}
                        </Text>
                        <Tag
                          color={
                            cert.status === 'critical'
                              ? 'error'
                              : cert.status === 'warn'
                                ? 'warning'
                                : 'default'
                          }
                          style={{ fontSize: 10 }}
                        >
                          {cert.days_left != null ? `${cert.days_left}天` : cert.status}
                        </Tag>
                      </Row>
                      <Text type="secondary" style={{ fontSize: 10 }}>
                        {cert.component} · {cert.purpose}
                      </Text>
                    </div>
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>
      </Row>

      {/* ═══ Zone 4: CPU/内存趋势（全宽） ═══ */}
      <Row gutter={12} style={{ marginBottom: 12 }}>
        <Col span={24}>
          <Card
            title={
              <Space>
                <DashboardOutlined style={{ color: COLOR.cyan }} />
                CPU / 内存 24h 趋势
                <Tag color="blue" style={{ fontSize: 10 }}>
                  {activeMetricNode ? `Node：${activeMetricNode}` : '全部节点聚合'}
                </Tag>
                {!overview.cluster.api_ok && (
                  <Tag color="error" style={{ fontSize: 10 }}>
                    集群 API 不可达
                  </Tag>
                )}
                {overview.cluster.api_ok && overview.meta?.metrics_available === false && (
                  <Tag color="warning" style={{ fontSize: 10 }}>
                    metrics-server 不可用
                  </Tag>
                )}
              </Space>
            }
            extra={
              <Select
                showSearch
                value={activeMetricNode || '__cluster__'}
                onChange={setSelectedMetricNode}
                optionFilterProp="label"
                options={metricNodeOptions}
                placeholder="按 Node 名称或 IP 搜索"
                style={{ width: 300 }}
                popupMatchSelectWidth={false}
              />
            }
            size="small"
          >
            <Text type="secondary" style={{ display: 'block', fontSize: 11, marginBottom: 8 }}>
              {activeMetricNode
                ? `口径：Node ${activeMetricNode} 的 metrics.k8s.io usage ÷ 该 Node allocatable`
                : '口径：metrics.k8s.io 本次成功采样 Node 的 usage 总和 ÷ 同批 Node allocatable 总和'}
              ；仅展示最近 24 小时真实采样，共 {activeTrend.sample_count ?? trendLabels.length}{' '}
              个点。
            </Text>
            {overview.cluster.api_ok && cpuMemData.length > 0 ? (
              <Suspense fallback={<Spin style={{ display: 'block', margin: '80px auto' }} />}>
                <div style={{ position: 'relative' }}>
                  <LazyLine
                    data={cpuMemData}
                    encode={{ x: 'time', y: 'value', color: 'type' }}
                    color={[COLOR.cyan, COLOR.purple]}
                    smooth
                    height={240}
                    point={{ size: 2 }}
                    xAxis={{ label: { autoRotate: true, style: { fontSize: 10 } } }}
                    yAxis={{ min: 0, max: 100, label: { formatter: (v: number) => `${v}%` } }}
                    legend={{ position: 'topRight' }}
                    annotations={[
                      {
                        type: 'line',
                        yField: 80,
                        style: { stroke: COLOR.warning, lineDash: [4, 4] },
                        label: {
                          text: '80%',
                          position: 'end',
                          style: { fill: COLOR.warning, fontSize: 10 },
                        },
                      },
                      {
                        type: 'line',
                        yField: 90,
                        style: { stroke: COLOR.critical, lineDash: [4, 4] },
                        label: {
                          text: '90%',
                          position: 'end',
                          style: { fill: COLOR.critical, fontSize: 10 },
                        },
                      },
                    ]}
                  />
                  {overview.meta?.metrics_available === false && (
                    <Text
                      type="secondary"
                      style={{
                        position: 'absolute',
                        inset: 0,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        pointerEvents: 'none',
                      }}
                    >
                      暂无指标采样数据，坐标轴为参考范围
                    </Text>
                  )}
                </div>
              </Suspense>
            ) : (
              <div
                style={{
                  height: 240,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 8,
                }}
              >
                <DashboardOutlined style={{ fontSize: 36, color: COLOR.idle }} />
                <Text type="secondary" style={{ fontSize: 13 }}>
                  {!overview.cluster.api_ok ? '集群 API 不可达，无法获取监控数据' : '暂无监控数据'}
                </Text>
                <Text type="secondary" style={{ fontSize: 11 }}>
                  {!overview.cluster.api_ok
                    ? '请检查集群连接状态或 kubeconfig 配置'
                    : 'metrics.k8s.io 指标服务可能未部署或暂无数据'}
                </Text>
                <Button
                  size="small"
                  icon={<ReloadOutlined />}
                  onClick={() => refetch()}
                  style={{ marginTop: 4 }}
                >
                  重新获取
                </Button>
                {overview.cluster.api_ok && overview.meta?.metrics_available === false && (
                  <Button
                    size="small"
                    type="link"
                    onClick={() => history.push(`/k8s/${clusterId}/resource-metrics`)}
                  >
                    查看指标服务诊断
                  </Button>
                )}
              </div>
            )}
          </Card>
        </Col>
      </Row>

      {/* ═══ Zone 4.5: 近期告警事件（全宽，每条独占一行） ═══ */}
      <Card
        title={
          <Space>
            <WarningOutlined
              style={{ color: warningEvents.length > 0 ? COLOR.critical : COLOR.idle }}
            />
            <span>近期告警事件</span>
            {warningEvents.length > 0 && (
              <Badge count={warningEvents.length} style={{ backgroundColor: COLOR.critical }} />
            )}
          </Space>
        }
        size="small"
        style={{ marginBottom: 12 }}
        bodyStyle={{ padding: 0 }}
      >
        {warningEvents.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="集群运行正常，无告警事件"
            style={{ margin: '40px 0' }}
          />
        ) : (
          <Table
            dataSource={warningEvents}
            rowKey={(_, idx) => String(idx)}
            pagination={false}
            size="small"
            scroll={{ y: 320 }}
            style={{ wordBreak: 'break-word' }}
            columns={[
              {
                title: '级别',
                dataIndex: 'reason',
                width: 140,
                render: (reason: string) => {
                  const isError =
                    reason === 'Unhealthy' || reason === 'CrashLoopBackOff' || reason === 'Failed'
                  return (
                    <Space size={4}>
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          background: isError ? COLOR.critical : COLOR.warning,
                          flexShrink: 0,
                          boxShadow: `0 0 4px ${isError ? COLOR.critical : COLOR.warning}`,
                        }}
                      />
                      <Tag
                        color={isError ? 'error' : 'warning'}
                        style={{
                          margin: 0,
                          fontSize: 11,
                          whiteSpace: 'normal',
                          lineHeight: '18px',
                        }}
                      >
                        {reason}
                      </Tag>
                    </Space>
                  )
                },
              },
              {
                title: '事件详情',
                dataIndex: 'message',
                render: (msg: string) => (
                  <Text
                    style={{
                      fontSize: 13,
                      wordBreak: 'break-word',
                      overflowWrap: 'break-word',
                      whiteSpace: 'normal',
                    }}
                  >
                    {msg}
                  </Text>
                ),
              },
              {
                title: '关联资源',
                dataIndex: 'involvedObject',
                width: 180,
                render: (obj: string) =>
                  obj ? (
                    <Tooltip title={obj}>
                      <Text
                        type="secondary"
                        style={{
                          fontSize: 12,
                          display: 'block',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {obj}
                      </Text>
                    </Tooltip>
                  ) : (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      -
                    </Text>
                  ),
              },
              {
                title: '时间',
                dataIndex: 'lastTimestamp',
                width: 80,
                render: (ts: string) => (
                  <Text type="secondary" style={{ fontSize: 11, whiteSpace: 'nowrap' }}>
                    {timeAgo(ts)}
                  </Text>
                ),
              },
            ]}
          />
        )}
      </Card>

      {/* ═══ Zone 5: 工作负载概览 ═══ */}
      <Row gutter={12}>
        <Col xs={24} lg={10}>
          <Card
            title={
              <Space>
                <AppstoreOutlined style={{ color: COLOR.purple }} /> Namespace Pod Top 10
              </Space>
            }
            size="small"
          >
            {overview.charts.namespace_pods_top.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="暂无数据"
                style={{ marginTop: 60 }}
              />
            ) : (
              <Suspense fallback={<Spin style={{ display: 'block', margin: '80px auto' }} />}>
                <LazyColumn
                  data={overview.charts.namespace_pods_top}
                  xField="namespace"
                  yField="pods"
                  height={220}
                  color={COLOR.purple}
                  columnStyle={{ radius: [4, 4, 0, 0] }}
                  xAxis={{ label: { autoRotate: true, style: { fontSize: 10 } } }}
                  yAxis={{ label: { style: { fontSize: 10 } } }}
                />
              </Suspense>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={14}>
          <Card
            title={
              <Space>
                <CloudServerOutlined style={{ color: COLOR.primary }} /> Top 5 工作负载快照
              </Space>
            }
            size="small"
            bodyStyle={{ padding: '8px 12px' }}
          >
            {topWorkloads.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无数据" />
            ) : (
              <Table
                dataSource={topWorkloads}
                rowKey={(r) => `${r.namespace}/${r.name}`}
                pagination={false}
                size="small"
                onRow={(record) => ({
                  style: { cursor: 'pointer' },
                  onClick: () =>
                    history.push(
                      `/k8s/${clusterId}/${record.kind === 'StatefulSet' ? 'statefulsets' : record.kind === 'DaemonSet' ? 'daemonsets' : 'deployments'}`,
                    ),
                })}
                columns={[
                  {
                    title: '#',
                    width: 36,
                    render: (_, __, idx) => (
                      <Text strong style={{ color: COLOR.primary }}>
                        {idx + 1}
                      </Text>
                    ),
                  },
                  {
                    title: '名称',
                    dataIndex: 'name',
                    render: (v: string) => <EllipsisText text={v} />,
                  },
                  {
                    title: '类型',
                    dataIndex: 'kind',
                    width: 100,
                    render: (v: string) => <Tag style={{ fontSize: 10 }}>{v}</Tag>,
                  },
                  {
                    title: 'Namespace',
                    dataIndex: 'namespace',
                    width: 120,
                    render: (v: string) => <EllipsisText text={v} tag />,
                  },
                  {
                    title: '副本',
                    dataIndex: 'replicas',
                    width: 60,
                    render: (v: number, r: any) => (
                      <Text strong style={{ color: r.ready >= v ? COLOR.healthy : COLOR.warning }}>
                        {r.ready}/{v}
                      </Text>
                    ),
                  },
                ]}
              />
            )}
          </Card>
        </Col>
      </Row>
    </AppPage>
  )
}

export default K8sDashboardPage
