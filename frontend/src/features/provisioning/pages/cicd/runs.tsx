/**
 * 执行记录 - CI/CD Pipeline 运行历史与监控
 */
import { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Card, Input, Select, Space, Table, Tag, Tooltip } from 'antd'
import {
  EyeOutlined,
  ReloadOutlined,
  StopOutlined,
} from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import type { ProColumns } from '@ant-design/pro-components'
import { getRuns, type Run } from '@/features/provisioning/api/cicd'

interface RunRecord {
  id: string
  pipeline: string
  trigger: string
  status: 'running' | 'success' | 'failed' | 'canceled'
  duration: string
  startedAt: string
}

const STATUS_MAP: Record<string, { color: string; text: string }> = {
  running: { color: 'processing', text: '执行中' },
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  canceled: { color: 'default', text: '已取消' },
}

const columns: ProColumns<RunRecord>[] = [
  {
    title: '流水线',
    dataIndex: 'pipeline',
    key: 'pipeline',
    render: (_, record) => <strong>{record.pipeline}</strong>,
  },
  {
    title: '触发者',
    dataIndex: 'trigger',
    key: 'trigger',
    width: 120,
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: 100,
    render: (_, record) => {
      const cfg = STATUS_MAP[record.status] ?? STATUS_MAP.failed!
      return <Tag color={cfg.color}>{cfg.text}</Tag>
    },
  },
  {
    title: '耗时',
    dataIndex: 'duration',
    key: 'duration',
    width: 100,
  },
  {
    title: '开始时间',
    dataIndex: 'startedAt',
    key: 'startedAt',
    width: 180,
    ellipsis: true,
  },
  {
    title: '操作',
    key: 'action',
    width: 140,
    render: (_, record) => (
      <Space>
        <Tooltip title="查看详情">
          <a><EyeOutlined /></a>
        </Tooltip>
        {record.status === 'running' ? (
          <Tooltip title="取消执行">
            <a style={{ color: '#dc2626' }}><StopOutlined /></a>
          </Tooltip>
        ) : (
          <Tooltip title="重新执行">
            <a><ReloadOutlined /></a>
          </Tooltip>
        )}
      </Space>
    ),
  },
]

const MOCK_RUNS: RunRecord[] = [
  { id: 'r128', pipeline: 'frontend-ci', trigger: 'admin', status: 'success', duration: '3m 24s', startedAt: '07-26 14:32' },
  { id: 'r127', pipeline: 'backend-deploy', trigger: 'schedule', status: 'failed', duration: '8m 12s', startedAt: '07-26 12:00' },
  { id: 'r126', pipeline: 'api-gateway-build', trigger: 'admin', status: 'success', duration: '5m 43s', startedAt: '07-26 10:15' },
  { id: 'r125', pipeline: 'helm-chart-release', trigger: 'admin', status: 'running', duration: '2m 18s', startedAt: '07-26 09:30' },
  { id: 'r124', pipeline: 'frontend-ci', trigger: 'push', status: 'success', duration: '3m 15s', startedAt: '07-25 18:22' },
  { id: 'r123', pipeline: 'db-migration', trigger: 'schedule', status: 'success', duration: '12m 05s', startedAt: '07-25 18:00' },
  { id: 'r122', pipeline: 'nginx-deploy', trigger: 'push', status: 'failed', duration: '1m 30s', startedAt: '07-25 10:05' },
  { id: 'r121', pipeline: 'frontend-ci', trigger: 'push', status: 'canceled', duration: '0m 45s', startedAt: '07-25 09:20' },
]
void MOCK_RUNS

const RunsPage: React.FC = () => {
  const [runs, setRuns] = useState<RunRecord[]>([])
  const load = () => getRuns({ page: 1, pageSize: 100 }).then((res) => setRuns((res.list || []).map((r: Run) => ({ id: String(r.id), pipeline: r.pipelineName || r.pipeline_name || String(r.pipelineId || r.pipelineID || '-'), trigger: r.triggerType || r.trigger_type || 'manual', status: r.status as RunRecord['status'], duration: '-', startedAt: r.startedAt || r.started_at || '-' }))))
  useEffect(() => { load() }, [])
  return (
    <AppPage>
      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Input.Search placeholder="搜索流水线名称" style={{ width: 240 }} allowClear />
            <Select
              placeholder="执行状态"
              style={{ width: 140 }}
              allowClear
              options={[
                { value: 'running', label: '执行中' },
                { value: 'success', label: '成功' },
                { value: 'failed', label: '失败' },
                { value: 'canceled', label: '已取消' },
              ]}
            />
          </Space>
        </div>
        <Table<RunRecord>
          rowKey="id"
          columns={columns as any}
          dataSource={runs}
          pagination={false}
          onRow={(record) => ({ onClick: () => history.push(`/cicd/runs/${record.id}`), style: { cursor: 'pointer' } })}
          locale={{
            emptyText: <EmptyState description="暂无执行记录" />,
          }}
        />
      </Card>
    </AppPage>
  )
}

export default RunsPage
