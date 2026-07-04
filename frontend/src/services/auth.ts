/**
 * 认证服务
 */
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

/** 登录 */
export async function login(data: { username: string; password: string }): Promise<LoginResponse> {
  if (MOCK_ENABLED) {
    await new Promise((r) => setTimeout(r, 300))
    if (data.username === 'admin' && data.password === 'admin123') {
      const token = 'mock_token_' + Date.now()
      localStorage.setItem(TOKEN_KEY, token)
      return { token }
    }
    throw new Error('用户名或密码错误')
  }
  // 后端返回 { access_token, expires_in, user }，前端期望 { token }
  const res = await request<any>('/api/v1/auth/login', { method: 'POST', data })
  const loginRes: LoginResponse = { token: res?.access_token || '' }
  if (loginRes.token) {
    localStorage.setItem(TOKEN_KEY, loginRes.token)
  }
  return loginRes
}

/** 获取当前用户 */
export async function getCurrentUser(): Promise<User> {
  if (MOCK_ENABLED) {
    const token = localStorage.getItem(TOKEN_KEY)
    if (!token) throw new Error('未登录')
    return MOCK_USER
  }
  // 后端返回 { id, username, status, roles: string[], permissions: string[] }
  // 前端 User 期望 { id, username, nickname, email, enabled, roles: Role[], createdAt }
  const res = await request<any>('/api/v1/auth/me')
  return {
    id: res?.id || 0,
    username: res?.username || '',
    nickname: res?.nickname || res?.username || '',
    email: res?.email || '',
    enabled: res?.status === 'active',
    roles: Array.isArray(res?.roles) ? res.roles.map((r: string, i: number) => ({
      id: i + 1,
      name: r,
      code: r,
      description: '',
      permissions: Array.isArray(res?.permissions) ? res.permissions.map((p: string, j: number) => ({
        id: j + 1,
        code: p,
        name: p,
      })) : [],
      createdAt: '',
    })) : [],
    createdAt: res?.created_at || '',
  }
}

/** 退出登录 */
export async function logout(): Promise<void> {
  localStorage.removeItem(TOKEN_KEY)
  if (MOCK_ENABLED) {
    return
  }
  return request('/api/v1/auth/logout', { method: 'POST' })
}

/** 修改当前登录用户密码 */
export async function changePassword(data: { oldPassword: string; newPassword: string }): Promise<void> {
  return request('/api/v1/auth/change-password', {
    method: 'POST',
    data: {
      old_password: data.oldPassword,
      new_password: data.newPassword,
    },
  })
}

/** 获取滑块验证码 */
export async function getCaptcha(): Promise<{ enabled: boolean; token?: string; target_x?: number; track_width?: number }> {
  return request('/api/v1/auth/captcha')
}

/** 请求密码重置 */
export async function requestPasswordReset(identifier: string): Promise<{ token?: string; username?: string; message: string }> {
  return request('/api/v1/auth/password-reset/request', {
    method: 'POST',
    data: { identifier },
  })
}

/** 通过 token 重置密码 */
export async function confirmPasswordReset(token: string, newPassword: string): Promise<{ message: string }> {
  return request('/api/v1/auth/password-reset/confirm', {
    method: 'POST',
    data: { token, new_password: newPassword },
  })
}
