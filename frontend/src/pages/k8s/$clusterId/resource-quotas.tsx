import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Drawer, Descriptions, Space, Tooltip, Button, Input, Select, Typography, Tabs, Table } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listResourceQuotas, deleteResourceQuota } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ResourceQuota } from '@/types'

const { Text } = Typography

const ResourceQuotasPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailQuota, setDetailQuota] = useState<ResourceQuota | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-resourcequotas', clusterId, namespace],
    queryFn: ({ signal }) => listResourceQuotas(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailQuota ? false : 60_000,
    staleTime: 60_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: ResourceQuota) =>
      deleteResourceQuota(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('ResourceQuota 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-resourcequotas', clusterId] })
    },
  })

  const filteredData = (data?.items || []).filter((item) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
  })

  const columns: ProColumns<ResourceQuota>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
      copyable: true,
      render: (_, record) => <Text strong>{record.name}</Text>,
    },
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: '硬限制',
      dataIndex: 'hard',
      width: 250,
      search: false,
      ellipsis: true,
      render: (_, record) => {
        const text =
          Object.entries(record.hard || {})
            .map(([k, v]) => `${k}: ${v}`)
            .join(', ') || '-'
        return <Tooltip title={text}>{text}</Tooltip>
      },
    },
    {
      title: '已使用',
      dataIndex: 'used',
      width: 250,
      search: false,
      ellipsis: true,
      render: (_, record) => {
        const text =
          Object.entries(record.used || {})
            .map(([k, v]) => `${k}: ${v}`)
            .join(', ') || '-'
        return <Tooltip title={text}>{text}</Tooltip>
      },
    },
    {
      title: 'Age',
      dataIndex: 'createdAt',
      width: 110,
      align: 'center' as const,
      ellipsis: true,
      search: false,
      render: (_, record) => formatDate(record.createdAt),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 140,
      align: 'center' as const,
      fixed: 'right',
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailQuota(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ResourceQuota？"
            onConfirm={() => deleteMutation.mutate(record)}
          >
            <Tooltip title="删除">
              <a style={{ color: '#dc2626' }}>
                <DeleteOutlined />
              </a>
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <ProTable<ResourceQuota>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 900 }}
        headerTitle={<Text strong>ResourceQuota 列表</Text>}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : '按标签搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'label', label: '标签' }]} />}
            prefix={<SearchOutlined />}
          />,
          <NamespaceSelector
            key="ns"
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 180 }}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>,
        ]}
      />
      <YamlDrawer
        clusterId={clusterId}
        resourceType="resourcequotas"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
      <Drawer
        title="资源配额详情"
        open={!!detailQuota}
        onClose={() => setDetailQuota(null)}
        width={700}
        destroyOnClose
      >
        {detailQuota && (
          <Tabs
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <Descriptions column={2} bordered size="small">
                    <Descriptions.Item label="名称">{detailQuota.name}</Descriptions.Item>
                    <Descriptions.Item label="Namespace">
                      {detailQuota.namespace || namespace}
                    </Descriptions.Item>
                    <Descriptions.Item label="硬限制">
                      {Object.entries(detailQuota.hard || {}).map(([k, v]) => (
                        <Tag key={k} color="blue" style={{ marginBottom: 2 }}>
                          {k}: {v}
                        </Tag>
                      ))}
                    </Descriptions.Item>
                    <Descriptions.Item label="已使用">
                      {Object.entries(detailQuota.used || {}).map(([k, v]) => (
                        <Tag key={k} color="orange" style={{ marginBottom: 2 }}>
                          {k}: {v}
                        </Tag>
                      ))}
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间" span={2}>
                      {formatDate(detailQuota.createdAt)}
                    </Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'usage',
                label: '配额使用',
                children: (
                  <Table
                    size="small"
                    rowKey="resource"
                    pagination={false}
                    dataSource={Object.entries(detailQuota.hard || {}).map(([key, hardLimit]) => ({
                      key,
                      resource: key,
                      hard: hardLimit,
                      used: (detailQuota.used || {})[key] || '-',
                      percentage:
                        hardLimit && detailQuota.used?.[key]
                          ? Math.round(
                              (parseInt(String(detailQuota.used[key])) / parseInt(String(hardLimit))) * 100,
                            )
                          : 0,
                    }))}
                    columns={[
                      { title: '资源', dataIndex: 'resource' },
                      { title: '硬限制', dataIndex: 'hard' },
                      { title: '已使用', dataIndex: 'used' },
                      {
                        title: '使用率',
                        dataIndex: 'percentage',
                        align: 'center' as const,
                        render: (val: number) => {
                          const color = val > 80 ? 'red' : val > 50 ? 'orange' : 'green'
                          return <Tag color={color}>{val}%</Tag>
                        },
                      },
                    ]}
                  />
                ),
              },
            ]}
          />
        )}
      </Drawer>
    </AppPage>
  )
}

export default ResourceQuotasPage
