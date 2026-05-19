import axios, { AxiosError } from 'axios'
import qs from 'qs'

import type { ApiResponse } from '@/shared/types/api'
import { getToken, clearToken, isTokenExpired } from '@/shared/utils/auth'
import { ApiError } from '@/shared/utils/error'

// ─── 基础配置 ──────────────────────────────────────────

function normalizeBaseUrl(input: string): string {
  const raw = String(input ?? '').trim()
  if (!raw) return ''
  try {
    const u = new URL(raw.startsWith('http') ? raw : `http://${raw}`)
    return `${u.protocol}//${u.host}${u.pathname.replace(/\/+$/, '')}`
  } catch {
    return ''
  }
}

const apiBaseUrl = normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL ?? '')
const API_BASE_KEY = 'k8s_platform_api_base'

function getRuntimeBaseUrl(): string {
  return String(localStorage.getItem(API_BASE_KEY) ?? '').trim()
}

// ─── Axios 实例 ────────────────────────────────────────

export const http = axios.create({
  baseURL: apiBaseUrl,
  timeout: 30_000,
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' })
})

// ─── 拦截器 ────────────────────────────────────────────

function emitAuthExpired(): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent('auth:expired'))
}

http.interceptors.request.use((config) => {
  const runtimeBase = getRuntimeBaseUrl()
  const normalized = normalizeBaseUrl(runtimeBase)
  if (normalized) {
    config.baseURL = normalized
  }

  const baseURL = String(config.baseURL ?? '').replace(/\/+$/, '')
  let url = String(config.url ?? '')
  if (baseURL && url && !/^https?:\/\//.test(url)) {
    if (baseURL.endsWith('/api/v1') && url.startsWith('/api/v1/')) {
      url = url.slice('/api/v1'.length)
    } else if (baseURL.endsWith('/api') && url.startsWith('/api/')) {
      url = url.slice('/api'.length)
    }
    if (url.startsWith('/')) {
      url = url.slice(1)
    }
    config.url = url
  }

  if (isTokenExpired()) {
    clearToken()
    emitAuthExpired()
  }

  const token = getToken()
  if (token) {
    config.headers = config.headers ?? {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (resp) => {
    const data = resp.data as ApiResponse<unknown>
    if (!data || typeof data.code !== 'number') {
      throw new ApiError({ code: 5000, message: '响应格式错误' })
    }
    if (data.code !== 0) {
      if (data.code === 1002) {
        clearToken()
        emitAuthExpired()
      }
      throw new ApiError({
        code: data.code,
        message: data.message || '请求失败',
        data: data.data
      })
    }
    return data as unknown as typeof resp
  },
  (err: AxiosError) => {
    const status = err.response?.status
    if (status === 401) {
      clearToken()
      emitAuthExpired()
      throw new ApiError({ code: 1002, message: '未登录或登录已过期' })
    }
    const message = err.message || '网络错误'
    throw new ApiError({ code: 5000, message, data: { http_status: status ?? null } })
  }
)
