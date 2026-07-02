/**
 * K8s 资源 API - 支持 Mock 模式
 * 当后端不可用时，自动使用本地模拟数据
 */
import { request } from '@umijs/max'
import type {
  Pod,
  PodList,
  PodListParams,
  Deployment,
  DeploymentList,
  DeploymentListParams,
  K8sService,
  K8sServiceList,
  ConfigMap,
  ConfigMapList,
  K8sEvent,
  Namespace,
  Node,
  NodeDetail,
  Secret,
  SecretList,
  Ingress,
  IngressList,
  PersistentVolume,
  PersistentVolumeList,
  PersistentVolumeClaim,
  PersistentVolumeClaimList,
  Job,
  JobList,
  CronJob,
  CronJobList,
  HPA,
  HPAList,
  PDB,
  PDBList,
  ReplicaSet,
  ReplicaSetList,
  StorageClass,
  StorageClassList,
  ServiceAccount,
  ServiceAccountList,
  RBACRole,
  RBACRoleList,
  RBACRoleBinding,
  RBACRoleBindingList,
  NetworkPolicy,
  NetworkPolicyList,
  ResourceQuota,
  ResourceQuotaList,
} from '@/types'

const MOCK_ENABLED = false

/** 从后端响应中提取列表数据（后端返回 { list: [...] } 格式） */
function extractList<T>(res: { list?: T[] } | T[] | undefined | null): T[] {
  if (Array.isArray(res)) return res
  if (res && 'list' in res && Array.isArray(res.list)) return res.list
  return []
}

/** 从后端 { list: [...] } 格式提取分页数据，转换为 PageResult<T> 格式 */
function extractPageList<T>(res: { list?: T[] } | { items?: T[] } | T[] | undefined | null): { items: T[]; total: number; page: number; pageSize: number } {
  let items: T[] = []
  if (Array.isArray(res)) {
    items = res
  } else if (res && 'list' in res && Array.isArray(res.list)) {
    items = res.list
  } else if (res && 'items' in res && Array.isArray(res.items)) {
    items = res.items
  }
  return { items, total: items.length, page: 1, pageSize: items.length || 10 }
}

// ═══════════════════════════════════════════════════════════
//  K8s 原始对象 → 前端类型 映射函数
//  后端返回标准 K8s API 格式 { metadata, spec, status }
//  前端期望扁平结构 { name, namespace, status, ... }
// ═══════════════════════════════════════════════════════════

/** 安全取值 */
function getNested(obj: any, ...keys: string[]): any {
  let cur = obj
  for (const k of keys) {
    if (cur == null || typeof cur !== 'object') return undefined
    cur = cur[k]
  }
  return cur
}

/** 计算 Pod Ready 字符串 (如 "1/2") */
function podReadyStr(pod: any): string {
  const statuses = getNested(pod, 'status', 'containerStatuses')
  if (!Array.isArray(statuses) || statuses.length === 0) return '0/0'
  const ready = statuses.filter((s: any) => s?.ready).length
  return `${ready}/${statuses.length}`
}

/** 计算 Pod 总重启次数 */
function podRestartCount(pod: any): number {
  const statuses = getNested(pod, 'status', 'containerStatuses')
  if (!Array.isArray(statuses)) return 0
  return statuses.reduce((sum: number, s: any) => sum + (s?.restartCount || 0), 0)
}

/** 映射原始 K8s Pod 对象 → 前端 Pod 类型 */
function mapPod(raw: any): Pod {
  const m = raw?.metadata || {}
  const s = raw?.status || {}
  const spec = raw?.spec || {}
  // 提取 ownerReferences
  const ownerRef = Array.isArray(m.ownerReferences) ? m.ownerReferences[0] : undefined
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    status: s.phase || 'Unknown',
    ready: podReadyStr(raw),
    restarts: podRestartCount(raw),
    nodeName: spec.nodeName || '',
    ip: s.podIP || '',
    createdAt: m.creationTimestamp || '',
    labels: m.labels || {},
    ownerName: ownerRef?.name || '',
    ownerKind: ownerRef?.kind || '',
    qosClass: s.qosClass || '',
    annotations: m.annotations || {},
    containers: Array.isArray(spec.containers) ? spec.containers.map((c: any) => ({
      name: c.name || '',
      image: c.image || '',
      ready: Array.isArray(s.containerStatuses) ? s.containerStatuses.find((cs: any) => cs.name === c.name)?.ready : undefined,
      restartCount: Array.isArray(s.containerStatuses) ? s.containerStatuses.find((cs: any) => cs.name === c.name)?.restartCount : undefined,
    })) : [],
    conditions: Array.isArray(s.conditions) ? s.conditions.map((c: any) => ({
      type: c.type || '',
      status: c.status || '',
      lastTransitionTime: c.lastTransitionTime || '',
    })) : [],
  }
}

/** 映射原始 K8s Deployment 对象 → 前端 Deployment 类型 */
function mapDeployment(raw: any): Deployment {
  const m = raw?.metadata || {}
  const s = raw?.status || {}
  const spec = raw?.spec || {}
  const tplSpec = getNested(spec, 'template', 'spec') || {}
  const images = Array.isArray(tplSpec.containers) ? tplSpec.containers.map((c: any) => c.image || '') : []
  const containers = Array.isArray(tplSpec.containers)
    ? tplSpec.containers.map((c: any) => ({ name: c.name || '', image: c.image || '' }))
    : []
  const ready = s.readyReplicas || 0
  const total = spec.replicas || 0
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    ready: `${ready}/${total}`,
    upToDate: s.updatedReplicas || 0,
    available: s.availableReplicas || 0,
    replicas: total,
    readyReplicas: ready,
    containers,
    strategy: spec.strategy?.type || '',
    paused: spec.paused === true,
    age: m.creationTimestamp || '',
    images,
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s Service 对象 → 前端 K8sService 类型 */
function mapService(raw: any): K8sService {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  const ports = Array.isArray(s.ports) ? s.ports.map((p: any) => ({
    name: p.name || '',
    port: p.port || 0,
    targetPort: p.targetPort || 0,
    protocol: p.protocol || 'TCP',
    nodePort: p.nodePort,
  })) : []
  // 提取 externalIPs
  const externalIPs = s.externalIPs || st.loadBalancer?.ingress || []
  const externalIP = Array.isArray(externalIPs) && externalIPs.length > 0
    ? (externalIPs[0].ip || externalIPs[0].hostname || '')
    : ''
  const sel = s.selector
  const selector = sel && typeof sel === 'object' && Object.keys(sel).length > 0 ? sel : undefined
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    type: typeof s.type === 'string' ? s.type : 'ClusterIP',
    clusterIP: typeof s.clusterIP === 'string' ? s.clusterIP : '',
    externalIP: typeof externalIP === 'string' ? externalIP : '',
    ports,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
    selector,
    endpointsCount: undefined,
    sessionAffinity: typeof s.sessionAffinity === 'string' ? s.sessionAffinity : 'None',
  }
}

