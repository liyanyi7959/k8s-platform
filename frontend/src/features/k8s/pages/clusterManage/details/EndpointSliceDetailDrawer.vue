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
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Service:</div><div class="k8s-v">{{ serviceNameText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">AddressType:</div><div class="k8s-v">{{ addressTypeText }}</div></div>
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
              <el-descriptions-item label="Service">{{ serviceNameText }}</el-descriptions-item>
              <el-descriptions-item label="addressType">{{ addressTypeText }}</el-descriptions-item>
              <el-descriptions-item label="managed-by">{{ managedByText }}</el-descriptions-item>
              <el-descriptions-item label="endpoints">{{ endpointCount }}</el-descriptions-item>
              <el-descriptions-item label="ready">{{ readyCount }}</el-descriptions-item>
              <el-descriptions-item label="ports">{{ portCount }}</el-descriptions-item>
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

      <el-tab-pane label="端点与端口" name="endpoints">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Endpoints</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ endpointRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="endpointRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="indexText" label="#" width="72" align="center" header-align="center" />
              <el-table-column prop="addressesText" label="Addresses" min-width="240" show-overflow-tooltip />
              <el-table-column prop="readyText" label="Ready" width="100" align="center" header-align="center" />
              <el-table-column prop="servingText" label="Serving" width="100" align="center" header-align="center" />
              <el-table-column prop="terminatingText" label="Terminating" width="120" align="center" header-align="center" />
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
              <el-table-column prop="name" label="Name" min-width="160" show-overflow-tooltip />
              <el-table-column prop="portText" label="Port" width="100" align="center" header-align="center" />
              <el-table-column prop="protocol" label="Protocol" width="120" align="center" header-align="center" />
              <el-table-column prop="appProtocol" label="AppProtocol" min-width="160" show-overflow-tooltip />
            </el-table>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Raw Endpoints</div></template>
            <CodeMirrorViewer :text="endpointsViewText" language="json" :theme="props.editorTheme" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
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

type EndpointRow = {
  indexText: string
  addressesText: string
  readyText: string
  servingText: string
  terminatingText: string
  nodeName: string
  targetRefText: string
}

type PortRow = {
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

function boolText(value: unknown): string {
  if (typeof value !== 'boolean') return '-'
  return value ? 'yes' : 'no'
}

function buildEndpointRows(source: any): EndpointRow[] {
  const endpoints = Array.isArray(source?.endpoints) ? source.endpoints : []
  return endpoints.map((endpoint: any, index: number) => {
    const targetKind = String(endpoint?.targetRef?.kind ?? '').trim()
    const targetName = String(endpoint?.targetRef?.name ?? '').trim()
    return {
      indexText: String(index + 1),
      addressesText: Array.isArray(endpoint?.addresses) ? endpoint.addresses.map((item: unknown) => String(item ?? '').trim()).filter(Boolean).join(', ') || '-' : '-',
      readyText: boolText(endpoint?.conditions?.ready),
      servingText: boolText(endpoint?.conditions?.serving),
      terminatingText: boolText(endpoint?.conditions?.terminating),
      nodeName: String(endpoint?.nodeName ?? '-').trim() || '-',
      targetRefText: targetKind || targetName ? `${targetKind || '-'} / ${targetName || '-'}` : '-'
    }
  })
}

function buildPortRows(source: any): PortRow[] {
  const ports = Array.isArray(source?.ports) ? source.ports : []
  return ports.map((port: any) => ({
    name: String(port?.name ?? '-').trim() || '-',
    portText: port?.port != null ? String(port.port) : '-',
    protocol: String(port?.protocol ?? 'TCP').trim() || 'TCP',
    appProtocol: String(port?.appProtocol ?? '-').trim() || '-'
  }))
}

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'endpoints' | 'events'>('overview')
const row = ref<any>(null)

const detailNamespace = computed(() => getRowNamespace(row.value) || '')
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `EndpointSlice 详情：${detailName.value}` : 'EndpointSlice 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const serviceNameText = computed(() => String(row.value?.metadata?.labels?.['kubernetes.io/service-name'] ?? '-').trim() || '-')
const addressTypeText = computed(() => String(row.value?.addressType ?? '-').trim() || '-')
const managedByText = computed(() => String(row.value?.metadata?.labels?.['endpointslice.kubernetes.io/managed-by'] ?? '-').trim() || '-')
const endpointRows = computed(() => buildEndpointRows(row.value))
const endpointCount = computed(() => endpointRows.value.length)
const readyCount = computed(() => endpointRows.value.filter((item) => item.readyText === 'yes').length)
const portRows = computed(() => buildPortRows(row.value))
const portCount = computed(() => portRows.value.length)
const endpointsViewText = computed(() => JSON.stringify(row.value?.endpoints ?? [], null, 2))
const labelsViewText = computed(() => JSON.stringify(row.value?.metadata?.labels ?? {}, null, 2))
const annotationsViewText = computed(() => JSON.stringify(row.value?.metadata?.annotations ?? {}, null, 2))

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
      involved_object_kind: 'EndpointSlice',
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
    const data = await k8sApi.listEndpointSlices(props.clusterId, { namespace })
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
