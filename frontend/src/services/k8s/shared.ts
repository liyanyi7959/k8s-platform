/**
 * K8s service 共享工具与映射层
 * 后端返回标准 K8s API 格式 { metadata, spec, status }，前端期望扁平结构
 */
import type {
  Pod,
  PodVolume,
  Deployment,
  K8sService,
  ConfigMap,
  K8sEvent,
  Node,
  Secret,
  Ingress,
  PersistentVolume,
  PersistentVolumeClaim,
  Job,
  CronJob,
  HPA,
  PDB,
  ReplicaSet,
  StorageClass,
  ServiceAccount,
  RBACRole,
  RBACRoleBinding,
  NetworkPolicy,
  ResourceQuota,
} from '@/types'

/** 从后端响应中提取列表数据（后端返回 { list: [...] } 格式） */
export function extractList<T>(res: { list?: T[] } | T[] | undefined | null): T[] {
  if (Array.isArray(res)) return res
  if (res && 'list' in res && Array.isArray(res.list)) return res.list
  return []
}

/** 从后端 { list: [...] } 格式提取分页数据，转换为 PageResult<T> 格式 */
export function extractPageList<T>(res: { list?: T[] } | { items?: T[] } | T[] | undefined | null): { items: T[]; total: number; page: number; pageSize: number } {
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

/** 安全取值 */
export function getNested(obj: any, ...keys: string[]): any {
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
export function mapPod(raw: any): Pod {
  const m = raw?.metadata || {}
  const s = raw?.status || {}
  const spec = raw?.spec || {}
  const ownerRef = Array.isArray(m.ownerReferences) ? m.ownerReferences[0] : undefined

  // 提取容器级等待状态（CrashLoopBackOff / ImagePullBackOff / ContainerCreating 等）
  let containerReason = ''
  if (Array.isArray(s.containerStatuses)) {
    for (const cs of s.containerStatuses) {
      if (cs.state?.waiting?.reason) {
        containerReason = cs.state.waiting.reason
        break
      }
      if (cs.state?.terminated?.reason) {
        containerReason = cs.state.terminated.reason
        break
      }
    }
  }

  // 映射容器（含 initContainers）
  const mapContainer = (c: any): any => {
    const cs = Array.isArray(s.containerStatuses) ? s.containerStatuses.find((cs: any) => cs.name === c.name) :
      Array.isArray(s.initContainerStatuses) ? s.initContainerStatuses.find((cs: any) => cs.name === c.name) : undefined
    const res = c.resources || {}
    const req = res.requests || {}
    const lim = res.limits || {}
    const stateKey = cs?.state ? Object.keys(cs.state)[0] : undefined
    const stateObj = stateKey ? cs.state[stateKey] : undefined
    return {
      name: c.name || '',
      image: c.image || '',
      ready: cs?.ready,
      restartCount: cs?.restartCount,
      state: stateKey,
      stateReason: stateObj?.reason,
      stateMessage: stateObj?.message,
      exitCode: stateObj?.exitCode,
      startedAt: stateObj?.startedAt,
      finishedAt: stateObj?.finishedAt,
      cpuRequest: req.cpu,
      cpuLimit: lim.cpu,
      memoryRequest: req.memory,
      memoryLimit: lim.memory,
      ports: Array.isArray(c.ports) ? c.ports.map((p: any) => ({ containerPort: p.containerPort, protocol: p.protocol })) : undefined,
      command: c.command,
      args: c.args,
      workingDir: c.workingDir,
      imagePullPolicy: c.imagePullPolicy,
    }
  }

  // 提取探针（取第一个容器的探针）
  const firstContainer = Array.isArray(spec.containers) ? spec.containers[0] : undefined
  const probes = firstContainer ? extractProbes(firstContainer) : undefined

  // 提取调度信息
  const scheduling = extractScheduling(spec)

  // 提取安全上下文
  const security = extractSecurity(spec, firstContainer)

  // 提取环境变量
  const envVars = Array.isArray(firstContainer?.env) ? firstContainer.env.map((e: any) => ({
    name: e.name || '',
    value: e.value,
    valueFrom: e.valueFrom ? Object.keys(e.valueFrom)[0] : undefined,
  })) : undefined

  return {
    name: m.name || '',
    namespace: m.namespace || '',
    status: s.phase || 'Unknown',
    containerReason,
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
    containers: Array.isArray(spec.containers) ? spec.containers.map(mapContainer) : [],
    initContainers: Array.isArray(spec.initContainers) ? spec.initContainers.map(mapContainer) : [],
    conditions: Array.isArray(s.conditions) ? s.conditions.map((c: any) => ({
      type: c.type || '',
      status: c.status || '',
      reason: c.reason,
      message: c.message,
      lastTransitionTime: c.lastTransitionTime || '',
    })) : [],
    volumes: extractVolumes(spec),
    probes,
    scheduling,
    security,
    envVars,
    serviceAccountName: spec.serviceAccountName,
    restartPolicy: spec.restartPolicy,
    dnsPolicy: spec.dnsPolicy,
  }
}

/** 提取探针配置 */
function extractProbes(c: any): any {
  const map = (p: any) => {
    if (!p) return undefined
    const probeKeys = Object.keys(p)
    const probe = probeKeys.length > 0 ? p[probeKeys[0]!] : {}
    return {
      path: probe.httpGet?.path || probe.tcpSocket ? undefined : probe.httpGet?.path,
      port: probe.httpGet?.port ?? probe.tcpSocket?.port,
      delay: p.initialDelaySeconds,
      period: p.periodSeconds,
      timeout: p.timeoutSeconds,
      failure: p.failureThreshold,
    }
  }
  return {
    liveness: map(c.livenessProbe),
    readiness: map(c.readinessProbe),
    startup: map(c.startupProbe),
  }
}

/** 提取调度信息 */
function extractScheduling(spec: any): any {
  return {
    nodeSelector: spec.nodeSelector,
    nodeName: spec.nodeName,
    affinity: spec.affinity ? JSON.stringify(spec.affinity, null, 2) : undefined,
    tolerations: Array.isArray(spec.tolerations) ? spec.tolerations.map((t: any) => ({
      key: t.key, operator: t.operator, value: t.value, effect: t.effect, tolerationSeconds: t.tolerationSeconds,
    })) : undefined,
    priorityClassName: spec.priorityClassName,
    topologySpreadConstraints: spec.topologySpreadConstraints ? JSON.stringify(spec.topologySpreadConstraints, null, 2) : undefined,
  }
}

/** 提取安全上下文 */
function extractSecurity(spec: any, firstContainer?: any): any {
  const sc = firstContainer?.securityContext || {}
  const podSc = spec.securityContext || {}
  return {
    serviceAccountName: spec.serviceAccountName,
    runAsUser: sc.runAsUser ?? podSc.runAsUser,
    runAsGroup: sc.runAsGroup ?? podSc.runAsGroup,
    runAsNonRoot: sc.runAsNonRoot ?? podSc.runAsNonRoot,
    privileged: sc.privileged,
    readOnlyRootFilesystem: sc.readOnlyRootFilesystem,
    capabilities: sc.capabilities ? { add: sc.capabilities.add, drop: sc.capabilities.drop } : undefined,
    imagePullSecrets: Array.isArray(spec.imagePullSecrets) ? spec.imagePullSecrets.map((ips: any) => ips.name) : undefined,
  }
}

/** 从 spec.volumes + spec.containers.volumeMounts 提取卷挂载信息 */
function extractVolumes(spec: any): PodVolume[] {
  if (!Array.isArray(spec.volumes)) return []
  const mounts: Record<string, Array<{ name: string; path: string; readOnly?: boolean }>> = {}
  if (Array.isArray(spec.containers)) {
    for (const c of spec.containers) {
      if (Array.isArray(c.volumeMounts)) {
        for (const vm of c.volumeMounts) {
          const name = vm.name || ''
          if (!mounts[name]) mounts[name] = []
          mounts[name].push({ name: c.name, path: vm.mountPath || '', readOnly: vm.readOnly })
        }
      }
    }
  }
  return spec.volumes.map((v: any) => {
    let type = 'unknown'
    let source = ''
    if (v.configMap) { type = 'ConfigMap'; source = v.configMap.name || '' }
    else if (v.secret) { type = 'Secret'; source = v.secretName || '' }
    else if (v.emptyDir) { type = 'emptyDir' }
    else if (v.persistentVolumeClaim) { type = 'PVC'; source = v.persistentVolumeClaim.claimName || '' }
    else if (v.hostPath) { type = 'hostPath'; source = v.hostPath.path || '' }
    else if (v.projected) { type = 'projected' }
    else if (v.downwardAPI) { type = 'downwardAPI' }
    else if (v.serviceAccountToken) { type = 'serviceAccountToken' }
    else if (v.ephemeral) { type = 'ephemeral' }
    return {
      name: v.name || '',
      type,
      source,
      mountPaths: mounts[v.name || ''],
    }
  })
}

/** 映射原始 K8s Deployment 对象 → 前端 Deployment 类型 */
export function mapDeployment(raw: any): Deployment {
  const m = raw?.metadata || {}
  const s = raw?.status || {}
  const spec = raw?.spec || {}
  const tplSpec = getNested(spec, 'template', 'spec') || {}
  const images = Array.isArray(tplSpec.containers) ? tplSpec.containers.map((c: any) => c.image || '') : []
  const containers = Array.isArray(tplSpec.containers)
    ? tplSpec.containers.map((c: any) => ({ name: c.name || '', image: c.image || '' }))
    : []
  // 兼容 Deployment（readyReplicas/updatedReplicas）和 DaemonSet（numberReady/updatedNumberScheduled）
  const ready = s.readyReplicas ?? s.numberReady ?? 0
  const total = spec.replicas ?? s.desiredNumberScheduled ?? 0
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    ready: `${ready}/${total}`,
    upToDate: s.updatedReplicas ?? s.updatedNumberScheduled ?? 0,
    available: s.availableReplicas ?? s.numberAvailable ?? 0,
    replicas: total,
    readyReplicas: ready,
    containers,
    strategy: spec.strategy?.type || spec.updateStrategy?.type || '',
    paused: spec.paused === true,
    age: m.creationTimestamp || '',
    images,
    createdAt: m.creationTimestamp || '',
  }
}

/** 映射原始 K8s Service 对象 → 前端 K8sService 类型 */
export function mapService(raw: any): K8sService {
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
  const externalIPs = s.externalIPs || st.loadBalancer?.ingress || []
  const externalIP = Array.isArray(externalIPs) && externalIPs.length > 0
    ? (typeof externalIPs[0] === 'string'
        ? externalIPs[0]
        : (externalIPs[0].ip || externalIPs[0].hostname || ''))
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
export function mapConfigMap(raw: any): ConfigMap {
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
export function mapSecret(raw: any): Secret {
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
export function mapNode(raw: any): Node {
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
  const addresses = Array.isArray(s.addresses) ? s.addresses : []
  const internalIP = addresses.find((a: any) => a.type === 'InternalIP')
  const ip = internalIP?.address || ''
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
export function mapEvent(raw: any): K8sEvent {
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
export function mapIngress(raw: any): Ingress {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  const rules = Array.isArray(s.rules) ? s.rules : []
  const hosts = rules.map((r: any) => r.host || '').filter(Boolean)
  const addresses = Array.isArray(st.loadBalancer?.ingress)
    ? st.loadBalancer.ingress.map((i: any) => i.ip || i.hostname || '').filter(Boolean)
    : []
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
export function mapPV(raw: any): PersistentVolume {
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
export function mapPVC(raw: any): PersistentVolumeClaim {
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
export function mapJob(raw: any): Job {
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
  const tplSpec = getNested(spec, 'template', 'spec') || {}
  const images = Array.isArray(tplSpec.containers) ? tplSpec.containers.map((c: any) => c.image || '') : []
  const ownerRef = Array.isArray(m.ownerReferences) ? m.ownerReferences[0] : undefined
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    completions: `${s.succeeded || 0}/${spec.completions || 1}`,
    parallelism: spec.parallelism || 1,
    status,
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
    images,
    labels: m.labels || undefined,
    ownerKind: ownerRef?.kind || '',
    ownerName: ownerRef?.name || '',
  }
}

/** 映射原始 K8s CronJob 对象 → 前端 CronJob 类型 */
export function mapCronJob(raw: any): CronJob {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  const tplSpec = getNested(s, 'jobTemplate', 'spec', 'template', 'spec') || {}
  const images = Array.isArray(tplSpec.containers) ? tplSpec.containers.map((c: any) => c.image || '') : []
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    schedule: s.schedule || '',
    suspend: s.suspend || false,
    active: Array.isArray(st.active) ? st.active.length : 0,
    lastScheduleTime: st.lastScheduleTime || '',
    age: m.creationTimestamp || '',
    createdAt: m.creationTimestamp || '',
    images,
    labels: m.labels || undefined,
    concurrencyPolicy: s.concurrencyPolicy || 'Allow',
    successfulJobsHistoryLimit: s.successfulJobsHistoryLimit,
    failedJobsHistoryLimit: s.failedJobsHistoryLimit,
  }
}

/** 映射原始 K8s HPA 对象 → 前端 HPA 类型 */
export function mapHPA(raw: any): HPA {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  const ref = s.scaleTargetRef
  const reference = ref ? `${ref.kind}/${ref.name}` : ''
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
export function mapReplicaSet(raw: any): ReplicaSet {
  const m = raw?.metadata || {}
  const s = raw?.status || {}
  const spec = raw?.spec || {}
  const tplSpec = getNested(spec, 'template', 'spec') || {}
  const images = Array.isArray(tplSpec.containers) ? tplSpec.containers.map((c: any) => c.image || '') : []
  const ownerRef = Array.isArray(m.ownerReferences) ? m.ownerReferences[0] : undefined
  return {
    name: m.name || '',
    namespace: m.namespace || '',
    desired: spec.replicas || 0,
    current: s.replicas || 0,
    ready: s.readyReplicas || 0,
    age: m.creationTimestamp || '',
    images,
    createdAt: m.creationTimestamp || '',
    ownerKind: ownerRef?.kind || '',
    ownerName: ownerRef?.name || '',
  }
}

/** 映射原始 K8s StorageClass 对象 → 前端 StorageClass 类型 */
export function mapStorageClass(raw: any): StorageClass {
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
export function mapRBACRole(raw: any): RBACRole {
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
export function mapRBACRoleBinding(raw: any): RBACRoleBinding {
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
export function mapNetworkPolicy(raw: any): NetworkPolicy {
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
    ingress: s.ingress,
    egress: s.egress,
  }
}

/** 映射原始 K8s ResourceQuota 对象 → 前端 ResourceQuota 类型 */
export function mapResourceQuota(raw: any): ResourceQuota {
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
export function mapServiceAccount(raw: any): ServiceAccount {
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
export function mapPDB(raw: any): PDB {
  const m = raw?.metadata || {}
  const s = raw?.spec || {}
  const st = raw?.status || {}
  const matchLabels = s.selector?.matchLabels || {}
  const selectorStr = Object.entries(matchLabels).map(([k, v]) => `${k}=${v}`).join(', ')
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
    selector: selectorStr || undefined,
  }
}

/** 带映射的列表提取 */
export function extractMappedList<T>(mapper: (raw: any) => T) {
  return (res: any): { items: T[]; total: number; page: number; pageSize: number } => {
    const raw = extractPageList(res)
    return { ...raw, items: raw.items.map(mapper) }
  }
}

/** 带映射的简单列表提取 */
export function extractMappedSimpleList<T>(mapper: (raw: any) => T) {
  return (res: any): T[] => {
    const raw = extractList(res)
    return raw.map(mapper)
  }
}
