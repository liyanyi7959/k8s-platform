import React from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listAlertEvents } from '@/services/monitor'
import { formatDate } from '@/utils'
import type { AlertEvent } from '@/types'

/** 告警事件页 */
const EventsPage: React.FC = () => {
  const { data, isLoading } = useQuery({
    queryKey: ['alert-events'],
    queryFn: () => listAlertEvents(),
  })

  const columns: ProColumns<AlertEvent>[] = [
    { title: '事件 ID', dataIndex: 'id', width: 80 },
    { title: '规则名称', dataIndex: 'ruleName' },
    { title: '集群', dataIndex: 'clusterName', width: 150 },
    {
      title: '告警级别',
      dataIndex: 'severity',
      width: 100,
      render: (_, record) => {
        const colorMap: Record<string, string> = {
          info: 'blue',
          warning: 'orange',
          critical: 'red',
        }
        return <Tag color={colorMap[record.severity]}>{record.severity}</Tag>
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (_, record) => {
        const statusMap: Record<string, { label: string; color: string }> = {
          firing: { label: '触发中', color: 'red' },
          resolved: { label: '已恢复', color: 'green' },
        }
        const status = statusMap[record.status] || { label: record.status, color: 'default' }
        return <Tag color={status.color}>{status.label}</Tag>
      },
    },
    { title: '消息', dataIndex: 'message', ellipsis: true },
    { title: '当前值', dataIndex: 'value', width: 100 },
    {
      title: '触发时间',
      dataIndex: 'startedAt',
      width: 180,
      render: (_, record) => formatDate(record.startedAt),
    },
    {
      title: '恢复时间',
      dataIndex: 'resolvedAt',
      width: 180,
      render: (_, record) => (record.resolvedAt ? formatDate(record.resolvedAt) : '-'),
    },
  ]

  return (
    <AppPage>
      <ProTable<AlertEvent>
        headerTitle="告警事件"
        columns={columns}
        dataSource={data || []}
        loading={isLoading}
        rowKey="id"
        search={false}
        pagination={{ pageSize: 10, showTotal: (total) => `共 ${total} 条` }}
      />
    </AppPage>
  )
}

export default EventsPage
