/**
 * 部署计划详情页
 * 展示计划信息、节点拓扑、分阶段步骤树、SSE 实时日志
 */
import { useEffect, useRef, useState, useCallback, useMemo } from 'react'
import { Card, Descriptions, Tag, Badge, Button, Space, Typography, message, Popconfirm, Tooltip, Progress, Empty, Input, Collapse, Alert, Spin, Divider } from 'antd'
import { ArrowLeftOutlined, StopOutlined, RedoOutlined, DownloadOutlined, PlayCircleOutlined, SearchOutlined, ReloadOutlined, DesktopOutlined, SafetyCertificateOutlined, CheckCircleOutlined, CloseCircleOutlined, CaretRightOutlined, CodeOutlined } from '@ant-design/icons'
import { history, useParams } from '@umijs/max'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AppPage, YamlEditor } from '@/components'
import {
  getDeployPlanById,
  getDeployTask,
  getDeployTaskLogs,
  getDeployTaskLogSSEUrl,
  cancelDeployPlan,
  retryDeployPlan,
  retryDeployStep,
  executeDeployPlan,
  getServers,
  getPlanAnsibleConfig,
  preflightDeployPlan,
  setDeployPreflightIgnore,
} from '@/services/deploy'
import type { DeployTask, DeployTaskStep, DeployTaskSubStep } from '@/types'

const { Text, Title } = Typography

const statusMap: Record<string, { badge: string; text: string }> = {
  draft: { badge: 'default', text: '草稿' },
  running: { badge: 'processing', text: '执行中' },
  success: { badge: 'success', text: '成功' },
  failed: { badge: 'error', text: '失败' },
  cancelled: { badge: 'warning', text: '已取消' },
}

const taskStatusMap: Record<string, { badge: string; text: string }> = {
  pending: { badge: 'default', text: '等待中' },
  running: { badge: 'processing', text: '执行中' },
  success: { badge: 'success', text: '成功' },
  failed: { badge: 'error', text: '失败' },
  canceled: { badge: 'warning', text: '已取消' },
  timeout: { badge: 'error', text: '超时' },
}

const roleColorMap: Record<string, string> = {
  master: 'red',
  worker: 'blue',
}

// 步骤分组（stage -> steps）
const stageGroups = [
  { key: 'init', title: '初始化', steps: ['pre_check', 'bootstrap'] },
  { key: 'install', title: '安装 k8s 集群', steps: ['container_runtime', 'kubeadm_init', 'join_workers', 'install_cni'] },
  { key: 'extend', title: '补充扩展', steps: ['install_addons', 'register'] },
]

// 步骤标题映射
const stepTitleMap: Record<string, string> = {
  pre_check: '环境预检',
  bootstrap: '基础环境初始化',
  container_runtime: '容器运行时安装',
  kubeadm_init: 'Kubernetes Master 初始化',
  join_workers: 'Worker 节点加入集群',
  install_cni: '安装 CNI 网络插件',
  install_addons: '安装 Kubernetes 扩展组件',
  register: '节点注册到管理平台',
}

