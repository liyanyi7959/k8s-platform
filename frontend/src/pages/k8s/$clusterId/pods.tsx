/**
 * Pod 管理页
 * 完整功能：列表+搜索筛选+批量聚合日志(Tab)+批量删除+强制删除
 * 单行：日志(多容器Tab)/终端/详情(5Tab)/YAML(编辑应用)/巡检/删除
 */
import React, { useState, useMemo, useRef, useEffect } from 'react'
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
  FileTextOutlined,
  ReloadOutlined,
  DesktopOutlined,
  InfoCircleOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
  EditOutlined,
  CheckOutlined,
  CloseOutlined,
  ProfileOutlined,
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
  createPodLogSession,
  listPodMetrics,
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

/** 关联所属渲染：可点击跳转的工作负载显示为链接
 *  ReplicaSet 自动回溯到 Deployment（RS 名格式为 <deploy-name>-<hash>）
 */
const OwnerTag: React.FC<{ clusterId: number; pod: Pod }> = ({ clusterId, pod }) => {
  if (!pod.ownerName) return <Text type="secondary">-</Text>

  let displayKind = pod.ownerKind || 'ReplicaSet'
  let displayName = pod.ownerName
  let routeKind = pod.ownerKind

  // ReplicaSet → Deployment 回溯：RS 名通常为 <deploy-name>-<hash>
  if (displayKind === 'ReplicaSet') {
    const lastDash = displayName.lastIndexOf('-')
    if (lastDash > 0) {
      const possibleDeployName = displayName.substring(0, lastDash)
      // RS 名中 hash 段是最后一个 - 后的部分，前面是 Deployment 名
      routeKind = 'Deployment'
      displayName = possibleDeployName
    }
  }

  const clickable = isWorkloadOwner(routeKind)
  const full = `${displayKind}/${pod.ownerName}`
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
      {clickable ? `${displayName}` : full}
    </Tag>
  )
  return clickable ? (
    <Tooltip title={`${routeKind}: ${displayName}（源: ${full}）`}>
      <span onClick={() => goOwner(clusterId, routeKind)}>{tag}</span>
    </Tooltip>
  ) : (
    <Tooltip title={full}>{tag}</Tooltip>
  )
}

