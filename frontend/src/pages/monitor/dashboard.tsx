import React from 'react'
import { Card, Row, Col, Statistic, Progress, List, Tag, Typography, Space, Tooltip } from 'antd'
import {
  AlertOutlined,
  CloudServerOutlined,
  DashboardOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { getMetricsOverview, listAlertEvents } from '@/services/monitor'
import { formatDate } from '@/utils'
import type { ClusterMetric, AlertEvent } from '@/types'

const { Text } = Typography

/** 集群指标可视化卡片 */
const ClusterMetricCard: React.FC<{ cluster: ClusterMetric }> = ({ cluster }) => {
  const cpuPercent = Math.round(cluster.cpuUsage * 100)
  const memPercent = Math.round(cluster.memoryUsage * 100)
  const diskPercent = Math.round(cluster.diskUsage * 100)

  const getProgressColor = (percent: number) => {
    if (percent >= 90) return '#ff4d4f'
    if (percent >= 70) return '#faad14'
    return '#52c41a'
  }

  return (
    <Card size="small" title={cluster.clusterName} style={{ marginBottom: 16 }}>
      <Row gutter={[16, 12]}>
        <Col span={8}>
          <div style={{ textAlign: 'center' }}>
            <Progress
              type="dashboard"
              percent={cpuPercent}
              size={80}
              strokeColor={getProgressColor(cpuPercent)}
              format={(p) => `${p}%`}
            />
            <div style={{ marginTop: 4, fontSize: 12, color: '#666' }}>CPU</div>
          </div>
        </Col>
        <Col span={8}>
          <div style={{ textAlign: 'center' }}>
            <Progress
              type="dashboard"
              percent={memPercent}
              size={80}
              strokeColor={getProgressColor(memPercent)}
              format={(p) => `${p}%`}
            />
            <div style={{ marginTop: 4, fontSize: 12, color: '#666' }}>内存</div>
          </div>
        </Col>
        <Col span={8}>
          <div style={{ textAlign: 'center' }}>
            <Progress
              type="dashboard"
              percent={diskPercent}
              size={80}
              strokeColor={getProgressColor(diskPercent)}
              format={(p) => `${p}%`}
            />
            <div style={{ marginTop: 4, fontSize: 12, color: '#666' }}>磁盘</div>
          </div>
        </Col>
      </Row>
      <Row gutter={16} style={{ marginTop: 12 }}>
        <Col span={8}>
          <Statistic title="Pod" value={cluster.podCount} suffix={`/ ${cluster.podCapacity}`} />
        </Col>
        <Col span={8}>
          <Statistic title="节点" value={cluster.nodeCount} />
        </Col>
        <Col span={8}>
          <Statistic title="告警" value={cluster.alertCount} valueStyle={{ color: cluster.alertCount > 0 ? '#ff4d4f' : '#52c41a' }} />
        </Col>
      </Row>
    </Card>
  )
}

/** 资源使用条形图（纯 CSS 实现） */
const ResourceBar: React.FC<{ label: string; used: number; total: number; unit?: string }> = ({
  label,
  used,
  total,
  unit = '',
}) => {
  const percent = total > 0 ? Math.round((used / total) * 100) : 0
  const getBarColor = (p: number) => {
    if (p >= 90) return '#ff4d4f'
    if (p >= 70) return '#faad14'
    return '#1677ff'
  }

  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
        <Text>{label}</Text>
        <Text type="secondary">
          {used}
          {unit} / {total}
          {unit} ({percent}%)
        </Text>
      </div>
      <Progress
        percent={percent}
        strokeColor={getBarColor(percent)}
        showInfo={false}
        size="small"
      />
    </div>
  )
}

