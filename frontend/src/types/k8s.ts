/**
 * K8s 资源类型 — 扁平结构，前端直接使用
 */
import type { PageResult } from './common'

/** 命名空间 */
export interface Namespace {
  name: string
  status: string
  labels?: Record<string, string>
  createdAt: string
}

/** 节点 */
export interface Node {
  name: string
  status: string
  roles: string[]
  kubeletVersion: string
  osImage: string
  cpu: string
  memory: string
  cpuPercent?: number
  memoryPercent?: number
  podCount?: number
  age?: string
  createdAt: string
  ip?: string
  podCIDR?: string
  taints?: Array<{ key: string; effect: string; value?: string }>
}

/** Pod */
export interface Pod {
  name: string
  namespace: string
  status: string        // Running / Pending / Succeeded / Failed / Unknown
  containerReason?: string  // CrashLoopBackOff / ImagePullBackOff / ContainerCreating 等
  ready: string         // e.g. "1/1"
  restarts: number
  nodeName: string
  ip: string
  createdAt: string
  labels?: Record<string, string>
  ownerName?: string
  ownerKind?: string
  qosClass?: string
  annotations?: Record<string, string>
  containers?: PodContainer[]
  initContainers?: PodContainer[]
  conditions?: Array<{ type: string; status: string; reason?: string; message?: string; lastTransitionTime?: string }>
  volumes?: PodVolume[]
  probes?: PodProbes
  scheduling?: PodScheduling
  security?: PodSecurity
  envVars?: Array<{ name: string; value?: string; valueFrom?: string }>
  serviceAccountName?: string
  restartPolicy?: string
  dnsPolicy?: string
}

export interface PodContainer {
  name: string
  image: string
  ready?: boolean
  restartCount?: number
  state?: string
  stateReason?: string
  stateMessage?: string
  exitCode?: number
  startedAt?: string
  finishedAt?: string
  cpuRequest?: string
  cpuLimit?: string
  memoryRequest?: string
  memoryLimit?: string
  ports?: Array<{ containerPort: number; protocol?: string }>
  command?: string[]
  args?: string[]
  workingDir?: string
  imagePullPolicy?: string
}

export interface PodProbes {
  liveness?: { path?: string; port?: string | number; delay?: number; period?: number; timeout?: number; failure?: number }
  readiness?: { path?: string; port?: string | number; delay?: number; period?: number; timeout?: number; failure?: number }
  startup?: { path?: string; port?: string | number; delay?: number; period?: number; timeout?: number; failure?: number }
}

export interface PodScheduling {
  nodeSelector?: Record<string, string>
  nodeName?: string
  affinity?: string  // JSON string for display
  tolerations?: Array<{ key?: string; operator?: string; value?: string; effect?: string; tolerationSeconds?: number }>
  priorityClassName?: string
  topologySpreadConstraints?: string
}

export interface PodSecurity {
  serviceAccountName?: string
  runAsUser?: number
  runAsGroup?: number
  runAsNonRoot?: boolean
  privileged?: boolean
  readOnlyRootFilesystem?: boolean
  capabilities?: { add?: string[]; drop?: string[] }
  imagePullSecrets?: string[]
}

export interface PodVolume {
  name: string
  type: string          // configMap / secret / emptyDir / pvc / hostPath 等
  source?: string       // 来源名称
  mountPaths?: Array<{ name: string; path: string; readOnly?: boolean }>
}

export interface PodListParams {
  namespace?: string
  page?: number
  pageSize?: number
}

export type PodList = PageResult<Pod>

/** Deployment */
export interface Deployment {
  name: string
  namespace: string
  ready: string         // e.g. "3/3"
  upToDate: number
  available: number
  replicas?: number
  readyReplicas?: number
  containers?: Array<{ name: string; image: string }>
  strategy?: string
  paused?: boolean
  age: string
  images: string[]
  createdAt: string
}

export interface DeploymentListParams {
  namespace?: string
  page?: number
  pageSize?: number
}

export type DeploymentList = PageResult<Deployment>

/** Service */
export interface K8sService {
  name: string
  namespace: string
  type: string          // ClusterIP / NodePort / LoadBalancer
  clusterIP: string
  externalIP?: string
  ports: Array<{ port: number; nodePort?: number; protocol?: string }> | string
  age: string
  createdAt: string
  selector?: Record<string, string>
  endpointsCount?: number
  sessionAffinity?: string
}

/** Service 别名 */
export type Service = K8sService

export type K8sServiceList = PageResult<K8sService>

/** ConfigMap */
export interface ConfigMap {
  name: string
  namespace: string
  data: Record<string, string>
  age: string
  createdAt: string
}

export type ConfigMapList = PageResult<ConfigMap>

/** 事件 */
export interface K8sEvent {
  type: string          // Normal / Warning
  reason: string
  message: string
  namespace: string
  involvedObject: string
  firstTimestamp: string
  lastTimestamp: string
  count: number
}

/** 拓扑图节点 */
export interface TopologyNode {
  id: string
  name: string
  type: string
  namespace?: string
  status?: string
  metadata?: Record<string, unknown>
}

/** 拓扑图边 */
export interface TopologyEdge {
  source: string
  target: string
  type?: string
}

/** Secret */
export interface Secret {
  name: string
  namespace: string
  type: string           // Opaque / kubernetes.io/tls / kubernetes.io/dockerconfigjson 等
  dataKeys: string[]     // data 中的 key 列表
  data?: Record<string, string>
  createdAt: string
}

export type SecretList = PageResult<Secret>

