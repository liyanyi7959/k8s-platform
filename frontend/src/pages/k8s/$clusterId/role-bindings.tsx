import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tabs, Tag, Popconfirm, message, Drawer, Descriptions, Space, Tooltip, Typography, Input, Button } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listRoleBindings,
  listClusterRoleBindings,
  deleteRoleBinding,
  deleteClusterRoleBinding,
} from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { RBACRoleBinding } from '@/types'

const { Text } = Typography

const RoleBindingsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [keyword, setKeyword] = useState('')
  const [activeTab, setActiveTab] = useState('rolebindings')
  const [detailBinding, setDetailBinding] = useState<RBACRoleBinding | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data: roleBindings, isLoading: rbLoading, refetch: refetchRB } = useQuery({
    queryKey: ['k8s-rolebindings', clusterId, namespace],
    queryFn: ({ signal }) => listRoleBindings(clusterId, namespace, signal),
    enabled: !!clusterId && activeTab === 'rolebindings',
    refetchInterval: detailBinding ? false : 60_000,
    staleTime: 60_000,
  })

  const { data: clusterRoleBindings, isLoading: crbLoading, refetch: refetchCRB } = useQuery({
    queryKey: ['k8s-clusterrolebindings', clusterId],
    queryFn: ({ signal }) => listClusterRoleBindings(clusterId, signal),
    enabled: !!clusterId && activeTab === 'clusterrolebindings',
    refetchInterval: detailBinding ? false : 60_000,
    staleTime: 60_000,
  })

  const deleteRBMutation = useMutation({
    mutationFn: (record: RBACRoleBinding) =>
      deleteRoleBinding(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('RoleBinding 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-rolebindings', clusterId] })
    },
  })

  const deleteCRBMutation = useMutation({
    mutationFn: (name: string) => deleteClusterRoleBinding(clusterId, name),
    onSuccess: () => {
      message.success('ClusterRoleBinding 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-clusterrolebindings', clusterId] })
    },
  })

  const filteredRB = (roleBindings?.items || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const filteredCRB = (clusterRoleBindings?.items || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const rbColumns: ProColumns<RBACRoleBinding>[] = [
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
      render: (_, record) => <Text strong>{record.name}</Text>,
    },
    {
      title: '角色',
      dataIndex: 'roleRef',
      width: 150,
      search: false,
      render: (_, record) => (
        <Tag color="blue">
          {typeof record.roleRef === 'string' ? record.roleRef : record.roleRef?.name || '-'}
        </Tag>
      ),
    },
    {
      title: '主题',
      dataIndex: 'subjects',
      width: 200,
      align: 'center' as const,
      ellipsis: true,
      search: false,
      render: (_, record) =>
        (record.subjects || []).map((s) => (
          <Tag key={typeof s === 'string' ? s : s.name}>
            {typeof s === 'string' ? s : `${s.kind}/${s.name}`}
          </Tag>
        )),
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
      align: 'center' as const,
      fixed: 'right',
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailBinding(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 RoleBinding？"
            onConfirm={() => deleteRBMutation.mutate(record)}
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

  const crbColumns: ProColumns<RBACRoleBinding>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
      copyable: true,
      render: (_, record) => <Text strong>{record.name}</Text>,
    },
    {
      title: '角色',
      dataIndex: 'roleRef',
      width: 150,
      search: false,
      render: (_, record) => (
        <Tag color="blue">
          {typeof record.roleRef === 'string' ? record.roleRef : record.roleRef?.name || '-'}
        </Tag>
      ),
    },
    {
      title: '主题',
      dataIndex: 'subjects',
      width: 200,
      align: 'center' as const,
      ellipsis: true,
      search: false,
      render: (_, record) =>
        (record.subjects || []).map((s) => (
          <Tag key={typeof s === 'string' ? s : s.name}>
            {typeof s === 'string' ? s : `${s.kind}/${s.name}`}
          </Tag>
        )),
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
      align: 'center' as const,
      fixed: 'right',
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailBinding(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ClusterRoleBinding？"
            onConfirm={() => deleteCRBMutation.mutate(record.name)}
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
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'rolebindings',
            label: 'RoleBinding（命名空间级）',
            children: (
              <ProTable<RBACRoleBinding>
                columns={rbColumns}
                dataSource={filteredRB}
                loading={rbLoading}
                rowKey={(r) => `${r.namespace}/${r.name}`}
                search={false}
                options={{ reload: false }}
                pagination={{
                  defaultPageSize: 20,
                  showSizeChanger: true,
                  showTotal: (t) => `共 ${t} 条`,
                }}
                scroll={{ x: 800 }}
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
                  <NamespaceSelector
                    key="ns"
                    clusterId={clusterId}
                    value={namespace}
                    onChange={setNamespace}
                    style={{ width: 200 }}
                  />,
                  <Button
                    key="refresh"
                    icon={<ReloadOutlined />}
                    onClick={() => refetchRB()}
                  >
                    刷新
                  </Button>,
                ]}
              />
            ),
          },
          {
            key: 'clusterrolebindings',
            label: 'ClusterRoleBinding（集群级）',
            children: (
              <ProTable<RBACRoleBinding>
                columns={crbColumns}
                dataSource={filteredCRB}
                loading={crbLoading}
                rowKey="name"
                search={false}
                options={{ reload: false }}
                pagination={{
                  defaultPageSize: 20,
                  showSizeChanger: true,
                  showTotal: (t) => `共 ${t} 条`,
                }}
                scroll={{ x: 700 }}
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
                    onClick={() => refetchCRB()}
                  >
                    刷新
                  </Button>,
                ]}
              />
            ),
          },
        ]}
      />
      <YamlDrawer
        clusterId={clusterId}
        resourceType={activeTab === 'rolebindings' ? 'rolebindings' : 'clusterrolebindings'}
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
      <Drawer
        title="角色绑定详情"
        open={!!detailBinding}
        onClose={() => setDetailBinding(null)}
        width={600}
      >
        {detailBinding && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="名称">{detailBinding.name}</Descriptions.Item>
            <Descriptions.Item label="Namespace">
              {detailBinding.namespace || '集群级'}
            </Descriptions.Item>
            <Descriptions.Item label="角色">
              <Tag color="blue">
                {typeof detailBinding.roleRef === 'string'
                  ? detailBinding.roleRef
                  : detailBinding.roleRef?.name || '-'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="主题">
              {(detailBinding.subjects || []).map((s, i) => (
                <Tag key={i}>{typeof s === 'string' ? s : `${s.kind}/${s.name}`}</Tag>
              ))}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDate(detailBinding.createdAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </AppPage>
  )
}

export default RoleBindingsPage
