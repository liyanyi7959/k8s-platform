<template>
  <WorkloadDetailDrawerShell
    v-model="visible"
    :title="detailTitle"
    :loading="refreshing"
    ns="-"
    :name="detailName"
    @refresh="refreshDetail"
  >
    <template #summary>
      <div class="k8s-summary-grid">
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">名称:</div><div class="k8s-v">{{ detailName }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Phase:</div><div class="k8s-v"><span class="k8s-link">{{ phaseText }}</span></div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">StorageClass:</div><div class="k8s-v">{{ storageClassText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Claim:</div><div class="k8s-v">{{ claimText }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ createdAtText }}</el-descriptions-item>
              <el-descriptions-item label="phase">{{ phaseText }}</el-descriptions-item>
              <el-descriptions-item label="storageClass">{{ storageClassText }}</el-descriptions-item>
              <el-descriptions-item label="capacity">{{ capacityText }}</el-descriptions-item>
              <el-descriptions-item label="reclaimPolicy">{{ reclaimText }}</el-descriptions-item>
              <el-descriptions-item label="claimRef">{{ claimText }}</el-descriptions-item>
              <el-descriptions-item label="accessModes">{{ accessModesText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Labels</div></template>
            <CodeMirrorViewer :text="labelsViewText" language="json" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="关联资源" name="related">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">PVC（ClaimRef）</div>
                <div class="k8s-section-actions">
                  <el-space :size="8">
                    <el-tag v-if="relatedLoading" size="small" type="info" effect="light">加载中</el-tag>
                    <el-tag v-else size="small" type="info" effect="light">共 {{ relatedPvcs.length }} 条</el-tag>
                    <el-tooltip content="刷新" placement="top">
                      <el-button size="small" :icon="RefreshRight" circle :loading="relatedLoading" @click="loadRelated" />
                    </el-tooltip>
                  </el-space>
                </div>
              </div>
            </template>
            <el-table :data="relatedPvcs" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="名称" min-width="240" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
              <el-table-column prop="phase" label="Phase" width="140" show-overflow-tooltip />
              <el-table-column prop="storageClass" label="StorageClass" min-width="220" show-overflow-tooltip />
              <el-table-column prop="volumeName" label="Volume" min-width="220" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Pods（使用该 PV 对应 PVC）</div>
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
                <div class="k8s-section-title">资源（PVC OwnerReferences / Pod Owners）</div>
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

      <el-tab-pane label="YAML配置" name="yaml">
        <div class="k8s-tab-pane">
          <div class="k8s-pane-toolbar">
            <el-space :size="8" wrap>
              <el-tooltip content="复制" placement="top">
                <el-button size="small" :icon="CopyDocument" circle :disabled="!yamlViewText" @click="copyText(yamlViewText)" />
              </el-tooltip>
              <el-tooltip content="搜索" placement="top">
                <el-button size="small" :icon="Search" circle :disabled="!yamlViewText" @click="yamlViewerRef?.openSearch()" />
              </el-tooltip>
              <el-tooltip content="折叠全部" placement="top">
                <el-button size="small" :icon="Fold" circle :disabled="!yamlViewText" @click="yamlViewerRef?.foldAll()" />
              </el-tooltip>
              <el-tooltip content="展开全部" placement="top">
                <el-button size="small" :icon="Expand" circle :disabled="!yamlViewText" @click="yamlViewerRef?.unfoldAll()" />
              </el-tooltip>
              <el-switch v-model="yamlWrap" inline-prompt active-text="换行" inactive-text="单行" />
              <el-switch v-model="yamlLineNumbers" inline-prompt active-text="行号" inactive-text="无行号" />
              <el-tooltip :content="props.editorThemeEffectiveDark ? '浅色' : '深色'" placement="top">
                <el-button size="small" :icon="props.editorThemeEffectiveDark ? Sunny : Moon" circle @click="emit('toggle-editor-theme')" />
              </el-tooltip>
              <el-tooltip content="刷新" placement="top">
                <el-button size="small" :icon="RefreshRight" circle :loading="yamlLoading" @click="loadYaml" />
              </el-tooltip>
            </el-space>
          </div>
          <CodeMirrorViewer
            ref="yamlViewerRef"
            :text="yamlViewText"
            language="yaml"
            :theme="props.editorTheme"
            :wrap="yamlWrap"
            :line-numbers="yamlLineNumbers"
            height="100%"
            class="k8s-detail-box k8s-detail-box--fill"
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
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import type { ControllerRow, EventRow, PvcListRowVm } from '../../ClusterManageView.utils'
import {
  collectControllersFromPodsRaw,
  formatAgeMs,
  formatPvcClaimRefText,
  formatTs,
  getEventTimeMs,
  mergeControllers,
  normalizeMultilineText,
  podUsesPvc,
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

function findRow(name: string): any | null {
  for (const it of props.list ?? []) {
    if (String(it?.metadata?.name ?? '') === name) return it
  }
  return null
}

// ── state ──
const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'related' | 'events' | 'yaml'>('overview')
const row = ref<any>(null)

const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `PV 详情：${detailName.value}` : 'PV 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const phaseText = computed(() => String(row.value?.status?.phase ?? '-'))
const storageClassText = computed(() => String(row.value?.spec?.storageClassName ?? '-'))
const capacityText = computed(() => String(row.value?.spec?.capacity?.storage ?? '-'))
const reclaimText = computed(() => String(row.value?.spec?.persistentVolumeReclaimPolicy ?? '-'))
const accessModesText = computed(() => {
  const arr: any[] = Array.isArray(row.value?.spec?.accessModes) ? row.value.spec.accessModes : []
  return arr.map((it) => String(it ?? '').trim()).filter(Boolean).join(', ') || '-'
})
const claimText = computed(() => formatPvcClaimRefText(row.value))
const labelsViewText = computed(() => JSON.stringify(row.value?.metadata?.labels ?? {}, null, 2))

// ── related ──
const relatedPvcs = ref<PvcListRowVm[]>([])
const relatedPods = ref<any[]>([])
const relatedControllers = ref<ControllerRow[]>([])
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
  const name = detailName.value
  if (!name) return
  yamlLoading.value = true
  try { yamlText.value = (await k8sApi.getPVYaml(props.clusterId, name)).text ?? '' }
  catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { yamlLoading.value = false }
}

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const name = detailName.value
  const uid = String(row.value?.metadata?.uid ?? '').trim()
  if (!name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, {})
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const filtered = list.filter((ev) => {
      const inv = ev?.involvedObject ?? ev?.regarding ?? {}
      if (String(inv?.kind ?? '') && String(inv?.kind ?? '') !== 'PersistentVolume') return false
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
  const pvName = detailName.value
  if (!pvName) return
  relatedLoading.value = true
  try {
    relatedPvcs.value = []; relatedPods.value = []; relatedControllers.value = []
    const claimNs = String(row.value?.spec?.claimRef?.namespace ?? '').trim()
    const claimName = String(row.value?.spec?.claimRef?.name ?? '').trim()
    let pvcRow: any | null = null
    if (claimNs && claimName) {
      const pvcResp = await k8sApi.listPVCs(props.clusterId, { namespace: claimNs })
      const pvcList: any[] = Array.isArray(pvcResp.list) ? pvcResp.list : []
      pvcRow = pvcList.find((it) => String(it?.metadata?.name ?? '').trim() === claimName) ?? null
      if (pvcRow) {
        relatedPvcs.value = [{ namespace: claimNs, name: claimName, phase: String(pvcRow?.status?.phase ?? '-'), storageClass: String(pvcRow?.spec?.storageClassName ?? '-'), volumeName: String(pvcRow?.spec?.volumeName ?? pvName) || pvName }]
      }
      const podsResp = await k8sApi.listPods(props.clusterId, { namespace: claimNs })
      const pods = (Array.isArray(podsResp.list) ? podsResp.list : [])
        .filter((p: any) => podUsesPvc(p, claimName))
        .map((p: any) => toRelatedPodVmFromPod(p))
        .sort((a: any, b: any) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`))
      relatedPods.value = pods
      const ownerRefs: any[] = Array.isArray(pvcRow?.metadata?.ownerReferences) ? pvcRow.metadata.ownerReferences : []
      const ownersFromPvc: ControllerRow[] = ownerRefs.map((o) => ({ kind: String(o?.kind ?? '').trim(), namespace: claimNs, name: String(o?.name ?? '').trim() })).filter((c) => c.kind && c.namespace && c.name)
      const cFromPods = collectControllersFromPodsRaw(pods.map((p: any) => p.rawPod ?? p))
      relatedControllers.value = mergeControllers(ownersFromPvc, cFromPods)
    }
  } catch (e) { const err = e as ApiError; notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) }
  finally { relatedLoading.value = false }
}

async function refreshDetail() {
  if (!visible.value) return
  try {
    refreshing.value = true
    emit('refresh-list'); await new Promise(r => setTimeout(r, 300))
    const next = findRow(detailName.value)
    if (next) row.value = next
    if (tab.value === 'events') await loadEvents()
    if (tab.value === 'related') await loadRelated()
    if (tab.value === 'yaml') await loadYaml()
  } finally { refreshing.value = false }
}

watch(() => visible.value, (v) => {
  if (v) return
  tab.value = 'overview'; row.value = null
  relatedPvcs.value = []; relatedPods.value = []; relatedControllers.value = []
  events.value = []; yamlText.value = ''
})

watch(
  () => [visible.value, tab.value, detailName.value] as const,
  ([v, t]) => {
    if (!v) return
    if (t === 'events' && events.value.length === 0) void loadEvents()
    if (t === 'related' && relatedPvcs.value.length === 0 && relatedPods.value.length === 0 && relatedControllers.value.length === 0) void loadRelated()
    if (t === 'yaml' && !yamlText.value) void loadYaml()
  }
)

function open(r: any) { row.value = r; tab.value = 'overview'; visible.value = true }
function close() { visible.value = false }
defineExpose({ open, close })
</script>
