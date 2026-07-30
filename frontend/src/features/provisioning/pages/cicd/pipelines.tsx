/**
 * 流水线管理 - CI/CD Pipeline 定义与编排
 */
import { useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Drawer, Form, Input, message, Select, Space, Table, Tag, Tooltip } from 'antd'
import { PlusOutlined, PlayCircleOutlined } from '@ant-design/icons'
import { AppPage, EmptyState, YamlEditor } from '@/components'
import type { ProColumns } from '@ant-design/pro-components'

interface PipelineRecord {
  id: string
  name: string
  trigger: string
  status: 'idle' | 'running' | 'success' | 'failed'
  lastRun: string
  updatedAt: string
}

const STATUS_MAP: Record<string, { color: string; text: string }> = {
  idle: { color: 'default', text: '未执行' },
  running: { color: 'processing', text: '执行中' },
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
}

const TRIGGER_OPTIONS = [
  { value: 'push', label: '代码推送' },
  { value: 'schedule', label: '定时触发' },
  { value: 'manual', label: '手动执行' },
]

const DEFAULT_PIPELINE_YAML = `# 流水线定义
stages:
  - name: 代码检查
    steps:
      - name: ESLint
        run: npm run lint
      - name: 类型检查
        run: npm run type-check
  - name: 构建
    steps:
      - name: 安装依赖
        run: npm install
      - name: 构建产物
        run: npm run build
  - name: 镜像构建
    steps:
      - name: 构建镜像
        run: docker build -t app:latest .
      - name: 推送镜像
        run: docker push registry.example.com/app:latest
`

const columns: ProColumns<PipelineRecord>[] = [
  {
    title: '流水线名称',
    dataIndex: 'name',
    key: 'name',
    render: (_, record) => (
      <Space>
        <strong>{record.name}</strong>
      </Space>
    ),
  },
  {
    title: '触发方式',
    dataIndex: 'trigger',
    key: 'trigger',
    width: 140,
    render: (_, record) => <Tag>{record.trigger}</Tag>,
  },
  {
    title: '最近状态',
    dataIndex: 'status',
    key: 'status',
    width: 100,
    render: (_, record) => {
      const cfg = STATUS_MAP[record.status] ?? STATUS_MAP.idle!
      return <Tag color={cfg.color}>{cfg.text}</Tag>
    },
  },
  {
    title: '最近执行',
    dataIndex: 'lastRun',
    key: 'lastRun',
    width: 180,
    ellipsis: true,
  },
  {
    title: '更新时间',
    dataIndex: 'updatedAt',
    key: 'updatedAt',
    width: 180,
    ellipsis: true,
  },
  {
    title: '操作',
    key: 'action',
    width: 160,
    render: () => (
      <Space>
        <Tooltip title="执行">
          <a><PlayCircleOutlined /></a>
        </Tooltip>
        <a>编辑</a>
        <a style={{ color: '#dc2626' }}>删除</a>
      </Space>
    ),
  },
]

const MOCK_PIPELINES: PipelineRecord[] = [
  { id: '1', name: 'frontend-ci', trigger: '代码推送', status: 'success', lastRun: '2026-07-26 14:32', updatedAt: '2026-07-20 14:22' },
  { id: '2', name: 'backend-deploy', trigger: '定时触发', status: 'failed', lastRun: '2026-07-26 12:00', updatedAt: '2026-07-22 09:15' },
  { id: '3', name: 'api-gateway-build', trigger: '手动执行', status: 'success', lastRun: '2026-07-26 10:15', updatedAt: '2026-07-21 16:30' },
  { id: '4', name: 'helm-chart-release', trigger: '代码推送', status: 'running', lastRun: '2026-07-26 09:30', updatedAt: '2026-07-19 11:00' },
  { id: '5', name: 'db-migration', trigger: '定时触发', status: 'idle', lastRun: '2026-07-25 18:22', updatedAt: '2026-07-18 08:45' },
  { id: '6', name: 'nginx-deploy', trigger: '代码推送', status: 'success', lastRun: '2026-07-25 10:05', updatedAt: '2026-07-17 14:20' },
]

const PipelinesPage: React.FC = () => {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [form] = Form.useForm()
  const [pipelineYaml, setPipelineYaml] = useState(DEFAULT_PIPELINE_YAML)

  const handleAdd = () => {
    form.resetFields()
    form.setFieldsValue({ trigger: 'push' })
    setPipelineYaml(DEFAULT_PIPELINE_YAML)
    setDrawerOpen(true)
  }

  const handleSubmit = () => {
    form.validateFields().then(() => {
      message.success('流水线创建成功')
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
          <Space>
            <Input.Search placeholder="搜索流水线名称" style={{ width: 240 }} allowClear />
            <Select
              placeholder="触发方式"
              style={{ width: 140 }}
              allowClear
              options={TRIGGER_OPTIONS}
            />
          </Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>创建流水线</Button>
        </div>
        <Table<PipelineRecord>
          rowKey="id"
          columns={columns as any}
          dataSource={MOCK_PIPELINES}
          pagination={false}
          onRow={(record) => ({ onClick: () => history.push(`/cicd/pipelines/${record.id}`), style: { cursor: 'pointer' } })}
          locale={{
            emptyText: <EmptyState description="暂无流水线，点击「创建流水线」开始编排 CI/CD 流程" />,
          }}
        />
      </Card>

      {/* 创建流水线 Drawer */}
      <Drawer
        title="创建流水线"
        open={drawerOpen}
        onClose={handleClose}
        width={520}
        extra={
          <Space>
            <Button onClick={handleClose}>取消</Button>
            <Button type="primary" onClick={handleSubmit}>保存</Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="流水线名称" rules={[{ required: true, message: '请输入流水线名称' }]}>
            <Input placeholder="如 frontend-ci" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="流水线用途说明" />
          </Form.Item>
          <Form.Item name="trigger" label="触发方式" rules={[{ required: true, message: '请选择触发方式' }]}>
            <Select options={TRIGGER_OPTIONS} />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.trigger !== cur.trigger}>
            {({ getFieldValue }) => {
              const trigger = getFieldValue('trigger')
              if (trigger === 'push') {
                return (
                  <Form.Item name="branches" label="触发分支" extra="多个分支用逗号分隔">
                    <Input placeholder="main, develop" />
                  </Form.Item>
                )
              }
              if (trigger === 'schedule') {
                return (
                  <Form.Item name="cron" label="CRON 表达式" rules={[{ required: true, message: '请输入 CRON 表达式' }]} extra="如：0 2 * * *（每天凌晨2点）">
                    <Input placeholder="0 2 * * *" />
                  </Form.Item>
                )
              }
              return null
            }}
          </Form.Item>
        </Form>
        <div style={{ marginTop: 16 }}>
          <div style={{ fontSize: 14, fontWeight: 500, marginBottom: 8 }}>流水线配置</div>
          <YamlEditor
            value={pipelineYaml}
            onChange={setPipelineYaml}
            height={300}
          />
        </div>
      </Drawer>
    </AppPage>
  )
}

export default PipelinesPage
