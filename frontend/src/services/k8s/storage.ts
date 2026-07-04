/**
 * 存储资源 API：PersistentVolume / PersistentVolumeClaim / StorageClass
 */
import { request } from '@umijs/max'
import type {
  PersistentVolumeList,
  PersistentVolumeClaimList,
  StorageClassList,
} from '@/types'
import { mapPV, mapPVC, mapStorageClass, extractMappedList } from './shared'

// ==================== PersistentVolume ====================

/** 获取 PV 列表 */
export function listPersistentVolumes(clusterId: number, signal?: AbortSignal): Promise<PersistentVolumeList> {
  return request(`/api/v1/clusters/${clusterId}/pvs`, { signal }).then(extractMappedList(mapPV))
}

/** 删除 PV */
export function deletePersistentVolume(clusterId: number, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/pvs/${name}`, { method: 'DELETE' })
}

/** 创建 PV */
export function createPV(clusterId: number, data: any): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/pv`, { method: 'POST', data })
}

// ==================== PersistentVolumeClaim ====================

/** 获取 PVC 列表 */
export function listPersistentVolumeClaims(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<PersistentVolumeClaimList> {
  return request(`/api/v1/clusters/${clusterId}/pvcs`, { params: { namespace }, signal }).then(extractMappedList(mapPVC))
}

/** 删除 PVC */
export function deletePersistentVolumeClaim(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/pvcs/${namespace}/${name}`, { method: 'DELETE' })
}

/** 创建 PVC */
export function createPVC(clusterId: number, namespace: string, data: any): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/pvc`, { method: 'POST', data, params: { namespace } })
}

// ==================== StorageClass ====================

/** 获取 StorageClass 列表 */
export function listStorageClasses(clusterId: number, signal?: AbortSignal): Promise<StorageClassList> {
  return request(`/api/v1/clusters/${clusterId}/storageclasses`, { signal }).then(extractMappedList(mapStorageClass))
}

/** 删除 StorageClass */
export function deleteStorageClass(clusterId: number, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/storageclasses/${name}`, { method: 'DELETE' })
}
