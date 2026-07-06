import React from 'react'
import { Area } from '@ant-design/charts'
import { useModel } from '@umijs/max'
import { useQuery } from '@tanstack/react-query'
import { Alert, Card, Col, Empty, List, Row, Space, Spin, Statistic, Tag, Typography } from 'antd'
import {
  AlertOutlined,
  CheckCircleOutlined,
  CloudServerOutlined,
  DashboardOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { getClusterOverview } from '@/services/k8s'

const { Text, Title } = Typography

const toneColorMap = {
  success: '#059669',
  warning: '#d97706',
  danger: '#dc2626',
  info: '#2563eb',
} as const

const ClusterDashboardPage: React.FC = () => {
  const { currentCluster } = useModel('cluster')

  const { data, isLoading } = useQuery({
    queryKey: ['cluster-dashboard', currentCluster?.id],
    queryFn: () => getClusterOverview(Number(currentCluster!.id)),
    enabled: !!currentCluster,
    // 与后端缓存对齐，避免 120s 内重复请求聚合接口
    staleTime: 120_000,
  })

  if (!currentCluster) {
    return <Empty description="请先在顶部选择集群" />
  }

  if (isLoading || !data) {
    return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />
  }

  const { stats, charts, anomalies, risks } = data
  const healthLabel = stats.nodes.ready === stats.nodes.total ? '健康' : '异常'
  const healthTone: 'success' | 'warning' | 'danger' = stats.nodes.ready === stats.nodes.total ? 'success' : stats.nodes.ready === 0 ? 'danger' : 'warning'
  // 后端 cluster.status 为小写（active/degraded），需大小写不敏感判断
  const clusterHealthy = String(data.cluster.status || '').trim().toLowerCase() === 'active'

  const abnormalPods = anomalies?.failed_pods || []
  const certAlerts = risks?.certificates?.filter((c) => c.status !== 'ok') || []

  const areaConfig = {
    data: [
      ...(charts.cpu_memory_24h.labels.map((label, i) => ({
        time: label,
        value: charts.cpu_memory_24h.cpu[i] || 0,
        type: 'CPU',
      }))),
      ...(charts.cpu_memory_24h.labels.map((label, i) => ({
        time: label,
        value: charts.cpu_memory_24h.memory[i] || 0,
        type: '内存',
      }))),
    ],
    xField: 'time',
    yField: 'value',
    colorField: 'type',
    smooth: true,
    height: 300,
    color: ['#2563eb', '#059669'],
    areaStyle: () => ({ fillOpacity: 0.15 }),
    axis: {
      y: { labelFormatter: (value: number) => `${value}%` },
    },
    legend: {
      position: 'top' as const,
    },
    tooltip: {
      channel: 'y' as const,
    },
  }

  return (
    <div className="cluster-dashboard-page">
      <Space direction="vertical" size={20} style={{ width: '100%' }}>
        {/* ---- 集群信息头 ---- */}
        <Card bordered={false} style={{ borderRadius: 8 }}>
          <Space direction="vertical" size={4}>
            <Title level={4} style={{ margin: 0 }}>
              <CloudServerOutlined style={{ marginRight: 8 }} />
              {data.cluster.name} 集群详情
            </Title>
            <Space wrap>
              <Tag color={clusterHealthy ? 'success' : 'error'}>
                {clusterHealthy ? '健康' : '异常'}
              </Tag>
              <Text type="secondary">版本：{data.cluster.k8s_version || '-'}</Text>
              <Text type="secondary">集群 ID：{currentCluster.id}</Text>
            </Space>
          </Space>
        </Card>

        {/* ---- 顶部概览卡片行 ---- */}
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ borderRadius: 8 }}>
              <Statistic
                title="集群健康状态"
                value={healthLabel}
                prefix={<CheckCircleOutlined />}
                valueStyle={{ color: toneColorMap[healthTone], fontSize: 28 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ borderRadius: 8 }}>
              <Statistic
                title="节点就绪数"
                value={stats.nodes.ready}
                suffix={`/ ${stats.nodes.total}`}
                prefix={<CloudServerOutlined />}
                valueStyle={{ fontSize: 28 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ borderRadius: 8 }}>
              <Statistic
                title="Pending Pods"
                value={stats.pods.pending}
                prefix={<DashboardOutlined />}
                valueStyle={{ color: stats.pods.pending > 0 ? '#d97706' : '#059669', fontSize: 28 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ borderRadius: 8 }}>
              <Statistic
                title="告警数"
                value={certAlerts.length}
                prefix={<AlertOutlined />}
                valueStyle={{ color: certAlerts.length > 0 ? '#dc2626' : '#059669', fontSize: 28 }}
              />
            </Card>
          </Col>
        </Row>

        {/* ---- 中间核心指标卡 ---- */}
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} className="app-metric-card" style={{ borderRadius: 8 }}>
              <div className="app-metric-card__label">Pod 总数</div>
              <div className="app-metric-card__value" style={{ color: toneColorMap.success, fontSize: 32, fontWeight: 700 }}>
                {stats.pods.total}
              </div>
              <div className="app-metric-card__meta" style={{ color: '#8c8c8c', fontSize: 12, marginTop: 4 }}>
                运行中 {stats.pods.running} · 失败 {stats.pods.failed}
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} className="app-metric-card" style={{ borderRadius: 8 }}>
              <div className="app-metric-card__label">CPU 使用率</div>
              <div className="app-metric-card__value" style={{ color: stats.cpu.used_percent > 80 ? toneColorMap.danger : stats.cpu.used_percent > 60 ? toneColorMap.warning : toneColorMap.success, fontSize: 32, fontWeight: 700 }}>
                {stats.cpu.used_percent}%
              </div>
              <div className="app-metric-card__meta" style={{ color: '#8c8c8c', fontSize: 12, marginTop: 4 }}>
                集群级 CPU 使用率
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} className="app-metric-card" style={{ borderRadius: 8 }}>
              <div className="app-metric-card__label">内存使用率</div>
              <div className="app-metric-card__value" style={{ color: stats.memory.used_percent > 80 ? toneColorMap.danger : stats.memory.used_percent > 60 ? toneColorMap.warning : toneColorMap.success, fontSize: 32, fontWeight: 700 }}>
                {stats.memory.used_percent}%
              </div>
              <div className="app-metric-card__meta" style={{ color: '#8c8c8c', fontSize: 12, marginTop: 4 }}>
                集群级内存使用率
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} className="app-metric-card" style={{ borderRadius: 8 }}>
              <div className="app-metric-card__label">工作负载</div>
              <div className="app-metric-card__value" style={{ color: toneColorMap.info, fontSize: 32, fontWeight: 700 }}>
                {stats.workloads.deployments + stats.workloads.statefulsets + stats.workloads.daemonsets}
              </div>
              <div className="app-metric-card__meta" style={{ color: '#8c8c8c', fontSize: 12, marginTop: 4 }}>
                Deploy {stats.workloads.deployments} · STS {stats.workloads.statefulsets} · DS {stats.workloads.daemonsets}
              </div>
            </Card>
          </Col>
        </Row>

        {/* ---- 图表 + 异常列表 ---- */}
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={15}>
            <Card
              title={
                <Space>
                  <ThunderboltOutlined />
                  <span>CPU / 内存 24h 趋势</span>
                </Space>
              }
              bordered={false}
              style={{ borderRadius: 8 }}
            >
              <Area {...areaConfig} />
            </Card>
          </Col>
          <Col xs={24} lg={9}>
            <Card
              title={
                <Space>
                  <WarningOutlined />
                  <span>异常 Pods Top 5</span>
                </Space>
              }
              bordered={false}
              style={{ borderRadius: 8, height: '100%' }}
            >
              {abnormalPods.length ? (
                <List
                  dataSource={abnormalPods.slice(0, 5)}
                  renderItem={(pod) => (
                    <List.Item style={{ paddingLeft: 0, paddingRight: 0 }}>
                      <List.Item.Meta
                        avatar={
                          <Tag color="error" style={{ marginRight: 0 }}>
                            异常
                          </Tag>
                        }
                        title={<Text strong>{pod.name}</Text>}
                        description={
                          <Space size={8}>
                            <Text type="secondary">{pod.namespace}</Text>
                            <Tag>{pod.reason}</Tag>
                          </Space>
                        }
                      />
                    </List.Item>
                  )}
                />
              ) : (
                <Alert type="success" showIcon message="当前无异常 Pod" style={{ marginTop: 16 }} />
              )}
            </Card>
          </Col>
        </Row>
      </Space>
    </div>
  )
}

export default ClusterDashboardPage