import React from 'react'
import { Bubble } from '@ant-design/x'
import { ProCard } from '@ant-design/pro-components'
import { Alert, Col, Row, Skeleton, Statistic, Typography } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { fetchDashboardOverview } from '@/services/dashboard'
import type { DashboardStatistic, DashboardTimeRange } from '@/types/dashboard'

const { Text } = Typography

const DEFAULT_TIME_RANGE: DashboardTimeRange = '24h'

const fallbackStats: DashboardStatistic[] = [
  { key: 'clusters', title: '集群总数', value: 0, unit: '个' },
  { key: 'alerts', title: '活跃告警', value: 0, unit: '条' },
  { key: 'automation', title: '自动化任务', value: 0, unit: '次' },
  { key: 'rca', title: '根因定位', value: 0, unit: '次' },
]

const DashboardPage: React.FC = () => {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['dashboard-overview', DEFAULT_TIME_RANGE],
    queryFn: () => fetchDashboardOverview(DEFAULT_TIME_RANGE),
  })

  const statistics = data?.statistics ?? fallbackStats

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

        <ProCard className="app-dashboard__row" ghost>
          <Row gutter={[16, 16]}>
            {statistics.map((item) => (
              <Col key={item.key} xs={24} sm={12} xl={6}>
                <ProCard className="app-dashboard-stat-card" bordered>
                  <Skeleton loading={isLoading} active paragraph={false}>
                    <Statistic title={item.title} value={item.value} suffix={item.unit} />
                    <Text type="secondary">{item.description ?? '暂无数据'}</Text>
                  </Skeleton>
                </ProCard>
              </Col>
            ))}
          </Row>
        </ProCard>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={15}>
            <ProCard className="app-dashboard-panel" bordered>
              <div className="app-dashboard-placeholder app-dashboard-placeholder--topology">
                拓扑图占位 (React Flow)
              </div>
            </ProCard>
          </Col>
          <Col xs={24} xl={9}>
            <ProCard className="app-dashboard-panel" bordered>
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

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={14}>
            <ProCard className="app-dashboard-panel" bordered>
              <div className="app-dashboard-placeholder app-dashboard-placeholder--terminal">
                终端占位 (Xterm)
              </div>
            </ProCard>
          </Col>
          <Col xs={24} xl={10}>
            <ProCard className="app-dashboard-panel" bordered>
              <div className="app-dashboard-placeholder app-dashboard-placeholder--chart">
                图表占位 (Ant Design Charts)
              </div>
            </ProCard>
          </Col>
        </Row>
      </div>
    </AppPage>
  )
}

export default DashboardPage
