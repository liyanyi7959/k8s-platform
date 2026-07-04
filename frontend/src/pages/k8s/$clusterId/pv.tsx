import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Badge, Popconfirm, message, Space, Tooltip, Drawer, Descriptions } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listPersistentVolumes, deletePersistentVolume } from '@/services/k8s'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { AppPage } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { PersistentVolume } from '@/types'

const statusMap: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
  Available: 'success',
  Bound: 'success',
  Released: 'warning',
  Failed: 'error',
}

const PVPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [detailPV, setDetailPV] = useState<PersistentVolume | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-pvs', clusterId],
    queryFn: () => listPersistentVolumes(clusterId),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deletePersistentVolume(clusterId, name),
    onSuccess: () => {
      message.success('PV 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-pvs', clusterId] })
    },
  })

  const columns: ProColumns<PersistentVolume>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (_, record) => (
        <Badge status={statusMap[record.status] || 'default'} text={record.status} />
      ),
    },
    { title: '容量', dataIndex: 'capacity', width: 100 },
    {
      title: '访问模式',
      dataIndex: 'accessModes',
      width: 180,
      search: false,
      render: (_, record) => (record.accessModes || []).map((m) => <Tag key={m}>{m}</Tag>),
    },
    { title: '回收策略', dataIndex: 'reclaimPolicy', width: 120, search: false },
    { title: 'StorageClass', dataIndex: 'storageClass', width: 130, search: false },
    {
      title: 'Claim',
      dataIndex: 'claim',
      width: 150,
      search: false,
      render: (_, r) => r.claim || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      search: false,
      render: (_, record) => formatDate(record.createdAt),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 140,
      fixed: 'right',
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
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey="name"
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1100 }}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`PV 详情 - ${detailPV?.name}`}
        open={!!detailPV}
        onClose={() => setDetailPV(null)}
        width={640}
        destroyOnClose
      >
        {detailPV && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailPV.name}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Badge status={statusMap[detailPV.status] || 'default'} text={detailPV.status} />
            </Descriptions.Item>
            <Descriptions.Item label="容量">{detailPV.capacity}</Descriptions.Item>
            <Descriptions.Item label="访问模式">
              {(detailPV.accessModes || []).join(', ')}
            </Descriptions.Item>
            <Descriptions.Item label="回收策略">{detailPV.reclaimPolicy}</Descriptions.Item>
            <Descriptions.Item label="StorageClass">{detailPV.storageClass}</Descriptions.Item>
            <Descriptions.Item label="Claim">{detailPV.claim || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDate(detailPV.createdAt)}</Descriptions.Item>
          </Descriptions>
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
