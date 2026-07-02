/**
 * 部署配置页 — 流程步骤配置 + 仓库配置
 * 对标 frontend-old DeployConfigView
 */
import React, { useMemo, useState } from 'react'
import { Card, Tabs, Table, Button, Space, Tag, Modal, Form, Input, InputNumber, Switch, Select, Popconfirm, message, Tooltip, Badge, Alert, Typography, Empty } from 'antd'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import TerminalCodeBlock from '@/components/TerminalCodeBlock'
import {
  listDeployStepConfigs,
  listSupportedOSTypes,
  updateDeployStepConfig,
  listDeployConfigVersions,
  listRepositories,
  createRepository,
  updateRepository,
  deleteRepository,
} from '@/services/deploy'
import type { DeployStepConfig, RepositoryConfig } from '@/types'
import {
  EyeOutlined,
  EditOutlined,
  HistoryOutlined,
  PlusOutlined,
} from '@ant-design/icons'

const { Text } = Typography

const OS_TYPE_LABELS: Record<string, string> = {
  ubuntu: 'Ubuntu 系列',
  debian: 'Ubuntu 系列',
  centos: '红帽系列',
  rocky: '红帽系列',
  rhel: '红帽系列',
  almalinux: '红帽系列',
}

function getOSTypeLabel(value?: string) {
  if (!value) return 'Linux 系列'
  return OS_TYPE_LABELS[value] || value
}

const OS_FAMILY_OPTIONS = [
  { label: 'Ubuntu 系列', value: 'ubuntu' },
  { label: '红帽系列', value: 'centos' },
]

const DeployConfig: React.FC = () => {
  const [activeTab, setActiveTab] = useState('steps')

  return (
    <AppPage>
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            { key: 'steps', label: '流程步骤配置', children: <StepConfigPanel /> },
            { key: 'repos', label: '仓库配置', children: <RepoConfigPanel /> },
          ]}
        />
      </Card>
    </AppPage>
  )
}

// ==================== 流程步骤配置 ====================

