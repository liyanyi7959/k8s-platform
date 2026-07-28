/**
 * 部署计划详情页
 * 展示计划信息、节点拓扑、分阶段步骤树、SSE 实时日志
 */
import { useEffect, useRef, useState, useCallback, useMemo } from 'react'
import { Card, Descriptions, Tag, Badge, Button, Space, Typography, message, Popconfirm, Tooltip, Progress, Input, Spin, Modal, Checkbox } from 'antd'
import { ArrowLeftOutlined, StopOutlined, RedoOutlined, DownloadOutlined, PlayCircleOutlined, SearchOutlined, ReloadOutlined, DesktopOutlined, SafetyCertificateOutlined, CheckCircleOutlined, CloseCircleOutlined, CaretRightOutlined, CodeOutlined } from '@ant-design/icons'
import { history, useParams, useSearchParams } from '@umijs/max'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AppPage, YamlEditor } from '@/components'
import {
  getDeployPlanById,
  getDeployTask,
  getDeployPlanAddonTask,
  getDeployTaskLogs,
  getDeployTaskLogSSEUrl,
  cancelDeployPlan,
  retryDeployPlan,
  retryDeployStep,
  installDeployPlanAddons,
  retryDeployPlanAddons,
  executeDeployPlan,
  getServers,
  getPlanAnsibleConfig,
  preflightDeployPlan,
  setDeployPreflightIgnore,
} from '@/features/provisioning/api/deploy'
import type { DeployPreflightResult, DeployTask, DeployTaskStep, DeployTaskSubStep } from '@/features/provisioning/types'

const { Text, Title } = Typography

const statusMap: Record<string, { badge: string; text: string }> = {
  draft: { badge: 'default', text: '草稿' },
  running: { badge: 'processing', text: '执行中' },
  success: { badge: 'success', text: '成功' },
  failed: { badge: 'error', text: '失败' },
  canceled: { badge: 'warning', text: '已取消' },
  cancelled: { badge: 'warning', text: '已取消' },
}

const taskStatusMap: Record<string, { badge: string; text: string }> = {
  pending: { badge: 'default', text: '等待中' },
  running: { badge: 'processing', text: '执行中' },
  success: { badge: 'success', text: '成功' },
  failed: { badge: 'error', text: '失败' },
  canceled: { badge: 'warning', text: '已取消' },
  cancelled: { badge: 'warning', text: '已取消' },
  timeout: { badge: 'error', text: '超时' },
}

const roleColorMap: Record<string, string> = {
  master: 'red',
  worker: 'blue',
}

const addonOptions = [
  { value: 'metrics-server', label: 'metrics-server' },
  { value: 'ingress-nginx', label: 'ingress-nginx' },
  { value: 'local-storage', label: 'local-storage' },
  { value: 'helm', label: '安装 Helm' },
]

// 步骤分组（stage -> steps）
const stageGroups = [
  { key: 'init', title: '节点初始化', steps: ['pre_check', 'bootstrap'] },
  { key: 'install', title: '安装 k8s 集群', steps: ['container_runtime', 'kubeadm_init', 'join_workers', 'install_cni'] },
  { key: 'extend', title: '集群扩展', steps: ['install_addons', 'register'] },
  { key: 'supplement', title: '后续补装', steps: [] },
]
const deploymentStepKeys = stageGroups.flatMap((stage) => stage.steps)

// 步骤标题映射
const stepTitleMap: Record<string, string> = {
  pre_check: '节点环境准备',
  bootstrap: '基础环境初始化',
  container_runtime: '容器运行时安装',
  kubeadm_init: 'Kubernetes Master 初始化',
  join_workers: 'Worker 节点加入集群',
  install_cni: '安装 CNI 网络插件',
  install_addons: '安装 Kubernetes 扩展组件',
  register: '节点注册到管理平台',
}

const pendingDeployTask: DeployTask = {
  id: 0,
  type: 'deploy_cluster',
  status: 'pending',
  percent: 0,
  steps: Object.entries(stepTitleMap).map(([key, title]) => ({ key, title, status: 'pending', subSteps: [] })),
  createdAt: '',
  createdBy: 0,
}

