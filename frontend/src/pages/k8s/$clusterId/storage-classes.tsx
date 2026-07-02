import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions } from 'antd'
import { DeleteOutlined, CodeOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listStorageClasses, deleteStorageClass } from '@/services/k8s'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { AppPage } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { StorageClass } from '@/types'

const StorageClassesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [detailSC, setDetailSC] = useState<StorageClass | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-storageclasses', clusterId],
    queryFn: () => listStorageClasses(clusterId),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteStorageClass(clusterId, name),
    onSuccess: () => {
      message.success('StorageClass 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-storageclasses', clusterId] })
    },
  })

  const columns: ProColumns<StorageClass>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    { title: 'Provisioner', dataIndex: 'provisioner', ellipsis: true },
    { title: '回收策略', dataIndex: 'reclaimPolicy', width: 120, search: false },
    {
      title: '绑定模式',
      dataIndex: 'volumeBindingMode',
      width: 140,
      search: false,
      render: (_, record) => <Tag>{record.volumeBindingMode}</Tag>,
    },
    {
      title: '默认',
      width: 80,
      search: false,
      render: (_, record) => (record.isDefault ? <Tag color="blue">是</Tag> : '-'),
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
            <a onClick={() => setDetailSC(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name)}>
              <CodeOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 StorageClass？"
            onConfirm={() => deleteMutation.mutate(record.name)}
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
      <ProTable<StorageClass>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey="name"
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1000 }}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`StorageClass 详情 - ${detailSC?.name}`}
        open={!!detailSC}
        onClose={() => setDetailSC(null)}
        width={640}
        destroyOnClose
      >
        {detailSC && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailSC.name}</Descriptions.Item>
            <Descriptions.Item label="Provisioner">{detailSC.provisioner}</Descriptions.Item>
            <Descriptions.Item label="回收策略">{detailSC.reclaimPolicy}</Descriptions.Item>
            <Descriptions.Item label="绑定模式">{detailSC.volumeBindingMode}</Descriptions.Item>
            <Descriptions.Item label="默认">{detailSC.isDefault ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDate(detailSC.createdAt)}</Descriptions.Item>
          </Descriptions>
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
