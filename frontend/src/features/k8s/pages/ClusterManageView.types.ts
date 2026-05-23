export type WorkloadKind = 'Deployment' | 'StatefulSet' | 'DaemonSet'

export type ResourceKey =
  | 'dashboard'
  | 'manifestapply'
  | 'permissionaudits'
  | 'topology'
  | 'nodes'
  | 'namespaces'
  | 'workloads'
  | 'pdbs'
  | 'hpas'
  | 'pods'
  | 'podmetrics'
  | 'replicasets'
  | 'jobs'
  | 'cronjobs'
  | 'services'
  | 'endpoints'
  | 'endpointslices'
  | 'networkpolicies'
  | 'ingresses'
  | 'ingressclasses'
  | 'configmaps'
  | 'secrets'
  | 'serviceaccounts'
  | 'roles'
  | 'clusterroles'
  | 'rolebindings'
  | 'clusterrolebindings'
  | 'pvs'
  | 'pvcs'
  | 'volumesnapshots'
  | 'volumesnapshotclasses'
  | 'volumesnapshotcontents'
  | 'leases'
  | 'storageclasses'
  | 'csidrivers'
  | 'csinodes'
  | 'csistoragecapacities'
  | 'volumeattachments'
  | 'resourcequotas'
  | 'limitranges'
  | 'customresourcedefinitions'
  | 'apiservices'
  | 'priorityclasses'
  | 'runtimeclasses'
  | 'validatingwebhookconfigurations'
  | 'mutatingwebhookconfigurations'
  | 'validatingadmissionpolicies'
  | 'validatingadmissionpolicybindings'
  | 'events'

export type TreeKind = 'folder' | 'view'

export type TreeNode = {
  id: string
  label: string
  kind: TreeKind
  iconUrl?: string
  resource?: ResourceKey
  perm?: string | string[]
  namespaced?: boolean
  workloadKind?: WorkloadKind
  children?: TreeNode[]
}

export type SortOrder = 'asc' | 'desc'

export type K8sMetadata = {
  name?: unknown
  namespace?: unknown
  uid?: unknown
  creationTimestamp?: unknown
  labels?: unknown
  annotations?: unknown
}

export type K8sLikeObject = {
  kind?: unknown
  metadata?: K8sMetadata
  spec?: Record<string, unknown> | null
  status?: Record<string, unknown> | null
  [key: string]: unknown
}
