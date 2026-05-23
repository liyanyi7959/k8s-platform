import type { K8sLikeObject, ResourceKey, SortOrder, TreeNode } from './ClusterManageView.types'

export function filterTreeByPerms(nodes: TreeNode[], perms: string[]): TreeNode[] {
  const hasPerm = (need?: string | string[]) => {
    if (!need) return true
    if (Array.isArray(need)) return need.some((it) => perms.includes(it))
    return perms.includes(need)
  }

  const visit = (items: TreeNode[]): TreeNode[] => {
    return items.flatMap((item) => {
      if (item.kind === 'view') {
        return hasPerm(item.perm) ? [item] : []
      }
      const children = visit(item.children ?? [])
      if (children.length === 0) return []
      return [{ ...item, children }]
    })
  }

  return visit(nodes)
}

export function filterTreeByResourceSupport(nodes: TreeNode[], support: Partial<Record<ResourceKey, boolean>>): TreeNode[] {
  const visit = (items: TreeNode[]): TreeNode[] => {
    return items.flatMap((item) => {
      if (item.kind === 'view') {
        if (item.resource && support[item.resource] === false) {
          if (item.resource === 'podmetrics') return [item]
          return []
        }
        return [item]
      }
      const children = visit(item.children ?? [])
      if (children.length === 0) return []
      return [{ ...item, children }]
    })
  }

  return visit(nodes)
}

// ─── 共用类型 ──────────────────────────────────────────
export type ControllerRow = { kind: string; namespace: string; name: string }
export type EventRow = { type: string; reason: string; message: string; count: number; lastSeen: string }
export type ContainerResourceRow = {
  name: string
  cpuRequests: string
  cpuLimits: string
  memRequests: string
  memLimits: string
  ephemeralRequests: string
  ephemeralLimits: string
}
export type ContainerVm = {
  key: string
  displayName: string
  image: string
  imagePullPolicy: string
  commandText: string
  argsText: string
  portsText: string
  resources: any
}
export type RelatedRow = { group: string; kind: string; name: string; summary: string; action: string; raw?: any }
export type PvListRowVm = { name: string; phase: string; storageClass: string; capacity: string; reclaim: string }
export type PvcListRowVm = { namespace: string; name: string; phase: string; storageClass: string; volumeName: string }

// ─── 通用辅助函数 ──────────────────────────────────────
export function normalizeMultilineText(input: string): string {
  let text = String(input ?? '')
  if (!text) return ''
  text = text.replace(/\r\n/g, '\n')
  const quoted = (text.startsWith('"') && text.endsWith('"')) || (text.startsWith("'") && text.endsWith("'"))
  if (quoted && text.includes('\\n')) {
    text = text.slice(1, -1)
  }
  const hasRealNewline = text.includes('\n')
  const hasEscapedNewline = text.includes('\\n')
  if (!hasRealNewline && hasEscapedNewline) {
    text = text.replace(/\\r\\n/g, '\n').replace(/\\n/g, '\n').replace(/\\t/g, '\t')
  }
  return text
}

export function formatTs(ts: any): string {
  if (!ts) return '-'
  const t = new Date(String(ts)).getTime()
  if (!Number.isFinite(t)) return '-'
  return new Date(t).toLocaleString()
}

export function asIntText(v: any): string {
  if (v == null) return '-'
  const n = Number(v)
  if (!Number.isFinite(n)) return '-'
  return String(Math.trunc(n))
}

export function ownersListToText(owners: any): string {
  const list: any[] = Array.isArray(owners) ? owners : []
  const parts = list
    .map((o) => {
      const kind = String(o?.kind ?? '').trim()
      const name = String(o?.name ?? '').trim()
      if (!kind || !name) return ''
      return `${kind}/${name}`
    })
    .filter(Boolean)
  return parts.length ? parts.join(', ') : '-'
}

export function getHttpStatus(e: unknown): number | null {
  const s = Number((e as any)?.response?.status ?? (e as any)?.data?.http_status)
  return Number.isFinite(s) ? s : null
}

export function normalizeLabelRecord(v: any): Record<string, string> {
  if (!v || typeof v !== 'object' || Array.isArray(v)) return {}
  const out: Record<string, string> = {}
  for (const [k, val] of Object.entries(v as Record<string, any>)) {
    const key = String(k ?? '').trim()
    const value = val != null ? String(val).trim() : ''
    if (!key || !value) continue
    out[key] = value
  }
  return out
}

export function matchLabels(labels: any, required: Record<string, string>): boolean {
  const req = required ?? {}
  const entries = Object.entries(req)
  if (entries.length === 0) return false
  const lbl = normalizeLabelRecord(labels)
  for (const [k, v] of entries) {
    if (lbl[k] !== v) return false
  }
  return true
}

export function ingressUsesService(ingress: any, serviceName: string): boolean {
  const name = String(serviceName ?? '').trim()
  if (!name) return false
  const spec = ingress?.spec ?? {}

  const defaultBackend = spec?.defaultBackend ?? spec?.backend ?? null
  const defaultSvc =
    String(defaultBackend?.service?.name ?? '').trim() ||
    String(defaultBackend?.serviceName ?? '').trim() ||
    String(defaultBackend?.backend?.serviceName ?? '').trim()
  if (defaultSvc && defaultSvc === name) return true

  const rules: any[] = Array.isArray(spec?.rules) ? spec.rules : []
  for (const r of rules) {
    const paths: any[] = Array.isArray(r?.http?.paths) ? r.http.paths : []
    for (const p of paths) {
      const backend = p?.backend ?? {}
      const svc =
        String(backend?.service?.name ?? '').trim() ||
        String(backend?.serviceName ?? '').trim() ||
        String(backend?.backend?.serviceName ?? '').trim()
      if (svc && svc === name) return true
    }
  }
  return false
}

export function collectIngressServiceNames(ingress: any): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  const spec = ingress?.spec ?? {}
  const rules: any[] = Array.isArray(spec?.rules) ? spec.rules : []
  for (const r of rules) {
    const paths: any[] = Array.isArray(r?.http?.paths) ? r.http.paths : []
    for (const p of paths) {
      const backend = p?.backend ?? {}
      const svc =
        String(backend?.service?.name ?? '').trim() ||
        String(backend?.serviceName ?? '').trim() ||
        String(backend?.backend?.serviceName ?? '').trim()
      if (!svc || seen.has(svc)) continue
      seen.add(svc)
      out.push(svc)
    }
  }
  const defaultBackend = spec?.defaultBackend ?? spec?.backend ?? null
  const defaultSvc =
    String(defaultBackend?.service?.name ?? '').trim() ||
    String(defaultBackend?.serviceName ?? '').trim() ||
    String(defaultBackend?.backend?.serviceName ?? '').trim()
  if (defaultSvc && !seen.has(defaultSvc)) out.push(defaultSvc)
  out.sort((a, b) => a.localeCompare(b))
  return out
}

