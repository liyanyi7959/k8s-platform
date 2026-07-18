import React, { useMemo, useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Button, DatePicker, Descriptions, Drawer, Input, Select, Space, Tag, Tooltip } from 'antd'
import { DownloadOutlined, EyeOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { AppPage } from '@/components'
import { listAuditLogs } from '@/services/system'
import { formatDate } from '@/utils'
import type { AuditLog, AuditLogListParams } from '@/types'

const { RangePicker } = DatePicker

type StatusFilter = 'success' | 'failure' | undefined

const ACTION_OPTIONS = [
  { label: '创建', value: 'create' },
  { label: '更新', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '执行', value: 'exec' },
  { label: '停机', value: 'drain' },
  { label: '调度', value: 'cordon' },
  { label: '扩缩容', value: 'scale' },
  { label: '重启', value: 'restart' },
  { label: '滚动更新', value: 'rollout' },
  { label: '认证', value: 'auth' },
]

const RESOURCE_OPTIONS = [
  { label: '集群', value: 'cluster' },
  { label: '命名空间', value: 'namespace' },
  { label: '节点', value: 'node' },
  { label: 'Pod', value: 'pod' },
  { label: 'Deployment', value: 'deployment' },
  { label: 'StatefulSet', value: 'statefulset' },
  { label: 'DaemonSet', value: 'daemonset' },
  { label: 'Job', value: 'job' },
  { label: 'CronJob', value: 'cronjob' },
  { label: 'Service', value: 'service' },
  { label: 'Ingress', value: 'ingress' },
  { label: 'ConfigMap', value: 'configmap' },
  { label: 'Secret', value: 'secret' },
  { label: 'PVC', value: 'pvc' },
  { label: 'PV', value: 'pv' },
  { label: '用户', value: 'user' },
  { label: '角色', value: 'role' },
]

const getActionMeta = (action: string) => {
  const map: Record<string, { label: string; color: string }> = {
    create: { label: '创建', color: 'green' },
    update: { label: '更新', color: 'blue' },
    delete: { label: '删除', color: 'red' },
    exec: { label: '执行', color: 'purple' },
    drain: { label: '停机', color: 'orange' },
    cordon: { label: '调度', color: 'orange' },
    scale: { label: '扩缩容', color: 'cyan' },
    restart: { label: '重启', color: 'volcano' },
    rollout: { label: '滚动更新', color: 'geekblue' },
    auth: { label: '认证', color: 'default' },
    login: { label: '登录', color: 'default' },
    logout: { label: '登出', color: 'default' },
  }
  return map[action] || { label: action || '未知', color: 'default' }
}

const getStatusMeta = (statusCode?: number) => {
  if (!statusCode) return { label: '未知', color: 'default' }
  if (statusCode >= 200 && statusCode < 300) return { label: '成功', color: 'success' }
  if (statusCode >= 400) return { label: '失败', color: 'error' }
  return { label: `${statusCode}`, color: 'default' }
}

const exportCsv = (items: AuditLog[]) => {
  const headers = ['时间', '操作者', '动作', '资源', '资源名', '命名空间', '集群ID', '状态码', '结果', 'IP', '请求ID', '详情']
  const rows = items.map((item) => [
    item.timestamp,
    item.username,
    getActionMeta(item.action).label,
    item.resource,
    item.resourceName || '',
    item.namespace || '',
    item.clusterId ? String(item.clusterId) : '',
    item.statusCode ? String(item.statusCode) : '',
    getStatusMeta(item.statusCode).label,
    item.ip,
    item.requestId || '',
    item.detail,
  ])
  const csv = [headers, ...rows]
    .map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(','))
    .join('\n')
  const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `audit-logs-${dayjs().format('YYYY-MM-DD_HH-mm-ss')}.csv`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

/** 审计日志页 */
const AuditLogsPage: React.FC = () => {
  const [searchText, setSearchText] = useState('')
  const [debouncedKeyword, setDebouncedKeyword] = useState('')
  const [usernameFilter, setUsernameFilter] = useState('')
  const [actionFilter, setActionFilter] = useState<string | undefined>(undefined)
  const [resourceFilter, setResourceFilter] = useState<string | undefined>(undefined)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(undefined)
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null)
  const [pagination, setPagination] = useState({ page: 1, pageSize: 20 })
  const [detailLog, setDetailLog] = useState<AuditLog | null>(null)

  React.useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(searchText.trim()), 300)
    return () => clearTimeout(timer)
  }, [searchText])

  const params = useMemo<AuditLogListParams>(() => {
    const result: AuditLogListParams = {
      page: pagination.page,
      pageSize: pagination.pageSize,
    }
    if (debouncedKeyword) {
      result.keyword = debouncedKeyword
    }
    if (usernameFilter.trim()) {
      result.username = usernameFilter.trim()
    }
    if (actionFilter) {
      result.action = actionFilter
    }
    if (resourceFilter) {
      result.resource = resourceFilter
    }
    if (statusFilter) {
      result.status = statusFilter
    }
    if (dateRange?.[0]) {
      result.startTime = dateRange[0].startOf('day').toISOString()
    }
    if (dateRange?.[1]) {
      result.endTime = dateRange[1].endOf('day').toISOString()
    }
    return result
  }, [
    actionFilter,
    dateRange,
    debouncedKeyword,
    pagination.page,
    pagination.pageSize,
    resourceFilter,
    statusFilter,
    usernameFilter,
  ])

  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['audit-logs', params],
    queryFn: () => listAuditLogs(params),
  })

  const columns: ProColumns<AuditLog>[] = [
    {
      title: '操作者',
      dataIndex: 'username',
      width: 180,
      render: (_, record) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4, minWidth: 0 }}>
          <span style={{ fontWeight: 500 }}>{record.username || '未知用户'}</span>
          <span style={{ color: 'rgba(0, 0, 0, 0.45)', fontSize: 12 }}>{record.ip || '—'}</span>
        </div>
      ),
    },
    {
      title: '动作',
      dataIndex: 'action',
      width: 100,
      align: 'center',
      render: (_, record) => {
        const action = getActionMeta(record.action)
        return <Tag color={action.color}>{action.label}</Tag>
      },
    },
    {
      title: '资源',
      dataIndex: 'resource',
      width: 120,
      ellipsis: true,
    },
    {
      title: '资源名称 / 详情',
      dataIndex: 'detail',
      ellipsis: true,
      render: (_, record) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4, minWidth: 0 }}>
          <span style={{ fontWeight: 500 }}>{record.resourceName || record.resource || '—'}</span>
          <span style={{ color: 'rgba(0, 0, 0, 0.45)', fontSize: 12 }}>{record.detail || '—'}</span>
        </div>
      ),
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 130,
      ellipsis: true,
      render: (_, record) => record.namespace || '—',
    },
    {
      title: '状态',
      dataIndex: 'statusCode',
      width: 90,
      align: 'center',
      render: (_, record) => {
        const status = getStatusMeta(record.statusCode)
        return <Tag color={status.color}>{status.label}</Tag>
      },
    },
    {
      title: '时间',
      dataIndex: 'timestamp',
      width: 180,
      align: 'center',
      render: (_, record) => formatDate(record.timestamp),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      align: 'center',
      render: (_, record) => (
        <Tooltip title="查看详情">
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => setDetailLog(record)}
          />
        </Tooltip>
      ),
    },
  ]

  const hasFilters = Boolean(
    searchText || usernameFilter || actionFilter || resourceFilter || statusFilter || dateRange?.[0] || dateRange?.[1]
  )

  const resetFilters = () => {
    setSearchText('')
    setUsernameFilter('')
    setActionFilter(undefined)
    setResourceFilter(undefined)
    setStatusFilter(undefined)
    setDateRange(null)
    setPagination({ page: 1, pageSize: 20 })
  }

  return (
    <AppPage>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <section
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 12,
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Space size={12} wrap>
            <Input
              placeholder="搜索用户、资源、IP 或详情"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(event) => {
                setSearchText(event.target.value)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              style={{ width: 260 }}
              allowClear
            />
            <Input
              placeholder="操作者"
              value={usernameFilter}
              onChange={(event) => {
                setUsernameFilter(event.target.value)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              style={{ width: 140 }}
              allowClear
            />
            <Select
              allowClear
              placeholder="操作类型"
              style={{ width: 140 }}
              value={actionFilter}
              onChange={(value) => {
                setActionFilter(value)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              options={ACTION_OPTIONS}
            />
            <Select
              allowClear
              placeholder="资源类型"
              style={{ width: 140 }}
              value={resourceFilter}
              onChange={(value) => {
                setResourceFilter(value)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              options={RESOURCE_OPTIONS}
            />
            <Select
              allowClear
              placeholder="执行结果"
              style={{ width: 120 }}
              value={statusFilter}
              onChange={(value) => {
                setStatusFilter(value)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              options={[
                { label: '成功', value: 'success' },
                { label: '失败', value: 'failure' },
              ]}
            />
            <RangePicker
              value={dateRange}
              onChange={(values) => {
                setDateRange(values as [dayjs.Dayjs | null, dayjs.Dayjs | null] | null)
                setPagination((prev) => ({ ...prev, page: 1 }))
              }}
              placeholder={['开始日期', '结束日期']}
            />
            <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
              刷新
            </Button>
            {hasFilters ? <Button onClick={resetFilters}>重置</Button> : null}
          </Space>
          <Button icon={<DownloadOutlined />} onClick={() => exportCsv(data?.items || [])}>
            导出
          </Button>
        </section>

        <ProTable<AuditLog>
          columns={columns}
          dataSource={data?.items || []}
          loading={isLoading}
          rowKey="id"
          search={false}
          options={false}
          cardBordered
          tableAlertRender={false}
          pagination={{
            current: pagination.page,
            pageSize: pagination.pageSize,
            total: data?.total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => {
              setPagination({ page, pageSize: pageSize || 20 })
            },
          }}
          toolBarRender={false}
          scroll={{ x: 900 }}
        />
      </div>

      <Drawer title="审计日志详情" width={560} open={!!detailLog} onClose={() => setDetailLog(null)}>
        {detailLog && (
          <Descriptions column={1} bordered>
            <Descriptions.Item label="请求 ID">{detailLog.requestId || '—'}</Descriptions.Item>
            <Descriptions.Item label="操作者">{detailLog.username}</Descriptions.Item>
            <Descriptions.Item label="IP 地址">{detailLog.ip}</Descriptions.Item>
            <Descriptions.Item label="动作">
              <Tag color={getActionMeta(detailLog.action).color}>{getActionMeta(detailLog.action).label}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="资源类型">{detailLog.resource}</Descriptions.Item>
            <Descriptions.Item label="资源名称">{detailLog.resourceName || '—'}</Descriptions.Item>
            <Descriptions.Item label="命名空间">{detailLog.namespace || '—'}</Descriptions.Item>
            <Descriptions.Item label="集群 ID">{detailLog.clusterId || '—'}</Descriptions.Item>
            <Descriptions.Item label="请求路径">{detailLog.path || '—'}</Descriptions.Item>
            <Descriptions.Item label="状态码">{detailLog.statusCode ?? '—'}</Descriptions.Item>
            <Descriptions.Item label="执行结果">
              <Tag color={getStatusMeta(detailLog.statusCode).color}>{getStatusMeta(detailLog.statusCode).label}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="详情">{detailLog.detail}</Descriptions.Item>
            <Descriptions.Item label="发生时间">{formatDate(detailLog.timestamp)}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </AppPage>
  )
}

export default AuditLogsPage
