import React, { useMemo, useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Alert, Button, Space, Tag, Typography } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { listEvents } from '@/services/k8s'
import { formatDate } from '@/utils'
import type { K8sEvent } from '@/types'

const { Text } = Typography

const EventsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [eventType, setEventType] = useState<string>('')

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-events', clusterId, namespace],
    queryFn: () => listEvents(clusterId, namespace || undefined),
    enabled: !!clusterId,
    refetchInterval: 30_000,
  })

  const filteredData = useMemo(() => {
    const items = data || []
    if (!eventType) return items
    return items.filter((item) => item.type === eventType)
  }, [data, eventType])

  const columns: ProColumns<K8sEvent>[] = [
    {
      title: '级别',
      dataIndex: 'type',
      width: 100,
      render: (_, record) => (
        <Tag color={record.type === 'Warning' ? 'orange' : 'blue'}>{record.type || 'Normal'}</Tag>
      ),
    },
    {
      title: '原因',
      dataIndex: 'reason',
      width: 180,
      ellipsis: true,
      render: (_, record) => <span style={{ fontWeight: 600 }}>{record.reason || '-'}</span>,
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, record) => <EllipsisText text={record.namespace} tag />,
    },
    {
      title: '关联对象',
      dataIndex: 'involvedObject',
      width: 220,
      ellipsis: true,
      render: (_, record) => record.involvedObject || '-',
    },
    {
      title: '消息',
      dataIndex: 'message',
      ellipsis: true,
      render: (_, record) => record.message || '-',
    },
    {
      title: '次数',
      dataIndex: 'count',
      width: 90,
      render: (_, record) => record.count || 1,
    },
    {
      title: '最近时间',
      dataIndex: 'lastTimestamp',
      width: 180,
      render: (_, record) => formatDate(record.lastTimestamp || record.firstTimestamp),
    },
  ]

  return (
    <AppPage>
      <ProTable<K8sEvent>
        headerTitle="集群事件"
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(record, index) => `${record.namespace}/${record.involvedObject}/${record.reason}/${index}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        toolBarRender={() => [
          <NamespaceSelector
            key="namespace"
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 200 }}
          />,
          <Button
            key="all"
            type={eventType === '' ? 'primary' : 'default'}
            onClick={() => setEventType('')}
          >
            全部
          </Button>,
          <Button
            key="warning"
            type={eventType === 'Warning' ? 'primary' : 'default'}
            danger={eventType === 'Warning'}
            onClick={() => setEventType('Warning')}
          >
            Warning
          </Button>,
          <Button
            key="normal"
            type={eventType === 'Normal' ? 'primary' : 'default'}
            onClick={() => setEventType('Normal')}
          >
            Normal
          </Button>,
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => queryClient.invalidateQueries({ queryKey: ['k8s-events', clusterId] })}
          >
            刷新
          </Button>,
        ]}
      />

      <Alert
        style={{ marginTop: 16 }}
        type="info"
        showIcon
        message="事件用于定位调度失败、镜像拉取失败、容器重启等问题"
        description={
          <Space size={16} wrap>
            <Text type="secondary">默认每 30 秒自动刷新一次。</Text>
            <Text type="secondary">建议优先关注 Warning 事件。</Text>
          </Space>
        }
      />
    </AppPage>
  )
}

export default EventsPage