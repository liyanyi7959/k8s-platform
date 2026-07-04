/**
 * Pod 管理页
 * 完整功能：列表+搜索筛选+批量聚合日志(Tab)+批量删除+强制删除
 * 单行：日志(多容器Tab)/终端/详情(5Tab)/YAML(编辑应用)/巡检/删除
 */
import React, { useState, useMemo } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
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
  Switch,
  Alert,
  Input,
  Select,
  Table,
} from 'antd'
import type { TableProps } from 'antd'
import {
  DeleteOutlined,
  CodeOutlined,
  FileTextOutlined,
  ReloadOutlined,
  DesktopOutlined,
  InfoCircleOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
  EditOutlined,
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { history } from '@umijs/max'
import {
  listPods,
  deletePod,
  getPodLogs,
  getPodTerminalUrl,
  getPodYaml,
  getPodInspection,
  getPodEvents,
  applyYaml,
} from '@/services/k8s'
import { AppPage, PodStatusTag, NamespaceSelector, YamlEditor } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import { withCenterStyleBatch } from '@/utils/fieldStyle'
import type { Pod, PodContainer, PodVolume } from '@/types'

const { Text } = Typography

/** 工作负载类型 ownerKind → 路由资源名 */
const WORKLOAD_KINDS = new Set(['Deployment', 'StatefulSet', 'DaemonSet'])
const ownerToRoute = (kind?: string): string => (kind ? `${kind.toLowerCase()}s` : '')
const isWorkloadOwner = (kind?: string): boolean => !!kind && WORKLOAD_KINDS.has(kind)

/** 跳转到关联工作负载页面 */
const goOwner = (clusterId: number, kind?: string) => {
  if (isWorkloadOwner(kind)) history.push(`/k8s/${clusterId}/${ownerToRoute(kind)}`)
}

/** 关联所属渲染：可点击跳转的工作负载显示为链接 */
const OwnerTag: React.FC<{ clusterId: number; pod: Pod }> = ({ clusterId, pod }) => {
  if (!pod.ownerName) return <Text type="secondary">-</Text>
  const full = `${pod.ownerKind || 'ReplicaSet'}/${pod.ownerName}`
  const clickable = isWorkloadOwner(pod.ownerKind)
  const tag = (
    <Tag
      style={{
        maxWidth: '100%',
        display: 'inline-block',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        verticalAlign: 'middle',
        cursor: clickable ? 'pointer' : 'default',
      }}
      color={clickable ? 'blue' : undefined}
    >
      {full}
    </Tag>
  )
  return clickable ? (
    <Tooltip title={`点击查看 ${pod.ownerKind}`}>
      <span onClick={() => goOwner(clusterId, pod.ownerKind)}>{tag}</span>
    </Tooltip>
  ) : (
    <Tooltip title={full}>{tag}</Tooltip>
  )
}

// ═══════════════════════════════════════════
// 日志面板：每个 Tab 独立加载日志
// ═══════════════════════════════════════════
interface LogPaneProps {
  clusterId: number
  pod: Pod
  tailLines: number
  refreshNonce: number
}
const LogPane: React.FC<LogPaneProps> = ({ clusterId, pod, tailLines, refreshNonce }) => {
  const { data, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['pod-logs', clusterId, pod.namespace, pod.name, tailLines, refreshNonce],
    queryFn: () => getPodLogs(clusterId, pod.namespace, pod.name, tailLines),
    enabled: !!clusterId && !!pod.namespace && !!pod.name,
  })
  const content = isLoading ? '加载中...' : data?.logs || '暂无日志'
  return (
    <div>
      <div style={{ marginBottom: 8, textAlign: 'right' }}>
        <Button size="small" icon={<ReloadOutlined />} loading={isFetching} onClick={() => refetch()}>
          刷新
        </Button>
      </div>
      <pre
        style={{
          background: '#1e1e1e',
          color: '#d4d4d4',
          padding: 16,
          borderRadius: 8,
          height: 'calc(100vh - 260px)',
          overflow: 'auto',
          fontSize: 13,
          lineHeight: 1.6,
          fontFamily: 'Consolas, Monaco, monospace',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-all',
          margin: 0,
        }}
      >
        {content}
      </pre>
    </div>
  )
}

