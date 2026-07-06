import React, { useState } from 'react'
import {
  ProTable,
  type ProColumns,
  ModalForm,
  ProFormText,
  ProFormDigit,
} from '@ant-design/pro-components'
import { Popconfirm, message, Space, Tag, Tooltip, Drawer, Descriptions, Button, Input, Table, Typography } from 'antd'
import {
  DeleteOutlined,
  ProfileOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { history } from '@umijs/max'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listHPAs, deleteHPA, createHPA, updateHPA } from '@/services/k8s'
import { AppPage, NamespaceSelector, EllipsisText } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { HPA } from '@/types'

const { Text } = Typography

const HPAsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [keyword, setKeyword] = useState('')
  const [detailHPA, setDetailHPA] = useState<HPA | null>(null)
  const [editOpen, setEditOpen] = useState(false)
  const [editingHPA, setEditingHPA] = useState<HPA | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-hpas', clusterId, namespace],
    queryFn: ({ signal }) => listHPAs(clusterId, namespace, signal),
    enabled: !!clusterId,
    refetchInterval: detailHPA || editOpen || createOpen ? false : 60_000,
    staleTime: 60_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: HPA) => deleteHPA(clusterId, record.namespace || namespace, record.name),
    onSuccess: () => {
      message.success('HPA 已删除')
      queryClient.invalidateQueries({ queryKey: ['k8s-hpas', clusterId] })
    },
  })

  const createMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => {
      const ns = (values.namespace as string) || namespace || 'default'
      return createHPA(clusterId, ns, values)
    },
    onSuccess: () => {
      message.success('HPA 创建成功')
      setCreateOpen(false)
      queryClient.invalidateQueries({ queryKey: ['k8s-hpas', clusterId] })
    },
  })

  const updateMutation = useMutation({
    mutationFn: async (values: Record<string, unknown>) => {
      if (!editingHPA) return
      return updateHPA(clusterId, editingHPA.namespace || namespace, editingHPA.name, values)
    },
    onSuccess: () => {
      message.success('HPA 更新成功')
      setEditOpen(false)
      setEditingHPA(null)
      queryClient.invalidateQueries({ queryKey: ['k8s-hpas', clusterId] })
    },
  })

  const filteredData = (data?.items || []).filter((item) => {
    return !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
  })

  const columns: ProColumns<HPA>[] = [
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
      title: '目标',
      dataIndex: 'targetName',
      width: 160,
      ellipsis: true,
      search: false,
      render: (_, r) =>
        r.targetName ? (
          <a onClick={() => r.targetKind === 'Deployment' && history.push(`/k8s/${clusterId}/deployments`)}>
            <Tag color="blue">{r.targetKind || ''}</Tag> {r.targetName}
          </a>
        ) : (
          '-'
        ),
    },
    {
      title: '最小副本',
      dataIndex: 'minReplicas',
      width: 90,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.minReplicas || 0) - (b.minReplicas || 0),
    },
    {
      title: '最大副本',
      dataIndex: 'maxReplicas',
      width: 90,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.maxReplicas || 0) - (b.maxReplicas || 0),
    },
    {
      title: '当前副本',
      dataIndex: 'currentReplicas',
      width: 90,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.currentReplicas || 0) - (b.currentReplicas || 0),
    },
    {
      title: '当前CPU',
      width: 90,
      align: 'center' as const,
      search: false,
      sorter: (a, b) => (a.currentCPU || 0) - (b.currentCPU || 0),
      render: (_, r) =>
        r.currentCPU != null ? (
          <Tag color={r.currentCPU >= (r.targetCPU || 100) ? 'red' : 'green'}>{r.currentCPU}%</Tag>
        ) : (
          '-'
        ),
    },
    {
      title: 'CPU 目标',
      width: 90,
      align: 'center' as const,
      search: false,
      render: (_, r) => (r.targetCPU ? <Tag color="blue">{r.targetCPU}%</Tag> : '-'),
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
      width: 150,
      fixed: 'right',
      align: 'center' as const,
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="详情">
            <a onClick={() => setDetailHPA(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="编辑">
            <a
              onClick={() => {
                setEditingHPA(record)
                setEditOpen(true)
              }}
            >
              <EditOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm title="确定删除该 HPA？" onConfirm={() => deleteMutation.mutate(record)}>
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
      <ProTable<HPA>
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1180 }}
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
            创建 HPA
          </Button>,
        ]}
        headerTitle={<Text strong>HPA 列表</Text>}
      />

      {/* 详情抽屉 */}
      <Drawer
        title={`HPA 详情 - ${detailHPA?.name}`}
        open={!!detailHPA}
        onClose={() => setDetailHPA(null)}
        width={640}
        destroyOnClose
      >
        {detailHPA && (
          <>
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="名称">{detailHPA.name}</Descriptions.Item>
            <Descriptions.Item label="Namespace"><Tag>{detailHPA.namespace}</Tag></Descriptions.Item>
            <Descriptions.Item label="目标" span={2}>{detailHPA.targetName || '-'}</Descriptions.Item>
            <Descriptions.Item label="最小副本">{detailHPA.minReplicas}</Descriptions.Item>
            <Descriptions.Item label="最大副本">{detailHPA.maxReplicas}</Descriptions.Item>
            <Descriptions.Item label="当前副本">{detailHPA.currentReplicas ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="当前CPU">{detailHPA.currentCPU != null ? `${detailHPA.currentCPU}%` : '-'}</Descriptions.Item>
            <Descriptions.Item label="CPU 目标">
              {detailHPA.targetCPU ? <Tag color="blue">{detailHPA.targetCPU}%</Tag> : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间" span={2}>{formatDate(detailHPA.createdAt)}</Descriptions.Item>
          </Descriptions>
          {detailHPA.conditions && detailHPA.conditions.length > 0 && (
            <Table
              style={{ marginTop: 16 }}
              size="small"
              rowKey="type"
              pagination={false}
              dataSource={detailHPA.conditions}
              columns={[
                { title: '条件', dataIndex: 'type', width: 140 },
                { title: '状态', dataIndex: 'status', width: 80, align: 'center' as const, render: (v) => <Tag color={v === 'True' ? 'success' : 'default'}>{v}</Tag> },
                { title: '原因', dataIndex: 'reason', ellipsis: true },
                { title: '消息', dataIndex: 'message', ellipsis: true },
              ]}
            />
          )}
          </>
        )}
      </Drawer>

      {/* 创建 HPA 弹窗 */}
      <ModalForm
        title="创建 HPA"
        open={createOpen}
        onOpenChange={setCreateOpen}
        onFinish={async (values) => {
          createMutation.mutate(values)
          return true
        }}
        width={640}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]} placeholder="my-hpa" />
        <ProFormText name="namespace" label="Namespace" initialValue="default" rules={[{ required: true }]} />
        <ProFormText name="targetName" label="目标 Deployment" rules={[{ required: true }]} placeholder="my-deployment" />
        <ProFormDigit name="minReplicas" label="最小副本数" rules={[{ required: true }]} min={0} initialValue={1} />
        <ProFormDigit name="maxReplicas" label="最大副本数" rules={[{ required: true }]} min={1} initialValue={10} />
        <ProFormDigit name="targetCPU" label="CPU 目标 (%)" min={1} max={100} initialValue={80} />
      </ModalForm>

      {/* 编辑 HPA 弹窗 */}
      <ModalForm
        title={`编辑 HPA - ${editingHPA?.name}`}
        open={editOpen}
        onOpenChange={(open) => {
          setEditOpen(open)
          if (!open) setEditingHPA(null)
        }}
        onFinish={async (values) => {
          updateMutation.mutate(values)
          return true
        }}
        width={640}
        initialValues={
          editingHPA
            ? {
                name: editingHPA.name,
                namespace: editingHPA.namespace,
                targetName: editingHPA.targetName,
                minReplicas: editingHPA.minReplicas,
                maxReplicas: editingHPA.maxReplicas,
                targetCPU: editingHPA.targetCPU,
              }
            : {}
        }
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="名称" disabled />
        <ProFormText name="namespace" label="Namespace" disabled />
        <ProFormText name="targetName" label="目标 Deployment" disabled />
        <ProFormDigit name="minReplicas" label="最小副本数" min={0} />
        <ProFormDigit name="maxReplicas" label="最大副本数" min={1} />
        <ProFormDigit name="targetCPU" label="CPU 目标 (%)" min={1} max={100} />
      </ModalForm>

      <YamlDrawer
        clusterId={clusterId}
        resourceType="hpas"
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />
    </AppPage>
  )
}

export default HPAsPage
