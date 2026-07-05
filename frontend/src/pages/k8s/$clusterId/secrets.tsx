import React, { useState } from 'react'
import {
  ProTable,
  ModalForm,
  ProFormText,
  ProFormSelect,
  ProFormTextArea,
  type ProColumns,
} from '@ant-design/pro-components'
import { Space, message, Popconfirm, Tag, Button, Tooltip, Input, Typography, Drawer, Descriptions, Table } from 'antd'
import { PlusOutlined, EyeOutlined, DeleteOutlined, ProfileOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listSecrets, deleteSecret, createSecret, getSecretReveal } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { Secret } from '@/types'

const { Text } = Typography

const secretTypeOptions = [
  { label: 'Opaque', value: 'Opaque' },
  { label: 'kubernetes.io/tls', value: 'kubernetes.io/tls' },
  { label: 'kubernetes.io/dockerconfigjson', value: 'kubernetes.io/dockerconfigjson' },
  { label: 'kubernetes.io/service-account-token', value: 'kubernetes.io/service-account-token' },
]

const SecretsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [keyword, setKeyword] = useState('')
  const [selectedSecret, setSelectedSecret] = useState<Secret | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['secrets', clusterId, namespace],
    queryFn: ({ signal }) => listSecrets(clusterId, namespace, signal),
    refetchInterval: selectedSecret || createOpen ? false : 30_000,
  })

  const handleViewSecret = async (record: Secret) => {
    try {
      const res = await getSecretReveal(clusterId, record.namespace, record.name)
      setSelectedSecret({ ...record, data: { __raw__: res.text } as Record<string, string> })
    } catch {
      setSelectedSecret(record)
      message.error('获取 Secret 明文失败')
    }
  }

  const deleteMutation = useMutation({
    mutationFn: (params: { namespace: string; name: string }) =>
      deleteSecret(clusterId, params.namespace, params.name),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['secrets', clusterId] })
    },
  })

  const filteredData = (data?.items || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const columns: ProColumns<Secret>[] = [
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
      title: '类型',
      dataIndex: 'type',
      width: 180,
      align: 'center' as const,
      render: (_, record) => {
        const type = record.type || 'Opaque'
        const colorMap: Record<string, string> = {
          Opaque: 'default',
          'kubernetes.io/tls': 'green',
          'kubernetes.io/dockerconfigjson': 'purple',
          'kubernetes.io/service-account-token': 'orange',
        }
        return <Tag color={colorMap[type] || 'default'}>{type}</Tag>
      },
    },
    {
      title: '数据键',
      width: 200,
      search: false,
      ellipsis: true,
      render: (_, record) => (record.dataKeys || []).join(', ') || '-',
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
            <a onClick={() => handleViewSecret(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 Secret？"
            description="删除后不可恢复"
            onConfirm={() =>
              deleteMutation.mutate({ namespace: record.namespace || namespace, name: record.name })
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
      <ProTable<Secret>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 900 }}
        headerTitle={<Text strong>Secret 列表</Text>}
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
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            创建 Secret
          </Button>,
        ]}
      />

      <ModalForm
        title="创建 Secret"
        open={createOpen}
        onOpenChange={setCreateOpen}
        modalProps={{ destroyOnClose: true }}
        onFinish={async (values) => {
          const targetNamespace = namespace || 'default'
          await createSecret(clusterId, targetNamespace, {
            metadata: { name: values.name, namespace: targetNamespace },
            type: values.type || 'Opaque',
            data: values.data ? JSON.parse(values.data) : {},
          })
          message.success('创建成功')
          queryClient.invalidateQueries({ queryKey: ['secrets', clusterId] })
          return true
        }}
      >
        <ProFormText
          name="name"
          label="名称"
          rules={[{ required: true, message: '请输入 Secret 名称' }]}
        />
        <ProFormSelect name="type" label="类型" options={secretTypeOptions} initialValue="Opaque" />
        <ProFormTextArea
          name="data"
          label="数据（JSON 格式）"
          placeholder={'{"username": "admin", "password": "base64encoded"}'}
          fieldProps={{ rows: 6, style: { fontFamily: 'monospace' } }}
        />
      </ModalForm>

      <Drawer
        title={`Secret 详情 - ${selectedSecret?.name}`}
        open={!!selectedSecret}
        onClose={() => setSelectedSecret(null)}
        width={720}
        destroyOnClose
      >
        {selectedSecret && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="名称">{selectedSecret.name}</Descriptions.Item>
              <Descriptions.Item label="Namespace"><Tag>{selectedSecret.namespace}</Tag></Descriptions.Item>
              <Descriptions.Item label="类型" span={2}><Tag color="blue">{selectedSecret.type || 'Opaque'}</Tag></Descriptions.Item>
              <Descriptions.Item label="创建时间" span={2}>{formatDate(selectedSecret.createdAt)}</Descriptions.Item>
            </Descriptions>
            {Object.keys(selectedSecret.data || {}).length > 0 && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <Text type="secondary" style={{ fontSize: 13 }}>数据内容（{Object.keys(selectedSecret.data || {}).length} 项）</Text>
                {Object.entries(selectedSecret.data || {}).map(([k, v]) => (
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
        resourceType="secrets"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default SecretsPage
