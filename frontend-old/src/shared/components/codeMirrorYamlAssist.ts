import type { Completion, CompletionResult, CompletionSource } from '@codemirror/autocomplete'
import { autocompletion, snippetCompletion } from '@codemirror/autocomplete'
import type { Extension } from '@codemirror/state'
import type { EditorView } from '@codemirror/view'
import type { Diagnostic } from '@codemirror/lint'
import { lintGutter, linter } from '@codemirror/lint'
import { LineCounter, YAMLWarning, parseAllDocuments, stringify } from 'yaml'

export type K8sYamlAssistContext = {
  defaultNamespace?: string
  sourceResource?: string
  workloadKind?: string
}

export type K8sYamlAssistSummary = {
  docsCount: number
  issueCount: number
  association: string
  kinds: string[]
  hints: string[]
}

type KindSpec = {
  kind: string
  apiVersion: string
  namespaced: boolean
  topLevelKeys?: string[]
  specKeys?: string[]
}

type AnalysisDoc = {
  from: number
  to: number
  kind: string
  apiVersion: string
  namespaced: boolean
}

type AnalysisResult = {
  diagnostics: Diagnostic[]
  docs: AnalysisDoc[]
}

const KIND_SPECS: KindSpec[] = [
  { kind: 'Namespace', apiVersion: 'v1', namespaced: false },
  { kind: 'Pod', apiVersion: 'v1', namespaced: true, specKeys: ['containers', 'initContainers', 'restartPolicy', 'serviceAccountName', 'volumes', 'nodeSelector', 'tolerations', 'affinity'] },
  { kind: 'Service', apiVersion: 'v1', namespaced: true, specKeys: ['selector', 'ports', 'type', 'clusterIP', 'sessionAffinity'] },
  { kind: 'ConfigMap', apiVersion: 'v1', namespaced: true, topLevelKeys: ['data', 'binaryData'] },
  { kind: 'Secret', apiVersion: 'v1', namespaced: true, topLevelKeys: ['type', 'data', 'stringData'] },
  { kind: 'ServiceAccount', apiVersion: 'v1', namespaced: true },
  { kind: 'PersistentVolumeClaim', apiVersion: 'v1', namespaced: true, specKeys: ['accessModes', 'resources', 'storageClassName', 'volumeMode', 'volumeName'] },
  { kind: 'PersistentVolume', apiVersion: 'v1', namespaced: false, specKeys: ['capacity', 'accessModes', 'persistentVolumeReclaimPolicy', 'storageClassName', 'hostPath', 'csi', 'nfs'] },
  { kind: 'ResourceQuota', apiVersion: 'v1', namespaced: true, specKeys: ['hard', 'scopeSelector', 'scopes'] },
  { kind: 'LimitRange', apiVersion: 'v1', namespaced: true, specKeys: ['limits'] },
  { kind: 'Deployment', apiVersion: 'apps/v1', namespaced: true, specKeys: ['replicas', 'selector', 'strategy', 'template'] },
  { kind: 'StatefulSet', apiVersion: 'apps/v1', namespaced: true, specKeys: ['serviceName', 'replicas', 'selector', 'template', 'volumeClaimTemplates', 'updateStrategy'] },
  { kind: 'DaemonSet', apiVersion: 'apps/v1', namespaced: true, specKeys: ['selector', 'template', 'updateStrategy', 'minReadySeconds'] },
  { kind: 'ReplicaSet', apiVersion: 'apps/v1', namespaced: true, specKeys: ['replicas', 'selector', 'template'] },
  { kind: 'Job', apiVersion: 'batch/v1', namespaced: true, specKeys: ['parallelism', 'completions', 'backoffLimit', 'template'] },
  { kind: 'CronJob', apiVersion: 'batch/v1', namespaced: true, specKeys: ['schedule', 'jobTemplate', 'concurrencyPolicy', 'successfulJobsHistoryLimit', 'failedJobsHistoryLimit'] },
  { kind: 'HorizontalPodAutoscaler', apiVersion: 'autoscaling/v2', namespaced: true, specKeys: ['scaleTargetRef', 'minReplicas', 'maxReplicas', 'metrics', 'behavior'] },
  { kind: 'PodDisruptionBudget', apiVersion: 'policy/v1', namespaced: true, specKeys: ['minAvailable', 'maxUnavailable', 'selector'] },
  { kind: 'Ingress', apiVersion: 'networking.k8s.io/v1', namespaced: true, specKeys: ['ingressClassName', 'rules', 'tls', 'defaultBackend'] },
  { kind: 'IngressClass', apiVersion: 'networking.k8s.io/v1', namespaced: false, specKeys: ['controller', 'parameters'] },
  { kind: 'NetworkPolicy', apiVersion: 'networking.k8s.io/v1', namespaced: true, specKeys: ['podSelector', 'policyTypes', 'ingress', 'egress'] },
  { kind: 'Lease', apiVersion: 'coordination.k8s.io/v1', namespaced: true, specKeys: ['holderIdentity', 'leaseDurationSeconds', 'renewTime'] },
  { kind: 'Role', apiVersion: 'rbac.authorization.k8s.io/v1', namespaced: true, topLevelKeys: ['rules'] },
  { kind: 'ClusterRole', apiVersion: 'rbac.authorization.k8s.io/v1', namespaced: false, topLevelKeys: ['rules'] },
  { kind: 'RoleBinding', apiVersion: 'rbac.authorization.k8s.io/v1', namespaced: true, topLevelKeys: ['subjects', 'roleRef'] },
  { kind: 'ClusterRoleBinding', apiVersion: 'rbac.authorization.k8s.io/v1', namespaced: false, topLevelKeys: ['subjects', 'roleRef'] },
  { kind: 'StorageClass', apiVersion: 'storage.k8s.io/v1', namespaced: false, topLevelKeys: ['provisioner', 'reclaimPolicy', 'volumeBindingMode', 'allowVolumeExpansion', 'parameters'] },
  { kind: 'CSIDriver', apiVersion: 'storage.k8s.io/v1', namespaced: false, specKeys: ['attachRequired', 'podInfoOnMount', 'volumeLifecycleModes'] },
  { kind: 'CSINode', apiVersion: 'storage.k8s.io/v1', namespaced: false, specKeys: ['drivers'] },
  { kind: 'VolumeAttachment', apiVersion: 'storage.k8s.io/v1', namespaced: false, specKeys: ['attacher', 'nodeName', 'source'] },
  { kind: 'VolumeSnapshot', apiVersion: 'snapshot.storage.k8s.io/v1', namespaced: true, specKeys: ['volumeSnapshotClassName', 'source'] },
  { kind: 'VolumeSnapshotClass', apiVersion: 'snapshot.storage.k8s.io/v1', namespaced: false, topLevelKeys: ['driver', 'deletionPolicy', 'parameters'] },
  { kind: 'VolumeSnapshotContent', apiVersion: 'snapshot.storage.k8s.io/v1', namespaced: false, specKeys: ['volumeSnapshotRef', 'deletionPolicy', 'driver', 'source'] },
  { kind: 'CustomResourceDefinition', apiVersion: 'apiextensions.k8s.io/v1', namespaced: false, specKeys: ['group', 'scope', 'names', 'versions'] },
  { kind: 'APIService', apiVersion: 'apiregistration.k8s.io/v1', namespaced: false, specKeys: ['group', 'version', 'service', 'groupPriorityMinimum', 'versionPriority'] },
  { kind: 'PriorityClass', apiVersion: 'scheduling.k8s.io/v1', namespaced: false, topLevelKeys: ['value', 'globalDefault', 'description', 'preemptionPolicy'] },
  { kind: 'RuntimeClass', apiVersion: 'node.k8s.io/v1', namespaced: false, topLevelKeys: ['handler', 'overhead', 'scheduling'] },
  { kind: 'ValidatingWebhookConfiguration', apiVersion: 'admissionregistration.k8s.io/v1', namespaced: false, topLevelKeys: ['webhooks'] },
  { kind: 'MutatingWebhookConfiguration', apiVersion: 'admissionregistration.k8s.io/v1', namespaced: false, topLevelKeys: ['webhooks'] },
  { kind: 'ValidatingAdmissionPolicy', apiVersion: 'admissionregistration.k8s.io/v1', namespaced: false, specKeys: ['failurePolicy', 'matchConstraints', 'validations'] },
  { kind: 'ValidatingAdmissionPolicyBinding', apiVersion: 'admissionregistration.k8s.io/v1', namespaced: false, specKeys: ['policyName', 'validationActions', 'matchResources'] }
]

