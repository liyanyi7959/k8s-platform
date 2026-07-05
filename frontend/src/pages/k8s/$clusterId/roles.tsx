import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tabs, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Typography, Input, Button } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listRoles, listClusterRoles, deleteRole, deleteClusterRole } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { RBACRole } from '@/types'

const { Text } = Typography

const RolesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [keyword, setKeyword] = useState('')
  const [activeTab, setActiveTab] = useState('roles')
  const [detailRole, setDetailRole] = useState<RBACRole | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data: roles, isLoading: rolesLoading, refetch: refetchRoles } = useQuery({
    queryKey: ['k8s-roles', clusterId, namespace],
    queryFn: ({ signal }) => listRoles(clusterId, namespace, signal),
    enabled: !!clusterId && activeTab === 'roles',
    refetchInterval: detailRole ? false : 30_000,
  })

  const { data: clusterRoles, isLoading: clusterRolesLoading, refetch: refetchClusterRoles } = useQuery({
    queryKey: ['k8s-clusterroles', clusterId],
    queryFn: ({ signal }) => listClusterRoles(clusterId, signal),
    enabled: !!clusterId && activeTab === 'clusterroles',
    refetchInterval: detailRole ? false : 30_000,
  })

  const deleteRoleMutation = useMutation({
    mutationFn: (record: RBACRole) =>
      deleteRole(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('Role 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-roles', clusterId] })
    },
  })

  const deleteClusterRoleMutation = useMutation({
    mutationFn: (name: string) => deleteClusterRole(clusterId, name),
    onSuccess: () => {
      message.success('ClusterRole 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-clusterroles', clusterId] })
    },
  })

  const filteredRoles = (roles?.items || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const filteredClusterRoles = (clusterRoles?.items || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const roleColumns: ProColumns<RBACRole>[] = [
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
      title: '规则数',
      dataIndex: 'rules',
      width: 80,
      align: 'center' as const,
      search: false,
      render: (_, r) => r.rules?.length || 0,
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
            <a onClick={() => setDetailRole(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 Role？" onConfirm={() => deleteRoleMutation.mutate(record)}>
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

  const clusterRoleColumns: ProColumns<RBACRole>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
      copyable: true,
      render: (_, record) => <Text strong>{record.name}</Text>,
    },
    {
      title: '规则数',
      dataIndex: 'rules',
      width: 80,
      align: 'center' as const,
      search: false,
      render: (_, r) => r.rules?.length || 0,
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
            <a onClick={() => setDetailRole(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ClusterRole？"
            onConfirm={() => deleteClusterRoleMutation.mutate(record.name)}
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
            key: 'roles',
            label: 'Role（命名空间级）',
            children: (
              <ProTable<RBACRole>
                columns={roleColumns}
                dataSource={filteredRoles}
                loading={rolesLoading}
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
                    onClick={() => refetchRoles()}
                  >
                    刷新
                  </Button>,
                ]}
              />
            ),
          },
          {
            key: 'clusterroles',
            label: 'ClusterRole（集群级）',
            children: (
              <ProTable<RBACRole>
                columns={clusterRoleColumns}
                dataSource={filteredClusterRoles}
                loading={clusterRolesLoading}
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
                    onClick={() => refetchClusterRoles()}
                  >
                    刷新
                  </Button>,
                ]}
              />
            ),
          },
        ]}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`角色详情 - ${detailRole?.name}`}
        open={!!detailRole}
        onClose={() => setDetailRole(null)}
        width={640}
        destroyOnClose
      >
        {detailRole && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailRole.name}</Descriptions.Item>
            <Descriptions.Item label="Namespace">
              {detailRole.namespace || '集群级'}
            </Descriptions.Item>
            <Descriptions.Item label="规则数">{detailRole.rules?.length || 0}</Descriptions.Item>
            {detailRole.rules?.map((rule, idx) => (
              <Descriptions.Item key={idx} label={`规则 ${idx + 1}`}>
                <div>API Groups: {(rule.apiGroups || []).join(', ') || '*'}</div>
                <div>资源: {(rule.resources || []).join(', ')}</div>
                <div>操作: {(rule.verbs || []).join(', ')}</div>
              </Descriptions.Item>
            ))}
            <Descriptions.Item label="创建时间">
              {formatDate(detailRole.createdAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType={activeTab === 'roles' ? 'roles' : 'clusterroles'}
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default RolesPage
