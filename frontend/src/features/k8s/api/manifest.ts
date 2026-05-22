import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'
import type { PageResult } from '@/shared/types/api'

export interface ManifestApplyRequest {
  yaml: string
  default_namespace?: string
  dry_run?: boolean
  source_label?: string
  source_resource?: string
  workload_kind?: string
}

export interface ManifestApplyResultItem {
  api_version: string
  kind: string
  namespace?: string
  name: string
  operation: 'create' | 'update' | string
  resource: string
  scope: 'cluster' | 'namespace' | string
}

export interface ManifestApplyResponse {
  record_id: number
  status: string
  dry_run: boolean
  summary: string
  items: ManifestApplyResultItem[]
}

export interface ManifestApplyRecord {
  id: number
  status: string
  dry_run: boolean
  default_namespace: string
  source_label: string
  source_resource: string
  workload_kind: string
  result_count: number
  summary: string
  error_message: string
  created_by: number
  created_by_name: string
  created_at: string
  updated_at: string
}

export interface ManifestApplyRecordDetail extends ManifestApplyRecord {
  cluster_id: number
  yaml_content: string
  result_items: ManifestApplyResultItem[]
}

export interface ManifestApplyRecordListParams {
  page?: number
  page_size?: number
  keyword?: string
  status?: string
  mode?: string
  default_namespace?: string
}

export async function applyManifest(clusterId: number, req: ManifestApplyRequest): Promise<ManifestApplyResponse> {
  const resp = (await http.post(`/api/v1/clusters/${clusterId}/manifests/apply`, req)) as ApiResponse<ManifestApplyResponse>
  return resp.data
}

export async function getManifestApplyRecords(clusterId: number, params: ManifestApplyRecordListParams): Promise<PageResult<ManifestApplyRecord>> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/manifests/records`, { params })) as ApiResponse<PageResult<ManifestApplyRecord>>
  return resp.data
}

export async function getManifestApplyRecord(clusterId: number, recordId: number): Promise<ManifestApplyRecordDetail> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/manifests/records/${recordId}`)) as ApiResponse<ManifestApplyRecordDetail>
  return resp.data
}