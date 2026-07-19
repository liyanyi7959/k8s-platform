/**
 * 部署服务层
 * 对接后端 deploy_controller / deploy_config_controller / version_controller
 */
import { request } from '@umijs/max'
import type {
  DeployServer,
  Credential,
  DeployPlan,
  DeployTask,
  CreateDeployPlanRequest,
  CreateServerRequest,
  CreateCredentialRequest,
  DeployConfigStep,
  DeployVersion,
} from '@/types/deploy'

// ═══════════════════════════════════════════════════════════
//  辅助函数：snake_case → camelCase 转换 + 分页数据提取
// ═══════════════════════════════════════════════════════════

/** 将 snake_case key 转为 camelCase */
function toCamelCase(str: string): string {
  return str.replace(/_([a-z])/g, (_, c) => c.toUpperCase())
}

/** 递归转换对象 key 为 camelCase */
function camelizeKeys(obj: any): any {
  if (Array.isArray(obj)) return obj.map(camelizeKeys)
  if (obj !== null && typeof obj === 'object' && !(obj instanceof Date)) {
    const result: any = {}
    for (const [k, v] of Object.entries(obj)) {
      result[toCamelCase(k)] = camelizeKeys(v)
    }
    return result
  }
  return obj
}

function normalizeDeployAuthType(authType?: string): 'password' | 'key' | undefined {
  switch ((authType || '').trim()) {
    case 'ssh_password':
    case 'password':
      return 'password'
    case 'ssh_key':
    case 'key':
      return 'key'
    default:
      return undefined
  }
}

/** 从后端分页响应中提取列表数据 */
function extractPageData<T>(res: any): { items: T[]; total: number; page: number; pageSize: number } {
  if (Array.isArray(res)) {
    return { items: camelizeKeys(res), total: res.length, page: 1, pageSize: res.length || 10 }
  }
  if (res && typeof res === 'object') {
    const list = res.list || res.items || []
    return {
      items: camelizeKeys(list),
      total: res.total ?? list.length,
      page: res.page ?? 1,
      pageSize: res.page_size ?? res.pageSize ?? 10,
    }
  }
  return { items: [], total: 0, page: 1, pageSize: 10 }
}

/** 服务器状态映射 */
function mapServer(raw: any): DeployServer {
  return camelizeKeys(raw) as DeployServer
}

/** 凭证映射 */
function mapCredential(raw: any): Credential {
  const data = camelizeKeys(raw) as Credential & { type?: string }
  const authType = normalizeDeployAuthType((data as any).authType || (data as any).type)
  return {
    ...data,
    authType: authType || 'password',
  }
}

/** 部署计划映射 */
function mapPlan(raw: any): DeployPlan {
  return camelizeKeys(raw) as DeployPlan
}

function serializeStepOverrides(stepOverrides?: Record<string, any>) {
  if (!stepOverrides) return undefined
  return Object.fromEntries(
    Object.entries(stepOverrides).map(([key, value]: [string, any]) => ([key, {
      step_key: value.stepKey,
      node_role: value.nodeRole,
      node_server_id: value.nodeServerId,
      command_template: value.commandTemplate,
      description: value.description,
      timeout_seconds: value.timeoutSeconds,
      retry_count: value.retryCount,
      enabled: value.enabled,
    }]))
  )
}

// ═══════════════════════════════════════════════════════════
//  服务器管理 API
// ═══════════════════════════════════════════════════════════

export function getServers(params?: {
  page?: number
  pageSize?: number
}): Promise<{ items: DeployServer[]; total: number; page: number; pageSize: number }> {
  const { page, pageSize } = params || {}
  return request('/api/v1/deploy/servers', {
    params: { page, page_size: pageSize },
  }).then((res) => extractPageData<DeployServer>(res))
}

export function createServer(data: CreateServerRequest): Promise<DeployServer> {
  return request('/api/v1/deploy/servers', {
    method: 'POST',
    data: {
      name: data.name,
      ip: data.ip,
      ssh_port: data.sshPort,
      user: data.user,
      auth_type: normalizeDeployAuthType(data.authType),
      credential_id: data.credentialId,
      credential: data.credential,
      labels: data.labels,
      remark: data.remark,
    },
  }).then(mapServer)
}

export function updateServer(id: number, data: Partial<CreateServerRequest>): Promise<void> {
  return request(`/api/v1/deploy/servers/${id}`, {
    method: 'PUT',
    data: {
      name: data.name,
      ip: data.ip,
      ssh_port: data.sshPort,
      user: data.user,
      auth_type: data.authType ? normalizeDeployAuthType(data.authType) : undefined,
      credential_id: data.credentialId,
      credential: data.credential,
      labels: data.labels,
      remark: data.remark,
    },
  })
}

