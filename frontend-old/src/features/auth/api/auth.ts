import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export interface LoginReq {
  username: string
  password: string
}

export interface UserMe {
  id: number
  username: string
  status: 'active' | 'disabled'
  roles: string[]
  permissions: string[]
}

export interface LoginResp {
  access_token: string
  expires_in: number
  user: UserMe
}

export async function login(req: LoginReq): Promise<LoginResp> {
  const resp = (await http.post('/api/v1/auth/login', req)) as ApiResponse<LoginResp>
  return resp.data
}

export async function logout(refreshToken?: string): Promise<void> {
  await http.post('/api/v1/auth/logout', refreshToken ? { refresh_token: refreshToken } : {})
}

export async function getMe(): Promise<UserMe> {
  const resp = (await http.get('/api/v1/auth/me')) as ApiResponse<UserMe>
  return resp.data
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<void> {
  await http.post('/api/v1/auth/change-password', { old_password: oldPassword, new_password: newPassword })
}

