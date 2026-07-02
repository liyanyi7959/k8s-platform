import { computed, ref, type ComputedRef } from 'vue'
import { ElMessageBox } from 'element-plus'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import type { EditDeploymentRequest } from '@/features/k8s/api/workload'

type ProbeTimingForm = {
  initialDelaySeconds: number | null
  timeoutSeconds: number | null
  periodSeconds: number | null
  successThreshold: number | null
  failureThreshold: number | null
}

type ContainerEditForm = {
  scope: 'containers' | 'initContainers'
  name: string
  image: string
  imagePullPolicy: string
  requestsCpu: string
  requestsMemory: string
  limitsCpu: string
  limitsMemory: string
  liveness: ProbeTimingForm
  readiness: ProbeTimingForm
  startup: ProbeTimingForm
  envText: string
  envFromText: string
  volumeMountsText: string
}

type DeploymentEditForm = {
  replicas: number | null
  labelsText: string
  tolerationsText: string
  strategyType: string
  strategyMaxSurge: string
  strategyMaxUnavailable: string
  volumesText: string
  containers: ContainerEditForm[]
  initContainers: ContainerEditForm[]
}

type ContainerOriginal = {
  image: string
  imagePullPolicy: string
  requests: Record<string, string>
  limits: Record<string, string>
  liveness: ProbeTimingForm
  readiness: ProbeTimingForm
  startup: ProbeTimingForm
  envText: string
  envFromText: string
  volumeMountsText: string
}

function getRowNamespace(row: any): string | null {
  const ns = row?.metadata?.namespace
  const v = ns != null ? String(ns).trim() : ''
  return v ? v : null
}

function numOrNull(v: any): number | null {
  if (v == null || v === '') return null
  const n = Number(v)
  return Number.isFinite(n) ? n : null
}

function objToStringMap(v: any): Record<string, string> {
  if (!v || typeof v !== 'object') return {}
  const out: Record<string, string> = {}
  for (const [k, vv] of Object.entries(v as Record<string, any>)) {
    const kk = String(k ?? '').trim()
    const ss = vv != null ? String(vv).trim() : ''
    if (!kk) continue
    if (!ss) continue
    out[kk] = ss
  }
  return out
}

function objToLabelMap(v: any): Record<string, string> {
  if (!v || typeof v !== 'object') return {}
  const out: Record<string, string> = {}
  for (const [k, vv] of Object.entries(v as Record<string, any>)) {
    const kk = String(k ?? '').trim()
    if (!kk) continue
    const ss = vv != null ? String(vv).trim() : ''
    out[kk] = ss
  }
  return out
}

function sortJsonDeep(v: any): any {
  if (Array.isArray(v)) return v.map((x) => sortJsonDeep(x))
  if (v && typeof v === 'object') {
    const out: Record<string, any> = {}
    for (const k of Object.keys(v).sort()) out[k] = sortJsonDeep((v as any)[k])
    return out
  }
  return v
}

function stringifyJsonNormalized(v: any): string {
  return JSON.stringify(sortJsonDeep(v), null, 2)
}

function normalizeLabelsText(text: string): string | null {
  const t = String(text ?? '').trim()
  if (!t) return stringifyJsonNormalized({})
  try {
    const v = JSON.parse(t)
    if (!v || typeof v !== 'object' || Array.isArray(v)) return null
    return stringifyJsonNormalized(objToLabelMap(v))
  } catch {
    return null
  }
}

function normalizeTolerationsText(text: string): string | null {
  const t = String(text ?? '').trim()
  if (!t) return stringifyJsonNormalized([])
  try {
    const v = JSON.parse(t)
    if (!Array.isArray(v)) return null
    const out = v
      .map((it) => {
        if (!it || typeof it !== 'object' || Array.isArray(it)) return null
        const m = it as Record<string, any>
        const o: Record<string, any> = {}
        if (m.key != null) o.key = String(m.key).trim()
        if (m.operator != null) o.operator = String(m.operator).trim()
        if (m.value != null) o.value = String(m.value).trim()
        if (m.effect != null) o.effect = String(m.effect).trim()
        if (m.tolerationSeconds != null && m.tolerationSeconds !== '') o.tolerationSeconds = Number(m.tolerationSeconds)
        return o
      })
      .filter(Boolean)
    return stringifyJsonNormalized(out)
  } catch {
    return null
  }
}

