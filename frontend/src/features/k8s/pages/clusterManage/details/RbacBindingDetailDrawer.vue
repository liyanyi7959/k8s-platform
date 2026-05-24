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
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">Kind:</div><div class="k8s-v">{{ detailKind }}</div></div>
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">名称:</div><div class="k8s-v">{{ detailName }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">命名空间:</div><div class="k8s-v">{{ detailNamespace }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">RoleRef:</div><div class="k8s-v">{{ roleRefText }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="kind">{{ detailKind }}</el-descriptions-item>
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="命名空间">{{ detailNamespace }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ createdAtText }}</el-descriptions-item>
              <el-descriptions-item label="roleRef">{{ roleRefText }}</el-descriptions-item>
              <el-descriptions-item label="subjects">{{ subjectRows.length }}</el-descriptions-item>
              <el-descriptions-item label="labels">{{ labelsCount }}</el-descriptions-item>
              <el-descriptions-item label="annotations">{{ annotationsCount }}</el-descriptions-item>
              <el-descriptions-item label="uid">{{ uidText }}</el-descriptions-item>
            </el-descriptions>
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

      <el-tab-pane label="Subjects" name="subjects">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Subjects 明细</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ subjectRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="subjectRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="kind" label="Kind" width="160" show-overflow-tooltip />
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
              <el-table-column prop="apiGroup" label="API Group" min-width="160" show-overflow-tooltip />
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
import { formatAgeMs, formatTs, getEventTimeMs, getRowNamespace } from '../../ClusterManageView.utils'

type RbacBindingKind = 'RoleBinding' | 'ClusterRoleBinding'
type SubjectRow = {
  kind: string
  name: string
  namespace: string
  apiGroup: string
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

function findRow(kind: RbacBindingKind, namespace: string, name: string): any | null {
  for (const item of props.list ?? []) {
    const itemName = String(item?.metadata?.name ?? '')
    const itemNamespace = getRowNamespace(item) || ''
    if (itemName !== name) continue
    if (kind === 'ClusterRoleBinding') return item
    if (itemNamespace === namespace) return item
  }
  return null
}

function getRoleRefText(row: any): string {
  const kind = String(row?.roleRef?.kind ?? '').trim()
  const name = String(row?.roleRef?.name ?? '').trim()
  return kind && name ? `${kind}/${name}` : '-'
}

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'subjects' | 'events'>('overview')
const detailKind = ref<RbacBindingKind>('RoleBinding')
const row = ref<any>(null)

const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailNamespace = computed(() => getRowNamespace(row.value) || '-')
const detailTitle = computed(() => (detailName.value ? `${detailKind.value} 详情：${detailName.value}` : `${detailKind.value} 详情`))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const roleRefText = computed(() => getRoleRefText(row.value))
const labels = computed(() => (row.value?.metadata?.labels ?? {}) as Record<string, string>)
const labelsCount = computed(() => Object.keys(labels.value ?? {}).length)
const labelsViewText = computed(() => JSON.stringify(labels.value ?? {}, null, 2))
const annotations = computed(() => (row.value?.metadata?.annotations ?? {}) as Record<string, string>)
const annotationsCount = computed(() => Object.keys(annotations.value ?? {}).length)
const annotationsViewText = computed(() => JSON.stringify(annotations.value ?? {}, null, 2))
const uidText = computed(() => String(row.value?.metadata?.uid ?? '-'))
const subjectRows = computed<SubjectRow[]>(() => {
  const subjects = Array.isArray(row.value?.subjects) ? row.value.subjects : []
  return subjects.map((subject: any) => ({
    kind: String(subject?.kind ?? '-').trim() || '-',
    name: String(subject?.name ?? '-').trim() || '-',
    namespace: String(subject?.namespace ?? '-').trim() || '-',
    apiGroup: String(subject?.apiGroup ?? '-').trim() || '-'
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
      namespace: detailKind.value === 'RoleBinding' ? (getRowNamespace(row.value) || undefined) : undefined,
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
    const namespace = getRowNamespace(row.value) || ''
    const data = detailKind.value === 'RoleBinding'
      ? await k8sApi.listRoleBindings(props.clusterId, { namespace })
      : await k8sApi.listClusterRoleBindings(props.clusterId)
    const next = (Array.isArray(data.list) ? data.list : []).find((item: any) => String(item?.metadata?.name ?? '') === name)
    if (next) row.value = next
    else {
      emit('refresh-list')
      row.value = findRow(detailKind.value, namespace, name)
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
  detailKind.value = 'RoleBinding'
  events.value = []
})

function open(targetRow: any, kind: RbacBindingKind) {
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