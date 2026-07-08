/**
 * 项目管理页
 * 管理多租户/命名空间分组项目，支持配额配置
 */
import { useMemo, useState } from 'react'
import { history, useModel } from '@umijs/max'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import {
  Button,
  Descriptions,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
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
  RocketOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listClusters } from '@/services/clusters'
import { listHelmReleases } from '@/services/k8s'
import {
  assignNamespaces,
  createProject,
  deleteProject,
  getProjectResources,
  listProjects,
  updateProject,
  type NamespaceResources,
  type Project,
} from '@/services/project'
import { enterClusterWorkspace, formatDate } from '@/utils'
import type { Cluster as ManagedCluster } from '@/types'

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
  const { setCurrentCluster } = useModel('cluster')
  const queryClient = useQueryClient()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Project | null>(null)
  const [form] = Form.useForm()
  // 项目详情 Drawer
  const [detailProject, setDetailProject] = useState<Project | null>(null)

  // 项目列表
  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['projects'],
    queryFn: () => listProjects(),
  })

  // 项目命名空间资源统计（仅当 Drawer 打开时查询）
  const { data: resourcesData, isLoading: resourcesLoading } = useQuery({
    queryKey: ['project-resources', detailProject?.id],
    queryFn: ({ signal }) => getProjectResources(detailProject!.id, signal),
    enabled: !!detailProject,
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

  const clusterMap = useMemo(() => {
    const map = new Map<number, ManagedCluster>()
    ;(clustersData?.items || []).forEach((cluster) => map.set(cluster.id, cluster))
    return map
  }, [clustersData?.items])

  const clusterOptions = (clustersData?.items || []).map((c) => ({
    label: c.name,
    value: c.id,
  }))

  const handleEnterCluster = (cluster: ManagedCluster, targetPath?: string) => {
    enterClusterWorkspace(cluster, {
      setCurrentCluster,
      targetPath,
    })
  }

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

  // 分配命名空间
  const assignNsMutation = useMutation({
    mutationFn: ({ id, namespaces }: { id: number; namespaces: string[] }) =>
      assignNamespaces(id, namespaces),
    onSuccess: () => {
      message.success('命名空间分配成功')
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      queryClient.invalidateQueries({
        queryKey: ['project-resources', detailProject?.id],
      })
    },
    onError: (err: any) => message.error(err?.message || '分配失败'),
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
        <Button
          type="link"
          style={{ padding: 0, height: 'auto' }}
          onClick={() => setDetailProject(record)}
        >
          <Text strong>{record.name || '-'}</Text>
        </Button>
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

      {/* 项目详情 Drawer */}
      <Drawer
        title="项目详情"
        open={!!detailProject}
        onClose={() => setDetailProject(null)}
        width={640}
        destroyOnClose
      >
        {detailProject && (
          <ProjectDetail
            project={detailProject}
            cluster={clusterMap.get(detailProject.cluster_id)}
            clusterName={clusterNameMap.get(detailProject.cluster_id)}
            resourcesData={resourcesData}
            resourcesLoading={resourcesLoading}
            assignPending={assignNsMutation.isPending}
            onEnterCluster={handleEnterCluster}
            onAssign={(namespaces) =>
              assignNsMutation.mutate({
                id: detailProject.id,
                namespaces,
              })
            }
          />
        )}
      </Drawer>
    </AppPage>
  )
}

/** 命名空间资源统计行 */
function ResourceStatRow({
  name,
  stats,
  cluster,
  onEnterCluster,
}: {
  name: string
  stats?: NamespaceResources
  cluster?: ManagedCluster
  onEnterCluster: (cluster: ManagedCluster, targetPath?: string) => void
}) {
  return (
    <Descriptions.Item
      label={
        <a
          onClick={() => {
            if (!cluster) {
              return
            }
            onEnterCluster(cluster, `/k8s/${cluster.id}/pods?namespace=${encodeURIComponent(name)}`)
          }}
        >
          {name}
        </a>
      }
    >
      <Space size={4} wrap>
        <Tag color="blue">Pod: {stats?.pods ?? 0}</Tag>
        <Tag color="green">Deploy: {stats?.deployments ?? 0}</Tag>
        <Tag color="orange">Svc: {stats?.services ?? 0}</Tag>
        <Tag color="purple">CM: {stats?.configmaps ?? 0}</Tag>
      </Space>
    </Descriptions.Item>
  )
}

/** 项目详情内容 */
function ProjectDetail({
  project,
  cluster,
  clusterName,
  resourcesData,
  resourcesLoading,
  assignPending,
  onEnterCluster,
  onAssign,
}: {
  project: Project
  cluster?: ManagedCluster
  clusterName?: string
  resourcesData?: { cluster_id: number; namespaces: Record<string, NamespaceResources> }
  resourcesLoading: boolean
  assignPending: boolean
  onEnterCluster: (cluster: ManagedCluster, targetPath?: string) => void
  onAssign: (namespaces: string[]) => void
}) {
  const [nsForm] = Form.useForm()
  const nsList = splitNamespaces(project.namespaces)

  // 获取该集群的 Helm release 列表
  const { data: helmData, isLoading: helmLoading } = useQuery({
    queryKey: ['project-helm-releases', project.cluster_id],
    queryFn: ({ signal }) => listHelmReleases(Number(project.cluster_id), signal),
    enabled: !!project.cluster_id,
  })

  // 过滤出属于项目命名空间的 release
  const helmReleases = (helmData?.items || []).filter((r: any) =>
    nsList.includes(r.namespace),
  )

  const helmColumns = [
    { title: '名称', dataIndex: 'name', width: 160, ellipsis: true },
    { title: '命名空间', dataIndex: 'namespace', width: 120 },
    { title: 'Chart', dataIndex: 'chart', width: 160, ellipsis: true },
    { title: '版本', dataIndex: 'chart_ver', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => {
        const colorMap: Record<string, string> = {
          deployed: 'success',
          failed: 'error',
          'pending-install': 'processing',
          'pending-upgrade': 'warning',
          'pending-rollback': 'warning',
          uninstalled: 'default',
        }
        return <Tag color={colorMap[status] || 'default'}>{status}</Tag>
      },
    },
    {
      title: '更新时间',
      dataIndex: 'updated',
      width: 170,
      render: (v: string) => (v ? formatDate(v) : '-'),
    },
  ]

  const handleSaveNs = async () => {
    try {
      const values = await nsForm.validateFields()
      onAssign(values.namespaces || [])
    } catch {
      // 校验失败
    }
  }

  return (
    <Spin spinning={resourcesLoading}>
      {/* 部署应用入口 */}
      <Space style={{ marginBottom: 16 }}>
        <Button
          type="primary"
          icon={<RocketOutlined />}
          onClick={() =>
            history.push(
              `/app-store?cluster_id=${project.cluster_id}&namespaces=${project.namespaces}`,
            )
          }
        >
          部署应用
        </Button>
      </Space>

      <Descriptions title="基本信息" column={1} bordered size="small">
        <Descriptions.Item label="项目名称">{project.name}</Descriptions.Item>
        <Descriptions.Item label="描述">
          {project.description || <Text type="secondary">-</Text>}
        </Descriptions.Item>
        <Descriptions.Item label="关联集群">
          {clusterName || (project.cluster_id ? `#${project.cluster_id}` : <Text type="secondary">未绑定</Text>)}
        </Descriptions.Item>
      </Descriptions>

      <Descriptions
        title="命名空间资源统计"
        column={1}
        bordered
        size="small"
        style={{ marginTop: 16 }}
      >
        {nsList.length === 0 ? (
          <Descriptions.Item label="暂无">
            <Text type="secondary">未配置命名空间</Text>
          </Descriptions.Item>
        ) : (
          nsList.map((ns) => (
            <ResourceStatRow
              key={ns}
              name={ns}
              stats={resourcesData?.namespaces?.[ns]}
              cluster={cluster}
              onEnterCluster={onEnterCluster}
            />
          ))
        )}
      </Descriptions>

      {/* Helm Release 列表 */}
      <div style={{ marginTop: 16 }}>
        <Typography.Title level={5}>Helm Release</Typography.Title>
        <Table
          dataSource={helmReleases}
          columns={helmColumns}
          rowKey={(r: any) => `${r.namespace}/${r.name}`}
          loading={helmLoading}
          size="small"
          pagination={{
            pageSize: 5,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
          }}
          scroll={{ x: 800 }}
        />
      </div>

      <Descriptions title="配额信息" column={3} bordered size="small" style={{ marginTop: 16 }}>
        <Descriptions.Item label="CPU">{project.quota_cpu || '-'}</Descriptions.Item>
        <Descriptions.Item label="内存">{project.quota_memory || '-'}</Descriptions.Item>
        <Descriptions.Item label="Pod">{project.quota_pods || '-'}</Descriptions.Item>
      </Descriptions>

      <div style={{ marginTop: 16 }}>
        <Typography.Title level={5}>管理命名空间</Typography.Title>
        <Form form={nsForm} layout="vertical">
          <Form.Item name="namespaces" initialValue={nsList}>
            <Select
              mode="tags"
              placeholder="输入命名空间名称后回车"
              tokenSeparators={[',']}
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Button type="primary" loading={assignPending} onClick={handleSaveNs}>
            保存命名空间
          </Button>
        </Form>
      </div>
    </Spin>
  )
}

export default ProjectListPage
