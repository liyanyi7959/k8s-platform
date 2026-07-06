import React from 'react'
import { Bubble } from '@ant-design/x'
import { ProCard } from '@ant-design/pro-components'
import {
  AlertOutlined,
  ClusterOutlined,
  RobotOutlined,
  RocketOutlined,
  ApartmentOutlined,
  LineChartOutlined,
  CodeOutlined,
} from '@ant-design/icons'
import { Alert, Col, Empty, Row, Skeleton, Tag, Tooltip, Typography } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { fetchDashboardOverview } from '@/services/dashboard'
import { listClusters } from '@/services/clusters'
import { formatDate } from '@/utils'
import type { Cluster } from '@/types'
import type { DashboardStatistic, DashboardTimeRange } from '@/types/dashboard'

const { Text } = Typography

const DEFAULT_TIME_RANGE: DashboardTimeRange = '24h'

// 统计卡片的图标与主题色配置
const statVisual: Record<
  string,
  { icon: React.ReactNode; color: string }
> = {
  clusters: { icon: <ClusterOutlined />, color: '#2563eb' },
  alerts: { icon: <AlertOutlined />, color: '#dc2626' },
  automation: { icon: <RocketOutlined />, color: '#059669' },
  rca: { icon: <RobotOutlined />, color: '#7c3aed' },
}

const fallbackStats: DashboardStatistic[] = [
  { key: 'clusters', title: '集群总数', value: 0, unit: '个', description: '当前纳管集群' },
  { key: 'alerts', title: '活跃告警', value: 0, unit: '条', description: '待处理事件' },
  { key: 'automation', title: '自动化任务', value: 0, unit: '次', description: '近周期执行' },
  { key: 'rca', title: '根因定位', value: 0, unit: '次', description: 'AI 分析记录' },
]

// 集群状态判断（与集群列表页 StatusTag 保持一致）
const HEALTHY_STATUSES = new Set(['active', 'healthy', 'connected', 'success'])
const DEGRADED_STATUSES = new Set(['degraded', 'warning'])

function normalizeStatus(status?: string) {
  return String(status || '').trim().toLowerCase()
}

function getClusterStatusDisplay(status?: string): { color: string; text: string } {
  const normalized = normalizeStatus(status)
  if (!normalized || normalized === 'unknown') {
    return { color: 'default', text: '未知' }
  }
  if (HEALTHY_STATUSES.has(normalized)) {
    return { color: 'success', text: '健康' }
  }
  // 降级状态（degraded/warning）显示黄色，不算严重异常
  if (DEGRADED_STATUSES.has(normalized)) {
    return { color: 'warning', text: '降级' }
  }
  return { color: 'error', text: '异常' }
}

