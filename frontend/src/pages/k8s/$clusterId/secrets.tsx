import React, { useState } from 'react'
import {
  ProTable,
  ModalForm,
  ProFormText,
  ProFormSelect,
  ProFormTextArea,
  type ProColumns,
} from '@ant-design/pro-components'
import { Space, message, Popconfirm, Tag, Modal, Button, Card, Tooltip, Input, Typography } from 'antd'
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
  const [selectedSecret, setSelectedSecret] = useState<Secret | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['secrets', clusterId, namespace],
    queryFn: () => listSecrets(clusterId, namespace),
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

  const columns: ProColumns<Secret>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 180,
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

      <Modal
        title={
          <span>
            Secret: <Tag color="blue">{selectedSecret?.name}</Tag>
          </span>
        }
        open={!!selectedSecret}
        onCancel={() => setSelectedSecret(null)}
        footer={null}
        width={700}
      >
        {selectedSecret && (
          <Card size="small" title="数据内容" style={{ background: '#fafafa' }}>
            <pre
              style={{
                margin: 0,
                padding: 16,
                background: '#f5f5f5',
                borderRadius: 6,
                overflow: 'auto',
                maxHeight: 400,
                fontSize: 13,
              }}
            >
              {selectedSecret.data?.__raw__ || JSON.stringify(selectedSecret.data, null, 2)}
            </pre>
          </Card>
        )}
      </Modal>

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
