import React, { useEffect, useMemo, useState } from 'react'
import { history } from '@umijs/max'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Button, DatePicker, Descriptions, Drawer, Empty, Input, message, Select, Space, Steps, Tag, Typography } from 'antd'
import { CheckCircleOutlined, ClockCircleOutlined, FilterOutlined, RobotOutlined, SearchOutlined, UserOutlined, WarningOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'

import { AppPage, MetricGrid, StatusStrip } from '@/components'
import {
  acknowledgeIncident,
  getIncident,
  listIncidents,
  requestIncidentApproval,
  resolveIncident,
  startIncidentDiagnosis,
  startIncidentExecution,
  startIncidentVerification,
  type IncidentAction,
} from '@/features/incident/api'
import { formatDate, formatRelativeTime } from '@/utils'
import type { IncidentStatus, MonitorIncident } from '@/features/incident/types'

const { Text, Paragraph, Title } = Typography
const { RangePicker } = DatePicker

const statusMeta: Record<IncidentStatus, { label: string; color: string; step: number }> = {
  open: { label: '待认领', color: 'red', step: 0 },
  acknowledged: { label: '已认领', color: 'orange', step: 0 },
  diagnosing: { label: 'AI 诊断中', color: 'blue', step: 1 },
  awaiting_approval: { label: '待人工确认', color: 'purple', step: 2 },
  executing: { label: '自动化执行中', color: 'cyan', step: 3 },
  verifying: { label: '验证中', color: 'geekblue', step: 4 },
  resolved: { label: '已恢复', color: 'green', step: 5 },
}

const severityMeta = {
  critical: { label: '严重', color: 'red' },
  warning: { label: '警告', color: 'orange' },
  info: { label: '提示', color: 'blue' },
}

const nextAction: Partial<Record<IncidentStatus, { action: IncidentAction; label: string }>> = {
  open: { action: 'acknowledge', label: '认领事件' },
  diagnosing: { action: 'await_approval', label: '提交人工确认' },
  awaiting_approval: { action: 'execute', label: '批准并开始执行' },
  executing: { action: 'verify', label: '进入恢复验证' },
  verifying: { action: 'resolve', label: '确认恢复' },
}

const incidentCommands: Record<IncidentAction, (id: number, version: number, note?: string) => Promise<MonitorIncident>> = {
  acknowledge: acknowledgeIncident,
  diagnose: startIncidentDiagnosis,
  await_approval: requestIncidentApproval,
  execute: startIncidentExecution,
  verify: startIncidentVerification,
  resolve: resolveIncident,
}

const EventsPage: React.FC = () => {
  const queryClient = useQueryClient()
  const requestedIncidentId = Number(new URLSearchParams(history.location.search).get('incident') || 0)
  const [selected, setSelected] = useState<MonitorIncident | null>(null)
  const [keyword, setKeyword] = useState('')
  const [severity, setSeverity] = useState<string>()
  const [status, setStatus] = useState<IncidentStatus>()
  const [clusterId, setClusterId] = useState<number>()
  const [assignee, setAssignee] = useState<string>()
  const [timeRange, setTimeRange] = useState<any>(null)

  const incidentsQuery = useQuery({
    queryKey: ['monitor-incidents'],
    queryFn: ({ signal }) => listIncidents({ page: 1, pageSize: 100 }, signal),
    refetchInterval: 30_000,
  })
  const detailQuery = useQuery({
    queryKey: ['monitor-incident', selected?.id],
    queryFn: () => getIncident(selected!.id),
    enabled: Boolean(selected),
  })

  useEffect(() => {
    if (!requestedIncidentId || selected || !incidentsQuery.data?.items.length) return
    const target = incidentsQuery.data.items.find((item) => item.id === requestedIncidentId)
    if (target) setSelected(target)
  }, [incidentsQuery.data, requestedIncidentId, selected])

  const transition = useMutation({
    mutationFn: ({ id, action, version, note }: { id: number; action: IncidentAction; version: number; note?: string }) => incidentCommands[action](id, version, note),
    onSuccess: (updated, variables) => {
      message.success('事件状态已更新')
      setSelected((current) => current && current.id === variables.id ? updated : current)
      queryClient.invalidateQueries({ queryKey: ['monitor-incidents'] })
      queryClient.invalidateQueries({ queryKey: ['monitor-incident', variables.id] })
      queryClient.invalidateQueries({ queryKey: ['monitor-nav-incidents'] })
    },
    onError: (error) => {
      if ((error as Error & { code?: number }).code === 409) {
        queryClient.invalidateQueries({ queryKey: ['monitor-incidents'] })
        queryClient.invalidateQueries({ queryKey: ['monitor-incident', selected?.id] })
      }
      message.error(error instanceof Error ? error.message : '事件状态更新失败')
    },
  })

  const allIncidents = incidentsQuery.data?.items || []
  const activeIncidents = allIncidents.filter((item) => item.status !== 'resolved')
  const filteredIncidents = useMemo(() => allIncidents.filter((incident) => {
    const searchable = `${incident.alertName} ${incident.summary} ${incident.clusterName} ${incident.namespace} ${incident.resourceName}`.toLowerCase()
    const matchesTime = !timeRange || (
      dayjs(incident.startedAt).isAfter(timeRange[0].startOf('day')) &&
      dayjs(incident.startedAt).isBefore(timeRange[1].endOf('day'))
    )
    return (!keyword || searchable.includes(keyword.trim().toLowerCase()))
      && (!severity || incident.severity === severity)
      && (!status || incident.status === status)
      && (!clusterId || incident.clusterId === clusterId)
      && (!assignee || (assignee === '__unassigned__' ? !incident.assigneeName : incident.assigneeName === assignee))
      && matchesTime
  }), [allIncidents, assignee, clusterId, keyword, severity, status, timeRange])

  const clusterOptions = Array.from(new Map(allIncidents.map((item) => [item.clusterId, item.clusterName || `集群 ${item.clusterId}`])).entries())
    .map(([value, label]) => ({ value, label }))
  const assigneeOptions = Array.from(new Set(allIncidents.map((item) => item.assigneeName).filter(Boolean)))
    .map((value) => ({ value, label: value }))
  assigneeOptions.unshift({ value: '__unassigned__', label: '未认领' })

  const openAI = () => {
    if (!selected) return
    transition.mutate({ id: selected.id, action: 'diagnose', version: selected.version }, {
      onSuccess: () => history.push(`/ai/chat?clusterId=${selected.clusterId}&incidentId=${selected.id}&namespace=${selected.namespace || ''}`),
    })
  }

  const columns: ProColumns<MonitorIncident>[] = [
    {
      title: '事件', dataIndex: 'alertName',
      render: (_, incident) => <Space direction="vertical" size={2}><Text strong>{incident.alertName}</Text><Text type="secondary" ellipsis style={{ maxWidth: 420 }}>{incident.summary || '无事件摘要'}</Text></Space>,
    },
    {
      title: '影响范围', width: 220,
      render: (_, incident) => <Space direction="vertical" size={2}><Text>{incident.clusterName || `集群 ${incident.clusterId}`}</Text><Text type="secondary">{[incident.namespace, incident.resourceName].filter(Boolean).join(' / ') || '集群级事件'}</Text></Space>,
    },
    { title: '级别', dataIndex: 'severity', width: 90, render: (_, incident) => <Tag color={severityMeta[incident.severity].color}>{severityMeta[incident.severity].label}</Tag> },
    { title: '处置状态', dataIndex: 'status', width: 125, render: (_, incident) => <Tag color={statusMeta[incident.status].color}>{statusMeta[incident.status].label}</Tag> },
    { title: '负责人', dataIndex: 'assigneeName', width: 120, render: (_, incident) => incident.assigneeName || <Text type="danger">未认领</Text> },
    { title: '触发时间', dataIndex: 'startedAt', width: 170, render: (_, incident) => <Space direction="vertical" size={0}><Text>{formatDate(incident.startedAt)}</Text><Text type="secondary">{formatRelativeTime(incident.startedAt)}</Text></Space> },
    { title: '操作', valueType: 'option', width: 90, render: (_, incident) => <Button type="link" size="small" onClick={() => setSelected(incident)}>处置</Button> },
  ]

  const selectedStatus = selected ? statusMeta[selected.status] : null
  const selectedNextAction = selected ? nextAction[selected.status] : null

  return (
    <AppPage keepHeaderTitle title="事件处置中心">
      <div className="app-page-shell app-incident-inbox">
        <StatusStrip tone={activeIncidents.length ? 'warning' : 'success'}>
          <span>数据来源：Alertmanager 事件库</span>
          <span>范围：最近 100 条事件</span>
          <span><ClockCircleOutlined /> {incidentsQuery.dataUpdatedAt ? `更新于 ${dayjs(incidentsQuery.dataUpdatedAt).format('HH:mm:ss')}` : '正在同步'}</span>
        </StatusStrip>

        <MetricGrid
          items={[
            { key: 'critical', label: '严重未恢复', value: activeIncidents.filter((item) => item.severity === 'critical').length, icon: <WarningOutlined />, tone: 'danger' },
            { key: 'active', label: '全部未恢复', value: activeIncidents.length, icon: <AlertIcon />, tone: activeIncidents.length ? 'warning' : 'success' },
            { key: 'unassigned', label: '尚未认领', value: activeIncidents.filter((item) => !item.assigneeName).length, icon: <UserOutlined />, tone: activeIncidents.some((item) => !item.assigneeName) ? 'warning' : 'success' },
            { key: 'resolved', label: '已恢复', value: allIncidents.filter((item) => item.status === 'resolved').length, icon: <CheckCircleOutlined />, tone: 'success' },
          ]}
        />

        <section className="app-incident-filters" aria-label="事件筛选">
          <div className="app-incident-filters__title"><FilterOutlined />筛选事件</div>
          <Input allowClear prefix={<SearchOutlined />} placeholder="搜索事件、集群或资源" value={keyword} onChange={(event) => setKeyword(event.target.value)} />
          <Select allowClear placeholder="严重级别" value={severity} onChange={setSeverity} options={Object.entries(severityMeta).map(([value, meta]) => ({ value, label: meta.label }))} />
          <Select allowClear placeholder="处置状态" value={status} onChange={setStatus} options={Object.entries(statusMeta).map(([value, meta]) => ({ value, label: meta.label }))} />
          <Select allowClear showSearch optionFilterProp="label" placeholder="集群" value={clusterId} onChange={setClusterId} options={clusterOptions} />
          <Select allowClear placeholder="负责人" value={assignee} onChange={setAssignee} options={assigneeOptions} />
          <RangePicker value={timeRange} onChange={setTimeRange} />
          <Button onClick={() => { setKeyword(''); setSeverity(undefined); setStatus(undefined); setClusterId(undefined); setAssignee(undefined); setTimeRange(null) }}>重置</Button>
        </section>

        <ProTable<MonitorIncident>
          className="app-console-table"
          headerTitle={`事件收件箱 · ${filteredIncidents.length}`}
          columns={columns}
          dataSource={filteredIncidents}
          loading={incidentsQuery.isLoading}
          rowKey="id"
          rowClassName={(incident) => incident.status !== 'resolved' && incident.severity === 'critical' ? 'app-table-row--attention' : ''}
          search={false}
          options={false}
          toolBarRender={false}
          pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (total) => `共 ${total} 个事件` }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="没有匹配当前筛选条件的事件" /> }}
        />
      </div>

      <Drawer title="事件处置闭环" open={Boolean(selected)} width={720} onClose={() => { setSelected(null); history.replace('/monitor/events') }}>
        {selected ? (
          <Space direction="vertical" size={20} style={{ width: '100%' }}>
            <div className="app-incident-drawer__head">
              <Space><Tag color={severityMeta[selected.severity].color}>{severityMeta[selected.severity].label}</Tag><Tag color={selectedStatus?.color}>{selectedStatus?.label}</Tag></Space>
              <Title level={4}>{selected.alertName}</Title>
              <Paragraph>{selected.summary || 'Alertmanager 未提供事件摘要。'}</Paragraph>
            </div>
            <Descriptions size="small" column={2} bordered>
              <Descriptions.Item label="集群">{selected.clusterName || `集群 ${selected.clusterId}`}</Descriptions.Item>
              <Descriptions.Item label="负责人">{selected.assigneeName || '未认领'}</Descriptions.Item>
              <Descriptions.Item label="Namespace">{selected.namespace || '集群级'}</Descriptions.Item>
              <Descriptions.Item label="资源">{selected.resourceName ? `${selected.resourceKind} / ${selected.resourceName}` : '未指定'}</Descriptions.Item>
              <Descriptions.Item label="触发时间" span={2}>{formatDate(selected.startedAt)}</Descriptions.Item>
            </Descriptions>
            <Steps size="small" current={selectedStatus?.step || 0} items={[{ title: '认领' }, { title: 'AI 诊断' }, { title: '人工确认' }, { title: '自动化执行' }, { title: '验证' }, { title: '恢复' }]} />
            <div className="app-incident-next-action">
              <div><span>建议下一步</span><strong>{selected.status === 'acknowledged' ? '启动 AI 诊断并收集可验证证据' : selectedNextAction?.label || '事件已恢复，可进入复盘'}</strong></div>
              {selected.status === 'acknowledged' ? (
                <Button icon={<RobotOutlined />} type="primary" onClick={openAI} loading={transition.isPending}>AI 诊断</Button>
              ) : selectedNextAction ? (
                <Button type="primary" danger={selectedNextAction.action === 'resolve'} onClick={() => transition.mutate({ id: selected.id, action: selectedNextAction.action, version: selected.version, note: selectedNextAction.action === 'resolve' ? '人工确认恢复验证通过' : undefined })} loading={transition.isPending}>
                  {selectedNextAction.label}
                </Button>
              ) : null}
            </div>
            <Text strong>处置时间线</Text>
            {detailQuery.data?.timeline?.length ? (
              <Steps direction="vertical" size="small" items={detailQuery.data.timeline.map((item) => ({ title: item.title, description: <>{item.detail || '无补充说明'}<br /><Text type="secondary">{item.operator || '系统'} · {formatDate(item.createdAt)}</Text></> }))} />
            ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无处置记录" />}
          </Space>
        ) : null}
      </Drawer>
    </AppPage>
  )
}

const AlertIcon = WarningOutlined

export default EventsPage