export function deleteServer(id: number): Promise<void> {
  return request(`/api/v1/deploy/servers/${id}`, { method: 'DELETE' })
}

export function testServerSSH(id: number): Promise<{ success: boolean; message: string }> {
  return request(`/api/v1/deploy/servers/${id}/test-ssh`, { method: 'POST' })
}

// ═══════════════════════════════════════════════════════════
//  SSH 凭证管理 API
// ═══════════════════════════════════════════════════════════

export function getCredentials(params?: {
  page?: number
  pageSize?: number
}): Promise<{ items: Credential[]; total: number; page: number; pageSize: number }> {
  const { page, pageSize } = params || {}
  return request('/api/v1/deploy/credentials', {
    params: { page, page_size: pageSize },
  }).then((res) => extractPageData<Credential>(res))
}

export function createCredential(data: CreateCredentialRequest): Promise<Credential> {
  const authType = normalizeDeployAuthType(data.authType)
  const credential = authType === 'key' ? (data.privateKey || data.credential || '') : (data.password || data.credential || '')
  return request('/api/v1/deploy/credentials', {
    method: 'POST',
    data: {
      name: data.name,
      auth_type: authType,
      username: data.username,
      credential,
      remark: data.remark,
    },
  }).then(mapCredential)
}

export function updateCredential(id: number, data: Partial<CreateCredentialRequest>): Promise<void> {
  const authType = data.authType ? normalizeDeployAuthType(data.authType) : undefined
  const credential = authType === 'key'
    ? (data.privateKey || data.credential || '')
    : authType === 'password'
      ? (data.password || data.credential || '')
      : undefined
  return request(`/api/v1/deploy/credentials/${id}`, {
    method: 'PUT',
    data: {
      name: data.name,
      auth_type: authType,
      username: data.username,
      credential,
      remark: data.remark,
    },
  })
}

export function deleteCredential(id: number): Promise<void> {
  return request(`/api/v1/deploy/credentials/${id}`, { method: 'DELETE' })
}

export function batchDeleteCredentials(ids: number[]): Promise<void> {
  return request('/api/v1/deploy/credentials/batch-delete', {
    method: 'POST',
    data: { ids },
  })
}

// ═══════════════════════════════════════════════════════════
//  部署计划 API（K8s 集群部署方案）
// ═══════════════════════════════════════════════════════════

export function getDeployPlans(params?: {
  page?: number
  pageSize?: number
}): Promise<{ items: DeployPlan[]; total: number; page: number; pageSize: number }> {
  const { page, pageSize } = params || {}
  return request('/api/v1/deploy/plans', {
    params: { page, page_size: pageSize },
  }).then((res) => extractPageData<DeployPlan>(res))
}

export function getDeployPlanById(id: number): Promise<DeployPlan> {
  return request(`/api/v1/deploy/plans/${id}`).then(mapPlan)
}

export function createDeployPlan(data: CreateDeployPlanRequest): Promise<DeployPlan> {
  return request('/api/v1/deploy/plans', {
    method: 'POST',
    data: {
      name: data.name,
      cluster_name: data.clusterName,
      k8s_version: data.k8sVersion,
      pod_cidr: data.podCidr,
      svc_cidr: data.svcCidr,
      cni_type: data.cniType,
      cni_config: data.cniConfig,
      addons: data.addons,
      step_overrides: serializeStepOverrides(data.stepOverrides),
      nodes: (data.nodes || []).map((n) => ({
        server_id: n.serverId,
        role: n.role,
        sort_order: n.sortOrder,
      })),
    },
  }).then(mapPlan)
}

export function updateDeployPlan(id: number, data: Partial<CreateDeployPlanRequest>): Promise<void> {
  return request(`/api/v1/deploy/plans/${id}`, {
    method: 'PUT',
    data: {
      name: data.name,
      cluster_name: data.clusterName,
      k8s_version: data.k8sVersion,
      pod_cidr: data.podCidr,
      svc_cidr: data.svcCidr,
      cni_type: data.cniType,
      cni_config: data.cniConfig,
      addons: data.addons,
      step_overrides: serializeStepOverrides(data.stepOverrides),
      nodes: (data.nodes || []).map((n) => ({
        server_id: n.serverId,
        role: n.role,
        sort_order: n.sortOrder,
      })),
    },
  })
}

