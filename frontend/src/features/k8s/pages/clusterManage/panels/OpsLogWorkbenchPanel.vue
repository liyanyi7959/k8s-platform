<template>
  <div class="ops-log-page">
    <div class="ops-log-page__toolbar">
      <div class="ops-log-page__fields">
        <el-select
          v-model="selectedNamespace"
          class="ops-log-page__field ops-log-page__field--ns"
          placeholder="选择命名空间"
          clearable
          filterable
          :disabled="!props.clusterId || props.namespaces.length === 0"
        >
          <el-option v-for="ns in props.namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>

        <el-select
          v-model="selectedServicePortKey"
          class="ops-log-page__field ops-log-page__field--port"
          placeholder="按 Service / Port 选入口"
          clearable
          filterable
          :loading="loadingResources"
          :disabled="!selectedNamespace || servicePortOptions.length === 0"
        >
          <el-option v-for="item in servicePortOptions" :key="item.key" :label="item.label" :value="item.key" />
        </el-select>

        <el-select
          v-model="selectedPodKeys"
          class="ops-log-page__field ops-log-page__field--pods"
          placeholder="选择 Pod"
          multiple
          clearable
          filterable
          collapse-tags
          collapse-tags-tooltip
          :max-collapse-tags="2"
          :loading="loadingResources"
          :disabled="!selectedNamespace || podOptions.length === 0"
        >
          <el-option v-for="item in podOptions" :key="item.key" :label="item.label" :value="item.key" />
        </el-select>
      </div>

      <div class="ops-log-page__actions">
        <el-button :icon="RefreshRight" :loading="loadingResources" :disabled="!selectedNamespace" @click="loadNamespaceResources">刷新入口</el-button>
        <el-button :icon="CircleClose" :disabled="selectedPodKeys.length === 0 && !selectedServicePortKey" @click="clearSelection">清空选择</el-button>
      </div>
    </div>

    <div v-if="showSummary" class="ops-log-page__summary">
      <el-tag v-if="selectedNamespace" size="small" effect="plain" type="info">{{ selectedNamespace }}</el-tag>
      <span class="ops-log-page__summary-text">{{ selectedServiceSummary }}</span>
      <span class="ops-log-page__summary-text">{{ selectedPodsSummary }}</span>
    </div>

    <div class="ops-log-page__viewer">
      <MultiPodLogWorkbench ref="workbenchRef" :cluster-id="props.clusterId" title="运维日志工作台" compact navigation-mode="top" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { CircleClose, RefreshRight } from '@element-plus/icons-vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import { getPodRowKey, getRowNamespace, matchLabels } from '@/features/k8s/pages/ClusterManageView.utils'
import MultiPodLogWorkbench from '@/features/k8s/pages/clusterManage/overlays/MultiPodLogWorkbench.vue'
import type { ApiError } from '@/shared/utils/error'
import { notifyError } from '@/shared/utils/notify'

const props = defineProps<{
  clusterId: number
  namespaces: string[]
}>()

type PodTarget = {
  ns: string
  name: string
  containers?: string[]
  container?: string
}

type ServicePortOption = {
  key: string
  label: string
  namespace: string
  serviceName: string
  selector: Record<string, string>
  matchedPodKeys: string[]
}

const workbenchRef = ref<InstanceType<typeof MultiPodLogWorkbench> | null>(null)
const selectedNamespace = ref('')
const selectedServicePortKey = ref('')
const selectedPodKeys = ref<string[]>([])
const serviceRows = ref<any[]>([])
const podRows = ref<any[]>([])
const loadingResources = ref(false)

let loadSeq = 0

const podMap = computed(() => new Map(podRows.value.map((row) => [getPodRowKey(row), row])))

const podOptions = computed(() => {
  return podRows.value.map((row) => {
    const name = String(row?.metadata?.name ?? '-')
    const phase = String(row?.status?.phase ?? '').trim()
    return {
      key: getPodRowKey(row),
      label: phase ? `${name} · ${phase}` : name
    }
  })
})

