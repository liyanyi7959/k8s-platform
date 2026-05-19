import { http } from '@/shared/http/http'
import type { ApiResponse, PageResult } from '@/shared/types/api'

export type PermissionAuditSourceType = 'managed_cluster' | 'adhoc_kubeconfig'
export type PermissionAuditStatus = 'pending' | 'running' | 'success' | 'failed' | 'incomplete' | 'canceled'
export type PermissionAuditRiskLevel = 'critical' | 'high' | 'medium' | 'low'
export type PermissionAuditOwnershipClass = 'direct' | 'shared' | 'unrelated'
export type PermissionAuditPrivilegeClass = 'cluster_scoped' | 'runtime_high' | 'namespace_only_candidate' | 'shared_cluster_dependency' | ''

export interface PermissionAuditCreateRequest {
  mode?: 'full'
  include_runtime_rbac?: boolean
  include_ownership_detection?: boolean
  namespaces?: string[]
  label_selector?: string
  resource_allowlist?: string[]
}

export interface PermissionAuditSummary {
  total_resources?: number
  cluster_scoped_resources?: number
  namespaced_resources?: number
  high_privilege_workloads?: number
  namespace_only_candidate_workloads?: number
  unmapped_resources?: number
  ownership?: Partial<Record<PermissionAuditOwnershipClass, number>>
  risk?: Partial<Record<PermissionAuditRiskLevel, number>>
}

export interface PermissionAuditStats {
  risk?: Partial<Record<PermissionAuditRiskLevel, number>>
  blockers?: {
    deployment_blockers?: number
    shared_capabilities?: number
    partial_failures?: number
  }
}

export interface PermissionAuditListItem {
  id: number
  source_type: PermissionAuditSourceType
  cluster_id?: number
  cluster_name: string
  display_name: string
  status: PermissionAuditStatus
  task_id?: number
  summary: PermissionAuditSummary
  created_at: string
  created_by: number
}

export interface PermissionAuditDetail {
  id: number
  source_type: PermissionAuditSourceType
  cluster: { id?: number; name?: string }
  display_name: string
  status: PermissionAuditStatus
  task_id?: number
  request: PermissionAuditCreateRequest
  summary: PermissionAuditSummary
  stats: PermissionAuditStats
  error?: Record<string, any>
  created_at: string
  updated_at: string
  created_by: number
}

export interface PermissionAuditFindingItem {
  id: number
  finding_type: 'resource' | 'workload' | 'error'
  risk_level: PermissionAuditRiskLevel
  ownership_class: PermissionAuditOwnershipClass
  privilege_class: PermissionAuditPrivilegeClass
  namespace: string
  kind: string
  name: string
  deployment_blocker: boolean
  summary: string
  detail: Record<string, any>
}

export interface PermissionAuditCreateResult {
  audit_id: number
  task_id: number
  source_type: PermissionAuditSourceType
}

export interface PermissionAuditTaskLogsResult {
  task_id: number
  offset: number
  limit: number
  lines: string[]
  status: string
  can_cancel: boolean
}

export interface PermissionAuditComparePair {
  current?: PermissionAuditFindingItem
  baseline?: PermissionAuditFindingItem
}

export interface PermissionAuditCompareResult {
  audit_id: number
  baseline_audit_id: number
  baseline_label: string
  summary: {
    added_count: number
    removed_count: number
    changed_count: number
  }
  added: PermissionAuditFindingItem[]
  removed: PermissionAuditFindingItem[]
  changed: PermissionAuditComparePair[]
}

export async function createClusterPermissionAudit(clusterId: number, req: PermissionAuditCreateRequest): Promise<PermissionAuditCreateResult> {
  const resp = (await http.post(`/api/v1/clusters/${clusterId}/permission-audits`, req)) as ApiResponse<PermissionAuditCreateResult>
  return resp.data
}

export async function createAdhocPermissionAudit(req: { display_name: string; kubeconfig: string } & PermissionAuditCreateRequest): Promise<PermissionAuditCreateResult> {
  const resp = (await http.post('/api/v1/permission-audits/adhoc', req)) as ApiResponse<PermissionAuditCreateResult>
  return resp.data
}

