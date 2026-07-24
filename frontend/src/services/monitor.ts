/**
 * 监控告警 API - 支持 Mock 模式
 * 当后端不可用时，自动使用本地模拟数据
 */
import { request } from '@umijs/max'
import type {
  AlertRule,
  AlertRuleListParams,
  AlertRuleListResponse,
  CreateAlertRuleRequest,
  AlertEvent,
  MetricDataPoint,
  MetricQueryParams,
  MetricsOverview,
  K8sEventItem,
  EventListParams,
  EventListResponse,
  MonitorIncident,
  IncidentTimelineItem,
} from '@/types'

const MOCK_ENABLED = false

const toCamelCase = (value: string) => value.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase())

const camelize = <T>(input: unknown): T => {
  if (Array.isArray(input)) {
    return input.map((item) => camelize(item)) as T
  }
  if (input && typeof input === 'object') {
    return Object.entries(input as Record<string, unknown>).reduce<Record<string, unknown>>(
      (result, [key, value]) => {
        result[toCamelCase(key)] = camelize(value)
        return result
      },
      {},
    ) as T
  }
  return input as T
}

/** 模拟告警规则数据 */
const MOCK_ALERT_RULES: AlertRule[] = [
  {
    id: 1,
    name: 'CPU 使用率过高',
    clusterId: 1,
    clusterName: '生产集群-华东',
    severity: 'warning',
    condition: 'cpu_usage > 80%',
    duration: '5m',
    enabled: true,
    receivers: ['admin@example.com'],
    createdAt: '2025-03-01T10:00:00Z',
    updatedAt: '2025-03-15T14:30:00Z',
  },
  {
    id: 2,
    name: '内存使用率过高',
    clusterId: 1,
    clusterName: '生产集群-华东',
    severity: 'warning',
    condition: 'memory_usage > 85%',
    duration: '5m',
    enabled: true,
    receivers: ['admin@example.com'],
    createdAt: '2025-03-01T10:05:00Z',
    updatedAt: '2025-03-15T14:35:00Z',
  },
  {
    id: 3,
    name: 'Pod 重启频繁',
    clusterId: 1,
    clusterName: '生产集群-华东',
    severity: 'critical',
    condition: 'pod_restarts > 5 in 1h',
    duration: '1h',
    enabled: true,
    receivers: ['admin@example.com', 'ops@example.com'],
    createdAt: '2025-03-05T11:00:00Z',
    updatedAt: '2025-03-15T14:40:00Z',
  },
]

/** 模拟告警事件数据 */
const MOCK_ALERT_EVENTS: AlertEvent[] = [
  {
    id: 1,
    ruleId: 1,
    ruleName: 'CPU 使用率过高',
    clusterId: 1,
    clusterName: '生产集群-华东',
    severity: 'warning',
    status: 'firing',
    message: '节点 worker-1 CPU 使用率达到 85%',
    value: '85%',
    startedAt: '2025-03-20T14:30:00Z',
    resolvedAt: null,
  },
  {
    id: 2,
    ruleId: 3,
    ruleName: 'Pod 重启频繁',
    clusterId: 1,
    clusterName: '生产集群-华东',
    severity: 'critical',
    status: 'resolved',
    message: 'Pod redis-master-0 在过去 1 小时内重启了 6 次',
    value: '6',
    startedAt: '2025-03-19T10:00:00Z',
    resolvedAt: '2025-03-19T11:30:00Z',
  },
]

/** 模拟监控指标数据 */
const generateMockMetrics = (startTime: string, endTime: string): MetricDataPoint[] => {
  const start = new Date(startTime).getTime()
  const end = new Date(endTime).getTime()
  const step = (end - start) / 20
  const metrics: MetricDataPoint[] = []

  for (let i = 0; i <= 20; i++) {
    const timestamp = new Date(start + step * i).toISOString()
    metrics.push({
      timestamp,
      value: Math.random() * 40 + 30, // 30-70%
    })
  }
  return metrics
}