const KIND_SPEC_MAP = new Map(KIND_SPECS.map((item) => [item.kind, item]))
const RESOURCE_KIND_MAP: Record<string, string> = {
  pods: 'Pod',
  services: 'Service',
  ingresses: 'Ingress',
  configmaps: 'ConfigMap',
  secrets: 'Secret',
  pvcs: 'PersistentVolumeClaim',
  pvs: 'PersistentVolume',
  jobs: 'Job',
  cronjobs: 'CronJob',
  hpas: 'HorizontalPodAutoscaler',
  pdbs: 'PodDisruptionBudget',
  namespaces: 'Namespace',
  nodes: 'Node',
  roles: 'Role',
  clusterroles: 'ClusterRole',
  rolebindings: 'RoleBinding',
  clusterrolebindings: 'ClusterRoleBinding',
  storageclasses: 'StorageClass',
  resourcequotas: 'ResourceQuota',
  limitranges: 'LimitRange',
  leases: 'Lease',
  volumesnapshots: 'VolumeSnapshot',
  volumesnapshotclasses: 'VolumeSnapshotClass',
  volumesnapshotcontents: 'VolumeSnapshotContent'
}

const TOP_LEVEL_KEYS = ['apiVersion', 'kind', 'metadata', 'spec', 'data', 'stringData', 'type', 'rules', 'subjects', 'roleRef', 'webhooks']
const METADATA_KEYS = ['name', 'namespace', 'labels', 'annotations', 'finalizers']
const CONTAINER_KEYS = ['name', 'image', 'imagePullPolicy', 'command', 'args', 'ports', 'env', 'resources', 'volumeMounts', 'livenessProbe', 'readinessProbe']
const COMMON_API_VERSIONS = Array.from(new Set(KIND_SPECS.map((item) => item.apiVersion))).sort()
const COMMON_NAMESPACE_VALUES = ['default', 'kube-system', 'kube-public', 'kube-node-lease']

