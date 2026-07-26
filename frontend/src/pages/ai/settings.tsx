import React, { useEffect, useState } from 'react'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {
  Badge,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import {
  createAIModel,
  createAIProvider,
  deleteAIModel,
  deleteAIProvider,
  getAIRouteSettings,
  listAIModels,
  listAIProviders,
  updateAIRouteSettings,
  updateAIModel,
  updateAIProvider,
} from '@/services/ai'
import type { AIModel, AIProvider } from '@/types'

const { Text } = Typography

const PROVIDER_TYPE_OPTIONS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'azure-openai', label: 'Azure OpenAI' },
  { value: 'ollama', label: 'Ollama' },
  { value: 'qwen', label: '通义千问' },
]

const MODEL_TYPE_OPTIONS = [
  { value: 'chat', label: '对话模型' },
  { value: 'vision', label: '视觉模型' },
  { value: 'embedding', label: 'Embedding' },
  { value: 'image', label: '图像生成' },
]

const ROUTING_STRATEGY_OPTIONS = [
  { value: 'priority_first', label: '优先级优先' },
  { value: 'default_model_first', label: '默认模型优先' },
  { value: 'capability_first', label: '能力优先' },
]

const getErrorMessage = (error: unknown, fallback: string) =>
  error instanceof Error && error.message ? error.message : fallback

const AiSettings: React.FC = () => {
  const [activeTab, setActiveTab] = useState('providers')

  return (
    <AppPage>
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            { key: 'providers', label: 'AI 提供商', children: <ProviderPanel /> },
            { key: 'models', label: 'AI 模型', children: <ModelPanel /> },
            { key: 'routing', label: 'AI 路由', children: <RouteSettingsPanel /> },
          ]}
        />
      </Card>
    </AppPage>
  )
}

