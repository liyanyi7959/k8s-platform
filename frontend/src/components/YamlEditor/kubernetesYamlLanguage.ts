import type { Monaco } from '@monaco-editor/react'

export type YamlDiagnosticSummary = { errors: number; warnings: number }

type MarkerLike = {
  severity: number
  message: string
  source: string
  startLineNumber: number
  endLineNumber: number
  startColumn: number
  endColumn: number
}

const KIND_API_VERSIONS: Record<string, string[]> = {
  Pod: ['v1'], Service: ['v1'], ConfigMap: ['v1'], Secret: ['v1'], Namespace: ['v1'], PersistentVolumeClaim: ['v1'],
  Deployment: ['apps/v1'], StatefulSet: ['apps/v1'], DaemonSet: ['apps/v1'],
  Job: ['batch/v1'], CronJob: ['batch/v1'],
  Ingress: ['networking.k8s.io/v1'], NetworkPolicy: ['networking.k8s.io/v1'],
  HorizontalPodAutoscaler: ['autoscaling/v2'], PodDisruptionBudget: ['policy/v1'],
  ServiceAccount: ['v1'], Role: ['rbac.authorization.k8s.io/v1'], RoleBinding: ['rbac.authorization.k8s.io/v1'],
  ClusterRole: ['rbac.authorization.k8s.io/v1'], ClusterRoleBinding: ['rbac.authorization.k8s.io/v1'],
  ResourceQuota: ['v1'], LimitRange: ['v1'], PersistentVolume: ['v1'],
  StorageClass: ['storage.k8s.io/v1'],
}

const resourceSnippets = [
  ['pod', 'Pod 基础清单', `apiVersion: v1\nkind: Pod\nmetadata:\n  name: \${1:pod-name}\n  namespace: \${2:default}\n  labels:\n    app.kubernetes.io/name: \${1:pod-name}\nspec:\n  containers:\n    - name: \${3:app}\n      image: \${4:nginx:1.27-alpine}\n      ports:\n        - name: http\n          containerPort: \${5:80}\n      resources:\n        requests:\n          cpu: 100m\n          memory: 128Mi\n        limits:\n          cpu: 500m\n          memory: 512Mi\n  restartPolicy: Always`],
  ['deploy', 'Deployment 基础清单', `apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: \${1:app}\n  namespace: \${2:default}\nspec:\n  replicas: \${3:2}\n  selector:\n    matchLabels:\n      app: \${1:app}\n  template:\n    metadata:\n      labels:\n        app: \${1:app}\n    spec:\n      containers:\n        - name: \${1:app}\n          image: \${4:nginx:1.27-alpine}\n          ports:\n            - containerPort: \${5:80}`],
  ['svc', 'Service 基础清单', `apiVersion: v1\nkind: Service\nmetadata:\n  name: \${1:app}\n  namespace: \${2:default}\nspec:\n  selector:\n    app: \${1:app}\n  ports:\n    - name: http\n      port: \${3:80}\n      targetPort: \${4:80}\n  type: ClusterIP`],
  ['configmap', 'ConfigMap 基础清单', `apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: \${1:app-config}\n  namespace: \${2:default}\ndata:\n  \${3:KEY}: \${4:value}`],
  ['secret', 'Secret 基础清单', `apiVersion: v1\nkind: Secret\nmetadata:\n  name: \${1:app-secret}\n  namespace: \${2:default}\ntype: Opaque\nstringData:\n  \${3:password}: \${4:change-me}`],
  ['ingress', 'Ingress 基础清单', `apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: \${1:app}\n  namespace: \${2:default}\nspec:\n  ingressClassName: \${3:nginx}\n  rules:\n    - host: \${4:app.example.com}\n      http:\n        paths:\n          - path: /\n            pathType: Prefix\n            backend:\n              service:\n                name: \${1:app}\n                port:\n                  number: \${5:80}`],
  ['job', 'Job 基础清单', `apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: \${1:task}\n  namespace: \${2:default}\nspec:\n  backoffLimit: 3\n  template:\n    spec:\n      restartPolicy: Never\n      containers:\n        - name: \${1:task}\n          image: \${3:busybox:1.36}\n          command: [sh, -c, \"\${4:date}\"]`],
  ['cronjob', 'CronJob 基础清单', `apiVersion: batch/v1\nkind: CronJob\nmetadata:\n  name: \${1:scheduled-task}\n  namespace: \${2:default}\nspec:\n  schedule: \"\${3:0 * * * *}\"\n  concurrencyPolicy: Forbid\n  jobTemplate:\n    spec:\n      template:\n        spec:\n          restartPolicy: Never\n          containers:\n            - name: task\n              image: \${4:busybox:1.36}\n              command: [sh, -c, \"date\"]`],
  ['pvc', 'PVC 基础清单', `apiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  name: \${1:app-data}\n  namespace: \${2:default}\nspec:\n  accessModes: [ReadWriteOnce]\n  resources:\n    requests:\n      storage: \${3:10Gi}`],
  ['hpa', 'HPA 基础清单', `apiVersion: autoscaling/v2\nkind: HorizontalPodAutoscaler\nmetadata:\n  name: \${1:app}\n  namespace: \${2:default}\nspec:\n  scaleTargetRef:\n    apiVersion: apps/v1\n    kind: Deployment\n    name: \${1:app}\n  minReplicas: \${3:2}\n  maxReplicas: \${4:10}\n  metrics:\n    - type: Resource\n      resource:\n        name: cpu\n        target:\n          type: Utilization\n          averageUtilization: \${5:70}`],
] as const

