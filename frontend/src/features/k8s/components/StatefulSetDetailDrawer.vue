<template>
  <WorkloadDetailDrawerShell v-model="visible" title="StatefulSet 详情" :loading="loading" :ns="wlNamespace" :name="wlName" @refresh="refresh">
    <template #summary>
        <div class="k8s-summary-grid">
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">名称:</div>
            <div class="k8s-v">{{ wlName }}</div>
          </div>
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">命名空间:</div>
            <div class="k8s-v">{{ wlNamespace }}</div>
          </div>
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">Selector:</div>
            <div class="k8s-v"><span class="k8s-link">{{ selectorText }}</span></div>
            <div class="k8s-kv-actions">
              <el-tooltip content="复制" placement="top">
                <el-button size="small" text :icon="CopyDocument" @click="copyText(selectorText)" />
              </el-tooltip>
            </div>
          </div>
          <div class="k8s-kv k8s-kv--muted">
            <div class="k8s-k">创建时间:</div>
            <div class="k8s-v">{{ createdAtText }}</div>
          </div>

          <div class="k8s-kv k8s-kv--ok">
            <div class="k8s-k">副本(可用/期望):</div>
            <div class="k8s-v">{{ readyText }}/{{ replicasText }}</div>
          </div>
          <div class="k8s-kv k8s-kv--ok">
            <div class="k8s-k">更新副本:</div>
            <div class="k8s-v">{{ updatedText }}</div>
          </div>
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">策略:</div>
            <div class="k8s-v">{{ updateStrategyText }}</div>
          </div>
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">UID:</div>
            <div class="k8s-v"><span class="k8s-link">{{ uidShort }}</span></div>
            <div class="k8s-kv-actions">
              <el-tooltip content="复制" placement="top">
                <el-button size="small" text :icon="CopyDocument" @click="copyText(uidFull)" />
              </el-tooltip>
            </div>
          </div>
        </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
        <el-tab-pane label="概览" name="overview">
          <div class="k8s-tab-pane">
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">基础信息</div></template>
              <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
                <el-descriptions-item label="名称">{{ wlName }}</el-descriptions-item>
                <el-descriptions-item label="命名空间">{{ wlNamespace }}</el-descriptions-item>
                <el-descriptions-item label="UID">{{ uidFull }}</el-descriptions-item>
                <el-descriptions-item label="创建时间">{{ createdAtText }}</el-descriptions-item>
                <el-descriptions-item label="Selector">{{ selectorText }}</el-descriptions-item>
                <el-descriptions-item label="Service">{{ serviceNameText }}</el-descriptions-item>
                <el-descriptions-item label="Replicas">{{ replicasText }}</el-descriptions-item>
                <el-descriptions-item label="Pod 管理">{{ podManagementPolicyText }}</el-descriptions-item>
                <el-descriptions-item label="更新策略">{{ updateStrategyText }}</el-descriptions-item>
                <el-descriptions-item label="Current Revision">{{ currentRevisionText }}</el-descriptions-item>
                <el-descriptions-item label="Update Revision">{{ updateRevisionText }}</el-descriptions-item>
              </el-descriptions>
            </el-card>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">状态条件</div></template>
              <el-table :data="wlConditions" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="type" label="Type" width="180" />
                <el-table-column prop="status" label="Status" width="110" align="center" header-align="center">
                  <template #default="{ row }">
                    <el-tag :type="row.status === 'True' ? 'success' : row.status === 'False' ? 'danger' : 'info'" size="small">{{ row.status }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="reason" label="Reason" width="200" show-overflow-tooltip />
                <el-table-column prop="message" label="Message" min-width="260" show-overflow-tooltip />
                <el-table-column prop="lastTransitionTime" label="LastTransition" width="200" show-overflow-tooltip />
              </el-table>
            </el-card>

            <el-card v-if="wlLabels.length > 0" shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">Labels</div></template>
              <div class="k8s-label-tags">
                <el-tag v-for="item in wlLabels" :key="item.key" size="small" type="info" effect="plain" class="k8s-label-tag">
                  {{ item.key }}={{ item.value }}
                </el-tag>
              </div>
            </el-card>

            <el-card v-if="wlAnnotations.length > 0" shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">Annotations</div></template>
              <el-table :data="wlAnnotations" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="key" label="Key" width="320" show-overflow-tooltip />
                <el-table-column prop="value" label="Value" min-width="400" show-overflow-tooltip />
              </el-table>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="容器信息" name="containers">
          <div class="k8s-tab-pane">
            <div class="k8s-pane-toolbar">
              <el-radio-group v-model="activeContainer" size="small">
                <el-radio-button v-for="c in containers" :key="String(c?.name ?? '')" :value="String(c?.name ?? '')">
                  {{ String(c?.name ?? '-') }}
                </el-radio-button>
              </el-radio-group>
            </div>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">容器基础信息</div></template>
              <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
                <el-descriptions-item label="容器名称">{{ selectedContainer?.name || '-' }}</el-descriptions-item>
                <el-descriptions-item label="镜像地址">{{ selectedContainer?.image || '-' }}</el-descriptions-item>
                <el-descriptions-item label="镜像拉取策略">{{ selectedContainer?.imagePullPolicy || '-' }}</el-descriptions-item>
                <el-descriptions-item label="Ports">{{ selectedContainerPortsText }}</el-descriptions-item>
                <el-descriptions-item label="Command">{{ selectedContainerCommandText }}</el-descriptions-item>
                <el-descriptions-item label="Args">{{ selectedContainerArgsText }}</el-descriptions-item>
              </el-descriptions>
            </el-card>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">资源配置</div></template>
              <el-table :data="containerResourceRows" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="name" label="容器" min-width="140" />
                <el-table-column prop="cpuRequests" label="CPU Requests" width="140" />
                <el-table-column prop="cpuLimits" label="CPU Limits" width="140" />
                <el-table-column prop="memRequests" label="Memory Requests" width="160" />
                <el-table-column prop="memLimits" label="Memory Limits" width="160" />
                <el-table-column prop="ephemeralRequests" label="Ephemeral Requests" width="180" />
                <el-table-column prop="ephemeralLimits" label="Ephemeral Limits" width="180" />
              </el-table>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="关联资源" name="related">
          <div class="k8s-tab-pane">
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header>
                <div class="k8s-section-title-row">
                  <div class="k8s-section-title">Pods（匹配 Selector）</div>
                  <div class="k8s-section-actions">
                    <el-space :size="8">
                      <el-tag v-if="podsLoading" size="small" type="info" effect="light">加载中</el-tag>
                      <el-tag v-else size="small" type="info" effect="light">共 {{ relatedPods.length }} 条</el-tag>
                      <el-tooltip content="刷新" placement="top">
                        <el-button size="small" :icon="RefreshRight" circle :loading="podsLoading" @click="loadRelatedPods" />
                      </el-tooltip>
                    </el-space>
                  </div>
                </div>
              </template>
              <el-table :data="relatedPods" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip>
                  <template #default="{ row }">
                    <el-link type="primary" :underline="false" @click="emit('pod-detail', row._raw)">{{ row.name }}</el-link>
                  </template>
                </el-table-column>
                <el-table-column prop="phase" label="Phase" width="120" align="center" header-align="center">
                  <template #default="{ row }">
                    <el-tag
                      :type="row.phase === 'Running' ? 'success' : row.phase === 'Succeeded' ? 'info' : row.phase === 'Pending' ? 'warning' : 'danger'"
                      size="small"
                    >{{ row.phase }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="ready" label="Ready" width="90" align="center" header-align="center" />
                <el-table-column prop="restarts" label="Restarts" width="90" align="center" header-align="center" />
                <el-table-column prop="node" label="Node" min-width="180" show-overflow-tooltip />
                <el-table-column prop="age" label="Age" width="90" align="center" header-align="center" />
                <el-table-column label="操作" width="96" align="center" header-align="center" fixed="right">
                  <template #default="{ row }">
                    <div class="k8s-act-group">
                      <ActionIconButton :icon="Document" tooltip="查看日志" @click="emit('pod-log', row._raw)" />
                      <ActionIconButton :icon="Link" tooltip="打开终端" variant="success" @click="emit('pod-exec', row._raw)" />
                    </div>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header>
                <div class="k8s-section-title-row">
                  <div class="k8s-section-title">Services（匹配 Selector）</div>
                  <div class="k8s-section-actions">
                    <el-space :size="8">
                      <el-tag v-if="servicesLoading" size="small" type="info" effect="light">加载中</el-tag>
                      <el-tag v-else size="small" type="info" effect="light">共 {{ relatedServices.length }} 条</el-tag>
                      <el-tooltip content="刷新" placement="top">
                        <el-button size="small" :icon="RefreshRight" circle :loading="servicesLoading" @click="loadRelatedServices" />
                      </el-tooltip>
                    </el-space>
                  </div>
                </div>
              </template>
              <el-table :data="relatedServices" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="name" label="名称" min-width="220" show-overflow-tooltip />
                <el-table-column prop="type" label="Type" width="160" show-overflow-tooltip />
                <el-table-column prop="clusterIP" label="ClusterIP" width="180" show-overflow-tooltip />
                <el-table-column prop="ports" label="Ports" min-width="220" show-overflow-tooltip />
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
                <el-tooltip content="刷新事件" placement="top">
                  <el-button size="small" :icon="RefreshRight" circle :loading="eventsLoading" @click="loadEvents" />
                </el-tooltip>
              </el-space>
            </div>
            <el-table :data="events" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="type" label="Type" width="110" align="center" header-align="center">
                <template #default="{ row }">
                  <el-tag :type="row.type === 'Warning' ? 'danger' : row.type === 'Normal' ? 'success' : 'info'" size="small">{{ row.type }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="Reason" width="200" show-overflow-tooltip />
              <el-table-column prop="message" label="Message" min-width="360" show-overflow-tooltip />
              <el-table-column prop="count" label="Count" width="90" align="center" header-align="center" />
              <el-table-column prop="lastSeen" label="LastSeen" width="140" align="center" header-align="center" />
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="YAML配置" name="yaml">
          <div class="k8s-tab-pane">
            <K8sYamlPanel
              :meta="`${wlKind}: ${wlNamespace}/${wlName}`"
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
import { CopyDocument, RefreshRight, Document, Link } from '@element-plus/icons-vue'
import { computed, ref, watch } from 'vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import ActionIconButton from '@/shared/components/ActionIconButton.vue'
import K8sYamlPanel from '@/features/k8s/components/K8sYamlPanel.vue'
import WorkloadDetailDrawerShell from './WorkloadDetailDrawerShell.vue'

function normalizeMultilineText(input: string): string {
  let text = String(input ?? '')
  if (!text) return ''
  text = text.replace(/\r\n/g, '\n')
  const quoted = (text.startsWith('"') && text.endsWith('"')) || (text.startsWith("'") && text.endsWith("'"))
  if (quoted && text.includes('\\n')) {
    text = text.slice(1, -1)
  }
  const hasRealNewline = text.includes('\n')
  const hasEscapedNewline = text.includes('\\n')
  if (!hasRealNewline && hasEscapedNewline) {
    text = text.replace(/\\r\\n/g, '\n').replace(/\\n/g, '\n').replace(/\\t/g, '\t')
  }
  return text
}

function formatAgeMs(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '-'
  const sec = Math.floor(ms / 1000)
  const min = Math.floor(sec / 60)
  const hour = Math.floor(min / 60)
  const day = Math.floor(hour / 24)
  if (day > 0) return `${day}d`
  if (hour > 0) return `${hour}h`
  if (min > 0) return `${min}m`
  return `${sec}s`
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
      ta.style.top = '0'
      document.body.appendChild(ta)
      ta.focus()
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      notifySuccess('已复制')
    } catch (e) {
      const err = e as ApiError
      notifyError(err?.message ? `复制失败：${err.message}` : '复制失败')
    }
  }
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits<{
  (e: 'pod-detail', raw: any): void
  (e: 'pod-log', raw: any): void
  (e: 'pod-exec', raw: any): void
}>()

const visible = ref(false)
const loading = ref(false)
const tab = ref<'overview' | 'containers' | 'related' | 'events' | 'yaml'>('overview')
const wlRow = ref<any | null>(null)

function getRowNamespace(row: any): string | null {
  const ns = row?.metadata?.namespace
  const v = ns != null ? String(ns).trim() : ''
  return v ? v : null
}

const wlName = computed(() => String(wlRow.value?.metadata?.name ?? '').trim())
const wlNamespace = computed(() => String(wlRow.value?.metadata?.namespace ?? '').trim())
const wlKind = computed(() => String(wlRow.value?.kind ?? 'StatefulSet').trim() || 'StatefulSet')

const selectorLabels = computed<Record<string, string>>(() => {
  const m = wlRow.value?.spec?.selector?.matchLabels
  if (!m || typeof m !== 'object') return {}
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(m)) {
    const kk = String(k ?? '').trim()
    if (!kk) continue
    out[kk] = v != null ? String(v).trim() : ''
  }
  return out
})

const selectorText = computed(() => {
  const pairs = Object.entries(selectorLabels.value)
    .map(([k, v]) => `${k}=${v}`)
    .filter((s) => !s.endsWith('='))
  return pairs.length ? pairs.join(', ') : '-'
})

const serviceNameText = computed(() => String(wlRow.value?.spec?.serviceName ?? '-'))

const replicasText = computed(() => String(wlRow.value?.spec?.replicas ?? '-'))
const readyText = computed(() => String(wlRow.value?.status?.readyReplicas ?? 0))
const currentText = computed(() => String(wlRow.value?.status?.currentReplicas ?? 0))
const updatedText = computed(() => String(wlRow.value?.status?.updatedReplicas ?? 0))

const podManagementPolicyText = computed(() => String(wlRow.value?.spec?.podManagementPolicy ?? '-'))
const updateStrategyText = computed(() => String(wlRow.value?.spec?.updateStrategy?.type ?? '-'))
const currentRevisionText = computed(() => String(wlRow.value?.status?.currentRevision ?? '-'))
const updateRevisionText = computed(() => String(wlRow.value?.status?.updateRevision ?? '-'))

const uidFull = computed(() => String(wlRow.value?.metadata?.uid ?? '-'))
const uidShort = computed(() => {
  const uid = uidFull.value
  if (!uid || uid === '-') return '-'
  return uid.length > 12 ? uid.slice(0, 6) + '…' + uid.slice(-4) : uid
})

const createdAtText = computed(() => {
  const ts = wlRow.value?.metadata?.creationTimestamp
  if (!ts) return '-'
  const d = new Date(String(ts))
  if (!Number.isFinite(d.getTime())) return String(ts)
  return d.toLocaleString('zh-CN')
})

const wlConditions = computed(() => {
  const raw: any[] = Array.isArray(wlRow.value?.status?.conditions) ? wlRow.value.status.conditions : []
  return raw.map((it) => ({
    type: String(it?.type ?? '-'),
    status: String(it?.status ?? '-'),
    reason: String(it?.reason ?? '-'),
    message: String(it?.message ?? '-'),
    lastTransitionTime: String(it?.lastTransitionTime ?? '-')
  }))
})

const containers = computed<any[]>(() => (Array.isArray(wlRow.value?.spec?.template?.spec?.containers) ? wlRow.value!.spec.template.spec.containers : []))
const activeContainer = ref('')
watch(
  containers,
  (cs) => {
    if (!cs.length) {
      activeContainer.value = ''
      return
    }
    const first = String(cs[0]?.name ?? '').trim()
    if (!activeContainer.value) activeContainer.value = first
    if (!cs.some((c) => String(c?.name ?? '').trim() === activeContainer.value)) activeContainer.value = first
  },
  { immediate: true }
)
const selectedContainer = computed<any | null>(() => {
  const n = activeContainer.value
  if (!n) return containers.value.length ? containers.value[0] : null
  return containers.value.find((c) => String(c?.name ?? '').trim() === n) ?? (containers.value.length ? containers.value[0] : null)
})

const selectedContainerPortsText = computed(() => {
  const ports: any[] = Array.isArray(selectedContainer.value?.ports) ? selectedContainer.value.ports : []
  if (!ports.length) return '-'
  return ports
    .map((p) => {
      const name = p?.name ? String(p.name) : ''
      const c = p?.containerPort != null ? String(p.containerPort) : ''
      const proto = p?.protocol ? String(p.protocol) : 'TCP'
      const left = `${name ? `${name}:` : ''}${c}/${proto}`
      const host = p?.hostPort != null ? String(p.hostPort) : ''
      return host ? `${left}→${host}` : left
    })
    .join(', ')
})

const selectedContainerCommandText = computed(() => {
  const v: any[] = Array.isArray(selectedContainer.value?.command) ? selectedContainer.value.command : []
  return v.length ? v.join(' ') : '-'
})

const selectedContainerArgsText = computed(() => {
  const v: any[] = Array.isArray(selectedContainer.value?.args) ? selectedContainer.value.args : []
  return v.length ? v.join(' ') : '-'
})

function normalizeResourceValue(v: any): string {
  const s = v == null ? '' : String(v)
  return s.trim() ? s : '-'
}

const containerResourceRows = computed(() => {
  return containers.value.map((c) => {
    const req = c?.resources?.requests ?? {}
    const lim = c?.resources?.limits ?? {}
    return {
      name: String(c?.name ?? '-'),
      cpuRequests: normalizeResourceValue(req?.cpu),
      cpuLimits: normalizeResourceValue(lim?.cpu),
      memRequests: normalizeResourceValue(req?.memory),
      memLimits: normalizeResourceValue(lim?.memory),
      ephemeralRequests: normalizeResourceValue(req?.['ephemeral-storage']),
      ephemeralLimits: normalizeResourceValue(lim?.['ephemeral-storage'])
    }
  })
})

const wlLabels = computed(() => {
  const labels = wlRow.value?.metadata?.labels
  if (!labels || typeof labels !== 'object') return []
  return Object.keys(labels).sort().map((k) => ({ key: k, value: String(labels[k] ?? '') }))
})

const wlAnnotations = computed(() => {
  const annotations = wlRow.value?.metadata?.annotations
  if (!annotations || typeof annotations !== 'object') return []
  return Object.keys(annotations).sort().map((k) => ({ key: k, value: String(annotations[k] ?? '') }))
})

type RelatedPodRow = { name: string; phase: string; ready: string; restarts: number; node: string; age: string; _raw: any }
type RelatedServiceRow = { name: string; type: string; clusterIP: string; ports: string }
type EventRow = { type: string; reason: string; message: string; count: number; lastSeen: string }

const relatedPods = ref<RelatedPodRow[]>([])
const podsLoading = ref(false)
const relatedServices = ref<RelatedServiceRow[]>([])
const servicesLoading = ref(false)

function matchLabels(objLabels: Record<string, any> | null | undefined, required: Record<string, any>): boolean {
  const o = objLabels && typeof objLabels === 'object' ? objLabels : {}
  for (const k of Object.keys(required)) {
    if (String(o?.[k] ?? '') !== String(required[k] ?? '')) return false
  }
  return true
}

function getEventTimeMs(ev: any): number | null {
  const ts =
    ev?.lastTimestamp ??
    ev?.eventTime ??
    ev?.firstTimestamp ??
    ev?.deprecatedLastTimestamp ??
    ev?.deprecatedFirstTimestamp ??
    ev?.metadata?.creationTimestamp
  if (!ts) return null
  const t = new Date(String(ts)).getTime()
  return Number.isFinite(t) ? t : null
}

async function loadRelatedPods() {
  if (!props.clusterId || !wlRow.value) return
  const ns = getRowNamespace(wlRow.value)
  if (!ns) return
  podsLoading.value = true
  try {
    const data = await k8sApi.listPods(props.clusterId, { namespace: ns })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const required = selectorLabels.value
    const now = Date.now()
    const filtered = list.filter((p) => matchLabels(p?.metadata?.labels, required))
    relatedPods.value = filtered.map((p) => {
      const cs: any[] = Array.isArray(p?.status?.containerStatuses) ? p.status.containerStatuses : []
      const ready = cs.length ? `${cs.filter((it) => it?.ready).length}/${cs.length}` : '-'
      const restarts = cs.reduce((sum, it) => sum + (Number(it?.restartCount ?? 0) || 0), 0)
      const ts = new Date(String(p?.metadata?.creationTimestamp ?? '')).getTime()
      const age = Number.isFinite(ts) ? formatAgeMs(Math.max(0, now - ts)) : '-'
      return {
        name: String(p?.metadata?.name ?? '-'),
        phase: String(p?.status?.phase ?? '-'),
        ready,
        restarts,
        node: String(p?.spec?.nodeName ?? '-'),
        age,
        _raw: p
      }
    })
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    podsLoading.value = false
  }
}

function formatServicePorts(ports: any[]): string {
  const ps = Array.isArray(ports) ? ports : []
  if (!ps.length) return '-'
  return ps
    .map((p) => {
      const name = p?.name ? String(p.name) : ''
      const port = p?.port != null ? String(p.port) : ''
      const target = p?.targetPort != null ? String(p.targetPort) : ''
      const proto = p?.protocol ? String(p.protocol) : 'TCP'
      const left = `${name ? `${name}:` : ''}${port}/${proto}`
      return target ? `${left}→${target}` : left
    })
    .join(', ')
}

async function loadRelatedServices() {
  if (!props.clusterId || !wlRow.value) return
  const ns = getRowNamespace(wlRow.value)
  if (!ns) return
  servicesLoading.value = true
  try {
    const data = await k8sApi.listServices(props.clusterId, { namespace: ns })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const required = selectorLabels.value
    const filtered = list.filter((svc) => {
      const sel = svc?.spec?.selector
      if (!sel || typeof sel !== 'object') return false
      return matchLabels(sel, required)
    })
    relatedServices.value = filtered.map((svc) => ({
      name: String(svc?.metadata?.name ?? '-'),
      type: String(svc?.spec?.type ?? '-'),
      clusterIP: String(svc?.spec?.clusterIP ?? '-'),
      ports: formatServicePorts(svc?.spec?.ports)
    }))
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    servicesLoading.value = false
  }
}

const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

async function loadEvents() {
  if (!props.clusterId || !wlRow.value) return
  const ns = getRowNamespace(wlRow.value)
  const name = wlName.value
  const kind = wlKind.value
  const uid = String(wlRow.value?.metadata?.uid ?? '')
  if (!ns || !name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, { namespace: ns })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const filtered = list.filter((ev) => {
      const involved = ev?.involvedObject ?? ev?.regarding ?? {}
      const eNs = String(involved?.namespace ?? '')
      const eName = String(involved?.name ?? '')
      const eKind = String(involved?.kind ?? '')
      const eUid = String(involved?.uid ?? '')
      if (eKind && eKind !== kind) return false
      if (eNs && eNs !== ns) return false
      if (uid && eUid) return eUid === uid
      return eName === name
    })
    const mapped = filtered
      .map((ev) => {
        const t = getEventTimeMs(ev)
        const now = Date.now()
        const age = t != null && t > 0 ? formatAgeMs(Math.max(0, now - t)) : '-'
        return {
          type: String(ev?.type ?? '-'),
          reason: String(ev?.reason ?? '-'),
          message: String(ev?.message ?? '-'),
          count: Number(ev?.count ?? ev?.series?.count ?? 1) || 1,
          lastSeen: age
        }
      })
      .sort((a, b) => String(a.lastSeen).localeCompare(String(b.lastSeen)))
    events.value = mapped
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    eventsLoading.value = false
  }
}