const DOCUMENT_SNIPPETS: Completion[] = [
  snippetCompletion('apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: ${name}\n  namespace: ${namespace}\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: ${name}\n  template:\n    metadata:\n      labels:\n        app: ${name}\n    spec:\n      containers:\n        - name: app\n          image: nginx:1.27\n          ports:\n            - containerPort: 80\n', { label: 'Deployment 模板', detail: 'apps/v1', type: 'keyword' }),
  snippetCompletion('apiVersion: v1\nkind: Service\nmetadata:\n  name: ${name}\n  namespace: ${namespace}\nspec:\n  selector:\n    app: ${name}\n  ports:\n    - port: 80\n      targetPort: 80\n  type: ClusterIP\n', { label: 'Service 模板', detail: 'v1', type: 'keyword' }),
  snippetCompletion('apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: ${name}\n  namespace: ${namespace}\ndata:\n  app.yaml: |\n    key: value\n', { label: 'ConfigMap 模板', detail: 'v1', type: 'keyword' }),
  snippetCompletion('apiVersion: v1\nkind: Secret\nmetadata:\n  name: ${name}\n  namespace: ${namespace}\ntype: Opaque\nstringData:\n  username: admin\n  password: change-me\n', { label: 'Secret 模板', detail: 'v1', type: 'keyword' }),
  snippetCompletion('apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: ${name}\n  namespace: ${namespace}\nspec:\n  ingressClassName: nginx\n  rules:\n    - host: ${name}.example.com\n      http:\n        paths:\n          - path: /\n            pathType: Prefix\n            backend:\n              service:\n                name: ${service}\n                port:\n                  number: 80\n', { label: 'Ingress 模板', detail: 'networking.k8s.io/v1', type: 'keyword' }),
  snippetCompletion('apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: ${name}\n  namespace: ${namespace}\nspec:\n  template:\n    spec:\n      restartPolicy: Never\n      containers:\n        - name: job\n          image: busybox:1.36\n          command: ["sh", "-c", "echo hello"]\n', { label: 'Job 模板', detail: 'batch/v1', type: 'keyword' })
]

