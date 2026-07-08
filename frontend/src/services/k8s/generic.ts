/**
 * 通用资源 API：YAML 应用、通用资源列表/删除/YAML、权限审计
 * 承载 GENERIC_RESOURCE_ROUTE_MAP 路由映射表
 */
import { request } from '@umijs/max'
import { extractMappedList } from './shared'
import type { RBACMatrixRequest } from '@/types'

/** 应用 YAML 清单 */
export function applyYaml(clusterId: number, yaml: string): Promise<{ success: boolean; message: string }> {
  return request(`/api/v1/clusters/${clusterId}/manifests/apply`, { method: 'POST', data: { yaml } })
}

type GenericResourceRouteConfig = {
  path: string
  namespaced?: boolean
  workload?: boolean
}

const GENERIC_RESOURCE_ROUTE_MAP: Record<string, GenericResourceRouteConfig> = {
  pods: { path: 'pods', namespaced: true },
  podmetrics: { path: 'podmetrics', namespaced: true },
  services: { path: 'services', namespaced: true },
  configmaps: { path: 'configmaps', namespaced: true },
  secrets: { path: 'secrets', namespaced: true },
  ingresses: { path: 'ingresses', namespaced: true },
  jobs: { path: 'jobs', namespaced: true },
  cronjobs: { path: 'cronjobs', namespaced: true },
  hpas: { path: 'hpas', namespaced: true },
  pdbs: { path: 'pdbs', namespaced: true },
  replicasets: { path: 'replicasets', namespaced: true },
  networkpolicies: { path: 'networkpolicies', namespaced: true },
  'network-policies': { path: 'networkpolicies', namespaced: true },
  resourcequotas: { path: 'resourcequotas', namespaced: true },
  'resource-quotas': { path: 'resourcequotas', namespaced: true },
  serviceaccounts: { path: 'serviceaccounts', namespaced: true },
  'service-accounts': { path: 'serviceaccounts', namespaced: true },
  roles: { path: 'roles', namespaced: true },
  rolebindings: { path: 'rolebindings', namespaced: true },
  'role-bindings': { path: 'rolebindings', namespaced: true },
  pvc: { path: 'pvcs', namespaced: true },
  pvcs: { path: 'pvcs', namespaced: true },
  endpoints: { path: 'endpoints', namespaced: true },
  endpointslices: { path: 'endpointslices', namespaced: true },
  'endpoint-slices': { path: 'endpointslices', namespaced: true },
  leases: { path: 'leases', namespaced: true },
  limitranges: { path: 'limitranges', namespaced: true },
  'limit-ranges': { path: 'limitranges', namespaced: true },
  csistoragecapacities: { path: 'csistoragecapacities', namespaced: true },
  'csi-storage-capacities': { path: 'csistoragecapacities', namespaced: true },
  volumesnapshots: { path: 'volumesnapshots', namespaced: true },
  'volume-snapshots': { path: 'volumesnapshots', namespaced: true },
  deployments: { path: 'deployments', namespaced: true, workload: true },
  statefulsets: { path: 'statefulsets', namespaced: true, workload: true },
  daemonsets: { path: 'daemonsets', namespaced: true, workload: true },
  namespaces: { path: 'namespaces' },
  nodes: { path: 'nodes' },
  pv: { path: 'pvs' },
  pvs: { path: 'pvs' },
  storageclasses: { path: 'storageclasses' },
  'storage-classes': { path: 'storageclasses' },
  clusterroles: { path: 'clusterroles' },
  'cluster-roles': { path: 'clusterroles' },
  clusterrolebindings: { path: 'clusterrolebindings' },
  'cluster-role-bindings': { path: 'clusterrolebindings' },
  ingressclasses: { path: 'ingressclasses' },
  'ingress-classes': { path: 'ingressclasses' },
  apiservices: { path: 'apiservices' },
  'api-services': { path: 'apiservices' },
  priorityclasses: { path: 'priorityclasses' },
  'priority-classes': { path: 'priorityclasses' },
  runtimeclasses: { path: 'runtimeclasses' },
  'runtime-classes': { path: 'runtimeclasses' },
  csidrivers: { path: 'csidrivers' },
  'csi-drivers': { path: 'csidrivers' },
  csinodes: { path: 'csinodes' },
  'csi-nodes': { path: 'csinodes' },
  volumeattachments: { path: 'volumeattachments' },
  'volume-attachments': { path: 'volumeattachments' },
  volumesnapshotclasses: { path: 'volumesnapshotclasses' },
  'volume-snapshot-classes': { path: 'volumesnapshotclasses' },
  volumesnapshotcontents: { path: 'volumesnapshotcontents' },
  'volume-snapshot-contents': { path: 'volumesnapshotcontents' },
  customresourcedefinitions: { path: 'customresourcedefinitions' },
  crds: { path: 'customresourcedefinitions' },
  validatingwebhookconfigurations: { path: 'validatingwebhookconfigurations' },
  validatingwebhooks: { path: 'validatingwebhookconfigurations' },
  'validating-webhooks': { path: 'validatingwebhookconfigurations' },
  mutatingwebhookconfigurations: { path: 'mutatingwebhookconfigurations' },
  mutatingwebhooks: { path: 'mutatingwebhookconfigurations' },
  'mutating-webhooks': { path: 'mutatingwebhookconfigurations' },
  validatingadmissionpolicies: { path: 'validatingadmissionpolicies' },
  'validating-admission-policies': { path: 'validatingadmissionpolicies' },
  validatingadmissionpolicybindings: { path: 'validatingadmissionpolicybindings' },
  'validating-admission-policy-bindings': { path: 'validatingadmissionpolicybindings' },
}

