<template>
  <WorkloadDetailDrawerShell v-model="visible" title="Node 详情" :loading="loading" :name="nodeName" :ref-text="nodeName" @refresh="refresh">
    <template #actions>
      <el-tooltip content="打开资源关系图" placement="top">
        <el-button size="small" text @click="emit('open-topology', { mode: 'node', name: nodeName })">关系图</el-button>
      </el-tooltip>
    </template>
    <template #summary>
      <div class="k8s-summary-grid">
        <div class="k8s-kv k8s-kv--info">
          <div class="k8s-k">节点:</div>
          <div class="k8s-v">{{ nodeName }}</div>
        </div>
        <div :class="['k8s-kv', nodeReady ? 'k8s-kv--good' : 'k8s-kv--bad']">
          <div class="k8s-k">Ready:</div>
          <div class="k8s-v">
            <el-tag :type="nodeReady ? 'success' : 'danger'" size="small">{{ nodeReady ? 'True' : 'False' }}</el-tag>
          </div>
        </div>
        <div class="k8s-kv k8s-kv--info">
          <div class="k8s-k">调度:</div>
          <div class="k8s-v">{{ schedulableText }}</div>
        </div>
        <div class="k8s-kv k8s-kv--info">
          <div class="k8s-k">InternalIP:</div>
          <div class="k8s-v">{{ internalIpText }}</div>
        </div>
        <div class="k8s-kv k8s-kv--info">
          <div class="k8s-k">kubelet:</div>
          <div class="k8s-v">{{ kubeletVersionText }}</div>
        </div>
        <div class="k8s-kv k8s-kv--info">
          <div class="k8s-k">OS:</div>
          <div class="k8s-v">{{ osImageText }}</div>
        </div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="Name">{{ nodeName }}</el-descriptions-item>
              <el-descriptions-item label="Ready">{{ nodeReady ? 'True' : 'False' }}</el-descriptions-item>
              <el-descriptions-item label="Schedulable">{{ schedulableText }}</el-descriptions-item>
              <el-descriptions-item label="Roles">{{ rolesText }}</el-descriptions-item>
              <el-descriptions-item label="PodCIDR">{{ podCidrText }}</el-descriptions-item>
              <el-descriptions-item label="UID">{{ uidText }}</el-descriptions-item>
              <el-descriptions-item label="Created">{{ createdAtText }}</el-descriptions-item>
              <el-descriptions-item label="kubelet">{{ kubeletVersionText }}</el-descriptions-item>
              <el-descriptions-item label="OS Image">{{ osImageText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">地址</div></template>
            <el-table :data="addressRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="type" label="Type" width="160" />
              <el-table-column prop="address" label="Address" min-width="320" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">容量 / 可分配</div></template>
            <el-table :data="capacityRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="key" label="资源" width="160" />
              <el-table-column prop="capacity" label="Capacity" min-width="220" show-overflow-tooltip />
              <el-table-column prop="allocatable" label="Allocatable" min-width="220" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">状态条件</div></template>
            <el-table :data="conditionRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="type" label="Type" width="180" />
              <el-table-column prop="status" label="Status" width="110" align="center" header-align="center">
                <template #default="{ row }">
                  <el-tag :type="row.status === 'True' ? 'success' : row.status === 'False' ? 'danger' : 'info'" size="small">
                    {{ row.status }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="Reason" width="220" show-overflow-tooltip />
              <el-table-column prop="message" label="Message" min-width="360" show-overflow-tooltip />
              <el-table-column prop="lastTransitionTime" label="LastTransition" width="200" show-overflow-tooltip />
            </el-table>
          </el-card>

            <el-card v-if="nodeLabels.length > 0" shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">Labels</div></template>
              <div class="k8s-label-tags">
                <el-tag v-for="item in nodeLabels" :key="item.key" size="small" type="info" effect="plain" class="k8s-label-tag">
                  {{ item.key }}={{ item.value }}
                </el-tag>
              </div>
            </el-card>

            <el-card v-if="nodeAnnotations.length > 0" shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">Annotations</div></template>
              <el-table :data="nodeAnnotations" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="key" label="Key" width="320" show-overflow-tooltip />
                <el-table-column prop="value" label="Value" min-width="400" show-overflow-tooltip />
              </el-table>
            </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="Pods" name="pods">
        <div class="k8s-tab-pane">
          <div class="k8s-pane-toolbar">
            <el-space :size="8">
              <el-tag v-if="podsLoading" size="small" type="info" effect="light">加载中</el-tag>
              <el-tag v-else size="small" type="info" effect="light">共 {{ podRows.length }} 条</el-tag>
              <el-tooltip content="刷新" placement="top">
                <el-button size="small" :icon="RefreshRight" circle :loading="podsLoading" @click="loadPods" />
              </el-tooltip>
            </el-space>
          </div>
          <el-table :data="podRows" stripe size="small" class="k8s-detail-table" @row-dblclick="(r: any) => emit('open-pod-detail', r.raw)">
            <el-table-column prop="namespace" label="Namespace" width="180" show-overflow-tooltip />
            <el-table-column prop="name" label="Name" min-width="260" show-overflow-tooltip />
            <el-table-column prop="phase" label="Phase" width="140" show-overflow-tooltip />
            <el-table-column prop="ready" label="Ready" width="100" align="center" header-align="center" />
            <el-table-column prop="restarts" label="Restarts" width="100" align="center" header-align="center" />
            <el-table-column prop="age" label="Age" width="120" align="center" header-align="center" />
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="关联资源" name="related">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">CSINode（节点存储能力）</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ csiNodeRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="csiNodeRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="drivers" label="Drivers" width="100" align="center" header-align="center" />
              <el-table-column prop="firstDriver" label="FirstDriver" min-width="220" show-overflow-tooltip />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">VolumeAttachments（挂载链路）</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ volumeAttachmentRows.length }} 条</el-tag></div>
              </div>
            </template>
            <el-table :data="volumeAttachmentRows" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="pvName" label="PV" min-width="220" show-overflow-tooltip />
              <el-table-column prop="attached" label="Attached" width="120" align="center" header-align="center" />
            </el-table>
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="事件日志" name="events">
        <div class="k8s-tab-pane">
          <div class="k8s-pane-toolbar">
            <el-space :size="8">
              <el-tag v-if="eventsLoading" size="small" type="info" effect="light">加载中</el-tag>
              <el-tag v-else size="small" type="info" effect="light">共 {{ events.length }} 条</el-tag>
              <el-tooltip content="刷新" placement="top">
                <el-button size="small" :icon="RefreshRight" circle :loading="eventsLoading" @click="loadEvents" />
              </el-tooltip>
            </el-space>
          </div>
          <el-table :data="events" stripe size="small" class="k8s-detail-table">
            <el-table-column prop="type" label="Type" width="110" show-overflow-tooltip />
            <el-table-column prop="reason" label="Reason" width="200" show-overflow-tooltip />
            <el-table-column prop="message" label="Message" min-width="420" show-overflow-tooltip />
            <el-table-column prop="count" label="Count" width="90" align="center" header-align="center" />
            <el-table-column prop="lastSeen" label="LastSeen" width="160" align="center" header-align="center" />
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="YAML配置" name="yaml">
        <div class="k8s-tab-pane">
          <K8sYamlPanel
            :meta="`cluster=${props.clusterId}  ${nodeName}`"
            :text="yamlViewText"
            :loading="yamlLoading"
            height="60vh"
            @refresh="loadYaml"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
  </WorkloadDetailDrawerShell>
</template>

<script setup lang="ts">
import { CopyDocument, RefreshRight } from '@element-plus/icons-vue'
import { computed, ref, watch } from 'vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import K8sYamlPanel from '@/features/k8s/components/K8sYamlPanel.vue'
import { getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import WorkloadDetailDrawerShell from './WorkloadDetailDrawerShell.vue'

type TabKey = 'overview' | 'pods' | 'related' | 'events' | 'yaml'

const props = defineProps<{
  clusterId: number
}>()

const emit = defineEmits<{
  (e: 'open-pod-detail', row: any): void
  (e: 'open-topology', payload: { mode: 'node'; name: string }): void
}>()

const visible = ref(false)
const loading = ref(false)
const tab = ref<TabKey>('overview')
const nodeRow = ref<any>(null)
const nodeObj = ref<any>(null)

const nodeView = computed(() => nodeObj.value ?? nodeRow.value)

const nodeName = computed(() => String(nodeView.value?.metadata?.name ?? '').trim())
const uidText = computed(() => String(nodeView.value?.metadata?.uid ?? '-'))
const createdAtText = computed(() => String(nodeView.value?.metadata?.creationTimestamp ?? '-'))
const podCidrText = computed(() => String(nodeView.value?.spec?.podCIDR ?? nodeView.value?.spec?.podCIDRs?.[0] ?? '-'))

const nodeReady = computed(() => {
  const conds = nodeView.value?.status?.conditions
  if (Array.isArray(conds)) {
    const r = conds.find((c: any) => String(c?.type ?? '') === 'Ready')
    if (r) return String(r?.status ?? '') === 'True'
  }
  return Boolean(nodeRow.value?.ready)
})

const schedulableText = computed(() => {
  const uns = nodeView.value?.spec?.unschedulable === true
  return uns ? '不可调度' : '可调度'
})

const osImageText = computed(() => String(nodeView.value?.status?.nodeInfo?.osImage ?? '-'))
const kubeletVersionText = computed(() => String(nodeView.value?.status?.nodeInfo?.kubeletVersion ?? '-'))

function getRolesText(obj: any): string {
  const labels = obj?.metadata?.labels
  if (!labels || typeof labels !== 'object') return '-'
  const keys = Object.keys(labels)
  const roles = keys
    .filter((k) => k.startsWith('node-role.kubernetes.io/'))
    .map((k) => k.replace('node-role.kubernetes.io/', '').trim())
    .filter(Boolean)
  return roles.length ? roles.join(', ') : '-'
}

const rolesText = computed(() => getRolesText(nodeView.value))

type AddressRow = { type: string; address: string }
const addressRows = computed<AddressRow[]>(() => {
  const list = nodeView.value?.status?.addresses
  if (!Array.isArray(list) || list.length === 0) return []
  return list
    .map((a: any) => ({ type: String(a?.type ?? '-'), address: String(a?.address ?? '-') }))
    .filter((it: any) => it.type !== '-' || it.address !== '-')
})

const internalIpText = computed(() => {
  const ip = addressRows.value.find((a) => a.type === 'InternalIP')?.address ?? ''
  return ip || '-'
})

type CapacityRow = { key: string; capacity: string; allocatable: string }
const capacityRows = computed<CapacityRow[]>(() => {
  const cap = nodeView.value?.status?.capacity ?? {}
  const alloc = nodeView.value?.status?.allocatable ?? {}
  const keys = new Set<string>([...Object.keys(cap), ...Object.keys(alloc)])
  const list = Array.from(keys)
    .sort()
    .map((k) => ({ key: k, capacity: String(cap?.[k] ?? '-'), allocatable: String(alloc?.[k] ?? '-') }))
  return list.length ? list : [{ key: 'cpu', capacity: '-', allocatable: '-' }]
})

type ConditionRow = { type: string; status: string; reason: string; message: string; lastTransitionTime: string }
const conditionRows = computed<ConditionRow[]>(() => {
  const conds = nodeView.value?.status?.conditions
  if (!Array.isArray(conds) || conds.length === 0) return []
  return conds.map((c: any) => ({
    type: String(c?.type ?? '-'),
    status: String(c?.status ?? '-'),
    reason: String(c?.reason ?? '-'),
    message: String(c?.message ?? '-'),
    lastTransitionTime: String(c?.lastTransitionTime ?? '-')
  }))
})

const summaryTitle = computed(() => {
  if (!nodeName.value) return 'Node'
  return `Node  ${nodeName.value}`
})

const nodeLabels = computed(() => {
  const labels = nodeObj.value?.metadata?.labels
  if (!labels || typeof labels !== 'object') return []
  return Object.keys(labels).sort().map((k) => ({ key: k, value: String(labels[k] ?? '') }))
})

const nodeAnnotations = computed(() => {
  const annotations = nodeObj.value?.metadata?.annotations
  if (!annotations || typeof annotations !== 'object') return []
  return Object.keys(annotations).sort().map((k) => ({ key: k, value: String(annotations[k] ?? '') }))
})

function isHttp404(err: unknown): boolean {
  const e = err as ApiError
  const status = (e as any)?.data?.http_status
  return status === 404
}

async function loadDetail() {
  if (!props.clusterId || !nodeName.value) return
  loading.value = true
  try {
    const data = await k8sApi.getNodeDetail(props.clusterId, nodeName.value)
    nodeObj.value = data.obj
  } catch (e) {
    const err = e as ApiError
    if (!isHttp404(err)) {
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    }
  } finally {
    loading.value = false
  }
}

type PodRow = { namespace: string; name: string; phase: string; ready: string; restarts: number; age: string; raw: any }
const podsRaw = ref<any[]>([])
const podsLoading = ref(false)

function buildPodRow(pod: any): PodRow {
  const ns = String(pod?.metadata?.namespace ?? '-')
  const name = String(pod?.metadata?.name ?? '-')
  const phase = String(pod?.status?.phase ?? '-')
  const statuses = Array.isArray(pod?.status?.containerStatuses) ? pod.status.containerStatuses : []
  const total = Array.isArray(pod?.spec?.containers) ? pod.spec.containers.length : statuses.length
  const readyCount = statuses.filter((s: any) => s?.ready === true).length
  const ready = total > 0 ? `${readyCount}/${total}` : '-'
  const restarts = statuses.reduce((sum: number, s: any) => sum + (Number(s?.restartCount ?? 0) || 0), 0)
  const age = getCreationAgeText(pod)
  return { namespace: ns, name, phase, ready, restarts, age, raw: pod }
}

const podRows = computed<PodRow[]>(() => podsRaw.value.map(buildPodRow))
const csiNodeRows = ref<Array<{ name: string; drivers: number; firstDriver: string }>>([])
const volumeAttachmentRows = ref<Array<{ name: string; pvName: string; attached: string }>>([])
const relatedLoading = ref(false)

async function loadRelated() {
  if (!props.clusterId || !nodeName.value) return
  relatedLoading.value = true
  try {
    const [csiNodes, volumeAttachments] = await Promise.all([
      k8sApi.listCSINodes(props.clusterId, {}),
      k8sApi.listVolumeAttachments(props.clusterId, {})
    ])
    csiNodeRows.value = (Array.isArray(csiNodes.list) ? csiNodes.list : [])
      .filter((it: any) => String(it?.metadata?.name ?? '') === nodeName.value)
      .map((it: any) => {
        const drivers: any[] = Array.isArray(it?.spec?.drivers) ? it.spec.drivers : []
        return {
          name: String(it?.metadata?.name ?? '-'),
          drivers: drivers.length,
          firstDriver: String(drivers[0]?.name ?? '-')
        }
      })
    volumeAttachmentRows.value = (Array.isArray(volumeAttachments.list) ? volumeAttachments.list : [])
      .filter((it: any) => String(it?.spec?.nodeName ?? '') === nodeName.value)
      .map((it: any) => ({
        name: String(it?.metadata?.name ?? '-'),
        pvName: String(it?.spec?.source?.persistentVolumeName ?? '-'),
        attached: it?.status?.attached === true ? 'yes' : it?.status?.attached === false ? 'no' : '-'
      }))
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    relatedLoading.value = false
  }
}

async function loadPods() {
  if (!props.clusterId || !nodeName.value) return
  podsLoading.value = true
  try {
    const data = await k8sApi.listNodePods(props.clusterId, nodeName.value, { sort_by: 'metadata.creationTimestamp', order: 'desc' })
    podsRaw.value = data.list ?? []
  } catch (e) {
    const err = e as ApiError
    if (isHttp404(err)) {
      try {
        const data = await k8sApi.listPods(props.clusterId, { sort_by: 'metadata.creationTimestamp', order: 'desc' })
        const list: any[] = Array.isArray(data.list) ? data.list : []
        podsRaw.value = list.filter((p) => String(p?.spec?.nodeName ?? '') === nodeName.value)
      } catch (e2) {
        const err2 = e2 as ApiError
        notifyError(err2.requestId ? `${err2.message} (request_id=${err2.requestId})` : err2.message)
      }
    } else {
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    }
  } finally {
    podsLoading.value = false
  }
}

function normalizeMultilineText(input: string): string {
  let text = String(input ?? '')
  if (!text) return ''
  text = text.replace(/\r\n/g, '\n')
  text = text.replace(/\r/g, '\n')
  return text
}

const yamlLoading = ref(false)
const yamlText = ref('')
const yamlViewText = computed(() => normalizeMultilineText(yamlText.value))

async function loadYaml() {
  if (!props.clusterId || !nodeName.value) return
  yamlLoading.value = true
  try {
    const data = await k8sApi.getNodeYaml(props.clusterId, nodeName.value)
    yamlText.value = data.text ?? ''
  } catch (e) {
    const err = e as ApiError
    if (isHttp404(err)) {
      yamlText.value = JSON.stringify(nodeView.value ?? {}, null, 2) + '\n'
    } else {
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    }
  } finally {
    yamlLoading.value = false
  }
}

async function copyText(text: string) {
  const v = String(text ?? '')
  if (!v) return
  try {
    await navigator.clipboard.writeText(v)
    notifySuccess('已复制')
  } catch {
    try {
      const ta = document.createElement('textarea')
      ta.value = v
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      document.body.appendChild(ta)
      ta.focus()
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      notifySuccess('已复制')
    } catch {
      notifyError('复制失败')
    }
  }
}

type EventRow = { type: string; reason: string; message: string; count: number; lastSeen: string }
const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

function getEventTimeMs(ev: any): number | null {
  const ts =
    ev?.lastTimestamp ??
    ev?.eventTime ??
    ev?.firstTimestamp ??
    ev?.deprecatedLastTimestamp ??
    ev?.deprecatedFirstTimestamp ??
    ev?.metadata?.creationTimestamp
  const t = ts ? Date.parse(String(ts)) : NaN
  return Number.isFinite(t) ? t : null
}

async function loadEvents() {
  if (!props.clusterId || !nodeName.value) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listNodeEvents(props.clusterId, nodeName.value)
    const list = data.list ?? []
    events.value = list
      .map((ev: any) => ({
        row: {
          type: String(ev?.type ?? '-'),
          reason: String(ev?.reason ?? '-'),
          message: String(ev?.message ?? '-'),
          count: Number(ev?.count ?? 0) || 0,
          lastSeen: String(ev?.lastTimestamp ?? ev?.eventTime ?? ev?.firstTimestamp ?? ev?.metadata?.creationTimestamp ?? '-')
        },
        timeMs: getEventTimeMs(ev) ?? 0
      }))
      .sort((a: any, b: any) => b.timeMs - a.timeMs)
      .map((it: any) => it.row)
  } catch (e) {
    const err = e as ApiError
    if (isHttp404(err)) {
      try {
        const data = await k8sApi.listEvents(props.clusterId, { sort_by: 'lastTimestamp', order: 'desc' })
        const list = data.list ?? []
        const filtered = list.filter((ev: any) => String(ev?.involvedObject?.name ?? '') === nodeName.value)
        events.value = filtered
          .map((ev: any) => ({
            row: {
              type: String(ev?.type ?? '-'),
              reason: String(ev?.reason ?? '-'),
              message: String(ev?.message ?? '-'),
              count: Number(ev?.count ?? 0) || 0,
              lastSeen: String(ev?.lastTimestamp ?? ev?.eventTime ?? ev?.firstTimestamp ?? ev?.metadata?.creationTimestamp ?? '-')
            },
            timeMs: getEventTimeMs(ev) ?? 0
          }))
          .sort((a: any, b: any) => b.timeMs - a.timeMs)
          .map((it: any) => it.row)
      } catch (e2) {
        const err2 = e2 as ApiError
        notifyError(err2.requestId ? `${err2.message} (request_id=${err2.requestId})` : err2.message)
      }
    } else {
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    }
  } finally {
    eventsLoading.value = false
  }
}

async function refresh() {
  if (!props.clusterId || !nodeName.value) return
  await loadDetail()
  if (tab.value === 'pods') await loadPods()
  if (tab.value === 'related') await loadRelated()
  if (tab.value === 'events') await loadEvents()
  if (tab.value === 'yaml') await loadYaml()
}

function open(row: any) {
  nodeRow.value = row
  nodeObj.value = null
  tab.value = 'overview'
  visible.value = true
  podsRaw.value = []
  events.value = []
  yamlText.value = ''
}

watch(
  () => [visible.value, nodeName.value] as const,
  ([v]) => {
    if (!v) return
    if (!nodeObj.value) void loadDetail()
  }
)

watch(
  () => [visible.value, tab.value] as const,
  ([v, t]) => {
  if (!v) return
  if (t === 'pods' && podsRaw.value.length === 0) void loadPods()
  if (t === 'related' && csiNodeRows.value.length === 0 && volumeAttachmentRows.value.length === 0) void loadRelated()
  if (t === 'events' && events.value.length === 0) void loadEvents()
  if (t === 'yaml' && !yamlText.value) void loadYaml()
  }
)

watch(
  () => visible.value,
  (v) => {
    if (v) return
    tab.value = 'overview'
    nodeRow.value = null
    nodeObj.value = null
    podsRaw.value = []
    csiNodeRows.value = []
    volumeAttachmentRows.value = []
    events.value = []
    yamlText.value = ''
  }
)

defineExpose<{ open: (row: any) => void }>({ open })
</script>