/** Ingress */
export interface Ingress {
  name: string
  namespace: string
  ingressClass: string
  hosts: string[]
  addresses: string[]
  ports: string
  age: string
  createdAt: string
  tls?: boolean
  rules?: any[]
  tlsConfigs?: Array<{ hosts: string[]; secretName: string }>
}

export type IngressList = PageResult<Ingress>

/** PersistentVolume */
export interface PersistentVolume {
  name: string
  capacity: string       // 如 "10Gi"
  accessModes: string[]  // ReadWriteOnce / ReadOnlyMany / ReadWriteMany
  reclaimPolicy: string  // Retain / Delete / Recycle
  status: string         // Available / Bound / Released / Failed
  claim: string          // 如 "default/my-pvc"
  storageClass: string
  reason: string
  createdAt: string
}

export type PersistentVolumeList = PageResult<PersistentVolume>

/** PersistentVolumeClaim */
export interface PersistentVolumeClaim {
  name: string
  namespace: string
  status: string         // Bound / Pending / Lost
  volumeName: string
  capacity: string
  accessModes: string[]
  storageClass: string
  age: string
  createdAt: string
}

export type PersistentVolumeClaimList = PageResult<PersistentVolumeClaim>

/** 节点详情（扩展） */
export interface NodeDetail extends Node {
  version?: string
  containerRuntime?: string
  addresses: NodeAddress[]
  conditions: NodeCondition[]
  images: NodeImage[]
  allocatable: NodeResources
  capacity: NodeResources
}

export interface NodeAddress {
  type: string   // InternalIP / ExternalIP / Hostname
  address: string
}

export interface NodeCondition {
  type: string   // Ready / MemoryPressure / DiskPressure / PIDPressure
  status: string
  message: string
  lastTransitionTime: string
}

export interface NodeImage {
  names: string[]
  sizeBytes: number
}

export interface NodeResources {
  cpu: string
  memory: string
  pods: string
  'ephemeral-storage'?: string
}

/** Job */
export interface Job {
  name: string
  namespace: string
  completions: string   // 如 "1/1"
  parallelism: number
  status: string        // Active / Succeeded / Failed
  age: string
  createdAt: string
  images?: string[]
  labels?: Record<string, string>
  ownerKind?: string
  ownerName?: string
}

export type JobList = PageResult<Job>

/** CronJob */
export interface CronJob {
  name: string
  namespace: string
  schedule: string
  suspend: boolean
  active: number
  lastScheduleTime: string
  age: string
  createdAt: string
  images?: string[]
  labels?: Record<string, string>
  concurrencyPolicy?: string
  successfulJobsHistoryLimit?: number
  failedJobsHistoryLimit?: number
}

export type CronJobList = PageResult<CronJob>

/** HPA */
export interface HPA {
  name: string
  namespace: string
  reference: string     // 如 "Deployment/nginx"
  targets: string       // 如 "50%/80%"
  minPods: number
  maxPods: number
  replicas: number
  targetCPU?: number
  age: string
  createdAt: string
  targetName?: string
  minReplicas?: number
  maxReplicas?: number
  currentReplicas?: number
  currentCPU?: number
  targetKind?: string
  conditions?: Array<{ type: string; status: string; reason: string; message: string }>
}

export type HPAList = PageResult<HPA>

/** PDB */
export interface PDB {
  name: string
  namespace: string
  minAvailable: string
  maxUnavailable: string
  allowedDisruptions: number
  currentHealthy: number
  desiredHealthy: number
  age: string
  createdAt: string
  selector?: string
}

export type PDBList = PageResult<PDB>

/** ReplicaSet */
export interface ReplicaSet {
  name: string
  namespace: string
  desired: number
  current: number
  ready: number
  age: string
  images: string[]
  createdAt: string
  ownerKind?: string
  ownerName?: string
}

export type ReplicaSetList = PageResult<ReplicaSet>

/** StorageClass */
export interface StorageClass {
  name: string
  provisioner: string
  reclaimPolicy: string
  volumeBindingMode: string
  allowVolumeExpansion: boolean
  isDefault?: boolean
  parameters: Record<string, string>
  age: string
  createdAt: string
}

export type StorageClassList = PageResult<StorageClass>

/** ServiceAccount */
export interface ServiceAccount {
  name: string
  namespace: string
  secrets: number
  age: string
  createdAt: string
}

export type ServiceAccountList = PageResult<ServiceAccount>

/** RBAC Role */
export interface RBACRole {
  name: string
  namespace?: string
  rules: RBACRule[]
  age: string
  createdAt: string
}

export interface RBACRule {
  apiGroups: string[]
  resources: string[]
  verbs: string[]
}

/** RBAC RoleBinding */
export interface RBACRoleBinding {
  name: string
  namespace?: string
  roleRef: { kind?: string; name: string } | string
  subjects: Array<{ kind?: string; name: string }> | string[]
  age?: string
  createdAt: string
}

export type RBACRoleList = PageResult<RBACRole>
export type RBACRoleBindingList = PageResult<RBACRoleBinding>

/** NetworkPolicy */
export interface NetworkPolicy {
  name: string
  namespace: string
  podSelector: string
  policyTypes: string[]
  age: string
  createdAt: string
  ingress?: any[]
  egress?: any[]
}

export type NetworkPolicyList = PageResult<NetworkPolicy>

/** ResourceQuota */
export interface ResourceQuota {
  name: string
  namespace: string
  hard: Record<string, string>
  used: Record<string, string>
  age: string
  createdAt: string
}

export type ResourceQuotaList = PageResult<ResourceQuota>

/** K8s 资源树节点（用于左侧导航树） */
export interface K8sTreeNode {
  id: string
  label: string
  kind: 'folder' | 'leaf'
  resource?: string
  iconUrl?: string
  children?: K8sTreeNode[]
}
