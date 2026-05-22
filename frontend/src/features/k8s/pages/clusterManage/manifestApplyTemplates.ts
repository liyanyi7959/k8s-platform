import type { ResourceKey, WorkloadKind } from '../ClusterManageView.types'

export type ManifestTemplateResource = ResourceKey | 'manifestapply'

export type ManifestTemplatePreset = {
  defaultNamespace?: string
  initialYaml: string
  sourceLabel: string
}

type TemplateContext = {
  resource?: ManifestTemplateResource
  workloadKind?: WorkloadKind
  namespace?: string
}

type SimpleTemplateSpec = {
  apiVersion: string
  kind: string
  namespaced?: boolean
  name: string
  body?: string
}

const SIMPLE_TEMPLATE_MAP: Partial<Record<ResourceKey, SimpleTemplateSpec>> = {
  namespaces: { apiVersion: 'v1', kind: 'Namespace', name: 'sample-namespace', namespaced: false, body: 'metadata:\n  name: sample-namespace\n  labels:\n    env: dev\n' },
  replicasets: { apiVersion: 'apps/v1', kind: 'ReplicaSet', name: 'sample-rs', namespaced: true, body: 'spec:\n  replicas: 2\n  selector:\n    matchLabels:\n      app: sample-rs\n  template:\n    metadata:\n      labels:\n        app: sample-rs\n    spec:\n      containers:\n        - name: app\n          image: nginx:1.27\n          ports:\n            - containerPort: 80\n' },
  jobs: { apiVersion: 'batch/v1', kind: 'Job', name: 'sample-job', namespaced: true, body: 'spec:\n  template:\n    spec:\n      restartPolicy: Never\n      containers:\n        - name: job\n          image: busybox:1.36\n          command: ["sh", "-c", "echo hello from job"]\n' },
  cronjobs: { apiVersion: 'batch/v1', kind: 'CronJob', name: 'sample-cronjob', namespaced: true, body: 'spec:\n  schedule: "*/5 * * * *"\n  jobTemplate:\n    spec:\n      template:\n        spec:\n          restartPolicy: OnFailure\n          containers:\n            - name: cron\n              image: busybox:1.36\n              command: ["sh", "-c", "date; echo cronjob running"]\n' },
  endpoints: { apiVersion: 'v1', kind: 'Endpoints', name: 'sample-endpoints', namespaced: true, body: 'subsets:\n  - addresses:\n      - ip: 10.96.0.10\n    ports:\n      - name: http\n        port: 80\n        protocol: TCP\n' },
  endpointslices: { apiVersion: 'discovery.k8s.io/v1', kind: 'EndpointSlice', name: 'sample-endpointslice', namespaced: true, body: 'addressType: IPv4\nports:\n  - name: http\n    protocol: TCP\n    port: 80\nendpoints:\n  - addresses:\n      - 10.96.0.10\n' },
  networkpolicies: { apiVersion: 'networking.k8s.io/v1', kind: 'NetworkPolicy', name: 'sample-networkpolicy', namespaced: true, body: 'spec:\n  podSelector:\n    matchLabels:\n      app: sample-app\n  policyTypes:\n    - Ingress\n  ingress:\n    - from:\n        - namespaceSelector: {}\n' },
  ingressclasses: { apiVersion: 'networking.k8s.io/v1', kind: 'IngressClass', name: 'nginx', namespaced: false, body: 'spec:\n  controller: k8s.io/ingress-nginx\n' },
  serviceaccounts: { apiVersion: 'v1', kind: 'ServiceAccount', name: 'sample-sa', namespaced: true },
  roles: { apiVersion: 'rbac.authorization.k8s.io/v1', kind: 'Role', name: 'sample-role', namespaced: true, body: 'rules:\n  - apiGroups: [""]\n    resources: ["pods"]\n    verbs: ["get", "list", "watch"]\n' },
  clusterroles: { apiVersion: 'rbac.authorization.k8s.io/v1', kind: 'ClusterRole', name: 'sample-cluster-role', namespaced: false, body: 'rules:\n  - apiGroups: [""]\n    resources: ["nodes"]\n    verbs: ["get", "list", "watch"]\n' },
  rolebindings: { apiVersion: 'rbac.authorization.k8s.io/v1', kind: 'RoleBinding', name: 'sample-rolebinding', namespaced: true, body: 'subjects:\n  - kind: ServiceAccount\n    name: sample-sa\n    namespace: REPLACE_NAMESPACE\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: Role\n  name: sample-role\n' },
  clusterrolebindings: { apiVersion: 'rbac.authorization.k8s.io/v1', kind: 'ClusterRoleBinding', name: 'sample-cluster-rolebinding', namespaced: false, body: 'subjects:\n  - kind: ServiceAccount\n    name: sample-sa\n    namespace: REPLACE_NAMESPACE\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: ClusterRole\n  name: sample-cluster-role\n' },
  pvs: { apiVersion: 'v1', kind: 'PersistentVolume', name: 'sample-pv', namespaced: false, body: 'spec:\n  capacity:\n    storage: 10Gi\n  accessModes:\n    - ReadWriteOnce\n  persistentVolumeReclaimPolicy: Retain\n  storageClassName: standard\n  hostPath:\n    path: /data/sample-pv\n' },
  volumesnapshots: { apiVersion: 'snapshot.storage.k8s.io/v1', kind: 'VolumeSnapshot', name: 'sample-snapshot', namespaced: true, body: 'spec:\n  volumeSnapshotClassName: csi-snapclass\n  source:\n    persistentVolumeClaimName: sample-pvc\n' },
  volumesnapshotclasses: { apiVersion: 'snapshot.storage.k8s.io/v1', kind: 'VolumeSnapshotClass', name: 'csi-snapclass', namespaced: false, body: 'driver: disk.csi.example.com\ndeletionPolicy: Delete\n' },
  volumesnapshotcontents: { apiVersion: 'snapshot.storage.k8s.io/v1', kind: 'VolumeSnapshotContent', name: 'snapcontent-sample', namespaced: false, body: 'spec:\n  deletionPolicy: Delete\n  driver: disk.csi.example.com\n  source:\n    snapshotHandle: snapshot-handle-placeholder\n  volumeSnapshotClassName: csi-snapclass\n  volumeSnapshotRef:\n    name: sample-snapshot\n    namespace: REPLACE_NAMESPACE\n' },
  leases: { apiVersion: 'coordination.k8s.io/v1', kind: 'Lease', name: 'sample-lease', namespaced: true, body: 'spec:\n  holderIdentity: sample-client\n  leaseDurationSeconds: 30\n' },
  storageclasses: { apiVersion: 'storage.k8s.io/v1', kind: 'StorageClass', name: 'sample-storageclass', namespaced: false, body: 'provisioner: kubernetes.io/no-provisioner\nvolumeBindingMode: WaitForFirstConsumer\n' },
  csidrivers: { apiVersion: 'storage.k8s.io/v1', kind: 'CSIDriver', name: 'disk.csi.example.com', namespaced: false, body: 'spec:\n  attachRequired: true\n  podInfoOnMount: false\n' },
  csinodes: { apiVersion: 'storage.k8s.io/v1', kind: 'CSINode', name: 'node-1', namespaced: false, body: 'spec:\n  drivers:\n    - name: disk.csi.example.com\n      nodeID: node-1\n      topologyKeys:\n        - topology.kubernetes.io/zone\n' },
  csistoragecapacities: { apiVersion: 'storage.k8s.io/v1', kind: 'CSIStorageCapacity', name: 'sample-capacity', namespaced: true, body: 'storageClassName: sample-storageclass\ncapacity: 100Gi\nnodeTopology:\n  matchLabels:\n    topology.kubernetes.io/zone: zone-a\n' },
  volumeattachments: { apiVersion: 'storage.k8s.io/v1', kind: 'VolumeAttachment', name: 'sample-volumeattachment', namespaced: false, body: 'spec:\n  attacher: disk.csi.example.com\n  nodeName: node-1\n  source:\n    persistentVolumeName: sample-pv\n' },
  resourcequotas: { apiVersion: 'v1', kind: 'ResourceQuota', name: 'sample-quota', namespaced: true, body: 'spec:\n  hard:\n    requests.cpu: "4"\n    requests.memory: 8Gi\n    limits.cpu: "8"\n    limits.memory: 16Gi\n' },
  limitranges: { apiVersion: 'v1', kind: 'LimitRange', name: 'sample-limitrange', namespaced: true, body: 'spec:\n  limits:\n    - type: Container\n      default:\n        cpu: 500m\n        memory: 512Mi\n      defaultRequest:\n        cpu: 100m\n        memory: 128Mi\n' },
  customresourcedefinitions: { apiVersion: 'apiextensions.k8s.io/v1', kind: 'CustomResourceDefinition', name: 'widgets.example.com', namespaced: false, body: 'spec:\n  group: example.com\n  scope: Namespaced\n  names:\n    plural: widgets\n    singular: widget\n    kind: Widget\n  versions:\n    - name: v1alpha1\n      served: true\n      storage: true\n      schema:\n        openAPIV3Schema:\n          type: object\n          properties:\n            spec:\n              type: object\n              x-kubernetes-preserve-unknown-fields: true\n' },
  apiservices: { apiVersion: 'apiregistration.k8s.io/v1', kind: 'APIService', name: 'v1alpha1.example.com', namespaced: false, body: 'spec:\n  group: example.com\n  version: v1alpha1\n  groupPriorityMinimum: 2000\n  versionPriority: 15\n  service:\n    namespace: default\n    name: extension-apiserver\n' },
  priorityclasses: { apiVersion: 'scheduling.k8s.io/v1', kind: 'PriorityClass', name: 'sample-priority', namespaced: false, body: 'value: 100000\nglobalDefault: false\ndescription: Sample priority class\n' },
  runtimeclasses: { apiVersion: 'node.k8s.io/v1', kind: 'RuntimeClass', name: 'gvisor', namespaced: false, body: 'handler: runsc\n' },
  validatingwebhookconfigurations: { apiVersion: 'admissionregistration.k8s.io/v1', kind: 'ValidatingWebhookConfiguration', name: 'sample-validating-webhook', namespaced: false, body: 'webhooks:\n  - name: validate.example.com\n    sideEffects: None\n    admissionReviewVersions: ["v1"]\n    clientConfig:\n      service:\n        namespace: default\n        name: webhook-service\n        path: /validate\n    rules:\n      - apiGroups: [""]\n        apiVersions: ["v1"]\n        operations: ["CREATE", "UPDATE"]\n        resources: ["pods"]\n' },
  mutatingwebhookconfigurations: { apiVersion: 'admissionregistration.k8s.io/v1', kind: 'MutatingWebhookConfiguration', name: 'sample-mutating-webhook', namespaced: false, body: 'webhooks:\n  - name: mutate.example.com\n    sideEffects: None\n    admissionReviewVersions: ["v1"]\n    clientConfig:\n      service:\n        namespace: default\n        name: webhook-service\n        path: /mutate\n    rules:\n      - apiGroups: [""]\n        apiVersions: ["v1"]\n        operations: ["CREATE"]\n        resources: ["pods"]\n' },
  validatingadmissionpolicies: { apiVersion: 'admissionregistration.k8s.io/v1', kind: 'ValidatingAdmissionPolicy', name: 'sample-policy', namespaced: false, body: 'spec:\n  failurePolicy: Fail\n  matchConstraints:\n    resourceRules:\n      - apiGroups: [""]\n        apiVersions: ["v1"]\n        operations: ["CREATE", "UPDATE"]\n        resources: ["pods"]\n  validations:\n    - expression: "object.metadata.name.size() > 0"\n      message: metadata.name is required\n' },
  validatingadmissionpolicybindings: { apiVersion: 'admissionregistration.k8s.io/v1', kind: 'ValidatingAdmissionPolicyBinding', name: 'sample-policy-binding', namespaced: false, body: 'spec:\n  policyName: sample-policy\n  validationActions:\n    - Deny\n' },
  events: { apiVersion: 'events.k8s.io/v1', kind: 'Event', name: 'sample-event', namespaced: true, body: 'regarding:\n  apiVersion: v1\n  kind: Pod\n  name: sample-pod\nreason: SampleReason\nnote: Sample event created from manifest workbench\ntype: Normal\naction: Observe\nreportingController: manifest.workbench\nreportingInstance: manifest.workbench-1\n' }
}

