/**
 * 流水线详情 - 展示流水线配置、阶段步骤与执行历史
 */
import React, { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, message, Segmented, Space, Table, Tag, Tooltip, Typography } from 'antd'
import {
  ArrowLeftOutlined,
  PlayCircleOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  BuildOutlined,
  RocketOutlined,
} from '@ant-design/icons'
import { AppPage, EmptyState, YamlEditor } from '@/components'
import { getPipeline, getRuns, triggerPipeline } from '@/features/provisioning/api/cicd'
import { yamlToBuilder } from './pipeline-editor'

const { Text } = Typography

// 加载失败或新建状态下使用的空状态
const DEFAULT_PIPELINE = {
  id: '',
  name: '',
  description: '',
  trigger: 'manual',
  status: 'idle',
  createdAt: '-',
  updatedAt: '-',
  lastRun: '-',
}

const STATUS_META: Record<string, { color: string; text: string }> = {
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  running: { color: 'processing', text: '执行中' },
  queued: { color: 'processing', text: '排队中' },
  canceled: { color: 'default', text: '已取消' },
  idle: { color: 'default', text: '未执行' },
}

const TRIGGER_LABELS: Record<string, string> = { push: '代码推送', schedule: '定时触发', manual: '手动执行' }
const PLUGIN_STAGE_ICONS: Record<string, React.ReactNode> = {
  git: <CheckCircleOutlined />,
  bash: <BuildOutlined />,
  kubectl: <RocketOutlined />,
  helm: <RocketOutlined />,
  'docker-build': <RocketOutlined />,
}

