/**
 * 执行详情 - 展示执行概览、阶段时间线与实时日志
 */
import React, { useCallback, useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Descriptions, message, Modal, Space, Spin, Steps, Tag, Tooltip, Typography } from 'antd'
import {
  ArrowLeftOutlined,
  ReloadOutlined,
  StopOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import { AppPage, TerminalCodeBlock } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'
import { cancelRun, getRun, triggerPipeline, type Run } from '@/features/provisioning/api/cicd'

const { Text } = Typography

interface StageRun {
  key: string
  name: string
  status: 'success' | 'failed' | 'running' | 'pending'
  duration: string
  logs: string
}

const STAGE_STATUS_ICON: Record<string, React.ReactNode> = {
  success: <CheckCircleOutlined style={{ color: DESIGN_COLORS.success }} />,
  failed: <CloseCircleOutlined style={{ color: DESIGN_COLORS.danger }} />,
  running: <LoadingOutlined style={{ color: DESIGN_COLORS.primary }} />,
  pending: <ClockCircleOutlined style={{ color: DESIGN_COLORS.textMuted }} />,
}

const STATUS_META: Record<string, { color: string; text: string }> = {
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  running: { color: 'processing', text: '执行中' },
  pending: { color: 'default', text: '等待中' },
  cancelled: { color: 'warning', text: '已取消' },
}

/** 计算时长：< 60s 显示 "x.xs"，否则 "x.xm" */
const calcDuration = (start?: string, end?: string): string => {
  if (!start || !end) return '-'
  const startTime = new Date(start).getTime()
  const endTime = new Date(end).getTime()
  if (isNaN(startTime) || isNaN(endTime)) return '-'
  const diff = (endTime - startTime) / 1000
  if (diff < 0) return '-'
  if (diff < 60) return diff.toFixed(1) + 's'
  return (diff / 60).toFixed(1) + 'm'
}

const RunDetailPage: React.FC = () => {
  const [activeStage, setActiveStage] = useState(0)
  const [runData, setRunData] = useState<Run | null>(null)
  const [loading, setLoading] = useState(true)
  const runID = history.location.pathname.split('/').pop() || ''

  const fetchRun = useCallback(() => {
    if (!runID) return
    setLoading(true)
    getRun(runID)
      .then(setRunData)
      .catch(() => message.error('获取执行详情失败'))
      .finally(() => setLoading(false))
  }, [runID])

  useEffect(() => { fetchRun() }, [fetchRun])

  const handleCancel = () => {
    Modal.confirm({
      title: '取消执行',
      content: '确定要取消此次执行吗？',
      okText: '确定',
      cancelText: '取消',
      onOk: () => {
        cancelRun(runID)
          .then(() => { message.success('已发送取消请求'); fetchRun() })
          .catch(() => message.error('取消执行失败'))
      },
    })
  }

  const handleRerun = () => {
    const pipelineId = runData?.pipelineId ?? runData?.pipelineID
    if (!pipelineId) { message.error('缺少流水线信息'); return }
    Modal.confirm({
      title: '重新执行',
      content: '确定要重新触发此流水线吗？',
      okText: '确定',
      cancelText: '取消',
      onOk: () => {
        triggerPipeline(pipelineId)
          .then(() => message.success('已触发重新执行'))
          .catch(() => message.error('触发失败'))
      },
    })
  }

  if (loading && !runData) {
    return (
      <AppPage keepHeaderTitle title="执行详情">
        <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
          <Spin />
        </div>
      </AppPage>
    )
  }

  if (!runData) {
    return (
      <AppPage keepHeaderTitle title="执行详情">
        <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
          <Text type="secondary">未找到执行记录</Text>
        </div>
      </AppPage>
    )
  }

  const run = {
    id: runData.id,
    pipeline: runData.pipelineName || runData.pipeline_name || '-',
    pipelineId: runData.pipelineId ?? runData.pipelineID,
    status: runData.status,
    duration: calcDuration(runData.startedAt || runData.started_at, runData.finishedAt || runData.finished_at),
    trigger: runData.triggerType || runData.trigger_type || 'manual',
    startedAt: runData.startedAt || runData.started_at || '-',
    finishedAt: runData.finishedAt || runData.finished_at || '-',
    branch: runData.branch || '-',
    commit: runData.commitSha || runData.commit_sha || '-',
    commitMessage: runData.commitMessage || runData.commit_message || '-',
  }

  const stages: StageRun[] = (runData.stages || []).map((stage) => ({
    key: stage.stageKey || stage.stage_key || String(stage.id),
    name: stage.name,
    status: (stage.status || 'pending') as StageRun['status'],
    duration: calcDuration(stage.startedAt || stage.started_at, stage.finishedAt || stage.finished_at),
    logs: stage.log || '',
  }))

  const currentStage: StageRun = stages[activeStage] ?? stages[0] ?? { key: '', name: '暂无阶段', status: 'pending', duration: '-', logs: '' }
  const runStatus = STATUS_META[run.status] ?? { color: 'default', text: run.status }

  return (
    <AppPage keepHeaderTitle title={'执行 #' + run.id}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Tooltip title="返回执行记录"><Button aria-label="返回执行记录" icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/runs')} /></Tooltip>
            <Tag color={runStatus.color}>{runStatus.text}</Tag>
            <Text type="secondary">耗时 {run.duration}</Text>
          </Space>
          <Space>
            {run.status === 'running' ? (
              <Tooltip title="取消执行"><Button danger aria-label="取消执行" icon={<StopOutlined />} onClick={handleCancel} /></Tooltip>
            ) : (
              <Tooltip title="重新执行"><Button aria-label="重新执行" icon={<ReloadOutlined />} onClick={handleRerun} /></Tooltip>
            )}
          </Space>
        </div>

        {/* 执行概览 */}
        <Card title="执行概览">
          <Descriptions column={{ xs: 1, sm: 2, lg: 3 }}>
            <Descriptions.Item label="流水线">
              <a onClick={() => history.push('/cicd/pipelines/' + (run.pipelineId ?? ''))}>{run.pipeline}</a>
            </Descriptions.Item>
            <Descriptions.Item label="触发者">{run.trigger}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={runStatus.color}>{runStatus.text}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="开始时间">{run.startedAt}</Descriptions.Item>
            <Descriptions.Item label="结束时间">{run.finishedAt}</Descriptions.Item>
            <Descriptions.Item label="耗时">{run.duration}</Descriptions.Item>
            <Descriptions.Item label="代码分支">
              <Tag color="blue">{run.branch}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Commit">
              <Text code>{run.commit}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="提交信息">{run.commitMessage}</Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 阶段执行 + 日志 */}
        <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start' }}>
          {/* 左侧阶段时间线 */}
          <Card title="执行阶段" style={{ width: 280, flexShrink: 0 }}>
            <Steps
              direction="vertical"
              size="small"
              current={activeStage}
              items={stages.map((stage, idx) => ({
                title: stage.name,
                description: (
                  <span style={{ cursor: 'pointer', color: idx === activeStage ? DESIGN_COLORS.primary : DESIGN_COLORS.textSecondary }}>
                    {stage.duration}
                  </span>
                ),
                icon: STAGE_STATUS_ICON[stage.status],
                status: stage.status === 'success' ? 'finish' : stage.status === 'failed' ? 'error' : stage.status === 'running' ? 'process' : 'wait',
              }))}
              onChange={(current) => setActiveStage(current)}
            />
          </Card>

          {/* 右侧日志 */}
          <Card
            title={currentStage.name + ' - 日志'}
            extra={<Text type="secondary" style={{ fontSize: 13 }}>{currentStage.duration}</Text>}
            style={{ flex: 1, minWidth: 0 }}
          >
            <TerminalCodeBlock
              title={run.pipeline + '/' + currentStage.key + '.log'}
              content={currentStage.logs}
            />
          </Card>
        </div>
      </div>
    </AppPage>
  )
}

export default RunDetailPage
