/**
 * 全局资源视图
 * 展示所有集群的概览卡片，点击直接进入集群管理
 */
import React from 'react'
import { history } from '@umijs/max'
import { Card, Row, Col, Spin, Tag, Typography, Empty, Button } from 'antd'
import { ClusterOutlined, ArrowRightOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'

const { Text } = Typography

/** 集群概览卡片（简洁版，不重复统计） */
function ClusterCard({ cluster }: { cluster: any }) {
  const statusColor =
    cluster.status === 'healthy' ? 'success' : cluster.status === 'unhealthy' ? 'error' : 'default'
  const statusText = cluster.status === 'healthy' ? '健康' : cluster.status === 'unhealthy' ? '异常' : '未知'

  return (
    <Card
      hoverable
      size="small"
      style={{ borderRadius: 12, cursor: 'pointer' }}
      onClick={() => history.push(`/clusters/${cluster.id}`)}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ClusterOutlined style={{ color: '#2563eb', fontSize: 20 }} />
          <Text strong style={{ fontSize: 15 }}>{cluster.name}</Text>
        </div>
        <Tag color={statusColor}>{statusText}</Tag>
      </div>
      <div style={{ marginTop: 12, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {cluster.k8sVersion || '版本未知'} · {cluster.nodeCount || 0} 节点
        </Text>
        <ArrowRightOutlined style={{ color: '#cbd5e1' }} />
      </div>
    </Card>
  )
}

/** 全局资源视图页面 */
const ResourceOverviewPage: React.FC = () => {
  const { data, isLoading } = useQuery({
    queryKey: ['clusters-for-resource-overview'],
    queryFn: () => listClusters({ pageSize: 100 }),
  })

  const clusters = data?.items || []
  const healthyClusters = clusters.filter((c: any) => c.status === 'healthy').length

  return (
    <AppPage breadcrumbRender={false}>
      <div className="app-data-console">
        {/* 全局统计 */}
        <section className="app-data-console__statgrid">
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">集群总数</span>
            <strong className="app-data-console__stat-value">{clusters.length}</strong>
            <span className="app-data-console__stat-hint">健康 {healthyClusters} 个</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">资源视图</span>
            <strong className="app-data-console__stat-value" style={{ fontSize: 16 }}>
              点击集群进入管理
            </strong>
            <span className="app-data-console__stat-hint">详细统计在集群详情中查看</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">快捷操作</span>
            <strong className="app-data-console__stat-value" style={{ fontSize: 14 }}>
              <Button
                type="link"
                size="small"
                onClick={() => history.push('/clusters')}
              >
                集群管理
              </Button>
            </strong>
            <span className="app-data-console__stat-hint">导入/管理集群</span>
          </div>
        </section>

        {/* 集群卡片列表 */}
        <section>
          {isLoading ? (
            <div style={{ textAlign: 'center', padding: 60 }}>
              <Spin size="large" />
            </div>
          ) : clusters.length === 0 ? (
            <Empty description="暂无集群，请先导入集群" image={Empty.PRESENTED_IMAGE_SIMPLE}>
              <Button type="primary" onClick={() => history.push('/clusters/import')}>
                导入集群
              </Button>
            </Empty>
          ) : (
            <Row gutter={[16, 16]}>
              {clusters.map((cluster: any) => (
                <Col key={cluster.id} xs={24} sm={12} lg={8} xl={6}>
                  <ClusterCard cluster={cluster} />
                </Col>
              ))}
            </Row>
          )}
        </section>
      </div>
    </AppPage>
  )
}

export default ResourceOverviewPage
