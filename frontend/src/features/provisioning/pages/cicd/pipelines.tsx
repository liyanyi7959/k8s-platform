/**
 * 流水线管理 - CI/CD Pipeline 定义与编排
 */
import { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Drawer, Form, Input, InputNumber, message, Select, Space, Table, Tag, Tooltip } from 'antd'
import { PlusOutlined, PlayCircleOutlined, DeleteOutlined } from '@ant-design/icons'
import { AppPage, EmptyState, YamlEditor } from '@/components'
import type { ProColumns } from '@ant-design/pro-components'
import { createPipeline, getPipelines, triggerPipeline, type Pipeline } from '@/features/provisioning/api/cicd'

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

type BuilderStep = { key: string; name: string; plugin: string; script: string; repository: string; branch: string; path: string; command: string; image: string; tag: string; context: string }
type BuilderStage = { key: string; name: string; steps: BuilderStep[] }
const PLUGIN_OPTIONS = [{ value: 'git', label: 'Git 拉取代码' }, { value: 'bash', label: 'Bash 脚本' }, { value: 'kubectl', label: 'Kubectl 部署' }, { value: 'helm', label: 'Helm 发布' }, { value: 'docker-build', label: 'Docker 构建' }]
const DEFAULT_BUILDER: BuilderStage[] = [{ key: 'source', name: '代码拉取', steps: [{ key: 'checkout', name: 'Checkout', plugin: 'git', script: '', repository: '', branch: 'main', path: '/workspace/src', command: '', image: '', tag: '', context: '' }] }, { key: 'build', name: '构建', steps: [{ key: 'build-script', name: '构建脚本', plugin: 'bash', script: 'npm install\nnpm run build', repository: '', branch: '', path: '', command: '', image: '', tag: '', context: '' }] }]
const yamlScalar = (value: string) => JSON.stringify(value || '')
function builderToYaml(stages: BuilderStage[]) { const lines = ['stages:']; stages.forEach((stage) => { lines.push(`  - key: ${yamlScalar(stage.key)}`, `    name: ${yamlScalar(stage.name)}`, '    steps:'); stage.steps.forEach((step) => { lines.push(`      - key: ${yamlScalar(step.key)}`, `        name: ${yamlScalar(step.name)}`, `        plugin: ${yamlScalar(step.plugin)}`); if (step.plugin === 'git') { lines.push(`        repository: ${yamlScalar(step.repository)}`, `        branch: ${yamlScalar(step.branch)}`, `        path: ${yamlScalar(step.path)}`) } else if (step.plugin === 'bash') { lines.push('        script: |', ...step.script.split('\n').map((line) => `          ${line}`)) } else if (step.plugin === 'docker-build') { lines.push(`        image: ${yamlScalar(step.image)}`, `        tag: ${yamlScalar(step.tag)}`, `        context: ${yamlScalar(step.context)}`) } else { lines.push(`        command: ${yamlScalar(step.command)}`) } }) }); return lines.join('\n') }

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
    render: (_, record) => (
      <Space>
        <Tooltip title="执行">
          <a onClick={(event) => { event.stopPropagation(); triggerPipeline(record.id).then(() => message.success('流水线已触发')) }}><PlayCircleOutlined /></a>
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
void MOCK_PIPELINES

const PipelinesPage: React.FC = () => {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [form] = Form.useForm()
  const [pipelineYaml, setPipelineYaml] = useState(DEFAULT_PIPELINE_YAML)
  const [builderStages, setBuilderStages] = useState<BuilderStage[]>(DEFAULT_BUILDER)
  const [pipelines, setPipelines] = useState<PipelineRecord[]>([])
  const [loading, setLoading] = useState(false)
  const loadPipelines = () => {
    setLoading(true)
    getPipelines({ page: 1, pageSize: 100 }).then((res) => setPipelines((res.list || []).map((p: Pipeline) => ({ id: String(p.id), name: p.name, trigger: p.triggerType || p.trigger_type || 'manual', status: p.status, lastRun: p.lastRun || p.lastRunAt || '-', updatedAt: p.updatedAt || p.updated_at || '-' })))).finally(() => setLoading(false))
  }
  useEffect(() => { loadPipelines() }, [])

  const handleAdd = () => {
    form.resetFields()
    form.setFieldsValue({ trigger: 'push' })
    setPipelineYaml(DEFAULT_PIPELINE_YAML)
    setBuilderStages(DEFAULT_BUILDER)
    setDrawerOpen(true)
  }

  const handleSubmit = () => {
    form.validateFields().then((values) => {
      createPipeline({ ...values, configYaml: builderToYaml(builderStages) || pipelineYaml }).then(() => loadPipelines())
      message.success('流水线创建成功')
      setDrawerOpen(false)
      form.resetFields()
    })
  }

  const handleClose = () => {
    setDrawerOpen(false)
    form.resetFields()
  }

  const addStage = () => setBuilderStages((stages) => [...stages, { key: `stage-${stages.length + 1}`, name: `阶段 ${stages.length + 1}`, steps: [] }])
  const addStep = (stageIndex: number) => setBuilderStages((stages) => stages.map((stage, index) => index === stageIndex ? { ...stage, steps: [...stage.steps, { key: `step-${stage.steps.length + 1}`, name: `步骤 ${stage.steps.length + 1}`, plugin: 'bash', script: '', repository: '', branch: '', path: '', command: '', image: '', tag: '', context: '' }] } : stage))
  const updateStage = (stageIndex: number, patch: Partial<BuilderStage>) => setBuilderStages((stages) => stages.map((stage, index) => index === stageIndex ? { ...stage, ...patch } : stage))
  const updateStep = (stageIndex: number, stepIndex: number, patch: Partial<BuilderStep>) => setBuilderStages((stages) => stages.map((stage, index) => index === stageIndex ? { ...stage, steps: stage.steps.map((step, stepIndexValue) => stepIndexValue === stepIndex ? { ...step, ...patch } : step) } : stage))

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
          dataSource={pipelines}
          loading={loading}
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
          <Form.Item name="clusterId" label="执行集群" rules={[{ required: true, message: '请输入 Kubernetes 集群 ID' }]}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="namespace" label="执行命名空间" initialValue="cicd">
            <Input />
          </Form.Item>
          <Form.Item name="runnerImage" label="Runner 镜像" initialValue="alpine:3.20">
            <Input />
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
          <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 16 }}>
            <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}>
              <strong>插件步骤编排</strong>
              <Button size="small" icon={<PlusOutlined />} onClick={addStage}>添加阶段</Button>
            </Space>
            {builderStages.map((stage, stageIndex) => (
              <Card key={stage.key} size="small" title={<Input value={stage.name} onChange={(event) => updateStage(stageIndex, { name: event.target.value, key: event.target.value.toLowerCase().replace(/[^a-z0-9]+/g, '-') })} />} style={{ marginBottom: 12 }} extra={<Button type="text" danger icon={<DeleteOutlined />} onClick={() => setBuilderStages((stages) => stages.filter((_, index) => index !== stageIndex))} />}>
                {stage.steps.map((step, stepIndex) => (
                  <Card key={step.key} size="small" style={{ marginBottom: 8 }} bodyStyle={{ padding: 10 }}>
                    <Space direction="vertical" style={{ width: '100%' }} size={8}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Input value={step.name} placeholder="步骤名称" onChange={(event) => updateStep(stageIndex, stepIndex, { name: event.target.value, key: event.target.value.toLowerCase().replace(/[^a-z0-9]+/g, '-') })} />
                        <Select value={step.plugin} options={PLUGIN_OPTIONS} style={{ width: 150 }} onChange={(plugin) => updateStep(stageIndex, stepIndex, { plugin })} />
                        <Button danger icon={<DeleteOutlined />} onClick={() => setBuilderStages((stages) => stages.map((item, index) => index === stageIndex ? { ...item, steps: item.steps.filter((_, childIndex) => childIndex !== stepIndex) } : item))} />
                      </Space.Compact>
                      {step.plugin === 'git' ? <><Input placeholder="仓库地址" value={step.repository} onChange={(event) => updateStep(stageIndex, stepIndex, { repository: event.target.value })} /><Space.Compact style={{ width: '100%' }}><Input placeholder="分支" value={step.branch} onChange={(event) => updateStep(stageIndex, stepIndex, { branch: event.target.value })} /><Input placeholder="目录" value={step.path} onChange={(event) => updateStep(stageIndex, stepIndex, { path: event.target.value })} /></Space.Compact></> : step.plugin === 'bash' ? <Input.TextArea rows={3} placeholder="脚本内容" value={step.script} onChange={(event) => updateStep(stageIndex, stepIndex, { script: event.target.value })} /> : step.plugin === 'docker-build' ? <Space.Compact style={{ width: '100%' }}><Input placeholder="镜像仓库" value={step.image} onChange={(event) => updateStep(stageIndex, stepIndex, { image: event.target.value })} /><Input placeholder="Tag" value={step.tag} onChange={(event) => updateStep(stageIndex, stepIndex, { tag: event.target.value })} /><Input placeholder="构建上下文" value={step.context} onChange={(event) => updateStep(stageIndex, stepIndex, { context: event.target.value })} /></Space.Compact> : <Input placeholder="命令，例如 apply -f /workspace/src/k8s" value={step.command} onChange={(event) => updateStep(stageIndex, stepIndex, { command: event.target.value })} />}
                    </Space>
                  </Card>
                ))}
                <Button type="dashed" block icon={<PlusOutlined />} onClick={() => addStep(stageIndex)}>添加插件步骤</Button>
              </Card>
            ))}
          </div>
        </Form>
        <div style={{ marginTop: 16 }}>
          <div style={{ fontSize: 14, fontWeight: 500, marginBottom: 8 }}>流水线配置</div>
          <YamlEditor
            value={builderToYaml(builderStages)}
            readOnly
            height={300}
          />
        </div>
      </Drawer>
    </AppPage>
  )
}

export default PipelinesPage
