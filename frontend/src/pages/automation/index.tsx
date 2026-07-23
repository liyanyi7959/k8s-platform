import { history } from '@umijs/max'
import { Button, Card, Col, Divider, Row, Space, Tag, Typography } from 'antd'
import {
  ClusterOutlined,
  CodeOutlined,
  KeyOutlined,
  PlayCircleOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { AppPage } from '@/components'

const { Paragraph, Text, Title } = Typography

const capabilities = [
  {
    title: '运行手册资产',
    description: '统一维护 Playbook、Inventory 模板、软件源和执行环境检查，供各类自动化任务复用。',
    icon: <CodeOutlined />,
    action: '管理资产',
    path: '/automation/assets',
    color: '#1677ff',
  },
  {
    title: 'Kubernetes 集群交付',
    description: '这是一个集群生命周期动作：从主机预检、初始化到注册为受管集群，入口归属集群管理。',
    icon: <ClusterOutlined />,
    action: '部署集群',
    path: '/clusters/provision',
    color: '#13a8a8',
  },
  {
    title: '凭据与访问控制',
    description: 'SSH 密钥和密码属于平台级敏感资产，集中在管理后台管理、审计和授权。',
    icon: <KeyOutlined />,
    action: '打开凭据库',
    path: '/config/credentials',
    color: '#722ed1',
  },
]

export default function AutomationOverviewPage() {
  return (
    <AppPage>
      <Card
        bordered={false}
        style={{
          marginBottom: 16,
          background: 'linear-gradient(135deg, #eff6ff 0%, #f7fbff 58%, #f3fcfa 100%)',
        }}
      >
        <Space align="start" size={16}>
          <PlayCircleOutlined style={{ fontSize: 30, color: '#1677ff', marginTop: 4 }} />
          <div>
            <Title level={3} style={{ margin: 0 }}>
              自动化中心
            </Title>
            <Paragraph style={{ margin: '8px 0 0', maxWidth: 760, color: '#52616b' }}>
              自动化中心沉淀可复用的运行手册和执行资产；具体业务动作从对应业务域发起。这样既能复用
              Ansible 能力，也能让集群创建、应用交付和故障处置拥有清晰的责任边界。
            </Paragraph>
          </div>
        </Space>
      </Card>

      <Row gutter={[16, 16]}>
        {capabilities.map((item) => (
          <Col xs={24} lg={8} key={item.title}>
            <Card hoverable style={{ height: '100%' }}>
              <Space direction="vertical" size={14} style={{ width: '100%' }}>
                <div
                  style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                >
                  <span style={{ color: item.color, fontSize: 28 }}>{item.icon}</span>
                  <Tag color="blue">平台能力</Tag>
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

      <Card bordered={false} style={{ marginTop: 16 }}>
        <Space align="start" size={12}>
          <SafetyCertificateOutlined style={{ color: '#52c41a', fontSize: 20, marginTop: 2 }} />
          <div>
            <Text strong>规划边界</Text>
            <Divider type="vertical" />
            <Text type="secondary">
              后续新增应用发布、巡检修复、扩缩容等能力时，均从这里复用运行手册与资产；执行入口仍放在应用管理、告警处置或集群管理等业务菜单中。
            </Text>
          </div>
        </Space>
      </Card>
    </AppPage>
  )
}
