/**
 * 集群级资源 API：Namespace / Node / Event / Dashboard / Topology
 */
import { request } from '@umijs/max'
import type { Namespace, Node, NodeDetail, K8sEvent } from '@/features/kops/types'
import { mapNode, mapPod, mapService, mapEvent, extractMappedSimpleList } from './shared'

// ==================== Namespace ====================

/** 获取命名空间列表 */
export function listNamespaces(clusterId: number, signal?: AbortSignal): Promise<Namespace[]> {
  return request(`/api/v1/clusters/${clusterId}/namespaces`, { signal }).then(
    extractMappedSimpleList((r: any) => ({
      name: r?.metadata?.name || r?.name || '',
      status: r?.status?.phase || 'Active',
      labels: r?.metadata?.labels || {},
      createdAt: r?.metadata?.creationTimestamp || '',
    })),
  )
}

/** 创建命名空间 */
export function createNamespace(clusterId: number, name: string): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/namespaces`, {
    method: 'POST',
    data: { metadata: { name } },
  })
}

/** 删除命名空间 */
export function deleteNamespace(clusterId: number, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/namespaces/${name}`, { method: 'DELETE' })
}

/** 获取命名空间列表（带 items 包装，兼容拓扑图） */
export function getNamespaces(
  clusterId: number,
  signal?: AbortSignal,
): Promise<{ items: Namespace[] }> {
  return listNamespaces(clusterId, signal).then((namespaces) => ({ items: namespaces }))
}

/** Namespace 巡检诊断 */
export function getNamespaceInspection(
  clusterId: number,
  namespace: string,
  signal?: AbortSignal,
): Promise<{ text: string }> {
  return request(`/api/v1/clusters/${clusterId}/namespaces/${namespace}/inspection`, {
    signal,
  }).then((res: any) => ({
    text: res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** Namespace 资源摘要 */
export function getNamespaceResourcesSummary(
  clusterId: number,
  namespace: string,
  signal?: AbortSignal,
): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/namespaces/${namespace}/resources-summary`, {
    signal,
  })
}

/** Namespace 工作负载清单 */
export function getNamespaceWorkloadInventory(
  clusterId: number,
  namespace: string,
  signal?: AbortSignal,
): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/namespaces/${namespace}/workload-inventory`, {
    signal,
  })
}

// ==================== Node ====================

/** 获取节点列表 */
export function listNodes(clusterId: number, signal?: AbortSignal): Promise<Node[]> {
  return request(`/api/v1/clusters/${clusterId}/nodes`, { signal }).then(
    extractMappedSimpleList(mapNode),
  )
}

/** 获取节点详情 */
export function getNodeDetail(
  clusterId: number,
  name: string,
  signal?: AbortSignal,
): Promise<NodeDetail> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/detail`, { signal }).then(
    (raw: any) => {
      const base = mapNode(raw)
      const s = raw?.status || {}
      const addresses = Array.isArray(s.addresses)
        ? s.addresses.map((a: any) => ({ type: a.type || '', address: a.address || '' }))
        : []
      const conditions = Array.isArray(s.conditions)
        ? s.conditions.map((c: any) => ({
            type: c.type || '',
            status: c.status || '',
            reason: c.reason || '',
            message: c.message || '',
            lastTransitionTime: c.lastTransitionTime || '',
          }))
        : []
      const images = Array.isArray(s.images)
        ? s.images.map((img: any) => ({ names: img.names || [], sizeBytes: img.sizeBytes || 0 }))
        : []
      const allocatable = s.allocatable || {}
      const capacity = s.capacity || {}
      return { ...base, addresses, conditions, images, allocatable, capacity } as NodeDetail
    },
  )
}

/** 停止节点调度 (Cordon) */
export function cordonNode(clusterId: number, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/cordon-requests`, { method: 'POST' })
}

/** 恢复节点调度 (Uncordon) */
export function uncordonNode(clusterId: number, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/uncordon-requests`, { method: 'POST' })
}

/** 驱逐节点 (Drain) */
export function drainNode(
  clusterId: number,
  name: string,
  options?: { force?: boolean; timeout_seconds?: number; ignore_daemonsets?: boolean },
): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/drain-requests`, {
    method: 'POST',
    data: options,
  })
}

/** 删除节点 */
export function deleteNode(clusterId: number, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}`, { method: 'DELETE' })
}

/** 获取节点 YAML */
export function getNodeYaml(
  clusterId: number,
  name: string,
  signal?: AbortSignal,
): Promise<{ yaml: string }> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/yaml`, { signal }).then(
    (res: { text?: string; yaml?: string }) => ({
      yaml: res.yaml || res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
    }),
  )
}

/** 获取 Node 上的 Pod 列表 */
export function getNodePods(
  clusterId: number,
  nodeName: string,
  signal?: AbortSignal,
): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${nodeName}/pods`, { signal }).then(
    (res: any) => res.list || res || [],
  )
}

/** 获取 Node 事件 */
export function getNodeEvents(
  clusterId: number,
  nodeName: string,
  signal?: AbortSignal,
): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${nodeName}/events`, { signal }).then(
    (res: any) => res.list || res || [],
  )
}

/** 获取节点列表（带 items 包装，兼容拓扑图） */
export function getNodes(clusterId: number, signal?: AbortSignal): Promise<{ items: Node[] }> {
  return listNodes(clusterId, signal).then((nodes) => ({ items: nodes }))
}

