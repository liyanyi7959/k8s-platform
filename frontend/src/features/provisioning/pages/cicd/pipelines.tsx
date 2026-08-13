import React, { useCallback, useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Input, message, Modal, Select, Space, Table, Tag, Tooltip, Typography } from 'antd'
import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  FilterOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import {
  createPipeline,
  deletePipeline,
  getPipeline,
  getPipelines,
  triggerPipeline,
  type Pipeline,
} from '@/features/provisioning/api/cicd'
import { yamlToBuilder } from './pipeline-editor'

const { Text } = Typography

type PipelineStatus = 'idle' | 'running' | 'success' | 'failed' | 'canceled'

interface PipelineRecord {
  id: string
  name: string
  description: string
  trigger: string
  status: PipelineStatus
  cluster: string
  namespace: string
  stages: number
  steps: number
  lastRun: string
  updatedAt: string
  branches: string
  configYaml?: string
}

const STATUS_MAP: Record<string, { color: string; text: string }> = {
  idle: { color: 'default', text: '未执行' },
  running: { color: 'processing', text: '执行中' },
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  canceled: { color: 'default', text: '已取消' },
}

const TRIGGER_OPTIONS = [
  { value: 'push', label: '代码推送' },
  { value: 'schedule', label: '定时触发' },
  { value: 'manual', label: '手动执行' },
]

const STATUS_OPTIONS = Object.entries(STATUS_MAP).map(([value, item]) => ({ value, label: item.text }))

const formatDate = (value?: string) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false }).replace(/\//g, '-')
}

const toRecord = (pipeline: Pipeline & Record<string, any>): PipelineRecord => {
  const configYaml = pipeline.configYaml || pipeline.config_yaml || ''
  const stages = yamlToBuilder(configYaml, false)
  const trigger = pipeline.triggerType || pipeline.trigger_type || 'manual'
  return {
    id: String(pipeline.id),
    name: pipeline.name || '未命名流水线',
    description: pipeline.description || '',
    trigger,
    status: (pipeline.status || 'idle') as PipelineStatus,
    cluster: pipeline.clusterName || pipeline.cluster_name || (pipeline.clusterId ? String(pipeline.clusterId) : '-'),
    namespace: pipeline.namespace || '-',
    stages: stages.length,
    steps: stages.reduce((total, stage) => total + stage.steps.length, 0),
    lastRun: formatDate(pipeline.lastRunAt || pipeline.last_run_at || pipeline.lastRun),
    updatedAt: formatDate(pipeline.updatedAt || pipeline.updated_at),
    branches: pipeline.branches || '',
    configYaml,
  }
}