export function podUsesConfigMap(pod: any, cmName: string): boolean {
  const name = String(cmName ?? '').trim()
  if (!name) return false
  const spec = pod?.spec ?? {}
  const vols: any[] = Array.isArray(spec?.volumes) ? spec.volumes : []
  for (const v of vols) {
    const cm = String(v?.configMap?.name ?? '').trim()
    if (cm && cm === name) return true
    const srcs: any[] = Array.isArray(v?.projected?.sources) ? v.projected.sources : []
    for (const s of srcs) {
      const pCm = String(s?.configMap?.name ?? '').trim()
      if (pCm && pCm === name) return true
    }
  }
  const containers: any[] = Array.isArray(spec?.containers) ? spec.containers : []
  const initContainers: any[] = Array.isArray(spec?.initContainers) ? spec.initContainers : []
  const ephemeralContainers: any[] = Array.isArray(spec?.ephemeralContainers) ? spec.ephemeralContainers : []
  for (const c of [...initContainers, ...containers, ...ephemeralContainers]) {
    const envFrom: any[] = Array.isArray(c?.envFrom) ? c.envFrom : []
    for (const ef of envFrom) {
      const cm = String(ef?.configMapRef?.name ?? '').trim()
      if (cm && cm === name) return true
    }
    const env: any[] = Array.isArray(c?.env) ? c.env : []
    for (const e of env) {
      const cm = String(e?.valueFrom?.configMapKeyRef?.name ?? '').trim()
      if (cm && cm === name) return true
    }
  }
  return false
}

export function podUsesSecret(pod: any, secretName: string): boolean {
  const name = String(secretName ?? '').trim()
  if (!name) return false
  const spec = pod?.spec ?? {}
  const vols: any[] = Array.isArray(spec?.volumes) ? spec.volumes : []
  for (const v of vols) {
    const sec = String(v?.secret?.secretName ?? '').trim()
    if (sec && sec === name) return true
    const srcs: any[] = Array.isArray(v?.projected?.sources) ? v.projected.sources : []
    for (const s of srcs) {
      const pSec = String(s?.secret?.name ?? '').trim()
      if (pSec && pSec === name) return true
    }
  }
  const pulls: any[] = Array.isArray(spec?.imagePullSecrets) ? spec.imagePullSecrets : []
  for (const p of pulls) {
    const sec = String(p?.name ?? '').trim()
    if (sec && sec === name) return true
  }
  const containers: any[] = Array.isArray(spec?.containers) ? spec.containers : []
  const initContainers: any[] = Array.isArray(spec?.initContainers) ? spec.initContainers : []
  const ephemeralContainers: any[] = Array.isArray(spec?.ephemeralContainers) ? spec.ephemeralContainers : []
  for (const c of [...initContainers, ...containers, ...ephemeralContainers]) {
    const envFrom: any[] = Array.isArray(c?.envFrom) ? c.envFrom : []
    for (const ef of envFrom) {
      const sec = String(ef?.secretRef?.name ?? '').trim()
      if (sec && sec === name) return true
    }
    const env: any[] = Array.isArray(c?.env) ? c.env : []
    for (const e of env) {
      const sec = String(e?.valueFrom?.secretKeyRef?.name ?? '').trim()
      if (sec && sec === name) return true
    }
  }
  return false
}

export function collectControllersFromPodsRaw(pods: any[]): ControllerRow[] {
  const out: ControllerRow[] = []
  const seen = new Set<string>()
  for (const p of pods) {
    const namespace = String(p?.metadata?.namespace ?? '').trim()
    const owners: any[] = Array.isArray(p?.metadata?.ownerReferences) ? p.metadata.ownerReferences : []
    for (const o of owners) {
      const kind = String(o?.kind ?? '').trim()
      const name = String(o?.name ?? '').trim()
      if (!kind || !name) continue
      const key = `${namespace}:${kind}:${name}`
      if (seen.has(key)) continue
      seen.add(key)
      out.push({ kind, namespace, name })
    }
  }
  out.sort((a, b) =>
    a.namespace === b.namespace ? (a.kind === b.kind ? a.name.localeCompare(b.name) : a.kind.localeCompare(b.kind)) : a.namespace.localeCompare(b.namespace)
  )
  return out
}

export function mergeControllers(...groups: Array<ControllerRow[] | undefined | null>): ControllerRow[] {
  const out: ControllerRow[] = []
  const seen = new Set<string>()
  for (const g of groups) {
    const list: ControllerRow[] = Array.isArray(g) ? g : []
    for (const it of list) {
      const kind = String(it?.kind ?? '').trim()
      const namespace = String(it?.namespace ?? '').trim()
      const name = String(it?.name ?? '').trim()
      if (!kind || !name) continue
      const key = `${namespace}:${kind}:${name}`
      if (seen.has(key)) continue
      seen.add(key)
      out.push({ kind, namespace, name })
    }
  }
  out.sort((a, b) =>
    a.namespace === b.namespace ? (a.kind === b.kind ? a.name.localeCompare(b.name) : a.kind.localeCompare(b.kind)) : a.namespace.localeCompare(b.namespace)
  )
  return out
}

export function extractPodSpecFromWorkload(workload: any): any | null {
  const kind = String(workload?.kind ?? '').trim()
  const spec = workload?.spec ?? {}
  if (spec?.template?.spec) return spec.template.spec
  if (kind === 'Job' && spec?.template?.spec) return spec.template.spec
  if (kind === 'CronJob' && spec?.jobTemplate?.spec?.template?.spec) return spec.jobTemplate.spec.template.spec
  return null
}

export function workloadUsesConfigMap(workload: any, cmName: string): boolean {
  const podSpec = extractPodSpecFromWorkload(workload)
  if (!podSpec) return false
  return podUsesConfigMap({ spec: podSpec }, cmName)
}

export function workloadUsesSecret(workload: any, secretName: string): boolean {
  const podSpec = extractPodSpecFromWorkload(workload)
  if (!podSpec) return false
  return podUsesSecret({ spec: podSpec }, secretName)
}

export function ingressUsesSecret(ingress: any, secretName: string): boolean {
  const name = String(secretName ?? '').trim()
  if (!name) return false
  const tls: any[] = Array.isArray(ingress?.spec?.tls) ? ingress.spec.tls : []
  return tls.some((t) => String(t?.secretName ?? '').trim() === name)
}

export function tryPrettyJson(text: string): { text: string; ok: boolean } {
  const raw = normalizeMultilineText(text)
  const trimmed = raw.trim()
  if (!trimmed) return { text: '', ok: false }
  const first = trimmed[0]
  if (first !== '{' && first !== '[') return { text: raw, ok: false }
  try {
    const v = JSON.parse(trimmed)
    return { text: JSON.stringify(v, null, 2), ok: true }
  } catch {
    return { text: raw, ok: false }
  }
}

export function decodeBase64Utf8(input: string): string {
  const b64 = String(input ?? '').trim()
  if (!b64) return ''
  const binary = atob(b64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i)
  if (typeof TextDecoder !== 'undefined') {
    return new TextDecoder('utf-8', { fatal: false }).decode(bytes)
  }
  let out = ''
  for (let i = 0; i < bytes.length; i += 1) out += String.fromCharCode(bytes[i])
  return out
}

export function getEventTimeMs(ev: any): number | null {
  const ts =
    ev?.lastTimestamp ??
    ev?.eventTime ??
    ev?.firstTimestamp ??
    ev?.deprecatedLastTimestamp ??
    ev?.deprecatedFirstTimestamp ??
    ev?.metadata?.creationTimestamp
  if (!ts) return null
  const t = new Date(String(ts)).getTime()
  return Number.isFinite(t) ? t : null
}

export function formatPorts(ports: any[]): string {
  return ports
    .map((p) => {
      const port = Number(p?.port ?? 0)
      const targetPort = p?.targetPort != null ? String(p.targetPort) : ''
      const protocol = String(p?.protocol ?? 'TCP')
      const name = p?.name ? String(p.name) : ''
      const suffix = targetPort ? `->${targetPort}` : ''
      return `${name ? `${name}:` : ''}${port}${suffix}/${protocol}`
    })
    .join(', ')
}

export function formatSelector(sel: Record<string, string>): string {
  const entries = Object.entries(sel ?? {})
  if (entries.length === 0) return '-'
  return entries.map(([k, v]) => `${k}=${v}`).join(', ')
}