// ==================== Event ====================

/** 获取事件列表 */
export function listEvents(
  clusterId: number,
  namespace?: string,
  signal?: AbortSignal,
): Promise<K8sEvent[]> {
  return request(`/api/v1/clusters/${clusterId}/events`, { params: { namespace }, signal }).then(
    extractMappedSimpleList(mapEvent),
  )
}

// ==================== Dashboard ====================

/** 获取仪表盘统计数据 */
export function getDashboardStats(
  clusterId: number,
  signal?: AbortSignal,
): Promise<{
  namespaceCount: number
  podCount: number
  deploymentCount: number
  serviceCount: number
  cpuUsage: number
  memoryUsage: number
  podRunning: number
  podPending: number
  podFailed: number
}> {
  return request(`/api/v1/dashboard/clusters/${clusterId}/overview`, { signal })
}

/** 集群概览 - 完整仪表盘数据 */
export interface ClusterOverview {
  cluster: { name: string; status: string; api_ok?: boolean; k8s_version?: string }
  stats: {
    nodes: { total: number; ready: number }
    pods: { total: number; running: number; pending: number; failed: number; succeeded: number }
    cpu: { used_percent: number }
    memory: { used_percent: number }
    workloads: {
      deployments: number
      statefulsets: number
      daemonsets: number
      replicasets?: number
    }
    namespaces?: number
  }
  charts: {
    cpu_memory_24h: {
      labels: string[]
      timestamps?: string[]
      cpu: number[]
      memory: number[]
      scope?: 'cluster'
      sample_count?: number
    }
    node_cpu_memory_24h?: Array<{
      name: string
      ip?: string
      labels: string[]
      timestamps?: string[]
      cpu: number[]
      memory: number[]
      sample_count?: number
    }>
    pod_phase: { running: number; pending: number; failed: number; succeeded: number }
    namespace_pods_top: Array<{ namespace: string; pods: number }>
    node_ready: { ready: number; total: number }
  }
  risks?: {
    certificates: Array<{
      key: string
      name: string
      component: string
      purpose: string
      not_after?: string
      days_left?: number
      status: string
    }>
  }
  anomalies?: {
    failed_pods?: Array<{ name: string; namespace: string; reason: string }>
    unscheduled_pods?: Array<{ name: string; namespace: string; reason: string }>
  }
  top_workloads?: Array<{
    name: string
    namespace: string
    kind: string
    replicas: number
    ready: number
  }>
  meta?: {
    source: string
    updated_at: string
    cached: boolean
    metrics_available?: boolean
    metrics_source?: string
    metrics_scope?: 'cluster'
    metrics_basis?: string
  }
  events?: K8sEvent[]
}

export type ClusterCertificateRisk = NonNullable<
  NonNullable<ClusterOverview['risks']>['certificates']
>[number]

/** 获取集群概览 */
export function getClusterOverview(
  clusterId: number,
  signal?: AbortSignal,
): Promise<ClusterOverview> {
  return request(`/api/v1/dashboard/clusters/${clusterId}/overview`, { signal })
}

/** 证书探测独立于总览首屏加载，避免远程 TLS 握手阻塞核心健康数据。 */
export function getClusterCertificateRisks(
  clusterId: number,
  signal?: AbortSignal,
): Promise<ClusterCertificateRisk[]> {
  return request(`/api/v1/dashboard/clusters/${clusterId}/certificate-risks`, { signal })
}

// ==================== Topology ====================

/** 获取集群拓扑数据 - 从 nodes、pods、services 端点构造 */
export async function getTopology(clusterId: number, signal?: AbortSignal): Promise<any> {
  const [nodes, pods, services] = await Promise.all([
    request(`/api/v1/clusters/${clusterId}/nodes`, { signal }).then(
      extractMappedSimpleList(mapNode),
    ),
    request(`/api/v1/clusters/${clusterId}/pods`, { signal }).then(extractMappedSimpleList(mapPod)),
    request(`/api/v1/clusters/${clusterId}/services`, { signal }).then(
      extractMappedSimpleList(mapService),
    ),
  ])

  const topologyNodes: any[] = []
  const topologyEdges: any[] = []

  for (const node of nodes) {
    topologyNodes.push({
      id: node.name,
      type: 'node',
      data: {
        label: node.name,
        status: node.status,
        roles: node.roles,
        version: node.kubeletVersion,
      },
    })
  }

  for (const pod of pods) {
    topologyNodes.push({
      id: `${pod.namespace}-${pod.name}`,
      type: 'pod',
      data: {
        label: pod.name,
        namespace: pod.namespace,
        status: pod.status,
        nodeName: pod.nodeName,
      },
    })
    if (pod.nodeName) {
      topologyEdges.push({
        id: `${pod.nodeName}-${pod.namespace}-${pod.name}`,
        source: pod.nodeName,
        target: `${pod.namespace}-${pod.name}`,
      })
    }
  }

  for (const svc of services) {
    topologyNodes.push({
      id: `svc-${svc.namespace}-${svc.name}`,
      type: 'service',
      data: { label: svc.name, namespace: svc.namespace, type: svc.type, clusterIP: svc.clusterIP },
    })
  }

  return { nodes: topologyNodes, edges: topologyEdges }
}