const PipelinesPage: React.FC = () => {
  const [pipelines, setPipelines] = useState<PipelineRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [trigger, setTrigger] = useState<string>()
  const [status, setStatus] = useState<string>()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)

  const loadPipelines = useCallback(() => {
    setLoading(true)
    getPipelines({ page, pageSize, keyword: keyword.trim() || undefined, triggerType: trigger, status })
      .then((res) => {
        setPipelines((res.list || []).map((pipeline) => toRecord(pipeline as Pipeline & Record<string, any>)))
        setTotal(res.total || 0)
      })
      .catch((error) => message.error(error instanceof Error ? error.message : '加载流水线失败'))
      .finally(() => setLoading(false))
  }, [keyword, page, pageSize, status, trigger])

  useEffect(() => { loadPipelines() }, [loadPipelines])

  const runPipeline = (record: PipelineRecord) => {
    Modal.confirm({
      title: `运行流水线“${record.name}”？`,
      content: '将按当前配置创建 Kubernetes Job 执行。',
      okText: '运行',
      cancelText: '取消',
      onOk: () => triggerPipeline(record.id).then(() => { message.success('流水线已触发'); loadPipelines() }),
    })
  }

  const copyPipeline = async (record: PipelineRecord) => {
    try {
      const source: any = await getPipeline(record.id)
      await createPipeline({
        name: `${record.name} - 副本`,
        description: source.description || record.description,
        triggerType: source.triggerType || source.trigger_type || 'manual',
        branches: source.branches || record.branches,
        cron: source.cron,
        clusterId: source.clusterId,
        namespace: source.namespace || record.namespace,
        runnerImage: source.runnerImage || source.runner_image,
        configYaml: source.configYaml || source.config_yaml || record.configYaml,
      })
      message.success('流水线已复制')
      loadPipelines()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '复制流水线失败')
    }
  }

  const removePipeline = (record: PipelineRecord) => {
    Modal.confirm({
      title: `删除流水线“${record.name}”？`,
      content: '删除后不可恢复，相关执行记录也将无法从该流水线进入。',
      okText: '删除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: () => deletePipeline(record.id).then(() => { message.success('流水线已删除'); loadPipelines() }),
    })
  }

  const columns = [
    {
      title: '流水线名称',
      dataIndex: 'name',
      key: 'name',
      width: 250,
      render: (_: unknown, record: PipelineRecord) => <div className="cicd-pipeline-name"><strong>{record.name}</strong><Text type="secondary" ellipsis={{ tooltip: record.description }}>{record.description || '暂无描述'}</Text></div>,
    },
    { title: '触发方式', dataIndex: 'trigger', key: 'trigger', width: 120, render: (value: string) => <Tag>{TRIGGER_OPTIONS.find((item) => item.value === value)?.label || value}</Tag> },
    { title: '状态', dataIndex: 'status', key: 'status', width: 95, render: (value: string) => <Tag color={STATUS_MAP[value]?.color}>{STATUS_MAP[value]?.text || value}</Tag> },
    { title: '编排规模', key: 'scale', width: 125, render: (_: unknown, record: PipelineRecord) => <Text>{record.stages} 阶段 · {record.steps} 步骤</Text> },
    { title: '执行环境', key: 'environment', width: 155, render: (_: unknown, record: PipelineRecord) => <div className="cicd-pipeline-environment"><Text>{record.cluster}</Text><Text type="secondary">ns/{record.namespace}</Text></div> },
    { title: '最近执行', dataIndex: 'lastRun', key: 'lastRun', width: 145, render: (value: string) => <Text type="secondary">{value}</Text> },
    { title: '更新时间', dataIndex: 'updatedAt', key: 'updatedAt', width: 145, render: (value: string) => <Text type="secondary">{value}</Text> },
    {
      title: '操作', key: 'action', width: 180,
      render: (_: unknown, record: PipelineRecord) => <Space size={0} onClick={(event) => event.stopPropagation()}>
        <Tooltip title="运行"><Button type="text" aria-label="运行流水线" icon={<PlayCircleOutlined />} onClick={() => runPipeline(record)} /></Tooltip>
        <Tooltip title="查看详情"><Button type="text" aria-label="查看流水线详情" icon={<EyeOutlined />} onClick={() => history.push(`/cicd/pipelines/${record.id}`)} /></Tooltip>
        <Tooltip title="编辑"><Button type="text" aria-label="编辑流水线" icon={<EditOutlined />} onClick={() => history.push(`/cicd/pipelines/${record.id}/edit`)} /></Tooltip>
        <Tooltip title="复制"><Button type="text" aria-label="复制流水线" icon={<CopyOutlined />} onClick={() => copyPipeline(record)} /></Tooltip>
        <Tooltip title="删除"><Button type="text" danger aria-label="删除流水线" icon={<DeleteOutlined />} onClick={() => removePipeline(record)} /></Tooltip>
      </Space>,
    },
  ]

  return (
    <AppPage className="cicd-pipelines-page">
      <Card className="cicd-pipelines-card" bordered={false}>
        <div className="cicd-pipelines-toolbar">
          <Space wrap>
            <Input.Search allowClear value={keyword} onChange={(event) => { setKeyword(event.target.value); setPage(1) }} onSearch={() => { setPage(1); loadPipelines() }} placeholder="搜索流水线名称或描述" style={{ width: 290 }} />
            <Select allowClear value={trigger} onChange={(value) => { setTrigger(value); setPage(1) }} placeholder="触发方式" options={TRIGGER_OPTIONS} style={{ width: 140 }} />
            <Select allowClear value={status} onChange={(value) => { setStatus(value); setPage(1) }} placeholder="状态" options={STATUS_OPTIONS} style={{ width: 120 }} />
            <Tooltip title="刷新"><Button aria-label="刷新流水线列表" icon={<ReloadOutlined />} onClick={loadPipelines} /></Tooltip>
            <Tooltip title="筛选条件"><Button aria-label="清空筛选条件" icon={<FilterOutlined />} onClick={() => { setKeyword(''); setTrigger(undefined); setStatus(undefined); setPage(1) }} /></Tooltip>
          </Space>
          <Tooltip title="创建流水线"><Button type="primary" aria-label="创建流水线" icon={<PlusOutlined />} onClick={() => history.push('/cicd/pipelines/new')} /></Tooltip>
        </div>
        <Table<PipelineRecord>
          rowKey="id"
          columns={columns as any}
          dataSource={pipelines}
          loading={loading}
          pagination={{ current: page, pageSize, total, showSizeChanger: true, showTotal: (count) => `共 ${count} 条`, onChange: (nextPage, nextSize) => { setPage(nextPage); if (nextSize !== pageSize) { setPageSize(nextSize); setPage(1) } } }}
          onRow={(record) => ({ onClick: () => history.push(`/cicd/pipelines/${record.id}`), style: { cursor: 'pointer' } })}
          locale={{ emptyText: <EmptyState description="暂无流水线，创建一条流水线开始编排 CI/CD 流程" /> }}
        />
      </Card>
    </AppPage>
  )
}

export default PipelinesPage
