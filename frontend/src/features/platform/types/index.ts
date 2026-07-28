import type { PageResult } from '@/shared/types/common'

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

export interface SystemSettings {
  siteName: string
  sessionTimeout: number
  maxLoginAttempts: number
  passwordExpirationDays: number
  enableAuditLog: boolean
  enableTwoFactorAuth: boolean
}
