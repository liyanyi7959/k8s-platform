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
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Budget:</div><div class="k8s-v">{{ budgetText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Allowed:</div><div class="k8s-v">{{ disruptionsAllowedText }}</div></div>
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
              <el-descriptions-item label="budget">{{ budgetText }}</el-descriptions-item>
              <el-descriptions-item label="selector">{{ selectorText }}</el-descriptions-item>
              <el-descriptions-item label="匹配 Pods">{{ relatedPods.length }}</el-descriptions-item>
              <el-descriptions-item label="currentHealthy">{{ currentHealthyText }}</el-descriptions-item>
              <el-descriptions-item label="desiredHealthy">{{ desiredHealthyText }}</el-descriptions-item>
              <el-descriptions-item label="expectedPods">{{ expectedPodsText }}</el-descriptions-item>
              <el-descriptions-item label="disruptionsAllowed">{{ disruptionsAllowedText }}</el-descriptions-item>
              <el-descriptions-item label="disruptedPods">{{ disruptedPodsCount }}</el-descriptions-item>
              <el-descriptions-item label="uid">{{ uidText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Selector</div></template>
            <CodeMirrorViewer :text="selectorViewText" language="json" :theme="props.editorTheme" height="180px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Disrupted Pods</div></template>
            <CodeMirrorViewer :text="disruptedPodsViewText" language="json" :theme="props.editorTheme" height="180px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="关联 Pods" name="related">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Pods（匹配 Selector）</div>
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
              <el-table-column prop="node" label="Node" min-width="220" show-overflow-tooltip />
              <el-table-column prop="ownersText" label="Owners" min-width="240" show-overflow-tooltip />
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
import { computed, ref, watch } from 'vue'
import { RefreshRight } from '@element-plus/icons-vue'

import WorkloadDetailDrawerShell from '@/features/k8s/components/WorkloadDetailDrawerShell.vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import type { ApiError } from '@/shared/utils/error'
import { notifyError } from '@/shared/utils/notify'
import type { EventRow } from '../../ClusterManageView.utils'
import { formatAgeMs, formatTs, getEventTimeMs, getRowNamespace, normalizeLabelRecord, toRelatedPodVmFromPod } from '../../ClusterManageView.utils'

type MatchExpression = {
  key?: unknown
  operator?: unknown
  values?: unknown
}

const props = defineProps<{
  clusterId: number
  editorTheme: 'auto' | 'light' | 'dark'
  editorThemeEffectiveDark: boolean
  list: any[]
}>()

const emit = defineEmits<{
  (e: 'open-related-pod', row: any): void
  (e: 'refresh-list'): void
}>()

function findRow(ns: string, name: string): any | null {
  for (const item of props.list ?? []) {
    if (getRowNamespace(item) === ns && String(item?.metadata?.name ?? '') === name) return item
  }
  return null
}

function matchExpression(labels: Record<string, string>, expression: MatchExpression): boolean {
  const key = String(expression?.key ?? '').trim()
  const operator = String(expression?.operator ?? '').trim()
  const values = Array.isArray(expression?.values)
    ? expression.values.map((item: unknown) => String(item ?? '').trim()).filter(Boolean)
    : []
  if (!key || !operator) return true
  const current = labels[key]
  switch (operator) {
    case 'In':
      return current != null && values.includes(current)
    case 'NotIn':
      return current == null || !values.includes(current)
    case 'Exists':
      return current != null
    case 'DoesNotExist':
      return current == null
    default:
      return true
  }
}

function matchesPodSelector(pod: any, selector: any): boolean {
  const labels = normalizeLabelRecord(pod?.metadata?.labels)
  const matchLabels = normalizeLabelRecord(selector?.matchLabels)
  const expressions: MatchExpression[] = Array.isArray(selector?.matchExpressions) ? selector.matchExpressions : []
  const hasLabels = Object.keys(matchLabels).length > 0
  const hasExpressions = expressions.length > 0
  if (!hasLabels && !hasExpressions) return true
  for (const [key, value] of Object.entries(matchLabels)) {
    if (labels[key] !== value) return false
  }
  return expressions.every((expression) => matchExpression(labels, expression))
}

function formatSelectorText(selector: any): string {
  const labels = normalizeLabelRecord(selector?.matchLabels)
  const labelEntries = Object.entries(labels).map(([key, value]) => `${key}=${value}`)
  const expressions: MatchExpression[] = Array.isArray(selector?.matchExpressions) ? selector.matchExpressions : []
  const expressionEntries = expressions
    .map((expression) => {
      const key = String(expression?.key ?? '').trim()
      const operator = String(expression?.operator ?? '').trim()
      const values = Array.isArray(expression?.values)
        ? expression.values.map((item: unknown) => String(item ?? '').trim()).filter(Boolean)
        : []
      if (!key || !operator) return ''
      if (operator === 'In' || operator === 'NotIn') return `${key} ${operator} (${values.join(', ') || '-'})`
      return `${key} ${operator}`
    })
    .filter(Boolean)
  const parts = [...labelEntries, ...expressionEntries]
  return parts.length ? parts.join(', ') : 'all pods'
}

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'related' | 'events'>('overview')
const row = ref<any>(null)

const detailNamespace = computed(() => getRowNamespace(row.value) || '')
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `PDB 详情：${detailName.value}` : 'PDB 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const budgetText = computed(() => {
  const min = row.value?.spec?.minAvailable
  const max = row.value?.spec?.maxUnavailable
  if (min != null && String(min) !== '') return `min=${String(min)}`
  if (max != null && String(max) !== '') return `max=${String(max)}`
  return '-'
})
const selector = computed(() => row.value?.spec?.selector ?? {})
const selectorText = computed(() => formatSelectorText(selector.value))
const selectorViewText = computed(() => JSON.stringify(selector.value ?? {}, null, 2))
const currentHealthyText = computed(() => String(row.value?.status?.currentHealthy ?? 0))
const desiredHealthyText = computed(() => String(row.value?.status?.desiredHealthy ?? 0))
const expectedPodsText = computed(() => String(row.value?.status?.expectedPods ?? 0))
const disruptionsAllowedText = computed(() => String(row.value?.status?.disruptionsAllowed ?? 0))
const disruptedPods = computed(() => (row.value?.status?.disruptedPods ?? {}) as Record<string, string>)
const disruptedPodsCount = computed(() => Object.keys(disruptedPods.value ?? {}).length)
const disruptedPodsViewText = computed(() => JSON.stringify(disruptedPods.value ?? {}, null, 2))
const uidText = computed(() => String(row.value?.metadata?.uid ?? '-'))

const relatedPods = ref<any[]>([])
const relatedLoading = ref(false)
const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

async function loadRelated() {
  if (!props.clusterId || !row.value) return
  const namespace = detailNamespace.value
  if (!namespace) return
  relatedLoading.value = true
  try {
    const data = await k8sApi.listPods(props.clusterId, { namespace })
    relatedPods.value = (Array.isArray(data.list) ? data.list : [])
      .filter((pod: any) => matchesPodSelector(pod, selector.value))
      .map((pod: any) => toRelatedPodVmFromPod(pod))
      .sort((a: any, b: any) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`, 'zh-Hans-CN'))
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    relatedLoading.value = false
  }
}

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const namespace = detailNamespace.value
  const name = detailName.value
  if (!namespace || !name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, {
      namespace,
      involved_object_kind: 'PodDisruptionBudget',
      involved_object_name: name,
      involved_object_uid: String(row.value?.metadata?.uid ?? '').trim() || undefined
    })
    const now = Date.now()
    events.value = (Array.isArray(data.list) ? data.list : [])
      .map((event) => {
        const timeMs = getEventTimeMs(event)
        return {
          tMs: timeMs ?? -1,
          type: String(event?.type ?? '') || '-',
          reason: String(event?.reason ?? '') || '-',
          message: String(event?.message ?? '') || '-',
          count: Number(event?.count ?? event?.series?.count ?? 1) || 1,
          lastSeen: timeMs != null ? formatAgeMs(Math.max(0, now - timeMs)) : '-'
        }
      })
      .sort((a, b) => b.tMs - a.tMs)
      .map(({ tMs: _timeMs, ...rest }) => rest)
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    eventsLoading.value = false
  }
}

async function refreshDetail() {
  if (!visible.value || !props.clusterId) return
  const namespace = detailNamespace.value
  const name = detailName.value
  if (!namespace || !name) return
  try {
    refreshing.value = true
    const data = await k8sApi.listPDBs(props.clusterId, { namespace })
    const next = (Array.isArray(data.list) ? data.list : []).find((item: any) => String(item?.metadata?.name ?? '') === name)
    if (next) row.value = next
    else {
      emit('refresh-list')
      row.value = findRow(namespace, name)
    }
    if (tab.value === 'related') await loadRelated()
    if (tab.value === 'events') await loadEvents()
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    refreshing.value = false
  }
}

watch(
  () => [visible.value, tab.value] as const,
  ([isVisible, currentTab]) => {
    if (!isVisible) return
    if (currentTab === 'related' && relatedPods.value.length === 0) void loadRelated()
    if (currentTab === 'events' && events.value.length === 0) void loadEvents()
  }
)

watch(() => visible.value, (isVisible) => {
  if (isVisible) return
  tab.value = 'overview'
  row.value = null
  relatedPods.value = []
  events.value = []
})

function open(targetRow: any) {
  row.value = targetRow
  tab.value = 'overview'
  visible.value = true
}

function close() {
  visible.value = false
}

defineExpose({ open, close })
</script>