export function buildK8sYamlAssistExtensions(context?: K8sYamlAssistContext): Extension[] {
  return [
    lintGutter(),
    linter((view) => analyzeK8sYaml(view.state.doc.toString(), context).diagnostics),
    autocompletion({
      override: [buildCompletionSource(context)],
      activateOnTyping: true,
      icons: false
    })
  ]
}

export function analyzeK8sYamlSummary(text: string, context?: K8sYamlAssistContext): K8sYamlAssistSummary {
  const analysis = analyzeK8sYaml(text, context)
  const kinds = Array.from(new Set(analysis.docs.map((item) => item.kind).filter(Boolean)))
  const primary = analysis.docs[0]
  const association = primary?.kind
    ? `${primary.apiVersion || 'unknown'}/${primary.kind}`
    : inferAssociationLabel(context)

  const hints: string[] = []
  if (context?.defaultNamespace) hints.push(`默认命名空间 ${context.defaultNamespace}`)
  if (context?.workloadKind) hints.push(`模板关联 ${context.workloadKind}`)

  return {
    docsCount: analysis.docs.length,
    issueCount: analysis.diagnostics.length,
    association,
    kinds,
    hints
  }
}

export function formatK8sYaml(text: string): { text: string; changed: boolean; error?: string } {
  const value = String(text ?? '')
  if (!value.trim()) return { text: value, changed: false }

  const parsed = parseYamlDocuments(value)
  if (parsed.hasErrors) {
    const first = parsed.diagnostics.find((item) => item.severity === 'error')
    return { text: value, changed: false, error: first?.message || 'YAML 解析失败，无法格式化' }
  }

  const docs = parsed.objects.map((item) => normalizeManifestObject(item, {}, false).value)
  const next = stringifyDocuments(docs)
  return { text: next, changed: next !== value }
}

export function smartFixK8sYaml(text: string, context?: K8sYamlAssistContext): { text: string; changed: boolean; notes: string[]; error?: string } {
  const original = String(text ?? '')
  const notes: string[] = []
  let next = original.replace(/\t/g, '  ')
  if (next !== original) notes.push('已将制表符替换为空格缩进')

  const parsed = parseYamlDocuments(next)
  if (parsed.hasErrors) {
    return {
      text: next,
      changed: next !== original,
      notes,
      error: '存在 YAML 语法错误，已执行基础缩进修正；请先处理语法问题后再进行智能修正'
    }
  }

  const docs = parsed.objects.map((item) => {
    const normalized = normalizeManifestObject(item, context ?? {}, true)
    notes.push(...normalized.notes)
    return normalized.value
  })

  next = stringifyDocuments(docs)
  return { text: next, changed: next !== original, notes: Array.from(new Set(notes)) }
}

function analyzeK8sYaml(text: string, context?: K8sYamlAssistContext): AnalysisResult {
  return parseYamlDocuments(String(text ?? ''), context)
}