function normalizeNamespace(namespace?: string): string {
  return String(namespace ?? '').trim() || 'default'
}

function metadataBlock(name: string, namespace: string, namespaced = true): string {
  if (!namespaced) return `metadata:\n  name: ${name}\n`
  return `metadata:\n  name: ${name}\n  namespace: ${namespace}\n`
}

function simpleTemplate(spec: SimpleTemplateSpec, namespace: string): string {
  const body = (spec.body ?? 'spec: {}\n').replace(/REPLACE_NAMESPACE/g, namespace)
  const metadata = body.startsWith('metadata:') ? '' : metadataBlock(spec.name, namespace, spec.namespaced !== false)
  return `apiVersion: ${spec.apiVersion}\nkind: ${spec.kind}\n${metadata}${body}`
}

function genericTemplate(namespace: string): string {
  return `apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: sample-config\n  namespace: ${namespace}\ndata:\n  app.yaml: |\n    key: value\n---\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: sample-app\n  namespace: ${namespace}\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: sample-app\n  template:\n    metadata:\n      labels:\n        app: sample-app\n    spec:\n      containers:\n        - name: app\n          image: nginx:1.27\n          ports:\n            - containerPort: 80\n`
}

function podTemplate(namespace: string): string {
  return `apiVersion: v1\nkind: Pod\nmetadata:\n  name: sample-pod\n  namespace: ${namespace}\n  labels:\n    app: sample-pod\nspec:\n  containers:\n    - name: app\n      image: nginx:1.27\n      ports:\n        - containerPort: 80\n`
}

