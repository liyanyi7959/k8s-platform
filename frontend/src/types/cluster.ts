/**
 * 集群管理类型 — 与后端 ClusterItem / ClusterDetail 对齐
 */
import type { PageResult } from './common'

/** 集群（对应后端 ClusterItem） */
export interface Cluster {
  id: number
  name: string
  type?: string
  status: string
  k8sVersion?: string
  nodeCount: number
  createdAt?: string
  updatedAt?: string
  lastHealthAt?: string
}

export interface ClusterListParams {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
}

export type ClusterListResponse = PageResult<Cluster>

export interface CreateClusterRequest {
  name: string
  kubeconfig: string
  description?: string
}

export interface UpdateClusterRequest {
  name?: string
  kubeconfig?: string
}

/** 集群健康状态（对应后端 CheckClusterHealthResp） */
export interface ClusterHealth {
  apiOk: boolean
  nodeReady: number
  nodeTotal: number
  checkedAt: string
  status: string
  lastHealthAt?: string
}
