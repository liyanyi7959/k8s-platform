<template>
  <WorkloadDetailDrawerShell
    v-model="visible"
    :title="detailTitle"
    :loading="refreshing"
    :ns="detailNamespace"
    :name="detailName"
    @refresh="refreshDetail"
  >
    <template #summary>
      <div class="k8s-summary-grid">
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">名称:</div><div class="k8s-v">{{ detailName }}</div></div>
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">命名空间:</div><div class="k8s-v">{{ detailNamespace }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Class:</div><div class="k8s-v"><span class="k8s-link">{{ classText }}</span></div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Hosts:</div><div class="k8s-v">{{ hostsText }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="命名空间">{{ detailNamespace }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ createdAtText }}</el-descriptions-item>
              <el-descriptions-item label="class">{{ classText }}</el-descriptions-item>
              <el-descriptions-item label="address">{{ addressText }}</el-descriptions-item>
              <el-descriptions-item label="hosts">{{ hostsText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Rules</div></template>
            <CodeMirrorViewer :text="rulesViewText" language="text" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Labels</div></template>
            <CodeMirrorViewer :text="labelsViewText" language="json" height="200px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="关联资源" name="related">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Services（Ingress 后端）</div>
                <div class="k8s-section-actions">
                  <el-space :size="8">
                    <el-tag v-if="relatedLoading" size="small" type="info" effect="light">加载中</el-tag>
                    <el-tag v-else size="small" type="info" effect="light">共 {{ relatedServices.length }} 条</el-tag>
                    <el-tooltip content="刷新" placement="top">
                      <el-button size="small" :icon="RefreshRight" circle :loading="relatedLoading" @click="loadRelated" />
                    </el-tooltip>
                  </el-space>
                </div>
              </div>
            </template>
            <el-table :data="relatedServices" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
              <el-table-column prop="type" label="Type" width="140" show-overflow-tooltip />
              <el-table-column prop="portsText" label="Ports" min-width="240" show-overflow-tooltip />
              <el-table-column prop="selectorText" label="Selector" min-width="220" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Pods（匹配后端 Services Selector）</div>
                <div class="k8s-section-actions">
                  <el-tag size="small" type="info" effect="light">共 {{ relatedPods.length }} 条</el-tag>
                </div>
              </div>
            </template>
            <el-table :data="relatedPods" stripe size="small" class="k8s-detail-table" @row-dblclick="(r: any) => emit('open-related-pod', r)">
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
              <el-table-column prop="phase" label="Phase" width="140" show-overflow-tooltip />
              <el-table-column prop="ready" label="Ready" width="100" align="center" header-align="center" />
              <el-table-column prop="restarts" label="Restarts" width="100" align="center" header-align="center" />
              <el-table-column prop="node" label="Node" min-width="200" show-overflow-tooltip />
              <el-table-column prop="ownersText" label="Owners" min-width="240" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">资源（从 Pods ownerReferences + 工作负载模板聚合）</div>
                <div class="k8s-section-actions">
                  <el-tag size="small" type="info" effect="light">共 {{ relatedControllers.length }} 条</el-tag>
                </div>
              </div>
            </template>
            <el-table :data="relatedControllers" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="kind" label="Kind" width="180" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
              <el-table-column prop="name" label="名称" min-width="260" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Secrets（TLS）</div>
                <div class="k8s-section-actions">
                  <el-tag size="small" type="info" effect="light">共 {{ relatedSecrets.length }} 条</el-tag>
                </div>
              </div>
            </template>
            <el-table :data="relatedSecrets" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="名称" min-width="240" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
            </el-table>
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="事件日志" name="events">
        <div class="k8s-tab-pane">
          <div class="k8s-pane-toolbar">
            <el-space :size="8">
              <el-tooltip content="刷新" placement="top">
                <el-button size="small" :icon="RefreshRight" circle :loading="eventsLoading" @click="loadEvents" />
              </el-tooltip>
              <el-tag v-if="eventsLoading" size="small" type="info" effect="light">加载中</el-tag>
              <el-tag v-else size="small" type="info" effect="light">共 {{ events.length }} 条</el-tag>
            </el-space>
          </div>
          <el-table :data="events" stripe size="small" class="k8s-detail-table">
            <el-table-column prop="type" label="Type" width="110" />
            <el-table-column prop="reason" label="Reason" width="200" show-overflow-tooltip />
            <el-table-column prop="message" label="Message" min-width="360" show-overflow-tooltip />
            <el-table-column prop="count" label="Count" width="90" align="center" header-align="center" />
            <el-table-column prop="lastSeen" label="LastSeen" width="140" align="center" header-align="center" />
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="YAML配置" name="yaml">
        <div class="k8s-tab-pane">
          <K8sYamlPanel
            :meta="`cluster=${props.clusterId}  ${detailNamespace}/${detailName}`"
            :text="yamlViewText"
            :loading="yamlLoading"
            height="100%"
            @refresh="loadYaml"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
  </WorkloadDetailDrawerShell>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { RefreshRight, CopyDocument, Search, Fold, Expand, Sunny, Moon } from '@element-plus/icons-vue'
import WorkloadDetailDrawerShell from '@/features/k8s/components/WorkloadDetailDrawerShell.vue'
import K8sYamlPanel from '@/features/k8s/components/K8sYamlPanel.vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import type { ControllerRow, EventRow } from '../../ClusterManageView.utils'
import {
  collectControllersFromPodsRaw,
  collectIngressServiceNames,
  formatAgeMs,
  formatPorts,
  formatRules,
  formatSelector,
  formatTs,
  getEventTimeMs,
  getHosts,
  getRowNamespace,
  matchLabels,
  mergeControllers,
  normalizeLabelRecord,
  normalizeMultilineText,
  toRelatedPodVmFromPod
} from '../../ClusterManageView.utils'

const props = defineProps<{
  clusterId: number
  editorTheme: 'auto' | 'light' | 'dark'
  editorThemeEffectiveDark: boolean
  list: any[]
}>()

const emit = defineEmits<{
  (e: 'toggle-editor-theme'): void
  (e: 'open-related-pod', row: any): void
  (e: 'refresh-list'): void
}>()

function copyText(text: string) {
  if (!text) return
  navigator.clipboard.writeText(text).catch(() => notifyError('复制失败'))
}

function findRow(ns: string, name: string): any | null {
  for (const it of props.list ?? []) {
    if (getRowNamespace(it) === ns && String(it?.metadata?.name ?? '') === name) return it
  }
  return null
}

// ── state ──
const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'related' | 'events' | 'yaml'>('overview')
const row = ref<any>(null)

const detailNamespace = computed(() => getRowNamespace(row.value) || '')
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `Ingress 详情：${detailName.value}` : 'Ingress 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const classText = computed(() => String(row.value?.spec?.ingressClassName ?? '-'))
const hostsText = computed(() => getHosts(row.value).join(', ') || '-')
const addressText = computed(() => {
  const ing = row.value?.status?.loadBalancer?.ingress
  if (!Array.isArray(ing) || ing.length === 0) return '-'
  return ing.map((x: any) => String(x?.hostname ?? x?.ip ?? '').trim()).filter(Boolean).join(', ') || '-'
})
const rulesViewText = computed(() => formatRules(row.value))
const labelsViewText = computed(() => JSON.stringify(row.value?.metadata?.labels ?? {}, null, 2))

// ── related ──
const relatedServices = ref<any[]>([])
const relatedPods = ref<any[]>([])
const relatedControllers = ref<ControllerRow[]>([])
const relatedSecrets = ref<any[]>([])
const relatedLoading = ref(false)

// ── events ──
const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

// ── yaml ──
const yamlLoading = ref(false)
const yamlText = ref('')
const yamlWrap = ref(true)
const yamlLineNumbers = ref(true)
const yamlViewerRef = ref<{ openSearch: () => void; foldAll: () => void; unfoldAll: () => void } | null>(null)
const yamlViewText = computed(() => normalizeMultilineText(yamlText.value))

async function loadYaml() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value; const name = detailName.value
  if (!ns || !name) return
  yamlLoading.value = true
  try { yamlText.value = (await k8sApi.getIngressYaml(props.clusterId, ns, name)).text ?? '' }
  catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { yamlLoading.value = false }
}

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value; const name = detailName.value
  const uid = String(row.value?.metadata?.uid ?? '').trim()
  if (!ns || !name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, { namespace: ns })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const filtered = list.filter((ev) => {
      const inv = ev?.involvedObject ?? ev?.regarding ?? {}
      if (String(inv?.kind ?? '') && String(inv?.kind ?? '') !== 'Ingress') return false
      if (String(inv?.namespace ?? '') && String(inv?.namespace ?? '') !== ns) return false
      if (uid && String(inv?.uid ?? '')) return String(inv?.uid ?? '') === uid
      return String(inv?.name ?? '') === name
    })
    const now = Date.now()
    events.value = filtered
      .map((ev) => { const t = getEventTimeMs(ev); return { tMs: t ?? -1, type: String(ev?.type ?? '') || '-', reason: String(ev?.reason ?? '') || '-', message: String(ev?.message ?? '') || '-', count: Number(ev?.count ?? ev?.series?.count ?? 1) || 1, lastSeen: t != null ? formatAgeMs(Math.max(0, now - t)) : '-' } })
      .sort((a, b) => b.tMs - a.tMs)
      .map(({ tMs: _, ...rest }) => rest)
  } catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { eventsLoading.value = false }
}

