import React from 'react'
import { history } from '@umijs/max'
import { ClusterOutlined, FileSearchOutlined } from '@ant-design/icons'
import { Button, Card, Empty, Space, Typography } from 'antd'
import { AppPage } from '@/components'

const { Text, Title } = Typography

const LogsPage: React.FC = () => {
  return (
    <AppPage breadcrumbRender={false}>
      <div className="app-page-shell">
        <Card className="app-aiops-panel">
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <Space direction="vertical" size={8}>
                <Title level={3} style={{ margin: 0 }}>
                  日志分析
                </Title>
                <Text type="secondary">全局日志分析页正在建设中。</Text>
              </Space>
            }
          >
            <Space>
              <Button
                type="primary"
                icon={<ClusterOutlined />}
                onClick={() => history.push('/clusters')}
              >
                查看集群
              </Button>
              <Button
                icon={<FileSearchOutlined />}
                onClick={() => history.push('/monitor/dashboard')}
              >
                查看告警
              </Button>
            </Space>
          </Empty>
        </Card>
      </div>
    </AppPage>
  )
}

export default LogsPage