function parseYamlDocuments(text: string, context?: K8sYamlAssistContext): AnalysisResult & { objects: unknown[]; hasErrors: boolean } {
  const source = String(text ?? '')
  if (!source.trim()) {
    return { diagnostics: [], docs: [], objects: [], hasErrors: false }
  }

  const lineCounter = new LineCounter()
  const documents = parseAllDocuments(source, { lineCounter, prettyErrors: false, uniqueKeys: false })
  const diagnostics: Diagnostic[] = []
  const docs: AnalysisDoc[] = []
  const objects: unknown[] = []

  for (const document of documents) {
    const range = Array.isArray((document as { range?: [number, number, number?] }).range)
      ? (document as { range: [number, number, number?] }).range
      : [0, Math.max(source.length, 1)]
    const from = clampPosition(range[0], source.length)
    const to = clampRangeEnd(range[1], from, source.length)

    for (const error of document.errors) {
      diagnostics.push(toYamlDiagnostic(error, source, context))
    }
    for (const warning of document.warnings) {
      diagnostics.push(toYamlDiagnostic(warning, source, context))
    }

    if (document.errors.length > 0) continue

    const raw = typeof document.toJS === 'function' ? document.toJS() : null
    objects.push(raw)

    if (!isPlainObject(raw)) {
      diagnostics.push({
        from,
        to,
        severity: 'warning',
        source: 'k8s-yaml',
        message: 'Kubernetes 清单顶层应为对象映射，例如 apiVersion / kind / metadata',
        actions: [buildSmartFixAction(context)]
      })
      continue
    }

    const object = raw as Record<string, unknown>
    const kind = asText(object.kind)
    const apiVersion = asText(object.apiVersion)
    const inferredKind = kind || inferKindFromContext(context)
    const spec = inferredKind ? KIND_SPEC_MAP.get(inferredKind) : undefined
    const metadata = isPlainObject(object.metadata) ? (object.metadata as Record<string, unknown>) : null

    docs.push({
      from,
      to,
      kind: inferredKind,
      apiVersion: apiVersion || spec?.apiVersion || '',
      namespaced: spec?.namespaced ?? false
    })

    if (!kind) {
      diagnostics.push({
        from,
        to: Math.min(from + 1, source.length),
        severity: 'warning',
        source: 'k8s-yaml',
        message: '缺少 kind，无法准确关联 Kubernetes 资源模板',
        actions: [buildSmartFixAction(context)]
      })
    }

    if (!apiVersion) {
      diagnostics.push({
        from,
        to: Math.min(from + 1, source.length),
        severity: 'warning',
        source: 'k8s-yaml',
        message: spec?.apiVersion ? `缺少 apiVersion，${inferredKind || '当前资源'} 通常使用 ${spec.apiVersion}` : '缺少 apiVersion',
        actions: [buildSmartFixAction(context)]
      })
    }

    if (!metadata) {
      diagnostics.push({
        from,
        to: Math.min(from + 1, source.length),
        severity: 'warning',
        source: 'k8s-yaml',
        message: '缺少 metadata 块',
        actions: [buildSmartFixAction(context)]
      })
      continue
    }

    if (!asText(metadata.name)) {
      diagnostics.push({
        from,
        to,
        severity: 'warning',
        source: 'k8s-yaml',
        message: 'metadata.name 不能为空',
        actions: [buildSmartFixAction(context)]
      })
    }

    if (spec?.namespaced && !asText(metadata.namespace) && context?.defaultNamespace) {
      diagnostics.push({
        from,
        to,
        severity: 'info',
        source: 'k8s-yaml',
        message: `未显式指定 metadata.namespace，可自动关联到默认命名空间 ${context.defaultNamespace}`,
        actions: [buildSmartFixAction(context)]
      })
    }

    if (context?.workloadKind && kind && context.workloadKind !== kind) {
      diagnostics.push({
        from,
        to,
        severity: 'info',
        source: 'k8s-yaml',
        message: `当前工作台关联的模板类型为 ${context.workloadKind}，但文档 kind 为 ${kind}`
      })
    }
  }

  return {
    diagnostics,
    docs,
    objects,
    hasErrors: diagnostics.some((item) => item.severity === 'error')
  }
}

function toYamlDiagnostic(error: YAMLWarning | { name: string; pos?: [number, number]; message: string }, text: string, context?: K8sYamlAssistContext): Diagnostic {
  const start = clampPosition(error.pos?.[0] ?? 0, text.length)
  const end = clampRangeEnd(error.pos?.[1] ?? start + 1, start, text.length)
  return {
    from: start,
    to: end,
    severity: error instanceof YAMLWarning || error.name === 'YAMLWarning' ? 'warning' : 'error',
    source: 'yaml',
    message: error.message,
    actions: buildErrorActions(error.message, context)
  }
}

function buildErrorActions(message: string, context?: K8sYamlAssistContext) {
  const actions = [buildSmartFixAction(context)]
  if (/tab/i.test(message)) {
    actions.unshift({
      name: '替换制表符',
      apply(view) {
        const current = view.state.doc.toString()
        const next = current.replace(/\t/g, '  ')
        if (next === current) return
        replaceWholeDocument(view, next)
      }
    })
  }
  return actions
}

function buildSmartFixAction(context?: K8sYamlAssistContext) {
  return {
    name: '智能修正',
    apply(view: EditorView) {
      const current = view.state.doc.toString()
      const fixed = smartFixK8sYaml(current, context)
      if (!fixed.changed) return
      replaceWholeDocument(view, fixed.text)
    }
  }
}

function replaceWholeDocument(view: EditorView, next: string) {
  const current = view.state.doc.toString()
  if (current === next) return
  view.dispatch({ changes: { from: 0, to: current.length, insert: next } })
}

