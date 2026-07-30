/**
 * 批处理与自治资源 API：Job / CronJob / HPA / PDB
 */
import { request } from '@umijs/max'
import type {
  JobList,
  CronJobList,
  HPAList,
  PDBList,
} from '@/features/kops/types'
import { mapJob, mapCronJob, mapHPA, mapPDB, extractMappedList } from './shared'
import { applyYaml } from './generic'

// ==================== Job ====================

/** 获取 Job 列表 */
export function listJobs(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<JobList> {
  return request(`/api/v2/clusters/${clusterId}/jobs`, { params: { namespace }, signal }).then(extractMappedList(mapJob))
}

/** 删除 Job */
export function deleteJob(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v2/clusters/${clusterId}/jobs/${namespace}/${name}`, { method: 'DELETE' })
}

/** 批量删除已完成的 Job */
export function deleteCompletedJobs(clusterId: number): Promise<any> {
  return request(`/api/v2/clusters/${clusterId}/jobs/completed/deletion-requests`, { method: 'POST' })
}

// ==================== CronJob ====================

/** 获取 CronJob 列表 */
export function listCronJobs(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<CronJobList> {
  return request(`/api/v2/clusters/${clusterId}/cronjobs`, { params: { namespace }, signal }).then(extractMappedList(mapCronJob))
}

/** 删除 CronJob */
export function deleteCronJob(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v2/clusters/${clusterId}/cronjobs/${namespace}/${name}`, { method: 'DELETE' })
}

/** CronJob 手动触发 */
export function triggerCronJob(clusterId: number, namespace: string, name: string): Promise<any> {
  return request(`/api/v2/clusters/${clusterId}/cronjobs/${namespace}/${name}/execution-requests`, { method: 'POST' })
}

/** CronJob 暂停/恢复 */
export function suspendCronJob(clusterId: number, namespace: string, name: string, suspend: boolean): Promise<any> {
  return request(`/api/v2/clusters/${clusterId}/cronjobs/${namespace}/${name}/suspension-state`, { method: 'PATCH', data: { suspend } })
}

// ==================== HPA ====================

/** 构建 HPA YAML */
function buildHPAYaml(name: string, namespace: string, data: any): string {
  return [
    'apiVersion: autoscaling/v2',
    'kind: HorizontalPodAutoscaler',
    'metadata:',
    `  name: ${name}`,
    `  namespace: ${namespace}`,
    'spec:',
    '  scaleTargetRef:',
    '    apiVersion: apps/v1',
    '    kind: Deployment',
    `    name: ${data.targetName}`,
    `  minReplicas: ${Number(data.minReplicas || 1)}`,
    `  maxReplicas: ${Number(data.maxReplicas || 10)}`,
    '  metrics:',
    '    - type: Resource',
    '      resource:',
    '        name: cpu',
    '        target:',
    '          type: Utilization',
    `          averageUtilization: ${Number(data.targetCPU || 80)}`,
  ].join('\n')
}

/** 获取 HPA 列表 */
export function listHPAs(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<HPAList> {
  return request(`/api/v2/clusters/${clusterId}/hpas`, { params: { namespace }, signal }).then(extractMappedList(mapHPA))
}

/** 删除 HPA */
export function deleteHPA(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v2/clusters/${clusterId}/hpas/${namespace}/${name}`, { method: 'DELETE' })
}

/** 创建 HPA */
export function createHPA(clusterId: number, namespace: string, data: any): Promise<any> {
  return applyYaml(clusterId, buildHPAYaml(data.name, namespace, data))
}

/** 更新 HPA */
export function updateHPA(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  return request(`/api/v2/clusters/${clusterId}/hpas/${namespace}/${name}`, { method: 'PATCH', data: { namespace, yaml: buildHPAYaml(name, namespace, data) } })
}

// ==================== PDB ====================

/** 获取 PDB 列表 */
export function listPDBs(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<PDBList> {
  return request(`/api/v2/clusters/${clusterId}/pdbs`, { params: { namespace }, signal }).then(extractMappedList(mapPDB))
}

/** 删除 PDB */
export function deletePDB(clusterId: number, namespace: string, name: string): Promise<void> {
  return request(`/api/v2/clusters/${clusterId}/pdbs/${namespace}/${name}`, { method: 'DELETE' })
}