function deploymentTemplate(namespace: string): string {
  return `apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: sample-deployment\n  namespace: ${namespace}\nspec:\n  replicas: 2\n  selector:\n    matchLabels:\n      app: sample-deployment\n  template:\n    metadata:\n      labels:\n        app: sample-deployment\n    spec:\n      containers:\n        - name: app\n          image: nginx:1.27\n          ports:\n            - containerPort: 80\n`
}

function statefulSetTemplate(namespace: string): string {
  return `apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: sample-statefulset\n  namespace: ${namespace}\nspec:\n  serviceName: sample-statefulset\n  replicas: 1\n  selector:\n    matchLabels:\n      app: sample-statefulset\n  template:\n    metadata:\n      labels:\n        app: sample-statefulset\n    spec:\n      containers:\n        - name: app\n          image: nginx:1.27\n          ports:\n            - containerPort: 80\n  volumeClaimTemplates:\n    - metadata:\n        name: data\n      spec:\n        accessModes: ["ReadWriteOnce"]\n        resources:\n          requests:\n            storage: 5Gi\n`
}

function daemonSetTemplate(namespace: string): string {
  return `apiVersion: apps/v1\nkind: DaemonSet\nmetadata:\n  name: sample-daemonset\n  namespace: ${namespace}\nspec:\n  selector:\n    matchLabels:\n      app: sample-daemonset\n  template:\n    metadata:\n      labels:\n        app: sample-daemonset\n    spec:\n      containers:\n        - name: agent\n          image: busybox:1.36\n          command: ["sh", "-c", "sleep 3600"]\n`
}