function buildCompletionSource(context?: K8sYamlAssistContext): CompletionSource {
  return (completionContext) => {
    const docText = completionContext.state.doc.toString()
    const pos = completionContext.pos
    const line = completionContext.state.doc.lineAt(pos)
    const before = line.text.slice(0, pos - line.from)
    const token = completionContext.matchBefore(/[\w./-]*/)
    const currentKind = inferPrimaryKind(docText, context)
    const spec = currentKind ? KIND_SPEC_MAP.get(currentKind) : undefined

    if (!completionContext.explicit && token && token.from === token.to && !/[:\-\w]/.test(before.slice(-1))) {
      return null
    }

    if (/^\s*kind:\s*[\w-]*$/i.test(before)) {
      return buildCompletionResult(token?.from ?? pos, KIND_SPECS.map((item) => ({ label: item.kind, type: 'class', detail: item.apiVersion })))
    }

    if (/^\s*apiVersion:\s*[\w./-]*$/i.test(before)) {
      const options = spec
        ? [{ label: spec.apiVersion, type: 'constant', detail: `${spec.kind} 推荐` }, ...COMMON_API_VERSIONS.filter((item) => item !== spec.apiVersion).map((item) => ({ label: item, type: 'constant' }))]
        : COMMON_API_VERSIONS.map((item) => ({ label: item, type: 'constant' }))
      return buildCompletionResult(token?.from ?? pos, options)
    }

    if (/^\s*namespace:\s*[\w-]*$/i.test(before)) {
      const namespaces = Array.from(new Set([context?.defaultNamespace, ...COMMON_NAMESPACE_VALUES].filter(Boolean)))
      return buildCompletionResult(token?.from ?? pos, namespaces.map((item) => ({ label: item as string, type: 'variable' })))
    }

    const path = inferYamlPath(completionContext.state.doc.sliceString(0, pos))
    const options = buildContextCompletions(path, spec, context)
    if (options.length > 0) {
      return buildCompletionResult(token?.from ?? pos, options)
    }

    if (isDocumentStartPosition(docText, pos, before)) {
      return buildCompletionResult(token?.from ?? pos, DOCUMENT_SNIPPETS)
    }

    return buildCompletionResult(token?.from ?? pos, TOP_LEVEL_KEYS.map((item) => ({ label: item, type: 'property' })))
  }
}

function buildContextCompletions(path: string[], spec: KindSpec | undefined, context?: K8sYamlAssistContext): Completion[] {
  const pathKey = path.join('.')

  if (pathKey.endsWith('metadata')) {
    return METADATA_KEYS.map((item) => ({ label: item, type: 'property' }))
  }

  if (pathKey.endsWith('spec')) {
    return (spec?.specKeys ?? ['selector', 'template']).map((item) => ({ label: item, type: 'property' }))
  }

  if (pathKey.includes('containers')) {
    return CONTAINER_KEYS.map((item) => ({ label: item, type: 'property' }))
  }

  if (pathKey === '') {
    const items: Completion[] = TOP_LEVEL_KEYS.map((item) => ({ label: item, type: 'property' }))
    if (context?.defaultNamespace) {
      items.push({ label: 'namespace', detail: context.defaultNamespace, type: 'variable' })
    }
    return [...DOCUMENT_SNIPPETS, ...items]
  }

  return []
}

function inferYamlPath(text: string): string[] {
  const lines = text.split(/\r?\n/)
  const stack: Array<{ indent: number; key: string }> = []

  for (const rawLine of lines) {
    const line = rawLine.replace(/\t/g, '  ')
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue

    const indent = line.match(/^\s*/)?.[0].length ?? 0
    while (stack.length > 0 && indent <= stack[stack.length - 1].indent) {
      stack.pop()
    }

    const keyMatch = trimmed.match(/^([A-Za-z0-9_.-]+):/)
    if (keyMatch) {
      stack.push({ indent, key: keyMatch[1] })
      continue
    }

    if (/^-\s+/.test(trimmed) && stack[stack.length - 1]?.key === 'containers') {
      stack.push({ indent, key: 'containers' })
    }
  }

  return stack.map((item) => item.key)
}

