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
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Keys:</div><div class="k8s-v">{{ dataKeysCount }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">创建时间:</div><div class="k8s-v">{{ createdAtText }}</div></div>
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
              <el-descriptions-item label="immutable">{{ immutableText }}</el-descriptions-item>
              <el-descriptions-item label="dataKeys">{{ dataKeysCount }}</el-descriptions-item>
              <el-descriptions-item label="labels">{{ labelsCount }}</el-descriptions-item>
              <el-descriptions-item label="annotations">{{ annotationsCount }}</el-descriptions-item>
              <el-descriptions-item label="resourceVersion">{{ resourceVersionText }}</el-descriptions-item>
              <el-descriptions-item label="uid">{{ uidText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Labels</div></template>
            <CodeMirrorViewer :text="labelsViewText" language="json" height="260px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Annotations</div></template>
            <CodeMirrorViewer :text="annotationsViewText" language="json" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="Data" name="data">
        <div class="k8s-tab-pane">
          <div class="k8s-pane-toolbar">
            <el-space :size="8" wrap>
              <el-select v-model="activeKey" size="small" class="w-input-lg" filterable placeholder="选择 Key" :disabled="dataKeys.length === 0">
                <el-option v-for="k in dataKeys" :key="k" :label="k" :value="k" />
              </el-select>
              <el-tag v-if="activeFormat === 'json'" size="small" type="info">JSON</el-tag>
              <el-tooltip content="复制" placement="top">
                <el-button size="small" :icon="CopyDocument" circle :disabled="!activeViewText" @click="copyText(activeViewText)" />
              </el-tooltip>
              <el-tooltip content="下载" placement="top">
                <el-button size="small" :icon="Download" circle :disabled="!activeViewText" @click="downloadActiveKey" />
              </el-tooltip>
              <el-tooltip content="搜索" placement="top">
                <el-button size="small" :icon="Search" circle :disabled="!activeViewText" @click="viewerRef?.openSearch()" />
              </el-tooltip>
              <el-tooltip content="折叠全部" placement="top">
                <el-button size="small" :icon="Fold" circle :disabled="!activeViewText" @click="viewerRef?.foldAll()" />
              </el-tooltip>
              <el-tooltip content="展开全部" placement="top">
                <el-button size="small" :icon="Expand" circle :disabled="!activeViewText" @click="viewerRef?.unfoldAll()" />
              </el-tooltip>
              <el-switch v-model="wrap" inline-prompt active-text="换行" inactive-text="单行" />
              <el-switch v-model="lineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
              <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
              </el-tooltip>
            </el-space>
          </div>
          <CodeMirrorViewer
            ref="viewerRef"
            :text="activeViewText"
            :language="activeFormat === 'json' ? 'json' : 'text'"
            :theme="props.editorTheme"
            :wrap="wrap"
            :line-numbers="lineNumbers"
            height="100%"
            class="k8s-detail-box k8s-detail-box--fill"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="关联资源" name="related">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Pods（引用该 ConfigMap）</div>
                <div class="k8s-section-actions">
                  <el-space :size="8">
                    <el-tag v-if="relatedLoading" size="small" type="info" effect="light">加载中</el-tag>
                    <el-tag v-else size="small" type="info" effect="light">共 {{ relatedPods.length }} 条</el-tag>
                    <el-tooltip content="刷新" placement="top">
                      <el-button size="small" :icon="RefreshRight" circle :loading="relatedLoading" @click="loadRelated" />
                    </el-tooltip>
                  </el-space>
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
    </el-tabs>
  </WorkloadDetailDrawerShell>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { RefreshRight, CopyDocument, Search, Fold, Expand, Sunny, Moon, Download } from '@element-plus/icons-vue'
import WorkloadDetailDrawerShell from '@/features/k8s/components/WorkloadDetailDrawerShell.vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError } from '@/shared/utils/notify'
import { downloadBlob, sanitizeFileName } from '@/shared/utils/text'
import type { ApiError } from '@/shared/utils/error'
import type { ControllerRow, EventRow } from '../../ClusterManageView.utils'
import {
  collectControllersFromPodsRaw,
  formatAgeMs,
  formatTs,
  getEventTimeMs,
  getHttpStatus,
  getRowNamespace,
  mergeControllers,
  normalizeMultilineText,
  ownersListToText,
  podUsesConfigMap,
  toRelatedPodVmFromPod,
  tryPrettyJson,
  workloadUsesConfigMap
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
const tab = ref<'overview' | 'data' | 'related' | 'events'>('overview')
const row = ref<any>(null)
const activeKey = ref('')
const wrap = ref(true)
const lineNumbers = ref(false)
const viewerRef = ref<{ openSearch: () => void; foldAll: () => void; unfoldAll: () => void } | null>(null)

const detailNamespace = computed(() => getRowNamespace(row.value) || '')
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `ConfigMap 详情：${detailName.value}` : 'ConfigMap 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const dataKeys = computed(() => Object.keys((row.value?.data ?? {}) as Record<string, unknown>).map(String).sort())
const dataKeysCount = computed(() => dataKeys.value.length)
const immutableText = computed(() => (row.value?.immutable ? 'true' : 'false'))
const labels = computed(() => (row.value?.metadata?.labels ?? {}) as Record<string, string>)
const labelsCount = computed(() => Object.keys(labels.value ?? {}).length)
const labelsViewText = computed(() => JSON.stringify(labels.value ?? {}, null, 2))
const annotations = computed(() => (row.value?.metadata?.annotations ?? {}) as Record<string, string>)
const annotationsCount = computed(() => Object.keys(annotations.value ?? {}).length)
const annotationsViewText = computed(() => JSON.stringify(annotations.value ?? {}, null, 2))
const resourceVersionText = computed(() => String(row.value?.metadata?.resourceVersion ?? '-'))
const uidText = computed(() => String(row.value?.metadata?.uid ?? '-'))

const activeRawValue = computed(() => {
  const k = String(activeKey.value ?? ''); if (!row.value || !k) return ''
  const v = (row.value?.data ?? {})[k]; return v == null ? '' : typeof v === 'string' ? v : String(v)
})
const activePretty = computed(() => tryPrettyJson(activeRawValue.value))
const activeFormat = computed(() => (activePretty.value.ok ? 'json' : 'text'))
const activeViewText = computed(() => normalizeMultilineText(activePretty.value.text))

function downloadActiveKey() {
  if (!activeKey.value) return
  const text = normalizeMultilineText(activePretty.value.text); if (!text) return
  const ts = new Date().toISOString().replace(/[:.]/g, '-')
  downloadBlob(`configmap_${sanitizeFileName(`${detailNamespace.value}_${detailName.value}`)}_${sanitizeFileName(String(activeKey.value))}_${ts}.txt`, new Blob([text], { type: 'text/plain;charset=utf-8' }))
}

// ── related ──
const relatedPods = ref<any[]>([])
const relatedControllers = ref<ControllerRow[]>([])
const relatedLoading = ref(false)

// ── events ──
const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

async function loadRelated() {
  if (!props.clusterId || !row.value) return
  const ns = detailNamespace.value; const name = detailName.value
  if (!ns || !name) return
  relatedLoading.value = true
  try {
    const data = await k8sApi.getConfigMapRelated(props.clusterId, ns, name)
    const podsRaw: any[] = Array.isArray((data as any)?.pods) ? (data as any).pods : []
    relatedPods.value = podsRaw.map((p) => ({ ...p, ownersText: ownersListToText(p?.owners) })).sort((a, b) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`))
    const controllersRaw: any[] = Array.isArray((data as any)?.controllers) ? (data as any).controllers : []
    relatedControllers.value = controllersRaw.map((c) => ({ kind: String(c?.kind ?? '').trim(), namespace: ns, name: String(c?.name ?? '').trim() })).filter((c) => c.kind && c.name)
  } catch (e) {
    const err = e as ApiError
    const status = getHttpStatus(err)
    if (status === 404 || err.code === 4040 || err.code === 4000) {
      try {
        const data = await k8sApi.listPods(props.clusterId, { namespace: ns })
        const pods = (Array.isArray(data.list) ? data.list : []).filter((p: any) => podUsesConfigMap(p, name)).map((p: any) => toRelatedPodVmFromPod(p)).sort((a: any, b: any) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`))
        relatedPods.value = pods
        const cFromPods = collectControllersFromPodsRaw(pods.map((p: any) => p.rawPod ?? p))
        const w = await k8sApi.listWorkloads(props.clusterId, { namespace: ns })
        const cFromWl: ControllerRow[] = (Array.isArray(w.list) ? w.list : []).filter((it: any) => workloadUsesConfigMap(it, name)).map((it: any) => ({ kind: String(it?.kind ?? '').trim(), namespace: String(it?.metadata?.namespace ?? ns).trim(), name: String(it?.metadata?.name ?? '').trim() })).filter((c: ControllerRow) => c.kind && c.namespace && c.name)
        relatedControllers.value = mergeControllers(cFromPods, cFromWl)
      } catch (e2) { const err2 = e2 as ApiError; notifyError(err2.requestId ? `${err2.message} (request_id=${err2.requestId})` : err2.message) }
    } else { notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  } finally { relatedLoading.value = false }
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
      if (String(inv?.kind ?? '') && String(inv?.kind ?? '') !== 'ConfigMap') return false
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

async function refreshDetail() {
  if (!visible.value) return
  try {
    refreshing.value = true
    emit('refresh-list'); await new Promise(r => setTimeout(r, 300))
    const next = findRow(detailNamespace.value, detailName.value)
    if (next) row.value = next
    if (tab.value === 'events') await loadEvents()
    if (tab.value === 'related') await loadRelated()
  } finally { refreshing.value = false }
}

watch(
  () => [visible.value, tab.value, detailNamespace.value, detailName.value] as const,
  ([v, t]) => {
    if (!v) return
    if (t === 'events' && events.value.length === 0) void loadEvents()
    if (t === 'related' && relatedPods.value.length === 0) void loadRelated()
  }
)

watch(() => visible.value, (v) => {
  if (v) return
  tab.value = 'overview'; row.value = null; activeKey.value = ''; wrap.value = true; lineNumbers.value = false
  relatedPods.value = []; relatedControllers.value = []; events.value = []
})

watch(
  () => [visible.value, dataKeys.value.join('|')] as const,
  ([v]) => {
    if (!v) return
    if (activeKey.value && dataKeys.value.includes(activeKey.value)) return
    activeKey.value = dataKeys.value[0] ?? ''
  }
)

function open(r: any) {
  row.value = r; tab.value = 'overview'
  const keys = Object.keys(r?.data ?? {}); activeKey.value = keys.length ? String(keys[0]) : ''
  visible.value = true
}
function close() { visible.value = false }
defineExpose({ open, close })
</script>