function cloneProbeTiming(p: any): ProbeTimingForm {
  return {
    initialDelaySeconds: numOrNull(p?.initialDelaySeconds),
    timeoutSeconds: numOrNull(p?.timeoutSeconds),
    periodSeconds: numOrNull(p?.periodSeconds),
    successThreshold: numOrNull(p?.successThreshold),
    failureThreshold: numOrNull(p?.failureThreshold)
  }
}

function mapEqual(a: Record<string, string>, b: Record<string, string>): boolean {
  const ak = Object.keys(a).sort()
  const bk = Object.keys(b).sort()
  if (ak.length !== bk.length) return false
  for (let i = 0; i < ak.length; i++) {
    const k = ak[i]
    if (k !== bk[i]) return false
    if (String(a[k] ?? '') !== String(b[k] ?? '')) return false
  }
  return true
}

function buildProbePatch(cur: ProbeTimingForm, orig: ProbeTimingForm) {
  const out: Record<string, number> = {}
  if (cur.initialDelaySeconds != null && cur.initialDelaySeconds !== orig.initialDelaySeconds) out.initialDelaySeconds = cur.initialDelaySeconds
  if (cur.timeoutSeconds != null && cur.timeoutSeconds !== orig.timeoutSeconds) out.timeoutSeconds = cur.timeoutSeconds
  if (cur.periodSeconds != null && cur.periodSeconds !== orig.periodSeconds) out.periodSeconds = cur.periodSeconds
  if (cur.successThreshold != null && cur.successThreshold !== orig.successThreshold) out.successThreshold = cur.successThreshold
  if (cur.failureThreshold != null && cur.failureThreshold !== orig.failureThreshold) out.failureThreshold = cur.failureThreshold
  return Object.keys(out).length ? out : null
}

