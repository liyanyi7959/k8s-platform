/**
 * 系统管理 API
 */
import { request } from '@umijs/max'
import type {
  AuditLog,
  AuditLogListParams,
  AuditLogListResponse,
  CreateRoleRequest,
  CreateUserRequest,
  Permission,
  ResetPasswordRequest,
  Role,
  RoleListParams,
  RoleListResponse,
  SystemSettings,
  UpdateRoleRequest,
  UpdateUserRequest,
  User,
  UserListParams,
  UserListResponse,
} from '@/types'

interface BackendAuditLog {
  id: number
  user_id?: number
  username?: string
  action?: string
  resource?: string
  resource_name?: string
  namespace?: string
  cluster_id?: number
  detail?: string
  status_code?: number
  path?: string
  request_id?: string
  client_ip?: string
  created_at?: string
}

interface BackendAuditListResponse {
  items: BackendAuditLog[]
  total: number
  page: number
  pageSize: number
}

function normalizeAuditLog(raw: BackendAuditLog): AuditLog {
  return {
    id: raw.id,
    userId: raw.user_id,
    username: raw.username || '未知用户',
    action: raw.action || 'unknown',
    resource: raw.resource_name || raw.resource || '—',
    resourceName: raw.resource_name,
    namespace: raw.namespace,
    clusterId: raw.cluster_id,
    detail: raw.detail || '—',
    statusCode: raw.status_code,
    path: raw.path,
    requestId: raw.request_id,
    ip: raw.client_ip || '—',
    timestamp: raw.created_at || new Date().toISOString(),
  }
}

function normalizeAuditResponse(raw: BackendAuditListResponse): AuditLogListResponse {
  return {
    items: (raw.items || []).map(normalizeAuditLog),
    total: raw.total || 0,
    page: raw.page || 1,
    pageSize: raw.pageSize || 20,
  }
}

export function listUsers(params?: UserListParams): Promise<UserListResponse> {
  const payload: Record<string, unknown> = {}
  if (params?.page) {
    payload.page = params.page
  }
  if (params?.pageSize) {
    payload.page_size = params.pageSize
  }
  if (params?.keyword) {
    payload.keyword = params.keyword
  }
  if (params?.status) {
    payload.status = params.status
  }
  if (params?.roleId) {
    payload.role_id = params.roleId
  }
  return request('/api/v1/users', { params: payload })
}

export function getUserById(id: number): Promise<User> {
  return request(`/api/v1/users/${id}`)
}

export function createUser(data: CreateUserRequest): Promise<User> {
  return request('/api/v1/users', { method: 'POST', data })
}

export function updateUser(id: number, data: UpdateUserRequest): Promise<User> {
  return request(`/api/v1/users/${id}`, { method: 'PUT', data })
}

export function deleteUser(id: number): Promise<void> {
  return request(`/api/v1/users/${id}`, { method: 'DELETE' })
}

export function resetPassword(id: number, data: ResetPasswordRequest): Promise<void> {
  return request(`/api/v1/users/${id}/reset-password`, { method: 'POST', data })
}

export function listRoles(params?: RoleListParams): Promise<RoleListResponse> {
  const payload: Record<string, unknown> = {}
  if (params?.page) {
    payload.page = params.page
  }
  if (params?.pageSize) {
    payload.page_size = params.pageSize
  }
  return request('/api/v1/roles', { params: payload })
}

export async function listAllRoles(): Promise<Role[]> {
  const response = await request<RoleListResponse>('/api/v1/roles')
  return response?.items || []
}

export function listPermissions(): Promise<Permission[]> {
  return request('/api/v1/permissions')
}

export function createRole(data: CreateRoleRequest): Promise<Role> {
  return request('/api/v1/roles', { method: 'POST', data })
}

export function updateRole(id: number, data: UpdateRoleRequest): Promise<Role> {
  return request(`/api/v1/roles/${id}`, { method: 'PUT', data })
}

export function deleteRole(id: number): Promise<void> {
  return request(`/api/v1/roles/${id}`, { method: 'DELETE' })
}

export async function listAuditLogs(params?: AuditLogListParams): Promise<AuditLogListResponse> {
  const payload: Record<string, unknown> = {}
  if (params?.page) {
    payload.page = params.page
  }
  if (params?.pageSize) {
    payload.page_size = params.pageSize
  }
  if (params?.keyword) {
    payload.keyword = params.keyword
  }
  if (params?.username) {
    payload.username = params.username
  }
  if (params?.action) {
    payload.action = params.action
  }
  if (params?.resource) {
    payload.resource = params.resource
  }
  if (params?.clusterId) {
    payload.cluster_id = params.clusterId
  }
  if (params?.status) {
    payload.status = params.status
  }
  if (params?.startTime) {
    payload.start_time = params.startTime
  }
  if (params?.endTime) {
    payload.end_time = params.endTime
  }
  const response = await request<BackendAuditListResponse>('/api/v1/audit-logs', { params: payload })
  return normalizeAuditResponse(response)
}

export function getSystemSettings(): Promise<SystemSettings> {
  return request('/api/v1/system/settings')
}

export function updateSystemSettings(data: SystemSettings): Promise<SystemSettings> {
  return request('/api/v1/system/settings', { method: 'PUT', data })
}