export function deleteDeployPlan(id: number): Promise<void> {
  return request(`/api/v1/deploy/plans/${id}`, { method: 'DELETE' })
}

export function executeDeployPlan(id: number): Promise<{ taskId: number }> {
  return request(`/api/v1/deploy/plans/${id}/execute`, { method: 'POST' })
}

export function cancelDeployPlan(id: number): Promise<void> {
  return request(`/api/v1/deploy/plans/${id}/cancel`, { method: 'POST' })
}

export function retryDeployPlan(id: number): Promise<{ taskId: number }> {
  return request(`/api/v1/deploy/plans/${id}/retry`, { method: 'POST' })
}

// ═══════════════════════════════════════════════════════════
//  部署任务 API
// ═══════════════════════════════════════════════════════════

/** 获取部署任务详情 */
export function getDeployTask(taskId: number): Promise<DeployTask> {
  return request(`/api/v1/deploy/tasks/${taskId}`).then(camelizeKeys)
}

/** 获取部署任务日志（分页） */
export function getDeployTaskLogs(taskId: number, offset = 0, limit = 200): Promise<{ logs: string[]; total: number }> {
  return request(`/api/v1/deploy/tasks/${taskId}/logs`, {
    params: { offset, limit },
  })
}

/** 构建 SSE 日志流 URL */
export function getDeployTaskLogSSEUrl(taskId: number): string {
  return `/api/v1/deploy/tasks/${taskId}/logs/sse`
}

/** 干跑预览 - 返回部署计划的模拟运行流程 */
export function dryRunDeployPlan(id: number): Promise<any> {
  return request(`/api/v1/deploy/plans/${id}/dry-run`).then(camelizeKeys)
}

// ═══════════════════════════════════════════════════════════
//  部署配置 API（步骤配置）
// ═══════════════════════════════════════════════════════════

export function getDeployConfigSteps(): Promise<DeployConfigStep[]> {
  return request('/api/v1/deploy-config/steps').then((res: any) => {
    const list = Array.isArray(res) ? res : res?.list || res?.items || []
    return camelizeKeys(list) as DeployConfigStep[]
  })
}

export function createDeployConfigStep(data: {
  name: string
  type: string
  cmd: string
  orderNum?: number
  enabled?: boolean
}): Promise<DeployConfigStep> {
  return request('/api/v1/deploy-config/steps', {
    method: 'POST',
    data: {
      name: data.name,
      type: data.type,
      cmd: data.cmd,
      order_num: data.orderNum,
      enabled: data.enabled ?? true,
    },
  }).then((res) => camelizeKeys(res) as DeployConfigStep)
}

export function updateDeployConfigStep(id: number, data: {
  name?: string
  type?: string
  cmd?: string
  orderNum?: number
  enabled?: boolean
}): Promise<void> {
  return request(`/api/v1/deploy-config/steps/${id}`, {
    method: 'PUT',
    data: {
      name: data.name,
      type: data.type,
      cmd: data.cmd,
      order_num: data.orderNum,
      enabled: data.enabled,
    },
  })
}

export function deleteDeployConfigStep(id: number): Promise<void> {
  return request(`/api/v1/deploy-config/steps/${id}`, { method: 'DELETE' })
}

export function updateStepOrders(steps: { id: number; orderNum: number }[]): Promise<void> {
  return request('/api/v1/deploy-config/steps/orders', {
    method: 'PUT',
    data: { orders: steps.map((s) => ({ id: s.id, order_num: s.orderNum })) },
  })
}

// ═══════════════════════════════════════════════════════════
//  发版管理 API
// ═══════════════════════════════════════════════════════════

export function getVersions(params?: {
  page?: number
  pageSize?: number
}): Promise<{ items: DeployVersion[]; total: number; page: number; pageSize: number }> {
  const { page, pageSize } = params || {}
  return request('/api/v1/versions', {
    params: { page, page_size: pageSize },
  }).then((res) => extractPageData<DeployVersion>(res))
}

export function createVersion(data: {
  name: string
  version: string
  remark?: string
}): Promise<DeployVersion> {
  return request('/api/v1/versions', {
    method: 'POST',
    data,
  }).then((res) => camelizeKeys(res) as DeployVersion)
}

export function publishVersion(id: number): Promise<void> {
  return request(`/api/v1/versions/${id}/publish`, { method: 'POST' })
}

export function rollbackVersion(id: number): Promise<void> {
  return request(`/api/v1/versions/${id}/rollback`, { method: 'POST' })
}

