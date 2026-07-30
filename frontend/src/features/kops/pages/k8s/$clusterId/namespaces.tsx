import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Badge, Space, Tooltip, Popconfirm, message, Button, Typography, Input, Drawer, Descriptions, Tabs, Table, Select } from 'antd'
import { ProfileOutlined, DeleteOutlined, ReloadOutlined, SearchOutlined, EyeOutlined, PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listNamespaces, deleteNamespace, listResourceQuotas, listPods, listGenericResources, getPodEvents } from '@/features/kops/api/k8s'
import YamlDrawer, { useYamlDrawer } from '@/features/kops/components/YamlDrawer'
import { AppPage } from '@/components'
import { ManifestApplyDrawer } from '@/features/kops/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Namespace } from '@/features/kops/types'

const { Text } = Typography

const NamespacesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailNS, setDetailNS] = useState<Namespace | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  // ═══ Detail: ResourceQuotas / LimitRanges / Pods ═══
  const { data: rqData } = useQuery({
    queryKey: ['ns-resourcequotas', clusterId, detailNS?.name],
    queryFn: ({ signal }) => listResourceQuotas(clusterId, detailNS!.name, signal),
    enabled: !!detailNS,
  })

  const { data: lrData } = useQuery({
    queryKey: ['ns-limitranges', clusterId, detailNS?.name],
    queryFn: ({ signal }) => listGenericResources(clusterId, 'limitranges', detailNS!.name, signal),
    enabled: !!detailNS,
  })

  const { data: nsPodsData } = useQuery({
    queryKey: ['ns-pods', clusterId, detailNS?.name],
    queryFn: ({ signal }) => listPods(clusterId, { namespace: detailNS!.name }, signal),
    enabled: !!detailNS,
  })

  const { data: nsEventsData } = useQuery({
    queryKey: ['ns-events', clusterId, detailNS?.name],
    queryFn: ({ signal }) => getPodEvents(clusterId, detailNS!.name, '', signal),
    enabled: !!detailNS,
  })

  const { data: namespaces, isLoading, refetch } = useQuery({
    queryKey: ['k8s-namespaces', clusterId],
    queryFn: ({ signal }) => listNamespaces(clusterId, signal),
    enabled: !!clusterId,
    refetchInterval: detailNS || createOpen ? false : 60_000,
    staleTime: 60_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteNamespace(clusterId, name),
    onSuccess: () => {
      message.success('命名空间已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-namespaces', clusterId] })
    },
  })

  const filteredData = (namespaces || []).filter((item) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') {
      return item.name.toLowerCase().includes(v)
    }
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) ||
      k.toLowerCase().includes(v) ||
      String(val).toLowerCase().includes(v)
    )
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
        const text = Object.entries(labels).slice(0, 3).map(([k, v]) => `${k}=${v}`).join(', ')
        return <Text style={{ fontSize: 12 }}>{text}</Text>
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
                  <a style={{ color: '#dc2626' }}>
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
            placeholder={searchType === 'name' ? '按名称搜索' : '按标签搜索 (如 kubernetes.io)'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={
              <Select
                value={searchType}
                onChange={(v) => setSearchType(v)}
                style={{ width: 70 }}
                options={[
                  { value: 'name', label: '名称' },
                  { value: 'label', label: '标签' },
                ]}
              />
            }
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
        width={720}
        destroyOnClose
      >
        {detailNS && (
          <Tabs
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <Descriptions bordered column={2} size="small">
                    <Descriptions.Item label="名称">{detailNS.name}</Descriptions.Item>
                    <Descriptions.Item label="状态">
                      <Badge status={detailNS.status === 'Active' ? 'success' : 'warning'} text={detailNS.status} />
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间" span={2}>{formatDate(detailNS.createdAt)}</Descriptions.Item>
                    <Descriptions.Item label="标签" span={2}>
                      {detailNS.labels && Object.keys(detailNS.labels).length > 0
                        ? Object.entries(detailNS.labels).map(([k, v]) => <Text key={k} style={{ fontSize: 12 }}>{k}={v}; </Text>)
                        : '-'}
                    </Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'quotas',
                label: `配额 (${rqData?.items?.length || 0})`,
                children: (rqData?.items || []).length > 0 ? (
                  <Space direction="vertical" style={{ width: '100%' }} size="middle">
                    {(rqData?.items || []).map((rq: any) => (
                      <div key={rq.name}>
                        <Table
                          size="small"
                          title={() => rq.name}
                          rowKey="resource"
                          pagination={false}
                          dataSource={Object.keys(rq.hard || {}).map(k => ({
                            resource: k,
                            hard: rq.hard?.[k] || '-',
                            used: rq.used?.[k] || '-',
                          }))}
                          columns={[
                            { title: '资源', dataIndex: 'resource', width: 160, ellipsis: true },
                            { title: '硬限制', dataIndex: 'hard', align: 'center' },
                            { title: '已使用', dataIndex: 'used', align: 'center' },
                          ]}
                        />
                      </div>
                    ))}
                  </Space>
                ) : <Text type="secondary">该命名空间暂无 ResourceQuota</Text>,
              },
              {
                key: 'limits',
                label: `限制范围 (${lrData?.items?.length || 0})`,
                children: (lrData?.items || []).length > 0 ? (
                  <Space direction="vertical" style={{ width: '100%' }} size="middle">
                    {(lrData?.items || []).map((lr: any) => {
                      const limits = (lr.raw?.spec?.limits || []) as Array<{
                        type?: string
                        max?: Record<string, string>
                        min?: Record<string, string>
                        default?: Record<string, string>
                      }>
                      return (
                        <Table
                          key={lr.name}
                          size="small"
                          title={() => lr.name}
                          rowKey={(_, i) => String(i)}
                          pagination={false}
                          dataSource={limits}
                          columns={[
                            { title: '类型', dataIndex: 'type', width: 100 },
                            { title: 'Max', render: (_, r) => r.max ? Object.entries(r.max).map(([k, v]) => `${k}=${v}`).join(', ') : '-', ellipsis: true },
                            { title: 'Min', render: (_, r) => r.min ? Object.entries(r.min).map(([k, v]) => `${k}=${v}`).join(', ') : '-', ellipsis: true },
                            { title: 'Default', render: (_, r) => r.default ? Object.entries(r.default).map(([k, v]) => `${k}=${v}`).join(', ') : '-', ellipsis: true },
                          ]}
                        />
                      )
                    })}
                  </Space>
                ) : <Text type="secondary">该命名空间暂无 LimitRange</Text>,
              },
              {
                key: 'pods',
                label: `Pods (${nsPodsData?.items?.length || 0})`,
                children: (nsPodsData?.items || []).length > 0 ? (
                  <Table
                    size="small"
                    rowKey={(r: any) => `${r.namespace}/${r.name}`}
                    pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                    dataSource={nsPodsData?.items || []}
                    columns={[
                      { title: '名称', dataIndex: 'name', ellipsis: true },
                      { title: '状态', dataIndex: 'status', width: 100, align: 'center' as const, render: (v: string) => {
                        const colorMap: Record<string, string> = { Running: 'success', Pending: 'processing', Failed: 'error', Succeeded: 'default' }
                        return <Badge status={colorMap[v] as any || 'default'} text={v} />
                      }},
                      { title: '重启', dataIndex: 'restarts', width: 70, align: 'center' as const },
                      { title: 'Age', dataIndex: 'createdAt', width: 110, align: 'center' as const, ellipsis: true, render: (v: string) => v ? formatDate(v) : '-' },
                    ]}
                  />
                ) : <Text type="secondary">该命名空间暂无 Pod</Text>,
              },
              {
                key: 'events',
                label: `事件 (${nsEventsData?.length || 0})`,
                children: (nsEventsData || []).length > 0 ? (
                  <Table
                    size="small"
                    rowKey={(_, i) => String(i)}
                    pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                    dataSource={nsEventsData || []}
                    columns={[
                      { title: '类型', dataIndex: 'type', width: 80, align: 'center' as const, render: (v) => <Badge status={v === 'Warning' ? 'warning' : 'success'} text={v || 'Normal'} /> },
                      { title: '原因', dataIndex: 'reason', width: 140, ellipsis: true },
                      { title: '消息', dataIndex: 'message', ellipsis: true },
                      { title: '时间', dataIndex: 'lastTimestamp', width: 160, render: (v) => v ? formatDate(v) : '-' },
                    ]}
                  />
                ) : <Text type="secondary">该命名空间暂无事件</Text>,
              },
            ]}
          />
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