// ═══════════════════════════════════════════
// 事件 Tab：详情抽屉打开时按需加载
// ═══════════════════════════════════════════
interface PodEvent {
  type?: string
  reason?: string
  message?: string
  lastTimestamp?: string
}
const PodEventsTab: React.FC<{ clusterId: number; pod: Pod }> = ({ clusterId, pod }) => {
  const { data, isLoading } = useQuery({
    queryKey: ['pod-events', clusterId, pod.namespace, pod.name],
    queryFn: ({ signal }) => getPodEvents(clusterId, pod.namespace, pod.name, signal),
    enabled: !!clusterId && !!pod.namespace && !!pod.name,
  })
  const events: PodEvent[] = (data || []) as PodEvent[]
  const columns: TableProps<PodEvent>['columns'] = [
    {
      title: '级别',
      dataIndex: 'type',
      width: 80,
      render: (t: string) => (
        <Badge status={t === 'Warning' ? 'warning' : t === 'Normal' ? 'success' : 'default'} text={t || '-'} />
      ),
    },
    { title: '原因', dataIndex: 'reason', width: 140, ellipsis: true, render: (t: string) => t || '-' },
    { title: '消息', dataIndex: 'message', render: (t: string) => t || '-' },
    {
      title: '时间',
      dataIndex: 'lastTimestamp',
      width: 160,
      render: (t: string) => (t ? formatDate(t) : '-'),
    },
  ]
  return (
    <Table<PodEvent>
      rowKey={(r, i) => `${r.reason}-${i}`}
      size="small"
      columns={columns}
      dataSource={events}
      loading={isLoading}
      pagination={{ pageSize: 10, size: 'small' }}
      scroll={{ y: 320 }}
    />
  )
}

