import { request } from '@umijs/max'
import type { User, LoginResponse } from '@/types'

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
      permissions: [
        { id: 1, code: 'cluster:read', name: '查看集群' },
        { id: 2, code: 'cluster:write', name: '管理集群' },
        { id: 3, code: 'k8s:read', name: '查看K8s资源' },
        { id: 4, code: 'k8s:write', name: '管理K8s资源' },
      ],
      createdAt: '2025-01-01T00:00:00Z',
    },
  ],
  createdAt: '2025-01-01T00:00:00Z',
}

function toUser(raw: any): User {
  if (!raw || typeof raw.id !== 'number' || raw.id <= 0 || !String(raw.username || '').trim()) {
    throw new Error('未登录')
  }

  const roles = Array.isArray(raw.roles) ? raw.roles : []
  const permissions = Array.isArray(raw.permissions) ? raw.permissions : []

  return {
    id: raw.id,
    username: raw.username || '',
    nickname: raw.nickname || raw.username || '',
    email: raw.email || '',
    enabled: raw.status === 'active' || raw.enabled === true,
    roles: roles.map((role: string, roleIndex: number) => ({
      id: roleIndex + 1,
      name: role,
      code: role,
      description: '',
      permissions: permissions.map((permission: string, permissionIndex: number) => ({
        id: permissionIndex + 1,
        code: permission,
        name: permission,
      })),
      createdAt: '',
    })),
    createdAt: raw.created_at || '',
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

  const res = await request<any>('/api/v1/auth/login', {
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

  const res = await request<any>('/api/v1/auth/me')
  return toUser(res)
}

export async function logout(): Promise<void> {
  localStorage.removeItem(TOKEN_KEY)
  if (MOCK_ENABLED) {
    return
  }
  return request('/api/v1/auth/logout', { method: 'POST' })
}

export async function changePassword(data: {
  oldPassword: string
  newPassword: string
}): Promise<void> {
  return request('/api/v1/auth/change-password', {
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
  return request('/api/v1/auth/captcha')
}

export async function requestPasswordReset(
  identifier: string,
): Promise<{ token?: string; username?: string; message: string }> {
  return request('/api/v1/auth/password-reset/request', {
    method: 'POST',
    data: { identifier },
  })
}

export async function confirmPasswordReset(
  token: string,
  newPassword: string,
): Promise<{ message: string }> {
  return request('/api/v1/auth/password-reset/confirm', {
    method: 'POST',
    data: { token, new_password: newPassword },
  })
}
