/**
 * 工作负载管理页
 * 完整功能：Deployment/DaemonSet/StatefulSet 列表+创建+编辑+扩缩容+重启+镜像更新+暂停恢复+版本历史+YAML+删除
 */
import { useState, useRef, useMemo } from 'react'
import {
  ProTable,
  ModalForm,
  ProFormText,
  ProFormDigit,
  ProFormSelect,
  ProFormTextArea,
  type ActionType,
  type ProColumns,
} from '@ant-design/pro-components'
import {
  Button,
  Space,
  message,
  Popconfirm,
  Tag,
  Tooltip,
  Drawer,
  Descriptions,
  Tabs,
  Badge,
  Typography,
  Modal,
  Input,
  Select,
  Table,
} from 'antd'
import {
  PlusOutlined,
  ReloadOutlined,
  DeleteOutlined,
  EditOutlined,
  ProfileOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  HistoryOutlined,
  SyncOutlined,
  ColumnHeightOutlined,
  CloudSyncOutlined,
  CheckOutlined,
  CloseOutlined,
  SearchOutlined,
  RollbackOutlined,
  EyeOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getDeployments,
  createDeployment,
  updateDeployment,
  deleteDeployment,
  scaleDeployment,
  restartDeployment,
  getDeploymentYaml,
  getDeploymentHistory,
  updateWorkloadImage,
  updateWorkloadPaused,
  rollbackDeployment,
  applyYaml,
  listPods,
  getPodEvents,
} from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'
import { AppPage, NamespaceSelector, EllipsisText, YamlEditor } from '@/components'
import { formatDate } from '@/utils'
import { history } from '@umijs/max'

const { Text } = Typography

type WorkloadKind = 'Deployment' | 'DaemonSet' | 'StatefulSet'

type WorkloadsPageProps = {
  fixedKind?: WorkloadKind
}

