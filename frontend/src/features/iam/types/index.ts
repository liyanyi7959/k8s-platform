import type { PageResult } from '@/shared/types/common'

export interface Permission {
  id: number
  code: string
  name: string
  description?: string
  category?: string
  categoryLabel?: string
  builtin?: boolean
}

export interface Role {
  id: number
  name: string
  code: string
  description?: string
  permissions: string[]
  userCount?: number
  builtin?: boolean
  createdAt: string
}

export interface User {
  id: number
  username: string
  nickname: string
  email: string
  enabled: boolean
  status?: string
  roles: Role[]
  createdAt: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user?: User
}

export interface UserListParams {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
  roleId?: number
}

export type UserListResponse = PageResult<User>

export interface CreateUserRequest {
  username: string
  nickname?: string
  email?: string
  password: string
  roleIds: number[]
  enabled?: boolean
}

export interface UpdateUserRequest {
  nickname?: string
  email?: string
  enabled?: boolean
  roleIds?: number[]
}

export interface ResetPasswordRequest {
  password: string
}

export interface RoleListParams {
  page?: number
  pageSize?: number
}

export type RoleListResponse = PageResult<Role>

export interface CreateRoleRequest {
  name: string
  code: string
  description?: string
  permissions?: string[]
}

export interface UpdateRoleRequest {
  name?: string
  code?: string
  description?: string
  permissions?: string[]
}