function serviceTemplate(namespace: string): string {
  return `apiVersion: v1\nkind: Service\nmetadata:\n  name: sample-service\n  namespace: ${namespace}\nspec:\n  selector:\n    app: sample-app\n  ports:\n    - name: http\n      port: 80\n      targetPort: 80\n  type: ClusterIP\n`
}

function ingressTemplate(namespace: string): string {
  return `apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: sample-ingress\n  namespace: ${namespace}\nspec:\n  ingressClassName: nginx\n  rules:\n    - host: sample.example.com\n      http:\n        paths:\n          - path: /\n            pathType: Prefix\n            backend:\n              service:\n                name: sample-service\n                port:\n                  number: 80\n`
}

function configMapTemplate(namespace: string): string {
  return `apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: sample-config\n  namespace: ${namespace}\ndata:\n  application.yaml: |\n    server:\n      port: 8080\n`
}

function secretTemplate(namespace: string): string {
  return `apiVersion: v1\nkind: Secret\nmetadata:\n  name: sample-secret\n  namespace: ${namespace}\ntype: Opaque\nstringData:\n  username: admin\n  password: change-me\n`
}

function pvcTemplate(namespace: string): string {
  return `apiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  name: sample-pvc\n  namespace: ${namespace}\nspec:\n  accessModes:\n    - ReadWriteOnce\n  resources:\n    requests:\n      storage: 10Gi\n  storageClassName: standard\n`
}