/** 映射原始 K8s ConfigMap 对象 → 前端 ConfigMap 类型 */
function mapConfigMap(raw: any): ConfigMap {
  const m = raw?.metadata || {}
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    data: raw?.data || {},
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s Secret 对象 → 前端 Secret 类型 */
function mapSecret(raw: any): Secret {
  const m = raw?.metadata || {}
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    type: raw?.type || '',
    dataKeys: raw?.data ? Object.keys(raw.data) : [],
    data: raw?.data,
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s Node 对象 → 前端 Node 类型 */
function mapNode(raw: any): Node {
  const m = raw?.metadata || {}
  const spec = raw?.spec || {}
  const s = raw?.status || {}
  const info = s.nodeInfo || {}
  const conditions = Array.isArray(s.conditions) ? s.conditions : []
  const readyCond = conditions.find((c: any) => c.type === 'Ready')
  const isReady = readyCond?.status === 'True'
  const isUnschedulable = spec.unschedulable === true
  const status = !isReady ? 'NotReady' : isUnschedulable ? 'SchedulingDisabled' : 'Ready'
  const roles = Object.keys(m.labels || {})
    .filter((k) => k.startsWith('node-role.kubernetes.io/'))
    .map((k) => k.split('/')[1] || '')
    .filter(Boolean)
  // 提取 InternalIP
  const addresses = Array.isArray(s.addresses) ? s.addresses : []
  const internalIP = addresses.find((a: any) => a.type === 'InternalIP')
  const ip = internalIP?.address || ''
  // 提取 taints
  const taints = Array.isArray(spec.taints) ? spec.taints.map((t: any) => ({
    key: t.key || '',
    effect: t.effect || '',
    value: t.value,
  })) : undefined
  return {
    name: m.name || '',
    status,
    roles,
    kubeletVersion: info.kubeletVersion || '',
    osImage: info.osImage || '',
    cpu: s.capacity?.cpu || '',
    memory: s.capacity?.memory || '',
    cpuPercent: 0,
    memoryPercent: 0,
    podCount: 0,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
    ip,
    podCIDR: spec.podCIDR || '',
    taints,
  }
}

/** 映射原始 K8s Event 对象 → 前端 K8sEvent 类型 */
function mapEvent(raw: any): K8sEvent {
  const m = raw?.metadata || {}
  const involved = raw?.involvedObject
  const involvedStr = involved ? `${involved.kind}/${involved.name}` : ''
  return {
    type: raw?.type || 'Normal',
    reason: raw?.reason || '',
    message: raw?.message || '',
    namespace: m.namespace || raw?.involvedObject?.namespace || '',
    involvedObject: involvedStr,
    firstTimestamp: raw?.firstTimestamp || raw?.eventTime || m.creationTimestamp || '',
    lastTimestamp: raw?.lastTimestamp || raw?.eventTime || m.creationTimestamp || '',
    count: raw?.count || 1,
  }
}

/** 映射原始 K8s Ingress 对象 → 前端 Ingress 类型 */
function mapIngress(raw: any): Ingress {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  const rules = Array.isArray(s.rules) ? s.rules : []
  const hosts = rules.map((r: any) => r.host || '').filter(Boolean)
  // 提取 addresses
  const addresses = Array.isArray(st.loadBalancer?.ingress)
    ? st.loadBalancer.ingress.map((i: any) => i.ip || i.hostname || '').filter(Boolean)
    : []
  // 提取 ports
  const tlsPorts = Array.isArray(s.tls) ? ['443'] : []
  const ports = ['80', ...tlsPorts].join(', ')
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    ingressClass: s.ingressClassName || '',
    hosts,
    addresses,
    ports,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s PV 对象 → 前端 PersistentVolume 类型 */
function mapPV(raw: any): PersistentVolume {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  return {
    name: m.name || '',
    capacity: s.capacity?.storage || '',
    accessModes: s.accessModes || [],
    reclaimPolicy: s.persistentVolumeReclaimPolicy || '',
    status: st.phase || '',
    storageClass: s.storageClassName || '',
    claim: s.claimRef ? `${s.claimRef.namespace}/${s.claimRef.name}` : '',
    reason: st.reason || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s PVC 对象 → 前端 PersistentVolumeClaim 类型 */
function mapPVC(raw: any): PersistentVolumeClaim {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    status: st.phase || '',
    volumeName: s.volumeName || '',
    capacity: st.capacity?.storage || '',
    accessModes: s.accessModes || [],
    storageClass: s.storageClassName || '',
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s Job 对象 → 前端 Job 类型 */
function mapJob(raw: any): Job {
  const m = raw?.metadata || {}
  const s = raw?.status || {}
  const spec = raw?.spec || {}
  const conditions = Array.isArray(s.conditions) ? s.conditions : []
  const completed = conditions.some((c: any) => c.type === 'Complete' && c.status === 'True')
  const failed = conditions.some((c: any) => c.type === 'Failed' && c.status === 'True')
  let status = 'Running'
  if (completed) status = 'Succeeded'
  else if (failed) status = 'Failed'
  else if (s.active > 0) status = 'Active'
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    completions: `${s.succeeded || 0}/${spec.completions || 1}`,
    parallelism: spec.parallelism || 1,
    status,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s CronJob 对象 → 前端 CronJob 类型 */
function mapCronJob(raw: any): CronJob {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    schedule: s.schedule || '',
    suspend: s.suspend || false,
    active: Array.isArray(st.active) ? st.active.length : 0,
    lastScheduleTime: st.lastScheduleTime || '',
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s HPA 对象 → 前端 HPA 类型 */
function mapHPA(raw: any): HPA {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  const ref = s.scaleTargetRef
  const reference = ref ? `${ref.kind}/${ref.name}` : ''
  // 构建 targets 字符串
  const metrics = Array.isArray(s.metrics) ? s.metrics : []
  const targetStr = metrics.map((metric: any) => {
    if (metric.type === 'Resource' && metric.resource?.targetAverageUtilization) {
      return `${metric.resource.targetAverageUtilization}%`
    }
    return ''
  }).filter(Boolean).join(', ')
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    reference,
    targets: targetStr || `${st.currentReplicas || 0} pods`,
    minPods: s.minReplicas || 1,
    maxPods: s.maxReplicas || 10,
    replicas: st.currentReplicas || 0,
    targetCPU: metrics[0]?.resource?.targetAverageUtilization,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
    targetName: ref?.name || '',
    minReplicas: s.minReplicas || 1,
    maxReplicas: s.maxReplicas || 10,
    currentReplicas: st.currentReplicas || 0,
  }
}

/** 映射原始 K8s ReplicaSet 对象 → 前端 ReplicaSet 类型 */
function mapReplicaSet(raw: any): ReplicaSet {
  const m = raw?.metadata || {}
  const s = raw?.status || {}
  const spec = raw?.spec || {}
  const tplSpec = getNested(spec, 'template', 'spec') || {}
  const images = Array.isArray(tplSpec.containers) ? tplSpec.containers.map((c: any) => c.image || '') : []
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    desired: spec.replicas || 0,
    current: s.replicas || 0,
    ready: s.readyReplicas || 0,
    age: m.creationTimestamp || '',
    images,
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s StorageClass 对象 → 前端 StorageClass 类型 */
function mapStorageClass(raw: any): StorageClass {
  const m = raw?.metadata || {}
  const annotations = m.annotations || {}
  const isDefault = annotations['storageclass.kubernetes.io/is-default-class'] === 'true' ||
                    annotations['storageclass.beta.kubernetes.io/is-default-class'] === 'true'
  return {
    name: m.name || '',
    provisioner: raw?.provisioner || '',
    reclaimPolicy: raw?.reclaimPolicy || '',
    volumeBindingMode: raw?.volumeBindingMode || '',
    allowVolumeExpansion: raw?.allowVolumeExpansion || false,
    isDefault,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 通用 RBAC Role/ClusterRole 映射 */
function mapRBACRole(raw: any): RBACRole {
  const m = raw?.metadata || {}
  const rules = Array.isArray(raw?.rules) ? raw.rules.map((r: any) => ({
    apiGroups: r.apiGroups || [],
    resources: r.resources || [],
    verbs: r.verbs || [],
  })) : []
  return {
    name: m.name || '',
    namespace: m.namespace,
    rules,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 通用 RBAC RoleBinding/ClusterRoleBinding 映射 */
function mapRBACRoleBinding(raw: any): RBACRoleBinding {
  const m = raw?.metadata || {}
  return {
    name: m.name || '',
    namespace: m.namespace,
    roleRef: raw?.roleRef?.name || '',
    subjects: Array.isArray(raw?.subjects) ? raw.subjects.map((s: any) => `${s.kind}/${s.name}`) : [],
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s NetworkPolicy 对象 → 前端 NetworkPolicy 类型 */
function mapNetworkPolicy(raw: any): NetworkPolicy {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const podSelector = s.podSelector || {}
  const selectorStr = podSelector.matchLabels
    ? Object.entries(podSelector.matchLabels).map(([k, v]) => `${k}=${v}`).join(', ')
    : ''
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    podSelector: selectorStr || '<none>',
    policyTypes: s.policyTypes || [],
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s ResourceQuota 对象 → 前端 ResourceQuota 类型 */
function mapResourceQuota(raw: any): ResourceQuota {
  const m = raw?.metadata || {}
  const h = raw?.status?.hard || {}
  const u = raw?.status?.used || {}
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    hard: h,
    used: u,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s ServiceAccount 对象 → 前端 ServiceAccount 类型 */
function mapServiceAccount(raw: any): ServiceAccount {
  const m = raw?.metadata || {}
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    secrets: Array.isArray(raw?.secrets) ? raw.secrets.length : 0,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s PDB 对象 → 前端 PDB 类型 */
function mapPDB(raw: any): PDB {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    minAvailable: String(s.minAvailable ?? ''),
    maxUnavailable: String(s.maxUnavailable ?? ''),
    allowedDisruptions: st.disruptionsAllowed || 0,
    currentHealthy: st.currentHealthy || 0,
    desiredHealthy: st.desiredHealthy || 0,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
  }
}

/** 带映射的列表提取 */
function extractMappedList<T>(mapper: (raw: any) => T) {
  return (res: any): { items: T[]; total: number; page: number; pageSize: number } => {
    const raw = extractPageList(res)
    return { ...raw, items: raw.items.map(mapper) }
  }
}

/** 带映射的简单列表提取 */
function extractMappedSimpleList<T>(mapper: (raw: any) => T) {
  return (res: any): T[] => {
    const raw = extractList(res)
    return raw.map(mapper)
  }
}

/** 模拟命名空间数据 */
const MOCK_NAMESPACES: Namespace[] = [
  { name: 'default', status: 'Active', createdAt: '2025-01-01T00:00:00Z' },
  { name: 'kube-system', status: 'Active', createdAt: '2025-01-01T00:00:00Z' },
  { name: 'monitoring', status: 'Active', createdAt: '2025-02-15T10:30:00Z' },
  { name: 'production', status: 'Active', createdAt: '2025-03-01T08:00:00Z' },
]

/** 模拟节点数据 */
const MOCK_NODES: Node[] = [
  {
    name: 'master-1',
    status: 'Ready',
    roles: ['control-plane'],
    kubeletVersion: 'v1.28.2',
    osImage: 'Ubuntu 22.04.3 LTS',
    cpu: '8',
    memory: '16Gi',
    createdAt: '2025-01-10T09:00:00Z',
  },
  {
    name: 'worker-1',
    status: 'Ready',
    roles: ['worker'],
    kubeletVersion: 'v1.28.2',
    osImage: 'Ubuntu 22.04.3 LTS',
    cpu: '16',
    memory: '32Gi',
    createdAt: '2025-01-10T09:15:00Z',
  },
  {
    name: 'worker-2',
    status: 'Ready',
    roles: ['worker'],
    kubeletVersion: 'v1.28.2',
    osImage: 'Ubuntu 22.04.3 LTS',
    cpu: '16',
    memory: '32Gi',
    createdAt: '2025-01-10T09:20:00Z',
  },
]

/** 模拟 Pod 数据 */
const MOCK_PODS: Pod[] = [
  {
    name: 'nginx-deployment-6d4f5d8c9b-abc12',
    namespace: 'default',
    status: 'Running',
    ready: '1/1',
    restarts: 0,
    nodeName: 'worker-1',
    ip: '10.244.1.5',
    createdAt: '2025-03-15T10:00:00Z',
    labels: { app: 'nginx' },
  },
  {
    name: 'redis-master-0',
    namespace: 'default',
    status: 'Running',
    ready: '1/1',
    restarts: 2,
    nodeName: 'worker-2',
    ip: '10.244.2.10',
    createdAt: '2025-03-10T14:30:00Z',
    labels: { app: 'redis', role: 'master' },
  },
  {
    name: 'prometheus-server-5d8f7c6b4d-def34',
    namespace: 'monitoring',
    status: 'Running',
    ready: '1/1',
    restarts: 0,
    nodeName: 'worker-1',
    ip: '10.244.1.15',
    createdAt: '2025-02-20T08:45:00Z',
    labels: { app: 'prometheus' },
  },
  {
    name: 'grafana-7c8b9d0e1f-ghi56',
    namespace: 'monitoring',
    status: 'Running',
    ready: '1/1',
    restarts: 1,
    nodeName: 'worker-2',
    ip: '10.244.2.25',
    createdAt: '2025-02-20T09:00:00Z',
    labels: { app: 'grafana' },
  },
  {
    name: 'backend-api-6d4f5d8c9b-jkl78',
    namespace: 'production',
    status: 'Running',
    ready: '1/1',
    restarts: 0,
    nodeName: 'worker-1',
    ip: '10.244.1.35',
    createdAt: '2025-03-05T11:15:00Z',
    labels: { app: 'backend-api' },
  },
]

/** 模拟 Deployment 数据 */
const MOCK_DEPLOYMENTS: Deployment[] = [
  {
    name: 'nginx-deployment',
    namespace: 'default',
    ready: '3/3',
    upToDate: 3,
    available: 3,
    age: '35d',
    images: ['nginx:1.24'],
    createdAt: '2025-03-15T10:00:00Z',
  },
  {
    name: 'backend-api',
    namespace: 'production',
    ready: '2/2',
    upToDate: 2,
    available: 2,
    age: '15d',
    images: ['backend-api:v1.2.0'],
    createdAt: '2025-03-05T11:15:00Z',
  },
  {
    name: 'frontend-web',
    namespace: 'production',
    ready: '2/2',
    upToDate: 2,
    available: 2,
    age: '15d',
    images: ['frontend:v1.2.0'],
    createdAt: '2025-03-05T11:30:00Z',
  },
]

/** 模拟 Service 数据 */
const MOCK_SERVICES: K8sService[] = [
  {
    name: 'kubernetes',
    namespace: 'default',
    type: 'ClusterIP',
    clusterIP: '10.96.0.1',
    ports: '443/TCP',
    age: '90d',
    createdAt: '2025-01-01T00:00:00Z',
  },
  {
    name: 'nginx-service',
    namespace: 'default',
    type: 'LoadBalancer',
    clusterIP: '10.96.0.100',
    externalIP: '192.168.1.100',
    ports: '80:30080/TCP',
    age: '35d',
    createdAt: '2025-03-15T10:05:00Z',
  },
  {
    name: 'backend-api-service',
    namespace: 'production',
    type: 'ClusterIP',
    clusterIP: '10.96.0.200',
    ports: '8080/TCP',
    age: '15d',
    createdAt: '2025-03-05T11:20:00Z',
  },
]

/** 模拟 ConfigMap 数据 */
const MOCK_CONFIGMAPS: ConfigMap[] = [
  {
    name: 'kube-root-ca.crt',
    namespace: 'default',
    data: { 'ca.crt': '-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----' },
    age: '90d',
    createdAt: '2025-01-01T00:00:00Z',
  },
  {
    name: 'nginx-config',
    namespace: 'default',
    data: { 'nginx.conf': 'server { listen 80; ... }' },
    age: '35d',
    createdAt: '2025-03-15T10:10:00Z',
  },
]

/** 模拟事件数据 */
const MOCK_EVENTS: K8sEvent[] = [
  {
    type: 'Normal',
    reason: 'Scheduled',
    message: 'Successfully assigned default/nginx-deployment-6d4f5d8c9b-abc12 to worker-1',
    namespace: 'default',
    involvedObject: 'nginx-deployment-6d4f5d8c9b-abc12',
    firstTimestamp: '2025-03-15T10:00:00Z',
    lastTimestamp: '2025-03-15T10:00:00Z',
    count: 1,
  },
  {
    type: 'Normal',
    reason: 'Pulled',
    message: 'Container image "nginx:1.24" already present on machine',
    namespace: 'default',
    involvedObject: 'nginx-deployment-6d4f5d8c9b-abc12',
    firstTimestamp: '2025-03-15T10:00:05Z',
    lastTimestamp: '2025-03-15T10:00:05Z',
    count: 1,
  },
  {
    type: 'Normal',
    reason: 'Created',
    message: 'Created container nginx',
    namespace: 'default',
    involvedObject: 'nginx-deployment-6d4f5d8c9b-abc12',
    firstTimestamp: '2025-03-15T10:00:06Z',
    lastTimestamp: '2025-03-15T10:00:06Z',
    count: 1,
  },
  {
    type: 'Normal',
    reason: 'Started',
    message: 'Started container nginx',
    namespace: 'default',
    involvedObject: 'nginx-deployment-6d4f5d8c9b-abc12',
    firstTimestamp: '2025-03-15T10:00:07Z',
    lastTimestamp: '2025-03-15T10:00:07Z',
    count: 1,
  },
]

/** 获取命名空间列表 */
export function listNamespaces(clusterId: number, signal?: AbortSignal): Promise<Namespace[]> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve([...MOCK_NAMESPACES]), 100)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/namespaces`, { signal }).then(extractMappedSimpleList((r: any) => ({
    name: r?.metadata?.name || r?.name || '',
    status: r?.status?.phase || 'Active',
    labels: r?.metadata?.labels || {},
    createdAt: r?.metadata?.creationTimestamp || '',
  })))
}

/** 获取节点列表 */
export function listNodes(clusterId: number, signal?: AbortSignal): Promise<Node[]> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve([...MOCK_NODES]), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/nodes`, { signal }).then(extractMappedSimpleList(mapNode))
}

/** 获取 Pod 列表 */
export function listPods(
  clusterId: number,
  params: PodListParams,
  signal?: AbortSignal,
): Promise<PodList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_PODS]
      if (params.namespace) {
        filtered = filtered.filter((p) => p.namespace === params.namespace)
      }
      setTimeout(() => {
        resolve({
          items: filtered,
          total: filtered.length,
          page: params.page || 1,
          pageSize: params.pageSize || 10,
        })
      }, 200)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pods`, { params, signal }).then(extractMappedList(mapPod))
}

/** 获取 Pod 详情 */
export function getPod(
  clusterId: number,
  namespace: string,
  name: string,
  signal?: AbortSignal,
): Promise<Pod> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const pod = MOCK_PODS.find((p) => p.namespace === namespace && p.name === name)
      if (pod) {
        resolve(pod)
      } else {
        reject(new Error('Pod 不存在'))
      }
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}`, { signal })
}

/** 获取 Pod YAML */
export function getPodYaml(
  clusterId: number,
  namespace: string,
  name: string,
  signal?: AbortSignal,
): Promise<{ yaml: string }> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const pod = MOCK_PODS.find((p) => p.namespace === namespace && p.name === name)
      const yaml = `apiVersion: v1
kind: Pod
metadata:
  name: ${name}
  namespace: ${namespace}
  labels:
    app: ${pod?.labels?.app || 'unknown'}
spec:
  containers:
  - name: ${name.split('-')[0]}
    image: nginx:1.24
    ports:
    - containerPort: 80
  nodeName: ${pod?.nodeName || 'worker-1'}`
      setTimeout(() => resolve({ yaml }), 100)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/yaml`, { signal }).then((res: { text?: string; yaml?: string }) => ({
    yaml: res.yaml || res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** 删除 Pod */
export function deletePod(clusterId: number, namespace: string, name: string, _force?: boolean): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const index = MOCK_PODS.findIndex((p) => p.namespace === namespace && p.name === name)
      if (index !== -1) {
        MOCK_PODS.splice(index, 1)
        resolve()
      } else {
        reject(new Error('Pod 不存在'))
      }
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}`, { method: 'DELETE' })
}

/** 获取 Deployment 列表 */
export function listDeployments(
  clusterId: number,
  params: DeploymentListParams & { kind?: 'Deployment' | 'StatefulSet' | 'DaemonSet' },
  signal?: AbortSignal,
): Promise<DeploymentList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_DEPLOYMENTS]
      if (params.namespace) {
        filtered = filtered.filter((d) => d.namespace === params.namespace)
      }
      setTimeout(() => {
        resolve({
          items: filtered,
          total: filtered.length,
          page: params.page || 1,
          pageSize: params.pageSize || 10,
        })
      }, 200)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/workloads`, { params, signal }).then(extractMappedList(mapDeployment))
}

/** 删除 Deployment */
export function deleteDeployment(
  clusterId: number,
  namespace: string,
  name: string,
  kind: 'Deployment' | 'StatefulSet' | 'DaemonSet' = 'Deployment',
): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const index = MOCK_DEPLOYMENTS.findIndex((d) => d.namespace === namespace && d.name === name)
      if (index !== -1) {
        MOCK_DEPLOYMENTS.splice(index, 1)
        resolve()
      } else {
        reject(new Error('Deployment 不存在'))
      }
    })
  }
  return request(`/api/v1/clusters/${clusterId}/workloads/${kind}/${namespace}/${name}`, {
    method: 'DELETE',
  })
}

/** 更新 Deployment 副本数 */
export function scaleDeployment(
  clusterId: number,
  namespace: string,
  name: string,
  replicas: number,
  kind: 'Deployment' | 'StatefulSet' | 'DaemonSet' = 'Deployment',
): Promise<Deployment> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const deployment = MOCK_DEPLOYMENTS.find((d) => d.namespace === namespace && d.name === name)
      if (deployment) {
        deployment.ready = `${replicas}/${replicas}`
        deployment.upToDate = replicas
        deployment.available = replicas
        resolve(deployment)
      } else {
        reject(new Error('Deployment 不存在'))
      }
    })
  }
  return request(`/api/v1/clusters/${clusterId}/workloads/scale`, {
    method: 'PATCH',
    data: { kind, namespace, name, replicas },
  })
}

/** 回滚 Deployment */
export function rollbackDeployment(
  clusterId: number,
  namespace: string,
  name: string,
  revision: number,
): Promise<Deployment> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const deployment = MOCK_DEPLOYMENTS.find((d) => d.namespace === namespace && d.name === name)
      if (deployment) {
        resolve(deployment)
      } else {
        reject(new Error('Deployment 不存在'))
      }
    })
  }
  return request(`/api/v1/clusters/${clusterId}/workloads/deployments/${namespace}/${name}/rollout-undo`, {
    method: 'POST',
    data: { revision },
  })
}

/** 获取 Service 列表 */
export function listServices(
  clusterId: number,
  params: { namespace: string },
  signal?: AbortSignal,
): Promise<K8sServiceList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_SERVICES]
      if (params.namespace) {
        filtered = filtered.filter((s) => s.namespace === params.namespace)
      }
      setTimeout(() => {
        resolve({
          items: filtered,
          total: filtered.length,
          page: 1,
          pageSize: 10,
        })
      }, 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/services`, { params, signal }).then(extractMappedList(mapService))
}

/** 删除 Service */
export function deleteService(
  clusterId: number,
  namespace: string,
  name: string,
): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const index = MOCK_SERVICES.findIndex((s) => s.namespace === namespace && s.name === name)
      if (index !== -1) {
        MOCK_SERVICES.splice(index, 1)
        resolve()
      } else {
        reject(new Error('Service 不存在'))
      }
    })
  }
  return request(`/api/v1/clusters/${clusterId}/services/${namespace}/${name}`, {
    method: 'DELETE',
  })
}

/** 获取 ConfigMap 列表 */
export function listConfigMaps(
  clusterId: number,
  params: { namespace: string },
  signal?: AbortSignal,
): Promise<ConfigMapList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_CONFIGMAPS]
      if (params.namespace) {
        filtered = filtered.filter((c) => c.namespace === params.namespace)
      }
      setTimeout(() => {
        resolve({
          items: filtered,
          total: filtered.length,
          page: 1,
          pageSize: 10,
        })
      }, 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/configmaps`, { params, signal }).then(extractMappedList(mapConfigMap))
}

/** 删除 ConfigMap */
export function deleteConfigMap(
  clusterId: number,
  namespace: string,
  name: string,
): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const index = MOCK_CONFIGMAPS.findIndex((c) => c.namespace === namespace && c.name === name)
      if (index !== -1) {
        MOCK_CONFIGMAPS.splice(index, 1)
        resolve()
      } else {
        reject(new Error('ConfigMap 不存在'))
      }
    })
  }
  return request(`/api/v1/clusters/${clusterId}/configmaps/${namespace}/${name}`, {
    method: 'DELETE',
  })
}

/** 获取事件列表 */
export function listEvents(
  clusterId: number,
  namespace?: string,
  signal?: AbortSignal,
): Promise<K8sEvent[]> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_EVENTS]
      if (namespace) {
        filtered = filtered.filter((e) => e.namespace === namespace)
      }
      setTimeout(() => resolve(filtered), 100)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/events`, { params: { namespace }, signal }).then(extractMappedSimpleList(mapEvent))
}