/** 监控仪表盘页 */
const MonitorDashboardPage: React.FC = () => {
  const { data: metricsOverview, isLoading: metricsLoading } = useQuery({
    queryKey: ['monitor-metrics'],
    queryFn: () => getMetricsOverview(),
  })

  const { data: recentAlerts, isLoading: alertsLoading } = useQuery({
    queryKey: ['recent-alerts'],
    queryFn: () => listAlertEvents(),
    select: (data) => data.slice(0, 5),
  })

  const clusters = metricsOverview?.clusterMetrics || []
  const totalNodes = clusters.reduce((sum, c) => sum + c.nodeCount, 0)
  const totalPods = clusters.reduce((sum, c) => sum + c.podCount, 0)
  const totalAlerts = clusters.reduce((sum, c) => sum + c.alertCount, 0)

  // 全局资源汇总
  const totalCPUCapacity = clusters.reduce((sum, c) => sum + c.nodeCount * 4, 0)
  const totalCPUUsed = clusters.reduce((sum, c) => sum + c.cpuUsage * c.nodeCount * 4, 0)
  const totalMemCapacity = clusters.reduce((sum, c) => sum + c.nodeCount * 16, 0)
  const totalMemUsed = clusters.reduce((sum, c) => sum + c.memoryUsage * c.nodeCount * 16, 0)
  const totalDiskCapacity = clusters.reduce((sum, c) => sum + c.nodeCount * 100, 0)
  const totalDiskUsed = clusters.reduce((sum, c) => sum + c.diskUsage * c.nodeCount * 100, 0)

  return (
    <AppPage>
      {/* 核心指标卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="集群总数"
              value={clusters.length}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总节点数"
              value={totalNodes}
              prefix={<DashboardOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总 Pod 数"
              value={totalPods}
              prefix={<AlertOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="触发中告警"
              value={totalAlerts}
              prefix={<WarningOutlined />}
              valueStyle={{ color: totalAlerts > 0 ? '#ff4d4f' : '#52c41a' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 全局资源概览 */}
      {clusters.length > 0 && (
        <Card title="全局资源概览" style={{ marginBottom: 16 }}>
          <Row gutter={24}>
            <Col span={8}>
              <ResourceBar label="CPU 使用率" used={Math.round(totalCPUUsed * 10) / 10} total={totalCPUCapacity} unit=" 核" />
            </Col>
            <Col span={8}>
              <ResourceBar label="内存使用率" used={Math.round(totalMemUsed * 10) / 10} total={totalMemCapacity} unit=" GB" />
            </Col>
            <Col span={8}>
              <ResourceBar label="磁盘使用率" used={Math.round(totalDiskUsed)} total={totalDiskCapacity} unit=" GB" />
            </Col>
          </Row>
        </Card>
      )}

      {/* 各集群指标 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={16}>
          <Card title="集群资源使用">
            {metricsLoading ? (
              <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
            ) : (
              <Row gutter={16}>
                {clusters.map((cluster) => (
                  <Col span={12} key={cluster.clusterId}>
                    <ClusterMetricCard cluster={cluster} />
                  </Col>
                ))}
              </Row>
            )}
          </Card>
        </Col>
        <Col span={8}>
          <Card title="最近告警">
            <List
              loading={alertsLoading}
              dataSource={recentAlerts || []}
              locale={{ emptyText: '暂无告警' }}
              renderItem={(item: AlertEvent) => (
                <List.Item>
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <Space>
                      {item.status === 'firing' ? (
                        <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                      ) : (
                        <CheckCircleOutlined style={{ color: '#52c41a' }} />
                      )}
                      <Text strong>{item.ruleName}</Text>
                      <Tag
                        color={
                          item.severity === 'critical'
                            ? 'red'
                            : item.severity === 'warning'
                              ? 'orange'
                              : 'blue'
                        }
                      >
                        {item.severity}
                      </Tag>
                    </Space>
                    <Tooltip title={item.message}>
                      <Text type="secondary" ellipsis style={{ fontSize: 12 }}>
                        {item.message}
                      </Text>
                    </Tooltip>
                    <Text type="secondary" style={{ fontSize: 11 }}>
                      {formatDate(item.startedAt)} · {item.clusterName}
                    </Text>
                  </Space>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>
    </AppPage>
  )
}

export default MonitorDashboardPage