const StepConfigPanel: React.FC = () => {
  const [detailModalOpen, setDetailModalOpen] = useState(false)
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [historyModalOpen, setHistoryModalOpen] = useState(false)
  const [currentStep, setCurrentStep] = useState<DeployStepConfig | null>(null)
  const [selectedOSType, setSelectedOSType] = useState<string>('ubuntu')
  const [form] = Form.useForm()
  const queryClient = useQueryClient()
  const watchedCommandTemplate = Form.useWatch('commandTemplate', form)

  const { data: supportedOSTypes } = useQuery({
    queryKey: ['deploy-step-config-os-types'],
    queryFn: ({ signal }) => listSupportedOSTypes(signal),
  })

  const { data: steps, isLoading } = useQuery({
    queryKey: ['deploy-step-configs', selectedOSType],
    queryFn: ({ signal }) => listDeployStepConfigs({ osType: selectedOSType }, signal),
  })

  const { data: versions } = useQuery({
    queryKey: ['deploy-config-versions', currentStep?.id],
    queryFn: ({ signal }) => listDeployConfigVersions(currentStep!.id, signal),
    enabled: historyModalOpen && !!currentStep?.id,
  })

  const updateMutation = useMutation({
    mutationFn: (values: Partial<DeployStepConfig>) => updateDeployStepConfig(currentStep!.id, values),
    onSuccess: () => {
      message.success('配置已更新')
      queryClient.invalidateQueries({ queryKey: ['deploy-step-configs'] })
      setEditModalOpen(false)
    },
  })

  const handleDetail = (record: DeployStepConfig) => {
    setCurrentStep(record)
    setDetailModalOpen(true)
  }

  const handleEdit = (record: DeployStepConfig) => {
    setCurrentStep(record)
    form.setFieldsValue(record)
    setEditModalOpen(true)
  }

  const handleHistory = (record: DeployStepConfig) => {
    setCurrentStep(record)
    setHistoryModalOpen(true)
  }

  const osTypeOptions = useMemo(() => {
    const supported = new Set<string>((supportedOSTypes || []).map((item) => {
      if (['centos', 'rocky', 'rhel', 'almalinux'].includes(item)) return 'centos'
      return 'ubuntu'
    }))
    return OS_FAMILY_OPTIONS.filter((item) => supported.size === 0 || supported.has(item.value))
  }, [supportedOSTypes])

  const columns = [
    {
      title: '排序',
      dataIndex: 'stepOrder',
      key: 'stepOrder',
      width: 80,
      sorter: (a: DeployStepConfig, b: DeployStepConfig) => a.stepOrder - b.stepOrder,
    },
    {
      title: '步骤标识',
      dataIndex: 'stepKey',
      key: 'stepKey',
      width: 150,
      render: (text: string) => <Text code>{text}</Text>,
    },
    {
      title: '步',
      dataIndex: 'stepName',
      key: 'stepName',
      width: 120,
      render: (text: string) => <strong>{text}</strong>,
    },
    {
      title: '说明',
      dataIndex: 'description',
      key: 'description',
      width: 260,
      ellipsis: true,
    },
    {
      title: '超时(秒)',
      dataIndex: 'timeoutSeconds',
      key: 'timeoutSeconds',
      width: 90,
    },
    {
      title: '重试',
      dataIndex: 'retryCount',
      key: 'retryCount',
      width: 60,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) => (
        <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '禁用'} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: DeployStepConfig) => (
        <Space>
          <Tooltip title="命令详情">
            <a onClick={() => handleDetail(record)}><EyeOutlined /></a>
          </Tooltip>
          <Tooltip title="编辑">
            <a onClick={() => handleEdit(record)}><EditOutlined /></a>
          </Tooltip>
          <Tooltip title="变更历史">
            <a onClick={() => handleHistory(record)}><HistoryOutlined /></a>
          </Tooltip>
        </Space>
      ),
    },
  ]

  return (
    <>
      <Space direction="vertical" size={16} style={{ width: '100%', marginBottom: 16 }}>
        <Alert
          type="info"
          showIcon
          message="按 Linux 系列查看完整 6 步部署流程"
          description="不同 Linux 系列部署 K8s 都需要经过完整的 6 个步骤。这里按系列切换，列表只保留步骤清单，命令详情从操作栏按需查看。"
        />
        <Space align="center" wrap>
          <Text type="secondary">Linux 分类</Text>
          <Select
            value={selectedOSType}
            onChange={setSelectedOSType}
            options={osTypeOptions}
            style={{ width: 220 }}
          />
          <Text type="secondary">当前显示 {steps?.length || 0} / 6 个流程步骤</Text>
        </Space>
      </Space>

      <Card bodyStyle={{ padding: 0 }}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={steps}
          loading={isLoading}
          pagination={false}
          scroll={{ x: 980 }}
        />
      </Card>

      <Modal
        title="命令详情"
        open={detailModalOpen}
        onCancel={() => setDetailModalOpen(false)}
        footer={null}
        width={980}
      >
        {currentStep ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Space wrap>
              <Tag color="blue">步骤 {currentStep.stepOrder}</Tag>
              <Tag color="gold">{getOSTypeLabel(selectedOSType)}</Tag>
              <Tag color={currentStep.enabled ? 'green' : 'default'}>{currentStep.enabled ? '启用' : '禁用'}</Tag>
            </Space>
            <div>
              <Text strong style={{ fontSize: 18 }}>{currentStep.stepName}</Text>
              <div style={{ marginTop: 8 }}>
                <Text type="secondary">{currentStep.description || '暂无说明'}</Text>
              </div>
            </div>
            <Space>
              <Text type="secondary">超时 {currentStep.timeoutSeconds}s</Text>
              <Text type="secondary">重试 {currentStep.retryCount} 次</Text>
            </Space>
            <TerminalCodeBlock
              title={`${currentStep.stepKey}.${selectedOSType}.sh`}
              content={currentStep.commandTemplate}
            />
            <Space>
              <Button
                type="primary"
                icon={<EditOutlined />}
                onClick={() => {
                  setDetailModalOpen(false)
                  handleEdit(currentStep)
                }}
              >
                编辑当前步骤
              </Button>
              <Button
                icon={<HistoryOutlined />}
                onClick={() => {
                  setDetailModalOpen(false)
                  handleHistory(currentStep)
                }}
              >
                变更历史
              </Button>
            </Space>
          </Space>
        ) : (
          <Empty description="暂无可查看的命令详情" />
        )}
      </Modal>

      <Modal
        title={`编辑步骤 — ${currentStep?.stepName}`}
        open={editModalOpen}
        onCancel={() => setEditModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={updateMutation.isPending}
        width={640}
      >
        <Form form={form} layout="vertical" onFinish={(values) => updateMutation.mutate(values)}>
          <Form.Item name="stepName" label="步骤名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="commandTemplate" label="命令模板">
            <Input.TextArea rows={6} style={{ fontFamily: 'monospace' }} />
          </Form.Item>
          <Form.Item label="命令预览">
            <TerminalCodeBlock
              title={`${currentStep?.stepKey || 'step'}.${currentStep?.osType || 'linux'}.sh`}
              content={watchedCommandTemplate}
            />
          </Form.Item>
          <Space>
            <Form.Item name="timeoutSeconds" label="超时(秒)">
              <InputNumber min={10} max={3600} />
            </Form.Item>
            <Form.Item name="retryCount" label="重试次数">
              <InputNumber min={0} max={10} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>

      <Modal
        title={`变更历史 — ${currentStep?.stepName}`}
        open={historyModalOpen}
        onCancel={() => setHistoryModalOpen(false)}
        footer={null}
        width={640}
      >
        <Table
          rowKey="id"
          dataSource={versions}
          pagination={false}
          size="small"
          columns={[
            { title: '时间', dataIndex: 'changedAt', width: 180, render: (t: string) => new Date(t).toLocaleString() },
            { title: '类型', dataIndex: 'changeType', width: 80, render: (t: string) => <Tag color={t === 'create' ? 'blue' : 'orange'}>{t === 'create' ? '创建' : '更新'}</Tag> },
            { title: '摘要', dataIndex: 'changeSummary' },
          ]}
        />
      </Modal>
    </>
  )
}

