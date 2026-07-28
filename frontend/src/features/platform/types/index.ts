/**
 * 系统管理类型
 */
import type { PageResult } from '@/shared/types/common'

/** 权限 */
export interface Permission {
  id: number
  code: string
  name: string
  description?: string
  category?: string
  categoryLabel?: string
  builtin?: boolean
}

/** 角色 */
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

/** 用户 */
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

/** 登录请求 */
export interface LoginRequest {
  username: string
  password: string
}

/** 登录响应 */
export interface LoginResponse {
  token: string
  user?: User
}

/** 用户列表参数 */
export interface UserListParams {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
  roleId?: number
}

export type UserListResponse = PageResult<User>

/** 创建用户请求 */
export interface CreateUserRequest {
  username: string
  nickname?: string
  email?: string
  password: string
  roleIds: number[]
  enabled?: boolean
}

/** 更新用户请求 */
export interface UpdateUserRequest {
  nickname?: string
  email?: string
  enabled?: boolean
  roleIds?: number[]
}

/** 重置密码请求 */
export interface ResetPasswordRequest {
  password: string
}

/** 角色列表参数 */
export interface RoleListParams {
  page?: number
  pageSize?: number
}

export type RoleListResponse = PageResult<Role>

/** 创建角色请求 */
export interface CreateRoleRequest {
  name: string
  code: string
  description?: string
  permissions?: string[]
}

/** 更新角色请求 */
export interface UpdateRoleRequest {
  name?: string
  code?: string
  description?: string
  permissions?: string[]
}

/** 审计日志 */
export interface AuditLog {
  id: number
  userId?: number
  username: string
  action: string
  resource: string
  resourceName?: string
  namespace?: string
  clusterId?: number
  detail: string
  statusCode?: number
  path?: string
  requestId?: string
  ip: string
  timestamp: string
}

export interface AuditLogListParams {
  page?: number
  pageSize?: number
  keyword?: string
  username?: string
  action?: string
  resource?: string
  clusterId?: number
  status?: 'success' | 'failure'
  startTime?: string
  endTime?: string
}

export type AuditLogListResponse = PageResult<AuditLog>

/** 系统设置 */
export interface SystemSettings {
  siteName: string
  sessionTimeout: number
  maxLoginAttempts: number
  passwordExpirationDays: number
  enableAuditLog: boolean
  enableTwoFactorAuth: boolean
}
