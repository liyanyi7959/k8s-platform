/**
 * 监控告警类型 — 扁平结构
 */
import type { PageResult } from './common'

/** 告警规则 */
export interface AlertRule {
  id: number
  name: string
  clusterId: number
  clusterName: string
  severity: 'info' | 'warning' | 'critical'
  condition: string
  duration: string
  enabled: boolean
  receivers: string[]
  createdAt: string
  updatedAt: string
}

export interface AlertRuleListParams {
  page?: number
  pageSize?: number
}

export type AlertRuleListResponse = PageResult<AlertRule>

export interface CreateAlertRuleRequest {
  name: string
  clusterId: number
  severity: 'info' | 'warning' | 'critical'
  condition: string
  duration: string
  receivers?: string[]
}

/** 告警事件 */
export interface AlertEvent {
  id: number
  ruleId: number
  ruleName: string
  clusterId: number
  clusterName: string
  severity: string
  status: 'firing' | 'resolved'
  message: string
  value: string
  startedAt: string
  resolvedAt: string | null
}

export type IncidentStatus = 'open' | 'acknowledged' | 'diagnosing' | 'awaiting_approval' | 'executing' | 'verifying' | 'resolved'

export interface MonitorIncident {
  id: number
  alertName: string
  clusterId: number
  clusterName: string
  namespace: string
  resourceKind: string
  resourceName: string
  severity: 'info' | 'warning' | 'critical'
  status: IncidentStatus
  summary: string
  startedAt: string
  resolvedAt?: string
  aiConversationId?: number
  aiProposalId?: number
  assigneeName: string
  verificationNote?: string
}

export interface IncidentTimelineItem {
  id: number
  type: string
  title: string
  detail: string
  operator: string
  createdAt: string
}

/** 监控指标 */
export interface MetricDataPoint {
  timestamp: string
  value: number
}

export interface MetricQueryParams {
  clusterId: number
  metric: string
  startTime: string
  endTime: string
}

/** K8s 事件流 */
export interface K8sEventItem {
  id: string
  type: string
  reason: string
  message: string
  namespace: string
  object: string
  objectKind: string
  clusterId: number
  clusterName: string
  timestamp: string
}

export interface EventListParams {
  clusterId?: number
  namespace?: string
  type?: string
  page?: number
  pageSize?: number
}

export type EventListResponse = PageResult<K8sEventItem>

/** 集群资源指标 */
export interface ClusterMetric {
  clusterId: number
  clusterName: string
  cpuUsage: number
  memoryUsage: number
  diskUsage: number
  nodeCount: number
  nodeCapacity: number
  podCount: number
  podCapacity: number
  alertCount: number
}

/** 监控概览 */
export interface MetricsOverview {
  clusterMetrics: ClusterMetric[]
}
