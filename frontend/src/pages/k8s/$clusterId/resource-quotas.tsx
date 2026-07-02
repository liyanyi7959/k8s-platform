import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Drawer, Descriptions, Space, Tooltip } from 'antd'
import { DeleteOutlined, CodeOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listResourceQuotas, deleteResourceQuota } from '@/services/k8s'
import { AppPage, NamespaceSelector } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ResourceQuota } from '@/types'

const ResourceQuotasPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailQuota, setDetailQuota] = useState<ResourceQuota | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-resourcequotas', clusterId, namespace],
    queryFn: () => listResourceQuotas(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: ResourceQuota) =>
      deleteResourceQuota(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('ResourceQuota 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-resourcequotas', clusterId] })
    },
  })

  const columns: ProColumns<ResourceQuota>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 120,
      render: (_, r) => r.namespace || namespace,
    },
    {
      title: '硬限制',
      dataIndex: 'hard',
      width: 250,
      search: false,
      ellipsis: true,
      render: (_, record) => {
        const hard = record.hard || {}
        return Object.entries(hard).map(([k, v]) => (
          <Tag key={k} color="blue" style={{ marginBottom: 2 }}>
            {k}: {v}
          </Tag>
        ))
      },
    },
    {
      title: '已使用',
      dataIndex: 'used',
      width: 250,
      search: false,
      ellipsis: true,
      render: (_, record) => {
        const used = record.used || {}
        return Object.entries(used).map(([k, v]) => (
          <Tag key={k} color="orange" style={{ marginBottom: 2 }}>
            {k}: {v}
          </Tag>
        ))
      },
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
            <a onClick={() => setDetailQuota(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <CodeOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ResourceQuota？"
            onConfirm={() => deleteMutation.mutate(record)}
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
      <ProTable<ResourceQuota>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
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
      />
      <YamlDrawer
        clusterId={clusterId}
        resourceType="resourcequotas"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
      <Drawer
        title="资源配额详情"
        open={!!detailQuota}
        onClose={() => setDetailQuota(null)}
        width={600}
      >
        {detailQuota && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="名称">{detailQuota.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              {detailQuota.namespace || namespace}
            </Descriptions.Item>
            <Descriptions.Item label="硬限制">
              {Object.entries(detailQuota.hard || {}).map(([k, v]) => (
                <Tag key={k} color="blue" style={{ marginBottom: 2 }}>
                  {k}: {v}
                </Tag>
              ))}
            </Descriptions.Item>
            <Descriptions.Item label="已使用">
              {Object.entries(detailQuota.used || {}).map(([k, v]) => (
                <Tag key={k} color="orange" style={{ marginBottom: 2 }}>
                  {k}: {v}
                </Tag>
              ))}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDate(detailQuota.createdAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </AppPage>
  )
}

export default ResourceQuotasPage