const fieldSuggestions = [
  ['apiVersion', 'apiVersion: ${1:v1}', '资源 API 版本'], ['kind', 'kind: ${1:Pod}', 'Kubernetes 资源类型'],
  ['metadata', 'metadata:\n  name: ${1:name}\n  namespace: ${2:default}', '资源元数据'], ['spec', 'spec:\n  ${1}', '资源期望状态'],
  ['labels', 'labels:\n  app.kubernetes.io/name: ${1:app}', '资源标签'], ['annotations', 'annotations:\n  ${1:key}: ${2:value}', '资源注解'],
  ['containers', 'containers:\n  - name: ${1:app}\n    image: ${2:nginx:latest}', '容器列表'], ['resources', 'resources:\n  requests:\n    cpu: 100m\n    memory: 128Mi\n  limits:\n    cpu: 500m\n    memory: 512Mi', '资源请求与限制'],
  ['readinessProbe', 'readinessProbe:\n  httpGet:\n    path: ${1:/healthz}\n    port: ${2:http}', '就绪探针'], ['livenessProbe', 'livenessProbe:\n  httpGet:\n    path: ${1:/healthz}\n    port: ${2:http}', '存活探针'],
] as const

function currentWordRange(monaco: Monaco, model: any, lineNumber: number, column: number) {
  const word = model.getWordUntilPosition({ lineNumber, column })
  return new monaco.Range(lineNumber, word.startColumn, lineNumber, word.endColumn)
}

export function registerKubernetesYamlLanguage(monaco: Monaco) {
  const globalState = globalThis as typeof globalThis & { __k8sYamlLanguageRegistered?: boolean }
  if (globalState.__k8sYamlLanguageRegistered) return
  globalState.__k8sYamlLanguageRegistered = true

  monaco.languages.registerCompletionItemProvider('yaml', {
    triggerCharacters: [' ', ':', '\n'],
    provideCompletionItems(model: any, position: { lineNumber: number; column: number }) {
      const range = currentWordRange(monaco, model, position.lineNumber, position.column)
      const linePrefix = model.getLineContent(position.lineNumber).slice(0, position.column - 1)
      const enumGroups: Array<[RegExp, string[], string]> = [
        [/restartPolicy:\s*\w*$/, ['Always', 'OnFailure', 'Never'], 'Pod 重启策略'],
        [/imagePullPolicy:\s*\w*$/, ['Always', 'IfNotPresent', 'Never'], '镜像拉取策略'],
        [/concurrencyPolicy:\s*\w*$/, ['Allow', 'Forbid', 'Replace'], 'CronJob 并发策略'],
        [/pathType:\s*\w*$/, ['Exact', 'Prefix', 'ImplementationSpecific'], 'Ingress 路径类型'],
        [/protocol:\s*\w*$/, ['TCP', 'UDP', 'SCTP'], '网络协议'],
        [/kind:\s*\w*$/, Object.keys(KIND_API_VERSIONS), 'Kubernetes 资源类型'],
        [/apiVersion:\s*[\w/.-]*$/, Array.from(new Set(Object.values(KIND_API_VERSIONS).flat())), 'Kubernetes API 版本'],
      ]
      const enumGroup = enumGroups.find(([pattern]) => pattern.test(linePrefix))
      if (enumGroup) {
        return { suggestions: enumGroup[1].map((value) => ({
          label: value, detail: enumGroup[2], insertText: value, range,
          kind: monaco.languages.CompletionItemKind.EnumMember,
        })) }
      }
      const documentPrefix = model.getValueInRange(new monaco.Range(1, 1, position.lineNumber, position.column))
      const activeDocumentPrefix = documentPrefix.split(/^---\s*$/m).pop() || ''
      const isDocumentStart = activeDocumentPrefix.trim().split(/\s+/).length <= 2 && !linePrefix.includes(':')
      const snippets = isDocumentStart ? resourceSnippets.map(([label, detail, insertText]) => ({
        label, detail, insertText, range,
        kind: monaco.languages.CompletionItemKind.Snippet,
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        documentation: `输入 ${label} 后按 Tab/Enter 生成${detail}`,
        sortText: `0-${label}`,
      })) : []
      const indent = linePrefix.match(/^\s*/)?.[0].length || 0
      const availableFields = indent === 0 ? fieldSuggestions.slice(0, 4) : fieldSuggestions
      const fields = availableFields.map(([label, insertText, detail]) => ({
        label, detail, insertText, range,
        kind: monaco.languages.CompletionItemKind.Property,
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        sortText: `1-${label}`,
      }))
      return { suggestions: [...snippets, ...fields] }
    },
  })

  monaco.languages.registerHoverProvider('yaml', {
    provideHover(model: any, position: { lineNumber: number; column: number }) {
      const word = model.getWordAtPosition(position)?.word
      const help: Record<string, string> = {
        apiVersion: '资源使用的 Kubernetes API 组与版本。', kind: '资源类型，必须与 apiVersion 匹配。', metadata: '名称、命名空间、标签和注解。',
        spec: '资源期望状态。字段结构由 kind 决定。', containers: 'Pod 容器数组，必须使用 `- name:` 列表格式。', resources: '容器 CPU/内存 requests 与 limits。',
      }
      return word && help[word] ? { contents: [{ value: `**${word}**\n\n${help[word]}` }] } : null
    },
  })
}