export function getHosts(row: any): string[] {
  const rules: any[] = row?.spec?.rules ?? []
  return rules.map((r) => String(r?.host ?? '')).filter((h) => h)
}

export function formatRules(row: any): string {
  const rules: any[] = row?.spec?.rules ?? []
  const parts: string[] = []
  for (const r of rules) {
    const host = String(r?.host ?? '')
    const paths: any[] = r?.http?.paths ?? []
    for (const p of paths) {
      const path = String(p?.path ?? '/')
      const svc = p?.backend?.service?.name ? String(p.backend.service.name) : '-'
      const port = p?.backend?.service?.port?.number != null ? String(p.backend.service.port.number) : '-'
      parts.push(`${host || '*'} ${path} -> ${svc}:${port}`)
    }
  }
  return parts.join(' | ') || '-'
}

export function podUsesPvc(pod: any, claimName: string): boolean {
  const target = String(claimName ?? '').trim()
  if (!target) return false
  const vols: any[] = Array.isArray(pod?.spec?.volumes) ? pod.spec.volumes : []
  for (const v of vols) {
    const pvc = v?.persistentVolumeClaim
    if (!pvc) continue
    const n = String(pvc?.claimName ?? '').trim()
    if (n && n === target) return true
  }
  return false
}

export function formatPvcClaimRefText(row: any): string {
  const ns = String(row?.spec?.claimRef?.namespace ?? '').trim()
  const name = String(row?.spec?.claimRef?.name ?? '').trim()
  if (!ns || !name) return '-'
  return `${ns}/${name}`
}

export function uniq(arr: string[]): string[] {
  const out: string[] = []
  const set = new Set<string>()
  for (const a of arr) {
    const s = String(a ?? '').trim()
    if (!s || set.has(s)) continue
    set.add(s)
    out.push(s)
  }
  return out
}

export function collectTemplateConfigMapsSecrets(spec: any | null): { configMaps: string[]; secrets: string[] } {
  const cms: string[] = []
  const secs: string[] = []
  const vols: any[] = Array.isArray(spec?.volumes) ? spec.volumes : []
  for (const v of vols) {
    const cm = String(v?.configMap?.name ?? '').trim()
    if (cm) cms.push(cm)
    const sec = String(v?.secret?.secretName ?? '').trim()
    if (sec) secs.push(sec)
  }
  const containers: any[] = Array.isArray(spec?.containers) ? spec.containers : []
  const initContainers: any[] = Array.isArray(spec?.initContainers) ? spec.initContainers : []
  for (const c of [...initContainers, ...containers]) {
    const envFrom: any[] = Array.isArray(c?.envFrom) ? c.envFrom : []
    for (const ef of envFrom) {
      const cm = String(ef?.configMapRef?.name ?? '').trim()
      if (cm) cms.push(cm)
      const sec = String(ef?.secretRef?.name ?? '').trim()
      if (sec) secs.push(sec)
    }
    const env: any[] = Array.isArray(c?.env) ? c.env : []
    for (const e of env) {
      const cm = String(e?.valueFrom?.configMapKeyRef?.name ?? '').trim()
      if (cm) cms.push(cm)
      const sec = String(e?.valueFrom?.secretKeyRef?.name ?? '').trim()
      if (sec) secs.push(sec)
    }
  }
  return { configMaps: uniq(cms), secrets: uniq(secs) }
}

export function fmtListText(v: any): string {
  if (!Array.isArray(v)) return ''
  const parts = v.map((it) => String(it ?? '').trim()).filter(Boolean)
  return parts.join(' ')
}

export function fmtPortsText(ports: any): string {
  const arr: any[] = Array.isArray(ports) ? ports : []
  if (!arr.length) return ''
  return arr
    .map((p) => {
      const name = String(p?.name ?? '').trim()
      const port = p?.containerPort
      const proto = String(p?.protocol ?? 'TCP')
      if (!port) return ''
      return name ? `${name}=${port}/${proto}` : `${port}/${proto}`
    })
    .filter(Boolean)
    .join(', ')
}

export function getResVal(obj: any, key: string): string {
  if (!obj || typeof obj !== 'object') return '-'
  const v = obj[key]
  const s = String(v ?? '').trim()
  return s ? s : '-'
}

export function buildContainerVms(spec: any | null): {
  options: Array<{ key: string; label: string }>
  map: Map<string, ContainerVm>
  rows: ContainerResourceRow[]
} {
  const containers: any[] = Array.isArray(spec?.containers) ? spec.containers : []
  const initContainers: any[] = Array.isArray(spec?.initContainers) ? spec.initContainers : []
  const items = [
    ...initContainers.map((c, idx) => ({ kind: 'init' as const, idx, c })),
    ...containers.map((c, idx) => ({ kind: 'main' as const, idx, c }))
  ]
  const options: Array<{ key: string; label: string }> = []
  const map = new Map<string, ContainerVm>()
  const rows: ContainerResourceRow[] = []
  for (const it of items) {
    const name = String(it?.c?.name ?? '').trim()
    const key = `${it.kind}:${it.idx}:${name || 'container'}`
    const label = it.kind === 'init' ? `init/${name || it.idx}` : name || `container-${it.idx}`
    options.push({ key, label })
    map.set(key, {
      key,
      displayName: label,
      image: String(it?.c?.image ?? '').trim() || '-',
      imagePullPolicy: String(it?.c?.imagePullPolicy ?? '').trim() || '-',
      commandText: fmtListText(it?.c?.command) || '',
      argsText: fmtListText(it?.c?.args) || '',
      portsText: fmtPortsText(it?.c?.ports) || '',
      resources: it?.c?.resources ?? {}
    })
    const req = it?.c?.resources?.requests ?? {}
    const lim = it?.c?.resources?.limits ?? {}
    rows.push({
      name: label,
      cpuRequests: getResVal(req, 'cpu'),
      cpuLimits: getResVal(lim, 'cpu'),
      memRequests: getResVal(req, 'memory'),
      memLimits: getResVal(lim, 'memory'),
      ephemeralRequests: getResVal(req, 'ephemeral-storage'),
      ephemeralLimits: getResVal(lim, 'ephemeral-storage')
    })
  }
  return { options, map, rows }
}

export function jobPodTemplateSpec(row: any): any | null {
  const t = row?.spec?.template?.spec
  return t && typeof t === 'object' ? t : null
}

export function cronJobPodTemplateSpec(row: any): any | null {
  const t = row?.spec?.jobTemplate?.spec?.template?.spec
  return t && typeof t === 'object' ? t : null
}

export function isPodOwnedByJob(pod: any, jobName: string, jobUid?: string): boolean {
  const labels = pod?.metadata?.labels ?? {}
  if (labels && typeof labels === 'object' && String((labels as any)['job-name'] ?? '') === jobName) return true
  const owners: any[] = Array.isArray(pod?.metadata?.ownerReferences) ? pod.metadata.ownerReferences : []
  for (const o of owners) {
    const kind = String(o?.kind ?? '')
    const name = String(o?.name ?? '')
    const uid = String(o?.uid ?? '')
    if (kind === 'Job' && name === jobName) return true
    if (jobUid && uid && uid === jobUid) return true
  }
  return false
}

export function getPodReadyTextLocal(pod: any): string {
  const cs: any[] = Array.isArray(pod?.status?.containerStatuses) ? pod.status.containerStatuses : []
  if (!cs.length) return '-'
  const ready = cs.filter((it) => Boolean(it?.ready)).length
  return `${ready}/${cs.length}`
}

