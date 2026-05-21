import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

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

export function getUsers(params: UserListParams) {
  return http.get<ApiResponse<UserListResult>>('/users', { params })
}

export function createUser(data: CreateUserReq) {
  return http.post<ApiResponse<{ id: number }>>('/users', data)
}

export function updateUser(id: number, data: UpdateUserReq) {
  return http.put<ApiResponse<null>>(`/users/${id}`, data)
}

export function deleteUser(id: number) {
  return http.delete<ApiResponse<null>>(`/users/${id}`)
}

export function resetPassword(id: number, password: string) {
  return http.post<ApiResponse<null>>(`/users/${id}/reset-password`, { password })
}

// ─── 角色 ───

export interface RoleItem {
  id: number
  name: string
  description: string
  permissions: string[]
  user_count: number
}

export interface CreateRoleReq {
  name: string
  description: string
  permissions: string[]
}

export interface UpdateRoleReq {
  description?: string
  permissions?: string[]
}

export function getRoles() {
  return http.get<ApiResponse<RoleItem[]>>('/roles')
}

export function createRole(data: CreateRoleReq) {
  return http.post<ApiResponse<{ id: number }>>('/roles', data)
}

export function updateRole(id: number, data: UpdateRoleReq) {
  return http.put<ApiResponse<null>>(`/roles/${id}`, data)
}

export function deleteRole(id: number) {
  return http.delete<ApiResponse<null>>(`/roles/${id}`)
}

// ─── 权限 ───

export interface PermissionItem {
  id: number
  code: string
  description: string
}

export function getPermissions() {
  return http.get<ApiResponse<PermissionItem[]>>('/permissions')
}