/** 获取仪表盘统计数据 */
export function getDashboardStats(
  clusterId: number,
  signal?: AbortSignal,
): Promise<{
  namespaceCount: number
  podCount: number
  deploymentCount: number
  serviceCount: number
  cpuUsage: number
  memoryUsage: number
  podRunning: number
  podPending: number
  podFailed: number
}> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          namespaceCount: MOCK_NAMESPACES.length,
          podCount: MOCK_PODS.length,
          deploymentCount: MOCK_DEPLOYMENTS.length,
          serviceCount: MOCK_SERVICES.length,
          cpuUsage: 45,
          memoryUsage: 62,
          podRunning: MOCK_PODS.filter((p) => p.status === 'Running').length,
          podPending: MOCK_PODS.filter((p) => p.status === 'Pending').length,
          podFailed: MOCK_PODS.filter((p) => p.status === 'Failed').length,
        })
      }, 200)
    })
  }
  return request(`/api/v1/dashboard/clusters/${clusterId}/overview`, { signal })
}

/** 集群概览 - 完整仪表盘数据 */
export interface ClusterOverview {
  cluster: { name: string; status: string; api_ok?: boolean; k8s_version?: string }
  stats: {
    nodes: { total: number; ready: number }
    pods: { total: number; running: number; pending: number; failed: number; succeeded: number }
    cpu: { used_percent: number }
    memory: { used_percent: number }
    workloads: { deployments: number; statefulsets: number; daemonsets: number; replicasets?: number }
    namespaces?: number
  }
  charts: {
    cpu_memory_24h: { labels: string[]; cpu: number[]; memory: number[] }
    pod_phase: { running: number; pending: number; failed: number; succeeded: number }
    namespace_pods_top: Array<{ namespace: string; pods: number }>
    node_ready: { ready: number; total: number }
  }
  risks?: {
    certificates: Array<{ key: string; name: string; component: string; purpose: string; not_after?: string; days_left?: number; status: string }>
  }
  anomalies?: {
    failed_pods?: Array<{ name: string; namespace: string; reason: string }>
  }
  events?: K8sEvent[]
}