export default function WorkloadsPage({ fixedKind }: WorkloadsPageProps) {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const actionRef = useRef<ActionType>()
  const [namespace, setNamespace] = useState<string>('')
  const [activeTab, setActiveTab] = useState<WorkloadKind>(fixedKind || 'Deployment')
  const workloadKind = fixedKind || activeTab

  // ═══ Drawers/Modals ═══
  const [searchType, setSearchType] = useState<'name' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [createModal, setCreateModal] = useState(false)
  const [editDrawer, setEditDrawer] = useState<{ open: boolean; record?: any }>({ open: false })
  const [scaleModal, setScaleModal] = useState<{
    open: boolean
    name?: string
    ns?: string
    current?: number
  }>({ open: false })
  const [yamlDrawer, setYamlDrawer] = useState<{
    open: boolean
    name?: string
    yaml?: string
    loading: boolean
    editing: boolean
    yamlOriginal?: string
  }>({ open: false, loading: false, editing: false })
  const [historyDrawer, setHistoryDrawer] = useState<{
    open: boolean
    name?: string
    ns?: string
    history?: any[]
    loading: boolean
  }>({ open: false, loading: false })
  const [imageModal, setImageModal] = useState<{
    open: boolean
    kind?: string
    name?: string
    ns?: string
    containers?: any[]
    newImage: string
  }>({ open: false, newImage: '' })
  const [pauseModal, setPauseModal] = useState<{
    open: boolean
    name?: string
    ns?: string
    paused: boolean
  }>({ open: false, paused: false })
  const [detailDrawer, setDetailDrawer] = useState<{ open: boolean; record?: any }>({ open: false })
  const [detailYamlEditing, setDetailYamlEditing] = useState(false)
  const [detailYamlValue, setDetailYamlValue] = useState('')
  const [selectedKeys, setSelectedKeys] = useState<string[]>([])
  const [batchLoading, setBatchLoading] = useState(false)

  // 任意 Drawer/Modal 打开时暂停轮询
  const anyOverlayOpen =
    createModal ||
    editDrawer.open ||
    scaleModal.open ||
    yamlDrawer.open ||
    historyDrawer.open ||
    imageModal.open ||
    pauseModal.open ||
    detailDrawer.open

  // ═══ Queries ═══
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-workloads', clusterId, namespace, workloadKind],
    queryFn: ({ signal }) => getDeployments(clusterId, namespace || undefined, signal, workloadKind),
    enabled: !!clusterId,
    refetchInterval: anyOverlayOpen ? false : 60000,
    staleTime: 60_000,
  })

  // 详情抽屉：关联 Pod 列表
  const { data: podsData, isLoading: podsLoading } = useQuery({
    queryKey: ['k8s-detail-pods', clusterId, detailDrawer.record?.namespace],
    queryFn: ({ signal }) => listPods(clusterId, { namespace: detailDrawer.record?.namespace }, signal),
    enabled: !!clusterId && detailDrawer.open && !!detailDrawer.record?.namespace,
  })

  // 详情抽屉：关联事件
  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ['k8s-detail-events', clusterId, detailDrawer.record?.namespace, detailDrawer.record?.name],
    queryFn: ({ signal }) =>
      getPodEvents(clusterId, detailDrawer.record?.namespace || '', detailDrawer.record?.name || '', signal),
    enabled: !!clusterId && detailDrawer.open && !!detailDrawer.record?.namespace && !!detailDrawer.record?.name,
  })

  // 详情抽屉：版本历史
  const { data: detailHistoryData, isLoading: detailHistoryLoading } = useQuery({
    queryKey: ['k8s-detail-history', clusterId, detailDrawer.record?.namespace, detailDrawer.record?.name],
    queryFn: ({ signal }) => getDeploymentHistory(clusterId, detailDrawer.record?.namespace || '', detailDrawer.record?.name || ''),
    enabled: !!clusterId && detailDrawer.open && !!detailDrawer.record?.namespace && !!detailDrawer.record?.name,
  })

  // 详情抽屉：YAML
  const { data: detailYamlData, isLoading: detailYamlLoading } = useQuery({
    queryKey: ['k8s-detail-yaml', clusterId, detailDrawer.record?.namespace, detailDrawer.record?.name, workloadKind],
    queryFn: async ({ signal }) => {
      const res = await getDeploymentYaml(clusterId, detailDrawer.record?.namespace || '', detailDrawer.record?.name || '', undefined, workloadKind)
      return res.yaml || ''
    },
    enabled: !!clusterId && detailDrawer.open && !!detailDrawer.record?.namespace && !!detailDrawer.record?.name,
  })

  // 过滤后的数据（按 searchType/searchValue 和 statusFilter 过滤）
  const filteredData = useMemo(() => (data?.items || []).filter((item: any) => {
    let matchSearch = true
    if (searchValue) {
      const v = searchValue.toLowerCase()
      if (searchType === 'name') {
        matchSearch = String(item.name || '').toLowerCase().includes(v)
      } else {
        matchSearch = item.labels && Object.entries(item.labels).some(([k, val]) =>
          `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
      }
    }
    const ready = item.readyReplicas || Number(String(item.ready || '0/0').split('/')[0] || 0)
    const desired = item.replicas || 0
    const isReady = ready === desired
    let matchStatus = true
    if (statusFilter === 'ready') matchStatus = isReady
    else if (statusFilter === 'notReady') matchStatus = !isReady
    else if (statusFilter === 'paused') matchStatus = !!item.paused
    return matchSearch && matchStatus
  }), [data, searchValue, searchType, statusFilter])

  // 详情抽屉：关联 Pod（ownerKind 为 ReplicaSet 且 ownerName 以 deployment 名称开头）
  const relatedPods = useMemo(() => (podsData?.items || []).filter((pod: any) => {
    return (
      pod.ownerKind === 'ReplicaSet' &&
      (pod.ownerName || '').startsWith(detailDrawer.record?.name + '-')
    )
  }), [podsData, detailDrawer.record?.name])

  // ═══ Mutations ═══
  const createMutation = useMutation({
    mutationFn: (values: any) =>
      createDeployment(clusterId, values.namespace || 'default', values, workloadKind),
    onSuccess: () => {
      message.success('创建成功')
      setCreateModal(false)
      queryClient.invalidateQueries({ queryKey: ['k8s-workloads', clusterId] })
    },
    onError: () => message.error('创建失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: ({ name, namespace: recordNamespace }: { name: string; namespace: string }) =>
      deleteDeployment(clusterId, recordNamespace, name, workloadKind),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['k8s-workloads', clusterId] })
    },
    onError: () => message.error('删除失败'),
  })

  const scaleMutation = useMutation({
    mutationFn: ({ name, namespace: recordNamespace, replicas }: { name: string; namespace: string; replicas: number }) =>
      scaleDeployment(clusterId, recordNamespace, name, replicas, workloadKind),
    onSuccess: () => {
      message.success('扩缩容成功')
      setScaleModal({ open: false })
      queryClient.invalidateQueries({ queryKey: ['k8s-workloads', clusterId] })
    },
    onError: () => message.error('扩缩容失败'),
  })

  const restartMutation = useMutation({
    mutationFn: ({ name, namespace: recordNamespace }: { name: string; namespace: string }) =>
      restartDeployment(clusterId, recordNamespace, name, workloadKind),
    onSuccess: () => {
      message.success('已触发重新部署')
      queryClient.invalidateQueries({ queryKey: ['k8s-workloads', clusterId] })
    },
    onError: () => message.error('重新部署失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ name, namespace: recordNamespace, data }: { name: string; namespace: string; data: any }) =>
      updateDeployment(clusterId, recordNamespace, name, { ...data, kind: workloadKind }),
    onSuccess: () => {
      message.success('更新成功')
      setEditDrawer({ open: false })
      queryClient.invalidateQueries({ queryKey: ['k8s-workloads', clusterId] })
    },
    onError: () => message.error('更新失败'),
  })

  const updateImageMutation = useMutation({
    mutationFn: ({ namespace: recordNamespace, name, container, image }: { namespace: string; name: string; container: string; image: string }) =>
      updateWorkloadImage(clusterId, recordNamespace, name, container, image, workloadKind),
    onSuccess: () => {
      message.success('镜像更新成功')
      queryClient.invalidateQueries({ queryKey: ['k8s-workloads', clusterId] })
    },
    onError: () => message.error('镜像更新失败'),
  })

  const pauseMutation = useMutation({
    mutationFn: ({ namespace: recordNamespace, name, paused }: { namespace: string; name: string; paused: boolean }) =>
      updateWorkloadPaused(clusterId, recordNamespace, name, paused),
    onSuccess: () => {
      message.success('更新成功')
      queryClient.invalidateQueries({ queryKey: ['k8s-workloads', clusterId] })
    },
    onError: () => message.error('更新失败'),
  })

  const rollbackMutation = useMutation({
    mutationFn: ({ name, namespace: recordNamespace, revision }: { name: string; namespace: string; revision: number }) =>
      rollbackDeployment(clusterId, recordNamespace, name, revision),
    onSuccess: () => {
      message.success('回滚成功')
      queryClient.invalidateQueries({ queryKey: ['k8s-workloads', clusterId] })
      queryClient.invalidateQueries({ queryKey: ['k8s-detail-history', clusterId] })
      setHistoryDrawer({ open: false, loading: false })
      setDetailDrawer({ open: false })
    },
    onError: () => message.error('回滚失败'),
  })

  const applyYamlMutation = useMutation({
    mutationFn: (yaml: string) => applyYaml(clusterId, yaml),
    onSuccess: (res) => {
      if (res?.success) {
        message.success('YAML 应用成功')
        setYamlDrawer((p) => ({ ...p, editing: false }))
        setDetailYamlEditing(false)
      } else {
        message.error(res?.message || '应用失败')
      }
    },
    onError: () => message.error('YAML 应用失败'),
  })

  // ═══ Handlers ═══
  const handleViewYaml = async (record: any) => {
    setYamlDrawer({ open: true, name: record.name, loading: true, editing: false })
    try {
      const res = await getDeploymentYaml(clusterId, record.namespace, record.name, undefined, workloadKind)
      setYamlDrawer((prev) => ({
        ...prev,
        yaml: res.yaml,
        yamlOriginal: res.yaml,
        loading: false,
      }))
    } catch {
      setYamlDrawer((prev) => ({ ...prev, yaml: '获取 YAML 失败', loading: false }))
    }
  }

  const handleViewHistory = async (record: any) => {
    setHistoryDrawer({ open: true, name: record.name, ns: record.namespace, history: [], loading: true })
    try {
      const res = await getDeploymentHistory(clusterId, record.namespace, record.name)
      setHistoryDrawer((prev) => ({ ...prev, history: res || [], loading: false }))
    } catch {
      setHistoryDrawer((prev) => ({ ...prev, history: [], loading: false }))
    }
  }

  // 批量删除
  const handleBatchDelete = async () => {
    const rows = filteredData.filter((r: any) => selectedKeys.includes(r.name))
    setBatchLoading(true)
    try {
      await Promise.all(
        rows.map((r: any) => deleteMutation.mutateAsync({ name: r.name, namespace: r.namespace })),
      )
    } finally {
      setBatchLoading(false)
      setSelectedKeys([])
    }
  }

  // 批量重启
  const handleBatchRestart = async () => {
    const rows = filteredData.filter((r: any) => selectedKeys.includes(r.name))
    setBatchLoading(true)
    try {
      await Promise.all(
        rows.map((r: any) => restartMutation.mutateAsync({ name: r.name, namespace: r.namespace })),
      )
    } finally {
      setBatchLoading(false)
      setSelectedKeys([])
    }
  }

  // ═══ Columns ═══
  const columns: ProColumns[] = [
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
      render: (_, r) => <EllipsisText text={r.namespace} tag />,
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
      render: (_, r) => (
        <a onClick={() => setDetailDrawer({ open: true, record: r })}>
          <Text strong>{r.name}</Text>
        </a>
      ),
    },
    {
      title: 'READY',
      width: 120,
      align: 'center' as const,
      sorter: (a, b) => (a.readyReplicas || 0) - (b.readyReplicas || 0),
      render: (_, r) => {
        const ready = r.readyReplicas || Number(String(r.ready || '0/0').split('/')[0] || 0)
        const desired = r.replicas || 0
        const ok = ready === desired
        return <Badge status={ok ? 'success' : 'processing'} text={`${ready}/${desired}`} />
      },
    },
    {
      title: 'UP-TO-DATE',
      dataIndex: 'upToDate',
      width: 110,
      align: 'center' as const,
      render: (_, r) => <Text>{r.upToDate ?? 0}</Text>,
    },
    {
      title: 'AVAILABLE',
      dataIndex: 'available',
      width: 110,
      align: 'center' as const,
      render: (_, r) => <Text>{r.available ?? 0}</Text>,
    },
    {
      title: '策略',
      width: 120,
      render: (_, r) => <Tag>{r.strategy || 'RollingUpdate'}</Tag>,
    },
    {
      title: '镜像',
      width: 200,
      ellipsis: true,
      render: (_, r) => {
        const images = r.images || []
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
      title: '暂停',
      width: 70,
      align: 'center' as const,
      render: (_, r) =>
        r.paused ? <Tag color="warning">是</Tag> : <Text type="secondary">否</Text>,
    },
    {
      title: 'Age',
      dataIndex: 'age',
      width: 110,
      align: 'center' as const,
      ellipsis: true,
      sorter: (a, b) => new Date(a.age || 0).getTime() - new Date(b.age || 0).getTime(),
      render: (_, r) => (r.age ? formatDate(r.age) : '-'),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 250,
      fixed: 'right',
      render: (_, record) => (
        <Space size={8} style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Tooltip title="详情">
            <a onClick={() => setDetailDrawer({ open: true, record })}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="编辑">
            <a onClick={() => setEditDrawer({ open: true, record })}>
              <EditOutlined />
            </a>
          </Tooltip>
          <Tooltip title="扩缩容">
            <a
              onClick={() =>
                setScaleModal({
                  open: true,
                  name: record.name,
                  ns: record.namespace,
                  current: record.replicas,
                })
              }
            >
              <ColumnHeightOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定重新部署？"
            onConfirm={() => restartMutation.mutate({ name: record.name, namespace: record.namespace })}
          >
            <Tooltip title="重部署">
              <a>
                <SyncOutlined />
              </a>
            </Tooltip>
          </Popconfirm>
          <Tooltip title="镜像">
            <a
              onClick={() =>
                setImageModal({
                  open: true,
                  kind: workloadKind,
                  name: record.name,
                  ns: record.namespace,
                  containers: (record.images || []).map((image: string, index: number) => ({
                    name: `${record.name}-${index + 1}`,
                    image,
                  })),
                  newImage: '',
                })
              }
            >
              <CloudSyncOutlined />
            </a>
          </Tooltip>
          <Tooltip title={workloadKind === 'Deployment' ? (record.paused ? '恢复' : '暂停') : '仅 Deployment 支持暂停'}>
            <a
              style={{ color: workloadKind === 'Deployment' ? undefined : '#bfbfbf', pointerEvents: workloadKind === 'Deployment' ? undefined : 'none' }}
              onClick={() =>
                setPauseModal({
                  open: true,
                  name: record.name,
                  ns: record.namespace,
                  paused: !record.paused,
                })
              }
            >
              {record.paused ? <PlayCircleOutlined /> : <PauseCircleOutlined />}
            </a>
          </Tooltip>
          <Tooltip title="历史">
            <a
              style={{ color: workloadKind === 'Deployment' ? undefined : '#bfbfbf', pointerEvents: workloadKind === 'Deployment' ? undefined : 'none' }}
              onClick={() => handleViewHistory(record)}
            >
              <HistoryOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => handleViewYaml(record)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除？"
            onConfirm={() => deleteMutation.mutate({ name: record.name, namespace: record.namespace })}
          >
            <Tooltip title="删除">
              <a style={{ color: '#dc2626' }}>
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
      {!fixedKind && (
        <Tabs
          activeKey={activeTab}
          onChange={(k) => setActiveTab(k as WorkloadKind)}
          items={[
            { key: 'Deployment', label: 'Deployment' },
            { key: 'DaemonSet', label: 'DaemonSet' },
            { key: 'StatefulSet', label: 'StatefulSet' },
          ]}
          style={{ marginBottom: 0 }}
        />
      )}

      <ProTable
        headerTitle={`${workloadKind} 列表`}
        actionRef={actionRef}
        rowKey="name"
        rowSelection={{
          selectedRowKeys: selectedKeys,
          onChange: (keys) => setSelectedKeys(keys as string[]),
        }}
        tableAlertRender={() => (
          <Space size={12}>
            <Text type="secondary">已选择 {selectedKeys.length} 项</Text>
            <Popconfirm title="确定批量删除选中的工作负载？" onConfirm={handleBatchDelete}>
              <Button danger icon={<DeleteOutlined />} loading={batchLoading} size="small">
                批量删除
              </Button>
            </Popconfirm>
            <Popconfirm title="确定批量重启选中的工作负载？" onConfirm={handleBatchRestart}>
              <Button icon={<SyncOutlined />} loading={batchLoading} size="small">
                批量重启
              </Button>
            </Popconfirm>
          </Space>
        )}
        search={false}
        options={{ reload: false }}
        loading={isLoading}
        dataSource={filteredData}
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : '按标签搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'label', label: '标签' }]} />}
            prefix={<SearchOutlined />}
          />,
          <Select
            key="status"
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ width: 120 }}
            options={[
              { value: 'all', label: '全部' },
              { value: 'ready', label: 'Ready' },
              { value: 'notReady', label: '未就绪' },
              { value: 'paused', label: '已暂停' },
            ]}
          />,
          <NamespaceSelector
            key="ns"
            clusterId={clusterId}
            value={namespace}
            onChange={(v) => {
              setNamespace(v)
            }}
          />,
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateModal(true)}
          >
            创建 {workloadKind}
          </Button>,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>,
        ]}
        columns={columns}
        scroll={{ x: 1400 }}
      />

      {/* ═══ 创建 ModalForm ═══ */}
      <ModalForm
        title={`创建 ${workloadKind}`}
        open={createModal}
        onOpenChange={setCreateModal}
        onFinish={async (values) => {
          createMutation.mutate(values)
          return true
        }}
        width={600}
      >
        <ProFormText name="name" label="名称" rules={[{ required: true }]} />
        <ProFormText name="namespace" label="命名空间" initialValue="default" />
        <ProFormDigit
          name="replicas"
          label="副本数"
          initialValue={3}
          min={0}
          max={100}
          rules={[{ required: true }]}
          hidden={workloadKind === 'DaemonSet'}
        />
        <ProFormText
          name="image"
          label="镜像"
          rules={[{ required: true }]}
          placeholder="nginx:1.24"
        />
        <ProFormText name="port" label="容器端口" placeholder="80" />
        <ProFormSelect
          name="strategy"
          label="部署策略"
          initialValue="RollingUpdate"
          options={[
            { value: 'RollingUpdate', label: 'RollingUpdate' },
            { value: 'Recreate', label: 'Recreate' },
          ]}
        />
        <ProFormTextArea name="labels" label="标签 (JSON)" placeholder='{"app":"nginx"}' />
      </ModalForm>

      {/* ═══ 编辑 ModalForm ═══ */}
      <ModalForm
        title={`编辑 ${workloadKind} - ${editDrawer.record?.name}`}
        open={editDrawer.open}
        onOpenChange={(v) => !v && setEditDrawer({ open: false })}
        onFinish={async (values) => {
          if (!editDrawer.record) return false
          updateMutation.mutate({
            name: editDrawer.record.name,
            namespace: editDrawer.record.namespace,
            data: values,
          })
          return true
        }}
        initialValues={
          editDrawer.record
            ? {
                replicas: editDrawer.record.replicas,
                image: editDrawer.record.images?.[0],
                strategy: editDrawer.record.strategy,
              }
            : undefined
        }
        width={500}
      >
        <ProFormText name="name" label="名称" disabled initialValue={editDrawer.record?.name} />
        <ProFormText name="namespace" label="命名空间" disabled initialValue={editDrawer.record?.namespace} />
        <ProFormDigit name="replicas" label="副本数" min={0} max={100} />
        <ProFormText name="image" label="镜像" />
        <ProFormSelect
          name="strategy"
          label="部署策略"
          options={[
            { value: 'RollingUpdate', label: 'RollingUpdate' },
            { value: 'Recreate', label: 'Recreate' },
          ]}
        />
      </ModalForm>

      {/* ═══ 扩缩容 Modal ═══ */}
      <Modal
        title={`扩缩容 - ${scaleModal.name}`}
        open={scaleModal.open}
        onCancel={() => setScaleModal({ open: false })}
        onOk={() =>
          scaleMutation.mutate({ name: scaleModal.name!, namespace: scaleModal.ns!, replicas: scaleModal.current || 0 })
        }
        confirmLoading={scaleMutation.isPending}
      >
        <Descriptions column={1} size="small">
          <Descriptions.Item label="当前副本数">{scaleModal.current}</Descriptions.Item>
          <Descriptions.Item label="目标副本数">
            <Input
              type="number"
              min={0}
              max={100}
              value={scaleModal.current}
              onChange={(e) => setScaleModal((s) => ({ ...s, current: Number(e.target.value) }))}
            />
          </Descriptions.Item>
        </Descriptions>
      </Modal>

      {/* ═══ 镜像更新 Modal ═══ */}
      <Modal
        title={`更新镜像 - ${imageModal.name}`}
        open={imageModal.open}
        onCancel={() => setImageModal({ open: false, newImage: '' })}
        onOk={() => {
          const containerName = imageModal.containers?.[0]?.name
          const newImage = imageModal.newImage
          if (containerName && newImage) {
            updateImageMutation.mutate({
              name: imageModal.name!,
              namespace: imageModal.ns!,
              container: containerName,
              image: newImage,
            })
            setImageModal({ open: false, newImage: '' })
          }
        }}
      >
        <Descriptions column={1} size="small">
          <Descriptions.Item label="当前镜像">
            {imageModal.containers?.map((c: any) => (
              <Tag key={c.name}>
                {c.name}: {c.image}
              </Tag>
            ))}
          </Descriptions.Item>
          <Descriptions.Item label="新镜像">
            <Input
              placeholder="nginx:1.25"
              value={imageModal.newImage || ''}
              onChange={(e) => setImageModal((s) => ({ ...s, newImage: e.target.value }))}
            />
          </Descriptions.Item>
        </Descriptions>
      </Modal>

      {/* ═══ 暂停/恢复确认 ═══ */}
      <Modal
        title={pauseModal.paused ? `恢复 ${pauseModal.name}` : `暂停 ${pauseModal.name}`}
        open={pauseModal.open}
        onCancel={() => setPauseModal({ open: false, paused: false })}
        onOk={() => {
          pauseMutation.mutate({ name: pauseModal.name!, namespace: pauseModal.ns!, paused: pauseModal.paused })
          setPauseModal({ open: false, paused: false })
        }}
      >
        <Text>
          {pauseModal.paused
            ? '恢复后将自动触发新的部署流程。'
            : '暂停后将停止滚动更新，当前运行的 Pod 不受影响。'}
        </Text>
      </Modal>

      {/* ═══ YAML Drawer ═══ */}
      <Drawer
        title={`YAML - ${yamlDrawer.name}`}
        open={yamlDrawer.open}
        onClose={() => setYamlDrawer({ open: false, loading: false, editing: false })}
        width={800}
        extra={
          yamlDrawer.editing ? (
            <Space>
              <Button
                icon={<CheckOutlined />}
                type="primary"
                loading={applyYamlMutation.isPending}
                onClick={() => applyYamlMutation.mutate(yamlDrawer.yaml || '')}
              >
                应用
              </Button>
              <Button
                icon={<CloseOutlined />}
                onClick={() =>
                  setYamlDrawer((p) => ({ ...p, yaml: p.yamlOriginal, editing: false }))
                }
              >
                取消
              </Button>
            </Space>
          ) : (
            <Button
              icon={<EditOutlined />}
              onClick={() => setYamlDrawer((p) => ({ ...p, editing: true }))}
            >
              编辑
            </Button>
          )
        }
      >
        {yamlDrawer.loading ? (
          <Text type="secondary">加载中...</Text>
        ) : (
          <YamlEditor
            value={yamlDrawer.yaml || ''}
            readOnly={!yamlDrawer.editing}
            height={Math.max(window.innerHeight - 220, 400)}
            onChange={(val) => setYamlDrawer((p) => ({ ...p, yaml: val }))}
          />
        )}
      </Drawer>

      {/* ═══ 版本历史 Drawer ═══ */}
      <Drawer
        title={`版本历史 - ${historyDrawer.name}`}
        open={historyDrawer.open}
        onClose={() => setHistoryDrawer({ open: false, loading: false })}
        width={700}
      >
        {historyDrawer.loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ background: '#fafafa' }}>
                {['版本', '变更原因', '镜像', '时间', '状态', '操作'].map((h) => (
                  <th
                    key={h}
                    style={{
                      padding: '8px 12px',
                      borderBottom: '1px solid #f0f0f0',
                      textAlign: 'left',
                      fontSize: 12,
                    }}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {(historyDrawer.history || []).map((rev: any, i: number) => (
                <tr key={i}>
                  <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                    <Tag>v{rev.revision || i + 1}</Tag>
                  </td>
                  <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                    {rev.changeCause || '-'}
                  </td>
                  <td
                    style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', fontSize: 11 }}
                  >
                    {rev.images?.join(', ') || rev.image || '-'}
                  </td>
                  <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                    {rev.created_at || rev.createdAt || rev.date || '-'}
                  </td>
                  <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                    <Tag color={rev.is_current ? 'success' : 'processing'}>
                      {rev.is_current ? '当前版本' : '历史版本'}
                    </Tag>
                  </td>
                  <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                    {rev.is_current ? (
                      '-'
                    ) : (
                      <Popconfirm
                        title="确定回滚到此版本？"
                        onConfirm={() =>
                          rollbackMutation.mutate({
                            name: historyDrawer.name!,
                            namespace: historyDrawer.ns!,
                            revision: rev.revision,
                          })
                        }
                      >
                        <Button type="link" size="small" icon={<RollbackOutlined />}>
                          回滚到此版本
                        </Button>
                      </Popconfirm>
                    )}
                  </td>
                </tr>
              ))}
              {(!historyDrawer.history || historyDrawer.history.length === 0) && (
                <tr>
                  <td colSpan={6} style={{ padding: 20, textAlign: 'center' }}>
                    <Text type="secondary">暂无版本历史</Text>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </Drawer>

      {/* ═══ 详情 Drawer ═══ */}
      <Drawer
        title={`详情 - ${detailDrawer.record?.name}`}
        open={detailDrawer.open}
        onClose={() => { setDetailDrawer({ open: false }); setDetailYamlEditing(false) }}
        width={720}
      >
        {detailDrawer.record && (
          <Tabs
            defaultActiveKey="overview"
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <>
                    <div style={{ marginBottom: 16 }}>
                      <Space size={4} wrap>
                        <Text strong>关联 Pod：</Text>
                        <Text>共 {relatedPods.length} 个</Text>
                        <Text type="secondary">|</Text>
                        <Tag color="success">Running {relatedPods.filter((p: any) => p.status === 'Running').length}</Tag>
                        <Tag color="processing">Pending {relatedPods.filter((p: any) => p.status === 'Pending').length}</Tag>
                        <Tag color="error">Failed {relatedPods.filter((p: any) => p.status === 'Failed').length}</Tag>
                        <Tag>Succeeded {relatedPods.filter((p: any) => p.status === 'Succeeded').length}</Tag>
                      </Space>
                    </div>
                    <Descriptions column={2} size="small" bordered>
                      <Descriptions.Item label="名称">{detailDrawer.record.name}</Descriptions.Item>
                      <Descriptions.Item label="命名空间">{detailDrawer.record.namespace}</Descriptions.Item>
                      <Descriptions.Item label="副本数">
                        {detailDrawer.record.readyReplicas ?? detailDrawer.record.replicas ?? 0}/
                        {detailDrawer.record.replicas ?? 0}
                      </Descriptions.Item>
                      <Descriptions.Item label="策略">{detailDrawer.record.strategy || 'RollingUpdate'}</Descriptions.Item>
                      <Descriptions.Item label="镜像" span={2}>
                        {(detailDrawer.record.images || []).map((img: string, i: number) => (
                          <Tag key={i}>{img}</Tag>
                        ))}
                      </Descriptions.Item>
                      <Descriptions.Item label="暂停状态">
                        {detailDrawer.record.paused ? <Tag color="warning">已暂停</Tag> : <Text type="secondary">否</Text>}
                      </Descriptions.Item>
                      <Descriptions.Item label="创建时间">
                        {detailDrawer.record.age ? formatDate(detailDrawer.record.age) : '-'}
                      </Descriptions.Item>
                    </Descriptions>
                    <Table
                      style={{ marginTop: 16 }}
                      size="small"
                      rowKey={(_, i) => String(i)}
                      pagination={false}
                      dataSource={(detailDrawer.record.images || []).map((image: string, index: number) => ({
                        key: index,
                        name: `${detailDrawer.record.name}-${index + 1}`,
                        image,
                      }))}
                      columns={[
                        { title: '容器名称', dataIndex: 'name' },
                        { title: '镜像', dataIndex: 'image' },
                      ]}
                    />
                  </>
                ),
              },
              {
                key: 'pods',
                label: '关联 Pod',
                children: (
                  <>
                    <div style={{ marginBottom: 16 }}>
                      <Space size={4} wrap>
                        <Text strong>共 {relatedPods.length} 个 Pod</Text>
                        <Text type="secondary">|</Text>
                        <Tag color="success">Running {relatedPods.filter((p: any) => p.status === 'Running').length}</Tag>
                        <Tag color="processing">Pending {relatedPods.filter((p: any) => p.status === 'Pending').length}</Tag>
                        <Tag color="error">Failed {relatedPods.filter((p: any) => p.status === 'Failed').length}</Tag>
                      </Space>
                    </div>
                    <Table
                      size="small"
                      rowKey="name"
                      loading={podsLoading}
                      pagination={false}
                      dataSource={relatedPods}
                      columns={[
                        {
                          title: '名称',
                          dataIndex: 'name',
                          ellipsis: true,
                          render: (_, r) => (
                            <a onClick={() => history.push(`/k8s/${clusterId}/pods`)}>
                              <Text strong>{r.name}</Text>
                            </a>
                          ),
                        },
                        {
                          title: '状态',
                          dataIndex: 'status',
                          width: 100,
                          render: (_, r) => {
                            const colorMap: Record<string, string> = {
                              Running: 'success',
                              Pending: 'processing',
                              Failed: 'error',
                              Succeeded: 'default',
                            }
                            return <Tag color={colorMap[r.status] || 'default'}>{r.status}</Tag>
                          },
                        },
                        { title: '节点', dataIndex: 'nodeName', ellipsis: true },
                        { title: '重启次数', dataIndex: 'restarts', width: 90, align: 'center' as const },
                        {
                          title: 'Age',
                          dataIndex: 'createdAt',
                          width: 110,
                          render: (_, r) => (r.createdAt ? formatDate(r.createdAt) : '-'),
                        },
                      ]}
                    />
                  </>
                ),
              },
              {
                key: 'events',
                label: '事件',
                children: (
                  <Table
                    size="small"
                    rowKey={(_, i) => String(i)}
                    loading={eventsLoading}
                    pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                    dataSource={eventsData || []}
                    columns={[
                      {
                        title: '类型',
                        dataIndex: 'type',
                        width: 90,
                        render: (_, r) => (
                          <Tag color={r.type === 'Warning' ? 'warning' : 'success'}>{r.type || 'Normal'}</Tag>
                        ),
                      },
                      { title: '原因', dataIndex: 'reason', width: 140, ellipsis: true },
                      { title: '消息', dataIndex: 'message', ellipsis: true },
                      {
                        title: '时间',
                        dataIndex: 'lastTimestamp',
                        width: 150,
                        render: (_, r) =>
                          formatDate(r.lastTimestamp || r.firstTimestamp || r.metadata?.creationTimestamp || ''),
                      },
                    ]}
                  />
                ),
              },
              ...(workloadKind === 'Deployment'
                ? [
                    {
                      key: 'history',
                      label: '版本历史',
                      children: (
                        <Table
                          size="small"
                          rowKey={(_, i) => String(i)}
                          loading={detailHistoryLoading}
                          pagination={false}
                          dataSource={detailHistoryData || []}
                          columns={[
                            { title: '版本', dataIndex: 'revision', width: 70, render: (v, r: any, i: number) => <Tag>v{v || i + 1}</Tag> },
                            { title: '变更原因', dataIndex: 'changeCause', ellipsis: true, render: (v) => v || '-' },
                            { title: '镜像', dataIndex: 'images', ellipsis: true, render: (v, r: any) => r.images?.join(', ') || r.image || '-' },
                            { title: '时间', dataIndex: 'created_at', width: 150, render: (v, r: any) => v || r.createdAt || r.date || '-' },
                            { title: '状态', width: 90, render: (_, r: any) => <Tag color={r.is_current ? 'success' : 'processing'}>{r.is_current ? '当前' : '历史'}</Tag> },
                            { title: '操作', width: 120, render: (_, r: any) => r.is_current ? '-' : (
                              <Popconfirm title="确定回滚到此版本？" onConfirm={() => rollbackMutation.mutate({ name: detailDrawer.record.name, namespace: detailDrawer.record.namespace, revision: r.revision })}>
                                <Button type="link" size="small" icon={<RollbackOutlined />}>回滚</Button>
                              </Popconfirm>
                            )},
                          ]}
                        />
                      ),
                    },
                  ]
                : []),
              {
                key: 'yaml',
                label: 'YAML',
                children: (
                  <>
                    <div style={{ marginBottom: 12 }}>
                      <Space>
                        {detailYamlEditing ? (
                          <>
                            <Button
                              icon={<CheckOutlined />}
                              type="primary"
                              loading={applyYamlMutation.isPending}
                              onClick={() => applyYamlMutation.mutate(detailYamlValue)}
                            >
                              应用
                            </Button>
                            <Button
                              icon={<CloseOutlined />}
                              onClick={() => {
                                setDetailYamlEditing(false)
                                setDetailYamlValue('')
                              }}
                            >
                              取消
                            </Button>
                          </>
                        ) : (
                          <Button
                            icon={<EditOutlined />}
                            onClick={() => {
                              setDetailYamlEditing(true)
                              setDetailYamlValue(detailYamlData || '')
                            }}
                          >
                            编辑
                          </Button>
                        )}
                      </Space>
                    </div>
                    {detailYamlLoading ? (
                      <Text type="secondary">加载中...</Text>
                    ) : (
                      <YamlEditor
                        value={detailYamlEditing ? detailYamlValue : (detailYamlData || '')}
                        readOnly={!detailYamlEditing}
                        height={Math.max(window.innerHeight - 280, 400)}
                        onChange={(val) => setDetailYamlValue(val)}
                      />
                    )}
                  </>
                ),
              },
            ]}
          />
        )}
      </Drawer>
    </AppPage>
  )
}