const ProviderPanel: React.FC = () => {
  const [modalOpen, setModalOpen] = useState(false)
  const [editRecord, setEditRecord] = useState<AIProvider | null>(null)
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  const { data: providers = [], isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['ai-providers'],
    queryFn: ({ signal }) => listAIProviders(signal),
  })

  const createMutation = useMutation({
    mutationFn: createAIProvider,
    onSuccess: () => {
      message.success('提供商创建成功')
      queryClient.invalidateQueries({ queryKey: ['ai-providers'] })
      setModalOpen(false)
      form.resetFields()
    },
    onError: (error) => message.error(getErrorMessage(error, '提供商创建失败')),
  })

  const updateMutation = useMutation({
    mutationFn: (values: Partial<AIProvider>) => updateAIProvider(editRecord!.id, values),
    onSuccess: () => {
      message.success('提供商更新成功')
      queryClient.invalidateQueries({ queryKey: ['ai-providers'] })
      setModalOpen(false)
      setEditRecord(null)
      form.resetFields()
    },
    onError: (error) => message.error(getErrorMessage(error, '提供商更新失败')),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteAIProvider(id),
    onSuccess: () => {
      message.success('提供商删除成功')
      queryClient.invalidateQueries({ queryKey: ['ai-providers'] })
      queryClient.invalidateQueries({ queryKey: ['ai-models'] })
    },
    onError: (error) => message.error(getErrorMessage(error, '提供商删除失败')),
  })

  const handleAdd = () => {
    setEditRecord(null)
    form.resetFields()
    form.setFieldsValue({
      providerType: 'openai',
      authScheme: 'bearer',
      priority: 1,
      enabled: true,
    })
    setModalOpen(true)
  }

  const handleEdit = (record: AIProvider) => {
    setEditRecord(record)
    form.setFieldsValue({ ...record, apiKey: '' })
    setModalOpen(true)
  }

  const columns = [
    {
      title: '提供商',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: AIProvider) => (
        <div className="app-table-stack">
          <span className="app-table-stack__main">{text}</span>
          <span className="app-table-stack__sub">{record.baseUrl}</span>
        </div>
      ),
    },
    {
      title: '类型',
      dataIndex: 'providerType',
      key: 'providerType',
      width: 140,
      render: (value: string) => {
        const option = PROVIDER_TYPE_OPTIONS.find((item) => item.value === value)
        return <Tag color="blue">{option?.label || value}</Tag>
      },
    },
    {
      title: 'API Key',
      dataIndex: 'hasApiKey',
      key: 'hasApiKey',
      width: 120,
      render: (hasApiKey: boolean) =>
        hasApiKey ? (
          <Badge status="success" text="已配置" />
        ) : (
          <Badge status="default" text="未配置" />
        ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      align: 'center' as const,
      sorter: (a: AIProvider, b: AIProvider) => a.priority - b.priority,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      align: 'center' as const,
      render: (enabled: boolean) => (
        <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '停用'} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      align: 'center' as const,
      render: (_: unknown, record: AIProvider) => (
        <div className="app-table-actions app-table-actions--icon">
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确定删除该提供商吗？"
            description="若该提供商下仍有关联模型，系统会阻止删除。"
            onConfirm={() => deleteMutation.mutate(record.id)}
          >
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ]

  return (
    <div className="app-data-console">
      <section className="app-data-console__filters app-data-console__filters--actions-only">
        <div className="app-data-console__filters-right">
          <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            添加提供商
          </Button>
        </div>
      </section>

      <div className="app-data-console__table">
        <Table
          rowKey="id"
          columns={columns}
          dataSource={providers}
          loading={isLoading}
          pagination={false}
        />
      </div>

      <Modal
        title={editRecord ? '编辑提供商' : '添加提供商'}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditRecord(null)
          form.resetFields()
        }}
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
          <Form.Item
            name="name"
            label="提供商名称"
            rules={[{ required: true, message: '请输入提供商名称' }]}
          >
            <Input placeholder="例如 OpenAI Production" />
          </Form.Item>
          <Form.Item
            name="providerType"
            label="提供商类型"
            rules={[{ required: true, message: '请选择提供商类型' }]}
          >
            <Select options={PROVIDER_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item
            name="baseUrl"
            label="Base URL"
            rules={[{ required: true, message: '请输入 Base URL' }]}
          >
            <Input placeholder="https://api.openai.com/v1" />
          </Form.Item>
          <Form.Item name="apiKey" label="API Key" extra={editRecord ? '留空表示不修改' : ''}>
            <Input.Password placeholder="sk-..." />
          </Form.Item>
          <Space size={16} wrap>
            <Form.Item name="priority" label="优先级">
              <InputNumber min={1} max={100} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  )
}

