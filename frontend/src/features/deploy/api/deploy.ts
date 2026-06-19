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
  credential_id?: number
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
  server_count: number
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
  credential_id?: number
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

export function updateDeployServer(id: number, data: Partial<CreateDeployServerReq>) {
  return unwrap<null>(http.put(`/api/v1/deploy/servers/${id}`, data))
}

export function deleteDeployServer(id: number) {
  return unwrap<null>(http.delete(`/api/v1/deploy/servers/${id}`))
}

export interface SSHProbeResult {
  status: string
  message: string
  os?: string
  os_version?: string
  kernel?: string
}

export function testDeployServerSSH(id: number) {
  return unwrap<SSHProbeResult>(http.post(`/api/v1/deploy/servers/${id}/test-ssh`))
}

export function listSSHCredentials(params: { page?: number; page_size?: number; keyword?: string; auth_type?: string } = {}) {
  return unwrap<PageResult<SSHCredentialItem>>(http.get('/api/v1/deploy/credentials', { params }))
}

export function createSSHCredential(data: CreateSSHCredentialReq) {
  return unwrap<{ id: number }>(http.post('/api/v1/deploy/credentials', data))
}

export function getSSHCredential(id: number) {
  return unwrap<SSHCredentialItem>(http.get(`/api/v1/deploy/credentials/${id}`))
}

export function updateSSHCredential(id: number, data: Partial<CreateSSHCredentialReq>) {
  return unwrap<null>(http.put(`/api/v1/deploy/credentials/${id}`, data))
}

export function deleteSSHCredential(id: number) {
  return unwrap<null>(http.delete(`/api/v1/deploy/credentials/${id}`))
}

export function batchDeleteSSHCredentials(ids: number[]) {
  return unwrap<null>(http.post('/api/v1/deploy/credentials/batch-delete', { ids }))
}

export function listDeployPlans(params: { page?: number; page_size?: number; keyword?: string; status?: string } = {}) {
  return unwrap<PageResult<DeployPlanItem>>(http.get('/api/v1/deploy/plans', { params }))
}

export function createDeployPlan(data: CreateDeployPlanReq) {
  return unwrap<{ id: number }>(http.post('/api/v1/deploy/plans', data))
}

export function deleteDeployPlan(id: number) {
  return unwrap<null>(http.delete(`/api/v1/deploy/plans/${id}`))
}

export function executeDeployPlan(id: number) {
  return unwrap<{ task_id: number }>(http.post(`/api/v1/deploy/plans/${id}/execute`))
}

export interface DryRunStep {
  key: string
  title: string
  description: string
  phase: string
  commands: string[]
  depends_on?: string[]
}
export interface DryRunNodeFlow {
  server_id: number
  server_name: string
  ip: string
  role: string
  steps: DryRunStep[]
}
export interface DryRunResult {
  plan_id: number
  plan_name: string
  cluster_name: string
  k8s_version: string
  cni_type: string
  nodes: DryRunNodeFlow[]
  summary: Record<string, number>
}
export function dryRunDeployPlan(id: number) {
  return unwrap<DryRunResult>(http.get(`/api/v1/deploy/plans/${id}/dry-run`))
}

export function cancelDeployPlan(id: number) {
  return unwrap<null>(http.post(`/api/v1/deploy/plans/${id}/cancel`))
}

export function retryDeployPlan(id: number) {
  return unwrap<{ task_id: number }>(http.post(`/api/v1/deploy/plans/${id}/retry`))
}

// SSE 实时日志订阅
export function subscribeDeployLogs(taskId: number, onLog: (log: string) => void, onDone?: (status: string, message: string) => void): EventSource {
  const token = localStorage.getItem('token') || ''
  const url = `/api/v1/deploy/tasks/${taskId}/logs/sse`
  
  const eventSource = new EventSource(url)
  
  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      if (data.log) {
        onLog(data.log)
      }
    } catch (e) {
      console.error('Failed to parse SSE log:', e)
    }
  }
  
  eventSource.addEventListener('done', (event) => {
    try {
      const data = JSON.parse(event.data)
      onDone?.(data.status, data.message)
    } catch (e) {
      console.error('Failed to parse SSE done event:', e)
    }
    eventSource.close()
  })
  
  eventSource.onerror = (error) => {
    console.error('SSE connection error:', error)
    eventSource.close()
  }
  
  return eventSource
}

export interface TaskLogsResult {
  task_id: number
  logs: string[]
  total: number
}

export function getDeployTaskLogs(taskId: number, params?: { offset?: number; limit?: number }) {
  return unwrap<TaskLogsResult>(http.get(`/api/v1/deploy/tasks/${taskId}/logs`, { params }))
}

// ─── 部署配置管理 ───

export interface DeployConfigItem {
  id: number
  step_key: string
  os_type: string
  step_name: string
  step_order: number
  command_template: string
  description?: string
  enabled: boolean
  timeout_seconds: number
  retry_count: number
  created_by: number
  created_at: string
  updated_at: string
}

export interface DeployConfigVersion {
  id: number
  config_id: number
  step_key: string
  command_template: string
  description?: string
  change_type: string
  changed_by: number
  changed_at: string
  change_summary?: string
}

export function listDeployConfigs(params?: { os_type?: string }) {
  return unwrap<DeployConfigItem[]>(http.get('/api/v1/deploy/configs', { params }))
}

export function listSupportedOSTypes() {
  return unwrap<string[]>(http.get('/api/v1/deploy/configs/os-types'))
}

export function getDeployConfig(id: number) {
  return unwrap<DeployConfigItem>(http.get(`/api/v1/deploy/configs/${id}`))
}

export function updateDeployConfig(id: number, data: {
  command_template: string
  description?: string
  enabled?: boolean
  timeout_seconds?: number
  retry_count?: number
  change_summary?: string
}) {
  return unwrap<null>(http.put(`/api/v1/deploy/configs/${id}`, data))
}

export function getDeployConfigVersions(id: number) {
  return unwrap<DeployConfigVersion[]>(http.get(`/api/v1/deploy/configs/${id}/versions`))
}

// ─── 仓库配置管理 ───

export interface RepositoryItem {
  id: number
  repo_type: string
  name: string
  url: string
  description?: string
  auth_type: string
  priority: number
  enabled: boolean
  is_default: boolean
  mirror_of?: string
  created_by: number
  created_at: string
  updated_at: string
}

export function listRepositories(params?: { type?: string }) {
  return unwrap<RepositoryItem[]>(http.get('/api/v1/deploy/repositories', { params }))
}

export function getRepository(id: number) {
  return unwrap<RepositoryItem>(http.get(`/api/v1/deploy/repositories/${id}`))
}

export function createRepository(data: {
  repo_type: string
  name: string
  url: string
  description?: string
  auth_type?: string
  priority?: number
  is_default?: boolean
  mirror_of?: string
}) {
  return unwrap<{ id: number }>(http.post('/api/v1/deploy/repositories', data))
}

export function updateRepository(id: number, data: {
  name?: string
  url?: string
  description?: string
  auth_type?: string
  priority?: number
  enabled?: boolean
  is_default?: boolean
  mirror_of?: string
}) {
  return unwrap<null>(http.put(`/api/v1/deploy/repositories/${id}`, data))
}

export function deleteRepository(id: number) {
  return unwrap<null>(http.delete(`/api/v1/deploy/repositories/${id}`))
}