export function getJobStatusTextLocal(row: any): string {
  const failed = Number(row?.status?.failed ?? 0)
  const succeeded = Number(row?.status?.succeeded ?? 0)
  const active = Number(row?.status?.active ?? 0)
  if (Number.isFinite(succeeded) && succeeded > 0) return 'Succeeded'
  if (Number.isFinite(failed) && failed > 0) return 'Failed'
  if (Number.isFinite(active) && active > 0) return 'Running'
  return 'Pending'
}

export function formatJobCompletionsLocal(row: any): string {
  const succeeded = Number(row?.status?.succeeded ?? 0)
  const desired = row?.spec?.completions != null ? Number(row.spec.completions ?? 0) : null
  if (desired == null || !Number.isFinite(desired) || desired <= 0) return String(Number.isFinite(succeeded) ? succeeded : 0)
  return `${Number.isFinite(succeeded) ? succeeded : 0}/${desired}`
}

export function getPodReadyTextUtil(row: any): string {
  if (row?.ready != null) return String(row.ready)
  const css: any[] = Array.isArray(row?.status?.containerStatuses) ? row.status.containerStatuses : []
  const total = css.length
  if (total <= 0) return '-'
  const ready = css.reduce((sum: number, it: any) => sum + (it?.ready ? 1 : 0), 0)
  return `${ready}/${total}`
}

export function getPodRestartsUtil(row: any): number {
  if (row?.restarts != null) return Number(row.restarts ?? 0)
  const css: any[] = Array.isArray(row?.status?.containerStatuses) ? row.status.containerStatuses : []
  return css.reduce((sum: number, it: any) => sum + Number(it?.restartCount ?? 0), 0)
}

export function decoratePodRowUtil(row: any): any {
  if (!row || typeof row !== 'object') return row
  const restarts = getPodRestartsUtil(row)
  const ageMs = getCreationAgeMs(row)
  row.restarts = restarts
  row.ageMs = ageMs ?? undefined
  const ns = getRowNamespace(row) ?? ''
  const name = String(row?.metadata?.name ?? '')
  const phase = String(row?.status?.phase ?? '')
  const node = String(row?.spec?.nodeName ?? '')
  const podIP = String(row?.status?.podIP ?? '')
  const hostIP = String(row?.status?.hostIP ?? '')
  row.__search = `${ns} ${name} ${phase} ${node} ${podIP} ${hostIP}`.toLowerCase()
  return row
}

export function toRelatedPodVmFromPod(pod: any): any {
  const row = decoratePodRowUtil(pod)
  const ns = String(getRowNamespace(row) ?? '')
  const name = String(row?.metadata?.name ?? '')
  return {
    ...row,
    rawPod: row,
    name,
    namespace: ns,
    phase: String(row?.status?.phase ?? '-') || '-',
    ready: getPodReadyTextUtil(row),
    restarts: getPodRestartsUtil(row),
    node: String(row?.spec?.nodeName ?? '-') || '-',
    ownersText: ownersListToText(row?.metadata?.ownerReferences)
  }
}

export function buildStorageKey(prefix: string, clusterId: number, suffix: string): string {
  return `${prefix}:${clusterId}:${suffix}`
}