const PipelineDetailPage: React.FC = () => {
  const [pipelineData, setPipelineData] = useState<any>(null)
  const [runPage, setRunPage] = useState<{ list: any[]; total: number } | null>(null)
  const [detailView, setDetailView] = useState<'graph' | 'yaml' | 'runs'>('graph')
  const pipelineID = history.location.pathname.split('/').pop() || ''
  useEffect(() => {
    if (!pipelineID) return
    Promise.all([getPipeline(pipelineID), getRuns({ pipelineId: pipelineID, page: 1, pageSize: 100 })])
      .then(([pipeline, runs]) => { setPipelineData(pipeline); setRunPage(runs) })
      .catch((error) => message.error(error instanceof Error ? error.message : '加载流水线详情失败'))
  }, [pipelineID])
  const PIPELINE = pipelineData ? {
    ...DEFAULT_PIPELINE,
    ...pipelineData,
    trigger: TRIGGER_LABELS[pipelineData.triggerType || pipelineData.trigger_type] || pipelineData.triggerType || pipelineData.trigger_type || DEFAULT_PIPELINE.trigger,
    configYaml: pipelineData.configYaml || pipelineData.config_yaml || 'stages: []',
  } : DEFAULT_PIPELINE
  const pipelineStatus = STATUS_META[PIPELINE.status] ?? STATUS_META.idle!
  const stages = yamlToBuilder(PIPELINE.configYaml, false)
  const recentRuns = runPage?.list || []
  const totalRuns = runPage?.total ?? '-'
  const successRuns = recentRuns.filter((run) => run.status === 'success').length
  const successRate = runPage?.total ? `${((successRuns / Math.max(recentRuns.length, 1)) * 100).toFixed(1)}%` : '-'
  const displayRuns = recentRuns.map((run: any) => {
    const triggerType = run.triggerType || run.trigger_type || 'manual'
    return { ...run, trigger: TRIGGER_LABELS[triggerType] || triggerType, duration: '-', startedAt: run.startedAt || run.started_at || '-' }
  })
  const latestRun = recentRuns[0] as any
  const lastRun = latestRun?.startedAt || latestRun?.started_at || PIPELINE.lastRun || '-'

  return (
    <AppPage className="cicd-detail-page">
      <div className="cicd-detail-shell">
        <header className="cicd-detail-toolbar">
          <div className="cicd-detail-toolbar__identity">
            <Tooltip title="返回流水线列表"><Button type="text" aria-label="返回流水线列表" icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/pipelines')} /></Tooltip>
            <div><Typography.Text type="secondary">CI/CD</Typography.Text><Typography.Title level={3}>{PIPELINE.name}</Typography.Title></div>
            <Tag color={pipelineStatus.color}>{pipelineStatus.text}</Tag>
          </div>
          <Space>
            <Segmented value={detailView} onChange={(value) => setDetailView(value as 'graph' | 'yaml' | 'runs')} options={[{ label: '阶段', value: 'graph' }, { label: 'YAML', value: 'yaml' }, { label: '执行记录', value: 'runs' }]} />
            <Tooltip title="执行流水线"><Button type="primary" aria-label="执行流水线" icon={<PlayCircleOutlined />} onClick={() => triggerPipeline(pipelineID).then(() => message.success('流水线已触发'))} /></Tooltip>
            <Tooltip title="编辑流水线"><Button aria-label="编辑流水线" icon={<EditOutlined />} onClick={() => history.push(`/cicd/pipelines/${pipelineID}/edit`)} /></Tooltip>
            <Tooltip title="删除流水线"><Button danger aria-label="删除流水线" icon={<DeleteOutlined />} /></Tooltip>
          </Space>
        </header>

        <section className="cicd-detail-summary">
          <div><span>触发方式</span><strong>{PIPELINE.trigger}</strong></div>
          <div><span>最近执行</span><strong>{lastRun}</strong></div>
          <div><span>执行次数</span><strong>{totalRuns}</strong></div>
          <div><span>成功率</span><strong className="cicd-detail-summary__success">{successRate}</strong></div>
          <div><span>更新时间</span><strong>{PIPELINE.updatedAt}</strong></div>
        </section>

        <main className="cicd-detail-content">
          {detailView === 'graph' && <section className="cicd-detail-panel">
            <div className="cicd-detail-panel__heading"><Typography.Title level={4}>流水线阶段</Typography.Title><Text type="secondary">按顺序执行</Text></div>
            <div className="cicd-detail-stage-flow">
              {stages.length ? stages.map((stage: any, idx: number) => <React.Fragment key={stage.key}>
                <article className="cicd-detail-stage">
                  <header><span className="cicd-detail-stage__icon">{PLUGIN_STAGE_ICONS[stage.steps?.[0]?.plugin] || <BuildOutlined />}</span><div><strong>{stage.name}</strong><Text code>{stage.key}</Text></div></header>
                  <div className="cicd-detail-stage__steps">{(stage.steps || []).map((step: any) => <div key={step.key}><span>{step.name}</span><Text type="secondary">{step.plugin}</Text></div>)}</div>
                </article>
                {idx < stages.length - 1 && <span className="cicd-detail-stage__connector"><ArrowLeftOutlined rotate={180} /></span>}
              </React.Fragment>) : <EmptyState description="流水线未配置阶段" />}
            </div>
          </section>}

          {detailView === 'yaml' && <section className="cicd-detail-panel cicd-detail-yaml"><div className="cicd-detail-panel__heading"><Typography.Title level={4}>流水线 YAML</Typography.Title><Tag color="green">只读</Tag></div><YamlEditor readOnly value={PIPELINE.configYaml || 'stages: []'} height={620} /></section>}

          {detailView === 'runs' && <section className="cicd-detail-panel"><div className="cicd-detail-panel__heading"><Typography.Title level={4}>最近执行</Typography.Title><Tooltip title="查看全部执行记录"><Button type="text" aria-label="查看全部执行记录" icon={<ArrowLeftOutlined rotate={180} />} onClick={() => history.push('/cicd/runs')} /></Tooltip></div><Table rowKey="id" dataSource={displayRuns} pagination={false} size="small" onRow={(record) => ({ onClick: () => history.push(`/cicd/runs/${record.id}`), style: { cursor: 'pointer' } })} columns={[{ title: '执行 ID', dataIndex: 'id', key: 'id', width: 100, render: (id: string) => <Text code>#{id}</Text> }, { title: '状态', dataIndex: 'status', key: 'status', width: 110, render: (status: string) => <Tag color={STATUS_META[status]?.color}>{STATUS_META[status]?.text}</Tag> }, { title: '触发者', dataIndex: 'trigger', key: 'trigger', width: 110 }, { title: '耗时', dataIndex: 'duration', key: 'duration', width: 110 }, { title: '开始时间', dataIndex: 'startedAt', key: 'startedAt' }]} /></section>}
        </main>
      </div>
    </AppPage>
  )
}

export default PipelineDetailPage
