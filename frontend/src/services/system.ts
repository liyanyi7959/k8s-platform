/**
 * 系统管理 API - Mock 模式
 */
import { request } from '@umijs/max'
import type {
  User, UserListParams, UserListResponse, CreateUserRequest, UpdateUserRequest,
  Role, RoleListParams, RoleListResponse, CreateRoleRequest,
  AuditLog, AuditLogListParams, AuditLogListResponse,
  SystemSettings,
} from '@/types'

const MOCK_ENABLED = false

const MOCK_USERS: User[] = [
  {
    id: 1, username: 'admin', nickname: '管理员', email: 'admin@aiops.local',
    enabled: true,
    roles: [{ id: 1, name: '超级管理员', code: 'admin', permissions: [], createdAt: '2025-01-01T00:00:00Z' }],
    createdAt: '2025-01-01T00:00:00Z',
  },
  {
    id: 2, username: 'operator', nickname: '运维工程师', email: 'operator@aiops.local',
    enabled: true,
    roles: [{ id: 2, name: '运维人员', code: 'operator', permissions: [], createdAt: '2025-01-15T08:00:00Z' }],
    createdAt: '2025-02-15T10:00:00Z',
  },
  {
    id: 3, username: 'viewer', nickname: '只读用户', email: 'viewer@aiops.local',
    enabled: true,
    roles: [{ id: 3, name: '只读用户', code: 'viewer', permissions: [], createdAt: '2025-02-01T09:00:00Z' }],
    createdAt: '2025-03-01T14:30:00Z',
  },
]

const MOCK_ROLES: Role[] = [
  { id: 1, name: '超级管理员', code: 'admin', description: '拥有所有权限', permissions: [], createdAt: '2025-01-01T00:00:00Z' },
  { id: 2, name: '运维人员', code: 'operator', description: '运维人员', permissions: [], createdAt: '2025-01-15T08:00:00Z' },
  { id: 3, name: '只读用户', code: 'viewer', description: '只读用户', permissions: [], createdAt: '2025-02-01T09:00:00Z' },
]

const MOCK_AUDIT_LOGS: AuditLog[] = [
  { id: 1, username: 'admin', action: 'login', resource: 'auth', detail: '用户登录成功', ip: '192.168.1.100', timestamp: '2025-03-20T14:30:00Z' },
  { id: 2, username: 'admin', action: 'create', resource: 'cluster', detail: '导入集群: 生产集群-华东', ip: '192.168.1.100', timestamp: '2025-03-20T14:35:00Z' },
]

const MOCK_SETTINGS: SystemSettings = {
  siteName: 'AIOPS 智能运维平台',
  description: 'Kubernetes 集群管理与智能运维平台',
  timezone: 'Asia/Shanghai',
  language: 'zh-CN',
}

export function listUsers(params?: UserListParams): Promise<UserListResponse> {
  if (MOCK_ENABLED) {
    return new Promise((r) => setTimeout(() => r({ items: MOCK_USERS, total: MOCK_USERS.length, page: params?.page || 1, pageSize: params?.pageSize || 10 }), 200))
  }
  return request('/api/v1/users', { params })
}

export function getUserById(id: number): Promise<User> {
  if (MOCK_ENABLED) {
    const user = MOCK_USERS.find((u) => u.id === id)
    return user ? Promise.resolve(user) : Promise.reject(new Error('用户不存在'))
  }
  return request(`/api/v1/users/${id}`)
}

export function createUser(data: CreateUserRequest): Promise<User> {
  if (MOCK_ENABLED) {
    const newUser: User = {
      id: MOCK_USERS.length + 1, username: data.username,
      nickname: data.nickname || data.username, email: data.email || '',
      enabled: true, roles: [], createdAt: new Date().toISOString(),
    }
    MOCK_USERS.push(newUser)
    return Promise.resolve(newUser)
  }
  return request('/api/v1/users', { method: 'POST', data })
}

export function updateUser(id: number, data: UpdateUserRequest): Promise<User> {
  if (MOCK_ENABLED) {
    const idx = MOCK_USERS.findIndex((u) => u.id === id)
    if (idx === -1) return Promise.reject(new Error('用户不存在'))
    const user = MOCK_USERS[idx]!
    Object.assign(user, data)
    return Promise.resolve(user)
  }
  return request(`/api/v1/users/${id}`, { method: 'PUT', data })
}

export function deleteUser(id: number): Promise<void> {
  if (MOCK_ENABLED) {
    const idx = MOCK_USERS.findIndex((u) => u.id === id)
    if (idx !== -1) MOCK_USERS.splice(idx, 1)
    return Promise.resolve()
  }
  return request(`/api/v1/users/${id}`, { method: 'DELETE' })
}

export function resetPassword(id: number, newPassword: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => setTimeout(resolve, 200))
  }
  return request(`/api/v1/users/${id}/reset-password`, { method: 'POST', data: { password: newPassword } })
}

export function updateRole(id: number, data: Partial<CreateRoleRequest>): Promise<Role> {
  if (MOCK_ENABLED) {
    const idx = MOCK_ROLES.findIndex((r) => r.id === id)
    if (idx === -1) return Promise.reject(new Error('角色不存在'))
    const role = MOCK_ROLES[idx]!
    Object.assign(role, data)
    return Promise.resolve(role)
  }
  return request(`/api/v1/roles/${id}`, { method: 'PUT', data })
}

export function listRoles(params?: RoleListParams): Promise<RoleListResponse> {
  if (MOCK_ENABLED) {
    return new Promise((r) => setTimeout(() => r({ items: MOCK_ROLES, total: MOCK_ROLES.length, page: params?.page || 1, pageSize: params?.pageSize || 10 }), 150))
  }
  return request('/api/v1/roles', { params })
}

export function listAllRoles(): Promise<Role[]> {
  if (MOCK_ENABLED) return Promise.resolve([...MOCK_ROLES])
  return request('/api/v1/roles/all')
}

export function createRole(data: CreateRoleRequest): Promise<Role> {
  if (MOCK_ENABLED) {
    const newRole: Role = { id: MOCK_ROLES.length + 1, ...data, permissions: [], createdAt: new Date().toISOString() }
    MOCK_ROLES.push(newRole)
    return Promise.resolve(newRole)
  }
  return request('/api/v1/roles', { method: 'POST', data })
}

export function deleteRole(id: number): Promise<void> {
  if (MOCK_ENABLED) {
    const idx = MOCK_ROLES.findIndex((r) => r.id === id)
    if (idx !== -1) MOCK_ROLES.splice(idx, 1)
    return Promise.resolve()
  }
  return request(`/api/v1/roles/${id}`, { method: 'DELETE' })
}

export function listAuditLogs(params?: AuditLogListParams): Promise<AuditLogListResponse> {
  if (MOCK_ENABLED) {
    return new Promise((r) => setTimeout(() => r({ items: MOCK_AUDIT_LOGS, total: MOCK_AUDIT_LOGS.length, page: params?.page || 1, pageSize: params?.pageSize || 20 }), 200))
  }
  return request('/api/v1/audit-logs', { params })
}

export function getSystemSettings(): Promise<SystemSettings> {
  if (MOCK_ENABLED) return Promise.resolve({ ...MOCK_SETTINGS })
  return request('/api/v1/system/settings')
}

export function updateSystemSettings(data: Partial<SystemSettings>): Promise<SystemSettings> {
  if (MOCK_ENABLED) {
    Object.assign(MOCK_SETTINGS, data)
    return Promise.resolve({ ...MOCK_SETTINGS })
  }
  return request('/api/v1/system/settings', { method: 'PUT', data })
}
