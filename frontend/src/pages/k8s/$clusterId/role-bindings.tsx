import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tabs, Tag, Popconfirm, message, Drawer, Descriptions, Space, Tooltip } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined } from '@ant-design/icons'
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

const RoleBindingsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [activeTab, setActiveTab] = useState('rolebindings')
  const [detailBinding, setDetailBinding] = useState<RBACRoleBinding | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data: roleBindings, isLoading: rbLoading } = useQuery({
    queryKey: ['k8s-rolebindings', clusterId, namespace],
    queryFn: () => listRoleBindings(clusterId, namespace),
    enabled: !!clusterId && activeTab === 'rolebindings',
  })

  const { data: clusterRoleBindings, isLoading: crbLoading } = useQuery({
    queryKey: ['k8s-clusterrolebindings', clusterId],
    queryFn: () => listClusterRoleBindings(clusterId),
    enabled: !!clusterId && activeTab === 'clusterrolebindings',
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

  const rbColumns: ProColumns<RBACRoleBinding>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
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
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
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
                dataSource={roleBindings?.items || []}
                loading={rbLoading}
                rowKey={(r) => `${r.namespace}/${r.name}`}
                search={false}
                pagination={{
                  defaultPageSize: 20,
                  showSizeChanger: true,
                  showTotal: (t) => `共 ${t} 条`,
                }}
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
            ),
          },
          {
            key: 'clusterrolebindings',
            label: 'ClusterRoleBinding（集群级）',
            children: (
              <ProTable<RBACRoleBinding>
                columns={crbColumns}
                dataSource={clusterRoleBindings?.items || []}
                loading={crbLoading}
                rowKey="name"
                search={false}
                pagination={{
                  defaultPageSize: 20,
                  showSizeChanger: true,
                  showTotal: (t) => `共 ${t} 条`,
                }}
                scroll={{ x: 700 }}
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
            <Descriptions.Item label="命名空间">
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
