/**
 * Service 管理页
 * 完整功能：列表+筛选+创建+编辑+详情+YAML+删除
 */
import React, { useState } from 'react'
import {
  ProTable,
  type ProColumns,
  ModalForm,
  ProFormText,
  ProFormSelect,
  ProFormDigit,
} from '@ant-design/pro-components'
import { Button, Space, message, Popconfirm, Tag, Tooltip, Drawer, Descriptions } from 'antd'
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  ReloadOutlined,
  CopyOutlined,
  EyeOutlined,
  ProfileOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listServices, deleteService, createService, updateService } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Service } from '@/types'

const ServicesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editingService, setEditingService] = useState<Service | null>(null)
  const [detailService, setDetailService] = useState<Service | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['services', clusterId, namespace],
    queryFn: () => listServices(clusterId, { namespace: namespace || '' }),
    enabled: !!clusterId,
    refetchInterval: 30_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (params: { namespace: string; name: string }) =>
      deleteService(clusterId, params.namespace, params.name),
    onSuccess: () => {
      message.success('Service 已删除')
      queryClient.invalidateQueries({ queryKey: ['services', clusterId] })
    },
  })

  const createMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => {
      const ns = (values.namespace as string) || namespace || 'default'
      return createService(clusterId, ns, values)
    },
    onSuccess: () => {
      message.success('Service 创建成功')
      setCreateOpen(false)
      queryClient.invalidateQueries({ queryKey: ['services', clusterId] })
    },
  })

  const updateMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => {
      if (!editingService) return
      return updateService(clusterId, editingService.namespace, editingService.name, values)
    },
    onSuccess: () => {
      message.success('Service 更新成功')
      setEditOpen(false)
      setEditingService(null)
      queryClient.invalidateQueries({ queryKey: ['services', clusterId] })
    },
  })

  const copyExternalIP = (ip: string) => {
    navigator.clipboard.writeText(ip).then(() => message.success('已复制'))
  }

  const columns: ProColumns<Service>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (_, record) => <a style={{ fontWeight: 500 }}>{record.name}</a>,
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (t) => <EllipsisText text={t as string} tag />,
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 130,
      render: (t) => {
        const v = t as string
        const colorMap: Record<string, string> = {
          ClusterIP: 'blue',
          NodePort: 'green',
          LoadBalancer: 'purple',
          ExternalName: 'orange',
        }
        return <Tag color={colorMap[v] || 'default'}>{v}</Tag>
      },
    },
    {
      title: 'Cluster-IP',
      dataIndex: 'clusterIP',
      width: 140,
      render: (t) => {
        const v = typeof t === 'string' ? t : ''
        return v ? <Tag color="blue">{v}</Tag> : '-'
      },
    },
    {
      title: '外部 IP',
      dataIndex: 'externalIP',
      width: 150,
      render: (t) => {
        const v = t as string
        return v ? (
          <Space>
            <Tag color="green">{v}</Tag>
            <Tooltip title="复制">
              <CopyOutlined
                style={{ cursor: 'pointer', color: '#1677ff' }}
                onClick={() => copyExternalIP(v)}
              />
            </Tooltip>
          </Space>
        ) : (
          '-'
        )
      },
    },
    {
      title: '端口',
      dataIndex: 'ports',
      width: 200,
      render: (_, record) => {
        if (!record.ports || (typeof record.ports === 'string' && record.ports.length === 0))
          return '-'
        if (typeof record.ports === 'string') return record.ports
        return record.ports.map(
          (p: { port: number; nodePort?: number; protocol?: string }, i: number) => (
            <Tag key={i} color="blue">
              {p.port}
              {p.nodePort ? `:${p.nodePort}` : ''}/{p.protocol || 'TCP'}
            </Tag>
          ),
        )
      },
    },
    {
      title: 'Endpoints',
      dataIndex: 'endpointsCount',
      width: 100,
      render: (t) => {
        const v = typeof t === 'number' ? t : undefined
        return <Tag>{v ?? '-'}</Tag>
      },
    },
    {
      title: 'Session Affinity',
      dataIndex: 'sessionAffinity',
      width: 140,
      render: (t) => <Tag>{typeof t === 'string' ? t : 'None'}</Tag>,
    },
    {
      title: 'Selector',
      dataIndex: 'selector',
      width: 180,
      ellipsis: true,
      render: (_, record) => {
        if (!record.selector || typeof record.selector !== 'object') return '-'
        const entries = Object.entries(record.selector)
        if (entries.length === 0) return '-'
        return entries.map(([k, val]) => (
          <Tag key={k} color="blue">
            {k}={String(val)}
          </Tag>
        ))
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 170,
      render: (_, record) => formatDate(record.createdAt),
    },
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
            <a onClick={() => setDetailService(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="编辑">
            <a
              onClick={() => {
                setEditingService(record)
                setEditOpen(true)
              }}
            >
              <EditOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 Service？"
            onConfirm={() =>
              deleteMutation.mutate({ namespace: record.namespace, name: record.name })
            }
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
      <ProTable<Service>
        headerTitle="Service 列表"
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 个 Service` }}
        scroll={{ x: 1200 }}
        toolBarRender={() => [
          <NamespaceSelector
            key="ns"
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>,
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            创建 Service
          </Button>,
        ]}
      />

      {/* 创建 Service 弹窗 */}
      <ModalForm
        title="创建 Service"
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
          placeholder="my-service"
        />
        <ProFormText
          name="namespace"
          label="命名空间"
          initialValue="default"
          rules={[{ required: true }]}
        />
        <ProFormSelect
          name="type"
          label="类型"
          initialValue="ClusterIP"
          rules={[{ required: true }]}
          options={[
            { label: 'ClusterIP', value: 'ClusterIP' },
            { label: 'NodePort', value: 'NodePort' },
            { label: 'LoadBalancer', value: 'LoadBalancer' },
          ]}
        />
        <ProFormSelect
          name="protocol"
          label="协议"
          initialValue="TCP"
          options={[
            { label: 'TCP', value: 'TCP' },
            { label: 'UDP', value: 'UDP' },
          ]}
        />
        <ProFormDigit
          name="port"
          label="服务端口"
          rules={[{ required: true }]}
          min={1}
          max={65535}
          initialValue={80}
        />
        <ProFormDigit
          name="targetPort"
          label="目标端口"
          rules={[{ required: true }]}
          min={1}
          max={65535}
          initialValue={80}
        />
        <ProFormText name="selector" label="选择器 (JSON)" placeholder='{"app": "my-app"}' />
      </ModalForm>

      {/* 编辑 Service 弹窗 */}
      <ModalForm
        title={`编辑 Service - ${editingService?.name}`}
        open={editOpen}
        onOpenChange={(open) => {
          setEditOpen(open)
          if (!open) setEditingService(null)
        }}
        onFinish={async (values) => {
          updateMutation.mutate(values)
          return true
        }}
        width={640}
        initialValues={
          editingService
            ? {
                name: editingService.name,
                namespace: editingService.namespace,
                type: editingService.type,
                clusterIP: editingService.clusterIP,
              }
            : {}
        }
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="名称" disabled />
        <ProFormText name="namespace" label="命名空间" disabled />
        <ProFormSelect
          name="type"
          label="类型"
          rules={[{ required: true }]}
          options={[
            { label: 'ClusterIP', value: 'ClusterIP' },
            { label: 'NodePort', value: 'NodePort' },
            { label: 'LoadBalancer', value: 'LoadBalancer' },
          ]}
        />
        <ProFormSelect
          name="protocol"
          label="协议"
          options={[
            { label: 'TCP', value: 'TCP' },
            { label: 'UDP', value: 'UDP' },
          ]}
        />
        <ProFormDigit name="port" label="服务端口" min={1} max={65535} />
        <ProFormDigit name="targetPort" label="目标端口" min={1} max={65535} />
        <ProFormText name="selector" label="选择器 (JSON)" placeholder='{"app": "my-app"}' />
      </ModalForm>

      {/* 详情抽屉 */}
      <Drawer
        title={`Service 详情 - ${detailService?.name}`}
        open={!!detailService}
        onClose={() => setDetailService(null)}
        width={640}
        destroyOnClose
      >
        {detailService && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailService.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              <Tag>{detailService.namespace}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="类型">
              <Tag color="blue">{detailService.type}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Cluster IP">
              {typeof detailService.clusterIP === 'string' && detailService.clusterIP
                ? detailService.clusterIP
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="External IP">
              {typeof detailService.externalIP === 'string' && detailService.externalIP
                ? detailService.externalIP
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="端口">
              {typeof detailService.ports === 'string'
                ? detailService.ports
                : JSON.stringify(detailService.ports)}
            </Descriptions.Item>
            <Descriptions.Item label="Selector">
              {detailService.selector &&
              typeof detailService.selector === 'object' &&
              Object.keys(detailService.selector).length > 0
                ? JSON.stringify(detailService.selector)
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Session Affinity">
              {typeof detailService.sessionAffinity === 'string'
                ? detailService.sessionAffinity
                : 'None'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDate(detailService.createdAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      {/* YAML 抽屉 */}
      <YamlDrawer
        clusterId={clusterId}
        resourceType="services"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default ServicesPage
