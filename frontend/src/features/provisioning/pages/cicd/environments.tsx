/**
 * 环境管理 - 部署环境列表（dev / staging / prod）
 */
import { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Col, Drawer, Form, Input, message, Row, Select, Tag, Typography } from 'antd'
import { PlusOutlined, GlobalOutlined } from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'
import { createEnvironment, getEnvironments, type Environment } from '@/features/provisioning/api/cicd'

const { Text } = Typography

interface EnvItem {
  id: string
  name: string
  label: string
  type: 'production' | 'staging' | 'development'
  cluster: string
  namespace: string
  version: string
  status: 'deployed' | 'failed' | 'idle'
  lastDeploy: string
  deployedBy: string
  deployCount: number
}

const ENVIRONMENTS: EnvItem[] = [
  { id: '1', name: 'production', label: '生产', type: 'production', cluster: 'prod-cluster-01', namespace: 'default', version: 'v1.2.3', status: 'deployed', lastDeploy: '07-26 14:32', deployedBy: 'admin', deployCount: 18 },
  { id: '2', name: 'staging', label: '预发', type: 'staging', cluster: 'staging-cluster-01', namespace: 'default', version: 'v1.2.4-rc', status: 'deployed', lastDeploy: '07-26 10:15', deployedBy: 'admin', deployCount: 32 },
  { id: '3', name: 'development', label: '开发', type: 'development', cluster: 'dev-cluster-01', namespace: 'default', version: 'v1.2.4-dev', status: 'failed', lastDeploy: '07-26 09:00', deployedBy: 'ci-bot', deployCount: 86 },
]
void ENVIRONMENTS

const ENV_TYPE_COLOR: Record<string, string> = {
  production: DESIGN_COLORS.danger,
  staging: DESIGN_COLORS.warning,
  development: DESIGN_COLORS.success,
}

const STATUS_META: Record<string, { color: string; text: string }> = {
  deployed: { color: 'success', text: '已部署' },
  failed: { color: 'error', text: '部署失败' },
  idle: { color: 'default', text: '未部署' },
}

const ENV_TYPE_OPTIONS = [
  { value: 'production', label: '生产环境' },
  { value: 'staging', label: '预发环境' },
  { value: 'development', label: '开发环境' },
]

const CLUSTER_OPTIONS = [
  { value: 'prod-cluster-01', label: 'prod-cluster-01' },
  { value: 'staging-cluster-01', label: 'staging-cluster-01' },
  { value: 'dev-cluster-01', label: 'dev-cluster-01' },
]

const EnvironmentsPage: React.FC = () => {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [form] = Form.useForm()
  const [environments, setEnvironments] = useState<EnvItem[]>([])
  const loadEnvironments = () => getEnvironments().then((res) => setEnvironments((res.list || []).map((e: Environment) => ({ id: String(e.id), name: e.name, label: e.label, type: (e.environmentType || e.environment_type || 'development') as EnvItem['type'], cluster: '-', namespace: e.namespace || 'default', version: e.currentVersion || e.current_version || '-', status: e.status as EnvItem['status'], lastDeploy: e.lastDeployedAt || e.last_deployed_at || '-', deployedBy: e.deployedBy || e.deployed_by || '-', deployCount: e.deployCount || e.deploy_count || 0 }))))
  useEffect(() => { loadEnvironments() }, [])

  const handleAdd = () => {
    form.resetFields()
    form.setFieldsValue({ type: 'development', namespace: 'default' })
    setDrawerOpen(true)
  }

  const handleSubmit = () => {
    form.validateFields().then((values) => {
      createEnvironment(values).then(() => loadEnvironments())
      message.success('环境创建成功')
      setDrawerOpen(false)
      form.resetFields()
    })
  }

  const handleClose = () => {
    setDrawerOpen(false)
    form.resetFields()
  }

  return (
    <AppPage>
      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontSize: 16, fontWeight: 500 }}>环境列表</span>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>创建环境</Button>
        </div>

        {environments.length === 0 ? (
          <EmptyState description="暂无环境，点击「创建环境」配置 dev / staging / prod 等部署环境" />
        ) : (
          <Row gutter={[16, 16]}>
            {environments.map((env) => {
              const envColor = ENV_TYPE_COLOR[env.type]
              const statusMeta = STATUS_META[env.status] ?? STATUS_META.idle!
              return (
                <Col xs={24} sm={12} lg={8} key={env.id}>
                  <Card
                    hoverable
                    style={{ height: '100%', borderLeft: `4px solid ${envColor}` }}
                    onClick={() => history.push(`/cicd/environments/${env.id}`)}
                  >
                    {/* 顶部：名称 + 状态 */}
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <GlobalOutlined style={{ color: envColor, fontSize: 18 }} />
                        <strong style={{ fontSize: 15 }}>{env.name}</strong>
                        <Tag style={{ margin: 0, fontSize: 11, color: envColor, borderColor: `${envColor}40`, background: `${envColor}0d` }}>{env.label}</Tag>
                      </div>
                      <Tag color={statusMeta.color} style={{ margin: 0 }}>{statusMeta.text}</Tag>
                    </div>

                    {/* 基本信息 */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginBottom: 10 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>当前版本</Text>
                        <Text strong style={{ fontSize: 13 }}>{env.version}</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>关联集群</Text>
                        <Text style={{ fontSize: 13 }}>{env.cluster}</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>命名空间</Text>
                        <Text style={{ fontSize: 13 }}>{env.namespace}</Text>
                      </div>
                    </div>

                    {/* 底部：部署信息 */}
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 12, color: DESIGN_COLORS.textSecondary, borderTop: `1px solid ${DESIGN_COLORS.grid}`, paddingTop: 8 }}>
                      <span>最近部署 {env.lastDeploy}</span>
                      <span>近30天 {env.deployCount} 次</span>
                    </div>
                  </Card>
                </Col>
              )
            })}
          </Row>
        )}
      </Card>

      {/* 创建环境 Drawer */}
      <Drawer
        title="创建环境"
        open={drawerOpen}
        onClose={handleClose}
        width={480}
        extra={
          <Button type="primary" onClick={handleSubmit}>保存</Button>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="环境名称" rules={[{ required: true, message: '请输入环境名称' }]}>
            <Input placeholder="如 production" />
          </Form.Item>
          <Form.Item name="type" label="环境类型" rules={[{ required: true, message: '请选择环境类型' }]}>
            <Select options={ENV_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="cluster" label="关联集群" rules={[{ required: true, message: '请选择关联集群' }]}>
            <Select showSearch placeholder="选择集群" options={CLUSTER_OPTIONS} />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: '请输入命名空间' }]}>
            <Input placeholder="default" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="环境用途说明" />
          </Form.Item>
        </Form>
      </Drawer>
    </AppPage>
  )
}

export default EnvironmentsPage
