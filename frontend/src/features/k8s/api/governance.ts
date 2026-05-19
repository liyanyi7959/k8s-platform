import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

type NamespacedListParams = { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' }

async function listNamespaced(clusterId: number, resource: string, params: NamespacedListParams = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/${resource}`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

async function getYaml(clusterId: number, resource: string, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/${resource}/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

async function edit(clusterId: number, resource: string, req: { yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/${resource}/edit`, req)
}

async function remove(clusterId: number, resource: string, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/${resource}/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export const listNetworkPolicies = (clusterId: number, params: NamespacedListParams = {}) => listNamespaced(clusterId, 'networkpolicies', params)
export const getNetworkPolicyYaml = (clusterId: number, ns: string, name: string) => getYaml(clusterId, 'networkpolicies', ns, name)
export const editNetworkPolicy = (clusterId: number, req: { yaml: string }) => edit(clusterId, 'networkpolicies', req)
export const deleteNetworkPolicy = (clusterId: number, ns: string, name: string) => remove(clusterId, 'networkpolicies', ns, name)

export const listResourceQuotas = (clusterId: number, params: NamespacedListParams = {}) => listNamespaced(clusterId, 'resourcequotas', params)
export const getResourceQuotaYaml = (clusterId: number, ns: string, name: string) => getYaml(clusterId, 'resourcequotas', ns, name)
export const editResourceQuota = (clusterId: number, req: { yaml: string }) => edit(clusterId, 'resourcequotas', req)
export const deleteResourceQuota = (clusterId: number, ns: string, name: string) => remove(clusterId, 'resourcequotas', ns, name)

export const listLimitRanges = (clusterId: number, params: NamespacedListParams = {}) => listNamespaced(clusterId, 'limitranges', params)
export const getLimitRangeYaml = (clusterId: number, ns: string, name: string) => getYaml(clusterId, 'limitranges', ns, name)
export const editLimitRange = (clusterId: number, req: { yaml: string }) => edit(clusterId, 'limitranges', req)
export const deleteLimitRange = (clusterId: number, ns: string, name: string) => remove(clusterId, 'limitranges', ns, name)
