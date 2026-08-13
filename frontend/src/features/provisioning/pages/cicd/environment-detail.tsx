/**
 * 环境详情 - 展示环境信息、部署历史
 */
import React, { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Descriptions, Drawer, Form, Input, message, Modal, Select, Space, Table, Tag, Tooltip, Typography } from 'antd'
import {
  ArrowLeftOutlined,
  EditOutlined,
  DeleteOutlined,
  GlobalOutlined,
} from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'
import { getEnvironment, updateEnvironment, deleteEnvironment } from '@/features/provisioning/api/cicd'

const { Text } = Typography

const STATUS_META: Record<string, { color: string; text: string }> = {
  running: { color: 'processing', text: '运行中' },
  stopped: { color: 'default', text: '已停止' },
  deployed: { color: 'success', text: '已部署' },
  failed: { color: 'error', text: '失败' },
  idle: { color: 'default', text: '未部署' },
  success: { color: 'success', text: '成功' },
}

const ENV_TYPE_COLOR: Record<string, string> = {
  production: DESIGN_COLORS.danger,
  staging: DESIGN_COLORS.warning,
  development: DESIGN_COLORS.success,
}

const ENV_TYPE_OPTIONS = [
  { value: 'production', label: '生产环境' },
  { value: 'staging', label: '预发环境' },
  { value: 'development', label: '开发环境' },
]

const EnvironmentDetailPage: React.FC = () => {
  const [envData, setEnvData] = useState<any>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [form] = Form.useForm()
  const envId = history.location.pathname.split('/').pop() || ''

  useEffect(() => {
    if (!envId) return
    getEnvironment(envId)
      .then((data) => setEnvData(data))
      .catch((error) => message.error(error instanceof Error ? error.message : '加载环境详情失败'))
  }, [envId])

  const envType = envData?.environmentType || envData?.environment_type || 'development'
  const envColor = ENV_TYPE_COLOR[envType] || DESIGN_COLORS.primary
  const statusMeta = STATUS_META[envData?.status] ?? STATUS_META.idle!
  const deployHistory = envData?.deployHistory || envData?.deploy_history || []
  const envName = envData?.name || '-'
  const envLabel = envData?.label || '-'
  const envNamespace = envData?.namespace || 'default'
  const envVersion = envData?.currentVersion || envData?.current_version || '-'
  const envDeployedBy = envData?.deployedBy || envData?.deployed_by || '-'
  const envLastDeploy = envData?.lastDeployedAt || envData?.last_deployed_at || '-'
  const envDeployCount = envData?.deployCount || envData?.deploy_count || 0

  const handleEdit = () => {
    form.setFieldsValue({
      name: envData?.name,
      label: envData?.label,
      type: envType,
      namespace: envData?.namespace || 'default',
    })
    setDrawerOpen(true)
  }

  const handleUpdate = () => {
    form.validateFields().then((values) => {
      updateEnvironment(envId, values)
        .then(() => {
          message.success('环境更新成功')
          setDrawerOpen(false)
          getEnvironment(envId).then(setEnvData)
        })
        .catch((error) => message.error(error instanceof Error ? error.message : '更新失败'))
    })
  }

  const handleDelete = () => {
    Modal.confirm({
      title: '删除环境',
      content: '确定删除环境「' + envName + '」吗？此操作不可恢复。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => deleteEnvironment(envId).then(() => {
        message.success('环境已删除')
        history.push('/cicd/environments')
      }).catch((err) => message.error(err instanceof Error ? err.message : '删除失败')),
    })
  }

  return (
    <AppPage keepHeaderTitle title={envName}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Tooltip title="返回环境列表"><Button aria-label="返回环境列表" icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/environments')} /></Tooltip>
            <Tag style={{ color: envColor, borderColor: envColor + '40', background: envColor + '0d' }}>{envLabel}</Tag>
            <Tag color={statusMeta.color}>{statusMeta.text}</Tag>
          </Space>
          <Space>
            <Tooltip title="编辑环境"><Button aria-label="编辑环境" icon={<EditOutlined />} onClick={handleEdit} /></Tooltip>
            <Tooltip title="删除环境"><Button danger aria-label="删除环境" icon={<DeleteOutlined />} onClick={handleDelete} /></Tooltip>
          </Space>
        </div>

        {/* 基本信息 */}
        <Card title="基本信息">
          <div style={{ borderLeft: '4px solid ' + envColor, paddingLeft: 16, marginBottom: 16 }}>
            <Space>
              <GlobalOutlined style={{ color: envColor, fontSize: 18 }} />
              <Text strong style={{ fontSize: 16 }}>{envName}</Text>
              <Text type="secondary">{envLabel}</Text>
            </Space>
          </div>
          <Descriptions column={{ xs: 1, sm: 2, lg: 3 }}>
            <Descriptions.Item label="环境名称">{envName}</Descriptions.Item>
            <Descriptions.Item label="环境类型">{envLabel}</Descriptions.Item>
            <Descriptions.Item label="命名空间">{envNamespace}</Descriptions.Item>
            <Descriptions.Item label="当前版本">
              <Tag color="blue">{envVersion}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusMeta.color}>{statusMeta.text}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="部署次数">{envDeployCount}</Descriptions.Item>
            <Descriptions.Item label="最近部署">{envLastDeploy}</Descriptions.Item>
            <Descriptions.Item label="部署者">{envDeployedBy}</Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 部署历史 */}
        <Card title="部署历史">
          {deployHistory.length > 0 ? (
            <Table
              rowKey="id"
              dataSource={deployHistory}
              pagination={false}
              size="small"
              columns={[
                {
                  title: '版本', dataIndex: 'version', key: 'version', width: 120,
                  render: (version: string) => <Tag color="blue">{version}</Tag>,
                },
                {
                  title: '状态', dataIndex: 'status', key: 'status', width: 80,
                  render: (status: string) => <Tag color={STATUS_META[status]?.color}>{STATUS_META[status]?.text}</Tag>,
                },
                { title: '部署者', dataIndex: 'deployedBy', key: 'deployedBy', width: 100 },
                { title: '耗时', dataIndex: 'duration', key: 'duration', width: 100 },
                { title: '部署时间', dataIndex: 'deployedAt', key: 'deployedAt' },
              ]}
            />
          ) : (
            <EmptyState description="暂无部署历史" />
          )}
        </Card>
      </div>

      {/* 编辑环境 Drawer */}
      <Drawer
        title="编辑环境"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={480}
        extra={
          <Button type="primary" onClick={handleUpdate}>保存</Button>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="环境名称" rules={[{ required: true, message: '请输入环境名称' }]}>
            <Input placeholder="如 production" />
          </Form.Item>
          <Form.Item name="label" label="显示名称" rules={[{ required: true, message: '请输入显示名称' }]}>
            <Input placeholder="如 生产环境" />
          </Form.Item>
          <Form.Item name="type" label="环境类型" rules={[{ required: true, message: '请选择环境类型' }]}>
            <Select options={ENV_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: '请输入命名空间' }]}>
            <Input placeholder="default" />
          </Form.Item>
        </Form>
      </Drawer>
    </AppPage>
  )
}

export default EnvironmentDetailPage