const servicePortOptions = computed<ServicePortOption[]>(() => {
  const options: ServicePortOption[] = []
  for (const row of serviceRows.value) {
    const namespace = getRowNamespace(row) ?? selectedNamespace.value
    const serviceName = String(row?.metadata?.name ?? '').trim()
    if (!namespace || !serviceName) continue
    const selector = normalizeSelector(row?.spec?.selector)
    const matchedPodKeys = Object.keys(selector).length === 0
      ? []
      : podRows.value
          .filter((pod) => getRowNamespace(pod) === namespace && matchLabels(pod?.metadata?.labels, selector))
          .map((pod) => getPodRowKey(pod))
    const ports: any[] = Array.isArray(row?.spec?.ports) ? row.spec.ports : []
    for (const port of ports) {
      const portText = formatPortLabel(port)
      if (!portText) continue
      options.push({
        key: `${namespace}/${serviceName}/${String(port?.name ?? '').trim()}/${String(port?.port ?? '').trim()}`,
        label: `${serviceName} · ${portText}`,
        namespace,
        serviceName,
        selector,
        matchedPodKeys
      })
    }
  }
  return options.sort((left, right) => left.label.localeCompare(right.label))
})

const selectedServicePort = computed(() => servicePortOptions.value.find((item) => item.key === selectedServicePortKey.value) ?? null)
const workbenchTargets = computed<PodTarget[]>(() => {
  return selectedPodKeys.value
    .map((key) => podMap.value.get(key))
    .filter(Boolean)
    .map((row) => {
      const containers = Array.isArray(row?.spec?.containers)
        ? row.spec.containers.map((item: any) => String(item?.name ?? '').trim()).filter(Boolean)
        : []
      return {
        ns: getRowNamespace(row) ?? '',
        name: String(row?.metadata?.name ?? '').trim(),
        containers,
        container: containers[0] ?? undefined
      }
    })
    .filter((item) => item.ns && item.name)
})
const workbenchTargetSignature = computed(() => workbenchTargets.value.map((item) => `${item.ns}/${item.name}`).join('|'))
const showSummary = computed(() => loadingResources.value || Boolean(selectedNamespace.value) || Boolean(selectedServicePortKey.value) || selectedPodKeys.value.length > 0)
const selectedServiceSummary = computed(() => {
  if (!selectedNamespace.value) return '先选择命名空间，再筛选 Service 端口。'
  if (loadingResources.value) return '正在加载当前命名空间的 Service 与 Pod。'
  if (!selectedServicePort.value) return `当前命名空间共 ${servicePortOptions.value.length} 个可选端口入口。`
  if (Object.keys(selectedServicePort.value.selector).length === 0) {
    return `${selectedServicePort.value.serviceName} 未配置 selector，请手动选择 Pod。`
  }
  return `${selectedServicePort.value.serviceName} 已自动匹配 ${selectedServicePort.value.matchedPodKeys.length} 个 Pod。`
})
const selectedPodsSummary = computed(() => {
  if (selectedPodKeys.value.length === 0) return '当前未选择 Pod，下面日志区域会保持空态。'
  return `当前日志视图已选择 ${selectedPodKeys.value.length} 个 Pod。`
})

watch(() => props.namespaces.join('|'), () => {
  if (!selectedNamespace.value && props.namespaces.length === 1) {
    selectedNamespace.value = props.namespaces[0] ?? ''
  }
}, { immediate: true })

watch(() => props.clusterId, () => {
  resetNamespaceResources()
  if (!selectedNamespace.value && props.namespaces.length === 1) {
    selectedNamespace.value = props.namespaces[0] ?? ''
  }
}, { immediate: true })

watch(selectedNamespace, (value, prev) => {
  if (value === prev) return
  selectedServicePortKey.value = ''
  selectedPodKeys.value = []
  serviceRows.value = []
  podRows.value = []
  workbenchRef.value?.reset()
  if (!value || !props.clusterId) return
  void loadNamespaceResources()
})