async function loadRelated() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value
  if (!ns) return
  const serviceNames = collectIngressServiceNames(row.value)
  const tls: any[] = Array.isArray(row.value?.spec?.tls) ? row.value.spec.tls : []
  const secretNames = tls.map((t) => String(t?.secretName ?? '').trim()).filter(Boolean).filter((s, i, a) => a.indexOf(s) === i).sort()
  relatedLoading.value = true
  try {
    relatedServices.value = []; relatedPods.value = []; relatedControllers.value = []
    relatedSecrets.value = secretNames.map((n) => ({ namespace: ns, name: n }))
    const svcResp = await k8sApi.listServices(props.clusterId, { namespace: ns })
    const svcList: any[] = Array.isArray(svcResp.list) ? svcResp.list : []
    const svcByName = new Map<string, any>()
    for (const s of svcList) { const n = String(s?.metadata?.name ?? '').trim(); if (n) svcByName.set(n, s) }
    const services = serviceNames.map((n) => svcByName.get(n)).filter(Boolean)
    relatedServices.value = services.map((s: any) => {
      const sel = normalizeLabelRecord(s?.spec?.selector)
      return { namespace: ns, name: String(s?.metadata?.name ?? ''), type: String(s?.spec?.type ?? '') || '-', portsText: formatPorts(s?.spec?.ports ?? []) || '-', selectorText: formatSelector(sel), selector: sel, raw: s }
    })
    const selectors = relatedServices.value.map((s) => s.selector).filter((x) => x && Object.keys(x).length > 0)
    if (selectors.length > 0) {
      const podsResp = await k8sApi.listPods(props.clusterId, { namespace: ns })
      const podsList: any[] = Array.isArray(podsResp.list) ? podsResp.list : []
      const podSeen = new Set<string>(); const podsOut: any[] = []
      for (const p of podsList) {
        const pName = String(p?.metadata?.name ?? '').trim()
        if (!pName || !selectors.some((sel) => matchLabels(p?.metadata?.labels, sel))) continue
        const key = `${ns}/${pName}`; if (podSeen.has(key)) continue; podSeen.add(key); podsOut.push(p)
      }
      relatedPods.value = podsOut.map((p) => toRelatedPodVmFromPod(p)).sort((a, b) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`))
      const cFromPods = collectControllersFromPodsRaw(relatedPods.value.map((p: any) => p.rawPod ?? p))
      const w = await k8sApi.listWorkloads(props.clusterId, { namespace: ns })
      const cFromWl: ControllerRow[] = (Array.isArray(w.list) ? w.list : [])
        .filter((it: any) => selectors.some((sel) => matchLabels(it?.spec?.template?.metadata?.labels, sel)))
        .map((it: any) => ({ kind: String(it?.kind ?? '').trim(), namespace: String(it?.metadata?.namespace ?? ns).trim(), name: String(it?.metadata?.name ?? '').trim() }))
        .filter((c: ControllerRow) => c.kind && c.namespace && c.name)
      relatedControllers.value = mergeControllers(cFromPods, cFromWl)
    }
  } catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { relatedLoading.value = false }
}

async function refreshDetail() {
  if (!visible.value) return
  try {
    refreshing.value = true
    emit('refresh-list'); await new Promise(r => setTimeout(r, 300))
    const next = findRow(detailNamespace.value, detailName.value)
    if (next) row.value = next
    if (tab.value === 'events') await loadEvents()
    if (tab.value === 'related') await loadRelated()
    if (tab.value === 'yaml') await loadYaml()
  } finally { refreshing.value = false }
}

watch(() => visible.value, (v) => {
  if (v) return
  tab.value = 'overview'; row.value = null
  relatedServices.value = []; relatedPods.value = []; relatedControllers.value = []; relatedSecrets.value = []
  events.value = []; yamlText.value = ''
})

watch(
  () => [visible.value, tab.value, detailNamespace.value, detailName.value] as const,
  ([v, t]) => {
    if (!v) return
    if (t === 'events' && events.value.length === 0) void loadEvents()
    if (t === 'related' && relatedServices.value.length === 0 && relatedPods.value.length === 0 && relatedControllers.value.length === 0 && relatedSecrets.value.length === 0) void loadRelated()
    if (t === 'yaml' && !yamlText.value) void loadYaml()
  }
)

function open(r: any) { row.value = r; tab.value = 'overview'; visible.value = true }
function close() { visible.value = false }
defineExpose({ open, close })
</script>