export default function DeployPlanDetailPage() {
  const params = useParams<{ id: string }>()
  const planId = Number(params.id)
  const [searchParams] = useSearchParams()
  const autoExecute = searchParams.get('execute') === '1'
  const queryClient = useQueryClient()
  const [selectedStepKey, setSelectedStepKey] = useState<string | null>(null)
  const [selectedSubStepKey, setSelectedSubStepKey] = useState<string | null>(null)
  const [logViewerOpen, setLogViewerOpen] = useState(false)
  const [logTaskSource, setLogTaskSource] = useState<'deployment' | 'addon'>('deployment')
  const [logs, setLogs] = useState<string[]>([])
  const [logTimestamps, setLogTimestamps] = useState<string[]>([])
  const [sseConnected, setSseConnected] = useState(false)
  const [logFilter, setLogFilter] = useState('')
  const [showLogTimestamps, setShowLogTimestamps] = useState(false)
  const [logTheme, setLogTheme] = useState<'light' | 'dark'>('light')
  const [preflightCollapsed, setPreflightCollapsed] = useState(true)
  const [ansibleConfigOpen, setAnsibleConfigOpen] = useState(false)
  const [topologyOpen, setTopologyOpen] = useState(false)
  const [addonModalOpen, setAddonModalOpen] = useState(false)
  const [selectedAddons, setSelectedAddons] = useState<string[]>([])
  const logOffsetRef = useRef(0)
  const logContainerRef = useRef<HTMLDivElement>(null)
  const logViewerRef = useRef<HTMLElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // 获取计划详情
  const { data: plan, isLoading: planLoading } = useQuery({
    queryKey: ['deploy-plan-detail', planId],
    queryFn: () => getDeployPlanById(planId),
    enabled: Number.isFinite(planId) && planId > 0,
    refetchInterval: (query) => {
      return query.state.data?.status === 'running' ? 3000 : false
    },
  })

  // 获取服务器列表
  const { data: serversData } = useQuery({
    queryKey: ['deploy-servers-for-detail'],
    queryFn: () => getServers({ page: 1, pageSize: 200 }),
    enabled: Number.isFinite(planId) && planId > 0,
  })

  // 获取 Ansible 执行配置
  const { data: ansibleConfig, isLoading: ansibleConfigLoading } = useQuery({
    queryKey: ['deploy-plan-ansible-config', planId],
    queryFn: () => getPlanAnsibleConfig(planId),
    enabled: Number.isFinite(planId) && planId > 0,
  })

  const {
    data: preflight,
    isFetching: preflightLoading,
    isError: preflightFailed,
    refetch: runPreflight,
  } = useQuery({
    queryKey: ['deploy-plan-preflight', planId],
    queryFn: () => preflightDeployPlan(planId),
    enabled: !!plan && ['draft', 'failed', 'cancelled', 'canceled'].includes(plan.status),
    refetchOnWindowFocus: false,
    retry: false,
  })

  const preflightReady = preflight?.ready
  useEffect(() => {
    if (preflightReady === false) {
      setPreflightCollapsed(false)
    }
  }, [preflightReady])

  const taskId = plan?.taskId
  const installedAddons = useMemo(() => {
    const installed = new Set(plan?.addons || [])
    if (plan?.helmInstall) installed.add('helm')
    return installed
  }, [plan?.addons, plan?.helmInstall])
  const selectableAddonCount = addonOptions.filter((option) => !installedAddons.has(option.value)).length

  const openAddonModal = () => {
    setSelectedAddons([])
    setAddonModalOpen(true)
  }

  const preflightIgnoreMutation = useMutation({
    mutationFn: ({ key, ignored }: { key: string; ignored: boolean }) => setDeployPreflightIgnore(planId, key, ignored),
    onSuccess: () => {
      message.success('预检放行设置已保存')
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-preflight', planId] })
    },
    onError: (err: any) => message.error(err?.message || '保存预检设置失败'),
  })

  // 获取任务详情
  const { data: task } = useQuery({
    queryKey: ['deploy-task', taskId],
    queryFn: () => getDeployTask(taskId!),
    enabled: !!taskId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'running' || status === 'pending' ? 2000 : false
    },
  })

  const { data: addonTask } = useQuery({
    queryKey: ['deploy-addon-task', planId],
    queryFn: () => getDeployPlanAddonTask(planId),
    enabled: Number.isFinite(planId) && planId > 0,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'running' || status === 'pending' ? 2000 : false
    },
  })

  const activeLogTask = logTaskSource === 'addon' ? addonTask : task
  const activeLogTaskId = activeLogTask?.id

  // 构建节点信息
  const nodeDetails = useMemo(() => {
    if (!plan?.nodes || !serversData?.items) return []
    const serverMap = new Map(serversData.items.map((s: any) => [s.id, s]))
    return plan.nodes.map((node: any) => {
      const server = serverMap.get(node.serverId)
      return {
        ...node,
        serverName: server?.name || `#${node.serverId}`,
        serverIP: server?.ip || '-',
        serverOS: server?.os || '-',
        serverStatus: server?.status || '-',
        serverSSHPort: server?.sshPort || 22,
      }
    })
  }, [plan?.nodes, serversData])

  const ansibleYaml = useMemo(() => {
    if (!ansibleConfig) return '# 暂无 Ansible 执行配置'
    const inventory = (ansibleConfig.inventory || '# 暂无 inventory')
      .split('\n')
      .map((line) => `  ${line}`)
      .join('\n')
    const extraVars = Object.entries(ansibleConfig.extraVars || {})
      .map(([key, value]) => `  ${key}: ${JSON.stringify(value)}`)
      .join('\n')
    return [
      `playbook: ${JSON.stringify(ansibleConfig.playbookPath || '')}`,
      'inventory: |',
      inventory,
      'extra_vars:',
      extraVars || '  {}',
    ].join('\n')
  }, [ansibleConfig])

  // 执行部署
  const executeMutation = useMutation({
    mutationFn: () => executeDeployPlan(planId),
    onSuccess: () => {
      message.success('部署已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
    },
    onError: (err: any) => {
      const errMsg = err?.message || '执行失败'
      message.error({
        content: errMsg,
        duration: 8,
      })
    },
  })

  // 自动执行：从列表页带 execute=1 跳转后，等 plan 和 preflight 加载完成自动触发
  const autoExecuteTriggered = useRef(false)
  useEffect(() => {
    if (!autoExecute || autoExecuteTriggered.current || !plan || plan.status !== 'draft') return
    autoExecuteTriggered.current = true
    executeMutation.mutate()
  }, [autoExecute, plan, executeMutation])

  // 取消部署
  const cancelMutation = useMutation({
    mutationFn: () => cancelDeployPlan(planId),
    onSuccess: () => {
      message.success('取消指令已发送')
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
      queryClient.invalidateQueries({ queryKey: ['deploy-task', taskId] })
    },
    onError: () => message.error('取消失败'),
  })

  // 失败或取消后从上次失败位置恢复；后端会重新执行就绪检查。
  const retryPlanMutation = useMutation({
    mutationFn: () => retryDeployPlan(planId),
    onSuccess: () => {
      message.success('部署重试已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
      queryClient.invalidateQueries({ queryKey: ['deploy-task'] })
    },
    onError: (err: any) => message.error(err?.message || '部署重试失败'),
  })

  // 从失败步骤重试部署
  // 重试步骤
  const retryStepMutation = useMutation({
    mutationFn: ({ stepKey }: { stepKey: string }) => retryDeployStep(planId, stepKey),
    onSuccess: () => {
      message.success('步骤重试已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
    },
    onError: () => message.error('步骤重试失败'),
  })

  const addonInstallMutation = useMutation({
    mutationFn: (addons: string[]) => installDeployPlanAddons(planId, addons),
    onSuccess: () => {
      setSelectedAddons([])
      setAddonModalOpen(false)
      message.success('附加组件安装任务已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-addon-task', planId] })
    },
    onError: (err: any) => message.error(err?.message || '附加组件安装任务启动失败'),
  })

  const addonRetryMutation = useMutation({
    mutationFn: () => retryDeployPlanAddons(planId),
    onSuccess: () => {
      message.success('附加组件安装重试已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-addon-task', planId] })
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
    },
    onError: (err: any) => message.error(err?.message || '附加组件安装重试失败'),
  })

  useEffect(() => {
    if (!addonTask || (addonTask.status !== 'success' && addonTask.status !== 'failed' && addonTask.status !== 'canceled')) return
    queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
  }, [addonTask?.id, addonTask?.status, planId, queryClient])

  // 当前选中的 step key（用于日志过滤）
  const effectiveStepKey = useMemo(() => {
    if (selectedSubStepKey) {
      // sub step key 格式为 {stepKey}-{index}，提取大 step key
      return activeLogTask?.steps?.find((step: DeployTaskStep) =>
        step.subSteps?.some((sub: DeployTaskSubStep) => sub.key === selectedSubStepKey),
      )?.key || selectedStepKey
    }
    return selectedStepKey
  }, [selectedStepKey, selectedSubStepKey, activeLogTask?.steps])

  // 初始加载历史日志。旧任务在 Ansible PLAY 开始前产生的 Runner/SSH 错误没有
  // step_key；步骤日志为空时回退到完整任务日志，保证历史失败也可诊断。
  useEffect(() => {
    if (!activeLogTaskId) {
      setLogs([])
      setLogTimestamps([])
      logOffsetRef.current = 0
      return
    }
    // 默认选中第一个非成功的步骤，或第一个步骤
    const steps = activeLogTask?.steps || []
    if (!selectedStepKey && steps.length > 0) {
      const active = steps.find((s: DeployTaskStep) => s.status === 'running' || s.status === 'failed')
      setSelectedStepKey(active?.key || steps[0]?.key || null)
    }

    let active = true
    setLogs([])
    setLogTimestamps([])
    logOffsetRef.current = 0
    const loadLogs = async () => {
      try {
        let res = await getDeployTaskLogs(activeLogTaskId, 0, 500, effectiveStepKey || undefined)
        const selectedStepFailed = steps.find((step: DeployTaskStep) => step.key === effectiveStepKey)?.status === 'failed'
        if (effectiveStepKey && selectedStepFailed && (res.logs || []).length === 0) {
          res = await getDeployTaskLogs(activeLogTaskId, 0, 500)
        }
        if (!active) return
        const fetched = res.logs || []
        setLogs(fetched)
        setLogTimestamps((res.entries || []).map((entry) => entry.createdAt))
        logOffsetRef.current = fetched.length
      } catch {
        if (!active) return
        setLogs([])
        setLogTimestamps([])
        logOffsetRef.current = 0
      }
    }
    void loadLogs()
    return () => { active = false }
  }, [activeLogTaskId, effectiveStepKey, activeLogTask?.steps])

  // SSE 实时日志连接
  const connectSSE = useCallback(() => {
    if (!activeLogTaskId) return
    eventSourceRef.current?.close()
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }

    const token = localStorage.getItem('token') || ''
    const url = getDeployTaskLogSSEUrl(activeLogTaskId, effectiveStepKey || undefined)
    const sep = url.includes('?') ? '&' : '?'
    const fullUrl = `${url}${sep}token=${encodeURIComponent(token)}`

    const es = new EventSource(fullUrl)
    eventSourceRef.current = es

    es.onopen = () => setSseConnected(true)

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.log) {
          setLogs((prev) => [...prev, data.log])
          setLogTimestamps((prev) => [...prev, data.timestamp || ''])
        }
      } catch {
        setLogs((prev) => [...prev, event.data])
        setLogTimestamps((prev) => [...prev, ''])
      }
    }

    es.addEventListener('done', (event: any) => {
      try {
        const data = JSON.parse(event.data)
        message.info(`任务已结束：${data.status || ''}`)
      } catch { /* ignore */ }
      es.close()
      setSseConnected(false)
      if (activeLogTaskId) {
        if (logTaskSource === 'addon') {
          queryClient.invalidateQueries({ queryKey: ['deploy-addon-task', planId] })
        } else {
          queryClient.invalidateQueries({ queryKey: ['deploy-task', activeLogTaskId] })
        }
        queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
      }
    })

    es.onerror = () => {
      setSseConnected(false)
      es.close()
      const currentStatus = activeLogTask?.status
      if (currentStatus === 'running' || currentStatus === 'pending') {
        reconnectTimerRef.current = setTimeout(() => connectSSE(), 3000)
      }
    }
  }, [activeLogTaskId, planId, queryClient, activeLogTask?.status, effectiveStepKey, logTaskSource])

  useEffect(() => {
    if (!activeLogTaskId) return
    const taskStatus = activeLogTask?.status
    if (taskStatus === 'running' || taskStatus === 'pending' || taskStatus === undefined) {
      connectSSE()
    }
    return () => {
      eventSourceRef.current?.close()
      eventSourceRef.current = null
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = null
      }
    }
  }, [activeLogTaskId, activeLogTask?.status, connectSSE])

  // 自动滚动到底部
  useEffect(() => {
    if (logContainerRef.current && !logFilter) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }, [logs, logFilter])

  useEffect(() => {
    if (!logViewerOpen) return
    const closeWhenClickedOutside = (event: MouseEvent) => {
      if (!logViewerRef.current?.contains(event.target as Node)) {
        setLogViewerOpen(false)
      }
    }
    document.addEventListener('mousedown', closeWhenClickedOutside)
    return () => document.removeEventListener('mousedown', closeWhenClickedOutside)
  }, [logViewerOpen])

  // 下载日志
  const handleDownloadLogs = () => {
    const content = logs.join('\n')
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `deploy-plan-${planId}-logs.txt`
    a.click()
    URL.revokeObjectURL(url)
  }

  const planStatus = plan?.status || 'draft'
  const isRunning = planStatus === 'running'
  const isDraft = planStatus === 'draft'
  const canRetry = planStatus === 'failed' || planStatus === 'cancelled' || planStatus === 'canceled'

  const currentTaskStatus = task?.status
  const taskBadge = taskStatusMap[currentTaskStatus || ''] || { badge: 'default', text: currentTaskStatus || '-' }

  // 根据选中 sub step 进一步过滤日志（客户端过滤）
  const filteredLogEntries = useMemo(() => {
    let list = logs.map((content, index) => ({ content, timestamp: logTimestamps[index] || '' }))
    if (logFilter) {
      const keyword = logFilter.toLowerCase()
      list = list.filter((entry) => entry.content.toLowerCase().includes(keyword))
    }
    if (selectedSubStepKey && activeLogTask) {
      const step = activeLogTask.steps?.find((s: DeployTaskStep) => s.key === selectedStepKey)
      const sub = step?.subSteps?.find((s: DeployTaskSubStep) => s.key === selectedSubStepKey)
      if (sub) {
        // 高亮子步骤：只保留包含子步骤标题的 TASK 行及其后直到下一个 TASK 行
        const subTitle = sub.title
        const result: Array<{ content: string; timestamp: string }> = []
        let inSubStep = false
        for (const entry of list) {
          const trimmed = entry.content.trim()
          if (trimmed.startsWith('TASK [')) {
            inSubStep = trimmed.includes(subTitle)
          }
          if (inSubStep || trimmed.includes(subTitle)) {
            result.push(entry)
          }
        }
        // 如果没有匹配到 TASK 行（可能日志格式不同），返回整个 step 日志
        return result.length > 0 ? result : list
      }
    }
    return list
  }, [logs, logTimestamps, logFilter, selectedSubStepKey, selectedStepKey, activeLogTask])

  const filteredLogs = useMemo(() => filteredLogEntries.map((entry) => entry.content), [filteredLogEntries])
  const selectedFailureMessage = useMemo(() => {
    const selectedStep = activeLogTask?.steps?.find((step: DeployTaskStep) => step.key === selectedStepKey)
    if (selectedStep?.status === 'failed' && selectedStep.message) return selectedStep.message
    if (activeLogTask?.status === 'failed') return activeLogTask.message || '任务失败，请查看执行日志。'
    return ''
  }, [selectedStepKey, activeLogTask])
  const logThemeTokens = logTheme === 'dark'
    ? { panel: '#161616', header: '#202020', headerBorder: '#383838', text: '#f5f5f5', muted: '#a6a6a6', viewer: '#101010', viewerBorder: '#343434', empty: '#a6a6a6', button: '#262626', buttonBorder: '#454545' }
    : { panel: '#ffffff', header: '#fafafa', headerBorder: '#e8e8e8', text: '#262626', muted: '#8c8c8c', viewer: '#fafafa', viewerBorder: '#d9d9d9', empty: '#8c8c8c', button: '#ffffff', buttonBorder: '#d9d9d9' }

  // 步骤选择处理
  const handleSelectStep = (stepKey: string) => {
    setLogTaskSource('deployment')
    setSelectedStepKey(stepKey)
    setSelectedSubStepKey(null)
    setLogViewerOpen(true)
  }

  const handleSelectSubStep = (stepKey: string, subKey: string) => {
    setLogTaskSource('deployment')
    setSelectedStepKey(stepKey)
    setSelectedSubStepKey(subKey)
    setLogViewerOpen(true)
  }

  const handleSelectAddonTask = (subStepKey?: string) => {
    if (!addonTask) return
    setLogTaskSource('addon')
    setSelectedStepKey(addonTask.steps?.[0]?.key || (addonTask.meta?.addons?.length === 1 && addonTask.meta.addons[0] === 'helm' ? 'install_helm' : 'install_addons'))
    setSelectedSubStepKey(subStepKey || null)
    setLogViewerOpen(true)
  }

  return (
    <AppPage>
      <Card loading={planLoading}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => history.push('/clusters/provision')}>返回集群部署列表</Button>
            <Title level={4} style={{ margin: 0 }}>{plan?.name || '部署详情'}</Title>
            <Badge status={statusMap[planStatus]?.badge as any} text={statusMap[planStatus]?.text || planStatus} />
          </Space>
          <Space>
            {isDraft && (
              <Popconfirm title="确认执行该部署方案？" onConfirm={() => executeMutation.mutate()}>
                <Button type="primary" icon={<PlayCircleOutlined />} disabled={!preflight?.ready} loading={executeMutation.isPending}>执行部署</Button>
              </Popconfirm>
            )}
            {isRunning && (
              <Popconfirm title="确认取消该部署任务？" onConfirm={() => cancelMutation.mutate()}>
                <Button danger icon={<StopOutlined />} loading={cancelMutation.isPending}>取消部署</Button>
              </Popconfirm>
            )}
            {canRetry && (
              <Popconfirm title="将重新执行就绪检查，并从上次失败位置继续，确认重试？" onConfirm={() => retryPlanMutation.mutate()}>
                <Button type="primary" icon={<RedoOutlined />} loading={retryPlanMutation.isPending}>从失败处重试</Button>
              </Popconfirm>
            )}
            <Tooltip title="刷新">
              <Button icon={<ReloadOutlined />} onClick={() => {
                queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
                if (taskId) queryClient.invalidateQueries({ queryKey: ['deploy-task', taskId] })
                queryClient.invalidateQueries({ queryKey: ['deploy-addon-task', planId] })
              }} />
            </Tooltip>
          </Space>
        </div>

        {/* 计划基本信息 */}
        <Descriptions bordered column={3} size="small" style={{ marginBottom: 16 }}>
          <Descriptions.Item label="集群名称">{plan?.clusterName || '-'}</Descriptions.Item>
          <Descriptions.Item label="K8s 版本">
            {plan?.k8sVersion ? <Tag color="blue">{plan.k8sVersion}</Tag> : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="CNI 类型">
            {plan?.cniType ? <Tag color="green">{plan.cniType}</Tag> : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Pod 网段"><Text code>{plan?.podCidr || '-'}</Text></Descriptions.Item>
          <Descriptions.Item label="Service 网段"><Text code>{plan?.svcCidr || '-'}</Text></Descriptions.Item>
          <Descriptions.Item label="任务状态">
            {taskId ? <Badge status={taskBadge.badge as any} text={taskBadge.text} /> : <Text type="secondary">未执行</Text>}
          </Descriptions.Item>
          <Descriptions.Item label="附加组件" span={2}>
            <Space wrap>
              {plan?.addons?.length ? plan.addons.map((addon: string) => <Tag key={addon}>{addon}</Tag>) : <Text type="secondary">未安装</Text>}
              {planStatus === 'success' && !!plan?.clusterId && (
                <Button
                  size="small"
                  type="link"
                  onClick={openAddonModal}
                >
                  补充组件
                </Button>
              )}
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="Helm 安装">
            {plan?.helmInstall ? <Tag color="blue">已启用</Tag> : <Text type="secondary">未启用</Text>}
          </Descriptions.Item>
          <Descriptions.Item label="节点拓扑">
            <Button
              size="small"
              type="link"
              icon={<DesktopOutlined />}
              disabled={nodeDetails.length === 0}
              onClick={() => setTopologyOpen(true)}
            >
              查看拓扑
            </Button>
          </Descriptions.Item>
          <Descriptions.Item label="Ansible 配置" span={2}>
            <Button
              size="small"
              type="link"
              icon={<CodeOutlined />}
              loading={ansibleConfigLoading}
              onClick={() => setAnsibleConfigOpen(true)}
            >
              查看 YAML
            </Button>
          </Descriptions.Item>
        </Descriptions>

        {(!task || !task.steps || task.steps.length === 0) && (
          <Card size="small" title={<DeployProgressTitle />} style={{ marginBottom: 12 }}>
            <div style={{ height: 400, minHeight: 360, overflow: 'hidden' }}>
                <StepTree
                task={pendingDeployTask}
                supplementTask={addonTask}
                supplementSelected={false}
                canRetrySupplement={false}
                canRetry={false}
                selectedStepKey={null}
                selectedSubStepKey={null}
                onSelectStep={() => undefined}
                onSelectSubStep={() => undefined}
                onSelectSupplementTask={() => undefined}
                onRetrySupplement={() => undefined}
                supplementRetrying={false}
                onRetryStep={() => undefined}
                retrying={false}
                readiness={{
                  preflight,
                  failed: preflightFailed,
                  loading: preflightLoading,
                  collapsed: preflightCollapsed,
                  onToggle: () => setPreflightCollapsed((collapsed) => !collapsed),
                  onRefresh: () => { void runPreflight() },
                  refreshable: isDraft || canRetry,
                  onToggleIgnore: (key, ignored) => preflightIgnoreMutation.mutate({ key, ignored }),
                  ignoreLoading: preflightIgnoreMutation.isPending,
                }}
              />
            </div>
          </Card>
        )}

        {/* 部署进度 + 日志 左右布局 */}
        {task && task.steps && task.steps.length > 0 && (
          <Card size="small" title={<DeployProgressTitle task={task} />} style={{ marginBottom: 12 }}>
            <div style={{ height: 460, minHeight: 390, position: 'relative', overflow: 'hidden' }}>
              {/* 左侧步骤树 */}
              <div style={{ height: '100%', overflow: 'hidden' }}>
                <StepTree
                  task={task}
                  supplementTask={addonTask}
                  supplementSelected={logViewerOpen && logTaskSource === 'addon'}
                  canRetrySupplement={addonTask?.status === 'failed' || addonTask?.status === 'canceled'}
                  canRetry={canRetry}
                  selectedStepKey={selectedStepKey}
                  selectedSubStepKey={selectedSubStepKey}
                  onSelectStep={handleSelectStep}
                  onSelectSubStep={handleSelectSubStep}
                  onSelectSupplementTask={handleSelectAddonTask}
                  onRetrySupplement={() => addonRetryMutation.mutate()}
                  supplementRetrying={addonRetryMutation.isPending}
                  onRetryStep={(stepKey) => retryStepMutation.mutate({ stepKey })}
                  retrying={retryStepMutation.isPending}
                  readiness={{
                    preflight,
                    failed: preflightFailed,
                    loading: preflightLoading,
                    collapsed: preflightCollapsed,
                    onToggle: () => setPreflightCollapsed((collapsed) => !collapsed),
                    onRefresh: () => { void runPreflight() },
                    refreshable: isDraft || canRetry,
                    onToggleIgnore: (key, ignored) => preflightIgnoreMutation.mutate({ key, ignored }),
                    ignoreLoading: preflightIgnoreMutation.isPending,
                    assumedReady: !preflight && !preflightLoading && !preflightFailed && !canRetry,
                  }}
                />
              </div>

              {logViewerOpen && (
                <div
                  role="dialog"
                  aria-modal="true"
                  aria-label="步骤日志"
                  onClick={() => setLogViewerOpen(false)}
                  style={{
                    position: 'absolute',
                    inset: 0,
                    zIndex: 10,
                    padding: 12,
                    display: 'flex',
                    justifyContent: 'flex-end',
                  }}
                >
                  <section
                    ref={logViewerRef}
                    onClick={(event) => event.stopPropagation()}
                    style={{
                      height: '100%',
                      width: '70%',
                      minHeight: 0,
                      display: 'flex',
                      flexDirection: 'column',
                      overflow: 'hidden',
                      borderRadius: 10,
                      border: `1px solid ${logThemeTokens.viewerBorder}`,
                      background: logThemeTokens.panel,
                      boxShadow: '0 12px 30px rgba(0, 0, 0, 0.24)',
                    }}
                  >
                    <header
                      style={{
                        flex: '0 0 auto',
                        minHeight: 58,
                        padding: '0 18px',
                        display: 'flex',
                        alignItems: 'center',
                        gap: 12,
                        borderBottom: `1px solid ${logThemeTokens.headerBorder}`,
                        background: logThemeTokens.header,
                        color: logThemeTokens.text,
                      }}
                    >
                      <span style={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 16, fontWeight: 600 }}>
                        {selectedSubStepKey
                          ? activeLogTask?.steps?.find((step: DeployTaskStep) => step.key === selectedStepKey)?.subSteps?.find((sub: DeployTaskSubStep) => sub.key === selectedSubStepKey)?.title
                          : selectedStepKey
                            ? activeLogTask?.steps?.find((step: DeployTaskStep) => step.key === selectedStepKey)?.title || stepTitleMap[selectedStepKey] || selectedStepKey
                            : '任务日志'}
                      </span>
                      <span style={{ flex: '0 0 auto', color: logThemeTokens.muted, fontSize: 12 }}>共 {activeLogTask?.steps?.length || 0} 个步骤</span>
                      <Button
                        aria-label="关闭日志"
                        type="text"
                        size="small"
                        icon={<CloseCircleOutlined />}
                        onClick={() => setLogViewerOpen(false)}
                        style={{ flex: '0 0 auto', color: logThemeTokens.muted }}
                      />
                    </header>

                    <div style={{ minHeight: 0, flex: 1, padding: 18, display: 'flex', flexDirection: 'column', background: logThemeTokens.panel }}>
                      <div style={{ flex: '0 0 auto', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16, marginBottom: 12 }}>
                        <div style={{ minWidth: 0, display: 'flex', alignItems: 'center', gap: 10 }}>
                          <span style={{ color: logThemeTokens.text, fontSize: 15, fontWeight: 600 }}>日志输出</span>
                          <span style={{ color: logThemeTokens.muted, fontSize: 12 }}>{filteredLogs.length} / {logs.length} 行</span>
                        </div>
                        <Space size={8} style={{ flex: '0 0 auto' }}>
                          <Input
                            size="small"
                            placeholder="过滤日志..."
                            prefix={<SearchOutlined />}
                            value={logFilter}
                            onChange={(event) => setLogFilter(event.target.value)}
                            style={{ width: 220, background: logThemeTokens.viewer, borderColor: logThemeTokens.buttonBorder, color: logThemeTokens.text }}
                            allowClear
                          />
                          <Tooltip title="下载当前日志">
                            <Button size="small" icon={<DownloadOutlined />} onClick={handleDownloadLogs} disabled={logs.length === 0} style={{ background: logThemeTokens.button, borderColor: logThemeTokens.buttonBorder, color: logThemeTokens.text }} />
                          </Tooltip>
                          <Button size="small" onClick={() => setShowLogTimestamps((visible) => !visible)} style={{ background: logThemeTokens.button, borderColor: logThemeTokens.buttonBorder, color: logThemeTokens.text }}>
                            {showLogTimestamps ? '隐藏时间' : '时间戳'}
                          </Button>
                          <Button size="small" onClick={() => setLogTheme((theme) => theme === 'light' ? 'dark' : 'light')} style={{ background: logThemeTokens.button, borderColor: logThemeTokens.buttonBorder, color: logThemeTokens.text }}>
                            {logTheme === 'light' ? '深色' : '浅色'}
                          </Button>
                        </Space>
                      </div>

                      <div
                        ref={logContainerRef}
                        style={{
                          flex: 1,
                          minHeight: 0,
                          overflow: 'auto',
                          border: `1px solid ${logThemeTokens.viewerBorder}`,
                          borderRadius: 8,
                          padding: 18,
                          background: logThemeTokens.viewer,
                          color: logThemeTokens.text,
                          fontFamily: 'Consolas, Monaco, "Courier New", monospace',
                          fontSize: 13,
                          lineHeight: 1.7,
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                        }}
                      >
                        {logs.length === 0 ? (
                          <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24, color: logThemeTokens.empty }}>
                            {selectedFailureMessage ? (
                              <div style={{ width: '100%', maxWidth: 720, padding: '14px 16px', border: '1px solid #fecaca', borderRadius: 8, background: '#fff7f7', color: '#b91c1c' }}>
                                <div style={{ marginBottom: 6, fontWeight: 600 }}>失败原因</div>
                                <div style={{ color: '#7f1d1d', lineHeight: 1.6 }}>{selectedFailureMessage}</div>
                              </div>
                            ) : '该步骤暂未产生可展示的日志'}
                          </div>
                        ) : filteredLogs.length === 0 ? (
                          <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: logThemeTokens.empty }}>没有匹配的日志内容</div>
                        ) : (
                          filteredLogEntries.map((entry, index) => (
                            <div key={index} style={{ display: 'flex', gap: 10, color: getLogColor(entry.content, logTheme) }}>
                              {showLogTimestamps && <span style={{ flex: '0 0 auto', color: logThemeTokens.muted }}>{formatLogTimestamp(entry.timestamp)}</span>}
                              <span style={{ minWidth: 0 }}>{entry.content || ' '}</span>
                            </div>
                          ))
                        )}
                      </div>
                    </div>
                  </section>
                </div>
              )}

              {/* 右侧日志面板 */}
              <div style={{ display: 'none' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                  <Space>
                    <Text strong>
                      {selectedSubStepKey
                        ? task.steps?.find((s: DeployTaskStep) => s.key === selectedStepKey)?.subSteps?.find((s: DeployTaskSubStep) => s.key === selectedSubStepKey)?.title
                        : selectedStepKey
                          ? stepTitleMap[selectedStepKey] || selectedStepKey
                          : '全部日志'}
                    </Text>
                    {sseConnected && <Badge status="processing" text="实时连接" />}
                    {!sseConnected && isRunning && <Badge status="warning" text="重连中" />}
                    {task && task.status !== 'running' && task.status !== 'pending' && (
                      <Badge status="default" text="已结束" />
                    )}
                    {logs.length > 0 && <Text type="secondary" style={{ fontSize: 12 }}>({filteredLogs.length} / {logs.length} 行)</Text>}
                  </Space>
                  <Space>
                    <Input
                      size="small"
                      placeholder="过滤日志..."
                      prefix={<SearchOutlined />}
                      value={logFilter}
                      onChange={(e) => setLogFilter(e.target.value)}
                      style={{ width: 180 }}
                      allowClear
                    />
                    <Tooltip title="下载日志">
                      <Button size="small" icon={<DownloadOutlined />} onClick={handleDownloadLogs} disabled={logs.length === 0} />
                    </Tooltip>
                  </Space>
                </div>
                <div
                  ref={logContainerRef}
                  style={{
                    flex: 1,
                    background: '#1e1e1e',
                    color: '#d4d4d4',
                    fontFamily: 'Consolas, Monaco, "Courier New", monospace',
                    fontSize: 13,
                    lineHeight: 1.6,
                    padding: 16,
                    borderRadius: 6,
                    overflowY: 'auto',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-all',
                  }}
                >
                  {logs.length === 0 ? (
                    <span style={{ color: '#666' }}>
                      {isDraft ? '# 当前方案尚未执行，点击上方"执行部署"按钮开始' : '# 暂无日志，等待任务开始...'}
                    </span>
                  ) : filteredLogs.length === 0 ? (
                    <span style={{ color: '#666' }}># 无匹配的日志行</span>
                  ) : (
                    filteredLogs.map((line, i) => (
                      <div key={i} style={{ color: getLogColor(line) }}>
                        {line || ' '}
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          </Card>
        )}

        <Modal
          title="补充安装附加组件"
          open={addonModalOpen}
          okText="开始安装"
          cancelText="关闭"
          width={520}
          confirmLoading={addonInstallMutation.isPending}
          okButtonProps={{ disabled: selectedAddons.length === 0 || addonTask?.status === 'running' || addonTask?.status === 'pending' }}
          onOk={() => addonInstallMutation.mutate(selectedAddons)}
          onCancel={() => setAddonModalOpen(false)}
          destroyOnClose
          styles={{ body: { paddingTop: 12, paddingBottom: 16 } }}
        >
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
            <Text type="secondary" style={{ fontSize: 13 }}>选择需要补装的组件</Text>
            <Text type="secondary" style={{ fontSize: 12 }}>已安装组件不可重复选择</Text>
          </div>
          <Checkbox.Group value={selectedAddons} onChange={(values) => setSelectedAddons(values as string[])} style={{ width: '100%' }}>
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              {addonOptions.map((option) => {
                const installed = installedAddons.has(option.value)
                return (
                  <div
                    key={option.value}
                    style={{
                      minHeight: 42,
                      padding: '0 12px',
                      display: 'flex',
                      alignItems: 'center',
                      border: `1px solid ${installed ? '#f0f0f0' : '#e6edf8'}`,
                      borderRadius: 6,
                      background: installed ? '#fafafa' : '#fff',
                    }}
                  >
                    <Checkbox value={option.value} disabled={installed} style={{ flex: 1 }}>
                      <Text strong={!installed} type={installed ? 'secondary' : undefined}>{option.label}</Text>
                    </Checkbox>
                    {installed && <Tag style={{ margin: 0, color: '#8c8c8c', borderColor: '#d9d9d9', background: '#f5f5f5' }}>已安装</Tag>}
                  </div>
                )
              })}
            </Space>
          </Checkbox.Group>
          {selectableAddonCount === 0 && <Text type="secondary" style={{ display: 'block', marginTop: 12, textAlign: 'center' }}>当前集群的附加组件已全部安装</Text>}
        </Modal>

        <Modal
          title={<Space><DesktopOutlined />节点拓扑</Space>}
          open={topologyOpen}
          onCancel={() => setTopologyOpen(false)}
          footer={null}
          width="min(1120px, calc(100vw - 48px))"
          style={{ top: 48 }}
          destroyOnClose
          styles={{ body: { padding: '16px 20px 22px' } }}
        >
          <DeployNodeTopology nodes={nodeDetails} />
        </Modal>

        <Modal
          title={<Space><CodeOutlined />Ansible 执行配置</Space>}
          open={ansibleConfigOpen}
          onCancel={() => setAnsibleConfigOpen(false)}
          footer={null}
          width={860}
          destroyOnClose
        >
          <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
            Inventory 已脱敏，配置由后端根据当前部署方案动态生成。
          </Text>
          <YamlEditor readOnly value={ansibleYaml} height={520} />
        </Modal>

        <Modal
          open={false}
          onCancel={() => setLogViewerOpen(false)}
          footer={null}
          width="62vw"
          style={{ top: 140, marginLeft: 'auto', marginRight: '4vw', paddingBottom: 0 }}
          title={
            <div style={{ color: '#f5f5f5' }}>
            <Space size={10}>
              {selectedSubStepKey
                ? task?.steps?.find((step: DeployTaskStep) => step.key === selectedStepKey)?.subSteps?.find((sub: DeployTaskSubStep) => sub.key === selectedSubStepKey)?.title
                : selectedStepKey
                  ? task?.steps?.find((step: DeployTaskStep) => step.key === selectedStepKey)?.title || stepTitleMap[selectedStepKey] || selectedStepKey
                  : '任务日志'}
              {task?.status === 'running' && <Badge status="processing" text="实时日志" />}
              {task?.steps && <Text style={{ color: '#a6a6a6', fontSize: 12 }}>共 {task.steps.length} 个步骤</Text>}
            </Space>
            </div>
          }
          closeIcon={<CloseCircleOutlined style={{ color: '#d9d9d9' }} />}
          styles={{ header: { background: '#202020', margin: 0, padding: '16px 24px', borderBottom: '1px solid #383838' }, body: { padding: 0 } }}
        >
          {task && (
            <div style={{ height: 'calc(100vh - 340px)', minHeight: 540, display: 'flex', background: '#161616' }}>
              <div style={{ display: 'none' }}>
                {stageGroups.map((stage, stageIndex) => {
                  const steps = stage.steps.map((key) => task.steps?.find((step: DeployTaskStep) => step.key === key)).filter(Boolean) as DeployTaskStep[]
                  const completed = steps.filter((step) => step.status === 'success').length
                  const failed = steps.filter((step) => step.status === 'failed').length
                  return (
                    <div key={stage.key} style={{ marginBottom: 16 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#f5f5f5', marginBottom: 8 }}>
                        <span style={{ width: 24, height: 24, borderRadius: '50%', background: failed ? '#dc2626' : completed === steps.length && steps.length ? '#047857' : '#2563eb', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 700, fontSize: 12 }}>{stageIndex + 1}</span>
                        <span style={{ flex: 1, fontWeight: 600 }}>{stage.title}</span>
                        <span style={{ color: '#a6a6a6', fontSize: 12 }}>{steps.length} 个任务</span>
                      </div>
                      <div style={{ paddingLeft: 10, borderLeft: '1px solid #454545' }}>
                        {steps.map((step, index) => {
                          const selected = selectedStepKey === step.key && !selectedSubStepKey
                          const failedStep = step.status === 'failed'
                          const runningStep = step.status === 'running'
                          const color = failedStep ? '#dc2626' : step.status === 'success' ? '#047857' : runningStep ? '#2563eb' : '#8c8c8c'
                          return (
                            <div key={step.key} style={{ marginBottom: 6 }}>
                              <button
                                type="button"
                                onClick={() => { setSelectedStepKey(step.key); setSelectedSubStepKey(null) }}
                                style={{ width: '100%', border: `1px solid ${selected ? color : '#454545'}`, background: selected ? (failedStep ? '#431a1a' : step.status === 'success' ? '#1f3a22' : '#1f2d3d') : '#262626', color: '#f0f0f0', cursor: 'pointer', borderRadius: 5, padding: '9px 10px', textAlign: 'left', display: 'flex', alignItems: 'center', gap: 7 }}
                              >
                                {failedStep ? <CloseCircleOutlined style={{ color }} /> : step.status === 'success' ? <CheckCircleOutlined style={{ color }} /> : <span style={{ color }}>○</span>}
                                <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{stageIndex + 1}-{index + 1} {step.title}</span>
                                <span style={{ color: '#a6a6a6', fontSize: 11 }}>{step.subSteps?.length || 0}</span>
                              </button>
                              {step.subSteps?.map((sub) => {
                                const selectedSub = selectedSubStepKey === sub.key
                                const subFailed = sub.status === 'failed'
                                const subColor = subFailed ? '#dc2626' : sub.status === 'success' ? '#047857' : sub.status === 'running' ? '#2563eb' : '#8c8c8c'
                                return (
                                  <button
                                    type="button"
                                    key={sub.key}
                                    onClick={() => { setSelectedStepKey(step.key); setSelectedSubStepKey(sub.key) }}
                                    style={{ width: 'calc(100% - 12px)', margin: '5px 0 0 12px', border: `1px solid ${selectedSub ? subColor : '#353535'}`, background: selectedSub ? '#303030' : 'transparent', color: '#c9c9c9', cursor: 'pointer', borderRadius: 4, padding: '7px 8px', textAlign: 'left', display: 'flex', alignItems: 'center', gap: 7 }}
                                  >
                                    {subFailed ? <CloseCircleOutlined style={{ color: subColor, fontSize: 13 }} /> : sub.status === 'success' ? <CheckCircleOutlined style={{ color: subColor, fontSize: 13 }} /> : <span style={{ color: subColor }}>○</span>}
                                    <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{sub.title}</span>
                                  </button>
                                )
                              })}
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  )
                })}
              </div>

              <div style={{ flex: 1, minWidth: 0, padding: 28, display: 'flex', flexDirection: 'column' }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16, marginBottom: 14 }}>
                  <Space>
                    <Text strong style={{ color: '#f5f5f5' }}>日志输出</Text>
                    {sseConnected && <Badge status="processing" text="实时连接" />}
                    {!sseConnected && isRunning && <Badge status="warning" text="连接中" />}
                    <Text style={{ color: '#8c8c8c', fontSize: 12 }}>{filteredLogs.length} / {logs.length} 行</Text>
                  </Space>
                  <Space>
                    <Input
                      size="small"
                      placeholder="过滤日志..."
                      prefix={<SearchOutlined />}
                      value={logFilter}
                      onChange={(event) => setLogFilter(event.target.value)}
                      style={{ width: 220 }}
                      allowClear
                    />
                    <Tooltip title="下载当前日志">
                      <Button size="small" icon={<DownloadOutlined />} onClick={handleDownloadLogs} disabled={logs.length === 0} />
                    </Tooltip>
                  </Space>
                </div>
                <div
                  ref={logContainerRef}
                  style={{ flex: 1, minHeight: 0, overflowY: 'auto', border: '1px solid #343434', borderRadius: 8, padding: 20, background: '#101010', color: '#d4d4d4', fontFamily: 'Consolas, Monaco, "Courier New", monospace', fontSize: 14, lineHeight: 1.7, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}
                >
                  {logs.length === 0 ? (
                    <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#8c8c8c' }}>该任务暂未产生可展示的日志</div>
                  ) : filteredLogs.length === 0 ? (
                    <div style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#8c8c8c' }}>没有匹配的日志内容</div>
                  ) : (
                    filteredLogs.map((line, index) => <div key={index} style={{ color: getLogColor(line) }}>{line || ' '}</div>)
                  )}
                </div>
              </div>
            </div>
          )}
        </Modal>
      </Card>
    </AppPage>
  )
}

// 步骤树组件
type DeployTopologyNode = {
  role: string
  serverName: string
  serverIP: string
  serverOS: string
  serverStatus: string
  serverSSHPort: number
}

type DeployReadinessProps = {
  preflight?: DeployPreflightResult
  failed: boolean
  loading: boolean
  collapsed: boolean
  onToggle: () => void
  onRefresh: () => void
  refreshable: boolean
  onToggleIgnore: (key: string, ignored: boolean) => void
  ignoreLoading: boolean
  assumedReady?: boolean
}

function DeployProgressTitle({ task }: { task?: DeployTask }) {
  const status = task?.status
  const percent = task?.percent ?? 0
  const progressStatus = status === 'failed'
    ? 'exception'
    : status === 'success'
      ? 'success'
      : status === 'running'
        ? 'active'
        : 'normal'

  return (
    <div className="app-deploy-progress-title">
      <span>部署进度</span>
      <Progress
        percent={percent}
        status={progressStatus}
        strokeColor={status === 'canceled' || status === 'cancelled' ? '#b45309' : undefined}
        size="small"
      />
    </div>
  )
}

function DeployReadinessStage({
  preflight,
  failed,
  loading,
  collapsed,
  onToggle,
  onRefresh,
  refreshable,
  onToggleIgnore,
  ignoreLoading,
  assumedReady = false,
}: DeployReadinessProps) {
  const ready = Boolean(preflight?.ready || assumedReady)
  const state = failed || (preflight && !preflight.ready) ? 'failed' : ready ? 'success' : loading ? 'running' : 'pending'
  const passedCount = preflight?.checks.filter((check) => check.status === 'passed').length || 0
  const totalCount = preflight?.checks.length || 0
  const stateLabel = state === 'success'
    ? '检查通过'
    : state === 'failed'
      ? '存在阻断项'
      : state === 'running'
        ? '检查中'
        : '等待检查'

  const style = state === 'failed'
    ? { color: '#dc2626', soft: '#fff7f7', border: '#fecaca' }
    : state === 'success'
      ? { color: '#047857', soft: '#f7fffb', border: '#a7e3c4' }
      : state === 'running'
        ? { color: '#2563eb', soft: '#f6f9ff', border: '#b9cff8' }
        : { color: '#64748b', soft: '#f8fbff', border: '#dbe7f5' }

  return (
    <section style={{ height: '100%', minWidth: 0, flex: 1, border: `1px solid ${style.border}`, borderRadius: 8, background: '#fff', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 3px 12px rgba(0, 0, 0, 0.04)' }}>
      <div style={{ position: 'relative', padding: '12px 44px', background: style.soft, borderBottom: `1px solid ${style.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <span style={{ position: 'absolute', left: 14, top: '50%', width: 25, height: 25, marginTop: -12, borderRadius: '50%', background: style.color, color: '#fff', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 12 }}>1</span>
        <span style={{ minWidth: 0, maxWidth: '100%', textAlign: 'center' }}>
          <span style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 6, color: '#262626', fontWeight: 600 }}>
            {state === 'success' ? <CheckCircleOutlined style={{ color: style.color }} /> : state === 'failed' ? <CloseCircleOutlined style={{ color: style.color }} /> : <SafetyCertificateOutlined style={{ color: style.color }} />}
            部署就绪检查
          </span>
          <span style={{ display: 'block', color: '#8c8c8c', fontSize: 12, marginTop: 3 }}>
            {stateLabel}{totalCount > 0 ? ` · ${passedCount}/${totalCount} 项通过` : ''}
          </span>
        </span>
      </div>

      <div style={{ padding: 10, overflowY: 'auto', background: '#fff', flex: 1 }}>
        <div style={{ overflow: 'hidden', borderRadius: 6, border: `1px solid ${style.border}`, background: '#fff' }}>
          <div style={{ width: '100%', minHeight: 46, padding: '9px 10px', borderBottom: `1px solid ${style.border}`, background: style.soft, display: 'flex', gap: 8, alignItems: 'center', color: style.color }}>
            {state === 'success' ? <CheckCircleOutlined /> : state === 'failed' ? <CloseCircleOutlined /> : <SafetyCertificateOutlined />}
            <span style={{ flex: 1, minWidth: 0, fontWeight: 600 }}>1-1 部署就绪检查</span>
            {totalCount > 0 && (
              <button
                type="button"
                aria-label={collapsed ? '展开就绪检查' : '收起就绪检查'}
                onClick={onToggle}
                style={{ flex: '0 0 auto', width: 26, height: 26, padding: 0, border: `1px solid ${style.border}`, borderRadius: 4, background: '#fff', color: style.color, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}
              >
                <CaretRightOutlined style={{ fontSize: 11, transform: collapsed ? 'rotate(0deg)' : 'rotate(90deg)', transition: 'transform .2s' }} />
              </button>
            )}
          </div>
          <div style={{ minHeight: 38, padding: '7px 10px', display: 'flex', gap: 8, alignItems: 'center', justifyContent: 'space-between', color: '#8c8c8c', fontSize: 12 }}>
            <span>{preflight?.checkedAt ? new Date(preflight.checkedAt).toLocaleString('zh-CN') : stateLabel}</span>
            {refreshable && (
              <Tooltip title="重新检查">
                <Button aria-label="重新检查" type="text" size="small" icon={<ReloadOutlined />} loading={loading} onClick={onRefresh} />
              </Tooltip>
            )}
          </div>
        </div>

        {!collapsed && (
          <div className="app-deploy-readiness__details" style={{ margin: '7px 7px 0', padding: '0 0 0 10px', borderTop: 0, borderLeft: `2px solid ${style.border}` }}>
            {failed ? (
              <div className="app-deploy-readiness__error">无法完成就绪检查，请确认后端服务与部署权限后重试。</div>
            ) : loading && !preflight ? (
              <Spin size="small" />
            ) : preflight?.checks.length ? (
              preflight.checks.map((check) => (
                <div className="app-deploy-readiness__check" key={check.key}>
                  {check.status === 'passed'
                    ? <CheckCircleOutlined className="app-deploy-readiness__check-icon app-deploy-readiness__check-icon--success" />
                  : <CloseCircleOutlined className={`app-deploy-readiness__check-icon app-deploy-readiness__check-icon--${check.status}`} />}
                <div className="app-deploy-readiness__check-content">
                    <Tooltip
                      title={
                        <div>
                          <div>{check.serverName ? `${check.serverName} · ` : ''}{check.message}</div>
                          {check.remediation && <div>处理建议：{check.remediation}</div>}
                        </div>
                      }
                    >
                      <span className="app-deploy-readiness__check-text">{check.serverName ? `${check.serverName} · ` : ''}{check.message}</span>
                    </Tooltip>
                  </div>
                  {check.ignorable && (
                    <Popconfirm title={check.ignored ? '恢复该项强制检查？' : '确认已评估风险并忽略该项？'} onConfirm={() => onToggleIgnore(check.key, !check.ignored)}>
                      <Button type="link" size="small" loading={ignoreLoading}>{check.ignored ? '恢复' : '忽略'}</Button>
                    </Popconfirm>
                  )}
                </div>
              ))
            ) : (
              <Text type="secondary">任务启动前会重新执行完整检查。</Text>
            )}
          </div>
        )}
      </div>
    </section>
  )
}

function DeployNodeCard({ node }: { node: DeployTopologyNode }) {
  const isMaster = node.role === 'master'
  const ports = isMaster
    ? [`${node.serverSSHPort}/SSH`, '6443/API Server', '2379-2380/etcd', '10250/Kubelet']
    : [`${node.serverSSHPort}/SSH`, '10250/Kubelet', '30000-32767/NodePort']

  return (
    <article className={`app-deploy-topology__node app-deploy-topology__node--${isMaster ? 'master' : 'worker'}`}>
      <header>
        <Tag color={roleColorMap[node.role] || 'default'}>{node.role?.toUpperCase()}</Tag>
        <strong>{node.serverName}</strong>
        <Badge status={node.serverStatus === 'available' ? 'success' : 'default'} />
      </header>
      <div className="app-deploy-topology__node-address">{node.serverIP}</div>
      <div className="app-deploy-topology__node-os">{node.serverOS}</div>
      <div className="app-deploy-topology__ports-label">监听端口</div>
      <div className="app-deploy-topology__ports">
        {ports.map((port) => <span key={port}>{port}</span>)}
      </div>
    </article>
  )
}

function DeployNodeTopology({ nodes }: { nodes: DeployTopologyNode[] }) {
  const masters = nodes.filter((node) => node.role === 'master')
  const workers = nodes.filter((node) => node.role !== 'master')
  return (
    <div className="app-deploy-topology">
      <div className="app-deploy-topology__controller">
        <CodeOutlined />
        <div><strong>AIOPS 部署控制器</strong><span>Ansible · SSH 编排</span></div>
      </div>
      <span className="app-deploy-topology__connector" />
      <div className="app-deploy-topology__masters">
        {masters.map((node) => <DeployNodeCard key={`${node.role}-${node.serverIP}`} node={node} />)}
      </div>
      {workers.length > 0 && (
        <>
          <span className="app-deploy-topology__connector" />
          <div className={`app-deploy-topology__workers${workers.length === 1 ? ' app-deploy-topology__workers--single' : ''}`}>
            {workers.map((node) => (
              <div className="app-deploy-topology__worker-branch" key={`${node.role}-${node.serverIP}`}>
                <DeployNodeCard node={node} />
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

function StepTree({
  task,
  supplementTask,
  supplementSelected,
  canRetrySupplement,
  canRetry,
  selectedStepKey,
  selectedSubStepKey,
  onSelectStep,
  onSelectSubStep,
  onSelectSupplementTask,
  onRetrySupplement,
  supplementRetrying,
  onRetryStep,
  retrying,
  readiness,
}: {
  task: DeployTask
  supplementTask?: DeployTask
  supplementSelected: boolean
  canRetrySupplement: boolean
  canRetry: boolean
  selectedStepKey: string | null
  selectedSubStepKey: string | null
  onSelectStep: (key: string) => void
  onSelectSubStep: (stepKey: string, subKey: string) => void
  onSelectSupplementTask: (subStepKey?: string) => void
  onRetrySupplement: () => void
  supplementRetrying: boolean
  onRetryStep: (key: string) => void
  retrying: boolean
  readiness: DeployReadinessProps
}) {
  const stepMap = useMemo(() => {
    const map = new Map<string, DeployTaskStep>()
    task.steps?.forEach((s) => map.set(s.key, s))
    return map
  }, [task.steps])

  // stage 展开/收缩状态：默认收缩全部步骤都已成功的 stage，其余展开
  const [collapsedStages, setCollapsedStages] = useState<Record<string, boolean>>(() => {
    const result: Record<string, boolean> = {}
    for (const stage of stageGroups) {
      const allSuccess = stage.steps.every((key) => {
        const step = stepMap.get(key)
        return step && step.status === 'success'
      })
      if (allSuccess) result[stage.key] = true
    }
    return result
  })
  const toggleStage = (key: string) => {
    setCollapsedStages((prev) => ({ ...prev, [key]: !prev[key] }))
  }
  const [expandedSteps, setExpandedSteps] = useState<Record<string, boolean>>({})
  const [expandedSupplementTasks, setExpandedSupplementTasks] = useState<Record<number, boolean>>({})
  const toggleStep = (stepKey: string, defaultExpanded: boolean) => {
    setExpandedSteps((prev) => ({ ...prev, [stepKey]: !(prev[stepKey] ?? defaultExpanded) }))
  }
  const toggleSupplementTask = (taskID: number, defaultExpanded: boolean) => {
    setExpandedSupplementTasks((prev) => ({ ...prev, [taskID]: !(prev[taskID] ?? defaultExpanded) }))
  }

  // 格式化步骤耗时
  const formatDuration = (startedAt?: string, finishedAt?: string): string | null => {
    if (!startedAt || !finishedAt) return null
    const start = new Date(startedAt).getTime()
    const end = new Date(finishedAt).getTime()
    const diff = end - start
    if (diff <= 0) return null
    if (diff < 1000) return `${diff}ms`
    if (diff < 60000) return `${(diff / 1000).toFixed(1)}s`
    const min = Math.floor(diff / 60000)
    const sec = Math.round((diff % 60000) / 1000)
    return `${min}m${sec}s`
  }

  const statusStyle = (status: string) => {
    if (status === 'failed') return { color: '#dc2626', soft: '#fff7f7', border: '#fecaca', label: '失败' }
    if (status === 'canceled' || status === 'cancelled') return { color: '#b45309', soft: '#fffaf0', border: '#fed7aa', label: '已取消' }
    if (status === 'blocked') return { color: '#64748b', soft: '#f7faff', border: '#cbdcf4', label: '已阻断' }
    if (status === 'success') return { color: '#047857', soft: '#f7fffb', border: '#a7e3c4', label: '成功' }
    if (status === 'running') return { color: '#2563eb', soft: '#f6f9ff', border: '#b9cff8', label: '执行中' }
    return { color: '#64748b', soft: '#f8fbff', border: '#dbe7f5', label: '等待中' }
  }

  const statusIcon = (status: string, size = 15) => {
    const style = { color: statusStyle(status).color, fontSize: size }
    if (status === 'success') return <CheckCircleOutlined style={style} />
    if (status === 'failed') return <CloseCircleOutlined style={style} />
    if (status === 'canceled' || status === 'cancelled') return <StopOutlined style={style} />
    if (status === 'blocked') return <StopOutlined style={style} />
    if (status === 'running') return <Badge status="processing" />
    return <span style={{ ...style, fontWeight: 700 }}>○</span>
  }

  const taskCanceled = task.status === 'canceled' || task.status === 'cancelled'
  const firstFailedStepIndex = deploymentStepKeys.findIndex((key) => stepMap.get(key)?.status === 'failed')
  const resolveStepStatus = (stepKey: string, stepStatus: string) => {
    if (taskCanceled && (stepStatus === 'pending' || stepStatus === 'running')) return 'canceled'
    const stepIndex = deploymentStepKeys.indexOf(stepKey)
    if (task.status === 'failed' && firstFailedStepIndex >= 0 && stepIndex > firstFailedStepIndex && stepStatus === 'pending') return 'blocked'
    return stepStatus
  }
  const resolveSubStepStatus = (parentStatus: string, subStepStatus: string) => {
    if (taskCanceled && (subStepStatus === 'pending' || subStepStatus === 'running')) return 'canceled'
    if (parentStatus === 'blocked' && subStepStatus === 'pending') return 'blocked'
    if (subStepStatus === 'running' && (parentStatus === 'success' || parentStatus === 'failed')) {
      return parentStatus
    }
    return subStepStatus
  }

  return (
    <div style={{ height: '100%', overflowX: 'auto', overflowY: 'hidden', padding: '2px 2px 12px' }}>
      <style>{`@keyframes deploy-step-pulse { 0% { box-shadow: 0 0 0 0 rgba(22, 119, 255, .45); } 70% { box-shadow: 0 0 0 7px rgba(22, 119, 255, 0); } 100% { box-shadow: 0 0 0 0 rgba(22, 119, 255, 0); } }`}</style>
      <div style={{ display: 'flex', alignItems: 'stretch', gap: 0, minWidth: 1220, height: '100%' }}>
        <div style={{ display: 'flex', alignItems: 'stretch', minWidth: 300, flex: '1 1 0' }}>
          <DeployReadinessStage {...readiness} />
          <div aria-hidden="true" style={{ position: 'relative', flex: '0 0 46px', width: 46 }}>
            <span style={{ position: 'absolute', top: '50%', left: 8, right: 13, height: 2, marginTop: -1, borderRadius: 2, background: '#aebfd1' }} />
            <span style={{ position: 'absolute', top: '50%', right: 7, width: 9, height: 9, marginTop: -5, borderTop: '2px solid #7f96ad', borderRight: '2px solid #7f96ad', transform: 'rotate(45deg)' }} />
          </div>
        </div>
        {stageGroups.filter((stage) => stage.key !== 'supplement' || !!supplementTask).map((stage, stageIdx, visibleStages) => {
          const steps = stage.steps.map((key) => stepMap.get(key)).filter(Boolean) as DeployTaskStep[]
          const supplementaryTask = stage.key === 'supplement' ? supplementTask : undefined
          const supplementaryStatus = supplementaryTask?.status || 'pending'
          const completed = steps.filter((step) => step.status === 'success').length
          const completedCount = completed + (supplementaryStatus === 'success' ? 1 : 0)
          const failed = steps.filter((step) => step.status === 'failed').length + (supplementaryStatus === 'failed' ? 1 : 0)
          const stepStatuses = steps.map((step) => resolveStepStatus(step.key, step.status))
          const canceled = stepStatuses.filter((status) => status === 'canceled').length + (supplementaryStatus === 'canceled' ? 1 : 0)
          const blocked = stepStatuses.filter((status) => status === 'blocked').length
          const running = stepStatuses.filter((status) => status === 'running').length + (supplementaryStatus === 'running' ? 1 : 0)
          const stageTaskCount = steps.length + (supplementaryTask ? 1 : 0)
          const aggregateStatus = failed > 0 ? 'failed' : canceled > 0 ? 'canceled' : running > 0 ? 'running' : blocked > 0 ? 'blocked' : completedCount === stageTaskCount && stageTaskCount > 0 ? 'success' : 'pending'
          const stageStyle = statusStyle(aggregateStatus)
          return (
            <div key={stage.key} style={{ display: 'flex', alignItems: 'stretch', minWidth: 300, flex: '1 1 0' }}>
              <div style={{ height: '100%', minWidth: 0, flex: 1, border: `1px solid ${stageStyle.border}`, borderRadius: 8, background: '#fff', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 3px 12px rgba(0, 0, 0, 0.04)' }}>
                <div
                  style={{ position: 'relative', padding: '12px 44px', background: stageStyle.soft, borderBottom: `1px solid ${stageStyle.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
                >
                  <span style={{ position: 'absolute', left: 14, top: '50%', width: 25, height: 25, marginTop: -12, borderRadius: '50%', background: stageStyle.color, color: '#fff', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 12 }}>{stageIdx + 2}</span>
                  <span style={{ minWidth: 0, maxWidth: '100%', textAlign: 'center' }}>
                    <span style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 6, color: '#262626', fontWeight: 600 }}>{statusIcon(aggregateStatus)} {stage.title}</span>
                    <span style={{ display: 'block', color: '#8c8c8c', fontSize: 12, marginTop: 3 }}>任务 {stageTaskCount} · 成功 {completedCount}{failed ? ` · 失败 ${failed}` : ''}{canceled ? ` · 已取消 ${canceled}` : ''}{blocked ? ` · 已阻断 ${blocked}` : ''}</span>
                  </span>
                </div>

                <div style={{ padding: 10, overflowY: 'auto', background: '#fff', flex: 1 }}>
                    {steps.map((step, index) => {
                      const isSelected = selectedStepKey === step.key && !selectedSubStepKey
                      const displayStepStatus = resolveStepStatus(step.key, step.status)
                      const itemStyle = statusStyle(displayStepStatus)
                      const subSteps = step.subSteps || []
                      const subCompleted = subSteps.filter((sub) => resolveSubStepStatus(displayStepStatus, sub.status) === 'success').length
                      const defaultExpanded = displayStepStatus !== 'success' || isSelected || subSteps.some((sub) => sub.key === selectedSubStepKey)
                      const expanded = subSteps.length > 0 && (expandedSteps[step.key] ?? defaultExpanded)
                      return (
                        <div key={step.key} style={{ marginBottom: 10 }}>
                          <div
                            role="button"
                            tabIndex={0}
                            onClick={() => onSelectStep(step.key)}
                            onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') onSelectStep(step.key) }}
                            style={{ cursor: 'pointer', overflow: 'hidden', borderRadius: 6, border: `1px solid ${itemStyle.border}`, background: '#fff', boxShadow: isSelected ? `0 0 0 2px ${itemStyle.color}` : 'none' }}
                          >
                            <div style={{ background: itemStyle.soft, borderBottom: `1px solid ${itemStyle.border}`, padding: '9px 10px', display: 'flex', gap: 8, alignItems: 'center' }}>
                              {statusIcon(displayStepStatus)}
                              <span style={{ flex: 1, minWidth: 0, color: itemStyle.color, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{stageIdx + 2}-{index + 1} {stepTitleMap[step.key] || step.title}</span>
                              {step.status === 'failed' && canRetry && (
                                <Button
                                  size="small"
                                  type="primary"
                                  danger
                                  icon={<RedoOutlined />}
                                  loading={retrying}
                                  onClick={(event) => { event.stopPropagation(); onRetryStep(step.key) }}
                                >
                                  重试
                                </Button>
                              )}
                              {subSteps.length > 0 && (
                                <button
                                  type="button"
                                  aria-label={expanded ? '收起任务步骤' : '展开任务步骤'}
                                  title={expanded ? '收起任务步骤' : '展开任务步骤'}
                                  onClick={(event) => { event.stopPropagation(); toggleStep(step.key, defaultExpanded) }}
                                  onKeyDown={(event) => event.stopPropagation()}
                                  style={{ flex: '0 0 auto', width: 26, height: 26, padding: 0, border: `1px solid ${itemStyle.border}`, borderRadius: 4, background: '#fff', color: itemStyle.color, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}
                                >
                                  <CaretRightOutlined style={{ fontSize: 11, transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)', transition: 'transform .2s' }} />
                                </button>
                              )}
                            </div>
                            <div style={{ padding: '7px 10px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', color: '#8c8c8c', fontSize: 12 }}>
                              <Tooltip title={displayStepStatus === 'failed' ? step.message : undefined}>
                                <span style={{ minWidth: 0, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: displayStepStatus === 'failed' ? itemStyle.color : undefined }}>
                                  {displayStepStatus === 'failed' && step.message
                                    ? step.message
                                    : subSteps.length ? `任务 ${subSteps.length} · 已完成 ${subCompleted}` : itemStyle.label}
                                </span>
                              </Tooltip>
                              {displayStepStatus === 'running' ? (
                                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5, color: '#2563eb', fontWeight: 600 }}>
                                  <span style={{ width: 7, height: 7, borderRadius: '50%', background: '#2563eb', boxShadow: '0 0 0 0 rgba(22, 119, 255, .45)', animation: 'deploy-step-pulse 1.4s ease-out infinite' }} />
                                  进行中
                                </span>
                              ) : displayStepStatus === 'failed' ? (
                                <Button
                                  type="link"
                                  size="small"
                                  icon={<CodeOutlined />}
                                  onClick={(event) => { event.stopPropagation(); onSelectStep(step.key) }}
                                  style={{ flex: '0 0 auto', height: 22, padding: '0 2px', color: itemStyle.color }}
                                >
                                  日志
                                </Button>
                              ) : (
                                <span style={{ flex: '0 0 auto' }}>{formatDuration(step.startedAt, step.finishedAt) || (displayStepStatus === 'canceled' ? itemStyle.label : '')}</span>
                              )}
                            </div>
                          </div>

                          {expanded && subSteps.length > 0 && (
                            <div style={{ margin: '7px 7px 0', paddingLeft: 10, borderLeft: `2px solid ${itemStyle.border}` }}>
                              {subSteps.map((sub) => {
                                const subSelected = selectedSubStepKey === sub.key
                                const subStatus = resolveSubStepStatus(displayStepStatus, sub.status)
                                const subStyle = statusStyle(subStatus)
                                return (
                                  <div
                                    key={sub.key}
                                    role="button"
                                    tabIndex={0}
                                    onClick={() => onSelectSubStep(step.key, sub.key)}
                                    onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') onSelectSubStep(step.key, sub.key) }}
                                    style={{ cursor: 'pointer', marginTop: 6, padding: '7px 8px', borderRadius: 4, border: `1px solid ${subSelected ? subStyle.color : '#e8e8e8'}`, background: subSelected ? subStyle.soft : '#fff', display: 'flex', alignItems: 'center', gap: 6 }}
                                  >
                                    {statusIcon(subStatus, 13)}
                                    <span style={{ color: '#595959', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{sub.title}</span>
                                    <span style={{ color: '#8c8c8c', fontSize: 11 }}>{formatDuration(sub.startedAt, sub.finishedAt) || ''}</span>
                                  </div>
                                )
                              })}
                            </div>
                          )}
                        </div>
                      )
                    })}
                    {supplementaryTask && (() => {
                      const itemStyle = statusStyle(supplementaryStatus)
                      const addonNames = Array.isArray(supplementaryTask.meta?.addons) ? supplementaryTask.meta.addons.join('、') : ''
                      const selectedAddonKeys = Array.isArray(supplementaryTask.meta?.addons) ? supplementaryTask.meta.addons : []
                      const helmOnly = selectedAddonKeys.length === 1 && selectedAddonKeys[0] === 'helm'
                      const supplementTitle = helmOnly ? '补充安装 Helm' : '补充安装组件'
                      const supplementStep = supplementaryTask.steps?.find((step) => step.key === 'install_addons') || supplementaryTask.steps?.[0]
                      const persistedSupplementSubSteps = supplementStep?.subSteps || []
                      // 旧版本解析器会把 pre_check 等未选 PLAY 的子任务挂到补装
                      // 步骤下；这类历史数据不能继续被展示为“补装成功”。
                      const hasForeignLegacySubSteps = persistedSupplementSubSteps.some((sub) => /^(pre_check|bootstrap|container_runtime|kubeadm_init|join_workers|install_cni|register)\s*:/.test(sub.title))
                      const actualSupplementSubSteps = hasForeignLegacySubSteps
                        ? []
                        : helmOnly
                          ? persistedSupplementSubSteps.filter((sub) => sub.title === 'Gathering Facts' || sub.title.startsWith('install_helm :'))
                          : persistedSupplementSubSteps
                      // 兼容早期补装任务：它们没有持久化 Ansible 子步骤，也必须保持标准步骤卡的展开结构。
                      const supplementSubSteps = actualSupplementSubSteps.length > 0
                        ? actualSupplementSubSteps
                        : [{ key: `addon-task-${supplementaryTask.id}`, title: helmOnly ? '执行 Helm 安装' : '执行附加组件安装', status: supplementaryStatus }]
                      const supplementCompleted = supplementSubSteps.filter((sub) => sub.status === 'success').length
                      const supplementDefaultExpanded = supplementaryStatus !== 'success' || supplementSelected || supplementSubSteps.some((sub) => sub.key === selectedSubStepKey)
                      const supplementExpanded = supplementSubSteps.length > 0 && (expandedSupplementTasks[supplementaryTask.id] ?? supplementDefaultExpanded)
                      const supplementMessage = supplementStep?.message || supplementaryTask.message
                      return (
                        <div style={{ marginBottom: 10 }}>
                          <div
                            role="button"
                            tabIndex={0}
                            onClick={() => onSelectSupplementTask()}
                            onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') onSelectSupplementTask() }}
                            style={{ cursor: 'pointer', overflow: 'hidden', borderRadius: 6, border: `1px solid ${itemStyle.border}`, background: '#fff', boxShadow: supplementSelected ? `0 0 0 2px ${itemStyle.color}` : 'none' }}
                          >
                            <div style={{ background: itemStyle.soft, borderBottom: `1px solid ${itemStyle.border}`, padding: '9px 10px', display: 'flex', gap: 8, alignItems: 'center' }}>
                              {statusIcon(supplementaryStatus)}
                              <span style={{ flex: 1, minWidth: 0, color: itemStyle.color, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{stageIdx + 2}-{steps.length + 1} {supplementTitle}</span>
                              {supplementaryStatus === 'failed' && canRetrySupplement && (
                                <Button
                                  size="small"
                                  type="primary"
                                  danger
                                  icon={<RedoOutlined />}
                                  loading={supplementRetrying}
                                  onClick={(event) => { event.stopPropagation(); onRetrySupplement() }}
                                >
                                  重试
                                </Button>
                              )}
                              <button
                                type="button"
                                aria-label={supplementExpanded ? '收起任务步骤' : '展开任务步骤'}
                                title={supplementExpanded ? '收起任务步骤' : '展开任务步骤'}
                                onClick={(event) => { event.stopPropagation(); toggleSupplementTask(supplementaryTask.id, supplementDefaultExpanded) }}
                                onKeyDown={(event) => event.stopPropagation()}
                                style={{ flex: '0 0 auto', width: 26, height: 26, padding: 0, border: `1px solid ${itemStyle.border}`, borderRadius: 4, background: '#fff', color: itemStyle.color, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}
                              >
                                <CaretRightOutlined style={{ fontSize: 11, transform: supplementExpanded ? 'rotate(90deg)' : 'rotate(0deg)', transition: 'transform .2s' }} />
                              </button>
                            </div>
                            <div style={{ padding: '7px 10px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', color: '#8c8c8c', fontSize: 12, gap: 8 }}>
                              <Tooltip title={supplementaryStatus === 'failed' ? supplementMessage : undefined}>
                                <span style={{ minWidth: 0, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: supplementaryStatus === 'failed' ? itemStyle.color : undefined }}>
                                  {supplementaryStatus === 'failed' && supplementMessage
                                    ? supplementMessage
                                    : supplementSubSteps.length ? `任务 ${supplementSubSteps.length} · 已完成 ${supplementCompleted}` : `任务 #${supplementaryTask.id}${addonNames ? ` · ${addonNames}` : ''}`}
                                </span>
                              </Tooltip>
                              {supplementaryStatus === 'running' ? (
                                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5, color: '#2563eb', fontWeight: 600 }}>
                                  <span style={{ width: 7, height: 7, borderRadius: '50%', background: '#2563eb', boxShadow: '0 0 0 0 rgba(22, 119, 255, .45)', animation: 'deploy-step-pulse 1.4s ease-out infinite' }} />
                                  进行中
                                </span>
                              ) : (
                                <span style={{ flex: '0 0 auto' }}>{itemStyle.label}</span>
                              )}
                            </div>
                          </div>
                          {supplementExpanded && (
                            <div style={{ margin: '7px 7px 0', paddingLeft: 10, borderLeft: `2px solid ${itemStyle.border}` }}>
                              {supplementSubSteps.map((sub) => {
                                const subSelected = selectedSubStepKey === sub.key
                                const subStyle = statusStyle(sub.status)
                                return (
                                  <div
                                    key={sub.key}
                                    role="button"
                                    tabIndex={0}
                                    onClick={() => onSelectSupplementTask(actualSupplementSubSteps.some((actual) => actual.key === sub.key) ? sub.key : undefined)}
                                    onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') onSelectSupplementTask(actualSupplementSubSteps.some((actual) => actual.key === sub.key) ? sub.key : undefined) }}
                                    style={{ cursor: 'pointer', marginTop: 6, padding: '7px 8px', borderRadius: 4, border: `1px solid ${subSelected ? subStyle.color : '#e8e8e8'}`, background: subSelected ? subStyle.soft : '#fff', display: 'flex', alignItems: 'center', gap: 6 }}
                                  >
                                    {statusIcon(sub.status, 13)}
                                    <span style={{ color: '#595959', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{sub.title}</span>
                                    <span style={{ color: '#8c8c8c', fontSize: 11 }}>{formatDuration(sub.startedAt, sub.finishedAt) || ''}</span>
                                  </div>
                                )
                              })}
                            </div>
                          )}
                        </div>
                      )
                    })()}
                </div>
              </div>
              {stageIdx < visibleStages.length - 1 && (
                <div aria-hidden="true" style={{ position: 'relative', flex: '0 0 46px', width: 46 }}>
                  <span style={{ position: 'absolute', top: '50%', left: 8, right: 13, height: 2, marginTop: -1, borderRadius: 2, background: '#aebfd1' }} />
                  <span style={{ position: 'absolute', top: '50%', right: 7, width: 9, height: 9, marginTop: -5, borderTop: '2px solid #7f96ad', borderRight: '2px solid #7f96ad', transform: 'rotate(45deg)' }} />
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {stageGroups.map((stage, stageIdx) => {
        const collapsed = !!collapsedStages[stage.key]
        return (
        <div key={stage.key}>
          <div
            onClick={() => toggleStage(stage.key)}
            style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, cursor: 'pointer', userSelect: 'none' }}
          >
            <CaretRightOutlined style={{ fontSize: 12, color: '#8c8c8c', transition: 'transform 0.2s', transform: collapsed ? 'rotate(0deg)' : 'rotate(90deg)' }} />
            <div style={{
              width: 24,
              height: 24,
              borderRadius: '50%',
              background: '#2563eb',
              color: '#fff',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 12,
              fontWeight: 'bold',
            }}>
              {stageIdx + 2}
            </div>
            <Text strong style={{ fontSize: 14 }}>{stage.title}</Text>
          </div>
          {!collapsed && (
          <div style={{ paddingLeft: 12, borderLeft: '2px solid #e6e6e6', marginLeft: 11 }}>
            {stage.steps.map((stepKey, idx) => {
              const step = stepMap.get(stepKey)
              if (!step) return null
              const isSelected = selectedStepKey === stepKey && !selectedSubStepKey
              const stepFailed = step.status === 'failed'
              const stepSuccess = step.status === 'success'
              const stepRunning = step.status === 'running'
              return (
                <div key={stepKey} style={{ marginBottom: 8 }}>
                  <div
                    onClick={() => onSelectStep(stepKey)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      padding: '8px 12px',
                      borderRadius: 6,
                      background: isSelected ? '#eff6ff' : stepSuccess ? '#ecfdf5' : '#fafafa',
                      border: `1px solid ${isSelected ? '#91caff' : stepSuccess ? '#b7eb8f' : '#f0f0f0'}`,
                      cursor: 'pointer',
                      transition: 'all 0.2s',
                    }}
                  >
                    <Space>
                      <span style={{ color: stepSuccess ? '#047857' : stepFailed ? '#dc2626' : stepRunning ? '#2563eb' : '#999', fontWeight: 500 }}>
                        {stageIdx + 2}-{idx + 1} {step.title}
                      </span>
                      {stepRunning && <Badge status="processing" />}
                      {formatDuration(step.startedAt, step.finishedAt) && (
                        <span style={{ fontSize: 11, color: '#8c8c8c' }}>
                          {formatDuration(step.startedAt, step.finishedAt)}
                        </span>
                      )}
                    </Space>
                    <Space>
                      {stepSuccess && <CheckCircleOutlined style={{ color: '#047857' }} />}
                      {stepFailed && <CloseCircleOutlined style={{ color: '#dc2626' }} />}
                      {stepFailed && canRetry && (
                        <Button
                          size="small"
                          type="primary"
                          danger
                          icon={<RedoOutlined />}
                          loading={retrying}
                          onClick={(e) => {
                            e.stopPropagation()
                            onRetryStep(stepKey)
                          }}
                        >
                          重试
                        </Button>
                      )}
                    </Space>
                  </div>
                  {step.subSteps && step.subSteps.length > 0 && (
                    <div style={{ paddingLeft: 12, marginTop: 4 }}>
                      {step.subSteps.map((sub) => {
                        const subSelected = selectedSubStepKey === sub.key
                        const subSuccess = sub.status === 'success'
                        const subFailed = sub.status === 'failed'
                        const subRunning = sub.status === 'running'
                        return (
                          <div
                            key={sub.key}
                            onClick={() => onSelectSubStep(stepKey, sub.key)}
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 8,
                              padding: '6px 10px',
                              marginBottom: 4,
                              borderRadius: 4,
                              background: subSelected ? '#eff6ff' : '#fff',
                              cursor: 'pointer',
                            }}
                          >
                            <CodeOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
                            <Text style={{ fontSize: 13, flex: 1, color: subSuccess ? '#047857' : subFailed ? '#dc2626' : subRunning ? '#2563eb' : '#595959' }}>
                              {sub.title}
                            </Text>
                            {subSuccess && <CheckCircleOutlined style={{ color: '#047857', fontSize: 12 }} />}
                            {subFailed && <CloseCircleOutlined style={{ color: '#dc2626', fontSize: 12 }} />}
                            {subRunning && <Badge status="processing" />}
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
          )}
        </div>
        )
      })}
    </Space>
  )
}

/** 根据 Ansible 输出格式着色 */
function getLogColor(line: string, theme: 'light' | 'dark' = 'dark'): string {
  const trimmed = line.trim()
  const dark = theme === 'dark'
  if (trimmed.startsWith('PLAY [') || trimmed.startsWith('PLAY RECAP')) return dark ? '#569cd6' : '#0958d9'
  if (trimmed.startsWith('TASK [')) return dark ? '#c586c0' : '#6d5bd0'
  if (trimmed.includes('ok:') && !trimmed.includes('failed')) return dark ? '#4ec9b0' : '#2563eb'
  if (trimmed.startsWith('changed:') || trimmed.includes('changed=1')) return dark ? '#cca700' : '#b45309'
  if (trimmed.includes('failed:') || trimmed.includes('FAILED') || trimmed.includes('fatal:')) return dark ? '#f44747' : '#dc2626'
  if (trimmed.startsWith('skipping:') || trimmed.includes('skipped')) return dark ? '#808080' : '#8c8c8c'
  if (line.includes('[error]') || line.includes('[ERROR]')) return dark ? '#f44747' : '#dc2626'
  if (line.includes('[warn]') || line.includes('[WARN]')) return dark ? '#cca700' : '#b45309'
  if (line.includes('[info]') || line.includes('[INFO]')) return dark ? '#4ec9b0' : '#2563eb'
  return dark ? '#d4d4d4' : '#262626'
}

function formatLogTimestamp(timestamp?: string): string {
  if (!timestamp) return '--:--:--'
  const time = new Date(timestamp)
  if (Number.isNaN(time.getTime())) return '--:--:--'
  const pad = (value: number, length = 2) => String(value).padStart(length, '0')
  return `${pad(time.getHours())}:${pad(time.getMinutes())}:${pad(time.getSeconds())}.${pad(time.getMilliseconds(), 3)}`
}