export function getClusterOverview(clusterId: number, signal?: AbortSignal): Promise<ClusterOverview> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const labels = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}:00`)
      const cpu = Array.from({ length: 24 }, () => Math.round(30 + Math.random() * 40))
      const memory = Array.from({ length: 24 }, () => Math.round(45 + Math.random() * 30))
      const failedPods = MOCK_PODS.filter((p) => p.status === 'Failed')
      setTimeout(() => {
        resolve({
          cluster: { name: 'k8s-prod-cluster', status: 'Active' },
          stats: {
            nodes: { total: MOCK_NODES.length, ready: MOCK_NODES.filter((n) => n.status === 'Ready').length },
            pods: {
              total: MOCK_PODS.length,
              running: MOCK_PODS.filter((p) => p.status === 'Running').length,
              pending: MOCK_PODS.filter((p) => p.status === 'Pending').length,
              failed: failedPods.length,
              succeeded: 0,
            },
            cpu: { used_percent: cpu[cpu.length - 1]! },
            memory: { used_percent: memory[memory.length - 1]! },
            workloads: {
              deployments: MOCK_DEPLOYMENTS.length,
              statefulsets: 2,
              daemonsets: 1,
              replicasets: MOCK_REPLICASETS.length,
            },
            namespaces: MOCK_NAMESPACES.length,
          },
          charts: {
            cpu_memory_24h: { labels, cpu, memory },
            pod_phase: {
              running: MOCK_PODS.filter((p) => p.status === 'Running').length,
              pending: MOCK_PODS.filter((p) => p.status === 'Pending').length,
              failed: failedPods.length,
              succeeded: 0,
            },
            namespace_pods_top: [
              { namespace: 'default', pods: 3 },
              { namespace: 'production', pods: 2 },
              { namespace: 'monitoring', pods: 2 },
              { namespace: 'kube-system', pods: 1 },
            ],
            node_ready: { ready: MOCK_NODES.filter((n) => n.status === 'Ready').length, total: MOCK_NODES.length },
          },
          risks: {
            certificates: [
              { key: 'apiserver', name: 'API Server', component: 'kube-apiserver', purpose: 'API Server TLS', not_after: '2026-01-15T00:00:00Z', days_left: 240, status: 'ok' },
              { key: 'etcd', name: 'etcd CA', component: 'etcd', purpose: 'etcd peer/client', not_after: '2025-12-01T00:00:00Z', days_left: 195, status: 'ok' },
              { key: 'front-proxy', name: 'Front Proxy', component: 'kube-apiserver', purpose: 'Aggregation layer', not_after: '2025-07-01T00:00:00Z', days_left: 101, status: 'warn' },
            ],
          },
          anomalies: {
            failed_pods: failedPods.map((p) => ({ name: p.name, namespace: p.namespace, reason: 'CrashLoopBackOff' })),
          },
          events: MOCK_EVENTS,
        })
      }, 300)
    })
  }
  return request(`/api/v1/dashboard/clusters/${clusterId}/overview`, { signal })
}

/** 获取集群拓扑数据 - 从 nodes、pods、services 端点构造 */
export async function getTopology(clusterId: number, signal?: AbortSignal): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          nodes: MOCK_NODES.map((n) => ({
            id: n.name,
            type: 'node',
            data: { label: n.name, status: n.status },
          })),
          edges: MOCK_PODS.map((p) => ({
            id: `${p.nodeName}-${p.name}`,
            source: p.nodeName,
            target: p.name,
          })),
        })
      }, 300)
    })
  }
  // 后端没有拓扑端点，从 nodes、pods、services 构造拓扑数据
  const [nodes, pods, services] = await Promise.all([
    request(`/api/v1/clusters/${clusterId}/nodes`, { signal }).then(extractMappedSimpleList(mapNode)),
    request(`/api/v1/clusters/${clusterId}/pods`, { signal }).then(extractMappedSimpleList(mapPod)),
    request(`/api/v1/clusters/${clusterId}/services`, { signal }).then(extractMappedSimpleList(mapService)),
  ])

  const topologyNodes: any[] = []
  const topologyEdges: any[] = []

  // 添加节点
  for (const node of nodes) {
    topologyNodes.push({
      id: node.name,
      type: 'node',
      data: { label: node.name, status: node.status, roles: node.roles, version: node.kubeletVersion },
    })
  }

  // 添加 Pod 和边
  for (const pod of pods) {
    topologyNodes.push({
      id: `${pod.namespace}-${pod.name}`,
      type: 'pod',
      data: { label: pod.name, namespace: pod.namespace, status: pod.status, nodeName: pod.nodeName },
    })
    if (pod.nodeName) {
      topologyEdges.push({
        id: `${pod.nodeName}-${pod.namespace}-${pod.name}`,
        source: pod.nodeName,
        target: `${pod.namespace}-${pod.name}`,
      })
    }
  }

  // 添加 Service
  for (const svc of services) {
    topologyNodes.push({
      id: `svc-${svc.namespace}-${svc.name}`,
      type: 'service',
      data: { label: svc.name, namespace: svc.namespace, type: svc.type, clusterIP: svc.clusterIP },
    })
  }

  return { nodes: topologyNodes, edges: topologyEdges }
}

// ==================== Secret ====================

const MOCK_SECRETS: Secret[] = [
  { name: 'default-token-abc', namespace: 'default', type: 'Opaque', dataKeys: ['token', 'ca.crt'], createdAt: '2025-01-01T00:00:00Z' },
  { name: 'harbor-credentials', namespace: 'production', type: 'kubernetes.io/dockerconfigjson', dataKeys: ['.dockerconfigjson'], createdAt: '2025-03-10T09:30:00Z' },
  { name: 'tls-secret', namespace: 'production', type: 'kubernetes.io/tls', dataKeys: ['tls.crt', 'tls.key'], createdAt: '2025-03-15T08:00:00Z' },
]

export function listSecrets(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<SecretList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_SECRETS]
      if (namespace) filtered = filtered.filter((s) => s.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/secrets`, { params: { namespace }, signal }).then(extractMappedList(mapSecret))
}

