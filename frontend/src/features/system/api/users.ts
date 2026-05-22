import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

async function unwrap<T>(request: Promise<unknown>): Promise<T> {
  const resp = (await request) as ApiResponse<T>
  return resp.data
}

// ─── 用户 ───

export interface UserItem {
  id: number
  username: string
  status: string
  roles: string[]
  created_at: string
}

export interface UserListResult {
  total: number
  items: UserItem[]
}

export interface UserListParams {
  page?: number
  page_size?: number
  keyword?: string
  status?: string
}

export interface CreateUserReq {
  username: string
  password: string
  role_ids: number[]
}

export interface UpdateUserReq {
  status?: string
  role_ids?: number[]
}

export function getUsers(params: UserListParams = {}) {
  return unwrap<UserListResult>(http.get('/api/v1/users', { params }))
}

export function createUser(data: CreateUserReq) {
  return unwrap<{ id: number }>(http.post('/api/v1/users', data))
}

export function updateUser(id: number, data: UpdateUserReq) {
  return unwrap<null>(http.put(`/api/v1/users/${id}`, data))
}

export function deleteUser(id: number) {
  return unwrap<null>(http.delete(`/api/v1/users/${id}`))
}

export function resetPassword(id: number, password: string) {
  return unwrap<null>(http.post(`/api/v1/users/${id}/reset-password`, { password }))
}

// ─── 角色 ───

interface RawRoleItem {
  id: number
  name: string
  description?: string
  desc?: string
  permissions?: string[]
  namespace_scope?: RawRoleNamespaceScope | null
  user_count?: number
  builtin?: boolean
  created_at?: string
}

interface RawRoleNamespaceScope {
  cluster_id?: number
  cluster_name?: string
  namespaces?: string[]
}

export interface RoleNamespaceScope {
  cluster_id: number
  cluster_name: string
  namespaces: string[]
}

export interface RoleNamespaceScopePayload {
  cluster_id: number
  namespaces: string[]
}

export interface RoleItem {
  id: number
  name: string
  description: string
  permissions: string[]
  namespace_scope: RoleNamespaceScope | null
  user_count: number
  builtin: boolean
  created_at: string
}

export interface CreateRoleReq {
  name: string
  description: string
  permissions: string[]
  namespace_scope?: RoleNamespaceScopePayload | null
}

export interface UpdateRoleReq {
  description?: string
  permissions?: string[]
  namespace_scope?: RoleNamespaceScopePayload | null
}

function normalizeRoleNamespaceScope(raw?: RawRoleNamespaceScope | null): RoleNamespaceScope | null {
  if (!raw) {
    return null
  }
  const clusterId = Number(raw.cluster_id ?? 0)
  const namespaces = Array.isArray(raw.namespaces) ? raw.namespaces.map((item) => String(item)).filter(Boolean) : []
  if (!clusterId || namespaces.length === 0) {
    return null
  }
  return {
    cluster_id: clusterId,
    cluster_name: String(raw.cluster_name ?? ''),
    namespaces
  }
}

function normalizeRole(raw: RawRoleItem): RoleItem {
  return {
    id: Number(raw.id ?? 0),
    name: String(raw.name ?? ''),
    description: String(raw.description ?? raw.desc ?? ''),
    permissions: Array.isArray(raw.permissions) ? raw.permissions.map((item) => String(item)) : [],
    namespace_scope: normalizeRoleNamespaceScope(raw.namespace_scope),
    user_count: Number(raw.user_count ?? 0),
    builtin: Boolean(raw.builtin),
    created_at: String(raw.created_at ?? '')
  }
}

export async function getRoles(): Promise<RoleItem[]> {
  const items = await unwrap<RawRoleItem[]>(http.get('/api/v1/roles'))
  return Array.isArray(items) ? items.map(normalizeRole) : []
}

export function createRole(data: CreateRoleReq) {
  return unwrap<{ id: number }>(http.post('/api/v1/roles', data))
}

export function updateRole(id: number, data: UpdateRoleReq) {
  return unwrap<null>(http.put(`/api/v1/roles/${id}`, data))
}

export function deleteRole(id: number) {
  return unwrap<null>(http.delete(`/api/v1/roles/${id}`))
}

// ─── 权限 ───

interface RawPermissionItem {
  id: number
  code: string
  description?: string
  desc?: string
  category?: string
  category_label?: string
  builtin?: boolean
}

export interface PermissionItem {
  id: number
  code: string
  description: string
  category: string
  category_label: string
  builtin: boolean
}

function normalizePermission(raw: RawPermissionItem): PermissionItem {
  return {
    id: Number(raw.id ?? 0),
    code: String(raw.code ?? ''),
    description: String(raw.description ?? raw.desc ?? raw.code ?? ''),
    category: String(raw.category ?? 'custom'),
    category_label: String(raw.category_label ?? '自定义权限'),
    builtin: Boolean(raw.builtin)
  }
}

export async function getPermissions(): Promise<PermissionItem[]> {
  const items = await unwrap<RawPermissionItem[]>(http.get('/api/v1/permissions'))
  return Array.isArray(items) ? items.map(normalizePermission) : []
}
