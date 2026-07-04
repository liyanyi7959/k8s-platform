import React, { useState } from 'react'
import {
  ProTable,
  ModalForm,
  ProFormText,
  ProFormTextArea,
  type ProColumns,
} from '@ant-design/pro-components'
import { Space, message, Popconfirm, Tag, Modal, Button, Card, Tooltip } from 'antd'
import { PlusOutlined, EyeOutlined, DeleteOutlined, ProfileOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listConfigMaps, deleteConfigMap, createConfigMap } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ConfigMap } from '@/types'

const ConfigMapsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [selectedCM, setSelectedCM] = useState<ConfigMap | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['configmaps', clusterId, namespace],
    queryFn: () => listConfigMaps(clusterId, { namespace }),
  })

  const deleteMutation = useMutation({
    mutationFn: (params: { namespace: string; name: string }) =>
      deleteConfigMap(clusterId, params.namespace, params.name),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['configmaps', clusterId] })
    },
  })

  const columns: ProColumns<ConfigMap>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: '数据项',
      width: 100,
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
      <ProTable<ConfigMap>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey="name"
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 900 }}
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

      <Modal
        title={
          <span>
            ConfigMap: <Tag color="blue">{selectedCM?.name}</Tag>
          </span>
        }
        open={!!selectedCM}
        onCancel={() => setSelectedCM(null)}
        footer={null}
        width={700}
      >
        {selectedCM && (
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
              {JSON.stringify(selectedCM.data, null, 2)}
            </pre>
          </Card>
        )}
      </Modal>

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
