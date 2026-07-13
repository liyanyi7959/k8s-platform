/**
 * 集群管理 API
 */
import { request } from '@umijs/max'
import type {
  Cluster,
  ClusterListParams,
  ClusterListResponse,
  CreateClusterRequest,
  UpdateClusterRequest,
  ClusterHealth,
} from '@/types'

/** 后端 ClusterItem → 前端 Cluster 字段映射 */
function mapCluster(raw: any): Cluster {
  return {
    id: raw?.id || 0,
    name: raw?.name || '',
    type: raw?.type || '',
    status: raw?.status || '',
    k8sVersion: raw?.k8s_version || '',
    nodeCount: raw?.node_count || 0,
    createdAt: raw?.created_at || '',
    updatedAt: raw?.updated_at || '',
    lastHealthAt: raw?.last_health_at || '',
  }
}

/** 获取集群列表 */
export async function listClusters(
  params?: ClusterListParams,
  signal?: AbortSignal,
): Promise<ClusterListResponse> {
  const res = await request<any>('/api/v1/clusters', {
    params: {
      page: params?.page,
      page_size: params?.pageSize,
      keyword: params?.keyword,
      status: params?.status,
    },
    signal,
  })
  // 后端返回 { list, total, page, page_size }，前端期望 { items, total, page, pageSize }
  const list = Array.isArray(res?.list) ? res.list : []
  return {
    items: list.map(mapCluster),
    total: res?.total || 0,
    page: res?.page || 1,
    pageSize: res?.page_size || 10,
  }
}

/** 获取集群详情 */
export async function getClusterById(id: number, signal?: AbortSignal): Promise<Cluster> {
  const res = await request<any>(`/api/v1/clusters/${id}`, { signal })
  return mapCluster(res)
}

/** 导入集群 */
export async function importCluster(data: CreateClusterRequest, file?: File): Promise<{ clusterId: number }> {
  const formData = new FormData()
  formData.append('name', data.name)
  formData.append('description', data.description || '')
  if (file) formData.append('file', file)
  else formData.append('kubeconfig', data.kubeconfig)
  const res = await request<any>('/api/v1/clusters/import', { method: 'POST', data: formData })
  return { clusterId: res?.cluster_id || 0 }
}

/** 更新集群 (PATCH) */
export async function updateCluster(id: number, data: UpdateClusterRequest): Promise<void> {
  return request(`/api/v1/clusters/${id}`, { method: 'PATCH', data })
}

/** 删除集群 */
export async function deleteCluster(id: number): Promise<void> {
  return request(`/api/v1/clusters/${id}`, { method: 'DELETE' })
}

/** 检查集群健康状态 */
export async function checkClusterConnection(id: number, signal?: AbortSignal): Promise<ClusterHealth> {
  const res = await request<any>(`/api/v1/clusters/${id}/check-health`, { method: 'POST', signal })
  return {
    apiOk: res?.api_ok ?? false,
    nodeReady: res?.node_ready || 0,
    nodeTotal: res?.node_total || 0,
    checkedAt: res?.checked_at || '',
    status: res?.status || 'unknown',
    lastHealthAt: res?.last_health_at || '',
  }
}
