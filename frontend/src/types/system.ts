/**
 * 系统管理类型 — 扁平结构，前端直接使用
 */
import type { PageResult } from './common'

/** 权限 */
export interface Permission {
  id: number
  code: string
  name: string
}

/** 角色 */
export interface Role {
  id: number
  name: string
  code: string
  description?: string
  permissions: Permission[]
  createdAt: string
}

/** 用户 */
export interface User {
  id: number
  username: string
  nickname: string
  email: string
  enabled: boolean
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
}

/** 用户列表参数 */
export interface UserListParams {
  page?: number
  pageSize?: number
  keyword?: string
}

export type UserListResponse = PageResult<User>

/** 创建用户请求 */
export interface CreateUserRequest {
  username: string
  password: string
  nickname?: string
  email?: string
  roleIds?: number[]
}

/** 更新用户请求 */
export interface UpdateUserRequest {
  nickname?: string
  email?: string
  enabled?: boolean
  roleIds?: number[]
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
}

/** 审计日志 */
export interface AuditLog {
  id: number
  username: string
  action: string
  resource: string
  detail: string
  ip: string
  timestamp: string
}

export interface AuditLogListParams {
  page?: number
  pageSize?: number
}

export type AuditLogListResponse = PageResult<AuditLog>

/** 系统设置 */
export interface SystemSettings {
  siteName: string
  logo?: string
  description?: string
  timezone: string
  language: string
}
