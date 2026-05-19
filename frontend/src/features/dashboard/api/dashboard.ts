import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

export type DashboardClusterOverview = {
  cluster: { id: number; name: string; status: string }
  stats: {
    nodes: { total: number; ready: number }
    pods: { total: number; running: number; pending: number; failed: number; succeeded: number }
    workloads: { deployments: number; statefulsets: number; daemonsets: number }
    cpu: { used_percent: number }
    memory: { used_percent: number }
  }
  charts: {
    cpu_memory_24h: { labels: string[]; cpu: number[]; memory: number[] }
    pod_phase: { running: number; pending: number; failed: number; succeeded: number }
    namespace_pods_top: Array<{ namespace: string; pods: number }>
    node_ready: { ready: number; total: number }
  }
  risks?: {
    certificates: Array<{
      key: string
      name: string
      component: string
      purpose: string
      not_before?: string
      not_after?: string
      days_left?: number
      status: 'ok' | 'warn' | 'critical' | 'unknown'
    }>
  }
  anomalies?: {
    failed_pods: Array<{ name: string; namespace: string; reason: string }>
  }
}

export type DashboardCertificateRisk = {
  key: string
  name: string
  component: string
  purpose: string
  not_before?: string
  not_after?: string
  days_left?: number
  status: 'ok' | 'warn' | 'critical' | 'unknown'
}

export async function getClusterOverview(clusterId: number): Promise<DashboardClusterOverview> {
  const resp = (await http.get(`/api/v1/dashboard/clusters/${clusterId}/overview`)) as ApiResponse<DashboardClusterOverview>
  return resp.data
}

export async function getClusterCertificateRisks(clusterId: number): Promise<DashboardCertificateRisk[]> {
  const resp = (await http.get(`/api/v1/dashboard/clusters/${clusterId}/certificate-risks`)) as ApiResponse<DashboardCertificateRisk[]>
  return Array.isArray(resp.data) ? resp.data : []
}
