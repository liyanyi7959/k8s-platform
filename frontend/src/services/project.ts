/**
 * 项目管理 API
 */
import { request } from '@umijs/max'

export interface Project {
  id: number
  name: string
  description: string
  cluster_id: number
  namespaces: string
  quota_cpu: string
  quota_memory: string
  quota_pods: string
  creator_id: number
  created_at: string
  updated_at: string
}

export interface ProjectListResponse {
  list: Project[]
  total: number
  page: number
  page_size: number
}

/** 获取项目列表 */
export function listProjects(
  params?: { page?: number; page_size?: number },
  signal?: AbortSignal,
): Promise<ProjectListResponse> {
  return request('/api/v1/projects', {
    params: { page: 1, page_size: 100, ...params },
    signal,
  })
}

/** 创建项目 */
export function createProject(data: Partial<Project>) {
  return request('/api/v1/projects', { method: 'POST', data })
}

/** 更新项目 */
export function updateProject(id: number, data: Partial<Project>) {
  return request(`/api/v1/projects/${id}`, { method: 'PUT', data })
}

/** 删除项目 */
export function deleteProject(id: number) {
  return request(`/api/v1/projects/${id}`, { method: 'DELETE' })
}

/** 命名空间资源统计 */
export interface NamespaceResources {
  pods: number
  deployments: number
  services: number
  configmaps: number
  total: number
}

/** 项目资源统计响应 */
export interface ProjectResourcesResponse {
  cluster_id: number
  namespaces: Record<string, NamespaceResources>
}

/** 获取项目下所有命名空间的资源统计 */
export function getProjectResources(id: number, signal?: AbortSignal) {
  return request<ProjectResourcesResponse>(`/api/v1/projects/${id}/resources`, {
    signal,
  })
}

/** 分配命名空间到项目 */
export function assignNamespaces(id: number, namespaces: string[]) {
  return request(`/api/v1/projects/${id}/namespaces`, {
    method: 'PUT',
    data: { namespaces },
  })
}
