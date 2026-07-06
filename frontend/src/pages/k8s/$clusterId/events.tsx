import React, { useMemo, useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Alert, Button, Space, Tag, Typography, Input } from 'antd'
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { listEvents } from '@/services/k8s'
import { formatDate } from '@/utils'
import type { K8sEvent } from '@/types'

const { Text } = Typography

const EventsPage: React.FC = () => {
  const clusterId = useClusterId()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'type' | 'reason'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [eventType, setEventType] = useState<string>('')

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-events', clusterId, namespace],
    queryFn: ({ signal }) => listEvents(clusterId, namespace || undefined, signal),
    enabled: !!clusterId,
    refetchInterval: 30_000,
  })

  const filteredData = useMemo(() => {
    const items = data || []
    return items.filter((item) => {
      const matchType = !eventType || item.type === eventType
      let matchSearch = true
      if (searchValue) {
        const v = searchValue.toLowerCase()
        if (searchType === 'name') matchSearch = (item.involvedObject || '').toLowerCase().includes(v)
        else if (searchType === 'type') matchSearch = (item.type || '').toLowerCase().includes(v)
        else matchSearch = (item.reason || '').toLowerCase().includes(v)
      }
      return matchType && matchSearch
    })
  }, [data, eventType, searchType, searchValue])

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
      title: 'Namespace',
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
      align: 'center' as const,
      render: (_, record) => record.count || 1,
    },
    {
      title: 'Age',
      dataIndex: 'lastTimestamp',
      width: 110,
      align: 'center' as const,
      ellipsis: true,
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
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : searchType === 'type' ? '按类型搜索' : '按原因搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'type', label: '类型' }, { value: 'reason', label: '原因' }]} />}
            prefix={<SearchOutlined />}
          />,
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
            onClick={() => refetch()}
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
