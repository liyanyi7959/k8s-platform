import { request } from '@umijs/max'
import type {
  AuditLog,
  AuditLogListParams,
  AuditLogListResponse,
  SystemSettings,
} from '@/features/platform/types'

interface BackendAuditLog {
  id: number
  user_id?: number
  username?: string
  action?: string
  resource?: string
  resource_name?: string
  namespace?: string
  cluster_id?: number
  detail?: string
  status_code?: number
  path?: string
  request_id?: string
  client_ip?: string
  created_at?: string
}

interface BackendAuditListResponse {
  items: BackendAuditLog[]
  total: number
  page: number
  pageSize: number
}

function normalizeAuditLog(raw: BackendAuditLog): AuditLog {
  return {
    id: raw.id,
    userId: raw.user_id,
    username: raw.username || '未知用户',
    action: raw.action || 'unknown',
    resource: raw.resource_name || raw.resource || '—',
    resourceName: raw.resource_name,
    namespace: raw.namespace,
    clusterId: raw.cluster_id,
    detail: raw.detail || '—',
    statusCode: raw.status_code,
    path: raw.path,
    requestId: raw.request_id,
    ip: raw.client_ip || '—',
    timestamp: raw.created_at || new Date().toISOString(),
  }
}

export async function listAuditLogs(params?: AuditLogListParams): Promise<AuditLogListResponse> {
  const payload: Record<string, unknown> = {}
  if (params?.page) payload.page = params.page
  if (params?.pageSize) payload.page_size = params.pageSize
  if (params?.keyword) payload.keyword = params.keyword
  if (params?.username) payload.username = params.username
  if (params?.action) payload.action = params.action
  if (params?.resource) payload.resource = params.resource
  if (params?.clusterId) payload.cluster_id = params.clusterId
  if (params?.status) payload.status = params.status
  if (params?.startTime) payload.start_time = params.startTime
  if (params?.endTime) payload.end_time = params.endTime
  const response = await request<BackendAuditListResponse>('/api/v2/audit-logs', { params: payload })
  return {
    items: (response.items || []).map(normalizeAuditLog),
    total: response.total || 0,
    page: response.page || 1,
    pageSize: response.pageSize || 20,
  }
}

export function getSystemSettings(): Promise<SystemSettings> {
  return request('/api/v2/platform/settings')
}

export function updateSystemSettings(data: SystemSettings): Promise<SystemSettings> {
  return request('/api/v2/platform/settings', { method: 'PATCH', data })
}
