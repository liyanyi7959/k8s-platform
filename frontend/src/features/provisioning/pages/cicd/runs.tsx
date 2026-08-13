/**
 * 执行记录 - CI/CD Pipeline 运行历史与监控
 */
import { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Card, Input, message, Select, Space, Table, Tag, Tooltip } from 'antd'
import {
  EyeOutlined,
  ReloadOutlined,
  StopOutlined,
} from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import type { ProColumns } from '@ant-design/pro-components'
import { cancelRun, getRuns, triggerPipeline, type Run } from '@/features/provisioning/api/cicd'

interface RunRecord {
  id: string
  pipelineId: string
  pipeline: string
  trigger: string
  status: 'running' | 'success' | 'failed' | 'canceled'
  duration: string
  startedAt: string
}

const STATUS_META: Record<string, { color: string; text: string }> = {
  running: { color: 'processing', text: '执行中' },
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  canceled: { color: 'default', text: '已取消' },
}

/** 计算耗时：< 60s 显示 "x.xs"，否则 "x.xm" */
const formatDuration = (startedAt?: string, finishedAt?: string): string => {
  if (!startedAt || !finishedAt) return '-'
  const start = new Date(startedAt).getTime()
  const end = new Date(finishedAt).getTime()
  const diff = (end - start) / 1000
  if (diff < 0) return '-'
  if (diff < 60) return diff.toFixed(1) + 's'
  return (diff / 60).toFixed(1) + 'm'
}

const RunsPage: React.FC = () => {
  const [runs, setRuns] = useState<RunRecord[]>([])
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)

  const load = () => {
    getRuns({ page, pageSize, keyword, status: statusFilter }).then((res) => {
      setRuns((res.list || []).map((r: Run) => ({
        id: String(r.id),
        pipelineId: String(r.pipelineId || r.pipelineID || ''),
        pipeline: r.pipelineName || r.pipeline_name || String(r.pipelineId || r.pipelineID || '-'),
        trigger: r.triggerType || r.trigger_type || 'manual',
        status: r.status as RunRecord['status'],
        duration: formatDuration(r.startedAt || r.started_at, r.finishedAt || r.finished_at),
        startedAt: r.startedAt || r.started_at || '-',
      })))
      setTotal(res.total || 0)
    })
  }

  useEffect(() => { load() }, [page, pageSize, keyword, statusFilter])

  const handleCancel = async (id: string) => {
    try {
      await cancelRun(id)
      message.success('已取消执行')
      load()
    } catch {
      message.error('取消执行失败')
    }
  }

  const handleRerun = async (pipelineId: string) => {
    if (!pipelineId) return
    try {
      await triggerPipeline(pipelineId)
      message.success('已触发重新执行')
      load()
    } catch {
      message.error('触发执行失败')
    }
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
        const cfg = STATUS_META[record.status] ?? STATUS_META.failed!
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
            <a onClick={() => history.push('/cicd/runs/' + record.id)}><EyeOutlined /></a>
          </Tooltip>
          {record.status === 'running' ? (
            <Tooltip title="取消执行">
              <a style={{ color: '#dc2626' }} onClick={(e) => { e.stopPropagation(); handleCancel(record.id) }}><StopOutlined /></a>
            </Tooltip>
          ) : (
            <Tooltip title="重新执行">
              <a onClick={(e) => { e.stopPropagation(); handleRerun(record.pipelineId) }}><ReloadOutlined /></a>
            </Tooltip>
          )}
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Input.Search
              placeholder="搜索流水线名称"
              style={{ width: 240 }}
              allowClear
              onSearch={(value) => { setKeyword(value); setPage(1) }}
            />
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
              onChange={(value) => { setStatusFilter(value); setPage(1) }}
            />
          </Space>
        </div>
        <Table<RunRecord>
          rowKey="id"
          columns={columns as any}
          dataSource={runs}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showTotal: (t) => '共 ' + t + ' 条',
            onChange: (p, ps) => { setPage(p); setPageSize(ps) },
          }}
          onRow={(record) => ({ onClick: () => history.push('/cicd/runs/' + record.id), style: { cursor: 'pointer' } })}
          locale={{
            emptyText: <EmptyState description="暂无执行记录" />,
          }}
        />
      </Card>
    </AppPage>
  )
}

export default RunsPage
