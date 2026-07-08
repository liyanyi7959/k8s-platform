import { history } from '@umijs/max'
import { message } from 'antd'
import type { Cluster as ModelCluster } from '@/models/cluster'

export const HEALTHY_CLUSTER_STATUSES = new Set([
  'active',
  'healthy',
  'connected',
  'success',
])

export const ATTENTION_CLUSTER_STATUSES = new Set([
  'warning',
  'degraded',
  'error',
  'failed',
  'disconnected',
  'unknown',
  'unhealthy',
  'offline',
])

export type ClusterAccessTarget = {
  id: number | string
  name: string
  status?: string
  k8sVersion?: string
  version?: string
}

type ClusterSetter = (cluster: ModelCluster | null) => void

export function normalizeClusterStatus(status?: string) {
  return String(status || 'unknown')
    .trim()
    .toLowerCase()
}

export function isClusterHealthy(status?: string) {
  return HEALTHY_CLUSTER_STATUSES.has(normalizeClusterStatus(status))
}

export function needsClusterAttention(status?: string) {
  return ATTENTION_CLUSTER_STATUSES.has(normalizeClusterStatus(status))
}

export function getClusterStatusText(status?: string) {
  const normalized = normalizeClusterStatus(status)

  if (isClusterHealthy(normalized)) {
    return '健康'
  }

  if (normalized === 'warning' || normalized === 'degraded') {
    return '需关注'
  }

  if (
    normalized === 'error' ||
    normalized === 'failed' ||
    normalized === 'disconnected' ||
    normalized === 'unhealthy' ||
    normalized === 'offline'
  ) {
    return '异常'
  }

  return '待确认'
}

export function getClusterStatusColor(status?: string) {
  const normalized = normalizeClusterStatus(status)

  if (isClusterHealthy(normalized)) {
    return 'success'
  }

  if (normalized === 'warning' || normalized === 'degraded') {
    return 'warning'
  }

  if (
    normalized === 'error' ||
    normalized === 'failed' ||
    normalized === 'disconnected' ||
    normalized === 'unhealthy' ||
    normalized === 'offline'
  ) {
    return 'error'
  }

  return 'default'
}

export function buildClusterAccessBlockedMessage(cluster: ClusterAccessTarget) {
  return `集群“${cluster.name}”当前不可进入，请检查集群健康状态。`
}

export function toClusterModel(cluster: ClusterAccessTarget): ModelCluster {
  return {
    id: String(cluster.id),
    name: cluster.name,
    version: cluster.k8sVersion || cluster.version || '',
    status: cluster.status || 'unknown',
  }
}

export function enterClusterWorkspace(
  cluster: ClusterAccessTarget,
  options: {
    setCurrentCluster?: ClusterSetter
    targetPath?: string
    replace?: boolean
    blockedMessage?: string
    silent?: boolean
  } = {},
) {
  if (!isClusterHealthy(cluster.status)) {
    if (!options.silent) {
      message.warning(options.blockedMessage || buildClusterAccessBlockedMessage(cluster))
    }
    return false
  }

  options.setCurrentCluster?.(toClusterModel(cluster))

  const targetPath = options.targetPath || `/k8s/${cluster.id}/dashboard`
  if (options.replace) {
    history.replace(targetPath)
  } else {
    history.push(targetPath)
  }

  return true
}
