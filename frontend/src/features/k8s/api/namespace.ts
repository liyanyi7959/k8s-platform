// k8s 命名空间相关 API
import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export type NamespaceResourceSummaryItem = {
  key: string
  label: string
  count: number
}

export type NamespaceResourcesSummary = {
  namespace: string
  total: number
  items: NamespaceResourceSummaryItem[]
}

export async function listNamespaces(
  clusterId: number,
  params: { sort_by?: string; order?: 'asc' | 'desc' } = {},
  options: { signal?: AbortSignal } = {}
): Promise<{ list: Array<{ metadata: { name: string } }> }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/namespaces`, { params, signal: options.signal })) as ApiResponse<{
    list: Array<{ metadata: { name: string } }>
  }>
  return resp.data
}

export async function createNamespace(clusterId: number, name: string, labels?: Record<string, string>): Promise<void> {
  await http.post(`/api/v1/clusters/${clusterId}/namespaces`, { name, labels })
}

export async function deleteNamespace(clusterId: number, ns: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/namespaces/${encodeURIComponent(ns)}`)
}

export async function getNamespaceResourcesSummary(clusterId: number, ns: string): Promise<NamespaceResourcesSummary> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/namespaces/${encodeURIComponent(ns)}/resources-summary`)) as ApiResponse<NamespaceResourcesSummary>
  return resp.data
}

export async function getNamespaceYaml(clusterId: number, ns: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/namespaces/${encodeURIComponent(ns)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}
