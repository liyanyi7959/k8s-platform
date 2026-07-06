import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Button, Input, Select, Typography } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined, PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listServiceAccounts, deleteServiceAccount } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText, ManifestApplyDrawer } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ServiceAccount } from '@/types'

const { Text } = Typography

const ServiceAccountsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailSA, setDetailSA] = useState<ServiceAccount | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-serviceaccounts', clusterId, namespace],
    queryFn: ({ signal }) => listServiceAccounts(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailSA || createOpen ? false : 30_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: ServiceAccount) =>
      deleteServiceAccount(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('ServiceAccount 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-serviceaccounts', clusterId] })
    },
  })

  const filteredData = (data?.items || []).filter((item) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
  })

  const columns: ProColumns<ServiceAccount>[] = [
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
      copyable: true,
      render: (_, r) => <Text strong>{r.name}</Text>,
    },
    { title: 'Secrets', dataIndex: 'secrets', width: 80, align: 'center' as const, search: false },
    {
      title: 'Age',
      dataIndex: 'createdAt',
      width: 110,
      align: 'center' as const,
      search: false,
      ellipsis: true,
      sorter: (a, b) => new Date(a.createdAt || 0).getTime() - new Date(b.createdAt || 0).getTime(),
      render: (_, r) => r.createdAt ? formatDate(r.createdAt) : '-',
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
            <a onClick={() => setDetailSA(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ServiceAccount？"
            onConfirm={() => deleteMutation.mutate(record)}
          >
            <Tooltip title="删除">
              <a style={{ color: '#ff4d4f' }}>
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
      <ProTable<ServiceAccount>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 700 }}
        headerTitle={<Text strong>ServiceAccount 列表</Text>}
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
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建 ServiceAccount
          </Button>,
        ]}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`ServiceAccount 详情 - ${detailSA?.name}`}
        open={!!detailSA}
        onClose={() => setDetailSA(null)}
        width={720}
        destroyOnClose
      >
        {detailSA && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="名称">{detailSA.name}</Descriptions.Item>
              <Descriptions.Item label="Namespace"><Tag>{detailSA.namespace}</Tag></Descriptions.Item>
              <Descriptions.Item label="Secrets" span={2}>{detailSA.secrets ?? 0}</Descriptions.Item>
              <Descriptions.Item label="创建时间" span={2}>{formatDate(detailSA.createdAt)}</Descriptions.Item>
            </Descriptions>
          </Space>
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="serviceaccounts"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      <ManifestApplyDrawer
        clusterId={clusterId}
        open={createOpen}
        onClose={() => { setCreateOpen(false); queryClient.invalidateQueries({ queryKey: ['k8s-serviceaccounts', clusterId] }) }}
        title="创建 ServiceAccount"
        initialYaml={`apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: my-sa\n  namespace: ${namespace || 'default'}\n`}
      />
    </AppPage>
  )
}

export default ServiceAccountsPage
