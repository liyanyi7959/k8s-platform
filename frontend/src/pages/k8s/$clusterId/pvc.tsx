import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Badge, Popconfirm, message, Space, Tooltip, Drawer, Descriptions } from 'antd'
import { DeleteOutlined, CodeOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listPersistentVolumeClaims, deletePersistentVolumeClaim } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import type { PersistentVolumeClaim } from '@/types'

const statusMap: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
  Bound: 'success',
  Pending: 'warning',
  Lost: 'error',
}

const PVCPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailPVC, setDetailPVC] = useState<PersistentVolumeClaim | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-pvcs', clusterId, namespace],
    queryFn: () => listPersistentVolumeClaims(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: PersistentVolumeClaim) =>
      deletePersistentVolumeClaim(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('PVC 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-pvcs', clusterId] })
    },
  })

  const columns: ProColumns<PersistentVolumeClaim>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (_, record) => (
        <Badge status={statusMap[record.status] || 'default'} text={record.status} />
      ),
    },
    { title: 'Volume', dataIndex: 'volumeName', width: 150, search: false },
    { title: '容量', dataIndex: 'capacity', width: 100, search: false },
    {
      title: '访问模式',
      dataIndex: 'accessModes',
      width: 180,
      search: false,
      render: (_, record) => (record.accessModes || []).map((m) => <Tag key={m}>{m}</Tag>),
    },
    { title: 'StorageClass', dataIndex: 'storageClass', width: 130, search: false },
    { title: 'Age', dataIndex: 'age', width: 80, search: false },
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
            <a onClick={() => setDetailPVC(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <CodeOutlined />
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
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1000 }}
        headerTitle={
          <NamespaceSelector
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 200 }}
          />
        }
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`PVC 详情 - ${detailPVC?.name}`}
        open={!!detailPVC}
        onClose={() => setDetailPVC(null)}
        width={640}
        destroyOnClose
      >
        {detailPVC && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailPVC.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              <Tag>{detailPVC.namespace}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Badge status={statusMap[detailPVC.status] || 'default'} text={detailPVC.status} />
            </Descriptions.Item>
            <Descriptions.Item label="Volume">{detailPVC.volumeName || '-'}</Descriptions.Item>
            <Descriptions.Item label="容量">{detailPVC.capacity || '-'}</Descriptions.Item>
            <Descriptions.Item label="访问模式">
              {(detailPVC.accessModes || []).join(', ')}
            </Descriptions.Item>
            <Descriptions.Item label="StorageClass">{detailPVC.storageClass}</Descriptions.Item>
            <Descriptions.Item label="Age">{detailPVC.age || '-'}</Descriptions.Item>
          </Descriptions>
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
