import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export async function listPDBs(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pdbs`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deletePDB(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/pdbs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getPDBYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pdbs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editPDB(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/pdbs/edit`, req)
}

export async function listRoles(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/roles`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteRole(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/roles/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getRoleYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/roles/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editRole(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/roles/edit`, req)
}

export async function listClusterRoles(clusterId: number, params: { sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/clusterroles`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteClusterRole(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/clusterroles/${encodeURIComponent(name)}`)
}

export async function getClusterRoleYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/clusterroles/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editClusterRole(clusterId: number, req: { yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/clusterroles/edit`, req)
}

export async function listRoleBindings(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/rolebindings`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteRoleBinding(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/rolebindings/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getRoleBindingYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/rolebindings/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editRoleBinding(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/rolebindings/edit`, req)
}

export async function listClusterRoleBindings(clusterId: number, params: { sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/clusterrolebindings`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteClusterRoleBinding(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/clusterrolebindings/${encodeURIComponent(name)}`)
}

export async function getClusterRoleBindingYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/clusterrolebindings/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editClusterRoleBinding(clusterId: number, req: { yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/clusterrolebindings/edit`, req)
}
