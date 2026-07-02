import React, { useState } from 'react'
import { PlusOutlined } from '@ant-design/icons'
import { Badge, Button, Card, Form, Input, InputNumber, message, Modal, Select, Space, Switch, Table, Tabs, Tag } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import {
  createAIModel,
  createAIProvider,
  listAIModels,
  listAIProviders,
  updateAIModel,
  updateAIProvider,
} from '@/services/ai'
import type { AIModel, AIProvider } from '@/types'

const PROVIDER_TYPE_OPTIONS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'azure-openai', label: 'Azure OpenAI' },
  { value: 'ollama', label: 'Ollama' },
  { value: 'qwen', label: '通义千问' },
]

const MODEL_TYPE_OPTIONS = [
  { value: 'chat', label: '对话模型' },
  { value: 'vision', label: '视觉模型' },
  { value: 'embedding', label: '嵌入模型' },
  { value: 'image', label: '图像生成' },
]

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

  const { data: providers, isLoading } = useQuery({
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
  })

  const handleAdd = () => {
    setEditRecord(null)
    form.resetFields()
    form.setFieldsValue({ priority: 1, enabled: true })
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
      render: (text: string) => <strong>{text}</strong>,
    },
    {
      title: '类型',
      dataIndex: 'providerType',
      key: 'providerType',
      width: 120,
      render: (value: string) => {
        const option = PROVIDER_TYPE_OPTIONS.find((item) => item.value === value)
        return <Tag color="blue">{option?.label || value}</Tag>
      },
    },
    {
      title: 'Base URL',
      dataIndex: 'baseUrl',
      key: 'baseUrl',
      ellipsis: true,
    },
    {
      title: 'API Key',
      dataIndex: 'hasApiKey',
      key: 'hasApiKey',
      width: 100,
      render: (hasApiKey: boolean) =>
        hasApiKey ? <Badge status="success" text="已配置" /> : <Badge status="default" text="未配置" />,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sorter: (a: AIProvider, b: AIProvider) => a.priority - b.priority,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) => <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '禁用'} />,
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: AIProvider) => <a onClick={() => handleEdit(record)}>编辑</a>,
    },
  ]

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          添加提供商
        </Button>
      </div>

      <Table rowKey="id" columns={columns} dataSource={providers} loading={isLoading} pagination={false} />

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
          <Form.Item name="name" label="提供商名称" rules={[{ required: true, message: '请输入提供商名称' }]}>
            <Input placeholder="如 OpenAI Production" />
          </Form.Item>
          <Form.Item name="providerType" label="提供商类型" rules={[{ required: true, message: '请选择提供商类型' }]}>
            <Select options={PROVIDER_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="baseUrl" label="Base URL" rules={[{ required: true, message: '请输入 Base URL' }]}>
            <Input placeholder="https://api.openai.com/v1" />
          </Form.Item>
          <Form.Item name="apiKey" label="API Key" extra={editRecord ? '留空表示不修改' : ''}>
            <Input.Password placeholder="sk-..." />
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

const ModelPanel: React.FC = () => {
  const [modalOpen, setModalOpen] = useState(false)
  const [editRecord, setEditRecord] = useState<AIModel | null>(null)
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  const { data: models, isLoading } = useQuery({
    queryKey: ['ai-models'],
    queryFn: ({ signal }) => listAIModels(signal),
  })

  const { data: providers } = useQuery({
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
  })

  const handleAdd = () => {
    setEditRecord(null)
    form.resetFields()
    form.setFieldsValue({
      modelType: 'chat',
      maxInputTokens: 4096,
      supportsTools: true,
      supportsVision: false,
      enabled: true,
    })
    setModalOpen(true)
  }

  const handleEdit = (record: AIModel) => {
    setEditRecord(record)
    form.setFieldsValue(record)
    setModalOpen(true)
  }

  const providerOptions = providers?.map((provider) => ({ value: provider.id, label: provider.name })) || []

  const columns = [
    {
      title: '模型名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <strong>{text}</strong>,
    },
    {
      title: '模型标识',
      dataIndex: 'modelCode',
      key: 'modelCode',
      render: (text: string) => <code>{text}</code>,
    },
    {
      title: '提供商',
      dataIndex: 'providerName',
      key: 'providerName',
      render: (text: string) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: '类型',
      dataIndex: 'modelType',
      key: 'modelType',
      width: 100,
      render: (value: string) => {
        const option = MODEL_TYPE_OPTIONS.find((item) => item.value === value)
        return <Tag>{option?.label || value}</Tag>
      },
    },
    {
      title: '最大输入',
      dataIndex: 'maxInputTokens',
      key: 'maxInputTokens',
      width: 100,
      render: (value: number) => `${(value / 1000).toFixed(0)}K`,
    },
    {
      title: '能力',
      key: 'capabilities',
      width: 160,
      render: (_: unknown, record: AIModel) => (
        <Space size={4}>
          {record.supportsTools && <Tag color="green">工具调用</Tag>}
          {record.supportsVision && <Tag color="purple">视觉</Tag>}
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) => <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '禁用'} />,
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: AIModel) => <a onClick={() => handleEdit(record)}>编辑</a>,
    },
  ]

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          添加模型
        </Button>
      </div>

      <Table rowKey="id" columns={columns} dataSource={models} loading={isLoading} pagination={false} />

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
          <Form.Item name="providerId" label="提供商" rules={[{ required: true, message: '请选择提供商' }]}>
            <Select options={providerOptions} />
          </Form.Item>
          <Form.Item name="name" label="模型名称" rules={[{ required: true, message: '请输入模型名称' }]}>
            <Input placeholder="如 GPT-4.1" />
          </Form.Item>
          <Form.Item name="modelCode" label="模型标识" rules={[{ required: true, message: '请输入模型标识' }]}>
            <Input placeholder="如 gpt-4.1" />
          </Form.Item>
          <Form.Item name="modelType" label="模型类型" rules={[{ required: true, message: '请选择模型类型' }]}>
            <Select options={MODEL_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="maxInputTokens" label="最大输入 Token 数">
            <InputNumber min={1024} max={1000000} step={1024} style={{ width: '100%' }} />
          </Form.Item>
          <Space>
            <Form.Item name="supportsTools" label="支持工具调用" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="supportsVision" label="支持视觉" valuePropName="checked">
              <Switch />
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

export default AiSettings
