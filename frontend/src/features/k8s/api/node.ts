// k8s 节点相关 API
import { http } from '@/shared/http/http'
import type { ApiResponse, K8sListResponse, K8sObjectResponse } from '@/shared/types/api'

export async function listNodes(clusterId: number, params: { sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<K8sListResponse<any>> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/nodes`, { params })) as ApiResponse<K8sListResponse<any>>
  return resp.data
}

export async function getNodeYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/nodes/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function getNodeDetail(clusterId: number, name: string): Promise<K8sObjectResponse<any>> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/nodes/${encodeURIComponent(name)}/detail`)) as ApiResponse<K8sObjectResponse<any>>
  return resp.data
}

export async function listNodePods(
  clusterId: number,
  name: string,
  params: { sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<K8sListResponse<any>> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/nodes/${encodeURIComponent(name)}/pods`, { params })) as ApiResponse<K8sListResponse<any>>
  return resp.data
}

export async function listNodeEvents(clusterId: number, name: string): Promise<K8sListResponse<any>> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/nodes/${encodeURIComponent(name)}/events`)) as ApiResponse<K8sListResponse<any>>
  return resp.data
}

export async function cordonNode(clusterId: number, name: string): Promise<void> {
  await http.post(`/api/v1/clusters/${clusterId}/nodes/${encodeURIComponent(name)}/cordon`)
}

export async function uncordonNode(clusterId: number, name: string): Promise<void> {
  await http.post(`/api/v1/clusters/${clusterId}/nodes/${encodeURIComponent(name)}/uncordon`)
}

export async function drainNode(
  clusterId: number,
  name: string,
  params: { force?: boolean; timeout_seconds?: number; ignore_daemonsets?: boolean } = {}
): Promise<void> {
  await http.post(`/api/v1/clusters/${clusterId}/nodes/${encodeURIComponent(name)}/drain`, null, { params })
}

export async function deleteNode(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/nodes/${encodeURIComponent(name)}`)
}