export function readStorageString(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export function writeStorageString(key: string, value: string | null) {
  try {
    if (value == null) localStorage.removeItem(key)
    else localStorage.setItem(key, value)
  } catch {
    return
  }
}

export function readStorageJson<T>(key: string): T | null {
  const raw = readStorageString(key)
  if (!raw) return null
  try {
    return JSON.parse(raw) as T
  } catch {
    return null
  }
}

export function writeStorageJson(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {
    return
  }
}

export function normalizeNamespaceSelection(input: unknown, allNamespaceValue: string): string[] {
  const raw = Array.isArray(input) ? input : []
  const normalized = raw.map((v) => String(v ?? '').trim()).filter(Boolean)
  if (normalized.includes(allNamespaceValue)) return [allNamespaceValue]
  if (normalized.length === 0) return [allNamespaceValue]
  return Array.from(new Set(normalized))
}

export function computeNextNamespaceSelection(
  prev: string[],
  input: unknown,
  namespaces: string[],
  allNamespaceValue: string
): string[] {
  const raw = Array.isArray(input) ? input.map((it) => String(it ?? '').trim()).filter(Boolean) : []
  const allowed = new Set<string>([allNamespaceValue, ...namespaces])
  const nextRaw = raw.filter((it) => allowed.has(it))

  const nextHasAll = nextRaw.includes(allNamespaceValue)
  const nextWithoutAll = nextRaw.filter((it) => it !== allNamespaceValue)

  const uniqueWithoutAll: string[] = []
  const seen = new Set<string>()
  for (const it of nextWithoutAll) {
    if (seen.has(it)) continue
    seen.add(it)
    uniqueWithoutAll.push(it)
  }

  if (prev.includes(allNamespaceValue) && uniqueWithoutAll.length > 0) return uniqueWithoutAll
  if (nextHasAll) return [allNamespaceValue]
  if (namespaces.length > 0 && uniqueWithoutAll.length >= namespaces.length) {
    const allSet = new Set(namespaces)
    const selectedSet = new Set(uniqueWithoutAll)
    return allSet.size === selectedSet.size && [...allSet].every((it) => selectedSet.has(it)) ? [allNamespaceValue] : uniqueWithoutAll
  }
  if (uniqueWithoutAll.length === 0) return [allNamespaceValue]
  return uniqueWithoutAll
}

export function getNamespaceFilter(selection: unknown, allNamespaceValue: string): string[] | null {
  const v = normalizeNamespaceSelection(selection, allNamespaceValue)
  if (v.includes(allNamespaceValue)) return null
  return v
}

export function getRowNamespace(row: K8sLikeObject): string | null {
  const ns = row?.metadata?.namespace
  const v = ns != null ? String(ns).trim() : ''
  return v ? v : null
}

export function getResourceBadgeClass(r: ResourceKey): string {
  if (r === 'pods' || r === 'podmetrics') return 'resource-badge--pods'
  if (r === 'workloads' || r === 'replicasets') return 'resource-badge--workloads'
  if (r === 'pdbs') return 'resource-badge--workloads'
  if (r === 'hpas') return 'resource-badge--workloads'
  if (r === 'services' || r === 'endpoints' || r === 'endpointslices' || r === 'networkpolicies') return 'resource-badge--services'
  if (r === 'ingresses' || r === 'ingressclasses') return 'resource-badge--ingress'
  if (r === 'configmaps' || r === 'secrets' || r === 'serviceaccounts' || r === 'roles' || r === 'clusterroles' || r === 'rolebindings' || r === 'clusterrolebindings' || r === 'customresourcedefinitions' || r === 'apiservices' || r === 'priorityclasses' || r === 'runtimeclasses' || r === 'validatingwebhookconfigurations' || r === 'mutatingwebhookconfigurations' || r === 'validatingadmissionpolicies' || r === 'validatingadmissionpolicybindings') return 'resource-badge--config'
  if (r === 'pvs' || r === 'pvcs' || r === 'storageclasses' || r === 'csidrivers' || r === 'csinodes' || r === 'csistoragecapacities' || r === 'volumeattachments' || r === 'volumesnapshots' || r === 'volumesnapshotclasses' || r === 'volumesnapshotcontents' || r === 'resourcequotas' || r === 'limitranges') return 'resource-badge--storage'
  if (r === 'nodes') return 'resource-badge--nodes'
  if (r === 'namespaces') return 'resource-badge--namespaces'
  if (r === 'events') return 'resource-badge--events'
  if (r === 'jobs' || r === 'cronjobs') return 'resource-badge--jobs'
  return 'resource-badge--default'
}

export function getPodRowKey(row: K8sLikeObject): string {
  const ns = getRowNamespace(row) ?? '-'
  const name = String(row?.metadata?.name ?? '-')
  return `${ns}/${name}`
}

export function getNamespacedRowKey(row: K8sLikeObject): string {
  const ns = getRowNamespace(row) ?? '-'
  const name = String(row?.metadata?.name ?? '-')
  return `${ns}/${name}`
}

export function getWorkloadDesired(row: unknown): number {
  return Number((row as any)?.spec?.replicas ?? 0)
}

export function getWorkloadAvailable(row: unknown): number {
  return Number((row as any)?.status?.availableReplicas ?? (row as any)?.status?.readyReplicas ?? 0)
}

export function getWorkloadReadyText(row: unknown): string {
  const desired = getWorkloadDesired(row)
  const ready = getWorkloadAvailable(row)
  return `${ready}/${desired}`
}

export function getWorkloadCurrentReplicas(row: unknown): number {
  const raw = row as any
  const kind = String(raw?.kind ?? '')
  if (kind === 'DaemonSet') {
    return Number(raw?.status?.currentNumberScheduled ?? raw?.status?.numberAvailable ?? 0)
  }
  return Number(raw?.status?.replicas ?? raw?.status?.currentReplicas ?? 0)
}

export function isWorkloadProgressing(row: unknown): boolean {
  const raw = row as any
  if (!raw || typeof raw !== 'object') return false

  const kind = String(raw?.kind ?? '')
  const desired = Math.max(0, getWorkloadDesired(raw))
  const available = Math.max(0, getWorkloadAvailable(raw))
  const current = Math.max(0, getWorkloadCurrentReplicas(raw))
  const updated = Math.max(0, Number(raw?.status?.updatedReplicas ?? 0))
  const unavailable = Math.max(0, Number(raw?.status?.unavailableReplicas ?? 0))
  const observedGeneration = Math.max(0, Number(raw?.status?.observedGeneration ?? 0))
  const generation = Math.max(0, Number(raw?.metadata?.generation ?? 0))

  if (kind === 'DaemonSet') {
    const desiredScheduled = Math.max(0, Number(raw?.status?.desiredNumberScheduled ?? 0))
    const updatedScheduled = Math.max(0, Number(raw?.status?.updatedNumberScheduled ?? current))
    const readyScheduled = Math.max(0, Number(raw?.status?.numberReady ?? available))
    return observedGeneration < generation || updatedScheduled < desiredScheduled || readyScheduled < desiredScheduled
  }

  return observedGeneration < generation || current > desired || updated < desired || available < desired || unavailable > 0
}

export function getWorkloadProgressText(row: unknown): string {
  const raw = row as any
  if (!raw || typeof raw !== 'object') return ''
  if (!isWorkloadProgressing(raw)) return ''

  const kind = String(raw?.kind ?? '')
  const desired = Math.max(0, getWorkloadDesired(raw))
  const available = Math.max(0, getWorkloadAvailable(raw))
  const current = Math.max(0, getWorkloadCurrentReplicas(raw))
  const updated = Math.max(0, Number(raw?.status?.updatedReplicas ?? 0))

  if (kind === 'DaemonSet') {
    const desiredScheduled = Math.max(0, Number(raw?.status?.desiredNumberScheduled ?? 0))
    const readyScheduled = Math.max(0, Number(raw?.status?.numberReady ?? 0))
    return `滚动中 ${readyScheduled}/${desiredScheduled}`
  }
  if (current > desired && desired > 0) return `滚动中 ${current}/${desired} Pods`
  if (updated < desired && desired > 0) return `更新中 ${updated}/${desired}`
  if (available < desired && desired > 0) return `可用 ${available}/${desired}`
  return '同步中'
}

export function getListRowSearchText(row: K8sLikeObject, resource: ResourceKey | undefined): string {
  if (!row || typeof row !== 'object') return ''
  const meta = row?.metadata ?? {}
  const ns = meta?.namespace != null ? String(meta.namespace) : ''
  const name = meta?.name != null ? String(meta.name) : ''
  const kind = row?.kind != null ? String(row.kind) : ''

  const status = (row as any)?.status
  const spec = (row as any)?.spec
  const phase = status?.phase != null ? String(status.phase) : ''
  const node = spec?.nodeName != null ? String(spec.nodeName) : ''
  const podIP = status?.podIP != null ? String(status.podIP) : ''
  const hostIP = status?.hostIP != null ? String(status.hostIP) : ''

  if (resource === 'pods') return `${ns} ${name} ${phase} ${node} ${podIP} ${hostIP}`.toLowerCase()
  if (resource === 'podmetrics') return String((row as any)?.__search ?? `${ns} ${name}`).toLowerCase()
  if (resource === 'workloads') return `${kind} ${ns} ${name}`.toLowerCase()
  if (resource === 'replicasets') return `${ns} ${name} ${status?.replicas ?? ''} ${status?.readyReplicas ?? ''}`.toLowerCase()
  if (resource === 'jobs') {
    const active = status?.active != null ? String(status.active) : ''
    const succeeded = status?.succeeded != null ? String(status.succeeded) : ''
    const failed = status?.failed != null ? String(status.failed) : ''
    const completions = spec?.completions != null ? String(spec.completions) : ''
    return `${ns} ${name} ${active} ${succeeded} ${failed} ${completions}`.toLowerCase()
  }
  if (resource === 'cronjobs') {
    const schedule = spec?.schedule != null ? String(spec.schedule) : ''
    const suspend = spec?.suspend != null ? String(spec.suspend) : ''
    const concurrency = spec?.concurrencyPolicy != null ? String(spec.concurrencyPolicy) : ''
    const last = status?.lastScheduleTime != null ? String(status.lastScheduleTime) : ''
    return `${ns} ${name} ${schedule} ${suspend} ${concurrency} ${last}`.toLowerCase()
  }
  if (resource === 'pdbs') {
    const min = spec?.minAvailable != null ? String(spec.minAvailable) : ''
    const max = spec?.maxUnavailable != null ? String(spec.maxUnavailable) : ''
    const allowed = status?.disruptionsAllowed != null ? String(status.disruptionsAllowed) : ''
    return `${ns} ${name} ${min} ${max} ${allowed}`.toLowerCase()
  }
  if (resource === 'services') return `${ns} ${name} ${spec?.type ?? ''} ${spec?.clusterIP ?? ''}`.toLowerCase()
  if (resource === 'endpoints') {
    const subsets: any[] = Array.isArray((row as any)?.subsets) ? (row as any).subsets : []
    const addresses = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.addresses) ? subset.addresses.length : 0), 0)
    const ports = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.ports) ? subset.ports.length : 0), 0)
    return `${ns} ${name} ${addresses} ${ports}`.toLowerCase()
  }
  if (resource === 'endpointslices') {
    const endpoints: any[] = Array.isArray((row as any)?.endpoints) ? (row as any).endpoints : []
    const ports: any[] = Array.isArray((row as any)?.ports) ? (row as any).ports : []
    const serviceName = (row as any)?.metadata?.labels?.['kubernetes.io/service-name'] ?? ''
    return `${ns} ${name} ${serviceName} ${(row as any)?.addressType ?? ''} ${endpoints.length} ${ports.length}`.toLowerCase()
  }
  if (resource === 'networkpolicies') return `${ns} ${name} ${Array.isArray(spec?.policyTypes) ? spec.policyTypes.join(' ') : ''}`.toLowerCase()
  if (resource === 'ingresses') return `${ns} ${name} ${spec?.ingressClassName ?? ''}`.toLowerCase()
  if (resource === 'resourcequotas') return `${ns} ${name} ${Object.keys((status?.hard ?? {}) as Record<string, unknown>).join(' ')}`.toLowerCase()
  if (resource === 'limitranges') return `${ns} ${name} ${Array.isArray((spec?.limits ?? [])) ? spec.limits.length : 0}`.toLowerCase()
  if (resource === 'configmaps' || resource === 'secrets' || resource === 'serviceaccounts') return `${ns} ${name}`.toLowerCase()
  if (resource === 'roles') {
    const rules = Array.isArray((row as any)?.rules) ? (row as any).rules.length : 0
    return `${ns} ${name} ${rules}`.toLowerCase()
  }
  if (resource === 'clusterroles') {
    const rules = Array.isArray((row as any)?.rules) ? (row as any).rules.length : 0
    return `${name} ${rules}`.toLowerCase()
  }
  if (resource === 'rolebindings') {
    const roleRefKind = (row as any)?.roleRef?.kind != null ? String((row as any).roleRef.kind) : ''
    const roleRefName = (row as any)?.roleRef?.name != null ? String((row as any).roleRef.name) : ''
    return `${ns} ${name} ${roleRefKind} ${roleRefName}`.toLowerCase()
  }
  if (resource === 'clusterrolebindings') {
    const roleRefKind = (row as any)?.roleRef?.kind != null ? String((row as any).roleRef.kind) : ''
    const roleRefName = (row as any)?.roleRef?.name != null ? String((row as any).roleRef.name) : ''
    return `${name} ${roleRefKind} ${roleRefName}`.toLowerCase()
  }
  if (resource === 'customresourcedefinitions') {
    const group = spec?.group != null ? String(spec.group) : ''
    const plural = spec?.names?.plural != null ? String(spec.names.plural) : ''
    return `${name} ${group} ${plural}`.toLowerCase()
  }
  if (resource === 'apiservices') {
    const group = spec?.group != null ? String(spec.group) : ''
    const version = spec?.version != null ? String(spec.version) : ''
    const serviceName = spec?.service?.name != null ? String(spec.service.name) : ''
    return `${name} ${group} ${version} ${serviceName}`.toLowerCase()
  }
  if (resource === 'priorityclasses') {
    return `${name} ${(row as any)?.value ?? ''} ${Boolean((row as any)?.globalDefault)}`.toLowerCase()
  }
  if (resource === 'runtimeclasses') return `${name} ${String((row as any)?.handler ?? '')}`.toLowerCase()
  if (resource === 'csidrivers') return `${name} ${Boolean((row as any)?.spec?.attachRequired)} ${Boolean((row as any)?.spec?.podInfoOnMount)}`.toLowerCase()
  if (resource === 'csinodes') {
    const drivers: any[] = Array.isArray((row as any)?.spec?.drivers) ? (row as any).spec.drivers : []
    return `${name} ${drivers.length}`.toLowerCase()
  }
  if (resource === 'csistoragecapacities') {
    const storageClass = String((row as any)?.storageClassName ?? '')
    const capacity = String((row as any)?.capacity ?? '')
    return `${ns} ${name} ${storageClass} ${capacity}`.toLowerCase()
  }
  if (resource === 'volumeattachments') {
    const nodeName = spec?.nodeName != null ? String(spec.nodeName) : ''
    const pvName = spec?.source?.persistentVolumeName != null ? String(spec.source.persistentVolumeName) : ''
    return `${name} ${nodeName} ${pvName}`.toLowerCase()
  }
  if (resource === 'validatingwebhookconfigurations' || resource === 'mutatingwebhookconfigurations') {
    const count = Array.isArray((row as any)?.webhooks) ? (row as any).webhooks.length : 0
    return `${name} ${count}`.toLowerCase()
  }
  if (resource === 'validatingadmissionpolicies') {
    const policy = String((row as any)?.spec?.failurePolicy ?? '')
    return `${name} ${policy}`.toLowerCase()
  }
  if (resource === 'validatingadmissionpolicybindings') {
    const policyName = String((row as any)?.spec?.policyName ?? '')
    return `${name} ${policyName}`.toLowerCase()
  }
  if (resource === 'hpas') {
    const min = spec?.minReplicas != null ? String(spec.minReplicas) : ''
    const max = spec?.maxReplicas != null ? String(spec.maxReplicas) : ''
    const targetKind = spec?.scaleTargetRef?.kind != null ? String(spec.scaleTargetRef.kind) : ''
    const targetName = spec?.scaleTargetRef?.name != null ? String(spec.scaleTargetRef.name) : ''
    return `${ns} ${name} ${min} ${max} ${targetKind} ${targetName}`.toLowerCase()
  }
  if (resource === 'pvcs') return `${ns} ${name} ${status?.phase ?? ''} ${spec?.volumeName ?? ''}`.toLowerCase()
  if (resource === 'volumesnapshots') {
    const ready = status?.readyToUse != null ? String(status.readyToUse) : ''
    const pvc = spec?.source?.persistentVolumeClaimName != null ? String(spec.source.persistentVolumeClaimName) : ''
    const content = status?.boundVolumeSnapshotContentName != null ? String(status.boundVolumeSnapshotContentName) : ''
    const snapshotClass = spec?.volumeSnapshotClassName != null ? String(spec.volumeSnapshotClassName) : ''
    return `${ns} ${name} ${ready} ${pvc} ${content} ${snapshotClass}`.toLowerCase()
  }
  if (resource === 'leases') {
    const holder = spec?.holderIdentity ?? ''
    const renew = spec?.renewTime ?? ''
    return `${ns} ${name} ${holder} ${renew}`.toLowerCase()
  }
  if (resource === 'pvs') return `${name} ${status?.phase ?? ''} ${spec?.storageClassName ?? ''}`.toLowerCase()
  if (resource === 'volumesnapshotclasses') {
    const driver = (row as any)?.driver != null ? String((row as any).driver) : ''
    const deletionPolicy = (row as any)?.deletionPolicy != null ? String((row as any).deletionPolicy) : ''
    return `${name} ${driver} ${deletionPolicy}`.toLowerCase()
  }
  if (resource === 'volumesnapshotcontents') {
    const driver = spec?.driver != null ? String(spec.driver) : ''
    const deletionPolicy = spec?.deletionPolicy != null ? String(spec.deletionPolicy) : ''
    const snapshotHandle = spec?.source?.snapshotHandle != null ? String(spec.source.snapshotHandle) : ''
    const snapshotRef = spec?.volumeSnapshotRef?.name != null ? String(spec.volumeSnapshotRef.name) : ''
    return `${name} ${driver} ${deletionPolicy} ${snapshotHandle} ${snapshotRef}`.toLowerCase()
  }
  if (resource === 'nodes') return `${name} ${status?.nodeInfo?.kubeletVersion ?? ''} ${status?.nodeInfo?.osImage ?? ''}`.toLowerCase()
  if (resource === 'events') return `${(row as any)?.type ?? ''} ${(row as any)?.reason ?? ''} ${(row as any)?.message ?? ''}`.toLowerCase()
  return `${kind} ${ns} ${name}`.toLowerCase()
}

