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
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">配额项:</div><div class="k8s-v">{{ quotaRows.length }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">最高使用率:</div><div class="k8s-v">{{ peakUsageText }}</div></div>
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
              <el-descriptions-item label="配额项">{{ quotaRows.length }}</el-descriptions-item>
              <el-descriptions-item label="scopes">{{ scopesText }}</el-descriptions-item>
              <el-descriptions-item label="scopeSelector">{{ scopeSelectorCountText }}</el-descriptions-item>
              <el-descriptions-item label="labels">{{ labelsCount }}</el-descriptions-item>
              <el-descriptions-item label="annotations">{{ annotationsCount }}</el-descriptions-item>
              <el-descriptions-item label="uid">{{ uidText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Scope Selector</div></template>
            <CodeMirrorViewer :text="scopeSelectorViewText" language="json" :theme="props.editorTheme" height="180px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Labels</div></template>
            <CodeMirrorViewer :text="labelsViewText" language="json" :theme="props.editorTheme" height="180px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Annotations</div></template>
            <CodeMirrorViewer :text="annotationsViewText" language="json" :theme="props.editorTheme" height="180px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="配额使用" name="usage">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Hard / Used</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ quotaRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="quotaRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="key" label="资源" min-width="220" show-overflow-tooltip />
              <el-table-column prop="usedText" label="已使用" min-width="120" show-overflow-tooltip />
              <el-table-column prop="hardText" label="上限" min-width="120" show-overflow-tooltip />
              <el-table-column label="使用率" min-width="220">
                <template #default="{ row }">
                  <div class="quota-progress-cell">
                    <div class="quota-progress-head">
                      <span class="quota-progress-label">{{ formatQuotaPercent(row.percent) }}</span>
                    </div>
                    <el-progress
                      :percentage="capQuotaPercent(row.percent)"
                      :stroke-width="8"
                      :show-text="false"
                      :color="getQuotaProgressColor(row.percent)"
                    />
                  </div>
                </template>
              </el-table-column>
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
import { parseSiValue } from '@/shared/utils/parseSiValue'
import type { EventRow } from '../../ClusterManageView.utils'
import { formatAgeMs, formatTs, getEventTimeMs, getRowNamespace } from '../../ClusterManageView.utils'

type QuotaRow = {
  key: string
  usedText: string
  hardText: string
  percent: number | null
}

const props = defineProps<{
  clusterId: number
  editorTheme: 'auto' | 'light' | 'dark'
  editorThemeEffectiveDark: boolean
  list: any[]
}>()

const emit = defineEmits<{
  (e: 'refresh-list'): void
}>()

function findRow(ns: string, name: string): any | null {
  for (const item of props.list ?? []) {
    if (getRowNamespace(item) === ns && String(item?.metadata?.name ?? '') === name) return item
  }
  return null
}

function capQuotaPercent(percent: number | null): number {
  if (percent == null || !Number.isFinite(percent)) return 0
  return Math.max(0, Math.min(percent, 100))
}

function formatQuotaPercent(percent: number | null): string {
  if (percent == null || !Number.isFinite(percent)) return '-'
  return `${percent >= 100 ? percent.toFixed(0) : percent.toFixed(1)}%`
}

function getQuotaProgressColor(percent: number | null): string {
  if (percent != null && percent > 95) return '#dc2626'
  if (percent != null && percent > 80) return '#f59e0b'
  return '#16a34a'
}

function getQuotaRowsFromRow(source: any): QuotaRow[] {
  const hard = source?.status?.hard ?? source?.spec?.hard ?? {}
  const used = source?.status?.used ?? {}
  if (!hard || typeof hard !== 'object' || Array.isArray(hard)) return []
  return Object.keys(hard)
    .map((key) => {
      const hardText = String(hard?.[key] ?? '')
      const usedText = String(used?.[key] ?? '0')
      const hardValue = parseSiValue(hardText)
      const usedValue = parseSiValue(usedText)
      const percent = Number.isFinite(hardValue) && hardValue > 0 && Number.isFinite(usedValue)
        ? (usedValue / hardValue) * 100
        : null
      return { key, usedText, hardText, percent }
    })
    .sort((a, b) => (b.percent ?? -1) - (a.percent ?? -1) || a.key.localeCompare(b.key, 'zh-Hans-CN'))
}

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'usage' | 'events'>('overview')
const row = ref<any>(null)

const detailNamespace = computed(() => getRowNamespace(row.value) || '')
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `ResourceQuota 详情：${detailName.value}` : 'ResourceQuota 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const quotaRows = computed(() => getQuotaRowsFromRow(row.value))
const peakUsageText = computed(() => quotaRows.value[0] ? `${quotaRows.value[0].key} ${formatQuotaPercent(quotaRows.value[0].percent)}` : '-')
const scopes = computed(() => Array.isArray(row.value?.spec?.scopes) ? row.value.spec.scopes.map((item: unknown) => String(item ?? '').trim()).filter(Boolean) : [])
const scopesText = computed(() => (scopes.value.length ? scopes.value.join(', ') : '-'))
const scopeSelectorExpressions = computed(() => Array.isArray(row.value?.spec?.scopeSelector?.matchExpressions) ? row.value.spec.scopeSelector.matchExpressions : [])
const scopeSelectorCountText = computed(() => (scopeSelectorExpressions.value.length ? `${scopeSelectorExpressions.value.length} 条表达式` : '-'))
const scopeSelectorViewText = computed(() => JSON.stringify(row.value?.spec?.scopeSelector ?? {}, null, 2))
const labels = computed(() => (row.value?.metadata?.labels ?? {}) as Record<string, string>)
const labelsCount = computed(() => Object.keys(labels.value ?? {}).length)
const labelsViewText = computed(() => JSON.stringify(labels.value ?? {}, null, 2))
const annotations = computed(() => (row.value?.metadata?.annotations ?? {}) as Record<string, string>)
const annotationsCount = computed(() => Object.keys(annotations.value ?? {}).length)
const annotationsViewText = computed(() => JSON.stringify(annotations.value ?? {}, null, 2))
const uidText = computed(() => String(row.value?.metadata?.uid ?? '-'))

const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const namespace = detailNamespace.value
  const name = detailName.value
  if (!namespace || !name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, {
      namespace,
      involved_object_kind: 'ResourceQuota',
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
    const data = await k8sApi.listResourceQuotas(props.clusterId, { namespace })
    const next = (Array.isArray(data.list) ? data.list : []).find((item: any) => String(item?.metadata?.name ?? '') === name)
    if (next) row.value = next
    else {
      emit('refresh-list')
      row.value = findRow(namespace, name)
    }
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
    if (currentTab === 'events' && events.value.length === 0) void loadEvents()
  }
)

watch(() => visible.value, (isVisible) => {
  if (isVisible) return
  tab.value = 'overview'
  row.value = null
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

<style scoped>
.quota-progress-cell {
  min-width: 0;
}

.quota-progress-head {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 4px;
}

.quota-progress-label {
  font-size: 12px;
  color: var(--el-text-color-regular);
  font-variant-numeric: tabular-nums;
}
</style>