watch(selectedServicePortKey, () => {
  const option = selectedServicePort.value
  if (!option) return
  selectedPodKeys.value = option.matchedPodKeys.slice()
})

watch(workbenchTargetSignature, () => {
  void nextTick(() => {
    if (workbenchTargets.value.length > 0) {
      workbenchRef.value?.open(workbenchTargets.value)
      return
    }
    workbenchRef.value?.reset()
  })
})

function normalizeSelector(input: unknown): Record<string, string> {
  if (!input || typeof input !== 'object' || Array.isArray(input)) return {}
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(input as Record<string, unknown>)) {
    const label = String(key ?? '').trim()
    const text = value != null ? String(value).trim() : ''
    if (!label || !text) continue
    out[label] = text
  }
  return out
}

function formatPortLabel(port: any): string {
  const portNumber = port?.port != null ? String(port.port).trim() : ''
  if (!portNumber) return ''
  const portName = String(port?.name ?? '').trim()
  const protocol = String(port?.protocol ?? 'TCP').trim() || 'TCP'
  const targetPort = port?.targetPort != null ? String(port.targetPort).trim() : ''
  const left = portName ? `${portName}:${portNumber}` : portNumber
  return targetPort ? `${left} -> ${targetPort}/${protocol}` : `${left}/${protocol}`
}

function resetNamespaceResources() {
  selectedServicePortKey.value = ''
  selectedPodKeys.value = []
  serviceRows.value = []
  podRows.value = []
  loadingResources.value = false
  workbenchRef.value?.reset()
}

async function loadNamespaceResources() {
  if (!props.clusterId || !selectedNamespace.value) return
  const seq = ++loadSeq
  loadingResources.value = true
  try {
    const [servicesResp, podsResp] = await Promise.all([
      k8sApi.listServices(props.clusterId, { namespace: selectedNamespace.value }),
      k8sApi.listPods(props.clusterId, { namespace: selectedNamespace.value })
    ])
    if (seq !== loadSeq) return
    serviceRows.value = Array.isArray(servicesResp.list) ? servicesResp.list : []
    podRows.value = Array.isArray(podsResp.list) ? podsResp.list : []
  } catch (error) {
    if (seq !== loadSeq) return
    serviceRows.value = []
    podRows.value = []
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : (err.message || '加载日志入口失败'))
  } finally {
    if (seq === loadSeq) {
      loadingResources.value = false
    }
  }
}

function clearSelection() {
  selectedServicePortKey.value = ''
  selectedPodKeys.value = []
}
</script>

<style scoped>
.ops-log-page {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ops-log-page__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.9);
}

.ops-log-page__fields {
  flex: 1;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.ops-log-page__field {
  width: 180px;
}

.ops-log-page__field--port {
  width: 240px;
}

.ops-log-page__field--pods {
  flex: 1 1 280px;
  width: auto;
  min-width: 240px;
}

.ops-log-page__actions {
  display: flex;
  gap: 8px;
  flex: 0 0 auto;
}

.ops-log-page__summary {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 0 2px;
}

.ops-log-page__summary-text {
  color: #64748b;
  font-size: 12px;
}

.ops-log-page__viewer {
  min-height: 620px;
  height: clamp(620px, 78vh, 920px);
  overflow: hidden;
  border-radius: 22px;
}

.ops-log-page__viewer :deep(.multi-pod-log__shell) {
  height: 100%;
  padding: 0;
}

.ops-log-page__viewer :deep(.multi-pod-log__header),
.ops-log-page__viewer :deep(.multi-pod-log__tabs-card),
.ops-log-page__viewer :deep(.multi-pod-log__viewer) {
  border-radius: 20px;
}

.ops-log-page__viewer :deep(.multi-pod-log__empty) {
  min-height: 0;
}

@media (max-width: 960px) {
  .ops-log-page__toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .ops-log-page__actions {
    width: 100%;
  }

  .ops-log-page__field,
  .ops-log-page__field--port,
  .ops-log-page__field--pods {
    width: 100%;
  }
}
</style>