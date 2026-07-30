import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions, Button, Input, Table, Tabs, Typography } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined, PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listPDBs, deletePDB, listPods } from '@/features/kops/api/k8s'
import { AppPage, EllipsisText } from '@/components'
import { NamespaceSelector, ManifestApplyDrawer } from '@/features/kops/components'
import YamlDrawer, { useYamlDrawer } from '@/features/kops/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { PDB } from '@/features/kops/types'

const { Text } = Typography

const PDBsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [keyword, setKeyword] = useState('')
  const [detailPDB, setDetailPDB] = useState<PDB | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-pdbs', clusterId, namespace],
    queryFn: ({ signal }) => listPDBs(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailPDB ? false : 60_000,
    staleTime: 60_000,
  })

  const { data: podsData, isLoading: podsLoading } = useQuery({
    queryKey: ['k8s-pdb-pods', clusterId, detailPDB?.namespace],
    queryFn: ({ signal }) => listPods(clusterId, { namespace: detailPDB?.namespace }, signal),
    enabled: !!clusterId && !!detailPDB && !!detailPDB.namespace,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: PDB) => deletePDB(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('PDB 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-pdbs', clusterId] })
    },
  })

  const filteredData = (data?.items || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const relatedPods = (podsData?.items || []).filter((pod) => {
    if (!detailPDB?.selector || !pod.labels) return false
    const selectorPairs = detailPDB.selector.split(',').flatMap((s) => {
      const [key, value] = s.split('=').map((part) => part.trim())
      return key ? [[key, value ?? ''] as const] : []
    })
    return selectorPairs.every(([key, value]) => pod.labels?.[key] === value)
  })

  const columns: ProColumns<PDB>[] = [
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
      title: '最小可用',
      dataIndex: 'minAvailable',
      width: 100,
      align: 'center' as const,
      search: false,
      render: (_, r) => (r.minAvailable != null ? <Tag color="green">{r.minAvailable}</Tag> : '-'),
    },
    {
      title: '最大不可用',
      dataIndex: 'maxUnavailable',
      width: 110,
      align: 'center' as const,
      search: false,
      render: (_, r) => (r.maxUnavailable != null ? <Tag color="orange">{r.maxUnavailable}</Tag> : '-'),
    },
    {
      title: '当前健康',
      dataIndex: 'currentHealthy',
      width: 100,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.currentHealthy || 0) - (b.currentHealthy || 0),
    },
    {
      title: '期望健康',
      dataIndex: 'desiredHealthy',
      width: 100,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.desiredHealthy || 0) - (b.desiredHealthy || 0),
    },
    {
      title: '允许中断',
      dataIndex: 'allowedDisruptions',
      width: 100,
      align: 'center' as const,
      search: false,
    },
    {
      title: '状态',
      width: 90,
      align: 'center' as const,
      search: false,
      render: (_, r) => (
        <Tag color={r.currentHealthy >= r.desiredHealthy ? 'success' : 'error'}>
          {r.currentHealthy >= r.desiredHealthy ? '健康' : '异常'}
        </Tag>
      ),
    },
    {
      title: 'Age',
      dataIndex: 'createdAt',
      width: 110,
      align: 'center' as const,
      search: false,
      ellipsis: true,
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
            <a onClick={() => setDetailPDB(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 PDB？" onConfirm={() => deleteMutation.mutate(record)}>
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
      <ProTable<PDB>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1160 }}
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
            style={{ width: 180 }}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>创建 PDB</Button>,
        ]}
        headerTitle={<Text strong>PDB 列表</Text>}
      />

      <Drawer
        title={`PDB 详情 - ${detailPDB?.name}`}
        open={!!detailPDB}
        onClose={() => setDetailPDB(null)}
        width={720}
        destroyOnClose
      >
        {detailPDB && (
          <Tabs
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <Descriptions bordered column={2} size="small">
                    <Descriptions.Item label="名称">{detailPDB.name}</Descriptions.Item>
                    <Descriptions.Item label="Namespace"><Tag>{detailPDB.namespace}</Tag></Descriptions.Item>
                    <Descriptions.Item label="最小可用">
                      {detailPDB.minAvailable != null ? <Tag color="green">{detailPDB.minAvailable}</Tag> : '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="最大不可用">
                      {detailPDB.maxUnavailable != null ? <Tag color="orange">{detailPDB.maxUnavailable}</Tag> : '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="当前健康">{detailPDB.currentHealthy}</Descriptions.Item>
                    <Descriptions.Item label="期望健康">{detailPDB.desiredHealthy}</Descriptions.Item>
                    <Descriptions.Item label="允许中断">{detailPDB.allowedDisruptions}</Descriptions.Item>
                    <Descriptions.Item label="创建时间">{formatDate(detailPDB.createdAt)}</Descriptions.Item>
                    {detailPDB.selector && (
                      <Descriptions.Item label="Selector" span={2}>
                        <Tag color="blue">{detailPDB.selector}</Tag>
                      </Descriptions.Item>
                    )}
                  </Descriptions>
                ),
              },
              {
                key: 'pods',
                label: '关联Pod',
                children: (
                  <Table
                    size="small"
                    rowKey="name"
                    loading={podsLoading}
                    pagination={false}
                    dataSource={relatedPods}
                    columns={[
                      { title: '名称', dataIndex: 'name', ellipsis: true },
                      {
                        title: '状态',
                        dataIndex: 'status',
                        width: 100,
                        render: (_, r) => {
                          const colorMap: Record<string, string> = { Running: 'success', Pending: 'processing', Failed: 'error', Succeeded: 'default' }
                          return <Tag color={colorMap[r.status] || 'default'}>{r.status}</Tag>
                        },
                      },
                      { title: '节点', dataIndex: 'nodeName', ellipsis: true },
                      { title: '重启', dataIndex: 'restarts', width: 80, align: 'center' as const },
                      { title: 'Age', dataIndex: 'createdAt', width: 110, align: 'center' as const, render: (_, r) => (r.createdAt ? formatDate(r.createdAt) : '-') },
                    ]}
                  />
                ),
              },
            ]}
          />
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="pdbs"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      <ManifestApplyDrawer
        clusterId={clusterId}
        open={createOpen}
        onClose={() => { setCreateOpen(false); queryClient.invalidateQueries({ queryKey: ['k8s-pdbs', clusterId] }) }}
        title="创建 PDB"
        initialYaml={`apiVersion: policy/v1\nkind: PodDisruptionBudget\nmetadata:\n  name: my-pdb\n  namespace: ${namespace || 'default'}\nspec:\n  minAvailable: 1\n  selector:\n    matchLabels:\n      app: my-app\n`}
      />
    </AppPage>
  )
}

export default PDBsPage