/** 模拟事件流数据 */
const MOCK_EVENTS: K8sEventItem[] = [
  {
    id: '1',
    type: 'Normal',
    reason: 'Scheduled',
    message: 'Successfully assigned default/nginx-deployment-6d4f5d8c9b-abc12 to worker-1',
    namespace: 'default',
    object: 'nginx-deployment-6d4f5d8c9b-abc12',
    objectKind: 'Pod',
    clusterId: 1,
    clusterName: '生产集群-华东',
    timestamp: '2025-03-20T14:30:00Z',
  },
  {
    id: '2',
    type: 'Normal',
    reason: 'Pulled',
    message: 'Container image "nginx:1.24" already present on machine',
    namespace: 'default',
    object: 'nginx-deployment-6d4f5d8c9b-abc12',
    objectKind: 'Pod',
    clusterId: 1,
    clusterName: '生产集群-华东',
    timestamp: '2025-03-20T14:30:05Z',
  },
  {
    id: '3',
    type: 'Warning',
    reason: 'BackOff',
    message: 'Back-off restarting failed container',
    namespace: 'default',
    object: 'redis-master-0',
    objectKind: 'Pod',
    clusterId: 1,
    clusterName: '生产集群-华东',
    timestamp: '2025-03-20T14:25:00Z',
  },
  {
    id: '4',
    type: 'Normal',
    reason: 'ScalingReplicaSet',
    message: 'Scaled up replica set backend-api-6d4f5d8c9b to 2',
    namespace: 'production',
    object: 'backend-api',
    objectKind: 'Deployment',
    clusterId: 1,
    clusterName: '生产集群-华东',
    timestamp: '2025-03-20T14:20:00Z',
  },
]

/** 获取告警规则列表 */
export function listAlertRules(
  params?: AlertRuleListParams,
  signal?: AbortSignal,
): Promise<AlertRuleListResponse> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          items: MOCK_ALERT_RULES,
          total: MOCK_ALERT_RULES.length,
          page: params?.page || 1,
          pageSize: params?.pageSize || 10,
        })
      }, 200)
    })
  }
  return request('/api/v1/monitor/alerts', {
    params: { page: params?.page, page_size: params?.pageSize },
    signal,
  }).then(pageResult<AlertRule>)
}

/** 创建告警规则 */
export function createAlertRule(data: CreateAlertRuleRequest): Promise<AlertRule> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const newRule: AlertRule = {
        id: MOCK_ALERT_RULES.length + 1,
        name: data.name,
        clusterId: data.clusterId,
        clusterName: '未知集群',
        severity: data.severity,
        condition: data.condition,
        duration: data.duration,
        enabled: true,
        receivers: data.receivers || [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }
      MOCK_ALERT_RULES.push(newRule)
      resolve(newRule)
    })
  }
  return request('/api/v1/monitor/alerts', { method: 'POST', data })
}

/** 更新告警规则 */
export function updateAlertRule(
  id: number,
  data: Partial<CreateAlertRuleRequest>,
): Promise<AlertRule> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const index = MOCK_ALERT_RULES.findIndex((r) => r.id === id)
      if (index !== -1) {
        MOCK_ALERT_RULES[index] = { ...MOCK_ALERT_RULES[index]!, ...data, id, updatedAt: new Date().toISOString() }
        resolve(MOCK_ALERT_RULES[index]!)
      } else {
        reject(new Error('告警规则不存在'))
      }
    })
  }
  return request(`/api/v1/monitor/alerts/${id}`, { method: 'PUT', data })
}

/** 删除告警规则 */
export function deleteAlertRule(id: number): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const index = MOCK_ALERT_RULES.findIndex((r) => r.id === id)
      if (index !== -1) {
        MOCK_ALERT_RULES.splice(index, 1)
        resolve()
      } else {
        reject(new Error('告警规则不存在'))
      }
    })
  }
  return request(`/api/v1/monitor/alerts/${id}`, { method: 'DELETE' })
}

/** 启用/禁用告警规则 */
export function toggleAlertRule(id: number, enabled: boolean): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const rule = MOCK_ALERT_RULES.find((r) => r.id === id)
      if (rule) {
        rule.enabled = enabled
        rule.updatedAt = new Date().toISOString()
        resolve()
      } else {
        reject(new Error('告警规则不存在'))
      }
    })
  }
  return request(`/api/v1/monitor/alerts/${id}/toggle`, { method: 'PUT', data: { enabled } })
}

