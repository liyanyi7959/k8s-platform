// k8s 网络相关 API（Service / Ingress / IngressClass）
import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

// ── Service ──────────────────────────────────────────────

export async function listServices(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/services`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteService(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/services/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getServiceYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/services/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export type EditServiceRequest = {
  namespace: string
  name: string
  type?: string
  labels?: Record<string, string | null>
  annotations?: Record<string, string | null>
  selector?: Record<string, string | null>
}

export async function editService(clusterId: number, req: EditServiceRequest): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/services/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export interface CreateServicePort {
  name?: string
  port: number
  target_port?: number
  protocol?: string
}

export interface CreateServiceRequest {
  namespace: string
  name: string
  type: string
  selector?: Record<string, string>
  ports: CreateServicePort[]
}

export async function createService(clusterId: number, req: CreateServiceRequest): Promise<{ ok: boolean }> {
  const resp = (await http.post(`/api/v1/clusters/${clusterId}/services`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

// ── Endpoints ────────────────────────────────────────────

export async function listEndpoints(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/endpoints`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteEndpoints(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/endpoints/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getEndpointsYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/endpoints/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editEndpoints(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/endpoints/edit`, req)
}

// ── EndpointSlice ────────────────────────────────────────

export async function listEndpointSlices(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/endpointslices`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteEndpointSlice(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/endpointslices/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getEndpointSliceYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/endpointslices/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function editEndpointSlice(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/endpointslices/edit`, req)
}

// ── Ingress ──────────────────────────────────────────────

export async function listIngresses(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/ingresses`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function deleteIngress(clusterId: number, ns: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/ingresses/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}

export async function getIngressYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/ingresses/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export type EditIngressRequest = {
  namespace: string
  name: string
  ingressClassName?: string
  labels?: Record<string, string | null>
  annotations?: Record<string, string | null>
}

export async function editIngress(clusterId: number, req: EditIngressRequest): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/ingresses/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

export interface CreateIngressPath {
  path: string
  path_type?: string
  service_name: string
  service_port: number
}

export interface CreateIngressRule {
  host: string
  paths: CreateIngressPath[]
}

export interface CreateIngressRequest {
  namespace: string
  name: string
  ingress_class?: string
  rules: CreateIngressRule[]
  tls_secret_name?: string
  annotations?: Record<string, string>
}

export async function createIngress(clusterId: number, req: CreateIngressRequest): Promise<{ ok: boolean }> {
  const resp = (await http.post(`/api/v1/clusters/${clusterId}/ingresses`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}

// ── IngressClass ─────────────────────────────────────────

export async function listIngressClasses(
  clusterId: number,
  params: { sort_by?: string; order?: 'asc' | 'desc' } = {}
): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/ingressclasses`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

export async function getIngressClassYaml(clusterId: number, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/ingressclasses/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

export async function deleteIngressClass(clusterId: number, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/ingressclasses/${encodeURIComponent(name)}`)
}

export type EditIngressClassRequest = {
  name: string
  controller?: string
  isDefault?: boolean
  labels?: Record<string, string | null>
  annotations?: Record<string, string | null>
}

export async function editIngressClass(clusterId: number, req: EditIngressClassRequest): Promise<{ ok: boolean }> {
  const resp = (await http.patch(`/api/v1/clusters/${clusterId}/ingressclasses/edit`, req)) as ApiResponse<{ ok: boolean }>
  return resp.data
}
