// k8s 批处理相关 API（Job / CronJob）
import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export interface TriggerCronJobResp {
  job_name: string
}

export interface SuspendCronJobResp {
  suspend: boolean
}

export interface CleanCompletedJobsParams {
  namespace?: string
  older_than_hours?: number
}

export interface CleanCompletedJobsResp {
  deleted_count: number
}

// ── Job ──────────────────────────────────────────────────

export async function listJobs(
  clusterId: number,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/jobs`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getJobYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/jobs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deleteJob(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/jobs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function cleanCompletedJobs(clusterId: number, params: CleanCompletedJobsParams = {}): Promise<CleanCompletedJobsResp> {
  const resp = (await http.delete(`/api/v1/clusters/${clusterId}/jobs/completed`, { params })) as ApiResponse<CleanCompletedJobsResp>
  return resp.data
}

export async function editJob(
  clusterId: number,
  req: {
    namespace: string
    name: string
    labels?: Record<string, string>
    parallelism?: number
    completions?: number
    backoffLimit?: number
    ttlSecondsAfterFinished?: number
  }
): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/jobs/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

// ── CronJob ──────────────────────────────────────────────

export async function listCronJobs(
  clusterId: number,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/cronjobs`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getCronJobYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/cronjobs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deleteCronJob(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/cronjobs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function triggerCronJob(clusterId: number, ns: string, name: string): Promise<TriggerCronJobResp> {
  const resp = (await http.post(`/api/v1/clusters/${clusterId}/cronjobs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/trigger`)) as ApiResponse<TriggerCronJobResp>
  return resp.data
}

export async function suspendCronJob(clusterId: number, ns: string, name: string, suspend: boolean): Promise<SuspendCronJobResp> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/cronjobs/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/suspend`, { suspend })) as ApiResponse<SuspendCronJobResp>
  return resp.data
}

export async function editCronJob(
  clusterId: number,
  req: {
    namespace: string
    name: string
    labels?: Record<string, string>
    schedule: string
    suspend?: boolean
    concurrencyPolicy?: string
    successfulJobsHistoryLimit?: number
    failedJobsHistoryLimit?: number
  }
): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/cronjobs/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}
