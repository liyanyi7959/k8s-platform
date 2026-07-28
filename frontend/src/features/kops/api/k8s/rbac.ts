/**
 * 访问控制资源 API：ServiceAccount / Role / RoleBinding / ClusterRole / ClusterRoleBinding / ResourceQuota
 */
import { request } from '@umijs/max'
import type {
  ServiceAccountList,
  RBACRoleList,
  RBACRoleBindingList,
  ResourceQuotaList,
} from '@/features/kops/types'
import { mapServiceAccount, mapRBACRole, mapRBACRoleBinding, mapResourceQuota, extractMappedList } from './shared'

// ==================== ServiceAccount ====================

/** 获取 ServiceAccount 列表 */
export function listServiceAccounts(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<ServiceAccountList> {
  return request(`/api/v1/clusters/${clusterId}/serviceaccounts`, { params: { namespace }, signal }).then(extractMappedList(mapServiceAccount))
}

/** 删除 ServiceAccount */
export function deleteServiceAccount(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/serviceaccounts/${namespace}/${name}`, { method: 'DELETE' })
}

// ==================== Role / ClusterRole ====================

/** 获取 Role 列表 */
export function listRoles(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<RBACRoleList> {
  return request(`/api/v1/clusters/${clusterId}/roles`, { params: { namespace }, signal }).then(extractMappedList(mapRBACRole))
}

/** 获取 ClusterRole 列表 */
export function listClusterRoles(clusterId: number, signal?: AbortSignal): Promise<RBACRoleList> {
  return request(`/api/v1/clusters/${clusterId}/clusterroles`, { signal }).then(extractMappedList(mapRBACRole))
}

/** 删除 Role */
export function deleteRole(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/roles/${namespace}/${name}`, { method: 'DELETE' })
}

/** 删除 ClusterRole */
export function deleteClusterRole(clusterId: number, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/clusterroles/${name}`, { method: 'DELETE' })
}

// ==================== RoleBinding / ClusterRoleBinding ====================

/** 获取 RoleBinding 列表 */
export function listRoleBindings(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<RBACRoleBindingList> {
  return request(`/api/v1/clusters/${clusterId}/rolebindings`, { params: { namespace }, signal }).then(extractMappedList(mapRBACRoleBinding))
}

/** 获取 ClusterRoleBinding 列表 */
export function listClusterRoleBindings(clusterId: number, signal?: AbortSignal): Promise<RBACRoleBindingList> {
  return request(`/api/v1/clusters/${clusterId}/clusterrolebindings`, { signal }).then(extractMappedList(mapRBACRoleBinding))
}

/** 删除 RoleBinding */
export function deleteRoleBinding(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/rolebindings/${namespace}/${name}`, { method: 'DELETE' })
}

/** 删除 ClusterRoleBinding */
export function deleteClusterRoleBinding(clusterId: number, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/clusterrolebindings/${name}`, { method: 'DELETE' })
}

// ==================== ResourceQuota ====================

/** 获取 ResourceQuota 列表 */
export function listResourceQuotas(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<ResourceQuotaList> {
  return request(`/api/v1/clusters/${clusterId}/resourcequotas`, { params: { namespace }, signal }).then(extractMappedList(mapResourceQuota))
}

/** 删除 ResourceQuota */
export function deleteResourceQuota(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v1/clusters/${clusterId}/resourcequotas/${namespace}/${name}`, { method: 'DELETE' })
}
