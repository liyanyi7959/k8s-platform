import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export async function listLeases(clusterId: number, params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' } = {}): Promise<{ list: any[] }> {
	const resp = (await http.get(`/api/v1/clusters/${clusterId}/leases`, { params })) as ApiResponse<{ list: any[] }>
	return resp.data
}

export async function getLeaseYaml(clusterId: number, ns: string, name: string): Promise<{ text: string }> {
	const resp = (await http.get(`/api/v1/clusters/${clusterId}/leases/${encodeURIComponent(ns)}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
	return resp.data
}

export async function editLease(clusterId: number, req: { namespace: string; yaml: string }): Promise<void> {
	await http.patch(`/api/v1/clusters/${clusterId}/leases/edit`, req)
}

export async function deleteLease(clusterId: number, ns: string, name: string): Promise<void> {
	await http.delete(`/api/v1/clusters/${clusterId}/leases/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`)
}