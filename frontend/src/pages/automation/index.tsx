import { history } from '@umijs/max'
import { Button, Card, Col, Row, Space, Typography } from 'antd'
import {
  ClusterOutlined,
  CodeOutlined,
  KeyOutlined,
} from '@ant-design/icons'
import { AppPage } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'

const { Text } = Typography

const capabilities = [
  {
    title: '运行手册资产',
    description: '维护 Playbook、Inventory 模板、软件源与执行环境。',
    icon: <CodeOutlined />,
    action: '管理资产',
    path: '/automation/assets',
    color: DESIGN_COLORS.primary,
  },
  {
    title: 'Kubernetes 集群交付',
    description: '从主机预检、初始化到注册为受管集群。',
    icon: <ClusterOutlined />,
    action: '部署集群',
    path: '/clusters/provision',
    color: DESIGN_COLORS.primary,
  },
  {
    title: '凭据与访问控制',
    description: '集中管理 SSH 密钥与密码，并支持审计和授权。',
    icon: <KeyOutlined />,
    action: '打开凭据库',
    path: '/config/credentials',
    color: DESIGN_COLORS.primary,
  },
]

export default function AutomationOverviewPage() {
  return (
    <AppPage>
      <Row gutter={[16, 16]}>
        {capabilities.map((item) => (
          <Col xs={24} lg={8} key={item.title}>
            <Card hoverable style={{ height: '100%' }}>
              <Space direction="vertical" size={14} style={{ width: '100%' }}>
                <div
                  style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                >
                  <span style={{ color: item.color, fontSize: 28 }}>{item.icon}</span>
                </div>
                <Text strong style={{ fontSize: 16 }}>
                  {item.title}
                </Text>
                <Text type="secondary" style={{ minHeight: 66, lineHeight: 1.7 }}>
                  {item.description}
                </Text>
                <Button
                  type="link"
                  style={{ padding: 0, alignSelf: 'flex-start' }}
                  onClick={() => history.push(item.path)}
                >
                  {item.action} →
                </Button>
              </Space>
            </Card>
          </Col>
        ))}
      </Row>

    </AppPage>
  )
}
