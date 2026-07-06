import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Badge, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Tabs, Table, Typography, Input, Button } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listPersistentVolumeClaims, deletePersistentVolumeClaim, getPodEvents } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { PersistentVolumeClaim } from '@/types'

const { Text } = Typography

const statusMap: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
  Bound: 'success',
  Pending: 'warning',
  Lost: 'error',
}

const PVCPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'status' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailPVC, setDetailPVC] = useState<PersistentVolumeClaim | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-pvcs', clusterId, namespace],
    queryFn: ({ signal }) => listPersistentVolumeClaims(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailPVC ? false : 30_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: PersistentVolumeClaim) =>
      deletePersistentVolumeClaim(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('PVC 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-pvcs', clusterId] })
    },
  })

  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ['k8s-pvc-events', clusterId, detailPVC?.namespace, detailPVC?.name],
    queryFn: ({ signal }) => getPodEvents(clusterId, detailPVC?.namespace || '', detailPVC?.name || '', signal),
    enabled: !!clusterId && !!detailPVC && !!detailPVC.namespace && !!detailPVC.name,
  })

  const filteredData = (data?.items || []).filter((item) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    if (searchType === 'status') return item.status?.toLowerCase().includes(v)
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
  })

  const columns: ProColumns<PersistentVolumeClaim>[] = [
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
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
    { title: 'Volume', dataIndex: 'volumeName', width: 150, search: false },
    { title: '容量', dataIndex: 'capacity', width: 100, search: false, align: 'center' as const },
    {
      title: '访问模式',
      dataIndex: 'accessModes',
      width: 180,
      search: false,
      render: (_, record) => (record.accessModes || []).map((m) => <Tag key={m}>{m}</Tag>),
    },
    { title: 'StorageClass', dataIndex: 'storageClass', width: 130, search: false },
    { title: 'Age', dataIndex: 'createdAt', width: 110, align: 'center' as const, search: false, ellipsis: true, render: (_, r) => r.createdAt ? formatDate(r.createdAt) : '-' },
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
            <a onClick={() => setDetailPVC(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 PVC？" onConfirm={() => deleteMutation.mutate(record)}>
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
      <ProTable<PersistentVolumeClaim>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1300 }}
        headerTitle={<Text strong>PVC 列表</Text>}
        toolBarRender={() => [
          <NamespaceSelector
            key="namespace"
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 200 }}
          />,
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
        title={`PVC 详情 - ${detailPVC?.name}`}
        open={!!detailPVC}
        onClose={() => setDetailPVC(null)}
        width={720}
        destroyOnClose
      >
        {detailPVC && (
          <Tabs
            defaultActiveKey="overview"
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <Descriptions bordered column={2} size="small">
                    <Descriptions.Item label="名称">{detailPVC.name}</Descriptions.Item>
                    <Descriptions.Item label="Namespace">
                      <Tag>{detailPVC.namespace}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="状态">
                      <Badge status={statusMap[detailPVC.status] || 'default'} text={detailPVC.status} />
                    </Descriptions.Item>
                    <Descriptions.Item label="容量">{detailPVC.capacity || '-'}</Descriptions.Item>
                    <Descriptions.Item label="Volume" span={2}>
                      {detailPVC.volumeName || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="访问模式" span={2}>
                      {(detailPVC.accessModes || []).map((m) => <Tag key={m} color="blue">{m}</Tag>)}
                    </Descriptions.Item>
                    <Descriptions.Item label="StorageClass" span={2}>
                      {detailPVC.storageClass || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="Age">{formatDate(detailPVC.createdAt)}</Descriptions.Item>
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
        resourceType="pvc"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default PVCPage
