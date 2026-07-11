/**
 * 部署计划详情页
 * 展示计划信息、节点拓扑、任务步骤进度、SSE 实时日志
 */
import { useEffect, useRef, useState, useCallback, useMemo } from 'react'
import { Card, Descriptions, Tag, Badge, Button, Space, Steps, Typography, message, Popconfirm, Tooltip, Progress, Empty, Input } from 'antd'
import { ArrowLeftOutlined, StopOutlined, RedoOutlined, DownloadOutlined, PlayCircleOutlined, SearchOutlined, ReloadOutlined, DesktopOutlined } from '@ant-design/icons'
import { history, useParams } from '@umijs/max'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { getDeployPlanById, getDeployTask, getDeployTaskLogs, getDeployTaskLogSSEUrl, cancelDeployPlan, retryDeployPlan, executeDeployPlan, getServers } from '@/services/deploy'

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

const stepStatusMap: Record<string, 'wait' | 'process' | 'finish' | 'error'> = {
  pending: 'wait',
  running: 'process',
  success: 'finish',
  failed: 'error',
}

const roleColorMap: Record<string, string> = {
  master: 'red',
  worker: 'blue',
}

export default function DeployPlanDetailPage() {
  const params = useParams<{ id: string }>()
  const planId = Number(params.id)
  const queryClient = useQueryClient()
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
      // 运行中时轮询刷新计划状态
      return query.state.data?.status === 'running' ? 3000 : false
    },
  })

  // 获取服务器列表（用于节点信息展示）
  const { data: serversData } = useQuery({
    queryKey: ['deploy-servers-for-detail'],
    queryFn: () => getServers({ page: 1, pageSize: 200 }),
    enabled: Number.isFinite(planId) && planId > 0,
  })

  const taskId = plan?.taskId

  // 获取任务详情（运行中时轮询）
  const { data: task } = useQuery({
    queryKey: ['deploy-task', taskId],
    queryFn: () => getDeployTask(taskId!),
    enabled: !!taskId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'running' || status === 'pending' ? 2000 : false
    },
  })

  // 构建节点信息（匹配服务器数据）
  const nodeDetails = useMemo(() => {
    if (!plan?.nodes || !serversData?.items) return []
    const serverMap = new Map(serversData.items.map((s) => [s.id, s]))
    return plan.nodes.map((node) => {
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

  // 初始加载历史日志
  useEffect(() => {
    if (!taskId) {
      setLogs([])
      logOffsetRef.current = 0
      return
    }
    getDeployTaskLogs(taskId, 0, 500).then((res) => {
      setLogs(res.logs || [])
      logOffsetRef.current = (res.logs || []).length
    }).catch(() => {
      setLogs([])
      logOffsetRef.current = 0
    })
  }, [taskId])

  // SSE 实时日志连接
  const connectSSE = useCallback(() => {
    if (!taskId) return
    eventSourceRef.current?.close()
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }

    const token = localStorage.getItem('token') || ''
    const url = getDeployTaskLogSSEUrl(taskId)
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
      // 任务可能仍在运行，尝试重连
      const currentStatus = task?.status
      if (currentStatus === 'running' || currentStatus === 'pending') {
        reconnectTimerRef.current = setTimeout(() => connectSSE(), 3000)
      }
    }
  }, [taskId, planId, queryClient, task?.status])

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
  const stepItems = (task?.steps || []).map((step) => ({
    title: step.title,
    description: step.message,
    status: stepStatusMap[step.status] || 'wait',
  }))

  const currentTaskStatus = task?.status
  const taskBadge = taskStatusMap[currentTaskStatus || ''] || { badge: 'default', text: currentTaskStatus || '-' }
  const filteredLogs = logFilter ? logs.filter((line) => line.toLowerCase().includes(logFilter.toLowerCase())) : logs

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
                <Button type="primary" icon={<PlayCircleOutlined />} loading={executeMutation.isPending}>执行部署</Button>
              </Popconfirm>
            )}
            {isRunning && (
              <Popconfirm title="确认取消该部署任务？" onConfirm={() => cancelMutation.mutate()}>
                <Button danger icon={<StopOutlined />} loading={cancelMutation.isPending}>取消部署</Button>
              </Popconfirm>
            )}
            {canRetry && (
              <Popconfirm title="确认重试该部署方案？" onConfirm={() => retryMutation.mutate()}>
                <Button type="primary" icon={<RedoOutlined />} loading={retryMutation.isPending}>重试部署</Button>
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
                {plan.addons.map((addon) => <Tag key={addon}>{addon}</Tag>)}
              </Space>
            </Descriptions.Item>
          )}
        </Descriptions>

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

        {/* 步骤进度 */}
        {stepItems.length > 0 && (
          <Card size="small" title="部署进度" style={{ marginBottom: 16 }}>
            <Steps size="small" current={stepItems.findIndex((s) => s.status === 'process')} items={stepItems} />
            {task?.percent != null && (
              <Progress
                percent={task.percent}
                status={task.status === 'failed' ? 'exception' : task.status === 'success' ? 'success' : 'active'}
                style={{ marginTop: 12 }}
              />
            )}
          </Card>
        )}

        {/* 草稿状态提示 */}
        {isDraft && !taskId && (
          <Card style={{ marginBottom: 16 }}>
            <Empty
              description="该部署方案尚未执行"
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            >
              <Popconfirm title="确认执行该部署方案？" onConfirm={() => executeMutation.mutate()}>
                <Button type="primary" icon={<PlayCircleOutlined />} loading={executeMutation.isPending}>
                  立即执行部署
                </Button>
              </Popconfirm>
            </Empty>
          </Card>
        )}

        {/* 实时日志终端 */}
        <Card
          size="small"
          title={
            <Space>
              <span>实时日志</span>
              {sseConnected && <Badge status="processing" text="实时连接" />}
              {!sseConnected && isRunning && <Badge status="warning" text="重连中" />}
              {task && task.status !== 'running' && task.status !== 'pending' && taskId && (
                <Badge status="default" text="已结束" />
              )}
              {logs.length > 0 && <Text type="secondary" style={{ fontSize: 12 }}>({logs.length} 行)</Text>}
            </Space>
          }
          extra={
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
          }
        >
          <div
            ref={logContainerRef}
            style={{
              background: '#1e1e1e',
              color: '#d4d4d4',
              fontFamily: 'Consolas, Monaco, "Courier New", monospace',
              fontSize: 13,
              lineHeight: 1.6,
              padding: 16,
              borderRadius: 6,
              maxHeight: 500,
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
        </Card>
      </Card>
    </AppPage>
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
