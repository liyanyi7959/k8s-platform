/**
 * 流水线详情 - 展示流水线配置、阶段步骤与执行历史
 */
import React from 'react'
import { useParams, history } from '@umijs/max'
import { Button, Card, Col, Descriptions, Row, Space, Table, Tag, Tooltip, Typography } from 'antd'
import {
  ArrowLeftOutlined,
  PlayCircleOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  BuildOutlined,
  RocketOutlined,
} from '@ant-design/icons'
import { AppPage, YamlEditor } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'

const { Text } = Typography

// 静态数据（后端 API 就绪后替换）
const PIPELINE = {
  id: '1',
  name: 'frontend-ci',
  description: '前端项目 CI 流水线：Lint -> Build -> 镜像构建 -> 推送',
  trigger: '代码推送 (main / develop)',
  status: 'idle',
  createdAt: '2026-06-15 10:30',
  updatedAt: '2026-07-20 14:22',
  lastRun: '2026-07-26 14:32',
  totalRuns: 128,
  successRate: 96.1,
}

interface Stage {
  key: string
  name: string
  icon: React.ReactNode
  steps: { name: string; duration: string }[]
}

const STAGES: Stage[] = [
  {
    key: 'lint',
    name: '代码检查',
    icon: <CheckCircleOutlined />,
    steps: [
      { name: 'ESLint', duration: '12s' },
      { name: 'Prettier Check', duration: '8s' },
    ],
  },
  {
    key: 'build',
    name: '构建',
    icon: <BuildOutlined />,
    steps: [
      { name: 'npm install', duration: '45s' },
      { name: 'npm run build', duration: '1m 23s' },
    ],
  },
  {
    key: 'image',
    name: '镜像构建',
    icon: <RocketOutlined />,
    steps: [
      { name: 'docker build', duration: '2m 15s' },
      { name: 'docker push', duration: '48s' },
    ],
  },
]

const RECENT_RUNS = [
  { id: 'r128', status: 'success', duration: '3m 24s', trigger: 'admin', startedAt: '07-26 14:32' },
  { id: 'r127', status: 'success', duration: '3m 18s', trigger: 'push', startedAt: '07-25 18:22' },
  { id: 'r126', status: 'failed', duration: '1m 02s', trigger: 'push', startedAt: '07-25 16:10' },
  { id: 'r125', status: 'success', duration: '3m 30s', trigger: 'push', startedAt: '07-25 10:05' },
  { id: 'r124', status: 'success', duration: '3m 12s', trigger: 'schedule', startedAt: '07-25 09:00' },
]

const STATUS_META: Record<string, { color: string; text: string }> = {
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  running: { color: 'processing', text: '执行中' },
  idle: { color: 'default', text: '未执行' },
}

const PIPELINE_YAML = `# 流水线定义
stages:
  - name: 代码检查
    steps:
      - name: ESLint
        run: npm run lint
      - name: Prettier Check
        run: npm run prettier --check
  - name: 构建
    steps:
      - name: 安装依赖
        run: npm install
      - name: 构建产物
        run: npm run build
  - name: 镜像构建
    steps:
      - name: 构建镜像
        run: docker build -t registry.example.com/frontend:v1.2.3 .
      - name: 推送镜像
        run: docker push registry.example.com/frontend:v1.2.3
`