function unquote(value: string) {
  return value.trim().replace(/^['"]|['"]$/g, '')
}

export function validateKubernetesYaml(monaco: Monaco, model: any): YamlDiagnosticSummary {
  const markers: MarkerLike[] = []
  const lines: string[] = model.getLinesContent()
  const documents: Array<{ start: number; lines: Array<{ number: number; text: string; indent: number }> }> = [{ start: 1, lines: [] }]
  const add = (line: number, severity: number, message: string, startColumn = 1, endColumn?: number) => markers.push({
    severity, message, source: 'Kubernetes YAML', startLineNumber: line, endLineNumber: line,
    startColumn, endColumn: endColumn || Math.max(startColumn + 1, lines[line - 1]!.length + 1),
  })

  lines.forEach((text, index) => {
    const number = index + 1
    if (/^\s*---\s*$/.test(text)) { documents.push({ start: number + 1, lines: [] }); return }
    const indent = text.match(/^\s*/)?.[0].length || 0
    if (text.includes('\t')) add(number, monaco.MarkerSeverity.Error, 'YAML 不允许使用 Tab 缩进，请使用空格。')
    if (indent % 2 !== 0 && text.trim()) add(number, monaco.MarkerSeverity.Warning, '建议使用 2 个空格缩进，当前缩进层级不一致。')
    const content = text.trim()
    if (!content || content.startsWith('#')) return
    const sequenceItem = content.startsWith('- ')
    const mapping = sequenceItem ? content.slice(2).trim() : content
    if (!mapping.includes(':') && !sequenceItem) add(number, monaco.MarkerSeverity.Error, '映射项缺少冒号 `:`。')
    const square = (mapping.match(/\[/g) || []).length - (mapping.match(/\]/g) || []).length
    const curly = (mapping.match(/{/g) || []).length - (mapping.match(/}/g) || []).length
    if (square !== 0 || curly !== 0) add(number, monaco.MarkerSeverity.Error, '行内数组或对象括号未闭合。')
    const value = mapping.includes(':') ? mapping.slice(mapping.indexOf(':') + 1).trim() : ''
    if ((value.startsWith('"') && !value.endsWith('"')) || (value.startsWith("'") && !value.endsWith("'"))) add(number, monaco.MarkerSeverity.Error, '字符串引号未闭合。')
    documents[documents.length - 1]!.lines.push({ number, text: content, indent })
  })

  documents.forEach((document) => {
    if (!document.lines.length) return
    const topValue = (key: string) => unquote(document.lines.find((line) => line.indent === 0 && line.text.startsWith(`${key}:`))?.text.slice(key.length + 1) || '')
    const apiVersion = topValue('apiVersion')
    const kind = topValue('kind')
    const apiLine = document.lines.find((line) => line.indent === 0 && line.text.startsWith('apiVersion:'))?.number || document.start
    const kindLine = document.lines.find((line) => line.indent === 0 && line.text.startsWith('kind:'))?.number || document.start
    if (!apiVersion) add(apiLine, monaco.MarkerSeverity.Error, '缺少必填字段 apiVersion。')
    if (!kind) add(kindLine, monaco.MarkerSeverity.Error, '缺少必填字段 kind。')
    if (kind && !KIND_API_VERSIONS[kind]) add(kindLine, monaco.MarkerSeverity.Warning, `未知资源类型 ${kind}，将由集群 API Discovery 最终校验。`)
    if (kind && apiVersion && KIND_API_VERSIONS[kind] && !KIND_API_VERSIONS[kind]!.includes(apiVersion)) add(apiLine, monaco.MarkerSeverity.Error, `${kind} 应使用 apiVersion: ${KIND_API_VERSIONS[kind]!.join(' 或 ')}。`)
    const metadataIndex = document.lines.findIndex((line) => line.indent === 0 && line.text === 'metadata:')
    if (metadataIndex < 0) add(document.start, monaco.MarkerSeverity.Error, '缺少必填字段 metadata。')
    else {
      const nameLine = document.lines.slice(metadataIndex + 1).find((line) => line.indent === 2 && line.text.startsWith('name:'))
      const name = unquote(nameLine?.text.slice(5) || '')
      if (!name) add(document.lines[metadataIndex]!.number, monaco.MarkerSeverity.Error, 'metadata.name 不能为空。')
      else if (!/^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$/.test(name)) add(nameLine!.number, monaco.MarkerSeverity.Error, 'metadata.name 必须符合 Kubernetes DNS 命名规则。')
    }
    const needsSpec = !['ConfigMap', 'Secret', 'Namespace', 'ServiceAccount', 'Role', 'RoleBinding', 'ClusterRole', 'ClusterRoleBinding'].includes(kind)
    if (kind && needsSpec && !document.lines.some((line) => line.indent === 0 && line.text === 'spec:')) add(document.start, monaco.MarkerSeverity.Error, `${kind} 缺少必填字段 spec。`)
    if (['Pod', 'Deployment', 'StatefulSet', 'DaemonSet', 'Job', 'CronJob'].includes(kind) && !document.lines.some((line) => /^containers:\s*$/.test(line.text))) add(document.start, monaco.MarkerSeverity.Error, `${kind} 必须定义 containers 列表。`)
    if (['Deployment', 'StatefulSet', 'DaemonSet'].includes(kind) && !document.lines.some((line) => /^selector:\s*$/.test(line.text))) add(document.start, monaco.MarkerSeverity.Error, `${kind} 必须定义 spec.selector。`)

    const seen = new Map<string, number>()
    document.lines.forEach((line) => {
      const keyMatch = line.text.match(/^-?\s*([A-Za-z][\w.-]*):/)
      if (keyMatch) {
        const scopeKey = `${line.indent}:${keyMatch[1]}`
        if (seen.has(scopeKey) && line.indent <= 2) add(line.number, monaco.MarkerSeverity.Warning, `同一层级重复字段 ${keyMatch[1]}。`)
        seen.set(scopeKey, line.number)
      }
      const valueOf = (key: string) => line.text.startsWith(`${key}:`) ? unquote(line.text.slice(key.length + 1)) : undefined
      const restartPolicy = valueOf('restartPolicy')
      if (restartPolicy && !['Always', 'OnFailure', 'Never'].includes(restartPolicy)) add(line.number, monaco.MarkerSeverity.Error, 'restartPolicy 只能是 Always、OnFailure 或 Never。')
      const concurrency = valueOf('concurrencyPolicy')
      if (concurrency && !['Allow', 'Forbid', 'Replace'].includes(concurrency)) add(line.number, monaco.MarkerSeverity.Error, 'concurrencyPolicy 只能是 Allow、Forbid 或 Replace。')
      const pathType = valueOf('pathType')
      if (pathType && !['Exact', 'Prefix', 'ImplementationSpecific'].includes(pathType)) add(line.number, monaco.MarkerSeverity.Error, 'pathType 值无效。')
      const containers = valueOf('containers') ?? valueOf('initContainers')
      if (containers && containers !== '[]') add(line.number, monaco.MarkerSeverity.Error, 'containers 必须是 YAML 数组；请换行并使用 `- name:`。')
      const replicas = valueOf('replicas')
      if (replicas && !/^\d+$/.test(replicas)) add(line.number, monaco.MarkerSeverity.Error, 'replicas 必须是非负整数。')
      const schedule = valueOf('schedule')
      if (kind === 'CronJob' && schedule && schedule.split(/\s+/).length !== 5) add(line.number, monaco.MarkerSeverity.Error, 'CronJob schedule 必须包含 5 个 cron 字段。')
    })
  })

  monaco.editor.setModelMarkers(model as any, 'kubernetes-yaml', markers)
  return {
    errors: markers.filter((marker) => marker.severity === monaco.MarkerSeverity.Error).length,
    warnings: markers.filter((marker) => marker.severity === monaco.MarkerSeverity.Warning).length,
  }
}
