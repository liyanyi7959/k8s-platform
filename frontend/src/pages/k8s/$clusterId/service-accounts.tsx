import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listServiceAccounts, deleteServiceAccount } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ServiceAccount } from '@/types'

const ServiceAccountsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailSA, setDetailSA] = useState<ServiceAccount | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-serviceaccounts', clusterId, namespace],
    queryFn: () => listServiceAccounts(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: ServiceAccount) =>
      deleteServiceAccount(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('ServiceAccount 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-serviceaccounts', clusterId] })
    },
  })

  const columns: ProColumns<ServiceAccount>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    { title: 'Secrets', dataIndex: 'secrets', width: 80, search: false },
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
            <a onClick={() => setDetailSA(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ServiceAccount？"
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
      <ProTable<ServiceAccount>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 700 }}
        headerTitle={
          <NamespaceSelector
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
            style={{ width: 200 }}
          />
        }
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`ServiceAccount 详情 - ${detailSA?.name}`}
        open={!!detailSA}
        onClose={() => setDetailSA(null)}
        width={640}
        destroyOnClose
      >
        {detailSA && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailSA.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              <Tag>{detailSA.namespace}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Secrets 数量">{detailSA.secrets ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDate(detailSA.createdAt)}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="serviceaccounts"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default ServiceAccountsPage
