// k8s Pod 相关 API
import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export async function listPods(
  clusterId: number,
  params: { namespace?: string; label_selector?: string; sort_by?: string; order?: 'asc' | 'desc' } = {},
  options: { signal?: AbortSignal } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pods`, { params, signal: options.signal })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function listPodMetrics(
  clusterId: number,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {},
  options: { signal?: AbortSignal } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/podmetrics`, { params, signal: options.signal })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getPodYaml(
  clusterId: number,
  ns: string,
  pod: string,
  options: { signal?: AbortSignal } = {}
): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pods/${encodeURIComponent(ns)}/${encodeURIComponent(pod)}/yaml`, {
    signal: options.signal
  })) as ApiResponse<{ text: string }>
  return resp.data
}

export async function getPodLogs(
  clusterId: number,
  ns: string,
  pod: string,
  params: { container?: string; tail_lines?: number; previous?: boolean } = {},
  options: { signal?: AbortSignal } = {}
): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/pods/${encodeURIComponent(ns)}/${encodeURIComponent(pod)}/logs`, {
    params,
    signal: options.signal
  })) as ApiResponse<{
    text: string
  }>
  return resp.data
}

export async function createPodLogSession(
  clusterId: number,
  ns: string,
  pod: string,
  req: { container?: string; follow?: boolean; tail_lines?: number } = {}
): Promise<{ session_id: string; ws_url: string }> {
  const resp = (await http.post(`/api/v1/clusters/${clusterId}/pods/${encodeURIComponent(ns)}/${encodeURIComponent(pod)}/logs/session`, req)) as ApiResponse<{
    session_id: string
    ws_url: string
  }>
  return resp.data
}

export async function deletePod(
  clusterId: number,
  ns: string,
  pod: string,
  options: { force?: boolean } = {}
): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/pods/${encodeURIComponent(ns)}/${encodeURIComponent(pod)}`, {
    params: options.force ? { force: 'true' } : undefined
  })
}

export async function createPodExecSession(
  clusterId: number,
  ns: string,
  pod: string,
  req: { container?: string; command?: string[]; tty?: boolean }
): Promise<{ session_id: string; ws_url: string }> {
  const resp = (await http.post(`/api/v1/clusters/${clusterId}/pods/${encodeURIComponent(ns)}/${encodeURIComponent(pod)}/exec`, req)) as ApiResponse<{
    session_id: string
    ws_url: string
  }>
  return resp.data
}
