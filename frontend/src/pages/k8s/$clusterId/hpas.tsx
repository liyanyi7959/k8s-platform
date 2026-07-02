import React, { useState } from 'react'
import {
  ProTable,
  type ProColumns,
  ModalForm,
  ProFormText,
  ProFormDigit,
} from '@ant-design/pro-components'
import { Popconfirm, message, Space, Tag, Tooltip, Drawer, Descriptions, Button } from 'antd'
import {
  DeleteOutlined,
  CodeOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listHPAs, deleteHPA, createHPA, updateHPA } from '@/services/k8s'
import { AppPage, NamespaceSelector } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { HPA } from '@/types'

const HPAsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [detailHPA, setDetailHPA] = useState<HPA | null>(null)
  const [editOpen, setEditOpen] = useState(false)
  const [editingHPA, setEditingHPA] = useState<HPA | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const { data, isLoading } = useQuery({
    queryKey: ['k8s-hpas', clusterId, namespace],
    queryFn: () => listHPAs(clusterId, namespace),
    enabled: !!clusterId,
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

  const columns: ProColumns<HPA>[] = [
    { title: '名称', dataIndex: 'name', ellipsis: true, copyable: true },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 120,
      render: (_, r) => r.namespace || namespace,
    },
    { title: '目标', dataIndex: 'targetName', ellipsis: true, search: false },
    { title: '最小副本', dataIndex: 'minReplicas', width: 100, search: false },
    { title: '最大副本', dataIndex: 'maxReplicas', width: 100, search: false },
    { title: '当前副本', dataIndex: 'currentReplicas', width: 100, search: false },
    {
      title: 'CPU 目标',
      width: 100,
      search: false,
      render: (_, record) => (record.targetCPU ? `${record.targetCPU}%` : '-'),
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
      width: 150,
      fixed: 'right',
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
          <Tooltip title="YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace || namespace)}>
              <CodeOutlined />
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
        toolBarRender={() => [
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => queryClient.invalidateQueries({ queryKey: ['k8s-hpas', clusterId] })}
          >
            刷新
          </Button>,
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            创建 HPA
          </Button>,
        ]}
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
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="名称">{detailHPA.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">
              <Tag>{detailHPA.namespace}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="目标">{detailHPA.targetName || '-'}</Descriptions.Item>
            <Descriptions.Item label="最小副本">{detailHPA.minReplicas}</Descriptions.Item>
            <Descriptions.Item label="最大副本">{detailHPA.maxReplicas}</Descriptions.Item>
            <Descriptions.Item label="当前副本">
              {detailHPA.currentReplicas ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label="CPU 目标">
              {detailHPA.targetCPU ? `${detailHPA.targetCPU}%` : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {formatDate(detailHPA.createdAt)}
            </Descriptions.Item>
          </Descriptions>
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
        <ProFormText
          name="name"
          label="名称"
          rules={[{ required: true, message: '请输入名称' }]}
          placeholder="my-hpa"
        />
        <ProFormText
          name="namespace"
          label="命名空间"
          initialValue="default"
          rules={[{ required: true }]}
        />
        <ProFormText
          name="targetName"
          label="目标 Deployment"
          rules={[{ required: true }]}
          placeholder="my-deployment"
        />
        <ProFormDigit
          name="minReplicas"
          label="最小副本数"
          rules={[{ required: true }]}
          min={0}
          initialValue={1}
        />
        <ProFormDigit
          name="maxReplicas"
          label="最大副本数"
          rules={[{ required: true }]}
          min={1}
          initialValue={10}
        />
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
        <ProFormText name="namespace" label="命名空间" disabled />
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
