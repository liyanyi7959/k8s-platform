/**
 * Service 管理页
 * 完整功能：列表+筛选+创建+编辑+详情+YAML+删除
 */
import React, { useState, useMemo } from 'react'
import {
  ProTable,
  type ProColumns,
  ModalForm,
  ProFormText,
  ProFormSelect,
  ProFormDigit,
  ProFormList,
} from '@ant-design/pro-components'
import { Button, Space, message, Popconfirm, Tag, Tooltip, Drawer, Descriptions, Input, Tabs, Table, Typography, Select } from 'antd'
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  ReloadOutlined,
  CopyOutlined,
  EyeOutlined,
  ProfileOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listServices, deleteService, createService, updateService, listPods, getPodEvents, listGenericResources } from '@/features/kops/api/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Service } from '@/shared/types'

const { Text } = Typography

const ServicesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'type' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [selectedKeys, setSelectedKeys] = useState<string[]>([])
  const [batchLoading, setBatchLoading] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editingService, setEditingService] = useState<Service | null>(null)
  const [detailService, setDetailService] = useState<Service | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['services', clusterId, namespace],
    queryFn: ({ signal }) => listServices(clusterId, { namespace: namespace || '' }, signal),
    enabled: !!clusterId,
    refetchInterval: detailService || editOpen || createOpen ? false : 60_000,
    staleTime: 60_000,
  })

  const { data: podsData, isLoading: podsLoading } = useQuery({
    queryKey: ['k8s-service-pods', clusterId, detailService?.namespace],
    queryFn: ({ signal }) => listPods(clusterId, { namespace: detailService?.namespace }, signal),
    enabled: !!clusterId && !!detailService && !!detailService.namespace,
  })

  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ['k8s-service-events', clusterId, detailService?.namespace, detailService?.name],
    queryFn: ({ signal }) => getPodEvents(clusterId, detailService?.namespace || '', detailService?.name || '', signal),
    enabled: !!clusterId && !!detailService && !!detailService.namespace && !!detailService.name,
  })

  // 获取 Endpoints 数据，构建 Service → Endpoint 数量映射
  const { data: endpointsData } = useQuery({
    queryKey: ['k8s-endpoints', clusterId, namespace],
    queryFn: ({ signal }) => listGenericResources(clusterId, 'endpoints', namespace || undefined, signal),
    enabled: !!clusterId,
    refetchInterval: detailService || editOpen || createOpen ? false : 60_000,
    staleTime: 60_000,
  })

  const endpointsCountMap = useMemo(() => {
    const map = new Map<string, number>()
    ;(endpointsData?.items || []).forEach((ep) => {
      const subsets = (ep.raw?.subsets || []) as any[]
      let count = 0
      subsets.forEach((subset) => {
        count += (subset.addresses?.length || 0)
      })
      map.set(`${ep.namespace}/${ep.name}`, count)
    })
    return map
  }, [endpointsData])

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

  const filteredData = (data?.items || []).filter((item) => {
    const matchKeyword = !searchValue || (() => {
      const v = searchValue.toLowerCase()
      if (searchType === 'name') return item.name.toLowerCase().includes(v)
      if (searchType === 'type') return item.type?.toLowerCase().includes(v)
      // label（Service 没有 labels，使用 selector 代替）
      return item.selector && Object.entries(item.selector).some(([k, val]) =>
        `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
    })()
    const matchType = !typeFilter || item.type === typeFilter
    return matchKeyword && matchType
  })

  const relatedPods = (podsData?.items || []).filter((pod: any) => {
    if (!detailService?.selector || typeof detailService.selector !== 'object') return false
    if (!pod.labels) return false
    return Object.entries(detailService.selector).every(([k, v]) => pod.labels?.[k] === v)
  })

  const columns: ProColumns<Service>[] = [
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={String(r.namespace || '')} tag />,
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
      render: (_, record) => <Text strong>{record.name}</Text>,
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
      render: (_, r) => {
        const ip = r.externalIP
        if (!ip || !ip.trim()) return '-'
        return (
          <Space>
            <Tag color="green">{ip}</Tag>
            <Tooltip title="复制">
              <CopyOutlined
                style={{ cursor: 'pointer', color: '#2563eb' }}
                onClick={() => copyExternalIP(ip)}
              />
            </Tooltip>
          </Space>
        )
      },
    },
    {
      title: '端口',
      dataIndex: 'ports',
      width: 150,
      render: (_, record) => {
        if (!record.ports || (typeof record.ports === 'string' && record.ports.length === 0))
          return '-'
        if (typeof record.ports === 'string') return record.ports
        const ports = record.ports as Array<{ port: number; nodePort?: number; protocol?: string }>
        const visible = ports.slice(0, 2)
        const rest = ports.length - visible.length
        return (
          <Space size={4} wrap>
            {visible.map((p, i) => (
              <Tag key={i} color="blue">
                {p.port}
                {p.nodePort ? `:${p.nodePort}` : ''}/{p.protocol || 'TCP'}
              </Tag>
            ))}
            {rest > 0 && (
              <Tooltip title={ports.slice(2).map((p) => `${p.port}${p.nodePort ? `:${p.nodePort}` : ''}/${p.protocol || 'TCP'}`).join(', ')}>
                <Tag>+{rest}</Tag>
              </Tooltip>
            )}
          </Space>
        )
      },
    },
    {
      title: 'Endpoints',
      width: 100,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (endpointsCountMap.get(`${a.namespace}/${a.name}`) || 0) - (endpointsCountMap.get(`${b.namespace}/${b.name}`) || 0),
      render: (_, r) => {
        const count = endpointsCountMap.get(`${r.namespace}/${r.name}`)
        return count != null ? <Tag color={count > 0 ? 'green' : 'red'}>{count}</Tag> : '-'
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
      width: 200,
      search: false,
      render: (_, record) => {
        if (!record.selector || typeof record.selector !== 'object') return '-'
        const entries = Object.entries(record.selector)
        if (entries.length === 0) return '-'
        const text = entries.map(([k, v]) => `${k}=${v}`).join(', ')
        return (
          <Tooltip title={text}>
            <Text style={{ fontSize: 12 }} ellipsis>
              {text}
            </Text>
          </Tooltip>
        )
      },
    },
    {
      title: 'Age',
      dataIndex: 'createdAt',
      width: 110,
      align: 'center' as const,
      ellipsis: true,
      sorter: (a, b) => new Date(a.createdAt || 0).getTime() - new Date(b.createdAt || 0).getTime(),
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
              <a style={{ color: '#dc2626' }}>
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
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1600 }}
        rowSelection={{
          selectedRowKeys: selectedKeys,
          onChange: (keys) => setSelectedKeys(keys as string[]),
        }}
        tableAlertRender={({ selectedRowKeys }) => (
          <Space>
            <span>已选 {selectedRowKeys.length} 个</span>
            <Popconfirm title={`确定删除选中的 ${selectedRowKeys.length} 个 Service？`} onConfirm={async () => {
              setBatchLoading(true)
              try {
                await Promise.all(
                  (selectedRowKeys as string[]).map((key) => {
                    const [ns, name] = key.split('/')
                    return deleteService(clusterId, ns, name)
                  })
                )
                message.success(`已删除 ${selectedRowKeys.length} 个 Service`)
                setSelectedKeys([])
                queryClient.invalidateQueries({ queryKey: ['services', clusterId] })
              } catch {
                message.error('批量删除失败')
              } finally {
                setBatchLoading(false)
              }
            }}>
              <Button danger size="small" loading={batchLoading}>批量删除</Button>
            </Popconfirm>
          </Space>
        )}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : searchType === 'type' ? '按类型搜索' : '按标签搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'type', label: '类型' }, { value: 'label', label: '标签' }]} />}
            prefix={<SearchOutlined />}
          />,
          <Select
            key="typeFilter"
            placeholder="类型筛选"
            allowClear
            value={typeFilter || undefined}
            onChange={(v) => setTypeFilter(v || '')}
            style={{ width: 140 }}
            options={[
              { label: 'ClusterIP', value: 'ClusterIP' },
              { label: 'NodePort', value: 'NodePort' },
              { label: 'LoadBalancer', value: 'LoadBalancer' },
              { label: 'ExternalName', value: 'ExternalName' },
            ]}
          />,
          <NamespaceSelector
            key="ns"
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 180 }}
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
          label="Namespace"
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
        <ProFormList
          name="ports"
          label="端口配置"
          initialValue={[{ port: 80, targetPort: 80, protocol: 'TCP' }]}
          min={1}
          creatorButtonProps={{ creatorButtonText: '添加端口' }}
        >
          <Space key="port-group">
            <ProFormDigit name="port" label="服务端口" rules={[{ required: true }]} min={1} max={65535} />
            <ProFormDigit name="targetPort" label="目标端口" rules={[{ required: true }]} min={1} max={65535} />
            <ProFormSelect name="protocol" label="协议" options={[{ label: 'TCP', value: 'TCP' }, { label: 'UDP', value: 'UDP' }]} />
          </Space>
        </ProFormList>
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
                ports:
                  Array.isArray(editingService.ports) && editingService.ports.length > 0
                    ? editingService.ports.map((p: { port: number; nodePort?: number; protocol?: string }) => ({
                        port: p.port,
                        targetPort: p.port,
                        protocol: p.protocol || 'TCP',
                      }))
                    : [{ port: 80, targetPort: 80, protocol: 'TCP' }],
              }
            : {}
        }
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="名称" disabled />
        <ProFormText name="namespace" label="Namespace" disabled />
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
        <ProFormList
          name="ports"
          label="端口配置"
          min={1}
          creatorButtonProps={{ creatorButtonText: '添加端口' }}
        >
          <Space key="port-group">
            <ProFormDigit name="port" label="服务端口" rules={[{ required: true }]} min={1} max={65535} />
            <ProFormDigit name="targetPort" label="目标端口" rules={[{ required: true }]} min={1} max={65535} />
            <ProFormSelect name="protocol" label="协议" options={[{ label: 'TCP', value: 'TCP' }, { label: 'UDP', value: 'UDP' }]} />
          </Space>
        </ProFormList>
        <ProFormText name="selector" label="选择器 (JSON)" placeholder='{"app": "my-app"}' />
      </ModalForm>

      {/* 详情抽屉 */}
      <Drawer
        title={`Service 详情 - ${detailService?.name}`}
        open={!!detailService}
        onClose={() => setDetailService(null)}
        width={720}
        destroyOnClose
      >
        {detailService && (
          <Tabs
            defaultActiveKey="overview"
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <Descriptions bordered column={2} size="small">
                    <Descriptions.Item label="名称">{detailService.name}</Descriptions.Item>
                    <Descriptions.Item label="Namespace">
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
                    <Descriptions.Item label="Session Affinity">
                      {typeof detailService.sessionAffinity === 'string'
                        ? detailService.sessionAffinity
                        : 'None'}
                    </Descriptions.Item>
                    <Descriptions.Item label="端口" span={2}>
                      {typeof detailService.ports === 'string'
                        ? detailService.ports
                        : JSON.stringify(detailService.ports)}
                    </Descriptions.Item>
                    <Descriptions.Item label="Selector" span={2}>
                      {detailService.selector &&
                      typeof detailService.selector === 'object' &&
                      Object.keys(detailService.selector).length > 0
                        ? JSON.stringify(detailService.selector)
                        : '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间" span={2}>
                      {formatDate(detailService.createdAt)}
                    </Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'pods',
                label: '关联 Pod',
                children: (
                  <Table
                    size="small"
                    rowKey="name"
                    loading={podsLoading}
                    pagination={false}
                    dataSource={relatedPods}
                    columns={[
                      {
                        title: '名称',
                        dataIndex: 'name',
                        ellipsis: true,
                        render: (_, r) => <Text strong>{r.name}</Text>,
                      },
                      {
                        title: '状态',
                        dataIndex: 'status',
                        width: 100,
                        render: (_, r) => {
                          const colorMap: Record<string, string> = {
                            Running: 'success',
                            Pending: 'processing',
                            Failed: 'error',
                            Succeeded: 'default',
                          }
                          return <Tag color={colorMap[r.status] || 'default'}>{r.status}</Tag>
                        },
                      },
                      { title: '节点', dataIndex: 'nodeName', ellipsis: true },
                      { title: '重启', dataIndex: 'restarts', width: 90, align: 'center' as const },
                      {
                        title: 'Age',
                        dataIndex: 'createdAt',
                        width: 110,
                        render: (_, r) => (r.createdAt ? formatDate(r.createdAt) : '-'),
                      },
                    ]}
                  />
                ),
              },
              {
                key: 'events',
                label: '事件',
                children: (
                  <Table
                    size="small"
                    rowKey={(_, i) => String(i)}
                    loading={eventsLoading}
                    pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                    dataSource={eventsData || []}
                    columns={[
                      {
                        title: '类型',
                        dataIndex: 'type',
                        width: 90,
                        render: (_, r) => (
                          <Tag color={r.type === 'Warning' ? 'warning' : 'success'}>{r.type || 'Normal'}</Tag>
                        ),
                      },
                      { title: '原因', dataIndex: 'reason', width: 140, ellipsis: true },
                      { title: '消息', dataIndex: 'message', ellipsis: true },
                      {
                        title: '时间',
                        dataIndex: 'lastTimestamp',
                        width: 150,
                        render: (_, r) =>
                          formatDate(r.lastTimestamp || r.firstTimestamp || r.metadata?.creationTimestamp || ''),
                      },
                    ]}
                  />
                ),
              },
            ]}
          />
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
