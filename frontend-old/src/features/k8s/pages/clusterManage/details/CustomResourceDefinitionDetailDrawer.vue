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
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">Group:</div><div class="k8s-v">{{ groupText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Scope:</div><div class="k8s-v">{{ scopeText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Versions:</div><div class="k8s-v">{{ storageVersionsText }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="group">{{ groupText }}</el-descriptions-item>
              <el-descriptions-item label="scope">{{ scopeText }}</el-descriptions-item>
              <el-descriptions-item label="kind">{{ kindText }}</el-descriptions-item>
              <el-descriptions-item label="plural">{{ pluralText }}</el-descriptions-item>
              <el-descriptions-item label="singular">{{ singularText }}</el-descriptions-item>
              <el-descriptions-item label="listKind">{{ listKindText }}</el-descriptions-item>
              <el-descriptions-item label="versions">{{ versionsCount }}</el-descriptions-item>
              <el-descriptions-item label="storedVersions">{{ storageVersionsText }}</el-descriptions-item>
              <el-descriptions-item label="labels">{{ labelsCount }}</el-descriptions-item>
              <el-descriptions-item label="annotations">{{ annotationsCount }}</el-descriptions-item>
              <el-descriptions-item label="uid">{{ uidText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Names</div></template>
            <CodeMirrorViewer :text="namesViewText" language="json" :theme="props.editorTheme" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Conversion</div></template>
            <CodeMirrorViewer :text="conversionViewText" language="json" :theme="props.editorTheme" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="Versions" name="versions">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">版本明细</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ versionRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="versionRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="Version" width="140" show-overflow-tooltip />
              <el-table-column prop="served" label="Served" width="90" align="center" header-align="center" />
              <el-table-column prop="storage" label="Storage" width="90" align="center" header-align="center" />
              <el-table-column prop="deprecated" label="Deprecated" width="110" align="center" header-align="center" />
              <el-table-column prop="schema" label="Schema" width="90" align="center" header-align="center" />
              <el-table-column prop="subresources" label="Subresources" min-width="160" show-overflow-tooltip />
              <el-table-column prop="printerColumns" label="PrinterCols" width="110" align="center" header-align="center" />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">状态条件</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ conditionRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="conditionRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="type" label="Type" width="180" show-overflow-tooltip />
              <el-table-column prop="status" label="Status" width="100" show-overflow-tooltip />
              <el-table-column prop="reason" label="Reason" width="180" show-overflow-tooltip />
              <el-table-column prop="message" label="Message" min-width="320" show-overflow-tooltip />
              <el-table-column prop="lastTransitionTime" label="LastTransitionTime" width="180" show-overflow-tooltip />
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
import { formatAgeMs, formatTs, getEventTimeMs } from '../../ClusterManageView.utils'

type VersionRow = {
  name: string
  served: string
  storage: string
  deprecated: string
  schema: string
  subresources: string
  printerColumns: number
}

type ConditionRow = {
  type: string
  status: string
  reason: string
  message: string
  lastTransitionTime: string
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

function findRow(name: string): any | null {
  for (const item of props.list ?? []) {
    if (String(item?.metadata?.name ?? '') === name) return item
  }
  return null
}

function getStorageVersions(row: any): string[] {
  const stored = Array.isArray(row?.status?.storedVersions) ? row.status.storedVersions : []
  const storedText = stored.map((item: unknown) => String(item ?? '').trim()).filter(Boolean)
  if (storedText.length) return storedText
  const versions: any[] = Array.isArray(row?.spec?.versions) ? row.spec.versions : []
  return versions.map((item) => item?.storage ? String(item?.name ?? '').trim() : '').filter(Boolean)
}

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'versions' | 'events'>('overview')
const row = ref<any>(null)

const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `CRD 详情：${detailName.value}` : 'CRD 详情'))
const groupText = computed(() => String(row.value?.spec?.group ?? '-'))
const scopeText = computed(() => String(row.value?.spec?.scope ?? '-'))
const kindText = computed(() => String(row.value?.spec?.names?.kind ?? '-'))
const pluralText = computed(() => String(row.value?.spec?.names?.plural ?? '-'))
const singularText = computed(() => String(row.value?.spec?.names?.singular ?? '-'))
const listKindText = computed(() => String(row.value?.spec?.names?.listKind ?? '-'))
const versionsCount = computed(() => Array.isArray(row.value?.spec?.versions) ? row.value.spec.versions.length : 0)
const storageVersionsText = computed(() => getStorageVersions(row.value).join(', ') || '-')
const labels = computed(() => (row.value?.metadata?.labels ?? {}) as Record<string, string>)
const labelsCount = computed(() => Object.keys(labels.value ?? {}).length)
const annotations = computed(() => (row.value?.metadata?.annotations ?? {}) as Record<string, string>)
const annotationsCount = computed(() => Object.keys(annotations.value ?? {}).length)
const uidText = computed(() => String(row.value?.metadata?.uid ?? '-'))
const namesViewText = computed(() => JSON.stringify(row.value?.spec?.names ?? {}, null, 2))
const conversionViewText = computed(() => JSON.stringify(row.value?.spec?.conversion ?? {}, null, 2))
const versionRows = computed<VersionRow[]>(() => {
  const versions: any[] = Array.isArray(row.value?.spec?.versions) ? row.value.spec.versions : []
  return versions.map((item) => {
    const subresources = item?.subresources
    return {
      name: String(item?.name ?? '-'),
      served: item?.served ? 'yes' : 'no',
      storage: item?.storage ? 'yes' : 'no',
      deprecated: item?.deprecated ? 'yes' : 'no',
      schema: item?.schema?.openAPIV3Schema ? 'yes' : 'no',
      subresources: subresources && typeof subresources === 'object' && !Array.isArray(subresources)
        ? Object.keys(subresources).join(', ') || '-'
        : '-',
      printerColumns: Array.isArray(item?.additionalPrinterColumns) ? item.additionalPrinterColumns.length : 0
    }
  })
})
const conditionRows = computed<ConditionRow[]>(() => {
  const conditions: any[] = Array.isArray(row.value?.status?.conditions) ? row.value.status.conditions : []
  return conditions.map((item) => ({
    type: String(item?.type ?? '-'),
    status: String(item?.status ?? '-'),
    reason: String(item?.reason ?? '-'),
    message: String(item?.message ?? '-'),
    lastTransitionTime: formatTs(item?.lastTransitionTime)
  }))
})

const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const name = detailName.value
  if (!name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, {
      involved_object_kind: 'CustomResourceDefinition',
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
  const name = detailName.value
  if (!name) return
  try {
    refreshing.value = true
    const data = await k8sApi.listCustomResourceDefinitions(props.clusterId)
    const next = (Array.isArray(data.list) ? data.list : []).find((item: any) => String(item?.metadata?.name ?? '') === name)
    if (next) row.value = next
    else {
      emit('refresh-list')
      row.value = findRow(name)
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