const ModelPanel: React.FC = () => {
  const [modalOpen, setModalOpen] = useState(false)
  const [editRecord, setEditRecord] = useState<AIModel | null>(null)
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  const { data: models = [], isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['ai-models'],
    queryFn: ({ signal }) => listAIModels(signal),
  })

  const { data: providers = [] } = useQuery({
    queryKey: ['ai-providers'],
    queryFn: ({ signal }) => listAIProviders(signal),
  })

  const createMutation = useMutation({
    mutationFn: createAIModel,
    onSuccess: () => {
      message.success('模型创建成功')
      queryClient.invalidateQueries({ queryKey: ['ai-models'] })
      setModalOpen(false)
      form.resetFields()
    },
    onError: (error) => message.error(getErrorMessage(error, '模型创建失败')),
  })

  const updateMutation = useMutation({
    mutationFn: (values: Partial<AIModel>) => updateAIModel(editRecord!.id, values),
    onSuccess: () => {
      message.success('模型更新成功')
      queryClient.invalidateQueries({ queryKey: ['ai-models'] })
      setModalOpen(false)
      setEditRecord(null)
      form.resetFields()
    },
    onError: (error) => message.error(getErrorMessage(error, '模型更新失败')),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteAIModel(id),
    onSuccess: () => {
      message.success('模型删除成功')
      queryClient.invalidateQueries({ queryKey: ['ai-models'] })
    },
    onError: (error) => message.error(getErrorMessage(error, '模型删除失败')),
  })

  const handleAdd = () => {
    setEditRecord(null)
    form.resetFields()
    form.setFieldsValue({
      modelType: 'chat',
      maxInputTokens: 4096,
      supportsTools: true,
      supportsVision: false,
      supportsFileInput: false,
      enabled: true,
    })
    setModalOpen(true)
  }

  const handleEdit = (record: AIModel) => {
    setEditRecord(record)
    form.setFieldsValue(record)
    setModalOpen(true)
  }

  const providerOptions = providers.map((provider) => ({
    value: provider.id,
    label: provider.name,
  }))

  const columns = [
    {
      title: '模型名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: AIModel) => (
        <div className="app-table-stack">
          <span className="app-table-stack__main">{text}</span>
          <span className="app-table-stack__sub">
            {record.providerName} · <code>{record.modelCode}</code>
          </span>
        </div>
      ),
    },
    {
      title: '类型',
      dataIndex: 'modelType',
      key: 'modelType',
      width: 120,
      render: (value: string) => {
        const option = MODEL_TYPE_OPTIONS.find((item) => item.value === value)
        return <Tag>{option?.label || value}</Tag>
      },
    },
    {
      title: '最大输入',
      dataIndex: 'maxInputTokens',
      key: 'maxInputTokens',
      width: 110,
      align: 'center' as const,
      render: (value: number) => `${Math.max(1, Math.round(value / 1000))}K`,
    },
    {
      title: '能力',
      key: 'capabilities',
      width: 220,
      render: (_: unknown, record: AIModel) => (
        <Space size={[4, 4]} wrap>
          {record.supportsTools ? <Tag color="green">工具调用</Tag> : null}
          {record.supportsVision ? <Tag color="purple">视觉</Tag> : null}
          {record.supportsFileInput ? <Tag color="cyan">文件输入</Tag> : null}
          {record.supportsStreaming ? <Tag color="blue">流式</Tag> : null}
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      align: 'center' as const,
      render: (enabled: boolean) => (
        <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '停用'} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      align: 'center' as const,
      render: (_: unknown, record: AIModel) => (
        <div className="app-table-actions app-table-actions--icon">
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Popconfirm title="确定删除该模型吗？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ]

  return (
    <div className="app-data-console">
      <section className="app-data-console__filters app-data-console__filters--actions-only">
        <div className="app-data-console__filters-right">
          <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            添加模型
          </Button>
        </div>
      </section>

      <div className="app-data-console__table">
        <Table
          rowKey="id"
          columns={columns}
          dataSource={models}
          loading={isLoading}
          pagination={false}
        />
      </div>

      <Modal
        title={editRecord ? '编辑模型' : '添加模型'}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditRecord(null)
          form.resetFields()
        }}
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
          <Form.Item
            name="providerId"
            label="提供商"
            rules={[{ required: true, message: '请选择提供商' }]}
          >
            <Select options={providerOptions} />
          </Form.Item>
          <Form.Item
            name="name"
            label="模型名称"
            rules={[{ required: true, message: '请输入模型名称' }]}
          >
            <Input placeholder="例如 GPT-4.1" />
          </Form.Item>
          <Form.Item
            name="modelCode"
            label="模型标识"
            rules={[{ required: true, message: '请输入模型标识' }]}
          >
            <Input placeholder="例如 gpt-4.1" />
          </Form.Item>
          <Form.Item
            name="modelType"
            label="模型类型"
            rules={[{ required: true, message: '请选择模型类型' }]}
          >
            <Select options={MODEL_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="maxInputTokens" label="最大输入 Token 数">
            <InputNumber min={1024} max={1000000} step={1024} style={{ width: '100%' }} />
          </Form.Item>
          <Space size={16} wrap>
            <Form.Item name="supportsTools" label="支持工具调用" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="supportsVision" label="支持视觉" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="supportsFileInput" label="支持文件输入" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="supportsStreaming" label="支持流式" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  )
}

const RouteSettingsPanel: React.FC = () => {
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  const { data: routeSettings, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['ai-route-settings'],
    queryFn: ({ signal }) => getAIRouteSettings(signal),
  })
  const { data: models = [] } = useQuery({
    queryKey: ['ai-models'],
    queryFn: ({ signal }) => listAIModels(signal),
  })
  const { data: providers = [] } = useQuery({
    queryKey: ['ai-providers'],
    queryFn: ({ signal }) => listAIProviders(signal),
  })

  useEffect(() => {
    if (!routeSettings) {
      return
    }
    form.setFieldsValue({
      defaultChatModelId: routeSettings.defaultChatModelId,
      defaultDiagnoseModelId: routeSettings.defaultDiagnoseModelId,
      defaultVisionModelId: routeSettings.defaultVisionModelId,
      defaultImageGenerationModelId: routeSettings.defaultImageGenerationModelId,
      defaultFallbackProviderId: routeSettings.defaultFallbackProviderId,
      routingStrategy: routeSettings.routingStrategy || 'priority_first',
      allowFallback: routeSettings.allowFallback,
    })
  }, [form, routeSettings])

  const updateMutation = useMutation({
    mutationFn: updateAIRouteSettings,
    onSuccess: () => {
      message.success('路由设置更新成功')
      queryClient.invalidateQueries({ queryKey: ['ai-route-settings'] })
    },
    onError: (error) => message.error(getErrorMessage(error, '路由设置更新失败')),
  })

  const modelOptions = models
    .filter((item) => item.enabled)
    .map((item) => ({
      value: item.id,
      label: `${item.providerName} / ${item.name} (${item.modelCode})`,
    }))

  const providerOptions = providers
    .filter((item) => item.enabled)
    .map((item) => ({
      value: item.id,
      label: item.name,
    }))

  return (
    <div className="app-data-console">
      <section className="app-data-console__filters app-data-console__filters--actions-only">
        <div className="app-data-console__filters-right">
          <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
            刷新
          </Button>
          <Button type="primary" loading={updateMutation.isPending} onClick={() => form.submit()}>
            保存设置
          </Button>
        </div>
      </section>

      <Card loading={isLoading}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{ routingStrategy: 'priority_first', allowFallback: true }}
          onFinish={(values) => updateMutation.mutate(values)}
        >
          <Form.Item
            name="routingStrategy"
            label="路由策略"
            rules={[{ required: true, message: '请选择路由策略' }]}
          >
            <Select options={ROUTING_STRATEGY_OPTIONS} />
          </Form.Item>

          <Space size={16} wrap style={{ width: '100%' }}>
            <Form.Item name="defaultChatModelId" label="默认对话模型" style={{ minWidth: 320 }}>
              <Select allowClear options={modelOptions} />
            </Form.Item>
            <Form.Item name="defaultDiagnoseModelId" label="默认诊断模型" style={{ minWidth: 320 }}>
              <Select allowClear options={modelOptions} />
            </Form.Item>
            <Form.Item name="defaultVisionModelId" label="默认视觉模型" style={{ minWidth: 320 }}>
              <Select allowClear options={modelOptions.filter((item) => {
                const model = models.find((modelItem) => modelItem.id === item.value)
                return model?.supportsVision
              })} />
            </Form.Item>
            <Form.Item name="defaultImageGenerationModelId" label="默认图像生成模型" style={{ minWidth: 320 }}>
              <Select allowClear options={modelOptions.filter((item) => {
                const model = models.find((modelItem) => modelItem.id === item.value)
                return model?.supportsImageGeneration
              })} />
            </Form.Item>
            <Form.Item name="defaultFallbackProviderId" label="默认兜底提供商" style={{ minWidth: 320 }}>
              <Select allowClear options={providerOptions} />
            </Form.Item>
            <Form.Item name="allowFallback" label="允许自动兜底" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Card>
    </div>
  )
}

export default AiSettings