export function deleteSecret(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const idx = MOCK_SECRETS.findIndex((s) => s.namespace === namespace && s.name === name)
      if (idx !== -1) MOCK_SECRETS.splice(idx, 1)
      resolve()
    })
  }
  return request(`/api/v1/clusters/${clusterId}/secrets/${namespace}/${name}`, { method: 'DELETE' })
}

// ==================== Ingress ====================

const MOCK_INGRESSES: Ingress[] = [
  { name: 'nginx-ingress', namespace: 'production', ingressClass: 'nginx', hosts: ['app.example.com'], addresses: ['192.168.1.100'], ports: '80, 443', age: '30d', createdAt: '2025-02-20T10:00:00Z' },
  { name: 'api-ingress', namespace: 'production', ingressClass: 'nginx', hosts: ['api.example.com'], addresses: ['192.168.1.100'], ports: '443', age: '15d', createdAt: '2025-03-05T11:00:00Z' },
]

export function listIngresses(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<IngressList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_INGRESSES]
      if (namespace) filtered = filtered.filter((i) => i.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/ingresses`, { params: { namespace }, signal }).then(extractMappedList(mapIngress))
}

export function deleteIngress(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const idx = MOCK_INGRESSES.findIndex((i) => i.namespace === namespace && i.name === name)
      if (idx !== -1) MOCK_INGRESSES.splice(idx, 1)
      resolve()
    })
  }
  return request(`/api/v1/clusters/${clusterId}/ingresses/${namespace}/${name}`, { method: 'DELETE' })
}

// ==================== PersistentVolume ====================

const MOCK_PVS: PersistentVolume[] = [
  { name: 'pv-data-01', capacity: '100Gi', accessModes: ['ReadWriteOnce'], reclaimPolicy: 'Retain', status: 'Bound', claim: 'production/data-claim', storageClass: 'standard', reason: '', createdAt: '2025-01-20T08:00:00Z' },
  { name: 'pv-shared-01', capacity: '50Gi', accessModes: ['ReadWriteMany'], reclaimPolicy: 'Delete', status: 'Available', claim: '', storageClass: 'nfs', reason: '', createdAt: '2025-02-10T14:00:00Z' },
]

export function listPersistentVolumes(clusterId: number, signal?: AbortSignal): Promise<PersistentVolumeList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({ items: MOCK_PVS, total: MOCK_PVS.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pvs`, { signal }).then(extractMappedList(mapPV))
}

export function deletePersistentVolume(clusterId: number, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const idx = MOCK_PVS.findIndex((pv) => pv.name === name)
      if (idx !== -1) MOCK_PVS.splice(idx, 1)
      resolve()
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pvs/${name}`, { method: 'DELETE' })
}

// ==================== PersistentVolumeClaim ====================

const MOCK_PVCS: PersistentVolumeClaim[] = [
  { name: 'data-claim', namespace: 'production', status: 'Bound', volumeName: 'pv-data-01', capacity: '100Gi', accessModes: ['ReadWriteOnce'], storageClass: 'standard', age: '60d', createdAt: '2025-01-20T08:00:00Z' },
  { name: 'redis-data', namespace: 'default', status: 'Bound', volumeName: 'pv-data-02', capacity: '10Gi', accessModes: ['ReadWriteOnce'], storageClass: 'standard', age: '35d', createdAt: '2025-02-15T10:00:00Z' },
]

export function listPersistentVolumeClaims(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<PersistentVolumeClaimList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_PVCS]
      if (namespace) filtered = filtered.filter((p) => p.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pvcs`, { params: { namespace }, signal }).then(extractMappedList(mapPVC))
}

export function deletePersistentVolumeClaim(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const idx = MOCK_PVCS.findIndex((p) => p.namespace === namespace && p.name === name)
      if (idx !== -1) MOCK_PVCS.splice(idx, 1)
      resolve()
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pvcs/${namespace}/${name}`, { method: 'DELETE' })
}

// ==================== Node Detail ====================

export function getNodeDetail(clusterId: number, name: string, signal?: AbortSignal): Promise<NodeDetail> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const node = MOCK_NODES.find((n) => n.name === name) || MOCK_NODES[0]!
      setTimeout(() => {
        resolve({
          ...node,
          addresses: [
            { type: 'InternalIP', address: '192.168.1.10' },
            { type: 'Hostname', address: node.name },
          ],
          conditions: [
            { type: 'Ready', status: 'True', message: 'kubelet is posting ready status', lastTransitionTime: '2025-01-10T09:00:00Z' },
            { type: 'MemoryPressure', status: 'False', message: '', lastTransitionTime: '2025-01-10T09:00:00Z' },
            { type: 'DiskPressure', status: 'False', message: '', lastTransitionTime: '2025-01-10T09:00:00Z' },
          ],
          images: [
            { names: ['nginx:1.24', 'redis:7.0'], sizeBytes: 150000000 },
            { names: ['prometheus:v2.45.0'], sizeBytes: 200000000 },
          ],
          allocatable: { cpu: node.cpu, memory: node.memory, pods: '110' },
          capacity: { cpu: node.cpu, memory: node.memory, pods: '110' },
        })
      }, 200)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/detail`, { signal }).then((raw: any) => {
    const base = mapNode(raw)
    const s = raw?.status || {}
    const addresses = Array.isArray(s.addresses) ? s.addresses.map((a: any) => ({ type: a.type || '', address: a.address || '' })) : []
    const conditions = Array.isArray(s.conditions) ? s.conditions.map((c: any) => ({
      type: c.type || '', status: c.status || '', reason: c.reason || '', message: c.message || '', lastTransitionTime: c.lastTransitionTime || '',
    })) : []
    const images = Array.isArray(s.images) ? s.images.map((img: any) => ({ names: img.names || [], sizeBytes: img.sizeBytes || 0 })) : []
    const allocatable = s.allocatable || {}
    const capacity = s.capacity || {}
    return { ...base, addresses, conditions, images, allocatable, capacity } as NodeDetail
  })
}

/** 停止节点调度 (Cordon) */
export function cordonNode(clusterId: number, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const node = MOCK_NODES.find((n) => n.name === name)
      if (node) node.status = 'SchedulingDisabled'
      setTimeout(resolve, 200)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/cordon`, { method: 'POST' })
}

/** 恢复节点调度 (Uncordon) */
export function uncordonNode(clusterId: number, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const node = MOCK_NODES.find((n) => n.name === name)
      if (node) node.status = 'Ready'
      setTimeout(resolve, 200)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/uncordon`, { method: 'POST' })
}

/** 驱逐节点 (Drain) */
export function drainNode(clusterId: number, name: string, options?: { force?: boolean; timeout_seconds?: number; ignore_daemonsets?: boolean }): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => setTimeout(resolve, 500))
  }
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/drain`, { method: 'POST', data: options })
}

/** 删除节点 */
export function deleteNode(clusterId: number, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const idx = MOCK_NODES.findIndex((n) => n.name === name)
      if (idx >= 0) MOCK_NODES.splice(idx, 1)
      setTimeout(resolve, 200)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}`, { method: 'DELETE' })
}

// ==================== Job ====================

const MOCK_JOBS: Job[] = [
  { name: 'db-migration', namespace: 'production', completions: '1/1', parallelism: 1, status: 'Succeeded', age: '5d', createdAt: '2025-03-15T14:00:00Z' },
  { name: 'batch-report', namespace: 'default', completions: '0/1', parallelism: 1, status: 'Active', age: '1h', createdAt: '2025-03-20T09:00:00Z' },
]

