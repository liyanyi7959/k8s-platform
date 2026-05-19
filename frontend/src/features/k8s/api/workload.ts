// k8s 工作负载相关 API（Deployment / StatefulSet / DaemonSet 等）
import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export async function listWorkloads(
  clusterId: number,
  params: { kind?: string; namespace?: string; label_selector?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/workloads`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function scaleWorkload(
  clusterId: number,
  req: { kind: string; namespace: string; name: string; replicas: number }
): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/workloads/scale`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export async function restartWorkload(clusterId: number, req: { kind: string; namespace: string; name: string }): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/workloads/restart`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export async function updateWorkloadImage(
  clusterId: number,
  req: { kind: string; namespace: string; name: string; container: string; image: string }
): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/workloads/image`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export async function updateWorkloadPaused(
  clusterId: number,
  req: { kind: string; namespace: string; name: string; paused: boolean }
): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/workloads/rollout-pause`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export interface RolloutRevision {
  revision: number
  change_cause: string
  images: string[]
  created_at: string
  is_current: boolean
}

export async function getRolloutHistory(
  clusterId: number,
  namespace: string,
  name: string,
  kind: string
): Promise<{ history: RolloutRevision[] }> {
  if (kind !== 'Deployment') {
    throw new Error('当前仅支持 Deployment 版本历史')
  }
  const resp = (await http.get(
    `/api/v1/clusters/${clusterId}/workloads/deployments/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/rollout-history`
  )) as ApiResponse<{ history: RolloutRevision[] }>
  return resp.data
}

export async function rolloutUndo(
  clusterId: number,
  namespace: string,
  name: string,
  body: { kind: string; revision: number }
): Promise<{ ok: boolean }> {
  if (body.kind !== 'Deployment') {
    throw new Error('当前仅支持 Deployment 回滚')
  }
  const resp = (await http.post(
    `/api/v1/clusters/${clusterId}/workloads/deployments/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/rollout-undo`,
    body
  )) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export type EditDeploymentRequest = {
  namespace: string
  name: string
  replicas?: number
  labels?: Record<string, string>
  tolerations?: Array<{
    key?: string
    operator?: string
    value?: string
    effect?: string
    tolerationSeconds?: number
  }>
  strategy?: {
    type?: string
    maxSurge?: string
    maxUnavailable?: string
  }
  volumes?: Array<Record<string, any>>
  containers?: Array<{
    name: string
    image?: string
    imagePullPolicy?: string
    resources?: { requests?: Record<string, string>; limits?: Record<string, string> }
    env?: Array<Record<string, any>>
    envFrom?: Array<Record<string, any>>
    volumeMounts?: Array<Record<string, any>>
    probes?: {
      liveness?: Partial<{
        initialDelaySeconds: number
        timeoutSeconds: number
        periodSeconds: number
        successThreshold: number
        failureThreshold: number
      }>
      readiness?: Partial<{
        initialDelaySeconds: number
        timeoutSeconds: number
        periodSeconds: number
        successThreshold: number
        failureThreshold: number
      }>
      startup?: Partial<{
        initialDelaySeconds: number
        timeoutSeconds: number
        periodSeconds: number
        successThreshold: number
        failureThreshold: number
      }>
    }
  }>
  initContainers?: Array<{
    name: string
    image?: string
    imagePullPolicy?: string
    resources?: { requests?: Record<string, string>; limits?: Record<string, string> }
    env?: Array<Record<string, any>>
    envFrom?: Array<Record<string, any>>
    volumeMounts?: Array<Record<string, any>>
    probes?: {
      liveness?: Partial<{
        initialDelaySeconds: number
        timeoutSeconds: number
        periodSeconds: number
        successThreshold: number
        failureThreshold: number
      }>
      readiness?: Partial<{
        initialDelaySeconds: number
        timeoutSeconds: number
        periodSeconds: number
        successThreshold: number
        failureThreshold: number
      }>
      startup?: Partial<{
        initialDelaySeconds: number
        timeoutSeconds: number
        periodSeconds: number
        successThreshold: number
        failureThreshold: number
      }>
    }
  }>
}

export async function editDeployment(clusterId: number, req: EditDeploymentRequest): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/workloads/deployments/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export async function editStatefulSet(clusterId: number, req: EditDeploymentRequest): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/workloads/statefulsets/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export async function editDaemonSet(clusterId: number, req: EditDeploymentRequest): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/workloads/daemonsets/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export async function deleteWorkload(clusterId: number, req: { kind: string; namespace: string; name: string }): Promise<void> {
  await http.delete(
    `/api/v1/clusters/${clusterId}/workloads/${encodeURIComponent(req.kind)}/${encodeURIComponent(req.namespace)}/${encodeURIComponent(req.name)}`
  )
}

export async function getWorkloadYaml(clusterId: number, req: { kind: string; namespace: string; name: string }): Promise<{ text: string }> {
  const resp = (await http.get(
    `/api/v1/clusters/${clusterId}/workloads/${encodeURIComponent(req.kind)}/${encodeURIComponent(req.namespace)}/${encodeURIComponent(req.name)}/yaml`
  )) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editWorkloadYaml(
  clusterId: number,
  req: { kind: string; namespace: string; yaml: string }
): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/workloads/yaml/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}
