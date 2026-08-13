import { request } from '@umijs/max'

export type PipelineStatus = 'idle' | 'running' | 'success' | 'failed'
export interface Pipeline { id: number | string; name: string; description?: string; triggerType?: string; trigger_type?: string; branches?: string; cron?: string; configYaml?: string; config_yaml?: string; status: PipelineStatus; lastRunId?: number; lastRun?: string; lastRunAt?: string; updatedAt?: string; updated_at?: string }
export interface Run { id: number | string; pipelineId?: number; pipelineID?: number; pipelineName?: string; pipeline_name?: string; triggerType?: string; trigger_type?: string; commitSha?: string; commit_sha?: string; commitMessage?: string; commit_message?: string; branch?: string; status: string; startedAt?: string; started_at?: string; finishedAt?: string; finished_at?: string; stages?: Stage[] }
export interface Stage { id: number; stageKey?: string; stage_key?: string; name: string; status: string; log?: string; startedAt?: string; started_at?: string; finishedAt?: string; finished_at?: string }
export interface Artifact { id: number | string; runId?: number; run_id?: number; pipelineId?: number; pipeline_id?: number; name: string; artifactType?: string; artifact_type?: string; version: string; repository?: string; sizeBytes?: number; size_bytes?: number; digest?: string; status: string; createdAt?: string; created_at?: string }
export interface Environment { id: number | string; name: string; label: string; environmentType?: string; environment_type?: string; namespace?: string; currentVersion?: string; current_version?: string; status: string; deployedBy?: string; deployed_by?: string; deployCount?: number; deploy_count?: number; lastDeployedAt?: string; last_deployed_at?: string }
export interface Page<T> { list: T[]; total: number; page: number; page_size: number }
const camelize = (v: any): any => Array.isArray(v) ? v.map(camelize) : v && typeof v === 'object' ? Object.fromEntries(Object.entries(v).map(([k,x]) => [k.replace(/_([a-z])/g,(_,c)=>c.toUpperCase()), camelize(x)])) : v
const unwrap = <T>(res: any): T => camelize(res?.data ?? res)
export const getCicdSummary = () => request('/api/v2/cicd/summary').then(unwrap)
export const getPipelines = (params?: any): Promise<Page<Pipeline>> => request('/api/v2/cicd/pipelines', { params: { ...params, page_size: params?.pageSize, trigger_type: params?.triggerType } }).then((res) => unwrap<Page<Pipeline>>(res))
export const getPipeline = (id: number | string): Promise<Pipeline> => request(`/api/v2/cicd/pipelines/${id}`).then((res) => unwrap<Pipeline>(res))
export const createPipeline = (data: any) => request('/api/v2/cicd/pipelines', { method: 'POST', data: { ...data, trigger_type: data.triggerType ?? data.trigger, cluster_id: data.clusterId, runner_image: data.runnerImage, config_yaml: data.configYaml } }).then(unwrap)
export const updatePipeline = (id: number | string, data: any) => request(`/api/v2/cicd/pipelines/${id}`, { method: 'PATCH', data: { ...data, trigger_type: data.triggerType ?? data.trigger, cluster_id: data.clusterId, runner_image: data.runnerImage, config_yaml: data.configYaml } })
export const deletePipeline = (id: number | string) => request(`/api/v2/cicd/pipelines/${id}`, { method: 'DELETE' })
export const triggerPipeline = (id: number | string, data?: any) => request(`/api/v2/cicd/pipelines/${id}/runs`, { method: 'POST', data }).then(unwrap)
export const getRuns = (params?: any): Promise<Page<Run>> => request('/api/v2/cicd/runs', { params: { ...params, page_size: params?.pageSize, pipeline_id: params?.pipelineId } }).then((res) => unwrap<Page<Run>>(res))
export const getRun = (id: number | string): Promise<Run> => request(`/api/v2/cicd/runs/${id}`).then((res) => unwrap<Run>(res))
export const cancelRun = (id: number | string) => request(`/api/v2/cicd/runs/${id}/cancellation-requests`, { method: 'POST' })
export const getArtifacts = (params?: any): Promise<Page<Artifact>> => request('/api/v2/cicd/artifacts', { params: { ...params, page_size: params?.pageSize } }).then((res) => unwrap<Page<Artifact>>(res))
export const getArtifact = (id: number | string): Promise<Artifact> => request(`/api/v2/cicd/artifacts/${id}`).then((res) => unwrap<Artifact>(res))
export const getEnvironments = (): Promise<{ list: Environment[]; total: number }> => request('/api/v2/cicd/environments').then((res) => unwrap<{ list: Environment[]; total: number }>(res))
export const createEnvironment = (data: any) => request('/api/v2/cicd/environments', { method: 'POST', data: { ...data, environment_type: data.environmentType ?? data.type, cluster_id: data.clusterId ?? data.cluster } }).then(unwrap)
export const updateEnvironment = (id: number | string, data: any) => request(`/api/v2/cicd/environments/${id}`, { method: 'PATCH', data: { ...data, environment_type: data.environmentType ?? data.type, cluster_id: data.clusterId ?? data.cluster } })
export const deleteEnvironment = (id: number | string) => request(`/api/v2/cicd/environments/${id}`, { method: 'DELETE' })
export const getEnvironment = (id: number | string): Promise<Environment> => request(`/api/v2/cicd/environments/${id}`).then((res) => unwrap<Environment>(res))
