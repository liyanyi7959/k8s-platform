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
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Ready:</div><div class="k8s-v">{{ readyCount }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">NotReady:</div><div class="k8s-v">{{ notReadyCount }}</div></div>
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
              <el-descriptions-item label="服务名">{{ serviceNameText }}</el-descriptions-item>
              <el-descriptions-item label="subsets">{{ subsetCount }}</el-descriptions-item>
              <el-descriptions-item label="ports">{{ portCount }}</el-descriptions-item>
              <el-descriptions-item label="ready addresses">{{ readyCount }}</el-descriptions-item>
              <el-descriptions-item label="not ready">{{ notReadyCount }}</el-descriptions-item>
              <el-descriptions-item label="uid">{{ uidText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Subsets</div></template>
            <CodeMirrorViewer :text="subsetsViewText" language="json" :theme="props.editorTheme" height="240px" class="k8s-detail-box k8s-detail-box--fill" />
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

      <el-tab-pane label="端点明细" name="addresses">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Addresses</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ addressRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="addressRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="subsetText" label="Subset" width="92" align="center" header-align="center" />
              <el-table-column prop="state" label="State" width="110" align="center" header-align="center" />
              <el-table-column prop="ip" label="IP" min-width="180" show-overflow-tooltip />
              <el-table-column prop="hostname" label="Hostname" min-width="160" show-overflow-tooltip />
              <el-table-column prop="nodeName" label="Node" min-width="180" show-overflow-tooltip />
              <el-table-column prop="targetRefText" label="TargetRef" min-width="220" show-overflow-tooltip />
            </el-table>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Ports</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ portRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="portRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="subsetText" label="Subset" width="92" align="center" header-align="center" />
              <el-table-column prop="name" label="Name" min-width="160" show-overflow-tooltip />
              <el-table-column prop="portText" label="Port" width="100" align="center" header-align="center" />
              <el-table-column prop="protocol" label="Protocol" width="120" align="center" header-align="center" />
              <el-table-column prop="appProtocol" label="AppProtocol" min-width="160" show-overflow-tooltip />
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

type AddressRow = {
  subsetText: string
  state: string
  ip: string
  hostname: string
  nodeName: string
  targetRefText: string
}

type PortRow = {
  subsetText: string
  name: string
  portText: string
  protocol: string
  appProtocol: string
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

function findRow(namespace: string, name: string): any | null {
  for (const item of props.list ?? []) {
    if (getRowNamespace(item) === namespace && String(item?.metadata?.name ?? '') === name) return item
  }
  return null
}

function getAddressCount(source: any, key: 'addresses' | 'notReadyAddresses'): number {
  const subsets = Array.isArray(source?.subsets) ? source.subsets : []
  return subsets.reduce((sum: number, subset: any) => sum + (Array.isArray(subset?.[key]) ? subset[key].length : 0), 0)
}

function buildAddressRows(source: any): AddressRow[] {
  const subsets = Array.isArray(source?.subsets) ? source.subsets : []
  const rows: AddressRow[] = []
  subsets.forEach((subset: any, subsetIndex: number) => {
    ;(['addresses', 'notReadyAddresses'] as const).forEach((key) => {
      const state = key === 'addresses' ? 'Ready' : 'NotReady'
      const entries = Array.isArray(subset?.[key]) ? subset[key] : []
      entries.forEach((entry: any) => {
        const targetKind = String(entry?.targetRef?.kind ?? '').trim()
        const targetName = String(entry?.targetRef?.name ?? '').trim()
        rows.push({
          subsetText: String(subsetIndex + 1),
          state,
          ip: String(entry?.ip ?? '-').trim() || '-',
          hostname: String(entry?.hostname ?? '-').trim() || '-',
          nodeName: String(entry?.nodeName ?? '-').trim() || '-',
          targetRefText: targetKind || targetName ? `${targetKind || '-'} / ${targetName || '-'}` : '-'
        })
      })
    })
  })
  return rows
}

function buildPortRows(source: any): PortRow[] {
  const subsets = Array.isArray(source?.subsets) ? source.subsets : []
  const rows: PortRow[] = []
  subsets.forEach((subset: any, subsetIndex: number) => {
    const ports = Array.isArray(subset?.ports) ? subset.ports : []
    ports.forEach((port: any) => {
      rows.push({
        subsetText: String(subsetIndex + 1),
        name: String(port?.name ?? '-').trim() || '-',
        portText: port?.port != null ? String(port.port) : '-',
        protocol: String(port?.protocol ?? 'TCP').trim() || 'TCP',
        appProtocol: String(port?.appProtocol ?? '-').trim() || '-'
      })
    })
  })
  return rows
}

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'addresses' | 'events'>('overview')
const row = ref<any>(null)

const detailNamespace = computed(() => getRowNamespace(row.value) || '')
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `Endpoints 详情：${detailName.value}` : 'Endpoints 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const serviceNameText = computed(() => String(row.value?.metadata?.name ?? '-').trim() || '-')
const subsetCount = computed(() => (Array.isArray(row.value?.subsets) ? row.value.subsets.length : 0))
const readyCount = computed(() => getAddressCount(row.value, 'addresses'))
const notReadyCount = computed(() => getAddressCount(row.value, 'notReadyAddresses'))
const addressRows = computed(() => buildAddressRows(row.value))
const portRows = computed(() => buildPortRows(row.value))
const portCount = computed(() => portRows.value.length)
const subsetsViewText = computed(() => JSON.stringify(row.value?.subsets ?? [], null, 2))
const labelsViewText = computed(() => JSON.stringify(row.value?.metadata?.labels ?? {}, null, 2))
const annotationsViewText = computed(() => JSON.stringify(row.value?.metadata?.annotations ?? {}, null, 2))
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
      involved_object_kind: 'Endpoints',
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
    const data = await k8sApi.listEndpoints(props.clusterId, { namespace })
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