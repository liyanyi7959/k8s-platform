/**
 * 工作负载管理页
 * 完整功能：Deployment/DaemonSet/StatefulSet 列表+创建+编辑+扩缩容+重启+镜像更新+暂停恢复+版本历史+YAML+删除
 */
import { useState, useRef } from 'react'
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
} from 'antd'
import {
  PlusOutlined,
  ReloadOutlined,
  DeleteOutlined,
  EditOutlined,
  CodeOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  HistoryOutlined,
  SyncOutlined,
  ColumnHeightOutlined,
  CloudSyncOutlined,
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
} from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'
import { AppPage, NamespaceSelector } from '@/components'

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
  }>({ open: false, loading: false })
  const [historyDrawer, setHistoryDrawer] = useState<{
    open: boolean
    name?: string
    history?: any[]
    loading: boolean
  }>({ open: false, loading: false })
  const [imageModal, setImageModal] = useState<{
    open: boolean
    kind?: string
    name?: string
    ns?: string
    containers?: any[]
  }>({ open: false })
  const [pauseModal, setPauseModal] = useState<{
    open: boolean
    name?: string
    ns?: string
    paused: boolean
  }>({ open: false, paused: false })

  // ═══ Queries ═══
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['k8s-workloads', clusterId, namespace, workloadKind],
    queryFn: ({ signal }) => getDeployments(clusterId, namespace || undefined, signal, workloadKind),
    enabled: !!clusterId,
  })

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

  // ═══ Handlers ═══
  const handleViewYaml = async (record: any) => {
    setYamlDrawer({ open: true, name: record.name, loading: true })
    try {
      const res = await getDeploymentYaml(clusterId, record.namespace, record.name, undefined, workloadKind)
      setYamlDrawer((prev) => ({
        ...prev,
        yaml: res.yaml,
        loading: false,
      }))
    } catch {
      setYamlDrawer((prev) => ({ ...prev, yaml: '获取 YAML 失败', loading: false }))
    }
  }

  const handleViewHistory = async (record: any) => {
    setHistoryDrawer({ open: true, name: record.name, history: [], loading: true })
    try {
      const res = await getDeploymentHistory(clusterId, record.namespace, record.name)
      setHistoryDrawer((prev) => ({ ...prev, history: res || [], loading: false }))
    } catch {
      setHistoryDrawer((prev) => ({ ...prev, history: [], loading: false }))
    }
  }

  // ═══ Columns ═══
  const columns: ProColumns[] = [
    {
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 120,
      render: (_, r) => <Tag>{r.namespace}</Tag>,
    },
    { title: '名称', dataIndex: 'name', ellipsis: true },
    {
      title: 'READY',
      width: 120,
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
      render: (_, r) => <Text>{r.upToDate ?? 0}</Text>,
    },
    {
      title: 'AVAILABLE',
      dataIndex: 'available',
      width: 110,
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
      render: (_, r) =>
        r.paused ? <Tag color="warning">是</Tag> : <Text type="secondary">否</Text>,
    },
    { title: 'Age', dataIndex: 'age', width: 80 },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      fixed: 'right',
      render: (_, record) => (
        <Space size={8} style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
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
          <Tooltip title="YAML">
            <a onClick={() => handleViewYaml(record)}>
              <CodeOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除？"
            onConfirm={() => deleteMutation.mutate({ name: record.name, namespace: record.namespace })}
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
        search={false}
        loading={isLoading}
        dataSource={data?.items || []}
        toolBarRender={() => [
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
        scroll={{ x: 1600 }}
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

      {/* ═══ 编辑 Drawer ═══ */}
      <Drawer
        title={`编辑 ${workloadKind} - ${editDrawer.record?.name}`}
        open={editDrawer.open}
        onClose={() => setEditDrawer({ open: false })}
        width={600}
      >
        {editDrawer.record && (
          <ModalForm
            open={editDrawer.open}
            onOpenChange={(v) => !v && setEditDrawer({ open: false })}
            onFinish={async (values) => {
              updateMutation.mutate({ name: editDrawer.record.name, namespace: editDrawer.record.namespace, data: values })
              return true
            }}
            initialValues={{
              replicas: editDrawer.record.replicas,
              image: editDrawer.record.images?.[0],
              strategy: editDrawer.record.strategy,
            }}
          >
            <ProFormText name="name" label="名称" disabled initialValue={editDrawer.record.name} />
            <ProFormText
              name="namespace"
              label="命名空间"
              disabled
              initialValue={editDrawer.record.namespace}
            />
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
        )}
      </Drawer>

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
        onCancel={() => setImageModal({ open: false })}
        onOk={() => {
          const containerName = imageModal.containers?.[0]?.name
          const newImage = (document.getElementById('new-image-input') as HTMLInputElement)?.value
          if (containerName && newImage) {
            updateImageMutation.mutate({
              name: imageModal.name!,
              namespace: imageModal.ns!,
              container: containerName,
              image: newImage,
            })
            setImageModal({ open: false })
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
            <Input id="new-image-input" placeholder="nginx:1.25" />
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
        onClose={() => setYamlDrawer({ open: false, loading: false })}
        width={800}
      >
        <pre
          style={{
            background: '#1e1e1e',
            color: '#d4d4d4',
            padding: 16,
            borderRadius: 8,
            height: 'calc(100vh - 200px)',
            overflow: 'auto',
            fontSize: 13,
            lineHeight: 1.6,
            fontFamily: 'Consolas, Monaco, monospace',
          }}
        >
          {yamlDrawer.loading ? '加载中...' : yamlDrawer.yaml || '暂无数据'}
        </pre>
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
                {['版本', '变更原因', '镜像', '时间', '状态'].map((h) => (
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
                </tr>
              ))}
              {(!historyDrawer.history || historyDrawer.history.length === 0) && (
                <tr>
                  <td colSpan={5} style={{ padding: 20, textAlign: 'center' }}>
                    <Text type="secondary">暂无版本历史</Text>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </Drawer>
    </AppPage>
  )
}
