/**
 * 工作负载 API：Deployment / StatefulSet / DaemonSet / ReplicaSet
 */
import { request } from '@umijs/max'
import type {
  Deployment,
  DeploymentList,
  DeploymentListParams,
  ReplicaSetList,
} from '@/features/kops/types'
import { mapDeployment, mapReplicaSet, extractMappedList } from './shared'

type WorkloadKind = 'Deployment' | 'StatefulSet' | 'DaemonSet'

const WORKLOAD_ENDPOINT_MAP = {
  Deployment: 'deployments',
  StatefulSet: 'statefulsets',
  DaemonSet: 'daemonsets',
} as const

/** 获取工作负载列表 */
export function listWorkloads(
  clusterId: number,
  params: DeploymentListParams & { kind?: WorkloadKind },
  signal?: AbortSignal,
): Promise<DeploymentList> {
  return request(`/api/v1/clusters/${clusterId}/workloads`, { params, signal }).then(extractMappedList(mapDeployment))
}

/** 删除工作负载 */
export function deleteWorkload(
  clusterId: number,
  namespace: string,
  name: string,
  kind: WorkloadKind = 'Deployment',
): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/workloads/${kind}/${namespace}/${name}`, {
    method: 'DELETE',
  })
}

/** 更新工作负载副本数 */
export function scaleWorkload(
  clusterId: number,
  namespace: string,
  name: string,
  replicas: number,
  kind: WorkloadKind = 'Deployment',
): Promise<Deployment> {
  return request(`/api/v1/clusters/${clusterId}/workloads/${WORKLOAD_ENDPOINT_MAP[kind]}/${namespace}/${name}/scale-operations`, {
	method: 'POST',
    data: { kind, namespace, name, replicas },
  })
}

/** 回滚 Deployment */
export function rollbackDeployment(
  clusterId: number,
  namespace: string,
  name: string,
  revision: number,
): Promise<Deployment> {
  return request(`/api/v1/clusters/${clusterId}/workloads/deployments/${namespace}/${name}/rollback-attempts`, {
    method: 'POST',
    data: { revision },
  })
}

/** 创建工作负载 */
export function createWorkload(
  clusterId: number,
  namespace: string,
  data: any,
  kind: WorkloadKind = 'Deployment',
): Promise<any> {
  const payload = {
    namespace,
    name: data.name,
    replicas: Number(data.replicas || 1),
    labels: (() => {
      if (!data.labels) return undefined
      if (typeof data.labels === 'string') {
        try {
          return JSON.parse(data.labels)
        } catch {
          return undefined
        }
      }
      return data.labels
    })(),
    containers: [
      {
        name: data.name,
        image: data.image,
        command: data.command,
        cpu: data.cpu,
        memory: data.memory,
      },
    ],
  }
  return request(`/api/v1/clusters/${clusterId}/workloads/${WORKLOAD_ENDPOINT_MAP[kind]}`, { method: 'POST', data: payload })
}

/** 重启工作负载 */
export function restartWorkload(
  clusterId: number,
  namespace: string,
  name: string,
  kind: WorkloadKind = 'Deployment',
): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/workloads/${WORKLOAD_ENDPOINT_MAP[kind]}/${namespace}/${name}/restart-operations`, { method: 'POST', data: { kind, namespace, name } })
}

/** 获取 Deployment 详情 */
export function getDeploymentDetail(clusterId: number, namespace: string, name: string): Promise<Deployment> {
  return request(`/api/v1/clusters/${clusterId}/deployments/${namespace}/${name}`, {})
}

/** 更新工作负载（编辑） */
export function updateWorkload(clusterId: number, namespace: string, name: string, data: Record<string, unknown>): Promise<Deployment> {
  const kind = (data.kind as WorkloadKind | undefined) || 'Deployment'
  return request(`/api/v1/clusters/${clusterId}/workloads/${WORKLOAD_ENDPOINT_MAP[kind]}/${namespace}/${name}`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      replicas: data.replicas,
      containers: data.containers,
      strategy: data.strategy,
    },
  })
}

/** 暂停/恢复工作负载滚动更新 */
export function updateWorkloadPaused(
  clusterId: number,
  namespace: string,
  name: string,
  paused: boolean,
): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/workloads/deployments/${namespace}/${name}/pause-state`, {
    method: 'PATCH',
    data: { kind: 'Deployment', namespace, name, paused },
  })
}

/** 更新工作负载镜像 */
export function updateWorkloadImage(
  clusterId: number,
  namespace: string,
  name: string,
  container: string,
  image: string,
  kind: WorkloadKind = 'Deployment',
): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/workloads/${WORKLOAD_ENDPOINT_MAP[kind]}/${namespace}/${name}/image-updates`, {
    method: 'POST',
    data: { kind, namespace, name, container, image },
  })
}

/** 获取 Deployment YAML */
export function getDeploymentYaml(
  clusterId: number,
  namespace: string,
  name: string,
  signal?: AbortSignal,
  kind: WorkloadKind = 'Deployment',
): Promise<{ yaml: string }> {
  return request(`/api/v1/clusters/${clusterId}/workloads/${kind}/${namespace}/${name}/yaml`, { signal }).then((res: { text?: string; yaml?: string }) => ({
    yaml: res.yaml || res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** 获取 Deployment 版本历史 */
export function getDeploymentHistory(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<Array<{ revision: number; changeCause: string; date: string; image: string }>> {
  return request(`/api/v1/clusters/${clusterId}/workloads/deployments/${namespace}/${name}/rollout-history`, { signal }).then((res: any) => res.history || [])
}

/** 获取工作负载列表（兼容 workloads 页面调用签名） */
export function getWorkloads(
  clusterId: number,
  namespace?: string,
  signal?: AbortSignal,
  kind: WorkloadKind = 'Deployment',
): Promise<DeploymentList> {
  return listWorkloads(clusterId, { namespace: namespace || undefined, kind }, signal)
}

/** @deprecated 使用领域名称版本 */
export const listDeployments = listWorkloads
export const deleteDeployment = deleteWorkload
export const scaleDeployment = scaleWorkload
export const createDeployment = createWorkload
export const restartDeployment = restartWorkload
export const updateDeployment = updateWorkload
export const getDeployments = getWorkloads

/** 获取 ReplicaSet 列表 */
export function listReplicaSets(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<ReplicaSetList> {
  return request(`/api/v1/clusters/${clusterId}/replicasets`, { params: { namespace }, signal }).then(extractMappedList(mapReplicaSet))
}

/** 删除 ReplicaSet */
export function deleteReplicaSet(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/replicasets/${namespace}/${name}`, { method: 'DELETE' })
}
