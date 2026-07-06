import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Badge, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Tabs, Table, Typography, Input, Select, Button } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listPersistentVolumes, deletePersistentVolume, getPodEvents } from '@/services/k8s'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { AppPage } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { PersistentVolume } from '@/types'

const { Text } = Typography

const statusMap: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
  Available: 'success',
  Bound: 'success',
  Released: 'warning',
  Failed: 'error',
}

const PVPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [searchType, setSearchType] = useState<'name' | 'status' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailPV, setDetailPV] = useState<PersistentVolume | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-pvs', clusterId],
    queryFn: ({ signal }) => listPersistentVolumes(clusterId, signal),
    enabled: !!clusterId,
    refetchInterval: detailPV ? false : 60_000,
    staleTime: 60_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deletePersistentVolume(clusterId, name),
    onSuccess: () => {
      message.success('PV 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-pvs', clusterId] })
    },
  })

  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ['k8s-pv-events', clusterId, detailPV?.name],
    queryFn: ({ signal }) => getPodEvents(clusterId, '', detailPV?.name || '', signal),
    enabled: !!clusterId && !!detailPV && !!detailPV.name,
  })

  const filteredData = (data?.items || []).filter((item) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    if (searchType === 'status') return item.status?.toLowerCase().includes(v)
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
  })

  const columns: ProColumns<PersistentVolume>[] = [
    { title: '名称', dataIndex: 'name', width: 160, ellipsis: true, copyable: true, render: (_, r) => <Text strong>{r.name}</Text> },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      align: 'center' as const,
      render: (_, record) => (
        <Badge status={statusMap[record.status] || 'default'} text={record.status} />
      ),
    },
    { title: '容量', dataIndex: 'capacity', width: 100, align: 'center' as const },
    {
      title: '访问模式',
      dataIndex: 'accessModes',
      width: 180,
      align: 'center' as const,
      search: false,
      render: (_, record) => (record.accessModes || []).map((m) => <Tag key={m}>{m}</Tag>),
    },
    { title: '回收策略', dataIndex: 'reclaimPolicy', width: 120, search: false, align: 'center' as const },
    { title: 'StorageClass', dataIndex: 'storageClass', width: 130, search: false },
    {
      title: 'Claim',
      dataIndex: 'claim',
      width: 150,
      search: false,
      render: (_, r) => r.claim || '-',
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
            <a onClick={() => setDetailPV(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 PV？" onConfirm={() => deleteMutation.mutate(record.name)}>
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
      <ProTable<PersistentVolume>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey="name"
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1100 }}
        headerTitle={<Text strong>PV 列表</Text>}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : searchType === 'status' ? '按状态搜索' : '按标签搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'status', label: '状态' }, { value: 'label', label: '标签' }]} />}
            prefix={<SearchOutlined />}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>,
        ]}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`PV 详情 - ${detailPV?.name}`}
        open={!!detailPV}
        onClose={() => setDetailPV(null)}
        width={720}
        destroyOnClose
      >
        {detailPV && (
          <Tabs
            defaultActiveKey="overview"
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <Descriptions bordered column={2} size="small">
                    <Descriptions.Item label="名称">{detailPV.name}</Descriptions.Item>
                    <Descriptions.Item label="状态">
                      <Badge status={statusMap[detailPV.status] || 'default'} text={detailPV.status} />
                    </Descriptions.Item>
                    <Descriptions.Item label="容量">{detailPV.capacity}</Descriptions.Item>
                    <Descriptions.Item label="回收策略">
                      <Tag color="blue">{detailPV.reclaimPolicy}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="StorageClass" span={2}>
                      {detailPV.storageClass || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="访问模式" span={2}>
                      {(detailPV.accessModes || []).map((m) => <Tag key={m} color="blue">{m}</Tag>)}
                    </Descriptions.Item>
                    <Descriptions.Item label="Claim" span={2}>
                      {detailPV.claim || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间" span={2}>
                      {formatDate(detailPV.createdAt)}
                    </Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'events',
                label: '事件',
                children: (
                  <Table
                    size="small"
                    rowKey={(_, i) => String(i)}
                    loading={eventsLoading}
                    pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                    dataSource={eventsData || []}
                    columns={[
                      {
                        title: '类型',
                        dataIndex: 'type',
                        width: 90,
                        render: (_, r) => (
                          <Tag color={r.type === 'Warning' ? 'warning' : 'success'}>{r.type || 'Normal'}</Tag>
                        ),
                      },
                      { title: '原因', dataIndex: 'reason', width: 140, ellipsis: true },
                      { title: '消息', dataIndex: 'message', ellipsis: true },
                      {
                        title: '时间',
                        dataIndex: 'lastTimestamp',
                        width: 150,
                        render: (_, r) =>
                          r.lastTimestamp ? formatDate(r.lastTimestamp) : '-',
                      },
                    ]}
                  />
                ),
              },
            ]}
          />
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="pv"
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default PVPage