const DashboardPage: React.FC = () => {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['dashboard-overview', DEFAULT_TIME_RANGE],
    queryFn: () => fetchDashboardOverview(DEFAULT_TIME_RANGE),
  })

  // 接入真实集群列表，使「集群总数」与「集群健康概览」展示真实数据
  const { data: clusterRes, isLoading: clustersLoading } = useQuery({
    queryKey: ['dashboard-clusters'],
    queryFn: ({ signal }) => listClusters({ page: 1, pageSize: 100 }, signal),
  })
  const clusters: Cluster[] = clusterRes?.items ?? []

  // 用真实集群数覆盖 mock 的集群总数
  const statistics = (data?.statistics ?? fallbackStats).map((item) =>
    item.key === 'clusters' ? { ...item, value: clusters.length } : item,
  )

  return (
    <AppPage breadcrumbRender={false}>
      <div className="app-dashboard">
        {isError ? (
          <Alert
            type="warning"
            showIcon
            message="仪表盘数据暂不可用"
            description="当前展示的是安全占位骨架，稍后可重新刷新数据。"
          />
        ) : null}

        {/* ========== 统计卡片区 ========== */}
        <Row gutter={[16, 16]}>
          {statistics.map((item) => {
            const visual = statVisual[item.key] ?? statVisual.clusters
            return (
              <Col key={item.key} xs={24} sm={12} xl={6}>
                <ProCard className="app-dashboard-stat-card" bordered>
                  <div className="app-dashboard-stat-card__inner">
                    <div
                      className="app-dashboard-stat-card__icon"
                      style={{ color: visual.color, backgroundColor: `${visual.color}14` }}
                    >
                      {visual.icon}
                    </div>
                    <div className="app-dashboard-stat-card__body">
                      <div className="app-dashboard-stat-card__label">{item.title}</div>
                      <Skeleton loading={isLoading} active paragraph={false}>
                        <div className="app-dashboard-stat-card__value">
                          {item.value}
                          {item.unit ? (
                            <span className="app-dashboard-stat-card__unit">{item.unit}</span>
                          ) : null}
                        </div>
                      </Skeleton>
                      <div className="app-dashboard-stat-card__desc">
                        {item.description ?? '暂无数据'}
                      </div>
                    </div>
                  </div>
                </ProCard>
              </Col>
            )
          })}
        </Row>

        {/* ========== 集群健康概览 + AI 助手 ========== */}
        <Row gutter={[16, 16]}>
          <Col xs={24} xl={16}>
            <ProCard
              className="app-dashboard-panel app-dashboard-panel--clusters"
              bordered
              title="集群健康概览"
              extra={
                clusters.length > 0 ? (
                  <Text type="secondary" style={{ fontSize: 13 }}>
                    共 {clusters.length} 个集群
                  </Text>
                ) : null
              }
            >
              {clustersLoading ? (
                <Skeleton active paragraph={{ rows: 3 }} />
              ) : clusters.length === 0 ? (
                <Empty description="暂无纳管集群" style={{ margin: '32px 0' }} />
              ) : (
                <div className="app-dashboard-cluster-grid">
                  {clusters.map((c) => {
                    const sd = getClusterStatusDisplay(c.status)
                    return (
                      <Tooltip key={c.id} title={`原始状态：${c.status || '（空）'} · 最近健康检查：${c.lastHealthAt ? formatDate(c.lastHealthAt) : '-'}`}>
                        <div
                          className={
                            'app-dashboard-cluster-item' +
                            (sd.color === 'error' ? ' app-dashboard-cluster-item--danger' : '')
                          }
                        >
                          <div className="app-dashboard-cluster-item__header">
                            <ClusterOutlined className="app-dashboard-cluster-item__icon" />
                            <span className="app-dashboard-cluster-item__name">{c.name}</span>
                            <Tag color={sd.color}>{sd.text}</Tag>
                          </div>
                          <div className="app-dashboard-cluster-item__meta">
                            <span>{c.k8sVersion || '版本未知'}</span>
                            <span>{c.nodeCount} 节点</span>
                          </div>
                        </div>
                      </Tooltip>
                    )
                  })}
                </div>
              )}
            </ProCard>
          </Col>
          <Col xs={24} xl={8}>
            <ProCard className="app-dashboard-panel" bordered title="AI 运维助手">
              <div className="app-dashboard-placeholder app-dashboard-placeholder--ai">
                <Bubble
                  variant="outlined"
                  shape="corner"
                  content="AI 助手占位 (Ant Design X)"
                  styles={{ content: { width: '100%' } }}
                />
              </div>
            </ProCard>
          </Col>
        </Row>

        {/* ========== 资源拓扑 + 趋势图表 ========== */}
        <Row gutter={[16, 16]}>
          <Col xs={24} xl={16}>
            <ProCard
              className="app-dashboard-panel"
              bordered
              title={
                <span>
                  <ApartmentOutlined style={{ marginInlineEnd: 6 }} />
                  资源拓扑视图
                </span>
              }
            >
              <div className="app-dashboard-placeholder app-dashboard-placeholder--topology">
                拓扑图占位 (React Flow)
              </div>
            </ProCard>
          </Col>
          <Col xs={24} xl={8}>
            <ProCard
              className="app-dashboard-panel"
              bordered
              title={
                <span>
                  <LineChartOutlined style={{ marginInlineEnd: 6 }} />
                  趋势图表
                </span>
              }
            >
              <div className="app-dashboard-placeholder app-dashboard-placeholder--chart">
                图表占位 (Ant Design Charts)
              </div>
            </ProCard>
          </Col>
        </Row>

        {/* ========== 运维入口（终端 / 日志） ========== */}
        <Row gutter={[16, 16]}>
          <Col xs={24} xl={8}>
            <ProCard
              className="app-dashboard-panel"
              bordered
              title={
                <span>
                  <CodeOutlined style={{ marginInlineEnd: 6 }} />
                  快速终端
                </span>
              }
            >
              <div className="app-dashboard-placeholder app-dashboard-placeholder--terminal">
                终端占位 (Xterm)
              </div>
            </ProCard>
          </Col>
          <Col xs={24} xl={16}>
            <ProCard
              className="app-dashboard-panel"
              bordered
              title={
                <span>
                  <AlertOutlined style={{ marginInlineEnd: 6 }} />
                  近期事件
                </span>
              }
            >
              <div className="app-dashboard-placeholder app-dashboard-placeholder--chart">
                事件流占位
              </div>
            </ProCard>
          </Col>
        </Row>
      </div>
    </AppPage>
  )
}

export default DashboardPage