function hpaTemplate(namespace: string): string {
  return `apiVersion: autoscaling/v2\nkind: HorizontalPodAutoscaler\nmetadata:\n  name: sample-hpa\n  namespace: ${namespace}\nspec:\n  scaleTargetRef:\n    apiVersion: apps/v1\n    kind: Deployment\n    name: sample-deployment\n  minReplicas: 1\n  maxReplicas: 5\n  metrics:\n    - type: Resource\n      resource:\n        name: cpu\n        target:\n          type: Utilization\n          averageUtilization: 70\n`
}

function pdbTemplate(namespace: string): string {
  return `apiVersion: policy/v1\nkind: PodDisruptionBudget\nmetadata:\n  name: sample-pdb\n  namespace: ${namespace}\nspec:\n  minAvailable: 1\n  selector:\n    matchLabels:\n      app: sample-app\n`
}

function nodeTemplate(): string {
  return `apiVersion: v1\nkind: Node\nmetadata:\n  name: node-example\n  labels:\n    node-role.kubernetes.io/worker: "true"\n`
}

function buildYaml(resource: ManifestTemplateResource | undefined, workloadKind: WorkloadKind | undefined, namespace: string): { initialYaml: string; sourceLabel: string } {
  if (!resource || resource === 'manifestapply') {
    return { initialYaml: genericTemplate(namespace), sourceLabel: '通用 YAML 示例' }
  }
  if (resource === 'workloads') {
    if (workloadKind === 'StatefulSet') return { initialYaml: statefulSetTemplate(namespace), sourceLabel: 'StatefulSet 模板' }
    if (workloadKind === 'DaemonSet') return { initialYaml: daemonSetTemplate(namespace), sourceLabel: 'DaemonSet 模板' }
    return { initialYaml: deploymentTemplate(namespace), sourceLabel: 'Deployment 模板' }
  }
  if (resource === 'pods') return { initialYaml: podTemplate(namespace), sourceLabel: 'Pod 模板' }
  if (resource === 'services') return { initialYaml: serviceTemplate(namespace), sourceLabel: 'Service 模板' }
  if (resource === 'ingresses') return { initialYaml: ingressTemplate(namespace), sourceLabel: 'Ingress 模板' }
  if (resource === 'configmaps') return { initialYaml: configMapTemplate(namespace), sourceLabel: 'ConfigMap 模板' }
  if (resource === 'secrets') return { initialYaml: secretTemplate(namespace), sourceLabel: 'Secret 模板' }
  if (resource === 'pvcs') return { initialYaml: pvcTemplate(namespace), sourceLabel: 'PVC 模板' }
  if (resource === 'hpas') return { initialYaml: hpaTemplate(namespace), sourceLabel: 'HPA 模板' }
  if (resource === 'pdbs') return { initialYaml: pdbTemplate(namespace), sourceLabel: 'PDB 模板' }
  if (resource === 'nodes') return { initialYaml: nodeTemplate(), sourceLabel: 'Node 模板' }
  const spec = SIMPLE_TEMPLATE_MAP[resource]
  if (spec) {
    return { initialYaml: simpleTemplate(spec, namespace), sourceLabel: `${spec.kind} 模板` }
  }
  return { initialYaml: genericTemplate(namespace), sourceLabel: '通用 YAML 示例' }
}

export function buildManifestApplyPreset(input: TemplateContext): ManifestTemplatePreset {
  const defaultNamespace = String(input.namespace ?? '').trim()
  const templateNamespace = normalizeNamespace(input.namespace)
  const result = buildYaml(input.resource, input.workloadKind, templateNamespace)
  return {
    defaultNamespace,
    initialYaml: result.initialYaml,
    sourceLabel: result.sourceLabel
  }
}