export function listJobs(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<JobList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_JOBS]
      if (namespace) filtered = filtered.filter((j) => j.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/jobs`, { params: { namespace }, signal }).then(extractMappedList(mapJob))
}

// ==================== CronJob ====================

const MOCK_CRONJOBS: CronJob[] = [
  { name: 'nightly-backup', namespace: 'production', schedule: '0 2 * * *', suspend: false, active: 0, lastScheduleTime: '2025-03-20T02:00:00Z', age: '30d', createdAt: '2025-02-18T08:00:00Z' },
  { name: 'cleanup-logs', namespace: 'default', schedule: '0 */6 * * *', suspend: false, active: 0, lastScheduleTime: '2025-03-20T06:00:00Z', age: '15d', createdAt: '2025-03-05T10:00:00Z' },
]

export function listCronJobs(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<CronJobList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_CRONJOBS]
      if (namespace) filtered = filtered.filter((c) => c.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/cronjobs`, { params: { namespace }, signal }).then(extractMappedList(mapCronJob))
}

// ==================== HPA ====================

const MOCK_HPAS: HPA[] = [
  { name: 'nginx-hpa', namespace: 'production', reference: 'Deployment/nginx-deployment', targets: '50%/80%', minPods: 2, maxPods: 10, replicas: 3, age: '20d', createdAt: '2025-02-28T10:00:00Z' },
]

export function listHPAs(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<HPAList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_HPAS]
      if (namespace) filtered = filtered.filter((h) => h.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/hpas`, { params: { namespace }, signal }).then(extractMappedList(mapHPA))
}

// ==================== PDB ====================

const MOCK_PDBS: PDB[] = [
  { name: 'nginx-pdb', namespace: 'production', minAvailable: '1', maxUnavailable: '', allowedDisruptions: 2, currentHealthy: 3, desiredHealthy: 1, age: '20d', createdAt: '2025-02-28T10:00:00Z' },
]

export function listPDBs(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<PDBList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_PDBS]
      if (namespace) filtered = filtered.filter((p) => p.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pdbs`, { params: { namespace }, signal }).then(extractMappedList(mapPDB))
}

// ==================== ReplicaSet ====================

const MOCK_REPLICASETS: ReplicaSet[] = [
  { name: 'nginx-deployment-6d4f5d8c9b', namespace: 'production', desired: 3, current: 3, ready: 3, age: '10d', images: ['nginx:1.24'], createdAt: '2025-03-10T10:00:00Z' },
  { name: 'backend-api-7b8c9d0e1f', namespace: 'production', desired: 2, current: 2, ready: 2, age: '5d', images: ['backend-api:v1.2.0'], createdAt: '2025-03-15T11:00:00Z' },
]

export function listReplicaSets(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<ReplicaSetList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_REPLICASETS]
      if (namespace) filtered = filtered.filter((r) => r.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/replicasets`, { params: { namespace }, signal }).then(extractMappedList(mapReplicaSet))
}

// ==================== StorageClass ====================

const MOCK_STORAGECLASSES: StorageClass[] = [
  { name: 'standard', provisioner: 'kubernetes.io/aws-ebs', reclaimPolicy: 'Delete', volumeBindingMode: 'WaitForFirstConsumer', allowVolumeExpansion: true, age: '90d', createdAt: '2025-01-01T00:00:00Z' },
  { name: 'nfs', provisioner: 'nfs-subdir-external-provisioner', reclaimPolicy: 'Retain', volumeBindingMode: 'Immediate', allowVolumeExpansion: false, age: '60d', createdAt: '2025-01-20T08:00:00Z' },
]

export function listStorageClasses(clusterId: number, signal?: AbortSignal): Promise<StorageClassList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({ items: MOCK_STORAGECLASSES, total: MOCK_STORAGECLASSES.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/storageclasses`, { signal }).then(extractMappedList(mapStorageClass))
}

// ==================== ServiceAccount ====================

const MOCK_SERVICEACCOUNTS: ServiceAccount[] = [
  { name: 'default', namespace: 'default', secrets: 1, age: '90d', createdAt: '2025-01-01T00:00:00Z' },
  { name: 'deployer', namespace: 'production', secrets: 2, age: '30d', createdAt: '2025-02-18T08:00:00Z' },
]

export function listServiceAccounts(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<ServiceAccountList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_SERVICEACCOUNTS]
      if (namespace) filtered = filtered.filter((s) => s.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/serviceaccounts`, { params: { namespace }, signal }).then(extractMappedList(mapServiceAccount))
}

// ==================== RBAC ====================

const MOCK_ROLES: RBACRole[] = [
  { name: 'pod-reader', namespace: 'default', rules: [{ apiGroups: [''], resources: ['pods'], verbs: ['get', 'list', 'watch'] }], age: '60d', createdAt: '2025-01-20T08:00:00Z' },
  { name: 'cluster-admin', rules: [{ apiGroups: ['*'], resources: ['*'], verbs: ['*'] }], age: '90d', createdAt: '2025-01-01T00:00:00Z' },
]

const MOCK_ROLEBINDINGS: RBACRoleBinding[] = [
  { name: 'read-pods-binding', namespace: 'default', roleRef: 'pod-reader', subjects: ['User:dev-user'], age: '60d', createdAt: '2025-01-20T08:00:00Z' },
]

export function listRoles(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<RBACRoleList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const filtered = MOCK_ROLES.filter((r) => namespace ? r.namespace === namespace : !r.namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/roles`, { params: { namespace }, signal }).then(extractMappedList(mapRBACRole))
}

export function listClusterRoles(clusterId: number, signal?: AbortSignal): Promise<RBACRoleList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const filtered = MOCK_ROLES.filter((r) => !r.namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/clusterroles`, { signal }).then(extractMappedList(mapRBACRole))
}

export function listRoleBindings(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<RBACRoleBindingList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const filtered = MOCK_ROLEBINDINGS.filter((r) => namespace ? r.namespace === namespace : !r.namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/rolebindings`, { params: { namespace }, signal }).then(extractMappedList(mapRBACRoleBinding))
}

export function listClusterRoleBindings(clusterId: number, signal?: AbortSignal): Promise<RBACRoleBindingList> {
  if (MOCK_ENABLED) {
    const clusterBindings: RBACRoleBinding[] = [
      { name: 'cluster-admin-binding', roleRef: 'cluster-admin', subjects: ['kubernetes-admin'], age: '170d', createdAt: '2025-01-01T00:00:00Z' },
      { name: 'view-binding', roleRef: 'view', subjects: ['system:anonymous'], age: '139d', createdAt: '2025-02-01T00:00:00Z' },
    ]
    return new Promise((resolve) => {
      setTimeout(() => resolve({ items: clusterBindings, total: clusterBindings.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/clusterrolebindings`, { signal }).then(extractMappedList(mapRBACRoleBinding))
}

// ==================== NetworkPolicy ====================

const MOCK_NETWORKPOLICIES: NetworkPolicy[] = [
  { name: 'deny-all', namespace: 'production', podSelector: '', policyTypes: ['Ingress', 'Egress'], age: '30d', createdAt: '2025-02-18T08:00:00Z' },
]

export function listNetworkPolicies(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<NetworkPolicyList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_NETWORKPOLICIES]
      if (namespace) filtered = filtered.filter((n) => n.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/networkpolicies`, { params: { namespace }, signal }).then(extractMappedList(mapNetworkPolicy))
}

// ==================== ResourceQuota ====================

const MOCK_QUOTAS: ResourceQuota[] = [
  { name: 'compute-quota', namespace: 'production', hard: { 'requests.cpu': '8', 'requests.memory': '16Gi', 'limits.cpu': '16', 'limits.memory': '32Gi' }, used: { 'requests.cpu': '4', 'requests.memory': '8Gi', 'limits.cpu': '8', 'limits.memory': '16Gi' }, age: '60d', createdAt: '2025-01-20T08:00:00Z' },
]

export function listResourceQuotas(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<ResourceQuotaList> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      let filtered = [...MOCK_QUOTAS]
      if (namespace) filtered = filtered.filter((q) => q.namespace === namespace)
      setTimeout(() => resolve({ items: filtered, total: filtered.length, page: 1, pageSize: 10 }), 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/resourcequotas`, { params: { namespace }, signal }).then(extractMappedList(mapResourceQuota))
}

// ==================== YAML Apply ====================

export function applyYaml(clusterId: number, yaml: string): Promise<{ success: boolean; message: string }> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({ success: true, message: '资源已成功应用' }), 500)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/manifests/apply`, { method: 'POST', data: { yaml } })
}

// ==================== 获取资源 YAML ====================

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
  'validating-webhooks': { path: 'validatingwebhookconfigurations' },
  mutatingwebhookconfigurations: { path: 'mutatingwebhookconfigurations' },
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

export function getResourceYaml(clusterId: number, resource: string, namespace: string, name: string): Promise<{ yaml: string }> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const yaml = `apiVersion: v1
kind: ${resource}
metadata:
  name: ${name}
  namespace: ${namespace}`
      setTimeout(() => resolve({ yaml }), 100)
    })
  }
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

// ==================== 创建操作 ====================

export function createDeployment(
  clusterId: number,
  namespace: string,
  data: any,
  kind: 'Deployment' | 'StatefulSet' | 'DaemonSet' = 'Deployment',
): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          apiVersion: 'apps/v1',
          kind: 'Deployment',
          metadata: {
            name: data.metadata?.name || 'new-deployment',
            namespace,
            uid: `uid-${Date.now()}`,
            creationTimestamp: new Date().toISOString(),
          },
          spec: data.spec,
          status: { replicas: 1, readyReplicas: 0, availableReplicas: 0 },
        });
      }, 500);
    });
  }
  const endpointMap = {
    Deployment: 'deployments',
    StatefulSet: 'statefulsets',
    DaemonSet: 'daemonsets',
  } as const
  const payload = {
    namespace,
    name: data.name,
    replicas: Number(data.replicas || 1),
    labels: (() => {
      if (!data.labels) return undefined
      if (typeof data.labels === 'string') {
        try {
          return JSON.parse(data.labels)
        } catch {
          return undefined
        }
      }
      return data.labels
    })(),
    containers: [
      {
        name: data.name,
        image: data.image,
        command: data.command,
        cpu: data.cpu,
        memory: data.memory,
      },
    ],
  }
  return request(`/api/v1/clusters/${clusterId}/workloads/${endpointMap[kind]}`, { method: 'POST', data: payload })
}

