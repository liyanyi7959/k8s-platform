import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Drawer, Descriptions, Space, Tooltip } from 'antd'
import { DeleteOutlined, CodeOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listReplicaSets, deleteReplicaSet } from '@/services/k8s'
import { AppPage, NamespaceSelector } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ReplicaSet } from '@/types'

const ReplicaSetsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailRS, setDetailRS] = useState<ReplicaSet | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-replicasets', clusterId, namespace],
    queryFn: () => listReplicaSets(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: ReplicaSet) =>
      deleteReplicaSet(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('ReplicaSet 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-replicasets', clusterId] })
    },
  })

  const columns: ProColumns<ReplicaSet>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 120,
      render: (_, r) => r.namespace || namespace,
    },
    { title: '期望', dataIndex: 'desired', width: 80, search: false },
    { title: '当前', dataIndex: 'current', width: 80, search: false },
    { title: '就绪', dataIndex: 'ready', width: 80, search: false },
    {
      title: '镜像',
      dataIndex: 'images',
      width: 200,
      ellipsis: true,
      search: false,
      render: (_, record) => (record.images || []).map((img) => <Tag key={img}>{img}</Tag>),
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
            <a onClick={() => setDetailRS(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <CodeOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ReplicaSet？"
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
      <ProTable<ReplicaSet>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 800 }}
        headerTitle={
          <NamespaceSelector
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 200 }}
          />
        }
      />

      <YamlDrawer
        clusterId={clusterId}
        resourceType="replicasets"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
      <Drawer
        title="ReplicaSet 详情"
        open={!!detailRS}
        onClose={() => setDetailRS(null)}
        width={600}
      >
        {detailRS && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="名称">{detailRS.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              {detailRS.namespace || namespace}
            </Descriptions.Item>
            <Descriptions.Item label="期望副本">{detailRS.desired ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="当前副本">{detailRS.current ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="就绪副本">{detailRS.ready ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="镜像">
              {(detailRS.images || []).join(', ') || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDate(detailRS.createdAt)}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </AppPage>
  )
}

export default ReplicaSetsPage
