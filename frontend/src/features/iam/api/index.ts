import { request } from '@umijs/max'
import type {
  CreateRoleRequest,
  CreateUserRequest,
  LoginResponse,
  Permission,
  ResetPasswordRequest,
  Role,
  RoleListParams,
  RoleListResponse,
  UpdateRoleRequest,
  UpdateUserRequest,
  User,
  UserListParams,
  UserListResponse,
} from '@/features/iam/types'

const MOCK_ENABLED = false
const TOKEN_KEY = 'token'

const MOCK_USER: User = {
  id: 1,
  username: 'admin',
  nickname: '管理员',
  email: 'admin@aiops.local',
  enabled: true,
  roles: [
    {
      id: 1,
      name: '超级管理员',
      code: 'admin',
      description: '拥有所有权限',
      permissions: ['cluster:read', 'cluster:write', 'k8s:read', 'k8s:write'],
      createdAt: '2025-01-01T00:00:00Z',
    },
  ],
  createdAt: '2025-01-01T00:00:00Z',
}

function toUser(raw: unknown): User {
  if (!raw || typeof (raw as Record<string, unknown>).id !== 'number' || (raw as Record<string, unknown>).id as number <= 0 || !String((raw as Record<string, unknown>).username || '').trim()) {
    throw new Error('未登录')
  }

  const rawRoles = (raw as Record<string, unknown>).roles
  const rawPermissions = (raw as Record<string, unknown>).permissions
  const roles = Array.isArray(rawRoles) ? rawRoles as string[] : []
  const permissions = Array.isArray(rawPermissions) ? rawPermissions as string[] : []

  return {
    id: (raw as Record<string, unknown>).id as number,
    username: String((raw as Record<string, unknown>).username || ''),
    nickname: String((raw as Record<string, unknown>).nickname || (raw as Record<string, unknown>).username || ''),
    email: String((raw as Record<string, unknown>).email || ''),
    enabled: (raw as Record<string, unknown>).status === 'active' || (raw as Record<string, unknown>).enabled === true,
    roles: roles.map((role: string, roleIndex: number) => ({
      id: roleIndex + 1,
      name: role,
      code: role,
      description: '',
      permissions,
      createdAt: '',
    })),
    createdAt: String((raw as Record<string, unknown>).created_at || ''),
  }
}

export async function login(data: {
  username: string
  password: string
  captchaToken?: string
  captchaX?: number
}): Promise<LoginResponse> {
  if (MOCK_ENABLED) {
    await new Promise((resolve) => setTimeout(resolve, 300))
    if (data.username === 'admin' && data.password === 'admin@123') {
      const token = 'mock_token_' + Date.now()
      localStorage.setItem(TOKEN_KEY, token)
      return { token, user: MOCK_USER }
    }
    throw new Error('用户名或密码错误')
  }

  const res = await request<any>('/api/v2/session', {
    method: 'POST',
    data: {
      username: data.username,
      password: data.password,
      captcha_token: data.captchaToken,
      captcha_x: data.captchaX,
    },
  })
  const loginRes: LoginResponse = {
    token: res?.access_token || '',
    user: res?.user ? toUser(res.user) : undefined,
  }
  if (loginRes.token) {
    localStorage.setItem(TOKEN_KEY, loginRes.token)
  }
  return loginRes
}

export async function getCurrentUser(): Promise<User> {
  if (MOCK_ENABLED) {
    const token = localStorage.getItem(TOKEN_KEY)
    if (!token) throw new Error('未登录')
    return MOCK_USER
  }

  const res = await request<any>('/api/v2/identity')
  return toUser(res)
}

export async function logout(): Promise<void> {
  localStorage.removeItem(TOKEN_KEY)
  if (MOCK_ENABLED) {
    return
  }
  return request('/api/v2/session/logout', { method: 'POST' })
}

export async function changePassword(data: {
  oldPassword: string
  newPassword: string
}): Promise<void> {
  return request('/api/v2/identity/password-change-requests', {
    method: 'POST',
    data: {
      old_password: data.oldPassword,
      new_password: data.newPassword,
    },
  })
}

export async function getCaptcha(): Promise<{
  enabled: boolean
  token?: string
  target_x?: number
  track_width?: number
}> {
  return request('/api/v2/captcha')
}

export async function requestPasswordReset(
  identifier: string,
): Promise<{ token?: string; username?: string; message: string }> {
  return request('/api/v2/password-reset-requests', {
    method: 'POST',
    data: { identifier },
  })
}

export async function confirmPasswordReset(
  token: string,
  newPassword: string,
): Promise<{ message: string }> {
  return request('/api/v2/password-reset-confirmations', {
    method: 'POST',
    data: { token, new_password: newPassword },
  })
}

export function listUsers(params?: UserListParams): Promise<UserListResponse> {
  const payload: Record<string, unknown> = {}
  if (params?.page) payload.page = params.page
  if (params?.pageSize) payload.page_size = params.pageSize
  if (params?.keyword) payload.keyword = params.keyword
  if (params?.status) payload.status = params.status
  if (params?.roleId) payload.role_id = params.roleId
  return request('/api/v2/users', { params: payload })
}

export function getUserById(id: number): Promise<User> {
  return request(`/api/v2/users/${id}`)
}

export function createUser(data: CreateUserRequest): Promise<User> {
  return request('/api/v2/users', { method: 'POST', data })
}

export function updateUser(id: number, data: UpdateUserRequest): Promise<User> {
  return request(`/api/v2/users/${id}`, { method: 'PATCH', data })
}

export function deleteUser(id: number): Promise<void> {
  return request(`/api/v2/users/${id}`, { method: 'DELETE' })
}

export function resetPassword(id: number, data: ResetPasswordRequest): Promise<void> {
  return request(`/api/v2/users/${id}/password-reset-requests`, { method: 'POST', data })
}

export function listRoles(params?: RoleListParams): Promise<RoleListResponse> {
  const payload: Record<string, unknown> = {}
  if (params?.page) payload.page = params.page
  if (params?.pageSize) payload.page_size = params.pageSize
  return request('/api/v2/roles', { params: payload })
}

export async function listAllRoles(): Promise<Role[]> {
  const response = await request<RoleListResponse>('/api/v2/roles')
  return response?.items || []
}

export function listPermissions(): Promise<Permission[]> {
  return request('/api/v2/permissions')
}

export function createRole(data: CreateRoleRequest): Promise<Role> {
  return request('/api/v2/roles', { method: 'POST', data })
}

export function updateRole(id: number, data: UpdateRoleRequest): Promise<Role> {
  return request(`/api/v2/roles/${id}`, { method: 'PATCH', data })
}

export function deleteRole(id: number): Promise<void> {
  return request(`/api/v2/roles/${id}`, { method: 'DELETE' })
}
