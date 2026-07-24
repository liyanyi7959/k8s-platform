import React, { useState } from 'react'
import { ProTable, type ProColumns, ModalForm, ProFormText } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Button, Input, Table, Typography, Select } from 'antd'
import { DeleteOutlined, ProfileOutlined, EditOutlined, EyeOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listIngresses, deleteIngress, createIngress, updateIngress } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Ingress } from '@/types'

const { Text } = Typography

const IngressesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailIngress, setDetailIngress] = useState<Ingress | null>(null)
  const [editOpen, setEditOpen] = useState(false)
  const [editingIngress, setEditingIngress] = useState<Ingress | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-ingresses', clusterId, namespace],
    queryFn: ({ signal }) => listIngresses(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailIngress || editOpen || createOpen ? false : 60_000,
    staleTime: 60_000,
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
      return updateIngress(clusterId, editingIngress.namespace || namespace, editingIngress.name, values)
    },
    onSuccess: () => {
      message.success('Ingress 更新成功')
      setEditOpen(false)
      setEditingIngress(null)
      queryClient.invalidateQueries({ queryKey: ['k8s-ingresses', clusterId] })
    },
  })

  const filteredData = (data?.items || []).filter((item) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
  })

  const columns: ProColumns<Ingress>[] = [
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
      render: (_, r) => <Text strong>{r.name}</Text>,
    },
    { title: 'Ingress Class', dataIndex: 'ingressClass', width: 130, ellipsis: true, search: false, render: (t) => t ? <Tag color="blue">{t as string}</Tag> : '-' },
    {
      title: 'Hosts',
      dataIndex: 'hosts',
      width: 180,
      ellipsis: true,
      search: false,
      render: (_, record) => {
        const hosts = record.hosts || []
        if (hosts.length === 0) return '-'
        const visible = hosts.slice(0, 2)
        const rest = hosts.length - visible.length
        return (
          <Space size={4} wrap>
            {visible.map((h) => <Tag key={h} color="blue">{h}</Tag>)}
            {rest > 0 && <Tooltip title={hosts.join(', ')}><Tag>+{rest}</Tag></Tooltip>}
          </Space>
        )
      },
    },
    {
      title: 'Addresses',
      dataIndex: 'addresses',
      width: 130,
      ellipsis: true,
      search: false,
      render: (_, record) => {
        const addrs = record.addresses || []
        return addrs.length > 0 ? <Text style={{ fontSize: 12 }}>{addrs.join(', ')}</Text> : '-'
      },
    },
    {
      title: 'TLS',
      width: 80,
      align: 'center' as const,
      search: false,
      render: (_, record) => (record.tls ? <Tag color="green">是</Tag> : <Tag>无</Tag>),
    },
    {
      title: 'Ports',
      dataIndex: 'ports',
      width: 90,
      align: 'center' as const,
      search: false,
      render: (t) => t as string || '-',
    },
    {
      title: 'Age',
      dataIndex: 'createdAt',
      width: 110,
      align: 'center' as const,
      search: false,
      ellipsis: true,
      sorter: (a, b) => new Date(a.createdAt || 0).getTime() - new Date(b.createdAt || 0).getTime(),
      render: (_, r) => (r.createdAt ? formatDate(r.createdAt) : '-'),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 150,
      fixed: 'right',
      align: 'center' as const,
      render: (_, record) => (
        <Space size="small" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Tooltip title="详情">
            <a onClick={() => setDetailIngress(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="编辑">
            <a onClick={() => { setEditingIngress(record); setEditOpen(true) }}>
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
              <a style={{ color: '#dc2626' }}><DeleteOutlined /></a>
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  // 详情路由规则数据
  const routeRules = (detailIngress?.rules || []).flatMap((rule: any, ri: number) => {
    const host = rule.host || '*'
    const paths = rule.http?.paths || []
    return paths.map((p: any, pi: number) => ({
      key: `${ri}-${pi}`,
      host: host,
      path: p.path || '/',
      pathType: p.pathType || 'Prefix',
      backend: p.backend?.service
        ? `${p.backend.service.name}:${p.backend.service.port?.number || p.backend.service.port?.name || '*'}`
        : p.backend?.resource?.kind
          ? `${p.backend.resource.kind}/${p.backend.resource.name}`
          : '-',
    }))
  })

  return (
    <AppPage>
      <ProTable<Ingress>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1170 }}
        toolBarRender={() => [
          <Input.Search key="search" placeholder={searchType === 'name' ? '按名称搜索' : '按标签搜索'} allowClear value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)} style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'label', label: '标签' }]} />}
            prefix={<SearchOutlined />} />,
          <NamespaceSelector key="ns" clusterId={clusterId} value={namespace} onChange={setNamespace} style={{ width: 180 }} />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>创建 Ingress</Button>,
        ]}
        headerTitle={<Text strong>Ingress 列表</Text>}
      />

      <YamlDrawer
        clusterId={clusterId}
        resourceType="ingresses"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`Ingress 详情 - ${detailIngress?.name}`}
        open={!!detailIngress}
        onClose={() => setDetailIngress(null)}
        width={720}
        destroyOnClose
      >
        {detailIngress && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="名称">{detailIngress.name}</Descriptions.Item>
              <Descriptions.Item label="Namespace"><Tag>{detailIngress.namespace}</Tag></Descriptions.Item>
              <Descriptions.Item label="Ingress Class" span={2}>{detailIngress.ingressClass || '-'}</Descriptions.Item>
              <Descriptions.Item label="Hosts" span={2}>
                {(detailIngress.hosts || []).map((h) => <Tag key={h} color="blue">{h}</Tag>)}
              </Descriptions.Item>
              <Descriptions.Item label="Addresses" span={2}>
                {(detailIngress.addresses || []).join(', ') || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="TLS">
                {detailIngress.tls ? <Tag color="green">启用</Tag> : <Tag>无</Tag>}
              </Descriptions.Item>
              <Descriptions.Item label="Ports">{detailIngress.ports || '-'}</Descriptions.Item>
              <Descriptions.Item label="创建时间" span={2}>{formatDate(detailIngress.createdAt)}</Descriptions.Item>
            </Descriptions>

            {routeRules.length > 0 && (
              <Table
                size="small"
                title={() => `路由规则 (${routeRules.length} 条)`}
                rowKey="key"
                pagination={false}
                dataSource={routeRules}
                columns={[
                  { title: 'Host', dataIndex: 'host', width: 150, render: (v) => <Tag color="blue">{v}</Tag> },
                  { title: 'Path', dataIndex: 'path', width: 120 },
                  { title: 'PathType', dataIndex: 'pathType', width: 100 },
                  { title: 'Backend', dataIndex: 'backend', ellipsis: true },
                ]}
              />
            )}

            {detailIngress.tlsConfigs && detailIngress.tlsConfigs.length > 0 && (
              <Table
                size="small"
                title={() => `TLS 配置 (${detailIngress.tlsConfigs.length} 条)`}
                rowKey={(_, i) => String(i)}
                pagination={false}
                dataSource={detailIngress.tlsConfigs}
                columns={[
                  { title: 'Hosts', dataIndex: 'hosts', render: (v: string[]) => v?.join(', ') || '-' },
                  { title: 'Secret', dataIndex: 'secretName', render: (v) => <Tag>{v}</Tag> },
                ]}
              />
            )}
          </Space>
        )}
      </Drawer>

      {/* 创建 Ingress 弹窗 */}
      <ModalForm
        title="创建 Ingress"
        open={createOpen}
        onOpenChange={setCreateOpen}
        onFinish={async (values) => { createMutation.mutate(values); return true }}
        width={640}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]} placeholder="my-ingress" />
        <ProFormText name="namespace" label="Namespace" initialValue="default" rules={[{ required: true }]} />
        <ProFormText name="ingressClassName" label="Ingress Class" placeholder="nginx" />
        <ProFormText name="host" label="Host" rules={[{ required: true }]} placeholder="example.com" />
        <ProFormText name="path" label="Path" initialValue="/" placeholder="/" />
        <ProFormText name="serviceName" label="后端 Service 名称" rules={[{ required: true }]} placeholder="my-service" />
        <ProFormText name="servicePort" label="后端 Service 端口" rules={[{ required: true }]} initialValue="80" />
      </ModalForm>

      {/* 编辑 Ingress 弹窗 */}
      <ModalForm
        title={`编辑 Ingress - ${editingIngress?.name}`}
        open={editOpen}
        onOpenChange={(open) => { setEditOpen(open); if (!open) setEditingIngress(null) }}
        onFinish={async (values) => { updateMutation.mutate(values); return true }}
        width={640}
        initialValues={editingIngress ? { name: editingIngress.name, namespace: editingIngress.namespace, ingressClassName: editingIngress.ingressClass } : {}}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="名称" disabled />
        <ProFormText name="namespace" label="Namespace" disabled />
        <ProFormText name="ingressClassName" label="Ingress Class" />
        <ProFormText name="host" label="Host" />
        <ProFormText name="path" label="Path" />
        <ProFormText name="serviceName" label="后端 Service 名称" />
        <ProFormText name="servicePort" label="后端 Service 端口" />
      </ModalForm>
    </AppPage>
  )
}

export default IngressesPage