// ═══════════════════════════════════════════
// Pods 主页面
// ═══════════════════════════════════════════
const PodsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [selectedKeys, setSelectedKeys] = useState<string[]>([])
  // 搜索筛选
  const [keyword, setKeyword] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')

  // ═══ 抽屉状态 ═══
  const [logDrawer, setLogDrawer] = useState<{
    open: boolean
    pod?: Pod
    pods?: Pod[]
  }>({ open: false })
  const [detailDrawer, setDetailDrawer] = useState<{ open: boolean; pod?: Pod }>({ open: false })
  const [yamlDrawer, setYamlDrawer] = useState<{
    open: boolean
    pod?: Pod
    yaml: string
    loading: boolean
    editing: boolean
  }>({ open: false, yaml: '', loading: false, editing: false })
  const [inspectDrawer, setInspectDrawer] = useState<{
    open: boolean
    pod?: Pod
    text: string
    loading: boolean
  }>({ open: false, text: '', loading: false })
  const [batchDeleteModal, setBatchDeleteModal] = useState(false)
  // 日志参数
  const [tailLines, setTailLines] = useState<number>(200)
  const [logRefresh, setLogRefresh] = useState(0)
  // 强制删除
  const [batchForce, setBatchForce] = useState(false)
  const [singleForce, setSingleForce] = useState(false)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['pods', clusterId, namespace],
    queryFn: ({ signal }) => listPods(clusterId, { namespace: namespace || undefined }, signal),
    enabled: !!clusterId,
    refetchInterval: 30_000,
  })

  // 客户端搜索筛选
  const filteredData = useMemo(() => {
    let list = data?.items || []
    if (keyword.trim()) list = list.filter((p) => p.name.includes(keyword.trim()))
    if (statusFilter) list = list.filter((p) => p.status === statusFilter)
    return list
  }, [data, keyword, statusFilter])

  const selectedPods = useMemo(
    () => filteredData.filter((p) => selectedKeys.includes(`${p.namespace}/${p.name}`)),
    [filteredData, selectedKeys],
  )

  // ═══ Mutations ═══
  const deleteMutation = useMutation({
    mutationFn: (params: { namespace: string; name: string; force?: boolean }) =>
      deletePod(clusterId, params.namespace, params.name, params.force),
    onSuccess: () => {
      message.success('Pod 已删除')
      queryClient.invalidateQueries({ queryKey: ['pods', clusterId] })
    },
    onError: () => message.error('删除失败'),
  })

  const batchDeleteMutation = useMutation({
    mutationFn: async (pods: Pod[]) => {
      for (const pod of pods) {
        await deletePod(clusterId, pod.namespace, pod.name, batchForce)
      }
    },
    onSuccess: () => {
      message.success(`已删除 ${selectedKeys.length} 个 Pod`)
      setSelectedKeys([])
      setBatchDeleteModal(false)
      queryClient.invalidateQueries({ queryKey: ['pods', clusterId] })
    },
    onError: () => message.error('批量删除失败'),
  })

  const applyYamlMutation = useMutation({
    mutationFn: (yaml: string) => applyYaml(clusterId, yaml),
    onSuccess: (res) => {
      if (res?.success) {
        message.success('YAML 应用成功')
        queryClient.invalidateQueries({ queryKey: ['pods', clusterId] })
        setYamlDrawer((prev) => ({ ...prev, open: false, editing: false }))
      } else {
        message.error(res?.message || 'YAML 应用失败')
      }
    },
    onError: () => message.error('YAML 应用失败'),
  })

  // ═══ Handlers ═══
  const handleViewLogs = (pod: Pod) => setLogDrawer({ open: true, pod })
  const handleMultiPodLogs = (pods: Pod[]) => setLogDrawer({ open: true, pods })

  const handleTerminal = async (pod: Pod) => {
    try {
      const res = await getPodTerminalUrl(clusterId, pod.namespace, pod.name)
      const token = localStorage.getItem('token') || ''
      const params = new URLSearchParams({ ws_url: res.url })
      if (token) params.set('token', token)
      window.open(`/k8s/terminal?${params.toString()}`, '_blank', 'width=900,height=600')
    } catch {
      message.error('获取终端连接失败')
    }
  }

  const handleViewYaml = async (pod: Pod) => {
    setYamlDrawer({ open: true, pod, yaml: '', loading: true, editing: false })
    try {
      const res = await getPodYaml(clusterId, pod.namespace, pod.name)
      setYamlDrawer((prev) => ({ ...prev, yaml: res.yaml, loading: false }))
    } catch {
      setYamlDrawer((prev) => ({ ...prev, yaml: '获取 YAML 失败', loading: false }))
    }
  }

  const handleInspection = async (pod: Pod) => {
    setInspectDrawer({ open: true, pod, text: '', loading: true })
    try {
      const res = await getPodInspection(clusterId, pod.namespace, pod.name)
      setInspectDrawer((prev) => ({ ...prev, text: res.text, loading: false }))
    } catch {
      setInspectDrawer((prev) => ({ ...prev, text: '获取巡检结果失败', loading: false }))
    }
  }

  // ═══ 日志 Tab 项 ═══
  const logTabs = useMemo(() => {
    if (logDrawer.pods?.length) {
      return logDrawer.pods.map((p) => ({
        key: `${p.namespace}/${p.name}`,
        label: p.name,
        children: (
          <LogPane clusterId={clusterId} pod={p} tailLines={tailLines} refreshNonce={logRefresh} />
        ),
      }))
    }
    if (logDrawer.pod) {
      const pod = logDrawer.pod
      const containers = pod.containers
      if (containers && containers.length > 1) {
        return containers.map((c: PodContainer) => ({
          key: c.name,
          label: c.name,
          children: (
            <LogPane clusterId={clusterId} pod={pod} tailLines={tailLines} refreshNonce={logRefresh} />
          ),
        }))
      }
      return [
        {
          key: pod.name,
          label: pod.name,
          children: (
            <LogPane clusterId={clusterId} pod={pod} tailLines={tailLines} refreshNonce={logRefresh} />
          ),
        },
      ]
    }
    return []
  }, [logDrawer, clusterId, tailLines, logRefresh])

  // ═══ 详情-容器表格列 ═══
  const containerColumns: TableProps<PodContainer>['columns'] = [
    { title: '名称', dataIndex: 'name', width: 120, render: (t: string) => <Tag>{t}</Tag> },
    { title: '镜像', dataIndex: 'image', ellipsis: true, render: (t: string) => <Text style={{ fontSize: 11 }}>{t}</Text> },
    {
      title: '就绪',
      dataIndex: 'ready',
      width: 70,
      render: (t?: boolean) => (t ? <Badge status="success" text="是" /> : <Badge status="error" text="否" />),
    },
    { title: '重启', dataIndex: 'restartCount', width: 60, render: (t?: number) => t ?? 0 },
    { title: '状态', dataIndex: 'state', width: 90, render: (t?: string) => t || '-' },
    {
      title: 'CPU(Request/Limit)',
      width: 150,
      render: (_, r) => `${r.cpuRequest || '-'} / ${r.cpuLimit || '-'}`,
    },
    {
      title: '内存(Request/Limit)',
      width: 160,
      render: (_, r) => `${r.memoryRequest || '-'} / ${r.memoryLimit || '-'}`,
    },
    {
      title: '端口',
      dataIndex: 'ports',
      render: (_, r) =>
        r.ports?.length
          ? r.ports.map((p) => (
              <Tag key={p.containerPort}>
                {p.containerPort}
                {p.protocol ? `/${p.protocol}` : ''}
              </Tag>
            ))
          : '-',
    },
  ]

  // ═══ 详情-条件表格列 ═══
  type PodCondition = NonNullable<Pod['conditions']>[number]
  const conditionColumns: TableProps<PodCondition>['columns'] = [
    { title: '类型', dataIndex: 'type', width: 120, render: (t: string) => <Tag>{t}</Tag> },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (t: string) => <Badge status={t === 'True' ? 'success' : 'error'} text={t} />,
    },
    {
      title: '最近变更',
      dataIndex: 'lastTransitionTime',
      width: 160,
      render: (t?: string) => (t ? formatDate(t) : '-'),
    },
  ]

  // ═══ 详情-存储表格列 ═══
  const volumeColumns: TableProps<PodVolume>['columns'] = [
    { title: '卷名', dataIndex: 'name', width: 140, render: (t: string) => <Tag color="cyan">{t}</Tag> },
    {
      title: '类型',
      dataIndex: 'type',
      width: 110,
      render: (t: string) => <Tag color="geekblue">{t || '-'}</Tag>,
    },
    { title: '来源', dataIndex: 'source', width: 140, render: (t?: string) => t || '-' },
    {
      title: '挂载路径',
      dataIndex: 'mountPaths',
      render: (_, r) =>
        r.mountPaths?.length
          ? r.mountPaths.map((m) => (
              <Tag key={`${m.name}-${m.path}`} style={{ marginBottom: 4 }}>
                {m.path}
                {m.readOnly ? ' (ro)' : ''}
              </Tag>
            ))
          : '-',
    },
  ]

  // ═══ 列表列 ═══
  const rawColumns: ProColumns<Pod>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (_, record) => (
        <a onClick={() => setDetailDrawer({ open: true, pod: record })} style={{ fontWeight: 500 }}>
          {record.name}
        </a>
      ),
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 110,
      render: (t) => <Tag>{t as string}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 130,
      render: (_, record) => <PodStatusTag status={record.status} ready={record.ready} />,
    },
    {
      title: 'Ready',
      dataIndex: 'ready',
      width: 70,
      render: (t) => {
        const v = (t as string) || ''
        const ok = v.split('/')[0] === v.split('/')[1] && v !== '0/0' && v !== ''
        return ok ? <Badge status="success" text={v} /> : <Badge status="warning" text={v || '-'} />
      },
    },
    {
      title: '重启',
      dataIndex: 'restarts',
      width: 70,
      render: (t) => {
        const v = (t as number) || 0
        return <span style={{ color: v > 5 ? '#ff4d4f' : v > 0 ? '#faad14' : '#52c41a' }}>{v}</span>
      },
    },
    {
      title: 'Pod IP',
      dataIndex: 'ip',
      width: 120,
      render: (t) => <Tag color="blue">{(t as string) || '-'}</Tag>,
    },
    { title: '节点', dataIndex: 'nodeName', width: 130, ellipsis: true, render: (t) => (t as string) || '-' },
    {
      title: '所属',
      width: 180,
      ellipsis: true,
      render: (_, r) => <OwnerTag clusterId={clusterId} pod={r} />,
    },
    {
      title: 'QoS',
      dataIndex: 'qosClass',
      width: 90,
      render: (t) => <Tag>{(t as string) || 'BestEffort'}</Tag>,
    },
    { title: 'Age', dataIndex: 'createdAt', width: 90, render: (_, r) => formatDate(r.createdAt) },
    {
      title: '操作',
      valueType: 'option',
      width: 200,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Tooltip title="查看日志">
            <a onClick={() => handleViewLogs(record)}>
              <FileTextOutlined />
            </a>
          </Tooltip>
          <Tooltip title="终端">
            <a onClick={() => handleTerminal(record)}>
              <DesktopOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看详情">
            <a onClick={() => setDetailDrawer({ open: true, pod: record })}>
              <InfoCircleOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => handleViewYaml(record)}>
              <CodeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="巡检诊断">
            <a onClick={() => handleInspection(record)}>
              <SafetyCertificateOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 Pod？"
            placement="left"
            description={
              <Space size="small">
                <span style={{ fontSize: 12 }}>强制删除</span>
                <Switch size="small" checked={singleForce} onChange={setSingleForce} />
              </Space>
            }
            onConfirm={() => {
              deleteMutation.mutate({ namespace: record.namespace, name: record.name, force: singleForce })
              setSingleForce(false)
            }}
            onCancel={() => setSingleForce(false)}
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

  const columns = withCenterStyleBatch(rawColumns as Parameters<typeof withCenterStyleBatch>[0])

  const detailPod = detailDrawer.pod

  return (
    <AppPage>
      <ProTable<Pod>
        headerTitle="Pod 列表"
        columns={columns}
        dataSource={filteredData}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 个 Pod` }}
        scroll={{ x: 1500 }}
        rowSelection={{
          selectedRowKeys: selectedKeys,
          onChange: (keys) => setSelectedKeys(keys as string[]),
        }}
        tableAlertRender={({ selectedRowKeys }) => (
          <Space>
            <Text>已选 {selectedRowKeys.length} 个 Pod</Text>
            <Button
              size="small"
              icon={<FileTextOutlined />}
              onClick={() => handleMultiPodLogs(selectedPods)}
              disabled={!selectedPods.length}
            >
              聚合日志
            </Button>
            <Button
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => setBatchDeleteModal(true)}
              disabled={!selectedPods.length}
            >
              批量删除
            </Button>
          </Space>
        )}
        toolBarRender={() => [
          <Input.Search
            key="search"
            allowClear
            placeholder="按名称搜索"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onSearch={(v) => setKeyword(v)}
            style={{ width: 180 }}
            prefix={<SearchOutlined />}
          />,
          <Select
            key="status"
            allowClear
            placeholder="状态筛选"
            value={statusFilter || undefined}
            onChange={(v) => setStatusFilter(v || '')}
            style={{ width: 130 }}
            options={[
              { value: 'Running', label: 'Running' },
              { value: 'Pending', label: 'Pending' },
              { value: 'Failed', label: 'Failed' },
              { value: 'Succeeded', label: 'Succeeded' },
            ]}
          />,
          <NamespaceSelector key="ns" clusterId={clusterId} value={namespace} onChange={setNamespace} />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>,
        ]}
      />

      {/* ═══ 日志抽屉（Tab 标签页）═══ */}
      <Drawer
        title={
          logDrawer.pods
            ? `聚合日志 - ${logDrawer.pods.length} 个 Pod`
            : `日志 - ${logDrawer.pod?.name}`
        }
        open={logDrawer.open}
        onClose={() => setLogDrawer({ open: false })}
        width={820}
        extra={
          <Space>
            <Select
              size="small"
              value={tailLines}
              onChange={setTailLines}
              style={{ width: 90 }}
              options={[
                { value: 50, label: '50 行' },
                { value: 200, label: '200 行' },
                { value: 500, label: '500 行' },
                { value: 1000, label: '1000 行' },
              ]}
            />
            <Button
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => setLogRefresh((n) => n + 1)}
            >
              全部刷新
            </Button>
          </Space>
        }
      >
        {logTabs.length > 0 && (
          <Tabs items={logTabs} size="small" destroyInactiveTabPane={false} />
        )}
      </Drawer>

      {/* ═══ 详情抽屉（5 个 Tab）═══ */}
      <Drawer
        title={`Pod 详情 - ${detailPod?.name}`}
        open={detailDrawer.open}
        onClose={() => setDetailDrawer({ open: false })}
        width={760}
      >
        {detailPod && (
          <Tabs
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <Descriptions bordered column={2} size="small">
                    <Descriptions.Item label="名称" span={2}>
                      {detailPod.name}
                    </Descriptions.Item>
                    <Descriptions.Item label="命名空间">
                      <Tag>{detailPod.namespace}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="状态">
                      <PodStatusTag status={detailPod.status} ready={detailPod.ready} />
                    </Descriptions.Item>
                    <Descriptions.Item label="Pod IP">{detailPod.ip || '-'}</Descriptions.Item>
                    <Descriptions.Item label="节点">{detailPod.nodeName || '-'}</Descriptions.Item>
                    <Descriptions.Item label="重启次数">{detailPod.restarts}</Descriptions.Item>
                    <Descriptions.Item label="QoS">{detailPod.qosClass || 'BestEffort'}</Descriptions.Item>
                    <Descriptions.Item label="所属">
                      <OwnerTag clusterId={clusterId} pod={detailPod} />
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间" span={2}>
                      {formatDate(detailPod.createdAt)}
                    </Descriptions.Item>
                    {detailPod.labels && (
                      <Descriptions.Item label="标签" span={2}>
                        {Object.entries(detailPod.labels).map(([k, v]) => (
                          <Tag key={k} color="blue">
                            {k}={v}
                          </Tag>
                        ))}
                      </Descriptions.Item>
                    )}
                    {detailPod.annotations && (
                      <Descriptions.Item label="注解" span={2}>
                        {Object.entries(detailPod.annotations)
                          .slice(0, 10)
                          .map(([k, v]) => (
                            <Tooltip key={k} title={v}>
                              <Tag style={{ marginBottom: 4 }}>{k}</Tag>
                            </Tooltip>
                          ))}
                      </Descriptions.Item>
                    )}
                  </Descriptions>
                ),
              },
              {
                key: 'containers',
                label: '容器',
                children: detailPod.containers?.length ? (
                  <Table<PodContainer>
                    rowKey="name"
                    size="small"
                    columns={containerColumns}
                    dataSource={detailPod.containers}
                    pagination={false}
                    scroll={{ x: 900 }}
                  />
                ) : (
                  <Text type="secondary">暂无容器数据</Text>
                ),
              },
              {
                key: 'events',
                label: '事件',
                children: <PodEventsTab clusterId={clusterId} pod={detailPod} />,
              },
              {
                key: 'conditions',
                label: '条件',
                children: detailPod.conditions?.length ? (
                  <Table
                    rowKey="type"
                    size="small"
                    columns={conditionColumns}
                    dataSource={detailPod.conditions}
                    pagination={false}
                  />
                ) : (
                  <Text type="secondary">暂无条件数据</Text>
                ),
              },
              {
                key: 'volumes',
                label: '存储',
                children: detailPod.volumes?.length ? (
                  <Table<PodVolume>
                    rowKey="name"
                    size="small"
                    columns={volumeColumns}
                    dataSource={detailPod.volumes}
                    pagination={false}
                  />
                ) : (
                  <Text type="secondary">暂无存储数据</Text>
                ),
              },
            ]}
          />
        )}
      </Drawer>

      {/* ═══ YAML 抽屉（编辑 + 应用）═══ */}
      <Drawer
        title={`YAML - ${yamlDrawer.pod?.name}`}
        open={yamlDrawer.open}
        onClose={() => setYamlDrawer({ open: false, yaml: '', loading: false, editing: false })}
        width={820}
        extra={
          yamlDrawer.editing ? (
            <Space>
              <Button
                icon={<CheckOutlined />}
                type="primary"
                loading={applyYamlMutation.isPending}
                onClick={() => applyYamlMutation.mutate(yamlDrawer.yaml)}
              >
                应用
              </Button>
              <Button icon={<CloseOutlined />} onClick={() => setYamlDrawer((p) => ({ ...p, editing: false }))}>
                取消
              </Button>
            </Space>
          ) : (
            <Button icon={<EditOutlined />} onClick={() => setYamlDrawer((p) => ({ ...p, editing: true }))}>
              编辑
            </Button>
          )
        }
      >
        {yamlDrawer.loading ? (
          <Text type="secondary">加载中...</Text>
        ) : (
          <YamlEditor
            value={yamlDrawer.yaml}
            readOnly={!yamlDrawer.editing}
            height={Math.max(window.innerHeight - 220, 400)}
            onChange={(val) => setYamlDrawer((p) => ({ ...p, yaml: val }))}
          />
        )}
      </Drawer>

      {/* ═══ 巡检诊断抽屉 ═══ */}
      <Drawer
        title={`巡检诊断 - ${inspectDrawer.pod?.name}`}
        open={inspectDrawer.open}
        onClose={() => setInspectDrawer({ open: false, text: '', loading: false })}
        width={820}
      >
        <pre
          style={{
            background: '#1e1e1e',
            color: '#d4d4d4',
            padding: 16,
            borderRadius: 4,
            height: 'calc(100vh - 200px)',
            overflow: 'auto',
            fontSize: 13,
            lineHeight: 1.6,
            fontFamily: 'Consolas, Monaco, monospace',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-all',
            margin: 0,
          }}
        >
          {inspectDrawer.loading ? '巡检中...' : inspectDrawer.text || '暂无巡检数据'}
        </pre>
      </Drawer>

      {/* ═══ 批量删除确认 ═══ */}
      <Modal
        title="批量删除 Pod"
        open={batchDeleteModal}
        onCancel={() => setBatchDeleteModal(false)}
        onOk={() => batchDeleteMutation.mutate(selectedPods)}
        confirmLoading={batchDeleteMutation.isPending}
        okText="确认删除"
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          message={`确定删除选中的 ${selectedKeys.length} 个 Pod？`}
          style={{ marginBottom: 12 }}
        />
        <Descriptions column={1} size="small">
          <Descriptions.Item label="强制删除">
            <Switch checked={batchForce} onChange={setBatchForce} />
          </Descriptions.Item>
        </Descriptions>
        <div style={{ marginTop: 12 }}>
          {selectedPods.map((p) => (
            <Tag key={`${p.namespace}/${p.name}`} color="red" style={{ marginBottom: 4 }}>
              {p.namespace}/{p.name}
            </Tag>
          ))}
        </div>
      </Modal>
    </AppPage>
  )
}

export default PodsPage
