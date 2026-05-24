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
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">类型:</div><div class="k8s-v">{{ detailKind }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Webhooks:</div><div class="k8s-v">{{ webhookCount }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Rules:</div><div class="k8s-v">{{ rulesCount }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="kind">{{ detailKind }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ createdAtText }}</el-descriptions-item>
              <el-descriptions-item label="webhooks">{{ webhookCount }}</el-descriptions-item>
              <el-descriptions-item label="rules">{{ rulesCount }}</el-descriptions-item>
              <el-descriptions-item label="failurePolicy">{{ failurePolicyText }}</el-descriptions-item>
              <el-descriptions-item label="labels">{{ labelsCount }}</el-descriptions-item>
              <el-descriptions-item label="annotations">{{ annotationsCount }}</el-descriptions-item>
              <el-descriptions-item label="uid">{{ uidText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Webhook 明细</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ webhookRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="webhookRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="Name" min-width="220" show-overflow-tooltip />
              <el-table-column prop="rules" label="Rules" width="90" align="center" header-align="center" />
              <el-table-column prop="failurePolicy" label="FailurePolicy" width="150" show-overflow-tooltip />
              <el-table-column prop="sideEffects" label="SideEffects" width="150" show-overflow-tooltip />
              <el-table-column prop="timeout" label="Timeout" width="100" align="center" header-align="center" />
              <el-table-column prop="client" label="Client" min-width="260" show-overflow-tooltip />
              <el-table-column prop="selectors" label="Selectors" min-width="150" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Raw Webhooks</div></template>
            <CodeMirrorViewer :text="webhooksViewText" language="json" :theme="props.editorTheme" height="260px" class="k8s-detail-box k8s-detail-box--fill" />
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

type DetailKind = 'ValidatingWebhookConfiguration' | 'MutatingWebhookConfiguration'

type WebhookRow = {
  name: string
  rules: number
  failurePolicy: string
  sideEffects: string
  timeout: string
  client: string
  selectors: string
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

function getClientText(item: any): string {
  const svcNs = String(item?.clientConfig?.service?.namespace ?? '').trim()
  const svcName = String(item?.clientConfig?.service?.name ?? '').trim()
  const svcPath = String(item?.clientConfig?.service?.path ?? '').trim()
  const url = String(item?.clientConfig?.url ?? '').trim()
  if (svcName) return `${svcNs || '-'}/${svcName}${svcPath ? svcPath : ''}`
  return url || '-'
}

function getSelectorCount(selector: any): number {
  const labels = selector?.matchLabels
  const expr = Array.isArray(selector?.matchExpressions) ? selector.matchExpressions.length : 0
  const labelCount = labels && typeof labels === 'object' && !Array.isArray(labels)
    ? Object.keys(labels as Record<string, unknown>).length
    : 0
  return labelCount + expr
}

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'events'>('overview')
const row = ref<any>(null)
const detailKind = ref<DetailKind>('ValidatingWebhookConfiguration')

const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `${detailKind.value} 详情：${detailName.value}` : `${detailKind.value} 详情`))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const uidText = computed(() => String(row.value?.metadata?.uid ?? '-'))
const labelsCount = computed(() => Object.keys((row.value?.metadata?.labels ?? {}) as Record<string, string>).length)
const annotationsCount = computed(() => Object.keys((row.value?.metadata?.annotations ?? {}) as Record<string, string>).length)
const webhooks = computed<any[]>(() => Array.isArray(row.value?.webhooks) ? row.value.webhooks : [])
const webhookCount = computed(() => webhooks.value.length)
const rulesCount = computed(() => webhooks.value.reduce((sum, item) => sum + (Array.isArray(item?.rules) ? item.rules.length : 0), 0))
const failurePolicyText = computed(() => {
  const values = Array.from(new Set(webhooks.value.map((item) => String(item?.failurePolicy ?? '').trim()).filter(Boolean)))
  return values.join(', ') || '-'
})
const webhookRows = computed<WebhookRow[]>(() => webhooks.value.map((item) => ({
  name: String(item?.name ?? '-'),
  rules: Array.isArray(item?.rules) ? item.rules.length : 0,
  failurePolicy: String(item?.failurePolicy ?? '-'),
  sideEffects: String(item?.sideEffects ?? '-'),
  timeout: item?.timeoutSeconds != null ? `${String(item.timeoutSeconds)}s` : '-',
  client: getClientText(item),
  selectors: `ns=${getSelectorCount(item?.namespaceSelector)}, obj=${getSelectorCount(item?.objectSelector)}`
})))
const webhooksViewText = computed(() => JSON.stringify(row.value?.webhooks ?? [], null, 2))

const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

async function loadEvents() {
  if (!props.clusterId || !row.value) return
  const name = detailName.value
  if (!name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, {
      involved_object_kind: detailKind.value,
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
    const data = detailKind.value === 'ValidatingWebhookConfiguration'
      ? await k8sApi.listValidatingWebhookConfigurations(props.clusterId)
      : await k8sApi.listMutatingWebhookConfigurations(props.clusterId)
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
  detailKind.value = 'ValidatingWebhookConfiguration'
})

function open(targetRow: any, kind: DetailKind) {
  row.value = targetRow
  detailKind.value = kind
  tab.value = 'overview'
  visible.value = true
}

function close() {
  visible.value = false
}

defineExpose({ open, close })
</script>