export function isNamespacedResource(r: ResourceKey): boolean {
  return (
    r === 'workloads' ||
    r === 'pdbs' ||
    r === 'hpas' ||
    r === 'pods' ||
      r === 'podmetrics' ||
      r === 'replicasets' ||
      r === 'jobs' ||
      r === 'cronjobs' ||
      r === 'services' ||
      r === 'endpoints' ||
      r === 'endpointslices' ||
      r === 'networkpolicies' ||
      r === 'ingresses' ||
      r === 'configmaps' ||
      r === 'secrets' ||
      r === 'serviceaccounts' ||
      r === 'roles' ||
      r === 'rolebindings' ||
      r === 'volumesnapshots' ||
      r === 'leases' ||
      r === 'csistoragecapacities' ||
      r === 'resourcequotas' ||
      r === 'limitranges' ||
      r === 'pvcs' ||
      r === 'events'
  )
}

export function getByPathValue(obj: unknown, path: string): unknown {
  const parts = String(path || '')
    .split('.')
    .map((p) => p.trim())
    .filter(Boolean)
  let cur: any = obj
  for (const p of parts) cur = cur?.[p]
  return cur
}

export function sortItemsByPath<T>(items: T[], prop: string | undefined, dir: SortOrder | undefined): T[] {
  if (!prop || !dir) return items
  const factor = dir === 'asc' ? 1 : -1
  return items.slice().sort((a: any, b: any) => {
    const av = getByPathValue(a, prop)
    const bv = getByPathValue(b, prop)
    if (av == null && bv == null) return 0
    if (av == null) return -1 * factor
    if (bv == null) return 1 * factor
    if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * factor
    const as = String(av)
    const bs = String(bv)
    const cmp = as.localeCompare(bs, 'zh-Hans-CN', { numeric: true, sensitivity: 'base' })
    return cmp * factor
  })
}