const PipelineDetailPage: React.FC = () => {
  const params = useParams()
  const pipelineId = params.id || '1'

  return (
    <AppPage keepHeaderTitle title={PIPELINE.name}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/pipelines')}>返回</Button>
            <Tag color={STATUS_META[PIPELINE.status].color}>{STATUS_META[PIPELINE.status].text}</Tag>
          </Space>
          <Space>
            <Button type="primary" icon={<PlayCircleOutlined />}>执行</Button>
            <Button icon={<EditOutlined />}>编辑</Button>
            <Tooltip title="删除">
              <Button danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Space>
        </div>

        {/* 基本信息 */}
        <Card title="基本信息">
          <Descriptions column={{ xs: 1, sm: 2, lg: 3 }}>
            <Descriptions.Item label="流水线名称">{PIPELINE.name}</Descriptions.Item>
            <Descriptions.Item label="触发方式">{PIPELINE.trigger}</Descriptions.Item>
            <Descriptions.Item label="描述">{PIPELINE.description}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{PIPELINE.createdAt}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{PIPELINE.updatedAt}</Descriptions.Item>
            <Descriptions.Item label="最近执行">{PIPELINE.lastRun}</Descriptions.Item>
            <Descriptions.Item label="总执行次数">{PIPELINE.totalRuns}</Descriptions.Item>
            <Descriptions.Item label="成功率">
              <Text strong style={{ color: DESIGN_COLORS.success }}>{PIPELINE.successRate}%</Text>
            </Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 流水线阶段 */}
        <Card title="流水线阶段">
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 0, overflowX: 'auto', paddingBottom: 8 }}>
            {STAGES.map((stage, idx) => (
              <React.Fragment key={stage.key}>
                <div style={{ flexShrink: 0, width: 240 }}>
                  <div
                    style={{
                      border: `1px solid ${DESIGN_COLORS.grid}`,
                      borderRadius: 10,
                      padding: 16,
                      background: '#fff',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
                      <div
                        style={{
                          width: 32, height: 32, borderRadius: '50%',
                          background: DESIGN_COLORS.primarySoft, color: DESIGN_COLORS.primary,
                          display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 15,
                        }}
                      >
                        {stage.icon}
                      </div>
                      <div>
                        <Text strong>{stage.name}</Text>
                        <div><Text code style={{ fontSize: 11 }}>{stage.key}</Text></div>
                      </div>
                    </div>
                    {stage.steps.map((step, si) => (
                      <div
                        key={si}
                        style={{
                          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                          padding: '6px 0', borderBottom: si < stage.steps.length - 1 ? `1px solid ${DESIGN_COLORS.grid}` : 'none',
                        }}
                      >
                        <Text style={{ fontSize: 13 }}>{step.name}</Text>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          <ClockCircleOutlined /> {step.duration}
                        </Text>
                      </div>
                    ))}
                  </div>
                </div>
                {idx < STAGES.length - 1 && (
                  <div style={{ display: 'flex', alignItems: 'center', height: 80, padding: '0 4px', color: DESIGN_COLORS.textMuted }}>
                    <ArrowLeftOutlined rotate={180} />
                  </div>
                )}
              </React.Fragment>
            ))}
          </div>
        </Card>

        {/* 流水线配置 */}
        <Card title="流水线配置">
          <YamlEditor readOnly value={PIPELINE_YAML} height={360} />
        </Card>

        {/* 最近执行历史 */}
        <Card title="最近执行" extra={<Button type="link" onClick={() => history.push('/cicd/runs')}>查看全部</Button>}>
          <Table
            rowKey="id"
            dataSource={RECENT_RUNS}
            pagination={false}
            size="small"
            onRow={(record) => ({ onClick: () => history.push(`/cicd/runs/${record.id}`), style: { cursor: 'pointer' } })}
            columns={[
              { title: '执行ID', dataIndex: 'id', key: 'id', width: 80, render: (id: string) => <Text code>#{id}</Text> },
              {
                title: '状态', dataIndex: 'status', key: 'status', width: 100,
                render: (status: string) => <Tag color={STATUS_META[status]?.color}>{STATUS_META[status]?.text}</Tag>,
              },
              { title: '触发者', dataIndex: 'trigger', key: 'trigger', width: 100 },
              { title: '耗时', dataIndex: 'duration', key: 'duration', width: 100 },
              { title: '开始时间', dataIndex: 'startedAt', key: 'startedAt' },
            ]}
          />
        </Card>
      </div>
    </AppPage>
  )
}

export default PipelineDetailPage
