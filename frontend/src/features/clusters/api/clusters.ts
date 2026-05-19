import { http } from '@/shared/http/http'
import type { ApiResponse, PageResult } from '@/shared/types/api'

export interface ClusterItem {
  id: number
  name: string
  type: 'imported' | 'created'
  status: 'active' | 'disabled' | 'creating' | 'degraded' | 'deleting'
  k8s_version?: string
  node_count?: number
  created_at?: string
}

export interface ClusterDetail extends ClusterItem {
  last_health_at?: string
  health?: { api_ok: boolean; node_ready: number; node_total: number }
}

export async function listClusters(params: {
  page?: number
  page_size?: number
  keyword?: string
  status?: string
  type?: string
  sort_by?: string
  order?: 'asc' | 'desc'
} = {}): Promise<PageResult<ClusterItem>> {
  const resp = (await http.get('/api/v1/clusters', { params })) as ApiResponse<PageResult<ClusterItem>>
  return resp.data
}

export async function importCluster(req: { name: string; kubeconfig: string }): Promise<{ cluster_id: number }> {
  const resp = (await http.post('/api/v1/clusters/import', req)) as ApiResponse<{ cluster_id: number }>
  return resp.data
}

export async function getCluster(id: number): Promise<ClusterDetail> {
  const resp = (await http.get(`/api/v1/clusters/${id}`)) as ApiResponse<ClusterDetail>
  return resp.data
}

export async function patchCluster(id: number, req: { name?: string; kubeconfig?: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${id}`, req)
}

export async function deleteCluster(id: number): Promise<void> {
  await http.delete(`/api/v1/clusters/${id}`)
}

export async function checkClusterHealth(id: number): Promise<{
  api_ok: boolean
  node_ready: number
  node_total: number
  checked_at: string
  status?: ClusterItem['status']
  last_health_at?: string
}> {
  const resp = (await http.post(`/api/v1/clusters/${id}/check-health`)) as ApiResponse<{
    api_ok: boolean
    node_ready: number
    node_total: number
    checked_at: string
    status?: ClusterItem['status']
    last_health_at?: string
  }>
  return resp.data
}