const yamlLoading = ref(false)
const yamlText = ref('')
const yamlViewText = computed(() => normalizeMultilineText(yamlText.value))

async function loadYaml() {
  if (!props.clusterId || !wlRow.value) return
  const ns = getRowNamespace(wlRow.value)
  const name = wlName.value
  const kind = wlKind.value
  if (!ns || !name) return
  yamlLoading.value = true
  try {
    const data = await k8sApi.getWorkloadYaml(props.clusterId, { kind, namespace: ns, name })
    yamlText.value = data.text
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    yamlLoading.value = false
  }
}

async function refresh() {
  if (!props.clusterId || !wlRow.value) return
  const ns = wlNamespace.value
  const name = wlName.value
  if (!ns || !name) return
  loading.value = true
  try {
    const data = await k8sApi.listWorkloads(props.clusterId, { namespace: ns, kind: wlKind.value })
    const found = (data.list ?? []).find((it: any) => String(it?.metadata?.name ?? '') === name)
    if (found) wlRow.value = found
    if (tab.value === 'events') await loadEvents()
    if (tab.value === 'related') {
      await loadRelatedPods()
      await loadRelatedServices()
    }
    if (tab.value === 'yaml') await loadYaml()
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loading.value = false
  }
}

function open(row: any) {
  wlRow.value = row
  tab.value = 'overview'
  visible.value = true
  events.value = []
  relatedPods.value = []
  relatedServices.value = []
  yamlText.value = ''
  activeContainer.value = ''
}

watch(
  () => [visible.value, tab.value, wlName.value, wlNamespace.value] as const,
  ([v, t]) => {
    if (!v) return
    if (t === 'events' && events.value.length === 0) void loadEvents()
    if (t === 'related') {
      if (relatedPods.value.length === 0) void loadRelatedPods()
      if (relatedServices.value.length === 0) void loadRelatedServices()
    }
    if (t === 'yaml' && !yamlText.value) void loadYaml()
  }
)

watch(
  () => visible.value,
  (v) => {
    if (v) return
    wlRow.value = null
    tab.value = 'overview'
    events.value = []
    relatedPods.value = []
    relatedServices.value = []
    yamlText.value = ''
    activeContainer.value = ''
  }
)

defineExpose({ open })
</script>
