/**
 * 部署模块类型定义
 * 与后端 deploy_service.go / deploy_plan_service.go / deploy_config_service.go 保持一致
 */

/** 部署服务器 */
export interface DeployServer {
  id: number
  name: string
  ip: string
  sshPort: number
  user: string
  authType: string // key / password
  credentialId?: number
  os?: string
  osVersion?: string
  kernel?: string
  cpuCores?: number
  memoryMb?: number
  diskGb?: number
  status: string // registered / available / unavailable
  labels?: Record<string, any>
  remark?: string
  createdAt: string
  updatedAt: string
}

/** 凭据类型常量 */
export const CREDENTIAL_TYPE_SSH = 'ssh'
export const CREDENTIAL_TYPE_KUBECONFIG = 'kubeconfig'
export const CREDENTIAL_TYPE_GIT = 'git'
export const CREDENTIAL_TYPE_DOCKER_REGISTRY = 'docker-registry'
export const CREDENTIAL_TYPE_TOKEN = 'token'

export const CREDENTIAL_TYPES = [
  CREDENTIAL_TYPE_SSH,
  CREDENTIAL_TYPE_KUBECONFIG,
  CREDENTIAL_TYPE_GIT,
  CREDENTIAL_TYPE_DOCKER_REGISTRY,
  CREDENTIAL_TYPE_TOKEN,
] as const

export type CredentialType = typeof CREDENTIAL_TYPES[number]

/** SSH 凭证 */
export interface Credential {
  id: number
  name: string
  type: string // ssh / kubeconfig / git / docker-registry / token
  authType: string // password / key（仅 type=ssh 时有意义）
  username: string
  remark?: string
  serverCount?: number
  createdAt: string
  updatedAt: string
}

/** 部署计划节点 */
export interface DeployPlanNode {
  id: number
  serverId: number
  role: string // master / worker / etcd
  sortOrder: number
}

export interface DeployPlanStepOverride {
  stepKey: string
  nodeRole?: string
  nodeServerId?: number
  commandTemplate: string
  description?: string
  timeoutSeconds?: number
  retryCount?: number
  enabled?: boolean
}

/** 部署计划（K8s 集群部署方案） */
export interface DeployPlan {
  id: number
  name: string
  clusterName: string
  k8sVersion: string
  podCidr: string
  svcCidr: string
  cniType: string
  cniConfig?: Record<string, any>
  addons?: string[]
  helmInstall?: boolean
  stepOverrides?: Record<string, DeployPlanStepOverride>
  status: string // draft / running / success / failed / cancelled
  taskId?: number
  clusterId?: number
  createdBy: number
  nodes?: DeployPlanNode[]
  createdAt: string
  updatedAt: string
}

/** 部署任务子步骤 */
export interface DeployTaskSubStep {
  key: string
  title: string
  status: string // pending / running / success / failed
  startedAt?: string
  finishedAt?: string
}

/** 部署任务步骤 */
export interface DeployTaskStep {
  key: string
  title: string
  status: string // pending / running / success / failed
  startedAt?: string
  finishedAt?: string
  message?: string
  subSteps?: DeployTaskSubStep[]
}

/** 部署任务 */
export interface DeployTask {
  id: number
  type: string
  status: string // pending / running / success / failed / canceled / timeout
  title?: string
  percent?: number
  message?: string
  meta?: Record<string, any>
  steps?: DeployTaskStep[]
  createdAt: string
  createdBy: number
}

/** 部署配置步骤 */
export interface DeployConfigStep {
  id: number
  name: string
  type: string // before_install / install / after_install / check / rollback
  cmd: string
  orderNum: number
  enabled: boolean
  createdAt: string
  updatedAt: string
}

/** 发版记录 */
export interface DeployVersion {
  id: number
  name: string
  version: string
  remark: string
  status: string // pending / success / failed
  createdAt: string
  updatedAt: string
}

/** 创建部署计划请求 */
export interface CreateDeployPlanNodeRequest {
  serverId: number
  role: 'master' | 'worker'
  sortOrder: number
}

export interface CreateDeployPlanRequest {
  name: string
  clusterName: string
  k8sVersion: string
  podCidr: string
  svcCidr: string
  cniType: string
  cniConfig?: Record<string, any>
  addons?: string[]
  helmInstall?: boolean
  stepOverrides?: Record<string, DeployPlanStepOverride>
  nodes: CreateDeployPlanNodeRequest[]
}

/** 创建服务器请求 */
export interface CreateServerRequest {
  name: string
  ip: string
  sshPort: number
  user: string
  authType: string // key / password
  credentialId?: number
  credential?: string
  labels?: Record<string, any>
  remark?: string
}

/** 创建凭证请求 */
export interface CreateCredentialRequest {
  name: string
  type: string // ssh / kubeconfig / git / docker-registry / token
  authType?: string // password / key（仅 type=ssh 时使用）
  username?: string
  credential?: string
  password?: string
  privateKey?: string
  remark?: string
}

/** 部署步骤配置（config.tsx 使用） */
export interface DeployStepConfig {
  id: number
  stepKey: string
  osType: string
  stepName: string
  stepOrder: number
  commandTemplate: string
  description?: string
  enabled: boolean
  timeoutSeconds: number
  retryCount: number
  createdBy?: number
  createdAt?: string
  updatedAt?: string
}

export interface DeployDryRunStep {
  key: string
  title: string
  description: string
  phase: string
  tasks: string[]
  appliesTo: string
  dependsOn?: string[]
}

export interface DeployDryRunNodeFlow {
  serverId: number
  serverName: string
  ip: string
  role: string
  steps: DeployDryRunStep[]
}

export interface DeployDryRunResult {
  planId: number
  planName: string
  clusterName: string
  k8sVersion: string
  cniType: string
  nodes: DeployDryRunNodeFlow[]
  summary: Record<string, number>
}

export interface DeployPreflightCheck {
  key: string
  category: 'controller' | 'plan' | 'node'
  status: 'passed' | 'warning' | 'error'
  message: string
  remediation?: string
  serverId?: number
  serverName?: string
  ignorable?: boolean
  ignored?: boolean
}

export interface DeployPreflightResult {
  ready: boolean
  checkedAt: string
  checks: DeployPreflightCheck[]
}

/** 仓库配置（config.tsx 使用） */
export interface RepositoryConfig {
  id: number
  name: string
  repoType: string // container_mirror / registry / yum / apt
  url: string
  authType: string // none / basic / token
  username?: string
  password?: string
  token?: string
  description?: string
  isDefault?: boolean
  priority?: number
  enabled?: boolean
  /** YUM 源适配的系统标识，例如 centos、rocky、almalinux、rhel、kylin。 */
  mirrorOf?: string
  createdAt?: string
  updatedAt?: string
}

/** 配置版本历史 */
export interface ConfigVersion {
  id: number
  changedAt: string
  changeType: string // create / update
  changeSummary: string
}