export type GenericResourceItem = {
  name: string
  namespace?: string
  kind?: string
  status?: string
  createdAt?: string
  raw: any
}

function mapGenericResource(raw: any): GenericResourceItem {
  const metadata = raw?.metadata || {}
  const status = raw?.status || {}
  return {
    name: metadata.name || raw?.display_name || raw?.name || '',
    namespace: metadata.namespace || raw?.namespace || '',
    kind: raw?.kind || raw?.type || '',
    status: status.phase || raw?.status || status?.state || '',
    createdAt: metadata.creationTimestamp || raw?.created_at || raw?.updated_at || '',
    raw,
  }
}

function resolveGenericResourceConfig(resource: string): GenericResourceRouteConfig {
  return GENERIC_RESOURCE_ROUTE_MAP[resource] || {
    path: resource,
    namespaced: false,
  }
}

/** 通用资源列表 */
export function listGenericResources(
  clusterId: number,
  resource: string,
  namespace?: string,
  signal?: AbortSignal,
): Promise<{ items: GenericResourceItem[]; total: number; page: number; pageSize: number }> {
  const config = resolveGenericResourceConfig(resource)
  return request(`/api/v1/clusters/${clusterId}/${config.path}`, {
    params: config.namespaced ? { namespace } : undefined,
    signal,
  }).then(extractMappedList(mapGenericResource))
}

/** 通用资源删除 */
export function deleteGenericResource(
  clusterId: number,
  resource: string,
  name: string,
  namespace?: string,
): Promise<void> {
  const config = resolveGenericResourceConfig(resource)
  const endpoint = config.namespaced
    ? `/api/v1/clusters/${clusterId}/${config.path}/${namespace || 'default'}/${name}`
    : `/api/v1/clusters/${clusterId}/${config.path}/${name}`
  return request(endpoint, { method: 'DELETE' })
}

/** 权限审计列表 */
export function listPermissionAudits(
  clusterId: number,
  signal?: AbortSignal,
): Promise<{ items: any[]; total: number; page: number; pageSize: number }> {
  return request('/api/v1/permission-audits', {
    params: { cluster_id: clusterId, page: 1, page_size: 100 },
    signal,
  }).then((res: any) => ({
    items: Array.isArray(res?.items) ? res.items : [],
    total: Number(res?.total || 0),
    page: Number(res?.page || 1),
    pageSize: Number(res?.pageSize || res?.page_size || 100),
  }))
}

