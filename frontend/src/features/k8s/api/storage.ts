// k8s 存储相关 API（PVC / PV / StorageClass / VolumeSnapshot）
import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export type K8sResourceSupport = {
  replicasets?: boolean
  clusterroles?: boolean
  endpoints?: boolean
  endpointslices?: boolean
  networkpolicies?: boolean
  volumeattachments?: boolean
  resourcequotas?: boolean
  limitranges?: boolean
  customresourcedefinitions?: boolean
  apiservices?: boolean
  priorityclasses?: boolean
  validatingwebhookconfigurations?: boolean
  mutatingwebhookconfigurations?: boolean
  leases?: boolean
  volumesnapshots?: boolean
  volumesnapshotclasses?: boolean
  volumesnapshotcontents?: boolean
}

// ── PVC ──────────────────────────────────────────────────

export async function listPVCs(
  clusterId: number,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pvcs`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getPVCYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pvcs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deletePVC(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/pvcs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export type CreatePVCRequest = {
  namespace: string
  name: string
  storage_class?: string
  access_modes: string[]
  capacity: string
}

export async function createPVC(clusterId: number, req: CreatePVCRequest): Promise<void> {
  await http.post(`/api/v1/clusters/${clusterId}/pvcs`, req)
}

// ── PV ───────────────────────────────────────────────────

export async function listPVs(clusterId: number, params: { sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pvs`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getPVYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pvs/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deletePV(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/pvs/${encodeURIComponent(name)}`)
}

// ── StorageClass ─────────────────────────────────────────

export async function listStorageClasses(
  clusterId: number,
  params: { sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/storageclasses`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getStorageClassYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/storageclasses/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deleteStorageClass(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/storageclasses/${encodeURIComponent(name)}`)
}

export async function editStorageClass(clusterId: number, req: { yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/storageclasses/edit`, req)
}

export async function listCSIStorageCapacities(
  clusterId: number,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/csistoragecapacities`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getCSIStorageCapacityYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/csistoragecapacities/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deleteCSIStorageCapacity(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/csistoragecapacities/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function editCSIStorageCapacity(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/csistoragecapacities/edit`, req)
}

export async function getResourceSupport(clusterId: number): Promise<K8sResourceSupport> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/resource-support`)) as ApiResponse<K8sResourceSupport>
  return resp.data
}

export async function getStorageSnapshotSupport(clusterId: number): Promise<K8sResourceSupport> {
  return getResourceSupport(clusterId)
}

// ── VolumeSnapshot ───────────────────────────────────────

export async function listVolumeSnapshots(
  clusterId: number,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/volumesnapshots`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getVolumeSnapshotYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/volumesnapshots/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deleteVolumeSnapshot(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/volumesnapshots/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function editVolumeSnapshot(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/volumesnapshots/edit`, req)
}

// ── VolumeSnapshotClass ──────────────────────────────────

export async function listVolumeSnapshotClasses(
  clusterId: number,
  params: { sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/volumesnapshotclasses`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getVolumeSnapshotClassYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/volumesnapshotclasses/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deleteVolumeSnapshotClass(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/volumesnapshotclasses/${encodeURIComponent(name)}`)
}

export async function editVolumeSnapshotClass(clusterId: number, req: { yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/volumesnapshotclasses/edit`, req)
}

// ── VolumeSnapshotContent ────────────────────────────────

export async function listVolumeSnapshotContents(
  clusterId: number,
  params: { sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/volumesnapshotcontents`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getVolumeSnapshotContentYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/volumesnapshotcontents/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deleteVolumeSnapshotContent(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/volumesnapshotcontents/${encodeURIComponent(name)}`)
}

export async function editVolumeSnapshotContent(clusterId: number, req: { yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/volumesnapshotcontents/edit`, req)
}
