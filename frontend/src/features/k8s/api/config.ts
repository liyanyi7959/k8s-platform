// k8s 配置与发现相关 API（ConfigMap / Secret / ServiceAccount / HPA / Event）
import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

// ── ConfigMap ────────────────────────────────────────────

export async function listConfigMaps(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/configmaps`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteConfigMap(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/configmaps/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getConfigMapYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/configmaps/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function getConfigMapRelated(
  clusterId: number,
  ns: string,
  name: string
): Promise<{ pods: Array<{ namespace: string; name: string; phase: string; node: string; ready: string; restarts: number; owners: Array<{ kind: string; name: string; uid?: string }> }>; controllers: Array<{ kind: string; name: string }> }> {
  const resp = (await http.get(
    `/api/v1/clusters/${clusterId}/configmaps/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/related`
  )) as ApiResponse<{
    pods: Array<{ namespace: string; name: string; phase: string; node: string; ready: string; restarts: number; owners: Array<{ kind: string; name: string; uid?: string }> }>
    controllers: Array<{ kind: string; name: string }>
  }>
  return resp.data
}

export type EditConfigMapRequest = {
  namespace: string
  name: string
  labels?: Record<string, string | null>
  data?: Record<string, string | null>
}

export async function editConfigMap(clusterId: number, req: EditConfigMapRequest): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/configmaps/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

// ── Secret ───────────────────────────────────────────────

export async function listSecrets(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/secrets`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteSecret(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/secrets/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getSecretYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/secrets/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function getSecretReveal(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/secrets/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/reveal`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function getSecretRelated(
  clusterId: number,
  ns: string,
  name: string
): Promise<{ pods: Array<{ namespace: string; name: string; phase: string; node: string; ready: string; restarts: number; owners: Array<{ kind: string; name: string; uid?: string }> }>; controllers: Array<{ kind: string; name: string }> }> {
  const resp = (await http.get(
    `/api/v1/clusters/${clusterId}/secrets/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/related`
  )) as ApiResponse<{
    pods: Array<{ namespace: string; name: string; phase: string; node: string; ready: string; restarts: number; owners: Array<{ kind: string; name: string; uid?: string }> }>
    controllers: Array<{ kind: string; name: string }>
  }>
  return resp.data
}

export type EditSecretRequest = {
  namespace: string
  name: string
  type?: string
  labels?: Record<string, string | null>
  data?: Record<string, string | null>
}

export async function editSecret(clusterId: number, req: EditSecretRequest): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/secrets/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

// ── ServiceAccount ───────────────────────────────────────

export async function listServiceAccounts(
  clusterId: number,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/serviceaccounts`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteServiceAccount(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/serviceaccounts/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function editServiceAccount(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/serviceaccounts/edit`, req)
}

export async function getServiceAccountYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(
    `/api/v1/clusters/${clusterId}/serviceaccounts/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`
  )) as ApiResponse<{ text: string }>
  return resp.data
}

// ── HPA ──────────────────────────────────────────────────

export async function listHPAs(
  clusterId: number,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/hpas`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteHPA(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/hpas/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getHPAYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/hpas/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{
    text: string
  }>
  return resp.data
}

export type EditHPARequest = {
  namespace: string
  yaml: string
}

export async function editHPA(clusterId: number, req: EditHPARequest): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/hpas/edit`, req)
}

// ── Event ────────────────────────────────────────────────

export async function listEvents(
  clusterId: number,
  params: {
    namespace?: string
    involved_object_kind?: string
    involved_object_name?: string
    involved_object_uid?: string
    sort_by?: string
    order?: 'asc' | 'desc'
  } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/events`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}