export default function DeployPlanDetailPage() {
  const params = useParams<{ id: string }>()
  const planId = Number(params.id)
  const queryClient = useQueryClient()
  const [selectedStepKey, setSelectedStepKey] = useState<string | null>(null)
  const [selectedSubStepKey, setSelectedSubStepKey] = useState<string | null>(null)
  const [logs, setLogs] = useState<string[]>([])
  const [sseConnected, setSseConnected] = useState(false)
  const [logFilter, setLogFilter] = useState('')
  const logOffsetRef = useRef(0)
  const logContainerRef = useRef<HTMLDivElement>(null)
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
    enabled: !!plan && ['draft', 'failed', 'cancelled'].includes(plan.status),
    refetchOnWindowFocus: false,
    retry: false,
  })

  const taskId = plan?.taskId

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
      }
    })
  }, [plan?.nodes, serversData])

  // 执行部署
  const executeMutation = useMutation({
    mutationFn: () => executeDeployPlan(planId),
    onSuccess: () => {
      message.success('部署已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
    },
    onError: (err: any) => message.error(err?.message || '执行失败'),
  })

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

  // 重试部署
  const retryMutation = useMutation({
    mutationFn: () => retryDeployPlan(planId),
    onSuccess: () => {
      message.success('重试已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
    },
    onError: () => message.error('重试失败'),
  })

  // 重试步骤
  const retryStepMutation = useMutation({
    mutationFn: ({ stepKey }: { stepKey: string }) => retryDeployStep(planId, stepKey),
    onSuccess: () => {
      message.success('步骤重试已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
    },
    onError: () => message.error('步骤重试失败'),
  })

  // 当前选中的 step key（用于日志过滤）
  const effectiveStepKey = useMemo(() => {
    if (selectedSubStepKey) {
      // sub step key 格式为 {stepKey}-{index}，提取大 step key
      return selectedSubStepKey.split('-')[0]
    }
    return selectedStepKey
  }, [selectedStepKey, selectedSubStepKey])

  // 初始加载历史日志
  useEffect(() => {
    if (!taskId) {
      setLogs([])
      logOffsetRef.current = 0
      return
    }
    // 默认选中第一个非成功的步骤，或第一个步骤
    const steps = task?.steps || []
    if (!selectedStepKey && steps.length > 0) {
      const active = steps.find((s: DeployTaskStep) => s.status === 'running' || s.status === 'failed')
      setSelectedStepKey(active?.key || steps[0].key)
    }

    setLogs([])
    logOffsetRef.current = 0
    getDeployTaskLogs(taskId, 0, 500, effectiveStepKey || undefined).then((res) => {
      setLogs(res.logs || [])
      logOffsetRef.current = (res.logs || []).length
    }).catch(() => {
      setLogs([])
      logOffsetRef.current = 0
    })
  }, [taskId, effectiveStepKey, task?.steps])

  // SSE 实时日志连接
  const connectSSE = useCallback(() => {
    if (!taskId) return
    eventSourceRef.current?.close()
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }

    const token = localStorage.getItem('token') || ''
    const url = getDeployTaskLogSSEUrl(taskId, effectiveStepKey || undefined)
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
        }
      } catch {
        setLogs((prev) => [...prev, event.data])
      }
    }

    es.addEventListener('done', (event: any) => {
      try {
        const data = JSON.parse(event.data)
        message.info(`任务已结束：${data.status || ''}`)
      } catch { /* ignore */ }
      es.close()
      setSseConnected(false)
      if (taskId) {
        queryClient.invalidateQueries({ queryKey: ['deploy-task', taskId] })
        queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
      }
    })

    es.onerror = () => {
      setSseConnected(false)
      es.close()
      const currentStatus = task?.status
      if (currentStatus === 'running' || currentStatus === 'pending') {
        reconnectTimerRef.current = setTimeout(() => connectSSE(), 3000)
      }
    }
  }, [taskId, planId, queryClient, task?.status, effectiveStepKey])

  useEffect(() => {
    if (!taskId) return
    const taskStatus = task?.status
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
  }, [taskId, task?.status, connectSSE])

  // 自动滚动到底部
  useEffect(() => {
    if (logContainerRef.current && !logFilter) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }, [logs, logFilter])

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
  const canRetry = planStatus === 'failed' || planStatus === 'cancelled'

  const currentTaskStatus = task?.status
  const taskBadge = taskStatusMap[currentTaskStatus || ''] || { badge: 'default', text: currentTaskStatus || '-' }

  // 根据选中 sub step 进一步过滤日志（客户端过滤）
  const filteredLogs = useMemo(() => {
    let list = logFilter ? logs.filter((line) => line.toLowerCase().includes(logFilter.toLowerCase())) : logs
    if (selectedSubStepKey && task) {
      const step = task.steps?.find((s: DeployTaskStep) => s.key === selectedStepKey)
      const sub = step?.subSteps?.find((s: DeployTaskSubStep) => s.key === selectedSubStepKey)
      if (sub) {
        // 高亮子步骤：只保留包含子步骤标题的 TASK 行及其后直到下一个 TASK 行
        const subTitle = sub.title
        const result: string[] = []
        let inSubStep = false
        for (const line of list) {
          const trimmed = line.trim()
          if (trimmed.startsWith('TASK [')) {
            inSubStep = trimmed.includes(subTitle)
          }
          if (inSubStep || trimmed.includes(subTitle)) {
            result.push(line)
          }
        }
        // 如果没有匹配到 TASK 行（可能日志格式不同），返回整个 step 日志
        return result.length > 0 ? result : list
      }
    }
    return list
  }, [logs, logFilter, selectedSubStepKey, selectedStepKey, task])

  // 步骤选择处理
  const handleSelectStep = (stepKey: string) => {
    setSelectedStepKey(stepKey)
    setSelectedSubStepKey(null)
  }

  const handleSelectSubStep = (stepKey: string, subKey: string) => {
    setSelectedStepKey(stepKey)
    setSelectedSubStepKey(subKey)
  }

  return (
    <AppPage>
      <Card loading={planLoading}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => history.push('/deploy/plans')}>返回列表</Button>
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
              <Popconfirm title="确认重试该部署方案？" onConfirm={() => retryMutation.mutate()}>
                <Button type="primary" icon={<RedoOutlined />} disabled={!preflight?.ready} loading={retryMutation.isPending}>重试部署</Button>
              </Popconfirm>
            )}
            <Tooltip title="刷新">
              <Button icon={<ReloadOutlined />} onClick={() => {
                queryClient.invalidateQueries({ queryKey: ['deploy-plan-detail', planId] })
                if (taskId) queryClient.invalidateQueries({ queryKey: ['deploy-task', taskId] })
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
          {plan?.addons && plan.addons.length > 0 && (
            <Descriptions.Item label="附加组件" span={3}>
              <Space wrap>
                {plan.addons.map((addon: string) => <Tag key={addon}>{addon}</Tag>)}
              </Space>
            </Descriptions.Item>
          )}
        </Descriptions>

        {(isDraft || canRetry) && (
          <Card
            size="small"
            title={<Space><SafetyCertificateOutlined />部署就绪检查</Space>}
            extra={<Button size="small" icon={<ReloadOutlined />} loading={preflightLoading} onClick={() => runPreflight()}>重新检查</Button>}
            style={{ marginBottom: 16 }}
          >
            {preflightFailed ? (
              <Alert type="error" showIcon message="无法完成部署预检" description="请确认后端服务可用且当前账号具有部署执行权限，然后重新检查。" />
            ) : !preflight ? (
              <Spin />
            ) : (
              <>
                <Alert
                  type={preflight.ready ? 'success' : 'error'}
                  showIcon
                  message={preflight.ready ? '预检通过，可以执行部署' : '预检未通过，执行已被阻止'}
                  description={`检查时间：${new Date(preflight.checkedAt).toLocaleString()}`}
                  style={{ marginBottom: 12 }}
                />
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  {preflight.checks.map((check: any) => (
                    <div key={check.key} style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
                      {check.status === 'passed'
                        ? <CheckCircleOutlined style={{ color: '#52c41a', marginTop: 3 }} />
                        : <CloseCircleOutlined style={{ color: check.status === 'warning' ? '#faad14' : '#ff4d4f', marginTop: 3 }} />}
                      <div style={{ minWidth: 0, flex: 1 }}>
                        <Text strong>{check.serverName ? `${check.serverName} · ` : ''}{check.message}</Text>
                        {check.remediation && <div><Text type="secondary">处理建议：{check.remediation}</Text></div>}
                      </div>
                      {check.ignorable && (
                        <Popconfirm
                          title={check.ignored ? '恢复该项强制检查？' : '确认已评估风险并忽略该项？'}
                          onConfirm={() => preflightIgnoreMutation.mutate({ key: check.key, ignored: !check.ignored })}
                        >
                          <Button size="small" loading={preflightIgnoreMutation.isPending}>
                            {check.ignored ? '恢复检查' : '忽略此项'}
                          </Button>
                        </Popconfirm>
                      )}
                    </div>
                  ))}
                </Space>
              </>
            )}
          </Card>
        )}

        {/* 节点拓扑 */}
        {nodeDetails.length > 0 && (
          <Card size="small" title={<Space><DesktopOutlined />节点拓扑</Space>} style={{ marginBottom: 16 }}>
            <Space wrap size={[16, 8]}>
              {nodeDetails.map((node, i) => (
                <Card
                  key={i}
                  size="small"
                  style={{ width: 260, background: '#fafafa' }}
                  title={
                    <Space>
                      <Tag color={roleColorMap[node.role] || 'default'}>{node.role?.toUpperCase()}</Tag>
                      <Text strong>{node.serverName}</Text>
                    </Space>
                  }
                >
                  <Space direction="vertical" size={2} style={{ width: '100%' }}>
                    <Text type="secondary">IP: <Text code>{node.serverIP}</Text></Text>
                    <Text type="secondary">OS: {node.serverOS}</Text>
                    <Space>
                      <Badge status={node.serverStatus === 'available' ? 'success' : 'default'} text={node.serverStatus} />
                    </Space>
                  </Space>
                </Card>
              ))}
            </Space>
          </Card>
        )}

        {/* Ansible 执行配置 */}
        <Card size="small" title="Ansible 执行配置" style={{ marginBottom: 16 }} loading={ansibleConfigLoading}>
          {!ansibleConfig ? (
            <Alert type="warning" showIcon message="暂无执行配置" />
          ) : (
            <Collapse ghost>
              <Collapse.Panel header={<Text strong>Playbook：{ansibleConfig.playbookPath}</Text>} key="playbook">
                <Alert type="info" showIcon message="该 playbook 为项目内置，实际执行时会根据计划生成动态 inventory 和 extra vars。" style={{ marginBottom: 8 }} />
                <YamlEditor readOnly value={`# 实际执行命令示例\nansible-playbook ${ansibleConfig.playbookPath} -i <动态 inventory> \\\n  -e k8s_version=${ansibleConfig.extraVars?.k8sVersion || ''} \\\n  -e pod_cidr=${ansibleConfig.extraVars?.podCidr || ''} \\\n  -e svc_cidr=${ansibleConfig.extraVars?.svcCidr || ''} \\\n  -e cni_type=${ansibleConfig.extraVars?.cniType || ''}\\n\\n# 完整 extra vars\\n${JSON.stringify(ansibleConfig.extraVars || {}, null, 2)}`} height={260} />
              </Collapse.Panel>
              <Collapse.Panel header={<Text strong>Inventory（密码已脱敏）</Text>} key="inventory">
                <YamlEditor readOnly value={ansibleConfig.inventory || '# 暂无 inventory'} height={320} />
              </Collapse.Panel>
            </Collapse>
          )}
        </Card>

        {/* 草稿状态提示 */}
        {isDraft && !taskId && (
          <Card style={{ marginBottom: 16 }}>
            <Empty
              description="该部署方案尚未执行"
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            >
              <Popconfirm title="确认执行该部署方案？" onConfirm={() => executeMutation.mutate()}>
                <Button type="primary" icon={<PlayCircleOutlined />} disabled={!preflight?.ready} loading={executeMutation.isPending}>
                  立即执行部署
                </Button>
              </Popconfirm>
            </Empty>
          </Card>
        )}

        {/* 部署进度 + 日志 左右布局 */}
        {task && task.steps && task.steps.length > 0 && (
          <Card size="small" title="部署进度" style={{ marginBottom: 16 }}>
            {task?.percent != null && (
              <Progress
                percent={task.percent}
                status={task.status === 'failed' ? 'exception' : task.status === 'success' ? 'success' : 'active'}
                style={{ marginBottom: 16 }}
              />
            )}
            <div style={{ display: 'flex', gap: 16, height: 'calc(100vh - 340px)', minHeight: 420 }}>
              {/* 左侧步骤树 */}
              <div style={{ width: 360, flexShrink: 0, borderRight: '1px solid #f0f0f0', paddingRight: 12, overflowY: 'auto', overflowX: 'hidden' }}>
                <StepTree
                  task={task}
                  canRetry={canRetry}
                  selectedStepKey={selectedStepKey}
                  selectedSubStepKey={selectedSubStepKey}
                  onSelectStep={handleSelectStep}
                  onSelectSubStep={handleSelectSubStep}
                  onRetryStep={(stepKey) => retryStepMutation.mutate({ stepKey })}
                  retrying={retryStepMutation.isPending}
                />
              </div>

              {/* 右侧日志面板 */}
              <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
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
      </Card>
    </AppPage>
  )
}

// 步骤树组件
function StepTree({
  task,
  canRetry,
  selectedStepKey,
  selectedSubStepKey,
  onSelectStep,
  onSelectSubStep,
  onRetryStep,
  retrying,
}: {
  task: DeployTask
  canRetry: boolean
  selectedStepKey: string | null
  selectedSubStepKey: string | null
  onSelectStep: (key: string) => void
  onSelectSubStep: (stepKey: string, subKey: string) => void
  onRetryStep: (key: string) => void
  retrying: boolean
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
              background: '#1677ff',
              color: '#fff',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 12,
              fontWeight: 'bold',
            }}>
              {stageIdx + 1}
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
                      background: isSelected ? '#e6f4ff' : stepSuccess ? '#f6ffed' : '#fafafa',
                      border: `1px solid ${isSelected ? '#91caff' : stepSuccess ? '#b7eb8f' : '#f0f0f0'}`,
                      cursor: 'pointer',
                      transition: 'all 0.2s',
                    }}
                  >
                    <Space>
                      <span style={{ color: stepSuccess ? '#52c41a' : stepFailed ? '#ff4d4f' : stepRunning ? '#1677ff' : '#999', fontWeight: 500 }}>
                        {stageIdx + 1}-{idx + 1} {step.title}
                      </span>
                      {stepRunning && <Badge status="processing" />}
                    </Space>
                    <Space>
                      {stepSuccess && <CheckCircleOutlined style={{ color: '#52c41a' }} />}
                      {stepFailed && <CloseCircleOutlined style={{ color: '#ff4d4f' }} />}
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
                              background: subSelected ? '#e6f4ff' : '#fff',
                              cursor: 'pointer',
                            }}
                          >
                            <CodeOutlined style={{ color: '#8c8c8c', fontSize: 12 }} />
                            <Text style={{ fontSize: 13, flex: 1, color: subSuccess ? '#52c41a' : subFailed ? '#ff4d4f' : subRunning ? '#1677ff' : '#595959' }}>
                              {sub.title}
                            </Text>
                            {subSuccess && <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 12 }} />}
                            {subFailed && <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 12 }} />}
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
function getLogColor(line: string): string {
  const trimmed = line.trim()
  if (trimmed.startsWith('PLAY [') || trimmed.startsWith('PLAY RECAP')) return '#569cd6'
  if (trimmed.startsWith('TASK [')) return '#c586c0'
  if (trimmed.includes('ok:') && !trimmed.includes('failed')) return '#4ec9b0'
  if (trimmed.startsWith('changed:') || trimmed.includes('changed=1')) return '#cca700'
  if (trimmed.includes('failed:') || trimmed.includes('FAILED') || trimmed.includes('fatal:')) return '#f44747'
  if (trimmed.startsWith('skipping:') || trimmed.includes('skipped')) return '#808080'
  if (line.includes('[error]') || line.includes('[ERROR]')) return '#f44747'
  if (line.includes('[warn]') || line.includes('[WARN]')) return '#cca700'
  if (line.includes('[info]') || line.includes('[INFO]')) return '#4ec9b0'
  return '#d4d4d4'
}