/** 获取 RBAC 权限矩阵默认值 */
export function defaultRBACMatrix(
  clusterId: number,
  namespaces: string[],
  signal?: AbortSignal,
): Promise<RBACMatrixRequest> {
  return request(`/api/v1/clusters/${clusterId}/permission-audits/rbac-matrix/default`, {
    params: { namespaces: namespaces.join(',') },
    signal,
  })
}

/** 根据权限矩阵生成 RBAC YAML */
export function buildRBACFromMatrix(
  clusterId: number,
  matrix: RBACMatrixRequest,
): Promise<{ yaml_content: string }> {
  return request(`/api/v1/clusters/${clusterId}/permission-audits/rbac-matrix/yaml`, {
    method: 'POST',
    data: matrix,
  })
}

/** 获取任意资源 YAML */
export function getResourceYaml(clusterId: number, resource: string, namespace: string, name: string): Promise<{ yaml: string }> {
  const config = resolveGenericResourceConfig(resource)
  const endpoint = config.workload
    ? `/api/v1/clusters/${clusterId}/workloads/${config.path}/${namespace}/${name}/yaml`
    : config.namespaced
      ? `/api/v1/clusters/${clusterId}/${config.path}/${namespace}/${name}/yaml`
      : `/api/v1/clusters/${clusterId}/${config.path}/${name}/yaml`
  return request(endpoint, {}).then((res: { text?: string; yaml?: string }) => ({
    yaml: res.yaml || res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** Helm release 列表 */
export function listHelmReleases(clusterId: number, signal?: AbortSignal): Promise<{ items: any[] }> {
  return request(`/api/v1/clusters/${clusterId}/helm/releases`, { signal }).then((res: any) => ({
    items: Array.isArray(res?.list) ? res.list : [],
  }))
}

/** Helm release 详情 */
export function getHelmReleaseDetail(
  clusterId: number,
  namespace: string,
  name: string,
  signal?: AbortSignal,
): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/helm/releases/detail`, {
    params: { namespace, name },
    signal,
  })
}

/** 节点资源使用率 */
export function listNodeMetrics(clusterId: number, signal?: AbortSignal): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/nodes/metrics`, { signal }).then((res: any) =>
    Array.isArray(res?.list) ? res.list : (Array.isArray(res) ? res : []),
  )
}

/** Pod 资源使用率（Pod 维度 CPU/内存使用量，与 pod.ts 中返回原始 PodMetrics 资源的 listPodMetrics 区分） */
export function listPodMetricsUsage(clusterId: number, signal?: AbortSignal): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/pods/metrics`, { signal }).then((res: any) =>
    Array.isArray(res?.list) ? res.list : (Array.isArray(res) ? res : []),
  )
}

/** Helm 安装 chart */
export function helmInstall(clusterId: number, data: {
  release_name: string
  namespace: string
  chart: string
  repo_url?: string
  repo_name?: string
  values_yaml?: string
}) {
  return request(`/api/v1/clusters/${clusterId}/helm/install`, { method: 'POST', data })
}

/** Helm 卸载 release */
export function helmUninstall(clusterId: number, namespace: string, name: string) {
  return request(`/api/v1/clusters/${clusterId}/helm/releases/${namespace}/${name}`, { method: 'DELETE' })
}

/** Helm 仓库列表 */
export function listHelmRepos(clusterId: number, signal?: AbortSignal): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/helm/repos`, { signal }).then((res: any) =>
    Array.isArray(res) ? res : (Array.isArray(res?.list) ? res.list : []),
  )
}

/** Helm 搜索 chart */
export function helmSearch(clusterId: number, keyword: string, signal?: AbortSignal): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/helm/search`, { params: { keyword }, signal }).then((res: any) =>
    Array.isArray(res) ? res : (Array.isArray(res?.list) ? res.list : []),
  )
}