export function formatAgeMs(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '-'
  const sec = Math.floor(ms / 1000)
  const min = Math.floor(sec / 60)
  const hour = Math.floor(min / 60)
  const day = Math.floor(hour / 24)
  if (day > 0) return `${day}d`
  if (hour > 0) return `${hour}h`
  if (min > 0) return `${min}m`
  return `${sec}s`
}

export function getCreationAgeMs(row: K8sLikeObject): number | null {
  if (row?.ageMs != null) {
    const v = Number(row.ageMs)
    return Number.isFinite(v) ? v : null
  }
  const ts = row?.metadata?.creationTimestamp != null ? String((row as any).metadata.creationTimestamp) : ''
  if (!ts) return null
  const t = new Date(ts).getTime()
  if (!Number.isFinite(t)) return null
  return Math.max(0, Date.now() - t)
}

export function getCreationAgeText(row: K8sLikeObject): string {
  if (row?.age != null) return String((row as any).age)
  const ms = getCreationAgeMs(row)
  if (ms == null) return '-'
  return formatAgeMs(ms)
}

export function buildTree(icons: {
  k8sLogoUrl: string
  k8sIconCmUrl: string
  k8sIconCronJobUrl: string
  k8sIconDeploymentUrl: string
  k8sIconDaemonSetUrl: string
  k8sIconGroupUrl: string
  k8sIconIngressUrl: string
  k8sIconJobUrl: string
  k8sIconNodeUrl: string
  k8sIconNamespaceUrl: string
  k8sIconPodUrl: string
  k8sIconPvUrl: string
  k8sIconPvcUrl: string
  k8sIconStorageClassUrl: string
  k8sIconSecretUrl: string
  k8sIconStatefulSetUrl: string
  k8sIconServiceUrl: string
}): TreeNode[] {
  const dashboard: TreeNode = {
    id: 'group:dashboard',
    label: '仪表盘',
    kind: 'folder',
    iconUrl: icons.k8sLogoUrl,
    children: [{ id: 'dashboard:overview', label: '概览', kind: 'view', resource: 'dashboard', perm: 'k8s:read', iconUrl: icons.k8sLogoUrl }]
  }

  const operations: TreeNode = {
    id: 'group:operations',
    label: '运维工具',
    kind: 'folder',
    iconUrl: icons.k8sIconGroupUrl,
    children: [
      {
        id: 'tools:manifest-apply',
        label: 'YAML 部署',
        kind: 'view',
        resource: 'manifestapply',
        perm: 'k8s:write',
        iconUrl: icons.k8sLogoUrl
      }
    ]
  }

  const audit: TreeNode = {
    id: 'group:audit',
    label: '治理分析',
    kind: 'folder',
    iconUrl: icons.k8sIconGroupUrl,
    children: [
      { id: 'audit:permission-audits', label: '权限分析', kind: 'view', resource: 'permissionaudits', perm: 'k8s:permission_audit', iconUrl: icons.k8sIconGroupUrl },
      { id: 'audit:topology', label: '资源关系图', kind: 'view', resource: 'topology', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl }
    ]
  }

  const workloads: TreeNode = {
    id: 'group:workloads',
    label: '工作负载',
    kind: 'folder',
    iconUrl: icons.k8sIconDeploymentUrl,
    children: [
      { id: 'workloads:pods', label: 'Pods', kind: 'view', resource: 'pods', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconPodUrl },
      { id: 'workloads:podmetrics', label: 'PodMetrics', kind: 'view', resource: 'podmetrics', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconPodUrl },
      {
        id: 'workloads:deployments',
        label: 'Deployments',
        kind: 'view',
        resource: 'workloads',
        perm: 'k8s:read',
        namespaced: true,
        workloadKind: 'Deployment',
        iconUrl: icons.k8sIconDeploymentUrl
      },
      {
        id: 'workloads:statefulsets',
        label: 'StatefulSets',
        kind: 'view',
        resource: 'workloads',
        perm: 'k8s:read',
        namespaced: true,
        workloadKind: 'StatefulSet',
        iconUrl: icons.k8sIconStatefulSetUrl
      },
      {
        id: 'workloads:daemonsets',
        label: 'DaemonSets',
        kind: 'view',
        resource: 'workloads',
        perm: 'k8s:read',
        namespaced: true,
        workloadKind: 'DaemonSet',
        iconUrl: icons.k8sIconDaemonSetUrl
      },
      { id: 'workloads:replicasets', label: 'ReplicaSets', kind: 'view', resource: 'replicasets', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconGroupUrl },
      { id: 'workloads:pdbs', label: 'PDBs', kind: 'view', resource: 'pdbs', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconDeploymentUrl },
      { id: 'workloads:hpas', label: 'HPAs', kind: 'view', resource: 'hpas', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconDeploymentUrl }
    ]
  }

  const jobs: TreeNode = {
    id: 'group:jobs',
    label: '作业',
    kind: 'folder',
    iconUrl: icons.k8sIconJobUrl,
    children: [
      { id: 'jobs:jobs', label: 'Jobs', kind: 'view', resource: 'jobs', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconJobUrl },
      { id: 'jobs:cronjobs', label: 'CronJobs', kind: 'view', resource: 'cronjobs', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconCronJobUrl }
    ]
  }

  const config: TreeNode = {
    id: 'group:config',
    label: '配置文件',
    kind: 'folder',
    iconUrl: icons.k8sIconCmUrl,
    children: [
      { id: 'config:configmaps', label: 'ConfigMaps', kind: 'view', resource: 'configmaps', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconCmUrl },
      { id: 'config:secrets', label: 'Secrets', kind: 'view', resource: 'secrets', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconSecretUrl },

    ]
  }

  const auth: TreeNode = {
  id: 'group:auth',
  label: '访问控制',
  kind: 'folder',
  iconUrl: icons.k8sIconGroupUrl,
  children: [
    { id: 'config:serviceaccounts', label: 'ServiceAccounts', kind: 'view', resource: 'serviceaccounts', perm: 'k8s:rbac_read', namespaced: true, iconUrl: icons.k8sIconGroupUrl },
    { id: 'auth:roles', label: 'Roles', kind: 'view', resource: 'roles', perm: 'k8s:rbac_read', namespaced: true, iconUrl: icons.k8sIconGroupUrl },
    { id: 'auth:rolebindings', label: 'RoleBindings', kind: 'view', resource: 'rolebindings', perm: 'k8s:rbac_read', namespaced: true, iconUrl: icons.k8sIconGroupUrl },
    { id: 'auth:clusterroles', label: 'ClusterRoles', kind: 'view', resource: 'clusterroles', perm: 'k8s:rbac_read', iconUrl: icons.k8sIconGroupUrl },
    { id: 'auth:clusterrolebindings', label: 'ClusterRoleBindings', kind: 'view', resource: 'clusterrolebindings', perm: 'k8s:rbac_read', iconUrl: icons.k8sIconGroupUrl }
  ]
  }

  const network: TreeNode = {
    id: 'group:network',
    label: '网络资源',
    kind: 'folder',
    iconUrl: icons.k8sIconServiceUrl,
    children: [
      { id: 'network:services', label: 'Services', kind: 'view', resource: 'services', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconServiceUrl },
      { id: 'network:endpoints', label: 'Endpoints', kind: 'view', resource: 'endpoints', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconServiceUrl },
      { id: 'network:endpointslices', label: 'EndpointSlices', kind: 'view', resource: 'endpointslices', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconServiceUrl },
      { id: 'network:networkpolicies', label: 'NetworkPolicies', kind: 'view', resource: 'networkpolicies', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconGroupUrl },
      { id: 'network:ingresses', label: 'Ingresses', kind: 'view', resource: 'ingresses', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconIngressUrl },
      { id: 'network:ingressclasses', label: 'IngressClasses', kind: 'view', resource: 'ingressclasses', perm: 'k8s:read', iconUrl: icons.k8sIconIngressUrl }
    ]
  }

  const storage: TreeNode = {
    id: 'group:storage',
    label: '数据存储',
    kind: 'folder',
    iconUrl: icons.k8sIconPvcUrl,
    children: [
      { id: 'storage:pvcs', label: 'PVCs', kind: 'view', resource: 'pvcs', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconPvcUrl },
      { id: 'storage:pvs', label: 'PVs', kind: 'view', resource: 'pvs', perm: 'k8s:read', iconUrl: icons.k8sIconPvUrl },
      { id: 'storage:volumesnapshots', label: 'VolumeSnapshots', kind: 'view', resource: 'volumesnapshots', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconPvcUrl },
      { id: 'storage:volumesnapshotclasses', label: 'VolumeSnapshotClasses', kind: 'view', resource: 'volumesnapshotclasses', perm: 'k8s:read', iconUrl: icons.k8sIconStorageClassUrl },
      { id: 'storage:volumesnapshotcontents', label: 'VolumeSnapshotContents', kind: 'view', resource: 'volumesnapshotcontents', perm: 'k8s:read', iconUrl: icons.k8sIconPvUrl },
      { id: 'storage:storageclasses', label: 'StorageClasses', kind: 'view', resource: 'storageclasses', perm: 'k8s:read', iconUrl: icons.k8sIconStorageClassUrl },
      { id: 'storage:csidrivers', label: 'CSIDrivers', kind: 'view', resource: 'csidrivers', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'storage:csinodes', label: 'CSINodes', kind: 'view', resource: 'csinodes', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'storage:csistoragecapacities', label: 'CSIStorageCapacities', kind: 'view', resource: 'csistoragecapacities', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconGroupUrl },
      { id: 'storage:volumeattachments', label: 'VolumeAttachments', kind: 'view', resource: 'volumeattachments', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'storage:resourcequotas', label: 'ResourceQuotas', kind: 'view', resource: 'resourcequotas', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconGroupUrl },
      { id: 'storage:limitranges', label: 'LimitRanges', kind: 'view', resource: 'limitranges', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconGroupUrl }
    ]
  }

  const extensions: TreeNode = {
    id: 'group:extensions',
    label: '扩展治理',
    kind: 'folder',
    iconUrl: icons.k8sIconGroupUrl,
    children: [
      { id: 'extensions:crds', label: 'CRDs', kind: 'view', resource: 'customresourcedefinitions', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'extensions:apiservices', label: 'APIServices', kind: 'view', resource: 'apiservices', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'extensions:priorityclasses', label: 'PriorityClasses', kind: 'view', resource: 'priorityclasses', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'extensions:runtimeclasses', label: 'RuntimeClasses', kind: 'view', resource: 'runtimeclasses', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'extensions:validatingwebhooks', label: 'ValidatingWebhooks', kind: 'view', resource: 'validatingwebhookconfigurations', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'extensions:mutatingwebhooks', label: 'MutatingWebhooks', kind: 'view', resource: 'mutatingwebhookconfigurations', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'extensions:validatingadmissionpolicies', label: 'ValidatingAdmissionPolicies', kind: 'view', resource: 'validatingadmissionpolicies', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl },
      { id: 'extensions:validatingadmissionpolicybindings', label: 'ValidatingAdmissionPolicyBindings', kind: 'view', resource: 'validatingadmissionpolicybindings', perm: 'k8s:read', iconUrl: icons.k8sIconGroupUrl }
    ]
  }

  const cluster: TreeNode = {
    id: 'group:cluster',
    label: '集群资源',
    kind: 'folder',
    iconUrl: icons.k8sIconNodeUrl,
    children: [
      { id: 'cluster:namespaces', label: 'Namespaces', kind: 'view', resource: 'namespaces', perm: 'k8s:read', iconUrl: icons.k8sIconNamespaceUrl },
      { id: 'cluster:nodes', label: 'Nodes', kind: 'view', resource: 'nodes', perm: 'k8s:read', iconUrl: icons.k8sIconNodeUrl },
      { id: 'cluster:leases', label: 'Leases', kind: 'view', resource: 'leases', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconGroupUrl }
    ]
  }

  const misc: TreeNode = {
    id: 'group:misc',
    label: '事件',
    kind: 'folder',
    iconUrl: icons.k8sIconGroupUrl,
    children: [{ id: 'misc:events', label: 'Events', kind: 'view', resource: 'events', perm: 'k8s:read', namespaced: true, iconUrl: icons.k8sIconGroupUrl }]
  }

  return [dashboard, operations, workloads, jobs, network, storage, config, auth, cluster, misc, extensions, audit]
}