export function deleteVersion(id: number): Promise<void> {
  return request(`/api/v1/versions/${id}`, { method: 'DELETE' })
}

// ═══════════════════════════════════════════════════════════
//  部署步骤配置 + 仓库配置（config.tsx 使用）
// ═══════════════════════════════════════════════════════════

import type { DeployStepConfig, RepositoryConfig, ConfigVersion } from '@/types/deploy'

/** 列出所有部署步骤配置 */
export function listDeployStepConfigs(params?: { osType?: string }, signal?: AbortSignal): Promise<DeployStepConfig[]> {
  return request('/api/v1/deploy/configs', {
    signal,
    params: {
      os_type: params?.osType,
    },
  }).then((res: any) => {
    const list = Array.isArray(res) ? res : res?.list || res?.items || []
    return camelizeKeys(list) as DeployStepConfig[]
  })
}

/** 获取支持的 Linux 分类 */
export function listSupportedOSTypes(signal?: AbortSignal): Promise<string[]> {
  return request('/api/v1/deploy/configs/os-types', { signal }).then((res: any) => {
    const list = Array.isArray(res) ? res : res?.list || res?.items || []
    return camelizeKeys(list) as string[]
  })
}

/** 更新单个步骤配置 */
export function updateDeployStepConfig(id: number, data: Partial<DeployStepConfig>): Promise<void> {
  return request(`/api/v1/deploy/configs/${id}`, {
    method: 'PUT',
    data: {
      command_template: data.commandTemplate,
      description: data.description,
      enabled: data.enabled,
      timeout_seconds: data.timeoutSeconds,
      retry_count: data.retryCount,
    },
  })
}

/** 获取步骤配置的版本历史 */
export function listDeployConfigVersions(stepId: number, signal?: AbortSignal): Promise<ConfigVersion[]> {
  return request(`/api/v1/deploy/configs/${stepId}/versions`, { signal }).then((res: any) => {
    const list = Array.isArray(res) ? res : res?.list || res?.items || []
    return camelizeKeys(list) as ConfigVersion[]
  })
}

/** 列出所有仓库配置 */
export function listRepositories(_?: any, signal?: AbortSignal): Promise<RepositoryConfig[]> {
  return request('/api/v1/deploy/repositories', { signal }).then((res: any) => {
    const list = Array.isArray(res) ? res : res?.list || res?.items || []
    return camelizeKeys(list) as RepositoryConfig[]
  })
}

/** 创建仓库配置 */
export function createRepository(data: Partial<RepositoryConfig>): Promise<RepositoryConfig> {
  return request('/api/v1/deploy/repositories', {
    method: 'POST',
    data: {
      name: data.name,
      type: data.type,
      url: data.url,
      auth_type: data.authType,
      username: data.username,
      password: data.password,
      token: data.token,
      description: data.description,
    },
  }).then((res) => camelizeKeys(res) as RepositoryConfig)
}

/** 更新仓库配置 */
export function updateRepository(id: number, data: Partial<RepositoryConfig>): Promise<void> {
  return request(`/api/v1/deploy/repositories/${id}`, {
    method: 'PUT',
    data: {
      name: data.name,
      type: data.type,
      url: data.url,
      auth_type: data.authType,
      username: data.username,
      password: data.password,
      token: data.token,
      description: data.description,
    },
  })
}

/** 删除仓库配置 */
export function deleteRepository(id: number): Promise<void> {
  return request(`/api/v1/deploy/repositories/${id}`, { method: 'DELETE' })
}

/** 获取 Ansible playbook 源码 */
export function getAnsiblePlaybook(): Promise<{ content: string; path: string }> {
  return request('/api/v1/deploy/ansible/playbook').then(camelizeKeys)
}

/** 获取 Ansible inventory 模板 */
export function getAnsibleInventoryTemplate(): Promise<{ content: string; path: string }> {
  return request('/api/v1/deploy/ansible/inventory-template').then(camelizeKeys)
}

/** 检查 Ansible 环境 */
export function checkAnsibleEnv(): Promise<{ installed: boolean; version: string }> {
  return request('/api/v1/deploy/ansible/env-check').then(camelizeKeys)
}

/** 获取部署计划的 Ansible 执行配置 */
export function getPlanAnsibleConfig(id: number): Promise<{ playbookPath: string; inventory: string; extraVars: Record<string, any> }> {
  return request(`/api/v1/deploy/plans/${id}/ansible-config`).then(camelizeKeys)
}