// ==================== 仓库配置 ====================

const REPO_TYPE_OPTIONS = [
  { value: 'container_mirror', label: '容器镜像加速' },
  { value: 'registry', label: '容器镜像仓库' },
  { value: 'yum', label: 'YUM 源' },
  { value: 'apt', label: 'APT 源' },
]

const AUTH_TYPE_OPTIONS = [
  { value: 'none', label: '无需认证' },
  { value: 'basic', label: '用户名/密码' },
  { value: 'token', label: 'Token' },
]

const RepoConfigPanel: React.FC = () => {
  const [modalOpen, setModalOpen] = useState(false)
  const [editRecord, setEditRecord] = useState<RepositoryConfig | null>(null)
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  const { data: repos, isLoading } = useQuery({
    queryKey: ['repositories'],
    queryFn: ({ signal }) => listRepositories(undefined, signal),
  })

  const createMutation = useMutation({
    mutationFn: createRepository,
    onSuccess: () => {
      message.success('仓库创建成功')
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      setModalOpen(false)
      form.resetFields()
    },
  })

  const updateMutation = useMutation({
    mutationFn: (values: Partial<RepositoryConfig>) => updateRepository(editRecord!.id, values),
    onSuccess: () => {
      message.success('仓库更新成功')
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      setModalOpen(false)
      setEditRecord(null)
      form.resetFields()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteRepository,
    onSuccess: () => {
      message.success('仓库已删除')
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  const handleAdd = () => {
    setEditRecord(null)
    form.resetFields()
    form.setFieldsValue({ authType: 'none', priority: 1, enabled: true })
    setModalOpen(true)
  }

  const handleEdit = (record: RepositoryConfig) => {
    setEditRecord(record)
    form.setFieldsValue(record)
    setModalOpen(true)
  }

  const columns = [
    {
      title: '仓库名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: RepositoryConfig) => (
        <Space>
          <strong>{text}</strong>
          {record.isDefault && <Tag color="blue">默认</Tag>}
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'repoType',
      key: 'repoType',
      width: 120,
      render: (t: string) => {
        const opt = REPO_TYPE_OPTIONS.find((o) => o.value === t)
        return <Tag>{opt?.label || t}</Tag>
      },
    },
    {
      title: '地址',
      dataIndex: 'url',
      key: 'url',
      ellipsis: true,
    },
    {
      title: '认证',
      dataIndex: 'authType',
      key: 'authType',
      width: 100,
      render: (t: string) => {
        const opt = AUTH_TYPE_OPTIONS.find((o) => o.value === t)
        return opt?.label || t
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      sorter: (a: RepositoryConfig, b: RepositoryConfig) => (a.priority ?? 0) - (b.priority ?? 0),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) => (
        <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '禁用'} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: RepositoryConfig) => (
        <Space>
          <a onClick={() => handleEdit(record)}>编辑</a>
          <Popconfirm title="确认删除该仓库配置？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <a style={{ color: '#ff4d4f' }}>删除</a>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          添加仓库
        </Button>
      </div>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={repos}
        loading={isLoading}
        pagination={false}
      />

      <Modal
        title={editRecord ? '编辑仓库' : '添加仓库'}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditRecord(null); form.resetFields() }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        width={560}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            if (editRecord) {
              updateMutation.mutate(values)
            } else {
              createMutation.mutate(values)
            }
          }}
        >
          <Form.Item name="name" label="仓库名称" rules={[{ required: true, message: '请输入仓库名称' }]}>
            <Input placeholder="如 Harbor 私有仓库" />
          </Form.Item>
          <Form.Item name="repoType" label="仓库类型" rules={[{ required: true, message: '请选择仓库类型' }]}>
            <Select options={REPO_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="url" label="仓库地址" rules={[{ required: true, message: '请输入仓库地址' }]}>
            <Input placeholder="https://harbor.example.com" />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="authType" label="认证方式">
            <Select options={AUTH_TYPE_OPTIONS} />
          </Form.Item>
          <Space>
            <Form.Item name="priority" label="优先级">
              <InputNumber min={1} max={100} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </>
  )
}

export default DeployConfig
