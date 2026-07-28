/**
 * 应用商店模板 API
 */
import { request } from '@umijs/max'

export interface AppTemplate {
  id: number
  name: string
  display_name: string
  description: string
  category: string
  icon: string
  template: string
  variables: string
  is_builtin: boolean
  deploy_type?: string
  helm_repo_name?: string
  helm_repo_url?: string
  helm_chart_version?: string
  helm_values_yaml?: string
  created_at: string
  updated_at: string
}

export interface AppTemplateListResponse {
  list: AppTemplate[]
  total: number
  page: number
  page_size: number
}

/** 获取应用模板列表 */
export function listAppTemplates(
  params?: { category?: string; page?: number; page_size?: number },
  signal?: AbortSignal,
): Promise<AppTemplateListResponse> {
  return request('/api/v1/app-templates', {
    params: { page: 1, page_size: 100, ...params },
    signal,
  })
}

/** 获取应用模板详情 */
export function getAppTemplate(id: number, signal?: AbortSignal): Promise<AppTemplate> {
  return request(`/api/v1/app-templates/${id}`, { signal })
}

/** 创建应用模板 */
export function createAppTemplate(data: Partial<AppTemplate>) {
  return request('/api/v1/app-templates', { method: 'POST', data })
}

/** 更新应用模板 */
export function updateAppTemplate(id: number, data: Partial<AppTemplate>) {
  return request(`/api/v1/app-templates/${id}`, { method: 'PUT', data })
}

/** 删除应用模板 */
export function deleteAppTemplate(id: number) {
  return request(`/api/v1/app-templates/${id}`, { method: 'DELETE' })
}