/** 获取告警事件列表 */
export function listAlertEvents(clusterId?: number, signal?: AbortSignal): Promise<AlertEvent[]> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_ALERT_EVENTS]
      if (clusterId) {
        filtered = filtered.filter((e) => e.clusterId === clusterId)
      }
      setTimeout(() => resolve(filtered), 150)
    })
  }
  return request('/api/v1/monitor/alert-events', { params: { clusterId }, signal })
}

const pageResult = <T>(raw: any) => {
  const data = raw?.data || raw || {}
  const list = Array.isArray(data.items) ? data.items : Array.isArray(data.list) ? data.list : []
  return {
    items: camelize<T[]>(list),
    total: data.total || 0,
    page: data.page || 1,
    pageSize: data.pageSize || data.page_size || 20,
  } as { items: T[]; total: number; page: number; pageSize: number }
}

export function listIncidents(params?: { page?: number; pageSize?: number; status?: string }, signal?: AbortSignal) {
  return request('/api/v1/monitor/incidents', {
    params: { page: params?.page, page_size: params?.pageSize, status: params?.status },
    signal,
  }).then(pageResult<MonitorIncident>)
}

export function getIncident(id: number): Promise<{ incident: MonitorIncident; timeline: IncidentTimelineItem[] }> {
  return request(`/api/v1/monitor/incidents/${id}`).then((raw) => camelize(raw))
}

export function transitionIncident(id: number, action: string, note = ''): Promise<void> {
  return request(`/api/v1/monitor/incidents/${id}/transition`, { method: 'POST', data: { action, note } })
}

/** 查询监控指标 */
export function queryMetrics(
  params: MetricQueryParams,
  signal?: AbortSignal,
): Promise<MetricDataPoint[]> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve(generateMockMetrics(params.startTime, params.endTime))
      }, 200)
    })
  }
  return request('/api/v1/monitor/metrics/query', { params, signal })
}

/** 获取监控指标（仪表盘用） */
export function getMetrics(
  params: { clusterId: number; startTime: string; endTime: string },
  signal?: AbortSignal,
): Promise<MetricDataPoint[]> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve(generateMockMetrics(params.startTime, params.endTime))
      }, 200)
    })
  }
  return queryMetrics({ ...params, metric: 'cpu_usage' }, signal)
}

/** 获取集群资源概览 */
export function getMetricsOverview(signal?: AbortSignal): Promise<MetricsOverview> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          clusterMetrics: [
            { clusterId: 1, clusterName: '生产集群-华东', cpuUsage: 0.62, memoryUsage: 0.71, diskUsage: 0.45, nodeCount: 12, nodeCapacity: 20, podCount: 156, podCapacity: 500, alertCount: 3 },
            { clusterId: 2, clusterName: '生产集群-华北', cpuUsage: 0.45, memoryUsage: 0.58, diskUsage: 0.32, nodeCount: 8, nodeCapacity: 15, podCount: 89, podCapacity: 300, alertCount: 1 },
            { clusterId: 3, clusterName: '测试集群', cpuUsage: 0.28, memoryUsage: 0.35, diskUsage: 0.18, nodeCount: 4, nodeCapacity: 10, podCount: 32, podCapacity: 100, alertCount: 0 },
          ],
        })
      }, 200)
    })
  }
  return request('/api/v1/monitor/metrics/overview', { signal })
}

/** 获取事件流 */
export function listEvents(
  params: EventListParams,
  signal?: AbortSignal,
): Promise<EventListResponse> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_EVENTS]
      if (params.clusterId) {
        filtered = filtered.filter((e) => e.clusterId === params.clusterId)
      }
      if (params.namespace) {
        filtered = filtered.filter((e) => e.namespace === params.namespace)
      }
      if (params.type) {
        filtered = filtered.filter((e) => e.type === params.type)
      }
      setTimeout(() => {
        resolve({
          items: filtered,
          total: filtered.length,
          page: params.page || 1,
          pageSize: params.pageSize || 20,
        })
      }, 150)
    })
  }
  return request('/api/v1/monitor/events', { params, signal })
}
