import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tag, Popconfirm, message, Space, Tooltip, Drawer, Descriptions } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listPDBs, deletePDB } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { PDB } from '@/types'

const PDBsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailPDB, setDetailPDB] = useState<PDB | null>(null)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-pdbs', clusterId, namespace],
    queryFn: () => listPDBs(clusterId, namespace),
    enabled: !!clusterId,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: PDB) => deletePDB(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('PDB 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-pdbs', clusterId] })
    },
  })

  const columns: ProColumns<PDB>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace || namespace} tag />,
    },
    {
      title: '最小可用',
      dataIndex: 'minAvailable',
      width: 100,
      search: false,
      render: (_, record) => <Tag color="green">{record.minAvailable}</Tag>,
    },
    {
      title: '最大不可用',
      dataIndex: 'maxUnavailable',
      width: 110,
      search: false,
      render: (_, record) => <Tag color="orange">{record.maxUnavailable}</Tag>,
    },
    { title: '当前健康', dataIndex: 'currentHealthy', width: 100, search: false },
    { title: '期望健康', dataIndex: 'desiredHealthy', width: 100, search: false },
    { title: '允许中断', dataIndex: 'allowedDisruptions', width: 100, search: false },
    {
      title: '状态',
      width: 100,
      search: false,
      render: (_, record) => (
        <Tag color={record.currentHealthy >= record.desiredHealthy ? 'success' : 'error'}>
          {record.currentHealthy >= record.desiredHealthy ? '健康' : '异常'}
        </Tag>
      ),
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
      <ProTable<PDB>
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1000 }}
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
        title={`PDB 详情 - ${detailPDB?.name}`}
        open={!!detailPDB}
        onClose={() => setDetailPDB(null)}
        width={640}
        destroyOnClose
      >
        {detailPDB && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailPDB.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              <Tag>{detailPDB.namespace}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="最小可用">
              <Tag color="green">{detailPDB.minAvailable}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="最大不可用">
              <Tag color="orange">{detailPDB.maxUnavailable}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="当前健康">{detailPDB.currentHealthy}</Descriptions.Item>
            <Descriptions.Item label="期望健康">{detailPDB.desiredHealthy}</Descriptions.Item>
            <Descriptions.Item label="允许中断">{detailPDB.allowedDisruptions}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDate(detailPDB.createdAt)}
            </Descriptions.Item>
          </Descriptions>
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
    </AppPage>
  )
}

export default PDBsPage
