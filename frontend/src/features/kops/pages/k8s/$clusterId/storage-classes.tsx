import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Typography, Input, Select, Button, Table } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listStorageClasses, deleteStorageClass } from '@/features/kops/api/k8s'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { AppPage } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { StorageClass } from '@/shared/types'

const { Text } = Typography

const StorageClassesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [searchType, setSearchType] = useState<'name' | 'provisioner'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailSC, setDetailSC] = useState<StorageClass | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-storageclasses', clusterId],
    queryFn: ({ signal }) => listStorageClasses(clusterId, signal),
    enabled: !!clusterId,
    refetchInterval: detailSC ? false : 60_000,
    staleTime: 60_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteStorageClass(clusterId, name),
    onSuccess: () => {
      message.success('StorageClass 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-storageclasses', clusterId] })
    },
  })

  const filteredData = (data?.items || []).filter((item) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    return item.provisioner?.toLowerCase().includes(v)
  })

  const columns: ProColumns<StorageClass>[] = [
    { title: '名称', dataIndex: 'name', width: 160, ellipsis: true, copyable: true, render: (_, r) => <Text strong>{r.name}</Text> },
    { title: 'Provisioner', dataIndex: 'provisioner', width: 200, ellipsis: true, search: false },
    { title: '回收策略', dataIndex: 'reclaimPolicy', width: 120, search: false, align: 'center' as const },
    {
      title: '绑定模式',
      dataIndex: 'volumeBindingMode',
      width: 140,
      align: 'center' as const,
      search: false,
      render: (_, record) => <Tag>{record.volumeBindingMode}</Tag>,
    },
    {
      title: '默认',
      width: 80,
      search: false,
      align: 'center' as const,
      render: (_, record) => (record.isDefault ? <Tag color="blue">是</Tag> : '-'),
    },
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
      fixed: 'right',
      align: 'center' as const,
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailSC(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 StorageClass？"
            onConfirm={() => deleteMutation.mutate(record.name)}
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
      <ProTable<StorageClass>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey="name"
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1000 }}
        headerTitle={<Text strong>StorageClass 列表</Text>}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : '按 Provisioner 搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'provisioner', label: 'Provisioner' }]} />}
            prefix={<SearchOutlined />}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>,
        ]}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`StorageClass 详情 - ${detailSC?.name}`}
        open={!!detailSC}
        onClose={() => setDetailSC(null)}
        width={720}
        destroyOnClose
      >
        {detailSC && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="名称">{detailSC.name}</Descriptions.Item>
              <Descriptions.Item label="Provisioner">{detailSC.provisioner}</Descriptions.Item>
              <Descriptions.Item label="回收策略">
                <Tag color="blue">{detailSC.reclaimPolicy}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="绑定模式">
                <Tag>{detailSC.volumeBindingMode}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="允许扩容">
                {detailSC.allowVolumeExpansion ? <Tag color="green">是</Tag> : <Tag>否</Tag>}
              </Descriptions.Item>
              <Descriptions.Item label="默认">
                {detailSC.isDefault ? <Tag color="blue">是</Tag> : '否'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间" span={2}>
                {formatDate(detailSC.createdAt)}
              </Descriptions.Item>
            </Descriptions>

            <Table
              size="small"
              title={() => `参数 (${Object.keys(detailSC.parameters || {}).length})`}
              rowKey={(r) => r.key}
              pagination={false}
              dataSource={Object.entries(detailSC.parameters || {}).map(([k, v]) => ({ key: k, value: v }))}
              columns={[
                { title: '键', dataIndex: 'key', width: 200, ellipsis: true },
                { title: '值', dataIndex: 'value', ellipsis: true },
              ]}
            />
          </Space>
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="storageclasses"
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default StorageClassesPage
