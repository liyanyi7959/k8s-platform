import React, { useState } from 'react'
import { history } from '@umijs/max'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Button, Drawer, Space, Steps, Tag, Typography, message } from 'antd'
import { BugOutlined, RobotOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { getIncident, listIncidents, transitionIncident } from '@/services/monitor'
import { formatDate } from '@/utils'
import type { MonitorIncident } from '@/types'

const { Text, Paragraph } = Typography
const statusMeta: Record<string, { label: string; color: string }> = {
  open: { label: '待认领', color: 'red' }, acknowledged: { label: '已认领', color: 'orange' },
  diagnosing: { label: 'AI 诊断中', color: 'blue' }, awaiting_approval: { label: '待人工确认', color: 'purple' },
  executing: { label: '自动化执行中', color: 'cyan' }, verifying: { label: '验证中', color: 'geekblue' }, resolved: { label: '已恢复', color: 'green' },
}

const EventsPage: React.FC = () => {
  const qc = useQueryClient(); const [selected, setSelected] = useState<MonitorIncident | null>(null)
  const { data, isLoading } = useQuery({ queryKey: ['monitor-incidents'], queryFn: () => listIncidents() })
  const detail = useQuery({ queryKey: ['monitor-incident', selected?.id], queryFn: () => getIncident(selected!.id), enabled: !!selected })
  const transition = useMutation({ mutationFn: ({ id, action }: { id: number; action: string }) => transitionIncident(id, action), onSuccess: () => { message.success('事件状态已更新'); qc.invalidateQueries({ queryKey: ['monitor-incidents'] }); qc.invalidateQueries({ queryKey: ['monitor-incident'] }) }, onError: () => message.error('事件更新失败') })
  const openAI = () => { if (!selected) return; transition.mutate({ id: selected.id, action: 'diagnose' }); history.push(`/ai/chat?clusterId=${selected.clusterId}&incidentId=${selected.id}&namespace=${selected.namespace || ''}`) }
  const columns: ProColumns<MonitorIncident>[] = [
    { title: '事件', dataIndex: 'alertName', render: (_, r) => <Space direction="vertical" size={0}><Text strong>{r.alertName}</Text><Text type="secondary">{r.summary}</Text></Space> },
    { title: '对象', width: 180, render: (_, r) => `${r.clusterName || `集群 ${r.clusterId}`}${r.namespace ? ` / ${r.namespace}` : ''}${r.resourceName ? ` / ${r.resourceName}` : ''}` },
    { title: '级别', dataIndex: 'severity', width: 90, render: (_, r) => <Tag color={r.severity === 'critical' ? 'red' : r.severity === 'warning' ? 'orange' : 'blue'}>{r.severity}</Tag> },
    { title: '处置状态', dataIndex: 'status', width: 120, render: (_, r) => <Tag color={statusMeta[r.status]?.color}>{statusMeta[r.status]?.label || r.status}</Tag> },
    { title: '触发时间', dataIndex: 'startedAt', width: 170, render: (_, r) => formatDate(r.startedAt) },
    { title: '操作', valueType: 'option', width: 90, render: (_, r) => <a onClick={() => setSelected(r)}>处置</a> },
  ]
  return <AppPage><ProTable<MonitorIncident> headerTitle="事件中心" columns={columns} dataSource={data?.items || []} loading={isLoading} rowKey="id" search={false} pagination={{ total: data?.total, showTotal: total => `共 ${total} 个事件` }} />
    <Drawer title="事件处置闭环" open={!!selected} width={640} onClose={() => setSelected(null)}>
      {selected && <Space direction="vertical" size={18} style={{ width: '100%' }}>
        <Space><Tag color={statusMeta[selected.status]?.color}>{statusMeta[selected.status]?.label}</Tag><Text strong>{selected.alertName}</Text></Space>
        <Paragraph>{selected.summary}</Paragraph>
        <Steps size="small" current={selected.status === 'resolved' ? 5 : selected.status === 'verifying' ? 4 : selected.status === 'executing' ? 3 : selected.status === 'awaiting_approval' ? 2 : selected.status === 'diagnosing' ? 1 : 0} items={[{ title: '认领' }, { title: 'AI 诊断' }, { title: '人工确认' }, { title: '自动化执行' }, { title: '验证' }, { title: '恢复' }]} />
        <Space wrap>
          <Button onClick={() => transition.mutate({ id: selected.id, action: 'acknowledge' })}>认领事件</Button>
          <Button icon={<RobotOutlined />} type="primary" onClick={openAI}>AI 诊断</Button>
          <Button onClick={() => transition.mutate({ id: selected.id, action: 'await_approval' })}>等待确认</Button>
          <Button onClick={() => transition.mutate({ id: selected.id, action: 'execute' })}>开始执行</Button>
          <Button icon={<SafetyCertificateOutlined />} onClick={() => transition.mutate({ id: selected.id, action: 'verify' })}>验证结果</Button>
          <Button icon={<BugOutlined />} type="dashed" onClick={() => transition.mutate({ id: selected.id, action: 'resolve' })}>确认恢复</Button>
        </Space>
        <Text strong>处置时间线</Text>
        <Steps direction="vertical" size="small" items={(detail.data?.timeline || []).map(item => ({ title: item.title, description: <>{item.detail || '-'}<br /><Text type="secondary">{item.operator || '系统'} · {formatDate(item.createdAt)}</Text></> }))} />
      </Space>}
    </Drawer>
  </AppPage>
}
export default EventsPage