export function useDeploymentEdit(opts: {
  clusterId: ComputedRef<number | undefined>
  clusterName: ComputedRef<string>
  workloadKind?: 'Deployment' | 'StatefulSet' | 'DaemonSet'
  onSaved?: () => void | Promise<void>
}) {
  const workloadKind = computed(() => {
    if (opts.workloadKind === 'StatefulSet') return 'StatefulSet'
    if (opts.workloadKind === 'DaemonSet') return 'DaemonSet'
    return 'Deployment'
  })
  const editVisible = ref(false)
  const editSaving = ref(false)
  const editForm = ref<DeploymentEditForm | null>(null)
  const editActiveContainer = ref('')
  const editOriginal = ref<{
    replicas: number | null
    containers: Record<string, ContainerOriginal>
    initContainers: Record<string, ContainerOriginal>
    labelsText: string
    tolerationsText: string
    selectorLabels: Record<string, string>
    strategyType: string
    strategyMaxSurge: string
    strategyMaxUnavailable: string
    volumesText: string
  } | null>(null)
  const editTarget = ref<{ namespace: string; name: string; row: any } | null>(null)

  const editMeta = computed(() => {
    const cid = opts.clusterId.value
    const t = editTarget.value
    if (!cid || !t) return ''
    const cn = String(opts.clusterName.value ?? '').trim()
    return `cluster=${cn || String(cid)}  ${workloadKind.value}  ${t.namespace}/${t.name}`
  })

  function containerKey(scope: 'containers' | 'initContainers', name: string): string {
    return `${scope}:${name}`
  }

  function open(row: any) {
    try {
      const ns = getRowNamespace(row)
      // if (!opts.clusterId.value || !ns) return
      if (!ns) return
    const name = String(row?.metadata?.name ?? '').trim()
      if (!name) return

      const replicas = workloadKind.value === 'DaemonSet' ? null : numOrNull(row?.spec?.replicas)
    const selectorLabels = objToLabelMap(row?.spec?.selector?.matchLabels)
    const labelsText = stringifyJsonNormalized(objToLabelMap(row?.spec?.template?.metadata?.labels))
    const tolerationsText = stringifyJsonNormalized(Array.isArray(row?.spec?.template?.spec?.tolerations) ? row.spec.template.spec.tolerations : [])

    // Strategy
    const rawStrategy = row?.spec?.strategy ?? row?.spec?.updateStrategy ?? {}
    const origStrategyType = String(rawStrategy?.type ?? '').trim()
    const origMaxSurge = String(rawStrategy?.rollingUpdate?.maxSurge ?? '').trim()
    const origMaxUnavailable = String(rawStrategy?.rollingUpdate?.maxUnavailable ?? '').trim()

    // Volumes
    const rawVolumes: any[] = Array.isArray(row?.spec?.template?.spec?.volumes) ? row.spec.template.spec.volumes : []
    const origVolumesText = stringifyJsonNormalized(rawVolumes)

    const cs: any[] = Array.isArray(row?.spec?.template?.spec?.containers) ? row.spec.template.spec.containers : []
    const ics: any[] = Array.isArray(row?.spec?.template?.spec?.initContainers) ? row.spec.template.spec.initContainers : []
    const containers: ContainerEditForm[] = cs
      .map((c: any) => {
        const cn = String(c?.name ?? '').trim()
        if (!cn) return null
        const image = String(c?.image ?? '').trim()
        const imagePullPolicy = String(c?.imagePullPolicy ?? '').trim()
        const reqs = objToStringMap(c?.resources?.requests)
        const lims = objToStringMap(c?.resources?.limits)
        const envArr: any[] = Array.isArray(c?.env) ? c.env : []
        const envFromArr: any[] = Array.isArray(c?.envFrom) ? c.envFrom : []
        const vmArr: any[] = Array.isArray(c?.volumeMounts) ? c.volumeMounts : []
        return {
          scope: 'containers',
          name: cn,
          image,
          imagePullPolicy,
          requestsCpu: String(reqs.cpu ?? ''),
          requestsMemory: String(reqs.memory ?? ''),
          limitsCpu: String(lims.cpu ?? ''),
          limitsMemory: String(lims.memory ?? ''),
          liveness: cloneProbeTiming(c?.livenessProbe),
          readiness: cloneProbeTiming(c?.readinessProbe),
          startup: cloneProbeTiming(c?.startupProbe),
          envText: stringifyJsonNormalized(envArr),
          envFromText: stringifyJsonNormalized(envFromArr),
          volumeMountsText: stringifyJsonNormalized(vmArr)
        }
      })
      .filter(Boolean) as ContainerEditForm[]

    const initContainers: ContainerEditForm[] = ics
      .map((c: any) => {
        const cn = String(c?.name ?? '').trim()
        if (!cn) return null
        const image = String(c?.image ?? '').trim()
        const imagePullPolicy = String(c?.imagePullPolicy ?? '').trim()
        const reqs = objToStringMap(c?.resources?.requests)
        const lims = objToStringMap(c?.resources?.limits)
        const envArr: any[] = Array.isArray(c?.env) ? c.env : []
        const envFromArr: any[] = Array.isArray(c?.envFrom) ? c.envFrom : []
        const vmArr: any[] = Array.isArray(c?.volumeMounts) ? c.volumeMounts : []
        return {
          scope: 'initContainers',
          name: cn,
          image,
          imagePullPolicy,
          requestsCpu: String(reqs.cpu ?? ''),
          requestsMemory: String(reqs.memory ?? ''),
          limitsCpu: String(lims.cpu ?? ''),
          limitsMemory: String(lims.memory ?? ''),
          liveness: cloneProbeTiming(c?.livenessProbe),
          readiness: cloneProbeTiming(c?.readinessProbe),
          startup: cloneProbeTiming(c?.startupProbe),
          envText: stringifyJsonNormalized(envArr),
          envFromText: stringifyJsonNormalized(envFromArr),
          volumeMountsText: stringifyJsonNormalized(vmArr)
        }
      })
      .filter(Boolean) as ContainerEditForm[]

    const origContainers: Record<string, ContainerOriginal> = {}
    for (const c of containers) {
      const raw = cs.find((x) => String(x?.name ?? '').trim() === c.name)
      origContainers[c.name] = {
        image: String(raw?.image ?? '').trim(),
        imagePullPolicy: String(raw?.imagePullPolicy ?? '').trim(),
        requests: objToStringMap(raw?.resources?.requests),
        limits: objToStringMap(raw?.resources?.limits),
        liveness: cloneProbeTiming(raw?.livenessProbe),
        readiness: cloneProbeTiming(raw?.readinessProbe),
        startup: cloneProbeTiming(raw?.startupProbe),
        envText: c.envText,
        envFromText: c.envFromText,
        volumeMountsText: c.volumeMountsText
      }
    }

    const origInitContainers: Record<string, ContainerOriginal> = {}
    for (const c of initContainers) {
      const raw = ics.find((x) => String(x?.name ?? '').trim() === c.name)
      origInitContainers[c.name] = {
        image: String(raw?.image ?? '').trim(),
        imagePullPolicy: String(raw?.imagePullPolicy ?? '').trim(),
        requests: objToStringMap(raw?.resources?.requests),
        limits: objToStringMap(raw?.resources?.limits),
        liveness: cloneProbeTiming(raw?.livenessProbe),
        readiness: cloneProbeTiming(raw?.readinessProbe),
        startup: cloneProbeTiming(raw?.startupProbe),
        envText: c.envText,
        envFromText: c.envFromText,
        volumeMountsText: c.volumeMountsText
      }
    }

    editTarget.value = { namespace: ns, name, row }
    editOriginal.value = {
      replicas, containers: origContainers, initContainers: origInitContainers,
      labelsText, tolerationsText, selectorLabels,
      strategyType: origStrategyType, strategyMaxSurge: origMaxSurge, strategyMaxUnavailable: origMaxUnavailable,
      volumesText: origVolumesText
    }
    editForm.value = {
      replicas, labelsText, tolerationsText, containers, initContainers,
      strategyType: origStrategyType,
      strategyMaxSurge: origMaxSurge,
      strategyMaxUnavailable: origMaxUnavailable,
      volumesText: origVolumesText
    }
    if (containers.length > 0) editActiveContainer.value = containerKey('containers', containers[0].name)
    else if (initContainers.length > 0) editActiveContainer.value = containerKey('initContainers', initContainers[0].name)
    else editActiveContainer.value = ''
    editVisible.value = true
  } catch (e) {
    console.error('Error opening edit dialog:', e)
    notifyError('打开编辑窗口失败')
  }
  }

  function close() {
    editVisible.value = false
  }

  function getOrigContainer(scope: 'containers' | 'initContainers', name: string): ContainerOriginal | null {
    const o = scope === 'containers' ? editOriginal.value?.containers?.[name] : editOriginal.value?.initContainers?.[name]
    return o ?? null
  }

  function isReplicasChanged(): boolean {
    if (workloadKind.value === 'DaemonSet') return false
    const orig = editOriginal.value?.replicas ?? null
    const cur = editForm.value?.replicas ?? null
    return cur != null && cur !== orig
  }

  function isImageInvalid(c: ContainerEditForm): boolean {
    const img = String(c.image ?? '').trim()
    return !img
  }

  function isImageChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const img = String(c.image ?? '').trim()
    if (!img) return false
    return img !== o.image
  }

  function isImagePullPolicyChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const cur = String(c.imagePullPolicy ?? '').trim()
    const orig = String(o.imagePullPolicy ?? '').trim()
    return cur !== orig
  }

  function isRequestsCpuChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const v = String(c.requestsCpu ?? '').trim()
    if (!v) return false
    return v !== String(o.requests.cpu ?? '').trim()
  }

  function isRequestsMemoryChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const v = String(c.requestsMemory ?? '').trim()
    if (!v) return false
    return v !== String(o.requests.memory ?? '').trim()
  }

  function isLimitsCpuChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const v = String(c.limitsCpu ?? '').trim()
    if (!v) return false
    return v !== String(o.limits.cpu ?? '').trim()
  }

  function isLimitsMemoryChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const v = String(c.limitsMemory ?? '').trim()
    if (!v) return false
    return v !== String(o.limits.memory ?? '').trim()
  }

  function isContainerResourcesChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false

    const nextRequests: Record<string, string> = { ...o.requests }
    const nextLimits: Record<string, string> = { ...o.limits }

    const rCpu = String(c.requestsCpu ?? '').trim()
    const rMem = String(c.requestsMemory ?? '').trim()
    const lCpu = String(c.limitsCpu ?? '').trim()
    const lMem = String(c.limitsMemory ?? '').trim()

    if (rCpu) nextRequests.cpu = rCpu
    if (rMem) nextRequests.memory = rMem
    if (lCpu) nextLimits.cpu = lCpu
    if (lMem) nextLimits.memory = lMem

    return !mapEqual(nextRequests, o.requests) || !mapEqual(nextLimits, o.limits)
  }

  function isProbeValueChanged(c: ContainerEditForm, which: 'liveness' | 'readiness' | 'startup', key: keyof ProbeTimingForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const cur = c[which]?.[key] ?? null
    if (cur == null) return false
    const orig = o[which]?.[key] ?? null
    return cur !== orig
  }

  function isProbeChanged(c: ContainerEditForm, which: 'liveness' | 'readiness' | 'startup'): boolean {
    return (
      isProbeValueChanged(c, which, 'initialDelaySeconds') ||
      isProbeValueChanged(c, which, 'timeoutSeconds') ||
      isProbeValueChanged(c, which, 'periodSeconds') ||
      isProbeValueChanged(c, which, 'successThreshold') ||
      isProbeValueChanged(c, which, 'failureThreshold')
    )
  }

  function isContainerProbesChanged(c: ContainerEditForm): boolean {
    return isProbeChanged(c, 'liveness') || isProbeChanged(c, 'readiness') || isProbeChanged(c, 'startup')
  }

  function normalizeJsonArrayText(text: string): string | null {
    const t = String(text ?? '').trim()
    if (!t) return stringifyJsonNormalized([])
    try {
      const v = JSON.parse(t)
      if (!Array.isArray(v)) return null
      return stringifyJsonNormalized(v)
    } catch {
      return null
    }
  }

  function isEnvChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const cur = normalizeJsonArrayText(c.envText)
    if (cur == null) return true
    return cur !== o.envText
  }

  function isEnvFromChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const cur = normalizeJsonArrayText(c.envFromText)
    if (cur == null) return true
    return cur !== o.envFromText
  }

  function isVolumeMountsChanged(c: ContainerEditForm): boolean {
    const o = getOrigContainer(c.scope, c.name)
    if (!o) return false
    const cur = normalizeJsonArrayText(c.volumeMountsText)
    if (cur == null) return true
    return cur !== o.volumeMountsText
  }

  function isContainerEnvChanged(c: ContainerEditForm): boolean {
    return isEnvChanged(c) || isEnvFromChanged(c) || isVolumeMountsChanged(c)
  }

  function isStrategyChanged(): boolean {
    const orig = editOriginal.value
    if (!orig) return false
    const f = editForm.value
    if (!f) return false
    if (String(f.strategyType ?? '').trim() !== orig.strategyType) return true
    if (String(f.strategyMaxSurge ?? '').trim() !== orig.strategyMaxSurge) return true
    if (String(f.strategyMaxUnavailable ?? '').trim() !== orig.strategyMaxUnavailable) return true
    return false
  }

  function isVolumesChanged(): boolean {
    const orig = editOriginal.value
    if (!orig) return false
    const cur = normalizeJsonArrayText(editForm.value?.volumesText ?? '')
    if (cur == null) return true
    return cur !== orig.volumesText
  }

  function isContainerChanged(c: ContainerEditForm): boolean {
    return isImageChanged(c) || isImagePullPolicyChanged(c) || isContainerResourcesChanged(c) || isContainerProbesChanged(c) || isContainerEnvChanged(c)
  }

  function isLabelsChanged(): boolean {
    const orig = editOriginal.value?.labelsText ?? stringifyJsonNormalized({})
    const curRaw = editForm.value?.labelsText ?? ''
    const cur = normalizeLabelsText(curRaw)
    if (cur == null) return true
    return cur !== orig
  }

  function isTolerationsChanged(): boolean {
    const orig = editOriginal.value?.tolerationsText ?? stringifyJsonNormalized([])
    const curRaw = editForm.value?.tolerationsText ?? ''
    const cur = normalizeTolerationsText(curRaw)
    if (cur == null) return true
    return cur !== orig
  }

  function isMetaChanged(): boolean {
    return isLabelsChanged() || isTolerationsChanged()
  }

  function isEditChanged(): boolean {
    if (isReplicasChanged()) return true
    if (isMetaChanged()) return true
    if (isStrategyChanged()) return true
    if (isVolumesChanged()) return true
    const cs = editForm.value?.containers ?? []
    if (cs.some((c) => isContainerChanged(c))) return true
    const ics = editForm.value?.initContainers ?? []
    return ics.some((c) => isContainerChanged(c))
  }

  function parseActiveContainerKey(key: string): { scope: 'containers' | 'initContainers'; name: string } | null {
    const t = String(key ?? '').trim()
    if (!t) return null
    const idx = t.indexOf(':')
    if (idx <= 0) return null
    const scope = t.slice(0, idx) as 'containers' | 'initContainers'
    const name = t.slice(idx + 1)
    if ((scope !== 'containers' && scope !== 'initContainers') || !name) return null
    return { scope, name }
  }

  async function save() {
    const cid = opts.clusterId.value
    if (!cid || !editTarget.value || !editForm.value) return
    const orig = editOriginal.value
    if (!orig) return

    const payload: EditDeploymentRequest = { namespace: editTarget.value.namespace, name: editTarget.value.name }

    if (workloadKind.value !== 'DaemonSet' && editForm.value.replicas != null && editForm.value.replicas !== orig.replicas) {
      payload.replicas = editForm.value.replicas
    }

    if (isLabelsChanged()) {
      const normalized = normalizeLabelsText(editForm.value.labelsText)
      if (normalized == null) {
        notifyError('Labels JSON 格式错误')
        return
      }
      const labels = JSON.parse(normalized) as Record<string, string>
      for (const [k, v] of Object.entries(orig.selectorLabels || {})) {
        if (String(labels?.[k] ?? '') !== String(v ?? '')) {
          notifyError(`Labels 必须包含 selector: ${k}=${v}`)
          return
        }
      }
      payload.labels = labels
    }

    if (isTolerationsChanged()) {
      const normalized = normalizeTolerationsText(editForm.value.tolerationsText)
      if (normalized == null) {
        notifyError('Tolerations JSON 格式错误')
        return
      }
      const ts = JSON.parse(normalized) as Array<any>
      payload.tolerations = ts.map((x) => ({
        key: x?.key,
        operator: x?.operator,
        value: x?.value,
        effect: x?.effect,
        tolerationSeconds: x?.tolerationSeconds
      }))
    }

    const buildContainersPatch = (
      scope: 'containers' | 'initContainers',
      items: ContainerEditForm[],
      origItems: Record<string, ContainerOriginal>
    ): Array<any> => {
      const out: Array<any> = []
      for (const c of items) {
        const o = origItems[c.name]
        if (!o) continue

        const item: any = { name: c.name }
        let changed = false

        const img = String(c.image ?? '').trim()
        if (!img) {
          notifyError(`${scope === 'initContainers' ? 'InitContainer' : '容器'} ${c.name} 镜像不能为空`)
          throw new Error('validation_failed')
        }
        if (img !== o.image) {
          item.image = img
          changed = true
        }

        const ipp = String(c.imagePullPolicy ?? '').trim()
        if (ipp !== String(o.imagePullPolicy ?? '').trim()) {
          item.imagePullPolicy = ipp
          changed = true
        }

        const nextRequests: Record<string, string> = { ...o.requests }
        const nextLimits: Record<string, string> = { ...o.limits }

        const rCpu = String(c.requestsCpu ?? '').trim()
        const rMem = String(c.requestsMemory ?? '').trim()
        const lCpu = String(c.limitsCpu ?? '').trim()
        const lMem = String(c.limitsMemory ?? '').trim()

        if (rCpu) nextRequests.cpu = rCpu
        if (rMem) nextRequests.memory = rMem
        if (lCpu) nextLimits.cpu = lCpu
        if (lMem) nextLimits.memory = lMem

        const requestsChanged = !mapEqual(nextRequests, o.requests)
        const limitsChanged = !mapEqual(nextLimits, o.limits)
        if (requestsChanged || limitsChanged) {
          item.resources = {}
          if (requestsChanged) item.resources.requests = nextRequests
          if (limitsChanged) item.resources.limits = nextLimits
          changed = true
        }

        const liveness = buildProbePatch(c.liveness, o.liveness)
        const readiness = buildProbePatch(c.readiness, o.readiness)
        const startup = buildProbePatch(c.startup, o.startup)
        if (liveness || readiness || startup) {
          item.probes = {}
          if (liveness) item.probes.liveness = liveness
          if (readiness) item.probes.readiness = readiness
          if (startup) item.probes.startup = startup
          changed = true
        }

        // env / envFrom / volumeMounts
        if (isEnvChanged(c)) {
          const normalized = normalizeJsonArrayText(c.envText)
          if (normalized == null) {
            notifyError(`${c.name}: env JSON 格式错误`)
            throw new Error('validation_failed')
          }
          item.env = JSON.parse(normalized)
          changed = true
        }
        if (isEnvFromChanged(c)) {
          const normalized = normalizeJsonArrayText(c.envFromText)
          if (normalized == null) {
            notifyError(`${c.name}: envFrom JSON 格式错误`)
            throw new Error('validation_failed')
          }
          item.envFrom = JSON.parse(normalized)
          changed = true
        }
        if (isVolumeMountsChanged(c)) {
          const normalized = normalizeJsonArrayText(c.volumeMountsText)
          if (normalized == null) {
            notifyError(`${c.name}: volumeMounts JSON 格式错误`)
            throw new Error('validation_failed')
          }
          item.volumeMounts = JSON.parse(normalized)
          changed = true
        }

        if (changed) out.push(item)
      }
      return out
    }

    try {
      const containersPatch = buildContainersPatch('containers', editForm.value.containers, orig.containers)
      if (containersPatch.length) payload.containers = containersPatch
      const initContainersPatch = buildContainersPatch('initContainers', editForm.value.initContainers ?? [], orig.initContainers ?? {})
      if (initContainersPatch.length) payload.initContainers = initContainersPatch
    } catch (e) {
      if (e instanceof Error && e.message === 'validation_failed') return
      return
    }

    // Strategy
    if (isStrategyChanged()) {
      const st: NonNullable<EditDeploymentRequest['strategy']> = {}
      const t = String(editForm.value.strategyType ?? '').trim()
      if (t) st.type = t
      const ms = String(editForm.value.strategyMaxSurge ?? '').trim()
      if (ms) st.maxSurge = ms
      const mu = String(editForm.value.strategyMaxUnavailable ?? '').trim()
      if (mu) st.maxUnavailable = mu
      payload.strategy = st
    }

    // Volumes
    if (isVolumesChanged()) {
      const normalized = normalizeJsonArrayText(editForm.value.volumesText ?? '')
      if (normalized == null) {
        notifyError('Volumes JSON 格式错误')
        return
      }
      payload.volumes = JSON.parse(normalized)
    }

    if (
      payload.replicas == null &&
      payload.labels == null &&
      payload.tolerations == null &&
      payload.strategy == null &&
      payload.volumes == null &&
      (!payload.containers || payload.containers.length === 0) &&
      (!payload.initContainers || payload.initContainers.length === 0)
    ) {
      notifySuccess('无变更')
      editVisible.value = false
      return
    }

    try {
      await ElMessageBox.confirm(`确认保存 ${workloadKind.value} 修改？`, '确认', {
        type: 'warning',
        confirmButtonText: '保存',
        cancelButtonText: '取消'
      })
    } catch {
      return
    }

    editSaving.value = true
    try {
      if (workloadKind.value === 'StatefulSet') await k8sApi.editStatefulSet(cid, payload)
      else if (workloadKind.value === 'DaemonSet') await k8sApi.editDaemonSet(cid, payload)
      else await k8sApi.editDeployment(cid, payload)
      notifySuccess('保存成功')
      editVisible.value = false
      await opts.onSaved?.()
    } catch (e) {
      const err = e as ApiError
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    } finally {
      editSaving.value = false
    }
  }

  return {
    editVisible,
    editSaving,
    editForm,
    editActiveContainer,
    editTarget,
    editMeta,
    openEditDeployment: open,
    closeEditDeployment: close,
    saveEditDeployment: save,
    parseActiveContainerKey,
    containerKey,
    isEditChanged,
    isMetaChanged,
    isLabelsChanged,
    isTolerationsChanged,
    isContainerChanged,
    isContainerResourcesChanged,
    isContainerProbesChanged,
    isContainerEnvChanged,
    isReplicasChanged,
    isImageChanged,
    isImagePullPolicyChanged,
    isImageInvalid,
    isRequestsCpuChanged,
    isRequestsMemoryChanged,
    isLimitsCpuChanged,
    isLimitsMemoryChanged,
    isProbeChanged,
    isProbeValueChanged,
    isEnvChanged,
    isEnvFromChanged,
    isVolumeMountsChanged,
    isStrategyChanged,
    isVolumesChanged
  }
}