export function restartDeployment(
  clusterId: number,
  namespace: string,
  name: string,
  kind: 'Deployment' | 'StatefulSet' | 'DaemonSet' = 'Deployment',
): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({ message: `Deployment ${name} restart triggered` });
      }, 300);
    });
  }
  return request(`/api/v1/clusters/${clusterId}/workloads/restart`, { method: 'PATCH', data: { kind, namespace, name } })
}

export function createService(clusterId: number, namespace: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          apiVersion: 'v1',
          kind: 'Service',
          metadata: {
            name: data.metadata?.name || 'new-service',
            namespace,
            uid: `uid-${Date.now()}`,
            creationTimestamp: new Date().toISOString(),
          },
          spec: data.spec,
        });
      }, 500);
    });
  }
  const selector = (() => {
    if (!data.selector) return undefined
    if (typeof data.selector === 'string') {
      try {
        return JSON.parse(data.selector)
      } catch {
        return undefined
      }
    }
    return data.selector
  })()
  return request(`/api/v1/clusters/${clusterId}/services`, {
    method: 'POST',
    data: {
      namespace,
      name: data.name,
      type: data.type,
      selector,
      ports: [
        {
          name: data.portName,
          port: Number(data.port || 80),
          target_port: Number(data.targetPort || data.port || 80),
          protocol: data.protocol || 'TCP',
        },
      ],
    },
  })
}

export function updateService(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          apiVersion: 'v1',
          kind: 'Service',
          metadata: { name, namespace, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
          spec: data.spec || data,
        });
      }, 300);
    });
  }
  const selector = (() => {
    if (!data.selector) return undefined
    if (typeof data.selector === 'string') {
      try {
        return JSON.parse(data.selector)
      } catch {
        return undefined
      }
    }
    return data.selector
  })()
  return request(`/api/v1/clusters/${clusterId}/services/edit`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      type: data.type,
      selector,
    },
  })
}

export function createConfigMap(clusterId: number, namespace: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          apiVersion: 'v1',
          kind: 'ConfigMap',
          metadata: {
            name: data.metadata?.name || 'new-configmap',
            namespace,
            uid: `uid-${Date.now()}`,
            creationTimestamp: new Date().toISOString(),
          },
          data: data.data || {},
        });
      }, 500);
    });
  }
  return applyYaml(
    clusterId,
    [
      'apiVersion: v1',
      'kind: ConfigMap',
      'metadata:',
      `  name: ${data.metadata?.name || data.name}`,
      `  namespace: ${namespace}`,
      'data:',
      ...Object.entries((data.data || {}) as Record<string, string>).map(([key, value]) => `  ${key}: ${JSON.stringify(value)}`),
    ].join('\n'),
  )
}

export function createSecret(clusterId: number, namespace: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          apiVersion: 'v1',
          kind: 'Secret',
          metadata: {
            name: data.metadata?.name || 'new-secret',
            namespace,
            uid: `uid-${Date.now()}`,
            creationTimestamp: new Date().toISOString(),
          },
          type: data.type || 'Opaque',
          data: data.data || {},
        });
      }, 500);
    });
  }
  return applyYaml(
    clusterId,
    [
      'apiVersion: v1',
      'kind: Secret',
      'metadata:',
      `  name: ${data.metadata?.name || data.name}`,
      `  namespace: ${namespace}`,
      `type: ${data.type || 'Opaque'}`,
      'stringData:',
      ...Object.entries((data.data || {}) as Record<string, string>).map(([key, value]) => `  ${key}: ${JSON.stringify(value)}`),
    ].join('\n'),
  )
}

// ==================== 删除操作补齐 ====================

export function deleteJob(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/jobs/${namespace}/${name}`, { method: 'DELETE' })
}

export function deleteCronJob(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/cronjobs/${namespace}/${name}`, { method: 'DELETE' })
}

export function deleteHPA(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/hpas/${namespace}/${name}`, { method: 'DELETE' })
}

export function deletePDB(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/pdbs/${namespace}/${name}`, { method: 'DELETE' })
}

export function deleteReplicaSet(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/replicasets/${namespace}/${name}`, { method: 'DELETE' })
}

export function deleteStorageClass(clusterId: number, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/storageclasses/${name}`, { method: 'DELETE' })
}

export function deleteServiceAccount(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/serviceaccounts/${namespace}/${name}`, { method: 'DELETE' })
}

export function deleteRole(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/roles/${namespace}/${name}`, { method: 'DELETE' })
}

export function deleteClusterRole(clusterId: number, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/clusterroles/${name}`, { method: 'DELETE' })
}

export function deleteRoleBinding(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/rolebindings/${namespace}/${name}`, { method: 'DELETE' })
}

export function deleteClusterRoleBinding(clusterId: number, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/clusterrolebindings/${name}`, { method: 'DELETE' })
}

export function deleteNetworkPolicy(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/networkpolicies/${namespace}/${name}`, { method: 'DELETE' })
}

export function deleteResourceQuota(clusterId: number, namespace: string, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/resourcequotas/${namespace}/${name}`, { method: 'DELETE' })
}

// ==================== Pod 日志和终端 ====================

export function getPodLogs(clusterId: number, namespace: string, name: string, tailLines?: number): Promise<{ logs: string }> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const lines = tailLines || 100
      const logs = Array.from({ length: lines }, (_, i) => `[2025-03-15T10:${String(i).padStart(2, '0')}:00Z] Log line ${i + 1} from pod ${name}`).join('\n')
      setTimeout(() => resolve({ logs }), 200)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/logs`, { params: { tail_lines: tailLines ?? 200 } }).then((res: { text?: string; logs?: string }) => ({
    logs: res.text || res.logs || '',
  }))
}

export function getPodTerminalUrl(clusterId: number, namespace: string, name: string, options?: { container?: string; command?: string[]; tty?: boolean }): Promise<{ url: string }> {
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/exec`, {
    method: 'POST',
    data: {
      container: options?.container || undefined,
      command: options?.command?.length ? options.command : ['/bin/sh'],
      tty: options?.tty ?? true,
    },
  }).then((res: { session_id?: string; ws_url?: string; url?: string }) => ({
    url: res.ws_url || res.url || '',
  }))
}

export function getDeploymentDetail(clusterId: number, namespace: string, name: string): Promise<Deployment> {
  if (MOCK_ENABLED) {
    const found = MOCK_DEPLOYMENTS.find((d) => d.name === name && d.namespace === namespace)
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        if (found) resolve(found)
        else reject(new Error('Deployment 不存在'))
      }, 100)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/deployments/${namespace}/${name}`, {})
}

export function updateDeployment(clusterId: number, namespace: string, name: string, data: Record<string, unknown>): Promise<Deployment> {
  const kind = (data.kind as 'Deployment' | 'StatefulSet' | 'DaemonSet' | undefined) || 'Deployment'
  const endpointMap = {
    Deployment: 'deployments',
    StatefulSet: 'statefulsets',
    DaemonSet: 'daemonsets',
  } as const
  return request(`/api/v1/clusters/${clusterId}/workloads/${endpointMap[kind]}/edit`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      replicas: data.replicas,
      containers: data.containers,
      strategy: data.strategy,
    },
  })
}

export function updateWorkloadPaused(
  clusterId: number,
  namespace: string,
  name: string,
  paused: boolean,
): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/workloads/rollout-pause`, {
    method: 'PATCH',
    data: { kind: 'Deployment', namespace, name, paused },
  })
}

export function updateWorkloadImage(
  clusterId: number,
  namespace: string,
  name: string,
  container: string,
  image: string,
  kind: 'Deployment' | 'StatefulSet' | 'DaemonSet' = 'Deployment',
): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/workloads/image`, {
    method: 'PATCH',
    data: { kind, namespace, name, container, image },
  })
}

// ==================== 更新操作补齐 ====================

export function updateConfigMap(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'v1', kind: 'ConfigMap',
        metadata: { name, namespace, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        data: data.data || {},
      }), 300)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/configmaps/edit`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      data: data.data,
      labels: data.labels,
    },
  })
}

export function updateSecret(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'v1', kind: 'Secret',
        metadata: { name, namespace, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        type: data.type || 'Opaque', data: data.data || {},
      }), 300)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/secrets/edit`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      type: data.type,
      stringData: data.stringData,
      labels: data.labels,
    },
  })
}

export function createIngress(clusterId: number, namespace: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'networking.k8s.io/v1', kind: 'Ingress',
        metadata: { name: data.metadata?.name || 'new-ingress', namespace, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        spec: data.spec || {},
      }), 500)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/ingresses`, {
    method: 'POST',
    data: {
      namespace,
      name: data.name,
      ingress_class: data.ingressClassName,
      rules: [
        {
          host: data.host,
          paths: [
            {
              path: data.path || '/',
              path_type: 'Prefix',
              service_name: data.serviceName,
              service_port: Number(data.servicePort || 80),
            },
          ],
        },
      ],
    },
  })
}

export function updateIngress(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'networking.k8s.io/v1', kind: 'Ingress',
        metadata: { name, namespace, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        spec: data.spec || data,
      }), 300)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/ingresses/edit`, {
    method: 'PATCH',
    data: {
      namespace,
      name,
      ingressClassName: data.ingressClassName,
      annotations: data.annotations,
      labels: data.labels,
    },
  })
}

