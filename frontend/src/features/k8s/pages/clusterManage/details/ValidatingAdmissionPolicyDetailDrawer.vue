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
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">FailurePolicy:</div><div class="k8s-v">{{ failurePolicyText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">ParamKind:</div><div class="k8s-v">{{ paramKindText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Validations:</div><div class="k8s-v">{{ validationsCount }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="failurePolicy">{{ failurePolicyText }}</el-descriptions-item>
              <el-descriptions-item label="paramKind">{{ paramKindText }}</el-descriptions-item>
              <el-descriptions-item label="validations">{{ validationsCount }}</el-descriptions-item>
              <el-descriptions-item label="auditAnnotations">{{ auditAnnotationsCount }}</el-descriptions-item>
              <el-descriptions-item label="variables">{{ variablesCount }}</el-descriptions-item>
              <el-descriptions-item label="matchConditions">{{ matchConditionsCount }}</el-descriptions-item>
              <el-descriptions-item label="labels">{{ labelsCount }}</el-descriptions-item>
              <el-descriptions-item label="annotations">{{ annotationsCount }}</el-descriptions-item>
            </el-descriptions>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Validations</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ validationRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="validationRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="expression" label="Expression" min-width="320" show-overflow-tooltip />
              <el-table-column prop="message" label="Message" min-width="240" show-overflow-tooltip />
              <el-table-column prop="reason" label="Reason" width="180" show-overflow-tooltip />
              <el-table-column prop="actions" label="Actions" width="120" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Match Constraints</div></template>
            <CodeMirrorViewer :text="matchConstraintsViewText" language="json" :theme="props.editorTheme" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Status</div></template>
            <CodeMirrorViewer :text="statusViewText" language="json" :theme="props.editorTheme" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
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
import { formatAgeMs, getEventTimeMs } from '../../ClusterManageView.utils'

type ValidationRow = {
  expression: string
  message: string
  reason: string
  actions: string
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

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'events'>('overview')
const row = ref<any>(null)

const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `ValidatingAdmissionPolicy 详情：${detailName.value}` : 'ValidatingAdmissionPolicy 详情'))
const failurePolicyText = computed(() => String(row.value?.spec?.failurePolicy ?? '-'))
const paramKindText = computed(() => {
  const apiVersion = String(row.value?.spec?.paramKind?.apiVersion ?? '').trim()
  const kind = String(row.value?.spec?.paramKind?.kind ?? '').trim()
  if (!apiVersion && !kind) return '-'
  return apiVersion && kind ? `${apiVersion}/${kind}` : apiVersion || kind
})
const validations = computed<any[]>(() => Array.isArray(row.value?.spec?.validations) ? row.value.spec.validations : [])
const validationsCount = computed(() => validations.value.length)
const auditAnnotationsCount = computed(() => Array.isArray(row.value?.spec?.auditAnnotations) ? row.value.spec.auditAnnotations.length : 0)
const variablesCount = computed(() => Array.isArray(row.value?.spec?.variables) ? row.value.spec.variables.length : 0)
const matchConditionsCount = computed(() => Array.isArray(row.value?.spec?.matchConditions) ? row.value.spec.matchConditions.length : 0)
const labelsCount = computed(() => Object.keys((row.value?.metadata?.labels ?? {}) as Record<string, string>).length)
const annotationsCount = computed(() => Object.keys((row.value?.metadata?.annotations ?? {}) as Record<string, string>).length)
const validationRows = computed<ValidationRow[]>(() => validations.value.map((item) => ({
  expression: String(item?.expression ?? '-'),
  message: String(item?.message ?? '-'),
  reason: String(item?.reason ?? '-'),
  actions: Array.isArray(item?.messageExpression) ? String(item.messageExpression.length) : '-'
})))
const matchConstraintsViewText = computed(() => JSON.stringify({
  paramKind: row.value?.spec?.paramKind ?? {},
  matchConstraints: row.value?.spec?.matchConstraints ?? {},
  matchConditions: row.value?.spec?.matchConditions ?? []
}, null, 2))
const statusViewText = computed(() => JSON.stringify({
  status: row.value?.status ?? {},
  auditAnnotations: row.value?.spec?.auditAnnotations ?? [],
  variables: row.value?.spec?.variables ?? []
}, null, 2))

const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const name = detailName.value
  if (!name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, {
      involved_object_kind: 'ValidatingAdmissionPolicy',
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
    const data = await k8sApi.listValidatingAdmissionPolicies(props.clusterId)
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