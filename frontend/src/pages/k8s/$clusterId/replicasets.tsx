import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Drawer, Descriptions, Space, Tooltip, Button, Input, Table, Badge, Typography } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { history } from '@umijs/max'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listReplicaSets, deleteReplicaSet } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ReplicaSet } from '@/types'

const { Text } = Typography

const ReplicaSetsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [keyword, setKeyword] = useState('')
  const [detailRS, setDetailRS] = useState<ReplicaSet | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-replicasets', clusterId, namespace],
    queryFn: ({ signal }) => listReplicaSets(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailRS ? false : 30_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: ReplicaSet) =>
      deleteReplicaSet(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('ReplicaSet 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-replicasets', clusterId] })
    },
  })

  // 客户端按名称过滤
  const filteredData = (data?.items || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const columns: ProColumns<ReplicaSet>[] = [
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
      title: '期望',
      dataIndex: 'desired',
      width: 80,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.desired || 0) - (b.desired || 0),
    },
    {
      title: '当前',
      dataIndex: 'current',
      width: 80,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.current || 0) - (b.current || 0),
    },
    {
      title: '就绪',
      dataIndex: 'ready',
      width: 90,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.ready || 0) - (b.ready || 0),
      render: (_, r) => {
        const ok = (r.ready || 0) === (r.desired || 0)
        return <Badge status={ok ? 'success' : 'processing'} text={`${r.ready ?? 0}/${r.desired ?? 0}`} />
      },
    },
    {
      title: 'Owner',
      width: 160,
      search: false,
      ellipsis: true,
      render: (_, r) =>
        r.ownerKind ? (
          <Tooltip title={`${r.ownerKind} / ${r.ownerName}`}>
            <Tag color="blue">{r.ownerKind}</Tag>
            <Text style={{ fontSize: 12 }}>{r.ownerName}</Text>
          </Tooltip>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '镜像',
      dataIndex: 'images',
      width: 200,
      ellipsis: true,
      search: false,
      render: (_, record) => {
        const images = record.images || []
        return images.length > 0 ? (
          <Tooltip title={images.join('\n')}>
            <Text style={{ fontSize: 11 }}>{images[0]}</Text>
          </Tooltip>
        ) : (
          '-'
        )
      },
    },
    {
      title: 'Age',
      dataIndex: 'createdAt',
      width: 110,
      align: 'center' as const,
      search: false,
      ellipsis: true,
      sorter: (a, b) => new Date(a.createdAt || 0).getTime() - new Date(b.createdAt || 0).getTime(),
      render: (_, record) => (record.createdAt ? formatDate(record.createdAt) : '-'),
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
            <a onClick={() => setDetailRS(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 ReplicaSet？"
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
      <ProTable<ReplicaSet>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
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
        ]}
        headerTitle={<Text strong>ReplicaSet 列表</Text>}
      />

      <YamlDrawer
        clusterId={clusterId}
        resourceType="replicasets"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      <Drawer
        title={`详情 - ${detailRS?.name}`}
        open={!!detailRS}
        onClose={() => setDetailRS(null)}
        width={640}
      >
        {detailRS && (
          <>
            <Descriptions column={2} size="small" bordered>
              <Descriptions.Item label="名称">{detailRS.name}</Descriptions.Item>
              <Descriptions.Item label="命名空间">{detailRS.namespace || namespace}</Descriptions.Item>
              <Descriptions.Item label="期望副本">{detailRS.desired ?? 0}</Descriptions.Item>
              <Descriptions.Item label="当前副本">{detailRS.current ?? 0}</Descriptions.Item>
              <Descriptions.Item label="就绪副本">
                <Badge
                  status={detailRS.ready === detailRS.desired ? 'success' : 'processing'}
                  text={`${detailRS.ready ?? 0}/${detailRS.desired ?? 0}`}
                />
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">{formatDate(detailRS.createdAt)}</Descriptions.Item>
              {detailRS.ownerKind && (
                <Descriptions.Item label="Owner" span={2}>
                  <Tag color="blue">{detailRS.ownerKind}</Tag>
                  {detailRS.ownerKind === 'Deployment' ? (
                    <a onClick={() => history.push(`/k8s/${clusterId}/deployments`)}>{detailRS.ownerName}</a>
                  ) : (
                    <Text>{detailRS.ownerName}</Text>
                  )}
                </Descriptions.Item>
              )}
              <Descriptions.Item label="镜像" span={2}>
                {(detailRS.images || []).map((img) => <Tag key={img}>{img}</Tag>)}
              </Descriptions.Item>
            </Descriptions>
            <Table
              style={{ marginTop: 16 }}
              size="small"
              rowKey={(_, i) => String(i)}
              pagination={false}
              dataSource={(detailRS.images || []).map((image, index) => ({
                key: index,
                name: `${detailRS.name}-${index + 1}`,
                image,
              }))}
              columns={[
                { title: '容器名称', dataIndex: 'name' },
                { title: '镜像', dataIndex: 'image' },
              ]}
            />
          </>
        )}
      </Drawer>
    </AppPage>
  )
}

export default ReplicaSetsPage
