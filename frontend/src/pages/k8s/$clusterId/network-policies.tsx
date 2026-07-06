import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Drawer, Descriptions, Space, Tooltip, Button, Input, Select, Table, Typography } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined, PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listNetworkPolicies, deleteNetworkPolicy } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText, ManifestApplyDrawer } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { NetworkPolicy } from '@/types'

const { Text } = Typography

/** 格式化网络规则来源/目标 */
function formatPeer(peer: any): string {
  if (!peer) return '全部'
  if (peer.podSelector?.matchLabels) {
    const labels = Object.entries(peer.podSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(', ')
    return `Pod: ${labels || '全部'}`
  }
  if (peer.namespaceSelector?.matchLabels) {
    const labels = Object.entries(peer.namespaceSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(', ')
    return `Namespace: ${labels}`
  }
  if (peer.ipBlock?.cidr) return `IP: ${peer.ipBlock.cidr}`
  if (peer.ipBlock?.except) return `IP: ${peer.ipBlock.cidr} (排除 ${peer.ipBlock.except.join(', ')})`
  return '全部'
}

/** 格式化端口 */
function formatPorts(ports: any[]): string {
  if (!ports || ports.length === 0) return '全部'
  return ports.map((p) => `${p.protocol || 'TCP'}:${p.port || '*'}`).join(', ')
}

const NetworkPoliciesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [detailPolicy, setDetailPolicy] = useState<NetworkPolicy | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-networkpolicies', clusterId, namespace],
    queryFn: ({ signal }) => listNetworkPolicies(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailPolicy || createOpen ? false : 30_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: NetworkPolicy) =>
      deleteNetworkPolicy(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('NetworkPolicy 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-networkpolicies', clusterId] })
    },
  })

  const filteredData = (data?.items || []).filter((item) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    return item.labels && Object.entries(item.labels).some(([k, val]) =>
      `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
  })

  const columns: ProColumns<NetworkPolicy>[] = [
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
    {
      title: 'Pod 选择器',
      dataIndex: 'podSelector',
      width: 180,
      ellipsis: true,
      search: false,
      render: (_, record) => {
        const ps = record.podSelector
        if (!ps || ps === '<none>' || ps === '') return <Tag>全部 Pod</Tag>
        return <Text style={{ fontSize: 12 }}>{ps}</Text>
      },
    },
    {
      title: '策略类型',
      dataIndex: 'policyTypes',
      width: 150,
      align: 'center' as const,
      search: false,
      render: (_, record) =>
        (record.policyTypes || []).map((t) => (
          <Tag key={t} color="blue">{t}</Tag>
        )),
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
      width: 140,
      fixed: 'right',
      align: 'center' as const,
      render: (_, record) => (
        <Space size="small" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Tooltip title="详情">
            <a onClick={() => setDetailPolicy(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 NetworkPolicy？" onConfirm={() => deleteMutation.mutate(record)}>
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

  // 详情中 Ingress 规则数据
  const ingressRules = (detailPolicy?.ingress || []).map((rule, i) => ({
    key: i,
    from: (rule.from || []).map(formatPeer).join(' | ') || '全部',
    ports: formatPorts(rule.ports),
  }))

  // 详情中 Egress 规则数据
  const egressRules = (detailPolicy?.egress || []).map((rule, i) => ({
    key: i,
    to: (rule.to || []).map(formatPeer).join(' | ') || '全部',
    ports: formatPorts(rule.ports),
  }))

  return (
    <AppPage>
      <ProTable<NetworkPolicy>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 890 }}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : '按标签搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'label', label: '标签' }]} />}
            prefix={<SearchOutlined />}
          />,
          <NamespaceSelector
            key="ns"
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 180 }}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建 NetworkPolicy
          </Button>,
        ]}
        headerTitle={<Text strong>NetworkPolicy 列表</Text>}
      />

      <YamlDrawer
        clusterId={clusterId}
        resourceType="networkpolicies"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      <Drawer
        title={`NetworkPolicy 详情 - ${detailPolicy?.name}`}
        open={!!detailPolicy}
        onClose={() => setDetailPolicy(null)}
        width={720}
        destroyOnClose
      >
        {detailPolicy && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="名称">{detailPolicy.name}</Descriptions.Item>
              <Descriptions.Item label="Namespace"><Tag>{detailPolicy.namespace}</Tag></Descriptions.Item>
              <Descriptions.Item label="Pod 选择器" span={2}>
                {detailPolicy.podSelector === '<none>' || !detailPolicy.podSelector
                  ? <Tag>全部 Pod</Tag>
                  : <Tag color="blue">{detailPolicy.podSelector}</Tag>}
              </Descriptions.Item>
              <Descriptions.Item label="策略类型" span={2}>
                {(detailPolicy.policyTypes || []).map((t) => (
                  <Tag key={t} color="blue">{t}</Tag>
                ))}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间" span={2}>{formatDate(detailPolicy.createdAt)}</Descriptions.Item>
            </Descriptions>

            {ingressRules.length > 0 && (
              <Table
                size="small"
                title={() => `入站规则 (Ingress, ${ingressRules.length} 条)`}
                rowKey="key"
                pagination={false}
                dataSource={ingressRules}
                columns={[
                  { title: '来源', dataIndex: 'from', ellipsis: true },
                  { title: '端口', dataIndex: 'ports', width: 200 },
                ]}
              />
            )}

            {egressRules.length > 0 && (
              <Table
                size="small"
                title={() => `出站规则 (Egress, ${egressRules.length} 条)`}
                rowKey="key"
                pagination={false}
                dataSource={egressRules}
                columns={[
                  { title: '目标', dataIndex: 'to', ellipsis: true },
                  { title: '端口', dataIndex: 'ports', width: 200 },
                ]}
              />
            )}

            {ingressRules.length === 0 && egressRules.length === 0 && (
              <Text type="secondary">无入站/出站规则（默认行为：根据策略类型决定）</Text>
            )}
          </Space>
        )}
      </Drawer>

      <ManifestApplyDrawer
        clusterId={clusterId}
        open={createOpen}
        onClose={() => { setCreateOpen(false); queryClient.invalidateQueries({ queryKey: ['k8s-networkpolicies', clusterId] }) }}
        title="创建 NetworkPolicy"
        initialYaml={`apiVersion: networking.k8s.io/v1\nkind: NetworkPolicy\nmetadata:\n  name: my-netpol\n  namespace: ${namespace || 'default'}\nspec:\n  podSelector:\n    matchLabels:\n      app: my-app\n  policyTypes:\n  - Ingress\n  - Egress\n  ingress:\n  - from:\n    - podSelector:\n        matchLabels:\n          app: my-app\n    ports:\n    - protocol: TCP\n      port: 80\n`}
      />
    </AppPage>
  )
}

export default NetworkPoliciesPage
