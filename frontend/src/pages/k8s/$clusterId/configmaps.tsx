import React, { useState, useMemo } from 'react'
import {
  ProTable,
  ModalForm,
  ProFormText,
  ProFormTextArea,
  type ProColumns,
} from '@ant-design/pro-components'
import { Space, message, Popconfirm, Tag, Button, Tooltip, Input, Select, Typography, Drawer, Descriptions, Table } from 'antd'
import { PlusOutlined, EyeOutlined, DeleteOutlined, ProfileOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listConfigMaps, deleteConfigMap, createConfigMap } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ConfigMap } from '@/types'

const { Text } = Typography

const ConfigMapsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [selectedCM, setSelectedCM] = useState<ConfigMap | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['configmaps', clusterId, namespace],
    queryFn: ({ signal }) => listConfigMaps(clusterId, { namespace }, signal),
    refetchInterval: selectedCM || createOpen ? false : 60_000,
    staleTime: 60_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (params: { namespace: string; name: string }) =>
      deleteConfigMap(clusterId, params.namespace, params.name),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['configmaps', clusterId] })
    },
  })

  const filteredData = useMemo(() => (data?.items || []).filter((item: any) => {
    if (!searchValue) return true
    const v = searchValue.toLowerCase()
    if (searchType === 'name') return item.name.toLowerCase().includes(v)
    return item.data && Object.keys(item.data).some(k => k.toLowerCase().includes(v))
  }), [data, searchValue, searchType])

  const columns: ProColumns<ConfigMap>[] = [
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
      title: '数据项',
      width: 100,
      align: 'center' as const,
      search: false,
      render: (_, record) => <Tag color="blue">{Object.keys(record.data || {}).length} 项</Tag>,
    },
    {
      title: '数据键',
      width: 200,
      search: false,
      ellipsis: true,
      render: (_, record) => Object.keys(record.data || {}).join(', ') || '-',
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
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setSelectedCM(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ConfigMap？"
            description="删除后不可恢复"
            onConfirm={() =>
              deleteMutation.mutate({ namespace: record.namespace || namespace, name: record.name })
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
      <ProTable<ConfigMap>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey="name"
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 900 }}
        headerTitle={<Text strong>ConfigMap 列表</Text>}
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
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            创建 ConfigMap
          </Button>,
        ]}
      />

      <ModalForm
        title="创建 ConfigMap"
        open={createOpen}
        onOpenChange={setCreateOpen}
        modalProps={{ destroyOnClose: true }}
        onFinish={async (values) => {
          const targetNamespace = namespace || 'default'
          await createConfigMap(clusterId, targetNamespace, {
            metadata: { name: values.name, namespace: targetNamespace },
            data: values.data ? JSON.parse(values.data) : {},
          })
          message.success('创建成功')
          queryClient.invalidateQueries({ queryKey: ['configmaps', clusterId] })
          return true
        }}
      >
        <ProFormText
          name="name"
          label="名称"
          rules={[{ required: true, message: '请输入 ConfigMap 名称' }]}
        />
        <ProFormTextArea
          name="data"
          label="数据（JSON 格式）"
          placeholder={'{"key1": "value1", "key2": "value2"}'}
          fieldProps={{ rows: 6, style: { fontFamily: 'monospace' } }}
        />
      </ModalForm>

      <Drawer
        title={`ConfigMap 详情 - ${selectedCM?.name}`}
        open={!!selectedCM}
        onClose={() => setSelectedCM(null)}
        width={720}
        destroyOnClose
      >
        {selectedCM && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="名称">{selectedCM.name}</Descriptions.Item>
              <Descriptions.Item label="Namespace"><Tag>{selectedCM.namespace}</Tag></Descriptions.Item>
              <Descriptions.Item label="数据项" span={2}>{Object.keys(selectedCM.data || {}).length} 项</Descriptions.Item>
              <Descriptions.Item label="创建时间" span={2}>{formatDate(selectedCM.createdAt)}</Descriptions.Item>
            </Descriptions>
            {Object.keys(selectedCM.data || {}).length > 0 && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <Text type="secondary" style={{ fontSize: 13 }}>数据内容（{Object.keys(selectedCM.data || {}).length} 项）</Text>
                {Object.entries(selectedCM.data || {}).map(([k, v]) => (
                  <div key={k} style={{ border: '1px solid #d6e4ff', borderRadius: 6, overflow: 'hidden' }}>
                    <div style={{ padding: '6px 12px', background: '#f0f5ff', borderBottom: '1px solid #d6e4ff' }}>
                      <Tag color="blue" style={{ margin: 0, fontWeight: 500 }}>{k}</Tag>
                    </div>
                    <pre style={{ margin: 0, padding: 12, background: '#fafcff', fontSize: 13, fontFamily: 'monospace', whiteSpace: 'pre-wrap', wordBreak: 'break-all', maxHeight: 300, overflow: 'auto' }}>
                      {String(v)}
                    </pre>
                  </div>
                ))}
              </div>
            )}
          </Space>
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="configmaps"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default ConfigMapsPage
