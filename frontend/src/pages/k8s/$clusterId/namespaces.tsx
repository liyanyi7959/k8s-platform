import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Badge, Space, Tooltip, Popconfirm, message, Button, Typography, Input, Drawer, Descriptions } from 'antd'
import { ProfileOutlined, DeleteOutlined, ReloadOutlined, SearchOutlined, EyeOutlined, PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listNamespaces, deleteNamespace } from '@/services/k8s'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { AppPage, ManifestApplyDrawer } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Namespace } from '@/types'

const { Text } = Typography

const NamespacesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [keyword, setKeyword] = useState('')
  const [detailNS, setDetailNS] = useState<Namespace | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data: namespaces, isLoading, refetch } = useQuery({
    queryKey: ['k8s-namespaces', clusterId],
    queryFn: ({ signal }) => listNamespaces(clusterId, signal),
    enabled: !!clusterId,
    refetchInterval: 30_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteNamespace(clusterId, name),
    onSuccess: () => {
      message.success('命名空间已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-namespaces', clusterId] })
    },
  })

  const filteredData = (namespaces || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const columns: ProColumns<Namespace>[] = [
    {
      title: 'Namespace',
      dataIndex: 'name',
      ellipsis: true,
      copyable: true,
      render: (_, record) => <Text strong>{record.name}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      align: 'center' as const,
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
      fixed: 'right',
      align: 'center' as const,
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailNS(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name)}>
              <ProfileOutlined />
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
        dataSource={filteredData}
        loading={isLoading}
        rowKey="name"
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 800 }}
        headerTitle={<Text strong>Namespace 列表</Text>}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder="按名称搜索"
            allowClear
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            style={{ width: 180 }}
            prefix={<SearchOutlined />}
          />,
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => refetch()}
          >
            刷新
          </Button>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建 Namespace
          </Button>,
        ]}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`Namespace 详情 - ${detailNS?.name}`}
        open={!!detailNS}
        onClose={() => setDetailNS(null)}
        width={640}
        destroyOnClose
      >
        {detailNS && (
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="名称">{detailNS.name}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Badge status={detailNS.status === 'Active' ? 'success' : 'warning'} text={detailNS.status} />
            </Descriptions.Item>
            <Descriptions.Item label="创建时间" span={2}>{formatDate(detailNS.createdAt)}</Descriptions.Item>
            <Descriptions.Item label="标签" span={2}>
              {detailNS.labels && Object.keys(detailNS.labels).length > 0
                ? Object.entries(detailNS.labels).map(([k, v]) => <Tag key={k}>{k}={v}</Tag>)
                : '-'}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="namespaces"
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      <ManifestApplyDrawer
        clusterId={clusterId}
        open={createOpen}
        onClose={() => { setCreateOpen(false); queryClient.invalidateQueries({ queryKey: ['k8s-namespaces', clusterId] }) }}
        title="创建 Namespace"
        initialYaml={`apiVersion: v1\nkind: Namespace\nmetadata:\n  name: my-namespace\n`}
      />
    </AppPage>
  )
}

export default NamespacesPage
