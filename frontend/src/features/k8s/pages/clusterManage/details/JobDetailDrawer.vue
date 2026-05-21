<template>
  <WorkloadDetailDrawerShell
    v-model="visible"
    :title="detailTitle"
    :loading="refreshing || yamlLoading"
    :ns="detailNamespace"
    :name="detailName"
    @refresh="refreshDetail"
  >
    <template #summary>
      <div class="k8s-summary-grid">
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">名称:</div><div class="k8s-v">{{ detailName }}</div></div>
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">命名空间:</div><div class="k8s-v">{{ detailNamespace }}</div></div>
        <div :class="['k8s-kv', statusAccentClass]"><div class="k8s-k">状态:</div><div class="k8s-v">{{ statusText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">创建时间:</div><div class="k8s-v">{{ createdAtText }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础元数据</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="命名空间">{{ detailNamespace }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ createdAtText }}</el-descriptions-item>
              <el-descriptions-item label="parallelism">{{ parallelismText }}</el-descriptions-item>
              <el-descriptions-item label="completions">{{ completionsText }}</el-descriptions-item>
              <el-descriptions-item label="backoffLimit">{{ backoffLimitText }}</el-descriptions-item>
              <el-descriptions-item label="ttlSecondsAfterFinished">{{ ttlText }}</el-descriptions-item>
              <el-descriptions-item label="active">{{ activeText }}</el-descriptions-item>
              <el-descriptions-item label="succeeded">{{ succeededText }}</el-descriptions-item>
              <el-descriptions-item label="failed">{{ failedText }}</el-descriptions-item>
              <el-descriptions-item label="startTime">{{ startTimeText }}</el-descriptions-item>
              <el-descriptions-item label="completionTime">{{ completionTimeText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">状态条件</div></template>
            <el-table :data="conditionRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="type" label="Type" width="180" />
              <el-table-column prop="status" label="Status" width="110" align="center" header-align="center">
                <template #default="{ row: r }">
                  <el-tag :type="r.status === 'True' ? 'success' : r.status === 'False' ? 'danger' : 'info'" size="small">{{ r.status }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="Reason" width="200" show-overflow-tooltip />
              <el-table-column prop="message" label="Message" min-width="280" show-overflow-tooltip />
              <el-table-column prop="lastTransitionTime" label="LastTransition" width="200" show-overflow-tooltip />
            </el-table>
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="容器信息" name="containers">
        <div class="k8s-tab-pane">
          <div class="k8s-pane-toolbar">
            <el-radio-group v-model="activeContainerKey" size="small">
              <el-radio-button v-for="c in containerOptions" :key="c.key" :value="c.key">{{ c.label }}</el-radio-button>
            </el-radio-group>
          </div>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">容器基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="容器名称">{{ selectedContainer?.displayName || '-' }}</el-descriptions-item>
              <el-descriptions-item label="镜像地址">{{ selectedContainer?.image || '-' }}</el-descriptions-item>
              <el-descriptions-item label="镜像拉取策略">{{ selectedContainer?.imagePullPolicy || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Command">{{ selectedContainer?.commandText || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Args">{{ selectedContainer?.argsText || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Ports">{{ selectedContainer?.portsText || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">容器资源配置</div></template>
            <el-table :data="containerResourceRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="容器" min-width="160" />
              <el-table-column prop="cpuRequests" label="CPU Requests" width="140" />
              <el-table-column prop="cpuLimits" label="CPU Limits" width="140" />
              <el-table-column prop="memRequests" label="Memory Requests" width="160" />
              <el-table-column prop="memLimits" label="Memory Limits" width="160" />
              <el-table-column prop="ephemeralRequests" label="Ephemeral Requests" width="180" />
              <el-table-column prop="ephemeralLimits" label="Ephemeral Limits" width="180" />
            </el-table>
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="关联资源" name="related">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card" v-loading="relatedLoading">
            <template #header><div class="k8s-section-title">关联资源（列表）</div></template>
            <EmptyState v-if="relatedRows.length === 0" type="no-data" description="暂无关联资源" />
            <el-table v-else :data="relatedRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="group" label="分类" width="120" />
              <el-table-column prop="kind" label="Kind" width="140" />
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="summary" label="摘要" min-width="280" show-overflow-tooltip />
              <el-table-column label="操作" width="120" align="center" header-align="center">
                <template #default="{ row: r }">
                  <el-button size="small" type="primary" plain @click="onOpenRelated(r)">打开</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="事件日志" name="events">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">事件（Events）</div></template>
            <el-table :data="events" stripe size="small" class="k8s-detail-table" v-loading="eventsLoading">
              <el-table-column prop="type" label="Type" width="110" />
              <el-table-column prop="reason" label="Reason" width="180" show-overflow-tooltip />
              <el-table-column prop="message" label="Message" min-width="320" show-overflow-tooltip />
              <el-table-column prop="count" label="Count" width="110" align="center" header-align="center" />
              <el-table-column prop="lastSeen" label="LastSeen" width="130" align="center" header-align="center" />
            </el-table>
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="YAML配置" name="yaml">
        <div class="k8s-tab-pane">
          <K8sYamlPanel
            :meta="`cluster=${props.clusterId}  ${detailNamespace}/${detailName}`"
            :text="yamlViewText"
            :loading="yamlLoading"
            height="60vh"
            @refresh="loadYaml"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
  </WorkloadDetailDrawerShell>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { RefreshRight, CopyDocument, Search, Fold, Expand } from '@element-plus/icons-vue'
import K8sYamlPanel from '@/features/k8s/components/K8sYamlPanel.vue'
import WorkloadDetailDrawerShell from '@/features/k8s/components/WorkloadDetailDrawerShell.vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import EmptyState from '@/shared/components/EmptyState.vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import type { ContainerVm, EventRow, RelatedRow } from '../../ClusterManageView.utils'
import {
  asIntText,
  buildContainerVms,
  collectTemplateConfigMapsSecrets,
  formatAgeMs,
  formatTs,
  getEventTimeMs,
  getPodReadyTextLocal,
  getRowNamespace,
  isPodOwnedByJob,
  jobPodTemplateSpec,
  normalizeMultilineText
} from '../../ClusterManageView.utils'

const props = defineProps<{
  clusterId: number
  list: any[]
}>()

const emit = defineEmits<{
  (e: 'open-pod-detail', row: any): void
  (e: 'open-yaml', meta: string, loader: () => Promise<{ text: string }>): void
  (e: 'open-related', v: { action: string; kind?: string; name: string; namespace?: string }): void
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
const tab = ref<'overview' | 'containers' | 'related' | 'events' | 'yaml'>('overview')
const row = ref<any>(null)

const detailNamespace = computed(() => String(getRowNamespace(row.value) ?? ''))
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `Job 详情：${detailName.value}` : 'Job 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const parallelismText = computed(() => asIntText(row.value?.spec?.parallelism))
const completionsText = computed(() => asIntText(row.value?.spec?.completions))
const backoffLimitText = computed(() => asIntText(row.value?.spec?.backoffLimit))
const ttlText = computed(() => asIntText(row.value?.spec?.ttlSecondsAfterFinished))
const activeText = computed(() => asIntText(row.value?.status?.active))
const succeededText = computed(() => asIntText(row.value?.status?.succeeded))
const failedText = computed(() => asIntText(row.value?.status?.failed))
const startTimeText = computed(() => formatTs(row.value?.status?.startTime))
const completionTimeText = computed(() => formatTs(row.value?.status?.completionTime))
const conditionRows = computed(() => {
  const arr = row.value?.status?.conditions; if (!Array.isArray(arr)) return []
  return arr.map((c: any) => ({ type: String(c?.type ?? ''), status: String(c?.status ?? ''), reason: String(c?.reason ?? ''), message: String(c?.message ?? ''), lastTransitionTime: String(c?.lastTransitionTime ?? '') }))
})
const statusText = computed(() => {
  const a = Number(row.value?.status?.active ?? 0); const s = Number(row.value?.status?.succeeded ?? 0); const f = Number(row.value?.status?.failed ?? 0)
  if (Number.isFinite(s) && s > 0) return 'Succeeded'; if (Number.isFinite(f) && f > 0) return 'Failed'; if (Number.isFinite(a) && a > 0) return 'Running'; return 'Pending'
})
const statusAccentClass = computed(() => {
  if (statusText.value === 'Succeeded') return 'k8s-kv--ok'; if (statusText.value === 'Failed') return 'k8s-kv--bad'
  if (statusText.value === 'Running') return 'k8s-kv--info'; return 'k8s-kv--muted'
})

// ── containers ──
const activeContainerKey = ref('')
const containerBuilt = computed(() => buildContainerVms(jobPodTemplateSpec(row.value)))
const containerOptions = computed(() => containerBuilt.value.options)
const selectedContainer = computed<ContainerVm | null>(() => {
  const m = containerBuilt.value.map
  if (activeContainerKey.value && m.has(activeContainerKey.value)) return m.get(activeContainerKey.value) ?? null
  return containerOptions.value[0]?.key ? m.get(containerOptions.value[0].key) ?? null : null
})
const containerResourceRows = computed(() => containerBuilt.value.rows)

watch(
  () => [visible.value, containerOptions.value.map((o) => o.key).join('|')] as const,
  ([v]) => {
    if (!v) return
    if (activeContainerKey.value && containerBuilt.value.map.has(activeContainerKey.value)) return
    activeContainerKey.value = containerOptions.value[0]?.key ?? ''
  }
)

// ── events ──
const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value; const name = detailName.value
  const uid = String(row.value?.metadata?.uid ?? ''); if (!ns || !name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, { namespace: ns })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const filtered = list.filter((ev) => {
      const inv = ev?.involvedObject ?? ev?.regarding ?? {}
      if (String(inv?.kind ?? '') && String(inv?.kind ?? '') !== 'Job') return false
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

// ── related ──
const relatedRows = ref<RelatedRow[]>([])
const relatedLoading = ref(false)

async function loadRelated() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value; const name = detailName.value.trim()
  const uid = String(row.value?.metadata?.uid ?? '').trim(); if (!ns || !name) return
  relatedLoading.value = true
  try {
    const next: RelatedRow[] = []
    const ownerRefs: any[] = Array.isArray(row.value?.metadata?.ownerReferences) ? row.value.metadata.ownerReferences : []
    for (const o of ownerRefs) { const kind = String(o?.kind ?? '').trim(); const oName = String(o?.name ?? '').trim(); if (kind && oName) next.push({ group: '控制器', kind, name: oName, summary: 'ownerReference', action: 'owner' }) }
    const { configMaps, secrets } = collectTemplateConfigMapsSecrets(jobPodTemplateSpec(row.value))
    for (const cm of configMaps) next.push({ group: '配置', kind: 'ConfigMap', name: cm, summary: 'podTemplate 引用', action: 'configmap' })
    for (const sec of secrets) next.push({ group: '配置', kind: 'Secret', name: sec, summary: 'podTemplate 引用', action: 'secret' })
    const podsResp = await k8sApi.listPods(props.clusterId, { namespace: ns, label_selector: `job-name=${name}` })
    const pods: any[] = Array.isArray(podsResp.list) ? podsResp.list : []
    for (const p of pods) {
      if (!isPodOwnedByJob(p, name, uid)) continue
      const pName = String(p?.metadata?.name ?? '').trim(); if (!pName) continue
      next.push({ group: '运行', kind: 'Pod', name: pName, summary: `${String(p?.status?.phase ?? '-')}  ${getPodReadyTextLocal(p)}`, action: 'pod', raw: p })
    }
    relatedRows.value = next
  } catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { relatedLoading.value = false }
}

function onOpenRelated(r: any) {
  if (!props.clusterId) return
  const action = String(r?.action ?? ''); const name = String(r?.name ?? ''); const ns = detailNamespace.value
  if (!action || !name) return
  if (action === 'pod' && r?.raw) { emit('open-pod-detail', r.raw); return }
  if (action === 'configmap') { emit('open-yaml', `cluster=${props.clusterId}  ${ns}/${name}`, () => k8sApi.getConfigMapYaml(props.clusterId, ns, name)); return }
  if (action === 'secret') { emit('open-yaml', `cluster=${props.clusterId}  ${ns}/${name}`, () => k8sApi.getSecretYaml(props.clusterId, ns, name)); return }
  if (action === 'owner' && String(r?.kind ?? '') === 'CronJob') { emit('open-yaml', `cluster=${props.clusterId}  ${ns}/${name}`, () => k8sApi.getCronJobYaml(props.clusterId, ns, name)); return }
  if (action === 'owner') { emit('open-related', { action: 'owner', kind: String(r?.kind ?? ''), name, namespace: ns }) }
}

// ── yaml ──
const yamlLoading = ref(false)
const yamlText = ref('')
const yamlWrap = ref(true)
const yamlLineNumbers = ref(true)
const yamlViewerRef = ref<{ openSearch: () => void; foldAll: () => void; unfoldAll: () => void } | null>(null)
const yamlViewText = computed(() => normalizeMultilineText(yamlText.value))

async function loadYaml() {
  if (!props.clusterId || !detailNamespace.value || !detailName.value) return
  yamlLoading.value = true
  try { yamlText.value = (await k8sApi.getJobYaml(props.clusterId, detailNamespace.value, detailName.value)).text ?? '' }
  catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { yamlLoading.value = false }
}

async function refreshDetail() {
  try {
    refreshing.value = true
    emit('refresh-list'); await new Promise(r => setTimeout(r, 300))
    const next = findRow(detailNamespace.value, detailName.value)
    if (next) row.value = next
    if (tab.value === 'yaml') await loadYaml()
    if (tab.value === 'events') await loadEvents()
    if (tab.value === 'related') await loadRelated()
  } finally { refreshing.value = false }
}

watch(
  () => [visible.value, tab.value, detailNamespace.value, detailName.value] as const,
  ([v, t]) => {
    if (!v) return
    if (t === 'yaml' && !yamlText.value) void loadYaml()
    if (t === 'events' && events.value.length === 0) void loadEvents()
    if (t === 'related' && relatedRows.value.length === 0) void loadRelated()
  }
)

watch(() => visible.value, (v) => {
  if (v) return
  tab.value = 'overview'; row.value = null; yamlText.value = ''; activeContainerKey.value = ''
  events.value = []; relatedRows.value = []
})

function open(r: any) { row.value = r; tab.value = 'overview'; visible.value = true }
function close() { visible.value = false }
defineExpose({ open, close })
</script>
