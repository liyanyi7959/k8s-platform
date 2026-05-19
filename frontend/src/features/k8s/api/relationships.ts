import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export async function listReplicaSets(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/replicasets`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getReplicaSetYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/replicasets/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editReplicaSet(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/replicasets/edit`, req)
}

export async function deleteReplicaSet(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/replicasets/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function listVolumeAttachments(clusterId: number, params: { sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/volumeattachments`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getVolumeAttachmentYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/volumeattachments/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editVolumeAttachment(clusterId: number, req: { yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/volumeattachments/edit`, req)
}

export async function deleteVolumeAttachment(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/volumeattachments/${encodeURIComponent(name)}`)
}
