import { http } from '@/shared/http/http'
import type { ApiResponse, PageResult } from '@/shared/types/api'

async function unwrap<T>(request: Promise<unknown>): Promise<T> {
  const resp = (await request) as ApiResponse<T>
  return resp.data
}

export interface DeployServerItem {
  id: number
  name: string
  ip: string
  ssh_port: number
  user: string
  auth_type: 'password' | 'key'
  os?: string
  os_version?: string
  kernel?: string
  cpu_cores?: number
  memory_mb?: number
  disk_gb?: number
  status: string
  remark?: string
  created_at: string
  updated_at: string
}

export interface SSHCredentialItem {
  id: number
  name: string
  auth_type: 'password' | 'key'
  username: string
  remark?: string
  created_at: string
  updated_at: string
}

export interface DeployPlanNodeItem {
  id: number
  server_id: number
  role: 'master' | 'worker'
  sort_order: number
}

export interface DeployPlanItem {
  id: number
  name: string
  cluster_name: string
  k8s_version: string
  pod_cidr: string
  svc_cidr: string
  cni_type: 'flannel' | 'calico' | 'cilium'
  status: string
  task_id?: number
  cluster_id?: number
  nodes?: DeployPlanNodeItem[]
  created_at: string
  updated_at: string
}

export interface CreateDeployServerReq {
  name: string
  ip: string
  ssh_port: number
  user: string
  auth_type: 'password' | 'key'
  credential: string
  remark?: string
}

export interface CreateSSHCredentialReq {
  name: string
  auth_type: 'password' | 'key'
  username: string
  credential: string
  remark?: string
}

export interface CreateDeployPlanReq {
  name: string
  cluster_name: string
  k8s_version: string
  pod_cidr: string
  svc_cidr: string
  cni_type: 'flannel' | 'calico' | 'cilium'
  nodes: Array<{
    server_id: number
    role: 'master' | 'worker'
    sort_order: number
  }>
}

export function listDeployServers(params: { page?: number; page_size?: number; keyword?: string; status?: string } = {}) {
  return unwrap<PageResult<DeployServerItem>>(http.get('/api/v1/deploy/servers', { params }))
}

export function createDeployServer(data: CreateDeployServerReq) {
  return unwrap<{ id: number }>(http.post('/api/v1/deploy/servers', data))
}

export function deleteDeployServer(id: number) {
  return unwrap<null>(http.delete(`/api/v1/deploy/servers/${id}`))
}

export function testDeployServerSSH(id: number) {
  return unwrap<{ status: string; message: string }>(http.post(`/api/v1/deploy/servers/${id}/test-ssh`))
}

export function listSSHCredentials(params: { page?: number; page_size?: number } = {}) {
  return unwrap<PageResult<SSHCredentialItem>>(http.get('/api/v1/deploy/ssh-credentials', { params }))
}

export function createSSHCredential(data: CreateSSHCredentialReq) {
  return unwrap<{ id: number }>(http.post('/api/v1/deploy/ssh-credentials', data))
}

export function deleteSSHCredential(id: number) {
  return unwrap<null>(http.delete(`/api/v1/deploy/ssh-credentials/${id}`))
}

export function listDeployPlans(params: { page?: number; page_size?: number; keyword?: string; status?: string } = {}) {
  return unwrap<PageResult<DeployPlanItem>>(http.get('/api/v1/deploy/deploy-plans', { params }))
}

export function createDeployPlan(data: CreateDeployPlanReq) {
  return unwrap<{ id: number }>(http.post('/api/v1/deploy/deploy-plans', data))
}

export function deleteDeployPlan(id: number) {
  return unwrap<null>(http.delete(`/api/v1/deploy/deploy-plans/${id}`))
}

export function executeDeployPlan(id: number) {
  return unwrap<{ task_id: number }>(http.post(`/api/v1/deploy/deploy-plans/${id}/execute`))
}

export function cancelDeployPlan(id: number) {
  return unwrap<null>(http.post(`/api/v1/deploy/deploy-plans/${id}/cancel`))
}

export function retryDeployPlan(id: number) {
  return unwrap<{ task_id: number }>(http.post(`/api/v1/deploy/deploy-plans/${id}/retry`))
}