// ── Namespace 彩色标签系统 ─────────────────────────────────────────────
const NS_COLOR_COUNT = 12

/** 根据 namespace 名称生成 0~11 的颜色索引 */
export function nsColorIndex(ns: string): number {
  if (!ns) return 0
  let hash = 0
  for (let i = 0; i < ns.length; i++) {
    hash = ((hash << 5) - hash + ns.charCodeAt(i)) | 0
  }
  return ((hash % NS_COLOR_COUNT) + NS_COLOR_COUNT) % NS_COLOR_COUNT
}

/** 获取 Restarts 高亮 CSS class */
export function getRestartsClass(count: number | string): string {
  const n = Number(count) || 0
  if (n === 0) return 'k8s-restarts k8s-restarts--zero'
  if (n <= 5) return 'k8s-restarts k8s-restarts--low'
  return 'k8s-restarts k8s-restarts--high'
}

/** 获取 Up-to-date / Available 等数值列对比 desired 的颜色 class */
export function getReadyNumClass(current: number | string, desired: number | string): string {
  const c = Number(current) || 0
  const d = Number(desired) || 0
  if (d === 0 && c === 0) return 'k8s-ready-num'
  if (c >= d) return 'k8s-ready-num k8s-ready-num--ok'
  if (c > 0) return 'k8s-ready-num k8s-ready-num--partial'
  return 'k8s-ready-num k8s-ready-num--bad'
}
