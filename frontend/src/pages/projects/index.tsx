/**
 * 项目管理页
 * 管理多租户/命名空间分组项目，支持配额配置
 */
import { useMemo, useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import {
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'
import {
  createProject,
  deleteProject,
  listProjects,
  updateProject,
  type Project,
} from '@/services/project'
import { formatDate } from '@/utils'

const { Text } = Typography

/** 将逗号分隔的命名空间字符串拆分为数组 */
function splitNamespaces(ns: string): string[] {
  return ns
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 项目管理页 */
const ProjectListPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Project | null>(null)
  const [form] = Form.useForm()

  // 项目列表
  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['projects'],
    queryFn: () => listProjects(),
  })

  // 集群列表（用于表单中的集群选择）
  const { data: clustersData } = useQuery({
    queryKey: ['clusters-for-project'],
    queryFn: () => listClusters(),
  })

  // 集群 ID → 名称映射
  const clusterNameMap = useMemo(() => {
    const map = new Map<number, string>()
    ;(clustersData?.items || []).forEach((c) => map.set(c.id, c.name))
    return map
  }, [clustersData?.items])

  const clusterOptions = (clustersData?.items || []).map((c) => ({
    label: c.name,
    value: c.id,
  }))

  const createMutation = useMutation({
    mutationFn: (payload: Partial<Project>) => createProject(payload),
    onSuccess: () => {
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
    onError: (err: any) => message.error(err?.message || '创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<Project> }) =>
      updateProject(id, data),
    onSuccess: () => {
      message.success('更新成功')
      setModalOpen(false)
      setEditing(null)
      form.resetFields()
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
    onError: (err: any) => message.error(err?.message || '更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteProject(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
    onError: (err: any) => message.error(err?.message || '删除失败'),
  })

  const handleAdd = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: Project) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      cluster_id: record.cluster_id || undefined,
      namespaces: splitNamespaces(record.namespaces),
      quota_cpu: record.quota_cpu,
      quota_memory: record.quota_memory,
      quota_pods: record.quota_pods,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      // 命名空间数组转逗号分隔字符串
      const payload: Partial<Project> = {
        name: values.name,
        description: values.description || '',
        cluster_id: values.cluster_id || 0,
        namespaces: (values.namespaces || []).join(','),
        quota_cpu: values.quota_cpu || '',
        quota_memory: values.quota_memory || '',
        quota_pods: values.quota_pods || '',
      }
      if (editing) {
        updateMutation.mutate({ id: editing.id, data: payload })
      } else {
        createMutation.mutate(payload)
      }
    } catch {
      // 校验失败
    }
  }

  const columns: ProColumns<Project>[] = [
    {
      title: '项目名称',
      dataIndex: 'name',
      width: 160,
      render: (_: unknown, record: Project) => (
        <Text strong>{record.name || '-'}</Text>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      width: 200,
      ellipsis: true,
      render: (desc: string) => desc || <Text type="secondary">-</Text>,
    },
    {
      title: '关联集群',
      dataIndex: 'cluster_id',
      width: 140,
      render: (clusterId: number) =>
        clusterNameMap.get(clusterId) || (
          <Text type="secondary">{clusterId ? `#${clusterId}` : '-'}</Text>
        ),
    },
    {
      title: '命名空间',
      dataIndex: 'namespaces',
      width: 220,
      render: (ns: string) => {
        const list = splitNamespaces(ns)
        if (list.length === 0) return <Text type="secondary">-</Text>
        return (
          <Space size={[4, 4]} wrap>
            {list.map((n) => (
              <Tag key={n} color="blue">
                {n}
              </Tag>
            ))}
          </Space>
        )
      },
    },
    {
      title: 'CPU 配额',
      dataIndex: 'quota_cpu',
      width: 100,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '内存配额',
      dataIndex: 'quota_memory',
      width: 110,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: 'Pod 配额',
      dataIndex: 'quota_pods',
      width: 100,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) =>
        v ? formatDate(v, 'YYYY-MM-DD HH:mm') : <Text type="secondary">-</Text>,
    },
    {
      title: '操作',
      width: 140,
      align: 'center',
      fixed: 'right',
      render: (_: unknown, record: Project) => (
        <Space>
          <Tooltip title="编辑">
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确认删除该项目？"
            onConfirm={() => deleteMutation.mutate(record.id)}
          >
            <Tooltip title="删除">
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <ProTable<Project>
        headerTitle="项目列表"
        columns={columns}
        dataSource={data?.list || []}
        loading={isLoading || isFetching}
        rowKey="id"
        search={false}
        options={false}
        cardBordered={false}
        tableAlertRender={false}
        scroll={{ x: 1200 }}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        toolBarRender={() => [
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => queryClient.invalidateQueries({ queryKey: ['projects'] })}
          >
            刷新
          </Button>,
          <Button key="add" type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新建项目
          </Button>,
        ]}
      />

      <Modal
        title={editing ? '编辑项目' : '新建项目'}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditing(null)
          form.resetFields()
        }}
        onOk={handleSubmit}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="项目名称"
            rules={[{ required: true, message: '请输入项目名称' }]}
          >
            <Input placeholder="例如：team-platform" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="可选，项目用途说明" />
          </Form.Item>
          <Form.Item name="cluster_id" label="关联集群">
            <Select
              placeholder="选择关联集群（可选）"
              options={clusterOptions}
              allowClear
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item name="namespaces" label="命名空间">
            <Select
              mode="tags"
              placeholder="输入命名空间名称后回车，可添加多个"
              tokenSeparators={[',']}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="quota_cpu" label="CPU 配额" style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="例如：10" />
            </Form.Item>
            <Form.Item
              name="quota_memory"
              label="内存配额"
              style={{ flex: 1, marginBottom: 0 }}
            >
              <Input placeholder="例如：16Gi" />
            </Form.Item>
            <Form.Item name="quota_pods" label="Pod 配额" style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="例如：100" />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </AppPage>
  )
}

export default ProjectListPage