export async function getLatestClusterPermissionAudit(clusterId: number): Promise<PermissionAuditDetail> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/permission-audits/latest`)) as ApiResponse<PermissionAuditDetail>
  return resp.data
}

export async function listPermissionAudits(params: {
  page?: number
  page_size?: number
  source_type?: PermissionAuditSourceType
  status?: PermissionAuditStatus
  risk_level?: PermissionAuditRiskLevel
  cluster_id?: number
  keyword?: string
  sort_by?: string
  order?: 'asc' | 'desc'
} = {}): Promise<PageResult<PermissionAuditListItem>> {
  const resp = (await http.get('/api/v1/permission-audits', { params })) as ApiResponse<PageResult<PermissionAuditListItem>>
  return resp.data
}

export async function getPermissionAudit(id: number): Promise<PermissionAuditDetail> {
  const resp = (await http.get(`/api/v1/permission-audits/${id}`)) as ApiResponse<PermissionAuditDetail>
  return resp.data
}

export async function getPermissionAuditLogs(id: number, params: { offset?: number; limit?: number } = {}): Promise<PermissionAuditTaskLogsResult> {
  const resp = (await http.get(`/api/v1/permission-audits/${id}/logs`, { params })) as ApiResponse<PermissionAuditTaskLogsResult>
  return resp.data
}

export async function cancelPermissionAudit(id: number): Promise<void> {
  await http.post(`/api/v1/permission-audits/${id}/cancel`)
}

export async function comparePermissionAudit(id: number, baselineId?: number): Promise<PermissionAuditCompareResult> {
  const resp = (await http.get(`/api/v1/permission-audits/${id}/compare`, { params: { baseline_id: baselineId } })) as ApiResponse<PermissionAuditCompareResult>
  return resp.data
}

export async function listPermissionAuditFindings(id: number, params: {
  page?: number
  page_size?: number
  finding_type?: string
  risk_level?: PermissionAuditRiskLevel
  ownership_class?: PermissionAuditOwnershipClass
  privilege_class?: PermissionAuditPrivilegeClass
  namespace?: string
  kind?: string
  deployment_blocker?: boolean
  keyword?: string
  sort_by?: string
  order?: 'asc' | 'desc'
} = {}): Promise<PageResult<PermissionAuditFindingItem>> {
  const resp = (await http.get(`/api/v1/permission-audits/${id}/findings`, { params })) as ApiResponse<PageResult<PermissionAuditFindingItem>>
  return resp.data
}

export interface RBACRecommendationResult {
  yaml_content: string
  service_account: string
  sa_namespace: string
  target_namespaces: string[]
}

export async function generateRBACRecommendation(clusterId: number, namespaces: string[]): Promise<RBACRecommendationResult> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/permission-audits/recommend-rbac`, {
    params: { namespaces: namespaces.join(',') }
  })) as ApiResponse<RBACRecommendationResult>
  return resp.data
}

// -------- 权限矩阵 --------

export interface RBACMatrixRow {
  api_group: string
  resources: string[]
  verbs: string[]
  scope: 'cluster' | 'namespace'
  label: string
}

export interface RBACMatrixRequest {
  service_account: string
  sa_namespace: string
  target_namespaces: string[]
  cluster_rows: RBACMatrixRow[]
  namespace_rows: RBACMatrixRow[]
}

export async function getDefaultRBACMatrix(clusterId: number, namespaces: string[]): Promise<RBACMatrixRequest> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/permission-audits/rbac-matrix/default`, {
    params: { namespaces: namespaces.join(',') }
  })) as ApiResponse<RBACMatrixRequest>
  return resp.data
}

export async function buildRBACFromMatrix(clusterId: number, req: RBACMatrixRequest): Promise<{ yaml_content: string }> {
  const resp = (await http.post(`/api/v1/clusters/${clusterId}/permission-audits/rbac-matrix/yaml`, req)) as ApiResponse<{ yaml_content: string }>
  return resp.data
}
