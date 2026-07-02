import React from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Badge, Space, Tooltip, Popconfirm, message, Button } from 'antd'
import { CodeOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listNamespaces, deleteNamespace } from '@/services/k8s'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { AppPage } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Namespace } from '@/types'

const NamespacesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const yamlDrawer = useYamlDrawer()

  const { data: namespaces, isLoading } = useQuery({
    queryKey: ['k8s-namespaces', clusterId],
    queryFn: () => listNamespaces(clusterId),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteNamespace(clusterId, name),
    onSuccess: () => {
      message.success('命名空间已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-namespaces', clusterId] })
    },
  })

  const columns: ProColumns<Namespace>[] = [
    {
      title: '命名空间',
      dataIndex: 'name',
      ellipsis: true,
      copyable: true,
      render: (_, record) => <strong>{record.name}</strong>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (_, record) => (
        <Badge status={record.status === 'Active' ? 'success' : 'warning'} text={record.status} />
      ),
    },
    {
      title: '标签',
      dataIndex: 'labels',
      search: false,
      ellipsis: true,
      render: (_, record) => {
        const labels = record.labels
        if (!labels || Object.keys(labels).length === 0) return '-'
        return Object.entries(labels)
          .slice(0, 3)
          .map(([k, v]) => (
            <Tag key={k}>
              {k}={v}
            </Tag>
          ))
      },
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
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name)}>
              <CodeOutlined />
            </a>
          </Tooltip>
          {record.name !== 'default' &&
            record.name !== 'kube-system' &&
            record.name !== 'kube-public' && (
              <Popconfirm
                title={`确定删除命名空间 ${record.name}？此操作不可恢复！`}
                onConfirm={() => deleteMutation.mutate(record.name)}
                okText="删除"
                cancelText="取消"
                okButtonProps={{ danger: true }}
              >
                <Tooltip title="删除">
                  <a style={{ color: '#ff4d4f' }}>
                    <DeleteOutlined />
                  </a>
                </Tooltip>
              </Popconfirm>
            )}
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <ProTable<Namespace>
        columns={columns}
        dataSource={namespaces || []}
        loading={isLoading}
        rowKey="name"
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 800 }}
        toolBarRender={() => [
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() =>
              queryClient.invalidateQueries({ queryKey: ['k8s-namespaces', clusterId] })
            }
          >
            刷新
          </Button>,
        ]}
      />

      <YamlDrawer
        clusterId={clusterId}
        resourceType="namespaces"
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default NamespacesPage
