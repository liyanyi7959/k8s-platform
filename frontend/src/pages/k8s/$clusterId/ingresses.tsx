import React, { useState } from 'react'
import { ProTable, type ProColumns, ModalForm, ProFormText } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Button } from 'antd'
import {
  DeleteOutlined,
  ProfileOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listIngresses, deleteIngress, createIngress, updateIngress } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import type { Ingress } from '@/types'

const IngressesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailIngress, setDetailIngress] = useState<Ingress | null>(null)
  const [editOpen, setEditOpen] = useState(false)
  const [editingIngress, setEditingIngress] = useState<Ingress | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-ingresses', clusterId, namespace],
    queryFn: () => listIngresses(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: Ingress) =>
      deleteIngress(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('Ingress 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-ingresses', clusterId] })
    },
  })

  const createMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => {
      const ns = (values.namespace as string) || namespace || 'default'
      return createIngress(clusterId, ns, values)
    },
    onSuccess: () => {
      message.success('Ingress 创建成功')
      setCreateOpen(false)
      queryClient.invalidateQueries({ queryKey: ['k8s-ingresses', clusterId] })
    },
  })

  const updateMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => {
      if (!editingIngress) return
      return updateIngress(
        clusterId,
        editingIngress.namespace || namespace,
        editingIngress.name,
        values,
      )
    },
    onSuccess: () => {
      message.success('Ingress 更新成功')
      setEditOpen(false)
      setEditingIngress(null)
      queryClient.invalidateQueries({ queryKey: ['k8s-ingresses', clusterId] })
    },
  })

  const columns: ProColumns<Ingress>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    { title: 'Ingress Class', dataIndex: 'ingressClass', width: 140 },
    {
      title: 'Hosts',
      dataIndex: 'hosts',
      search: false,
      ellipsis: true,
      render: (_, record) =>
        (record.hosts || []).map((h) => (
          <Tag key={h} color="blue">
            {h}
          </Tag>
        )),
    },
    {
      title: 'Addresses',
      dataIndex: 'addresses',
      width: 150,
      search: false,
      render: (_, record) => (record.addresses || []).join(', ') || '-',
    },
    {
      title: 'TLS',
      width: 100,
      search: false,
      render: (_, record) => (record.tls ? <Tag color="green">启用</Tag> : <Tag>无</Tag>),
    },
    { title: 'Ports', dataIndex: 'ports', width: 100, search: false },
    { title: 'Age', dataIndex: 'age', width: 80, search: false },
    {
      title: '操作',
      valueType: 'option',
      width: 150,
      fixed: 'right',
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailIngress(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="编辑">
            <a
              onClick={() => {
                setEditingIngress(record)
                setEditOpen(true)
              }}
            >
              <EditOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 Ingress？" onConfirm={() => deleteMutation.mutate(record)}>
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
      <ProTable<Ingress>
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
        toolBarRender={() => [
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() =>
              queryClient.invalidateQueries({ queryKey: ['k8s-ingresses', clusterId] })
            }
          >
            刷新
          </Button>,
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            创建 Ingress
          </Button>,
        ]}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`Ingress 详情 - ${detailIngress?.name}`}
        open={!!detailIngress}
        onClose={() => setDetailIngress(null)}
        width={640}
        destroyOnClose
      >
        {detailIngress && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailIngress.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              <Tag>{detailIngress.namespace}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Ingress Class">
              {detailIngress.ingressClass || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Hosts">
              {(detailIngress.hosts || []).join(', ') || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Addresses">
              {(detailIngress.addresses || []).join(', ') || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Ports">{detailIngress.ports || '-'}</Descriptions.Item>
            <Descriptions.Item label="TLS">{detailIngress.tls ? '启用' : '无'}</Descriptions.Item>
            <Descriptions.Item label="Age">{detailIngress.age || '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      {/* 创建 Ingress 弹窗 */}
      <ModalForm
        title="创建 Ingress"
        open={createOpen}
        onOpenChange={setCreateOpen}
        onFinish={async (values) => {
          createMutation.mutate(values)
          return true
        }}
        width={640}
      >
        <ProFormText
          name="name"
          label="名称"
          rules={[{ required: true, message: '请输入名称' }]}
          placeholder="my-ingress"
        />
        <ProFormText
          name="namespace"
          label="命名空间"
          initialValue="default"
          rules={[{ required: true }]}
        />
        <ProFormText name="ingressClassName" label="Ingress Class" placeholder="nginx" />
        <ProFormText
          name="host"
          label="Host"
          rules={[{ required: true }]}
          placeholder="example.com"
        />
        <ProFormText name="path" label="Path" initialValue="/" placeholder="/" />
        <ProFormText
          name="serviceName"
          label="后端 Service 名称"
          rules={[{ required: true }]}
          placeholder="my-service"
        />
        <ProFormText
          name="servicePort"
          label="后端 Service 端口"
          rules={[{ required: true }]}
          initialValue="80"
        />
      </ModalForm>

      {/* 编辑 Ingress 弹窗 */}
      <ModalForm
        title={`编辑 Ingress - ${editingIngress?.name}`}
        open={editOpen}
        onOpenChange={(open) => {
          setEditOpen(open)
          if (!open) setEditingIngress(null)
        }}
        onFinish={async (values) => {
          updateMutation.mutate(values)
          return true
        }}
        width={640}
        initialValues={
          editingIngress
            ? {
                name: editingIngress.name,
                namespace: editingIngress.namespace,
                ingressClassName: editingIngress.ingressClass,
              }
            : {}
        }
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="名称" disabled />
        <ProFormText name="namespace" label="命名空间" disabled />
        <ProFormText name="ingressClassName" label="Ingress Class" />
        <ProFormText name="host" label="Host" />
        <ProFormText name="path" label="Path" />
        <ProFormText name="serviceName" label="后端 Service 名称" />
        <ProFormText name="servicePort" label="后端 Service 端口" />
      </ModalForm>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="ingresses"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default IngressesPage