export function createHPA(clusterId: number, namespace: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'autoscaling/v2', kind: 'HorizontalPodAutoscaler',
        metadata: { name: data.metadata?.name || 'new-hpa', namespace, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        spec: data.spec || {},
      }), 500)
    })
  }
  return applyYaml(
    clusterId,
    [
      'apiVersion: autoscaling/v2',
      'kind: HorizontalPodAutoscaler',
      'metadata:',
      `  name: ${data.name}`,
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
    ].join('\n'),
  )
}

export function updateHPA(clusterId: number, namespace: string, name: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'autoscaling/v2', kind: 'HorizontalPodAutoscaler',
        metadata: { name, namespace, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        spec: data.spec || data,
      }), 300)
    })
  }
  const yaml = [
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
  return request(`/api/v1/clusters/${clusterId}/hpas/edit`, { method: 'PATCH', data: { namespace, yaml } })
}

export function createPV(clusterId: number, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'v1', kind: 'PersistentVolume',
        metadata: { name: data.metadata?.name || 'new-pv', uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        spec: data.spec || {},
      }), 500)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pv`, { method: 'POST', data })
}

export function createPVC(clusterId: number, namespace: string, data: any): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'v1', kind: 'PersistentVolumeClaim',
        metadata: { name: data.metadata?.name || 'new-pvc', namespace, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        spec: data.spec || {},
      }), 500)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/pvc`, { method: 'POST', data, params: { namespace } })
}

export function createNamespace(clusterId: number, name: string): Promise<any> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve({
        apiVersion: 'v1', kind: 'Namespace',
        metadata: { name, uid: `uid-${Date.now()}`, creationTimestamp: new Date().toISOString() },
        status: 'Active',
      }), 500)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/namespaces`, { method: 'POST', data: { metadata: { name } } })
}

export function deleteNamespace(clusterId: number, name: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { setTimeout(() => resolve(), 200) })
  }
  return request(`/api/v1/clusters/${clusterId}/namespaces/${name}`, { method: 'DELETE' })
}

// ==================== 兼容性导出（拓扑图等页面使用的别名） ====================

/** 获取节点列表（带 items 包装，兼容拓扑图） */
export function getNodes(clusterId: number, signal?: AbortSignal): Promise<{ items: Node[] }> {
  return listNodes(clusterId, signal).then((nodes) => ({ items: nodes }))
}

/** 获取命名空间列表（带 items 包装，兼容拓扑图） */
export function getNamespaces(clusterId: number, signal?: AbortSignal): Promise<{ items: Namespace[] }> {
  return listNamespaces(clusterId, signal).then((namespaces) => ({ items: namespaces }))
}

/** 获取 Deployment 列表（兼容 workloads 页面调用签名） */
export function getDeployments(
  clusterId: number,
  namespace?: string,
  signal?: AbortSignal,
  kind: 'Deployment' | 'StatefulSet' | 'DaemonSet' = 'Deployment',
): Promise<DeploymentList> {
  return listDeployments(clusterId, { namespace: namespace || undefined, kind }, signal)
}

/** 获取 Pod 列表（兼容拓扑图页面调用签名） */
export function getPods(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<PodList> {
  return listPods(clusterId, { namespace: namespace || undefined }, signal)
}

/** 获取 Service 列表（兼容拓扑图页面调用签名） */
export function getK8sServices(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<K8sServiceList> {
  return listServices(clusterId, { namespace: namespace || '' }, signal)
}

/** 获取 ConfigMap 列表（兼容拓扑图页面调用签名） */
export function getConfigMaps(clusterId: number, namespace?: string, signal?: AbortSignal): Promise<ConfigMapList> {
  return listConfigMaps(clusterId, { namespace: namespace || '' }, signal)
}

// ==================== YAML / History ====================

/** 获取节点 YAML */
export function getNodeYaml(clusterId: number, name: string, signal?: AbortSignal): Promise<{ yaml: string }> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const node = MOCK_NODES.find((n) => n.name === name) || MOCK_NODES[0]!
      const yaml = `apiVersion: v1
kind: Node
metadata:
  name: ${node.name}
  labels:
    kubernetes.io/hostname: ${node.name}
    node-role.kubernetes.io/${node.roles[0] || 'worker'}: ""
status:
  capacity:
    cpu: "${node.cpu}"
    memory: "${node.memory}"
  nodeInfo:
    kubeletVersion: ${node.kubeletVersion}
    osImage: ${node.osImage}`
      setTimeout(() => resolve({ yaml }), 100)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/nodes/${name}/yaml`, { signal }).then((res: { text?: string; yaml?: string }) => ({
    yaml: res.yaml || res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** 获取 Deployment YAML */
export function getDeploymentYaml(
  clusterId: number,
  namespace: string,
  name: string,
  signal?: AbortSignal,
  kind: 'Deployment' | 'StatefulSet' | 'DaemonSet' = 'Deployment',
): Promise<{ yaml: string }> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const d = MOCK_DEPLOYMENTS.find((x) => x.name === name && x.namespace === namespace) || MOCK_DEPLOYMENTS[0]!
      const yaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${d.name}
  namespace: ${d.namespace}
spec:
  replicas: ${d.upToDate || 3}
  selector:
    matchLabels:
      app: ${d.name}
  template:
    metadata:
      labels:
        app: ${d.name}
    spec:
      containers:
      - name: ${d.name}
        image: ${d.images[0] || 'nginx:latest'}
        ports:
        - containerPort: 80`
      setTimeout(() => resolve({ yaml }), 100)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/workloads/${kind}/${namespace}/${name}/yaml`, { signal }).then((res: { text?: string; yaml?: string }) => ({
    yaml: res.yaml || res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** 获取 Deployment 版本历史 */
export function getDeploymentHistory(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<Array<{ revision: number; changeCause: string; date: string; image: string }>> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const d = MOCK_DEPLOYMENTS.find((x) => x.name === name && x.namespace === namespace)
      setTimeout(() => {
        resolve([
          { revision: 1, changeCause: 'Initial deployment', date: d?.createdAt || '2025-03-01T00:00:00Z', image: d?.images[0] || 'nginx:latest' },
          { revision: 2, changeCause: 'Updated image', date: '2025-03-10T10:00:00Z', image: d?.images[0] || 'nginx:latest' },
        ])
      }, 150)
    })
  }
  return request(`/api/v1/clusters/${clusterId}/workloads/deployments/${namespace}/${name}/rollout-history`, { signal }).then((res: any) => res.history || [])
}

export function getSecretReveal(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<{ text: string }> {
  return request(`/api/v1/clusters/${clusterId}/secrets/${namespace}/${name}/reveal`, { signal }).then((res: any) => ({
    text: res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

// ==================== 补齐：后端已有但前端缺失的操作 ====================

/** Pod 巡检诊断 */
export function getPodInspection(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<{ text: string }> {
  return request(`/api/v1/clusters/${clusterId}/pods/${namespace}/${name}/inspection`, { signal }).then((res: any) => ({
    text: res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** CronJob 手动触发 */
export function triggerCronJob(clusterId: number, namespace: string, name: string): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/cronjobs/${namespace}/${name}/trigger`, { method: 'POST' })
}

/** CronJob 暂停/恢复 */
export function suspendCronJob(clusterId: number, namespace: string, name: string, suspend: boolean): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/cronjobs/${namespace}/${name}/suspend`, { method: 'PATCH', data: { suspend } })
}

/** 获取 Node 上的 Pod 列表 */
export function getNodePods(clusterId: number, nodeName: string, signal?: AbortSignal): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${nodeName}/pods`, { signal }).then((res: any) => res.list || res || [])
}

/** 获取 Node 事件 */
export function getNodeEvents(clusterId: number, nodeName: string, signal?: AbortSignal): Promise<any[]> {
  return request(`/api/v1/clusters/${clusterId}/nodes/${nodeName}/events`, { signal }).then((res: any) => res.list || res || [])
}

/** Namespace 巡检诊断 */
export function getNamespaceInspection(clusterId: number, namespace: string, signal?: AbortSignal): Promise<{ text: string }> {
  return request(`/api/v1/clusters/${clusterId}/namespaces/${namespace}/inspection`, { signal }).then((res: any) => ({
    text: res.text || (typeof res === 'string' ? res : JSON.stringify(res, null, 2)),
  }))
}

/** Namespace 资源摘要 */
export function getNamespaceResourcesSummary(clusterId: number, namespace: string, signal?: AbortSignal): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/namespaces/${namespace}/resources-summary`, { signal })
}

/** 批量删除已完成的 Job */
export function deleteCompletedJobs(clusterId: number): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/jobs/completed`, { method: 'DELETE' })
}

/** 获取 ConfigMap 关联资源 */
export function getConfigMapRelated(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/configmaps/${namespace}/${name}/related`, { signal })
}

/** 获取 Secret 关联资源 */
export function getSecretRelated(clusterId: number, namespace: string, name: string, signal?: AbortSignal): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/secrets/${namespace}/${name}/related`, { signal })
}

/** Namespace 工作负载清单 */
export function getNamespaceWorkloadInventory(clusterId: number, namespace: string, signal?: AbortSignal): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/namespaces/${namespace}/workload-inventory`, { signal })
}