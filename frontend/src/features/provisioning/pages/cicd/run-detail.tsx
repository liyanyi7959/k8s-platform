/**
 * 执行详情 - 展示执行概览、阶段时间线与实时日志
 */
import React, { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Descriptions, Space, Steps, Tag, Typography } from 'antd'
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
import { getRun, type Run } from '@/features/provisioning/api/cicd'

const { Text } = Typography

const RUN = {
  id: 'r128',
  pipeline: 'frontend-ci',
  status: 'success',
  duration: '3m 24s',
  trigger: 'admin',
  startedAt: '2026-07-26 14:32:00',
  finishedAt: '2026-07-26 14:35:24',
  branch: 'main',
  commit: 'a1b2c3d',
  commitMessage: 'feat: add CI/CD dashboard page',
}

interface StageRun {
  key: string
  name: string
  status: 'success' | 'failed' | 'running' | 'pending'
  duration: string
  logs: string
}

const STAGES: StageRun[] = [
  {
    key: 'lint',
    name: '代码检查',
    status: 'success',
    duration: '20s',
    logs: `$ npm run lint\n\n> frontend@1.0.0 lint\n> eslint src --ext .ts,.tsx\n\n✓ 0 errors\n✓ 2 warnings\n\nDone in 20.1s`,
  },
  {
    key: 'build',
    name: '构建',
    status: 'success',
    duration: '2m 08s',
    logs: `$ npm run build\n\n> frontend@1.0.0 build\n> vite build\n\n  ✓ 1023 modules transformed\n  ✓ dist/index.html         0.46 kB\n  ✓ dist/assets/index.js   312.4 kB\n  ✓ dist/assets/index.css   45.2 kB\n\n✓ built in 128.3s`,
  },
  {
    key: 'image',
    name: '镜像构建',
    status: 'success',
    duration: '56s',
    logs: `$ docker build -t registry.example.com/frontend:v1.2.3 .\n\nStep 1/8 : FROM node:18-alpine AS builder\n ---> 0a3c5e2b...\nStep 2/8 : WORKDIR /app\n ---> Using cache\nStep 8/8 : EXPOSE 80\n ---> Running in 1a2b3c4d\n --->Successfully tagged registry.example.com/frontend:v1.2.3\n\n$ docker push registry.example.com/frontend:v1.2.3\nThe push refers to repository [registry.example.com/frontend]\nv1.2.3: digest: sha256:abc123... size: 5284\n✓ Image pushed successfully`,
  },
]

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
}

const RunDetailPage: React.FC = () => {
  const [activeStage, setActiveStage] = useState(0)
  const [runData, setRunData] = useState<Run | null>(null)
  const runID = history.location.pathname.split('/').pop() || ''
  useEffect(() => { if (runID) getRun(runID).then(setRunData) }, [runID])
  const run = runData ? { id: runData.id, pipeline: runData.pipelineName || runData.pipeline_name || '-', status: runData.status, duration: '-', trigger: runData.triggerType || runData.trigger_type || 'manual', startedAt: runData.startedAt || runData.started_at || '-', finishedAt: runData.finishedAt || runData.finished_at || '-', branch: runData.branch || '-', commit: runData.commitSha || runData.commit_sha || '-', commitMessage: runData.commitMessage || runData.commit_message || '-' } : RUN
  const stages = runData?.stages?.length ? runData.stages.map((stage) => ({ key: stage.stageKey || stage.stage_key || String(stage.id), name: stage.name, status: stage.status as StageRun['status'], duration: '-', logs: stage.log || '' })) : STAGES
  const currentStage = stages[activeStage] ?? stages[0]!
  const runStatus = STATUS_META[run.status] ?? STATUS_META.failed!

  return (
    <AppPage keepHeaderTitle title={`执行 #${run.id}`}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/runs')}>返回</Button>
            <Tag color={runStatus.color}>{runStatus.text}</Tag>
            <Text type="secondary">耗时 {run.duration}</Text>
          </Space>
          <Space>
            {run.status === 'running' ? (
              <Button danger icon={<StopOutlined />}>取消执行</Button>
            ) : (
              <Button icon={<ReloadOutlined />}>重新执行</Button>
            )}
          </Space>
        </div>

        {/* 执行概览 */}
        <Card title="执行概览">
          <Descriptions column={{ xs: 1, sm: 2, lg: 3 }}>
            <Descriptions.Item label="流水线">
              <a onClick={() => history.push('/cicd/pipelines/1')}>{run.pipeline}</a>
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
            title={`${currentStage.name} - 日志`}
            extra={<Text type="secondary" style={{ fontSize: 13 }}>{currentStage.duration}</Text>}
            style={{ flex: 1, minWidth: 0 }}
          >
            <TerminalCodeBlock
              title={`${run.pipeline}/${currentStage.key}.log`}
              content={currentStage.logs}
            />
          </Card>
        </div>
      </div>
    </AppPage>
  )
}

export default RunDetailPage
