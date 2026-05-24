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
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Automount:</div><div class="k8s-v">{{ automountText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Secrets:</div><div class="k8s-v">{{ secretRefsCount }} / Pull {{ imagePullSecretRefsCount }}</div></div>
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
              <el-descriptions-item label="automount">{{ automountText }}</el-descriptions-item>
              <el-descriptions-item label="secrets">{{ secretRefsCount }}</el-descriptions-item>
              <el-descriptions-item label="imagePullSecrets">{{ imagePullSecretRefsCount }}</el-descriptions-item>
              <el-descriptions-item label="labels">{{ labelsCount }}</el-descriptions-item>
              <el-descriptions-item label="annotations">{{ annotationsCount }}</el-descriptions-item>
              <el-descriptions-item label="resourceVersion">{{ resourceVersionText }}</el-descriptions-item>
              <el-descriptions-item label="uid">{{ uidText }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Labels</div></template>
            <CodeMirrorViewer :text="labelsViewText" language="json" :theme="props.editorTheme" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Annotations</div></template>
            <CodeMirrorViewer :text="annotationsViewText" language="json" :theme="props.editorTheme" height="220px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>

      <el-tab-pane label="关联资源" name="related">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Secrets（挂载 / 拉取）</div>
                <div class="k8s-section-actions">
                  <el-space :size="8">
                    <el-tag v-if="relatedLoading" size="small" type="info" effect="light">加载中</el-tag>
                    <el-tag v-else size="small" type="info" effect="light">共 {{ relatedSecrets.length }} 条</el-tag>
                    <el-tooltip content="刷新" placement="top">
                      <el-button size="small" :icon="RefreshRight" circle :loading="relatedLoading" @click="loadRelated" />
                    </el-tooltip>
                  </el-space>
                </div>
              </div>
            </template>
            <el-table :data="relatedSecrets" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
              <el-table-column prop="relationText" label="用途" width="120" show-overflow-tooltip />
              <el-table-column prop="type" label="Type" min-width="220" show-overflow-tooltip />
              <el-table-column prop="dataKeys" label="Keys" width="90" align="center" header-align="center" />
              <el-table-column prop="age" label="AGE" width="120" align="center" header-align="center" />
            </el-table>
          </el-card>

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">Pods（使用该 ServiceAccount）</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ relatedPods.length }} 条</el-tag></div>
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

          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header>
              <div class="k8s-section-title-row">
                <div class="k8s-section-title">资源（从活跃 Pods ownerReferences 聚合）</div>
                <div class="k8s-section-actions"><el-tag size="small" type="info" effect="light">共 {{ relatedControllers.length }} 条</el-tag></div>
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
import type { ControllerRow, EventRow } from '../../ClusterManageView.utils'
import {
  collectControllersFromPodsRaw,
  formatAgeMs,
  formatTs,
  getEventTimeMs,
  getRowNamespace,
  mergeControllers,
  toRelatedPodVmFromPod
} from '../../ClusterManageView.utils'

type RelatedSecretRow = {
  name: string
  relationText: string
  type: string
  dataKeys: number
  age: string
}

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

function findRow(ns: string, name: string): any | null {
  for (const item of props.list ?? []) {
    if (getRowNamespace(item) === ns && String(item?.metadata?.name ?? '') === name) return item
  }
  return null
}

function getAgeText(row: any): string {
  const ts = new Date(String(row?.metadata?.creationTimestamp ?? '')).getTime()
  if (!Number.isFinite(ts)) return '-'
  return formatAgeMs(Math.max(0, Date.now() - ts))
}

function normalizeSecretRefs(input: unknown, relation: string, target: Map<string, Set<string>>) {
  if (!Array.isArray(input)) return
  for (const item of input) {
    const name = typeof item === 'string'
      ? item.trim()
      : String((item as { name?: unknown })?.name ?? '').trim()
    if (!name) continue
    const relations = target.get(name) ?? new Set<string>()
    relations.add(relation)
    target.set(name, relations)
  }
}

function getPodServiceAccountName(pod: any): string {
  const explicit = String(pod?.spec?.serviceAccountName ?? pod?.spec?.serviceAccount ?? '').trim()
  return explicit || 'default'
}

function getSecretDataKeysCount(secret: any): number {
  const data = secret?.data
  if (!data || typeof data !== 'object' || Array.isArray(data)) return 0
  return Object.keys(data as Record<string, unknown>).length
}

const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview' | 'related' | 'events'>('overview')
const row = ref<any>(null)

const detailNamespace = computed(() => getRowNamespace(row.value) || '')
const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `ServiceAccount 详情：${detailName.value}` : 'ServiceAccount 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const automountText = computed(() => {
  if (row.value?.automountServiceAccountToken === true) return 'true'
  if (row.value?.automountServiceAccountToken === false) return 'false'
  return 'default'
})
const labels = computed(() => (row.value?.metadata?.labels ?? {}) as Record<string, string>)
const labelsCount = computed(() => Object.keys(labels.value ?? {}).length)
const labelsViewText = computed(() => JSON.stringify(labels.value ?? {}, null, 2))
const annotations = computed(() => (row.value?.metadata?.annotations ?? {}) as Record<string, string>)
const annotationsCount = computed(() => Object.keys(annotations.value ?? {}).length)
const annotationsViewText = computed(() => JSON.stringify(annotations.value ?? {}, null, 2))
const resourceVersionText = computed(() => String(row.value?.metadata?.resourceVersion ?? '-'))
const uidText = computed(() => String(row.value?.metadata?.uid ?? '-'))
const secretRelations = computed(() => {
  const result = new Map<string, Set<string>>()
  normalizeSecretRefs(row.value?.secrets, '挂载', result)
  normalizeSecretRefs(row.value?.imagePullSecrets, '拉取', result)
  return result
})
const secretRefsCount = computed(() => Array.isArray(row.value?.secrets) ? row.value.secrets.length : 0)
const imagePullSecretRefsCount = computed(() => Array.isArray(row.value?.imagePullSecrets) ? row.value.imagePullSecrets.length : 0)

const relatedSecrets = ref<RelatedSecretRow[]>([])
const relatedPods = ref<any[]>([])
const relatedControllers = ref<ControllerRow[]>([])
const relatedLoading = ref(false)

const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

function buildRelatedSecrets(secretList: any[]): RelatedSecretRow[] {
  const byName = new Map<string, any>()
  for (const secret of secretList) {
    const name = String(secret?.metadata?.name ?? '').trim()
    if (name) byName.set(name, secret)
  }
  return Array.from(secretRelations.value.entries())
    .map(([name, relations]) => {
      const secret = byName.get(name)
      return {
        name,
        relationText: Array.from(relations).sort((a, b) => a.localeCompare(b, 'zh-Hans-CN')).join(' + '),
        type: String(secret?.type ?? '-'),
        dataKeys: getSecretDataKeysCount(secret),
        age: getAgeText(secret)
      }
    })
    .sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
}

async function loadRelated() {
  if (!props.clusterId || !row.value) return
  const namespace = detailNamespace.value
  const name = detailName.value
  if (!namespace || !name) return
  relatedLoading.value = true
  try {
    const [podData, secretData] = await Promise.all([
      k8sApi.listPods(props.clusterId, { namespace }),
      secretRelations.value.size > 0 ? k8sApi.listSecrets(props.clusterId, { namespace }) : Promise.resolve({ list: [] })
    ])

    const podsRaw = (Array.isArray(podData.list) ? podData.list : []).filter((pod: any) => getPodServiceAccountName(pod) === name)
    relatedPods.value = podsRaw
      .map((pod: any) => toRelatedPodVmFromPod(pod))
      .sort((a: any, b: any) => `${a.namespace}/${a.name}`.localeCompare(`${b.namespace}/${b.name}`, 'zh-Hans-CN'))
    relatedControllers.value = mergeControllers(collectControllersFromPodsRaw(podsRaw))
    relatedSecrets.value = buildRelatedSecrets(Array.isArray(secretData.list) ? secretData.list : [])
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
      involved_object_kind: 'ServiceAccount',
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
    const data = await k8sApi.listServiceAccounts(props.clusterId, { namespace })
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
  () => [visible.value, tab.value, detailNamespace.value, detailName.value] as const,
  ([isVisible, currentTab]) => {
    if (!isVisible) return
    if (currentTab === 'related' && relatedPods.value.length === 0 && relatedSecrets.value.length === 0) void loadRelated()
    if (currentTab === 'events' && events.value.length === 0) void loadEvents()
  }
)

watch(() => visible.value, (isVisible) => {
  if (isVisible) return
  tab.value = 'overview'
  row.value = null
  relatedSecrets.value = []
  relatedPods.value = []
  relatedControllers.value = []
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