function inferPrimaryKind(text: string, context?: K8sYamlAssistContext): string {
  const analysis = analyzeK8sYaml(text, context)
  return analysis.docs[0]?.kind || inferKindFromContext(context)
}

function inferKindFromContext(context?: K8sYamlAssistContext): string {
  const workloadKind = asText(context?.workloadKind)
  if (workloadKind) return workloadKind
  const resource = asText(context?.sourceResource)
  return resource ? RESOURCE_KIND_MAP[resource] || '' : ''
}

function inferAssociationLabel(context?: K8sYamlAssistContext): string {
  const inferredKind = inferKindFromContext(context)
  const spec = inferredKind ? KIND_SPEC_MAP.get(inferredKind) : undefined
  if (spec) return `${spec.apiVersion}/${spec.kind}`
  return 'Kubernetes 通用清单'
}

function normalizeManifestObject(input: unknown, context: K8sYamlAssistContext, fixMissing: boolean): { value: unknown; notes: string[] } {
  if (!isPlainObject(input)) {
    return { value: input, notes: [] }
  }

  const value: Record<string, unknown> = { ...(input as Record<string, unknown>) }
  const notes: string[] = []
  let kind = asText(value.kind)

  if (!kind && fixMissing) {
    const inferredKind = inferKindFromContext(context)
    if (inferredKind) {
      value.kind = inferredKind
      kind = inferredKind
      notes.push(`已补充 kind=${inferredKind}`)
    }
  }

  const spec = kind ? KIND_SPEC_MAP.get(kind) : undefined
  if (!asText(value.apiVersion) && fixMissing && spec?.apiVersion) {
    value.apiVersion = spec.apiVersion
    notes.push(`已补充 apiVersion=${spec.apiVersion}`)
  }

  let metadata = isPlainObject(value.metadata) ? { ...(value.metadata as Record<string, unknown>) } : null
  if (!metadata && fixMissing) {
    metadata = {}
    notes.push('已补充 metadata 块')
  }

  if (metadata) {
    if (!asText(metadata.name) && fixMissing) {
      metadata.name = buildPlaceholderName(kind || 'resource')
      notes.push(`已补充 metadata.name=${String(metadata.name)}`)
    }
    if (spec?.namespaced && context.defaultNamespace && !asText(metadata.namespace) && fixMissing) {
      metadata.namespace = context.defaultNamespace
      notes.push(`已自动关联命名空间 ${context.defaultNamespace}`)
    }
    value.metadata = metadata
  }

  return { value: sortTopLevelKeys(value), notes }
}

function sortTopLevelKeys(value: Record<string, unknown>) {
  const order = ['apiVersion', 'kind', 'metadata', 'spec', 'type', 'data', 'stringData', 'subjects', 'roleRef', 'rules', 'webhooks']
  const next: Record<string, unknown> = {}
  for (const key of order) {
    if (key in value) next[key] = value[key]
  }
  for (const [key, item] of Object.entries(value)) {
    if (!(key in next)) next[key] = item
  }
  return next
}

function stringifyDocuments(documents: unknown[]): string {
  return documents
    .map((item) => stringify(item, { indent: 2, lineWidth: 0 }).trim())
    .filter(Boolean)
    .join('\n---\n') + '\n'
}

function buildPlaceholderName(kind: string): string {
  const normalized = String(kind ?? '')
    .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
    .replace(/[^a-zA-Z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .toLowerCase()
  return `sample-${normalized || 'resource'}`
}

function buildCompletionResult(from: number, options: readonly Completion[]): CompletionResult {
  return {
    from,
    options,
    validFor: /^[\w./-]*$/
  }
}

function isDocumentStartPosition(text: string, pos: number, before: string): boolean {
  if (!text.trim()) return true
  if (!before.trim()) return true
  const prefix = text.slice(Math.max(0, pos - 4), pos)
  return prefix === '---\n'
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function asText(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function clampPosition(value: number, max: number): number {
  const numeric = Number.isFinite(value) ? Math.max(0, Math.trunc(value)) : 0
  return Math.min(numeric, Math.max(max, 0))
}

function clampRangeEnd(value: number, from: number, max: number): number {
  const numeric = Number.isFinite(value) ? Math.trunc(value) : from + 1
  return Math.max(from + 1, Math.min(Math.max(max, 1), numeric))
}