// ═══════════════════════════════════════════
// 日志面板：每个 Tab 独立加载日志
// 支持 HTTP 一次性拉取 + WebSocket 实时流（follow）
// ═══════════════════════════════════════════
interface LogPaneProps {
  clusterId: number
  pod: Pod
  tailLines: number
  refreshNonce: number
  container?: string
  previous?: boolean
  keyword?: string
  live?: boolean
}
const LogPane: React.FC<LogPaneProps> = ({ clusterId, pod, tailLines, refreshNonce, container, previous, keyword, live }) => {
  const [liveLogs, setLiveLogs] = useState<string[]>([])
  const [liveConnected, setLiveConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const logEndRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // HTTP 一次性日志（非 live 模式使用）
  const { data, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['pod-logs', clusterId, pod.namespace, pod.name, container, tailLines, previous, refreshNonce],
    queryFn: () => getPodLogs(clusterId, pod.namespace, pod.name, { tailLines, container, previous }),
    enabled: !!clusterId && !!pod.namespace && !!pod.name && !live,
  })

  // WebSocket 实时流（live 模式使用）
  useEffect(() => {
    if (!live || !clusterId || !pod.namespace || !pod.name) return

    let cancelled = false
    setLiveLogs([])
    setLiveConnected(false)

    createPodLogSession(clusterId, pod.namespace, pod.name, { container, tailLines, follow: true, previous })
      .then(({ wsUrl }) => {
        if (cancelled || !wsUrl) return
        const ws = new WebSocket(wsUrl)
        wsRef.current = ws

        ws.onopen = () => setLiveConnected(true)
        ws.onmessage = (ev) => {
          const line = typeof ev.data === 'string' ? ev.data : ''
          if (line) {
            setLiveLogs((prev) => {
              const next = [...prev, line]
              return next.length > 5000 ? next.slice(-5000) : next // 限制 5000 行
            })
          }
        }
        ws.onclose = () => setLiveConnected(false)
        ws.onerror = () => setLiveConnected(false)
      })
      .catch(() => {})

    return () => {
      cancelled = true
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [live, clusterId, pod.namespace, pod.name, container, previous, tailLines, refreshNonce])

  // 自动滚动到底部
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [liveLogs])

  const rawContent = live
    ? liveLogs.join('\n')
    : (isLoading ? '加载中...' : data?.logs || '暂无日志')

  // 关键字过滤
  const content = keyword
    ? rawContent.split('\n').filter((l) => l.includes(keyword)).join('\n') || '无匹配日志'
    : rawContent

  return (
    <div>
      <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space size="small">
          {live && (
            <Badge status={liveConnected ? 'processing' : 'error'} text={liveConnected ? '实时连接中' : '连接断开'} />
          )}
          {previous && <Tag color="orange">历史日志</Tag>}
          {keyword && <Text type="secondary" style={{ fontSize: 11 }}>过滤: "{keyword}"</Text>}
        </Space>
        <Space size="small">
          <Tooltip title="复制日志">
            <Button size="small" onClick={() => { navigator.clipboard.writeText(rawContent); message.success('已复制') }}>
              复制
            </Button>
          </Tooltip>
          <Tooltip title="下载日志">
            <Button size="small" onClick={() => {
              const blob = new Blob([rawContent], { type: 'text/plain' })
              const a = document.createElement('a')
              a.href = URL.createObjectURL(blob)
              a.download = `${pod.name}${container ? '-' + container : ''}.log`
              a.click()
            }}>
              下载
            </Button>
          </Tooltip>
          {!live && (
            <Button size="small" icon={<ReloadOutlined />} loading={isFetching} onClick={() => refetch()}>
              刷新
            </Button>
          )}
        </Space>
      </div>
      <div
        ref={containerRef}
        style={{
          background: '#1e1e1e',
          color: '#d4d4d4',
          padding: 16,
          borderRadius: 8,
          height: 'calc(100vh - 300px)',
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
        <div ref={logEndRef} />
      </div>
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
      pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
      scroll={{ y: 320 }}
    />
  )
}

// ═══════════════════════════════════════════
// 资源用量 Tab：从 PodMetrics API 获取实际使用率
// ═══════════════════════════════════════════
const PodMetricsTab: React.FC<{ clusterId: number; pod: Pod }> = ({ clusterId, pod }) => {
  const { data: metricsList, isLoading } = useQuery({
    queryKey: ['pod-metrics', clusterId, pod.namespace],
    queryFn: ({ signal }) => listPodMetrics(clusterId, pod.namespace, signal),
    enabled: !!clusterId && !!pod.namespace,
    refetchInterval: 15_000, // 15s 刷新一次
  })

  // 从 metrics 列表中找到当前 Pod
  const podMetric = useMemo(() => {
    if (!metricsList || !pod.name) return undefined
    return metricsList.find((m: any) => {
      const name = m?.metadata?.name || m?.name
      return name === pod.name
    })
  }, [metricsList, pod.name])

  if (isLoading) return <Text type="secondary">加载中...</Text>
  if (!podMetric) return <Text type="secondary">暂无资源使用率数据（需部署 metrics-server）</Text>

  const containers = podMetric?.containers || podMetric?.status?.containers || []
  if (!containers.length) return <Text type="secondary">暂无容器指标数据</Text>

  return (
    <Table
      size="small"
      rowKey="name"
      pagination={false}
      dataSource={containers.map((c: any) => ({
        name: c.name || c.container || '',
        cpu: c.usage?.cpu || c.cpu || '-',
        memory: c.usage?.memory || c.memory || '-',
      }))}
      columns={[
        { title: '容器', dataIndex: 'name', width: 120, render: (t: string) => <Tag>{t}</Tag> },
        {
          title: 'CPU 使用',
          dataIndex: 'cpu',
          render: (v: string) => {
            if (!v || v === '-') return <Text type="secondary">-</Text>
            const millicores = v.endsWith('n') ? parseInt(v) / 1_000_000 : parseInt(v)
            return <Tag color="blue">{millicores >= 1000 ? `${(millicores / 1000).toFixed(2)} Core` : `${millicores} m`}</Tag>
          },
        },
        {
          title: '内存使用',
          dataIndex: 'memory',
          render: (v: string) => {
            if (!v || v === '-') return <Text type="secondary">-</Text>
            const bytes = parseInt(v.replace(/\D/g, '')) || 0
            const unit = v.replace(/[0-9]/g, '')
            let display = v
            if (unit === 'Ki') display = `${(bytes / 1024).toFixed(1)} MB`
            else if (unit === 'Mi') display = `${bytes.toFixed(1)} MB`
            else if (bytes > 0) display = bytes >= 1073741824 ? `${(bytes / 1073741824).toFixed(2)} GB` : `${(bytes / 1048576).toFixed(1)} MB`
            return <Tag color="purple">{display}</Tag>
          },
        },
      ]}
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
  const [searchType, setSearchType] = useState<'name' | 'node' | 'label'>('name')
  const [searchValue, setSearchValue] = useState('')
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
  const [logPrevious, setLogPrevious] = useState(false)
  const [logKeyword, setLogKeyword] = useState('')
  const [logLive, setLogLive] = useState(false)
  // YAML 原文备份（取消编辑时回滚）
  const [yamlOriginal, setYamlOriginal] = useState('')
  // 强制删除
  const [batchForce, setBatchForce] = useState(false)
  const [singleForce, setSingleForce] = useState(false)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['pods', clusterId, namespace],
    queryFn: ({ signal }) => listPods(clusterId, { namespace: namespace || undefined }, signal),
    enabled: !!clusterId,
    // 抽屉打开时暂停轮询，避免打断编辑/查看
    refetchInterval: (detailDrawer.open || yamlDrawer.open || logDrawer.open) ? false : 30_000,
  })

  // 客户端搜索筛选
  const filteredData = useMemo(() => {
    let list = data?.items || []
    if (searchValue.trim()) {
      const v = searchValue.trim().toLowerCase()
      list = list.filter((p) => {
        if (searchType === 'name') return p.name.toLowerCase().includes(v)
        if (searchType === 'node') return p.nodeName?.toLowerCase().includes(v)
        // label
        return p.labels && Object.entries(p.labels).some(([k, val]) =>
          `${k}=${val}`.toLowerCase().includes(v) || k.toLowerCase().includes(v) || String(val).toLowerCase().includes(v))
      })
    }
    if (statusFilter) list = list.filter((p) => p.status === statusFilter)
    return list
  }, [data, searchValue, searchType, statusFilter])

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
    // 多容器 Pod 选择容器
    let container: string | undefined
    if (pod.containers && pod.containers.length > 1) {
      const containers = pod.containers
      const firstContainerName = containers[0]!.name
      // 用 Modal 让用户选择容器
      const choice = await new Promise<string | undefined>((resolve) => {
        Modal.confirm({
          title: '选择终端容器',
          content: (
            <Select
              style={{ width: '100%', marginTop: 8 }}
              placeholder="选择容器"
              defaultValue={firstContainerName}
              onChange={(v) => { container = v }}
              options={containers.map((c) => ({ value: c.name, label: c.name }))}
            />
          ),
          onOk: () => resolve(container || firstContainerName),
          onCancel: () => resolve(undefined),
        })
      })
      if (!choice) return
      container = choice
    } else if (pod.containers && pod.containers.length === 1) {
      container = pod.containers[0]!.name
    }
    try {
      const res = await getPodTerminalUrl(clusterId, pod.namespace, pod.name, { container })
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
      setYamlOriginal(res.yaml)
    } catch {
      setYamlDrawer((prev) => ({ ...prev, yaml: '获取 YAML 失败', loading: false }))
      setYamlOriginal('')
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
          <LogPane clusterId={clusterId} pod={p} tailLines={tailLines} refreshNonce={logRefresh}
            previous={logPrevious} keyword={logKeyword} live={logLive} />
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
            <LogPane clusterId={clusterId} pod={pod} tailLines={tailLines} refreshNonce={logRefresh}
              container={c.name} previous={logPrevious} keyword={logKeyword} live={logLive} />
          ),
        }))
      }
      return [
        {
          key: pod.name,
          label: pod.name,
          children: (
            <LogPane clusterId={clusterId} pod={pod} tailLines={tailLines} refreshNonce={logRefresh}
              previous={logPrevious} keyword={logKeyword} live={logLive} />
          ),
        },
      ]
    }
    return []
  }, [logDrawer, clusterId, tailLines, logRefresh, logPrevious, logKeyword, logLive])

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
      title: 'Namespace',
      dataIndex: 'namespace',
      width: 110,
      render: (t) => <Tag>{t as string}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 130,
      render: (_, record) => <PodStatusTag status={record.status} containerReason={record.containerReason} ready={record.ready} />,
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
      sorter: (a, b) => a.restarts - b.restarts,
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
    { title: 'Age', dataIndex: 'createdAt', width: 90, ellipsis: true, sorter: (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(), render: (_, r) => formatDate(r.createdAt) },
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
              <ProfileOutlined />
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
        options={{ reload: false }}
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
            placeholder={searchType === 'name' ? '按名称搜索' : searchType === 'node' ? '按节点搜索' : '按标签搜索'}
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'node', label: '节点' }, { value: 'label', label: '标签' }]} />}
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
          <Space wrap>
            <Input.Search
              size="small"
              allowClear
              placeholder="日志过滤"
              value={logKeyword}
              onChange={(e) => setLogKeyword(e.target.value)}
              style={{ width: 120 }}
            />
            <Tooltip title="WebSocket 实时流式日志（follow）">
              <Button size="small" type={logLive ? 'primary' : 'default'} onClick={() => setLogLive(!logLive)}>
                实时
              </Button>
            </Tooltip>
            <Tooltip title="查看上一容器日志">
              <Button size="small" type={logPrevious ? 'primary' : 'default'} onClick={() => setLogPrevious(!logPrevious)} disabled={logLive}>
                历史
              </Button>
            </Tooltip>
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
        width={900}
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
                    <Descriptions.Item label="ServiceAccount">{detailPod.serviceAccountName || '-'}</Descriptions.Item>
                    <Descriptions.Item label="重启策略">{detailPod.restartPolicy || '-'}</Descriptions.Item>
                    <Descriptions.Item label="DNS 策略">{detailPod.dnsPolicy || '-'}</Descriptions.Item>
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
              {
                key: 'probes',
                label: '探针',
                children: detailPod.probes ? (
                  <Descriptions bordered column={1} size="small">
                    {[
                      { key: 'liveness', label: 'Liveness' },
                      { key: 'readiness', label: 'Readiness' },
                      { key: 'startup', label: 'Startup' },
                    ].map(({ key, label }) => {
                      const probe = (detailPod.probes as any)?.[key]
                      return (
                        <Descriptions.Item key={key} label={label}>
                          {probe ? (
                            <Space size="small" wrap>
                              {probe.path && <Tag>HTTP {probe.path}:{probe.port}</Tag>}
                              {!probe.path && probe.port && <Tag>TCP :{probe.port}</Tag>}
                              {probe.delay != null && <Text type="secondary" style={{ fontSize: 11 }}>延迟 {probe.delay}s</Text>}
                              {probe.period != null && <Text type="secondary" style={{ fontSize: 11 }}>周期 {probe.period}s</Text>}
                              {probe.timeout != null && <Text type="secondary" style={{ fontSize: 11 }}>超时 {probe.timeout}s</Text>}
                              {probe.failure != null && <Text type="secondary" style={{ fontSize: 11 }}>失败 {probe.failure}</Text>}
                            </Space>
                          ) : <Text type="secondary">未配置</Text>}
                        </Descriptions.Item>
                      )
                    })}
                  </Descriptions>
                ) : <Text type="secondary">暂无探针数据</Text>,
              },
              {
                key: 'scheduling',
                label: '调度',
                children: detailPod.scheduling ? (
                  <Descriptions bordered column={1} size="small">
                    <Descriptions.Item label="节点选择器">
                      {detailPod.scheduling.nodeSelector
                        ? Object.entries(detailPod.scheduling.nodeSelector).map(([k, v]) => <Tag key={k}>{k}={v}</Tag>)
                        : <Text type="secondary">无</Text>}
                    </Descriptions.Item>
                    <Descriptions.Item label="指定节点">{detailPod.scheduling.nodeName || '-'}</Descriptions.Item>
                    <Descriptions.Item label="优先级">{detailPod.scheduling.priorityClassName || '-'}</Descriptions.Item>
                    {detailPod.scheduling.tolerations?.length ? (
                      <Descriptions.Item label="容忍度">
                        <Table
                          size="small"
                          rowKey={(_, i) => String(i)}
                          pagination={false}
                          dataSource={detailPod.scheduling.tolerations}
                          columns={[
                            { title: 'Key', dataIndex: 'key', width: 100 },
                            { title: 'Operator', dataIndex: 'operator', width: 80 },
                            { title: 'Value', dataIndex: 'value', width: 80 },
                            { title: 'Effect', dataIndex: 'effect', width: 100 },
                            { title: 'Seconds', dataIndex: 'tolerationSeconds', width: 70 },
                          ]}
                        />
                      </Descriptions.Item>
                    ) : null}
                    {detailPod.scheduling.affinity && (
                      <Descriptions.Item label="亲和性">
                        <pre style={{ fontSize: 11, maxHeight: 200, overflow: 'auto' }}>{detailPod.scheduling.affinity}</pre>
                      </Descriptions.Item>
                    )}
                  </Descriptions>
                ) : <Text type="secondary">暂无调度数据</Text>,
              },
              {
                key: 'security',
                label: '安全',
                children: detailPod.security ? (
                  <Descriptions bordered column={1} size="small">
                    <Descriptions.Item label="ServiceAccount">{detailPod.security.serviceAccountName || '-'}</Descriptions.Item>
                    <Descriptions.Item label="runAsUser">{detailPod.security.runAsUser ?? '-'}</Descriptions.Item>
                    <Descriptions.Item label="runAsGroup">{detailPod.security.runAsGroup ?? '-'}</Descriptions.Item>
                    <Descriptions.Item label="runAsNonRoot">{detailPod.security.runAsNonRoot ? '是' : '否'}</Descriptions.Item>
                    <Descriptions.Item label="privileged">{detailPod.security.privileged ? <Tag color="error">是</Tag> : '否'}</Descriptions.Item>
                    <Descriptions.Item label="只读根文件系统">{detailPod.security.readOnlyRootFilesystem ? '是' : '否'}</Descriptions.Item>
                    {detailPod.security.capabilities && (
                      <Descriptions.Item label="Capabilities">
                        <Space direction="vertical" size={0}>
                          {detailPod.security.capabilities.add?.length ? (
                            <div><Text type="secondary" style={{ fontSize: 11 }}>添加: </Text>{detailPod.security.capabilities.add.map((c) => <Tag key={c} color="warning">{c}</Tag>)}</div>
                          ) : null}
                          {detailPod.security.capabilities.drop?.length ? (
                            <div><Text type="secondary" style={{ fontSize: 11 }}>移除: </Text>{detailPod.security.capabilities.drop.map((c) => <Tag key={c}>{c}</Tag>)}</div>
                          ) : null}
                        </Space>
                      </Descriptions.Item>
                    )}
                    {detailPod.security.imagePullSecrets?.length ? (
                      <Descriptions.Item label="镜像拉取密钥">
                        {detailPod.security.imagePullSecrets.map((s) => <Tag key={s}>{s}</Tag>)}
                      </Descriptions.Item>
                    ) : null}
                  </Descriptions>
                ) : <Text type="secondary">暂无安全数据</Text>,
              },
              {
                key: 'env',
                label: '环境变量',
                children: detailPod.envVars?.length ? (
                  <Table
                    size="small"
                    rowKey="name"
                    pagination={false}
                    dataSource={detailPod.envVars}
                    columns={[
                      { title: '名称', dataIndex: 'name', width: 180 },
                      { title: '值', dataIndex: 'value', ellipsis: true, render: (v: string) => v || '-' },
                      { title: '来源', dataIndex: 'valueFrom', width: 120, render: (v: string) => v ? <Tag>{v}</Tag> : '-' },
                    ]}
                  />
                ) : <Text type="secondary">暂无环境变量</Text>,
              },
              {
                key: 'metrics',
                label: '资源用量',
                children: <PodMetricsTab clusterId={clusterId} pod={detailPod} />,
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
              <Button icon={<CloseOutlined />} onClick={() => {
                setYamlDrawer((p) => ({ ...p, yaml: yamlOriginal, editing: false }))
              }}>
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
