import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Drawer, Descriptions, Space, Tooltip } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listNetworkPolicies, deleteNetworkPolicy } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { NetworkPolicy } from '@/types'

const NetworkPoliciesPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailPolicy, setDetailPolicy] = useState<NetworkPolicy | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-networkpolicies', clusterId, namespace],
    queryFn: () => listNetworkPolicies(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: NetworkPolicy) =>
      deleteNetworkPolicy(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('NetworkPolicy 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-networkpolicies', clusterId] })
    },
  })

  const columns: ProColumns<NetworkPolicy>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: 'Pod 选择器',
      dataIndex: 'podSelector',
      width: 200,
      ellipsis: true,
      search: false,
      render: (_, record) => {
        const ps = record.podSelector
        if (!ps || Object.keys(ps).length === 0) return <Tag>全部 Pod</Tag>
        return Object.entries(ps).map(([k, v]) => (
          <Tag key={k}>
            {k}={v}
          </Tag>
        ))
      },
    },
    {
      title: '策略类型',
      dataIndex: 'policyTypes',
      width: 180,
      search: false,
      render: (_, record) =>
        (record.policyTypes || []).map((t) => (
          <Tag key={t} color="blue">
            {t}
          </Tag>
        )),
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
            <a onClick={() => setDetailPolicy(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 NetworkPolicy？"
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
      <ProTable<NetworkPolicy>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 800 }}
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
        resourceType="networkpolicies"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
      <Drawer
        title="NetworkPolicy 详情"
        open={!!detailPolicy}
        onClose={() => setDetailPolicy(null)}
        width={600}
      >
        {detailPolicy && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="名称">{detailPolicy.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              {detailPolicy.namespace || namespace}
            </Descriptions.Item>
            <Descriptions.Item label="Pod 选择器">
              {Object.entries(detailPolicy.podSelector || {}).length === 0 ? (
                <Tag>全部 Pod</Tag>
              ) : (
                Object.entries(detailPolicy.podSelector || {}).map(([k, v]) => (
                  <Tag key={k}>
                    {k}={v}
                  </Tag>
                ))
              )}
            </Descriptions.Item>
            <Descriptions.Item label="策略类型">
              {(detailPolicy.policyTypes || []).map((t) => (
                <Tag key={t} color="blue">
                  {t}
                </Tag>
              ))}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDate(detailPolicy.createdAt)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </AppPage>
  )
}

export default NetworkPoliciesPage
