/**
 * 环境管理 - 部署环境列表（dev / staging / prod）
 */
import { useEffect, useMemo, useState } from 'react'
import { history } from '@umijs/max'
import { useQuery } from '@tanstack/react-query'
import { Button, Card, Col, Drawer, Form, Input, message, Popconfirm, Row, Select, Tag, Tooltip, Typography } from 'antd'
import { DeleteOutlined, EditOutlined, GlobalOutlined, PlusOutlined, SaveOutlined } from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'
import { createEnvironment, deleteEnvironment, getEnvironments, updateEnvironment, type Environment } from '@/features/provisioning/api/cicd'
import { listClusters } from '@/features/fleet'

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
  description: string
}

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

const EnvironmentsPage: React.FC = () => {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editingEnv, setEditingEnv] = useState<EnvItem | null>(null)
  const [form] = Form.useForm()
  const [environments, setEnvironments] = useState<EnvItem[]>([])
  const clustersQuery = useQuery({
    queryKey: ['cicd-environment-clusters'],
    queryFn: ({ signal }) => listClusters({ page: 1, pageSize: 100 }, signal),
  })
  const clusterOptions = (clustersQuery.data?.items || []).map((cluster) => ({ value: cluster.id, label: cluster.name }))
  const clusterMap = useMemo(() => {
    const m: Record<string, string> = {}
    ;(clustersQuery.data?.items || []).forEach((c) => { m[String(c.id)] = c.name })
    return m
  }, [clustersQuery.data])
  const loadEnvironments = () => getEnvironments().then((res) => setEnvironments((res.list || []).map((e: Environment) => ({ id: String(e.id), name: e.name, label: e.label, type: (e.environmentType || e.environment_type || 'development') as EnvItem['type'], cluster: String(e.clusterId || ''), namespace: e.namespace || 'default', version: e.currentVersion || e.current_version || '-', status: e.status as EnvItem['status'], lastDeploy: e.lastDeployedAt || e.last_deployed_at || '-', deployedBy: e.deployedBy || e.deployed_by || '-', deployCount: e.deployCount || e.deploy_count || 0, description: e.description || '' }))))
  useEffect(() => { loadEnvironments() }, [])

  const handleAdd = () => {
    setEditingEnv(null)
    form.resetFields()
    form.setFieldsValue({ type: 'development', namespace: 'default' })
    setDrawerOpen(true)
  }

  const handleEdit = (env: EnvItem) => {
    setEditingEnv(env)
    form.setFieldsValue({ name: env.name, type: env.type, clusterId: env.cluster ? Number(env.cluster) : undefined, namespace: env.namespace, description: env.description })
    setDrawerOpen(true)
  }

  const handleDelete = (env: EnvItem) => {
    deleteEnvironment(env.id).then(() => {
      message.success('环境删除成功')
      loadEnvironments()
    })
  }

  const handleSubmit = () => {
    form.validateFields().then((values) => {
      if (editingEnv) {
        updateEnvironment(editingEnv.id, values).then(() => {
          message.success('环境更新成功')
          loadEnvironments()
          setDrawerOpen(false)
          form.resetFields()
          setEditingEnv(null)
        })
      } else {
        createEnvironment(values).then(() => {
          message.success('环境创建成功')
          loadEnvironments()
          setDrawerOpen(false)
          form.resetFields()
        })
      }
    })
  }

  const handleClose = () => {
    setDrawerOpen(false)
    form.resetFields()
    setEditingEnv(null)
  }

  return (
    <AppPage>
      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontSize: 16, fontWeight: 500 }}>环境列表</span>
          <Tooltip title="创建环境"><Button type="primary" aria-label="创建环境" icon={<PlusOutlined />} onClick={handleAdd} /></Tooltip>
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
                      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                        <Tag color={statusMeta.color} style={{ margin: 0 }}>{statusMeta.text}</Tag>
                        <Tooltip title="编辑"><Button size="small" type="text" aria-label="编辑环境" icon={<EditOutlined />} onClick={(e) => { e.stopPropagation(); handleEdit(env) }} /></Tooltip>
                        <Popconfirm title="确认删除该环境？" description="删除后不可恢复" onConfirm={() => handleDelete(env)}>
                          <Button size="small" type="text" danger aria-label="删除环境" icon={<DeleteOutlined />} onClick={(e) => e.stopPropagation()} />
                        </Popconfirm>
                      </div>
                    </div>

                    {/* 基本信息 */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginBottom: 10 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>当前版本</Text>
                        <Text strong style={{ fontSize: 13 }}>{env.version}</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>关联集群</Text>
                        <Text style={{ fontSize: 13 }}>{clusterMap[env.cluster] || env.cluster || '-'}</Text>
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
        title={editingEnv ? '编辑环境' : '创建环境'}
        open={drawerOpen}
        onClose={handleClose}
        width={480}
        extra={
          <Tooltip title="保存"><Button type="primary" aria-label="保存" icon={<SaveOutlined />} onClick={handleSubmit} /></Tooltip>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="环境名称" rules={[{ required: true, message: '请输入环境名称' }]}>
            <Input placeholder="如 production" />
          </Form.Item>
          <Form.Item name="type" label="环境类型" rules={[{ required: true, message: '请选择环境类型' }]}>
            <Select options={ENV_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="clusterId" label="关联集群" rules={[{ required: true, message: '请选择关联集群' }]}>
            <Select showSearch optionFilterProp="label" loading={clustersQuery.isLoading} placeholder="选择集群" options={clusterOptions} />
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
