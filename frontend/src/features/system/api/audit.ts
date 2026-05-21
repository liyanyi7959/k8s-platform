import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export interface AuditLog {
  id: number
  user_id: number
  username: string
  action: string
  resource: string
  resource_name: string
  cluster_id: number
  namespace: string
  path: string
  status_code: number
  detail: string
  client_ip: string
  request_id: string
  created_at: string
}

export interface AuditLogListParams {
  page?: number
  page_size?: number
  username?: string
  action?: string
  resource?: string
  cluster_id?: number
  start_time?: string
  end_time?: string
}

export interface AuditLogListResult {
  total: number
  items: AuditLog[]
}

export async function getAuditLogs(params: AuditLogListParams = {}): Promise<AuditLogListResult> {
  const resp = (await http.get('/api/v1/audit-logs', { params })) as ApiResponse<AuditLogListResult>
  return resp.data
}
