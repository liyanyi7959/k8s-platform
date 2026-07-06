import React from 'react'
import { history } from '@umijs/max'
import { useQuery } from '@tanstack/react-query'
import {
  Alert,
  Card,
  Col,
  Row,
  Skeleton,
  Space,
  Statistic,
  Typography,
  List,
  Tag,
  Button,
} from 'antd'
import {
  AlertOutlined,
  CloudServerOutlined,
  ClusterOutlined,
  DashboardOutlined,
  FileSearchOutlined,
  RocketOutlined,
  RobotOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import { listClusters } from '@/services/clusters'
import { fetchDashboardOverview } from '@/services/dashboard'
import type { DashboardTimeRange } from '@/types/dashboard'

const { Text, Title } = Typography

const DEFAULT_TIME_RANGE: DashboardTimeRange = '24h'

const OverviewPage: React.FC = () => {
  const { data: clusterListRes, isLoading: clustersLoading } = useQuery({
    queryKey: ['cluster-options'],
    queryFn: () => listClusters(),
  })

  const clusters = clusterListRes?.items || []

  const { data: dashboardData, isLoading: dashboardLoading } = useQuery({
    queryKey: ['dashboard-overview', DEFAULT_TIME_RANGE],
    queryFn: () => fetchDashboardOverview(DEFAULT_TIME_RANGE),
  })

  const healthyCount = clusters.filter((c) => c.status === 'healthy' || c.status === 'active').length
  const unhealthyCount = clusters.filter((c) => c.status !== 'healthy' && c.status !== 'active').length
  const totalNodes = clusters.reduce((sum, c) => sum + (c.nodeCount || 0), 0)

  return (
    <div className="overview-page">
      <Space direction="vertical" size={20} style={{ width: '100%' }}>
        {/* ---- 页面标题 ---- */}
        <div>
          <Title level={3} style={{ margin: 0 }}>
            <DashboardOutlined style={{ marginRight: 8 }} />
            全局概览
          </Title>
          <Text type="secondary">欢迎使用 AIOPS 智能运维平台，查看所有集群的运行状态</Text>
        </div>

        {/* ---- 全局统计卡片 ---- */}
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ borderRadius: 8 }}>
              <Skeleton loading={clustersLoading} active paragraph={false}>
                <Statistic
                  title="集群总数"
                  value={clusters.length}
                  prefix={<ClusterOutlined />}
                  valueStyle={{ fontSize: 28 }}
                />
                <Text type="secondary">
                  健康 {healthyCount} / 异常 {unhealthyCount}
                </Text>
              </Skeleton>
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ borderRadius: 8 }}>
              <Skeleton loading={clustersLoading} active paragraph={false}>
                <Statistic
                  title="节点总数"
                  value={totalNodes}
                  prefix={<CloudServerOutlined />}
                  valueStyle={{ fontSize: 28 }}
                />
              </Skeleton>
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ borderRadius: 8 }}>
              <Skeleton loading={dashboardLoading} active paragraph={false}>
                <Statistic
                  title="活跃告警"
                  value={dashboardData?.statistics?.find((s) => s.key === 'alerts')?.value ?? 0}
                  prefix={<AlertOutlined />}
                  valueStyle={{ fontSize: 28 }}
                />
              </Skeleton>
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card bordered={false} style={{ borderRadius: 8 }}>
              <Skeleton loading={dashboardLoading} active paragraph={false}>
                <Statistic
                  title="自动化任务"
                  value={dashboardData?.statistics?.find((s) => s.key === 'automation')?.value ?? 0}
                  prefix={<RocketOutlined />}
                  valueStyle={{ fontSize: 28 }}
                />
              </Skeleton>
            </Card>
          </Col>
        </Row>

        {/* ---- 集群列表 + 快捷入口 ---- */}
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={15}>
            <Card
              title={
                <Space>
                  <ClusterOutlined />
                  <span>集群列表</span>
                </Space>
              }
              bordered={false}
              style={{ borderRadius: 8 }}
              extra={
                <Button type="link" onClick={() => history.push('/clusters')}>
                  查看全部
                </Button>
              }
            >
              <Skeleton loading={clustersLoading} active paragraph={{ rows: 3 }}>
                {clusters.length ? (
                  <List
                    dataSource={clusters}
                    renderItem={(cluster) => (
                      <List.Item
                        style={{ paddingLeft: 0, paddingRight: 0, cursor: 'pointer' }}
                        onClick={() => history.push(`/cluster/dashboard?cluster=${cluster.id}`)}
                      >
                        <List.Item.Meta
                          avatar={
                            <Tag color={cluster.status === 'healthy' || cluster.status === 'active' ? 'success' : 'error'}>
                              {cluster.status === 'healthy' || cluster.status === 'active' ? '健康' : '异常'}
                            </Tag>
                          }
                          title={
                            <Space>
                              <Text strong>{cluster.name}</Text>
                              <Text type="secondary" style={{ fontSize: 12 }}>
                                {cluster.k8sVersion}
                              </Text>
                            </Space>
                          }
                          description={`${cluster.nodeCount} 个节点 · ID: ${cluster.id}`}
                        />
                      </List.Item>
                    )}
                  />
                ) : (
                  <Alert type="info" showIcon message="暂无集群数据" />
                )}
              </Skeleton>
            </Card>
          </Col>
          <Col xs={24} lg={9}>
            <Card
              title={
                <Space>
                  <ThunderboltOutlined />
                  <span>快捷入口</span>
                </Space>
              }
              bordered={false}
              style={{ borderRadius: 8 }}
            >
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Button
                  block
                  icon={<AlertOutlined />}
                  onClick={() => history.push('/monitor/dashboard')}
                >
                  告警中心
                </Button>
                <Button
                  block
                  icon={<FileSearchOutlined />}
                  onClick={() => history.push('/logs')}
                >
                  日志分析
                </Button>
                <Button
                  block
                  icon={<RobotOutlined />}
                  onClick={() => history.push('/ai/chat')}
                >
                  智能根因定位
                </Button>
                <Button
                  block
                  icon={<RocketOutlined />}
                  onClick={() => history.push('/deploy/plans')}
                >
                  自动化运维
                </Button>
              </Space>
            </Card>
          </Col>
        </Row>
      </Space>
    </div>
  )
}

export default OverviewPage