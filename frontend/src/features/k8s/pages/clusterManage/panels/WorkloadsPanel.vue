<template>
  <div class="workloads-panel">
    <div v-if="selectedRows.length > 0" class="workloads-batchbar">
      <el-button size="small" type="warning" @click="props.restartSelectedWorkloads(selectedRows)">
        重启选中 {{ selectedRows.length }} 项
      </el-button>
    </div>

    <EnhancedTable
      ref="tableRef"
      :data="data"
      :columns="columns"
      :persist-key="persistKey"
      :show-tools="showTools"
      :row-key="getWorkloadRowKey"
      size="small"
      selectable
      stripe
      border
      @sort-change="emit('sort-change', $event)"
      @selection-change="onSelectionChange"
    >
    <template #cell-namespace="{ row }">
      <span class="k8s-ns" :data-ns-color="nsColorIndex(getNamespace(row))">{{ getNamespace(row) }}</span>
    </template>
    <template #cell-name="{ row }">
      <div class="k8s-name-cell">
        <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
        <el-tooltip v-if="getWarningEventCount(row) > 0" :content="`关联 Warning 事件 ${getWarningEventCount(row)} 条`" placement="top">
          <span class="event-warning-badge">{{ getWarningEventCount(row) }}</span>
        </el-tooltip>
      </div>
    </template>
    <template #cell-replicas="{ row }">
      <span :class="['k8s-status', getReadyStatusClass(row)]">{{ props.getReadyText(row) }}</span>
    </template>
    <template #cell-ready="{ row }">
      <span :class="['k8s-status', getReadyStatusClass(row)]">{{ props.getReadyText(row) }}</span>
    </template>
    <template #cell-dsDesired="{ row }">
      <span class="k8s-num">{{ getDsDesired(row) }}</span>
    </template>
    <template #cell-dsCurrent="{ row }">
      <span class="k8s-num">{{ getDsCurrent(row) }}</span>
    </template>
    <template #cell-dsReady="{ row }">
      <span :class="getReadyNumClass(getDsReady(row), getDsDesired(row))">{{ getDsReady(row) }}</span>
    </template>
    <template #cell-stsCurrent="{ row }">
      <span class="k8s-num">{{ getStsCurrent(row) }}</span>
    </template>
    <template #cell-stsUpdated="{ row }">
      <span class="k8s-num">{{ getStsUpdated(row) }}</span>
    </template>
    <template #cell-stsPods="{ row }">
      <div class="sts-pod-list">
        <template v-if="getStatefulSetPodList(row).length">
          <el-tag
            v-for="item in getStatefulSetPodList(row)"
            :key="item.name"
            size="small"
            :type="item.ready ? 'success' : 'warning'"
            effect="plain"
            class="sts-pod-item"
          >
            {{ item.ordinal }} · {{ item.ready ? 'Ready' : 'NotReady' }}
          </el-tag>
        </template>
        <span v-else class="k8s-empty">-</span>
      </div>
    </template>
    <template #cell-serviceName="{ row }">
      <span class="k8s-ns" :data-ns-color="nsColorIndex(getStsServiceName(row))">{{ getStsServiceName(row) }}</span>
    </template>
    <template #cell-updateStrategy="{ row }">
      <span class="k8s-num">{{ getStsUpdateStrategy(row) }}</span>
    </template>
    <template #cell-podManagementPolicy="{ row }">
      <span class="k8s-num">{{ getStsPodManagementPolicy(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getAgeText(row) }}</span>
    </template>
    <template #cell-upToDate="{ row }">
      <span :class="getReadyNumClass(getUpToDate(row), getDesired(row))">{{ getUpToDate(row) }}</span>
    </template>
    <template #cell-available="{ row }">
      <span :class="getReadyNumClass(getAvailable(row), getDesired(row))">{{ getAvailable(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="openDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="openEdit(row)"><el-icon><EditPen /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="!isDaemonSetRow(row)" content="伸缩" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--success" @click="props.openScale(row)"><el-icon><ScaleToOriginal /></el-icon></button>
        </el-tooltip>
        <span class="k8s-act-divider" />
        <el-tooltip content="重启" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--warn" @click="props.restartWorkloadRow(row)"><el-icon><RefreshRight /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="isDeploymentRow(row)" :content="getRolloutPauseTooltip(row)" placement="top" :show-after="300">
          <button :class="['k8s-act-btn', getRolloutPauseButtonClass(row)]" @click="props.toggleWorkloadPaused(row)">
            <el-icon><component :is="getRolloutPauseIcon(row)" /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip content="更新镜像" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="openUpdateImageDialog(row)"><el-icon><Upload /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="!isDaemonSetRow(row)" content="版本历史" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openWorkloadRollout(row)"><el-icon><Clock /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openWorkloadYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteWorkloadRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
    </EnhancedTable>

    <el-dialog
      v-model="imageDialogVisible"
      title="更新镜像"
      width="640px"
      destroy-on-close
      align-center
      :close-on-click-modal="false"
      class="workload-image-dialog"
      @closed="resetImageDialog"
    >
      <el-form class="image-dialog-form" label-width="92px" @submit.prevent>
        <el-form-item label="工作负载">
          <div class="image-dialog-target">
            <el-tag size="small" effect="plain">{{ imageDialog.kind }}</el-tag>
            <span>{{ imageDialog.namespace }}/{{ imageDialog.name }}</span>
          </div>
        </el-form-item>
        <el-form-item label="容器">
          <el-select v-model="imageDialog.container" placeholder="选择容器" style="width: 100%" @change="syncImageDialogContainer">
            <el-option v-for="option in imageDialogOptions" :key="option.value" :label="option.label" :value="option.value">
              <div class="image-dialog-option">
                <span>{{ option.label }}</span>
                <span class="image-dialog-option__meta">{{ option.image }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="当前镜像">
          <div class="image-dialog-current mono">{{ imageDialog.currentImage || '-' }}</div>
        </el-form-item>
        <el-form-item label="镜像仓库">
          <el-input :model-value="imageDialog.repository" readonly />
        </el-form-item>
        <el-form-item label="新 Tag">
          <el-input v-model="imageDialog.newTag" placeholder="例如 1.27.1 或 stable" clearable />
        </el-form-item>
        <el-form-item label="更新预览">
          <div class="image-dialog-preview mono">{{ nextImagePreview || '-' }}</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="image-dialog-footer">
          <el-button @click="imageDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="imageDialogSubmitting" :disabled="!nextImagePreview" @click="submitImageUpdate">
            确认更新
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { Clock, Delete, Document, EditPen, RefreshRight, ScaleToOriginal, Upload, VideoPause, VideoPlay, View } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import type { WorkloadKind } from '@/features/k8s/pages/ClusterManageView.types'
import { getCreationAgeText, getWorkloadAvailable, getWorkloadDesired, nsColorIndex, getReadyNumClass } from '@/features/k8s/pages/ClusterManageView.utils'

const props = defineProps<{
  data: any[]
  clusterId?: number
  persistKey: string
  showTools: boolean
  workloadKind: WorkloadKind
  getReadyText: (row: any) => string
  getWarningEventCount: (row: any) => number
  openDeploymentDetail: (row: any) => void
  openEditDeployment: (row: any) => void
  openStatefulSetDetail: (row: any) => void
  openEditStatefulSet: (row: any) => void
  openDaemonSetDetail: (row: any) => void
  openEditDaemonSet: (row: any) => void
  openScale: (row: any) => void
  restartWorkloadRow: (row: any) => void
  restartSelectedWorkloads: (rows: any[]) => void | Promise<void>
  updateWorkloadImage: (payload: { kind: string; namespace: string; name: string; container: string; image: string }) => void | Promise<void>
  toggleWorkloadPaused: (row: any) => void | Promise<void>
  openWorkloadRollout: (row: any) => void
  openWorkloadYaml: (row: any) => void
  deleteWorkloadRow: (row: any) => void
}>()

const columns = computed<EnhancedColumn[]>(() =>
  props.workloadKind === 'Deployment'
    ? [
        { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
        { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
        { key: 'ready', label: 'READY', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
        { key: 'upToDate', label: 'UP-TO-DATE', width: 140, align: 'center', headerAlign: 'center', defaultVisible: true },
        { key: 'available', label: 'AVAILABLE', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
        { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
        { key: 'kind', label: 'Kind', prop: 'kind', width: 140, sortable: 'custom', defaultVisible: false },
        { key: 'actions', label: '操作', width: 340, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
      ]
    : props.workloadKind === 'StatefulSet'
      ? [
          { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
          { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
          { key: 'ready', label: 'READY', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
          { key: 'stsCurrent', label: 'CURRENT', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
          { key: 'stsUpdated', label: 'UPDATED', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
          { key: 'stsPods', label: 'PODS(ORDINAL)', minWidth: 280, defaultVisible: true },
          { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
          { key: 'serviceName', label: 'Service', minWidth: 160, defaultVisible: false },
          { key: 'updateStrategy', label: 'UpdateStrategy', width: 150, defaultVisible: false },
          { key: 'podManagementPolicy', label: 'PodMgmt', width: 130, defaultVisible: false },
          { key: 'kind', label: 'Kind', prop: 'kind', width: 140, sortable: 'custom', defaultVisible: false },
          { key: 'actions', label: '操作', width: 304, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
        ]
      : [
          { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
          { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
          { key: 'dsDesired', label: 'DESIRED', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
          { key: 'dsCurrent', label: 'CURRENT', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
          { key: 'dsReady', label: 'READY', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
          { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
          { key: 'kind', label: 'Kind', prop: 'kind', width: 140, sortable: 'custom', defaultVisible: false },
          { key: 'actions', label: '操作', width: 228, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
        ]
)

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
const selectedRows = ref<any[]>([])
const imageDialogVisible = ref(false)
const imageDialogSubmitting = ref(false)
const imageDialogOptions = ref<WorkloadImageOption[]>([])
const statefulSetPods = ref<Record<string, Array<{ name: string; ordinal: number; ready: boolean }>>>({})
let statefulSetPodsSeq = 0
const imageDialog = reactive({
  kind: '',
  namespace: '',
  name: '',
  container: '',
  currentImage: '',
  repository: '',
  newTag: ''
})

const nextImagePreview = computed(() => {
  const repository = imageDialog.repository.trim()
  const newTag = imageDialog.newTag.trim()
  if (!repository || !newTag) return ''
  return `${repository}:${newTag}`
})

function clearSelection() {
  selectedRows.value = []
  tableRef.value?.clearSelection?.()
}

defineExpose({ clearSelection, getTable: () => tableRef.value })

function onSelectionChange(rows: any[]) {
  selectedRows.value = Array.isArray(rows) ? rows : []
}

type WorkloadImageOption = {
  value: string
  label: string
  image: string
  repository: string
  tag: string
}

function getUpToDate(row: any): number {
  return Number(row?.status?.updatedReplicas ?? 0)
}

function getAvailable(row: any): number {
  return Number(row?.status?.availableReplicas ?? row?.status?.readyReplicas ?? 0)
}

function getDesired(row: any): number {
  return getWorkloadDesired(row)
}

function getNamespace(row: any): string {
  const v = getNamespaceValue(row)
  return v || '-'
}

function getNamespaceValue(row: any): string {
  return row?.metadata?.namespace != null ? String(row.metadata.namespace).trim() : ''
}

function getWarningEventCount(row: any): number {
  const count = Number(props.getWarningEventCount(row) ?? 0)
  return Number.isFinite(count) && count > 0 ? Math.trunc(count) : 0
}

function getWorkloadRowKey(row: any): string {
  const ns = getNamespace(row)
  const name = row?.metadata?.name != null ? String(row.metadata.name).trim() : ''
  return `${ns}/${name || '-'}`
}

function getStsCurrent(row: any): number {
  return Number(row?.status?.currentReplicas ?? 0)
}

function getDsDesired(row: any): number {
  return Number(row?.status?.desiredNumberScheduled ?? 0)
}

function getDsCurrent(row: any): number {
  return Number(row?.status?.currentNumberScheduled ?? 0)
}

function getDsReady(row: any): number {
  return Number(row?.status?.numberReady ?? 0)
}

function getStsUpdated(row: any): number {
  return Number(row?.status?.updatedReplicas ?? 0)
}

function getStsServiceName(row: any): string {
  const v = row?.spec?.serviceName != null ? String(row.spec.serviceName).trim() : ''
  return v || '-'
}

function getStsUpdateStrategy(row: any): string {
  const v = row?.spec?.updateStrategy?.type != null ? String(row.spec.updateStrategy.type).trim() : ''
  return v || '-'
}

function getStsPodManagementPolicy(row: any): string {
  const v = row?.spec?.podManagementPolicy != null ? String(row.spec.podManagementPolicy).trim() : ''
  return v || '-'
}

function getStatefulSetKey(namespace: string, name: string): string {
  return `${namespace}/${name}`
}

function extractStatefulSetOrdinal(name: string, stsName: string): number {
  const prefix = `${stsName}-`
  if (!name.startsWith(prefix)) return -1
  const raw = name.slice(prefix.length)
  const n = Number(raw)
  return Number.isFinite(n) && n >= 0 ? Math.trunc(n) : -1
}

function getStatefulSetPodReady(pod: any): boolean {
  const statuses: any[] = Array.isArray(pod?.status?.containerStatuses) ? pod.status.containerStatuses : []
  if (statuses.length === 0) return false
  return statuses.every((it) => Boolean(it?.ready))
}

function getStatefulSetPodList(row: any): Array<{ name: string; ordinal: number; ready: boolean }> {
  const namespace = getNamespaceValue(row)
  const name = row?.metadata?.name != null ? String(row.metadata.name).trim() : ''
  if (!namespace || !name) return []
  return statefulSetPods.value[getStatefulSetKey(namespace, name)] ?? []
}

async function loadStatefulSetPods() {
  if (props.workloadKind !== 'StatefulSet' || !props.clusterId) {
    statefulSetPods.value = {}
    return
  }

  const rows = Array.isArray(props.data) ? props.data : []
  const targets = rows
    .map((row) => {
      const namespace = getNamespaceValue(row)
      const name = row?.metadata?.name != null ? String(row.metadata.name).trim() : ''
      return namespace && name ? { namespace, name, uid: String(row?.metadata?.uid ?? '').trim() } : null
    })
    .filter(Boolean) as Array<{ namespace: string; name: string; uid: string }>

  if (targets.length === 0) {
    statefulSetPods.value = {}
    return
  }

  const namespaces = Array.from(new Set(targets.map((it) => it.namespace)))
  const seq = ++statefulSetPodsSeq
  try {
    const namespacePods = await Promise.all(
      namespaces.map(async (ns) => {
        const res = await k8sApi.listPods(props.clusterId!, { namespace: ns })
        return { ns, list: Array.isArray(res.list) ? res.list : [] }
      })
    )
    if (seq !== statefulSetPodsSeq) return

    const targetMap = new Map<string, { namespace: string; name: string; uid: string }>()
    for (const target of targets) {
      targetMap.set(getStatefulSetKey(target.namespace, target.name), target)
    }

    const next: Record<string, Array<{ name: string; ordinal: number; ready: boolean }>> = {}
    for (const { ns, list } of namespacePods) {
      for (const pod of list) {
        const podName = String(pod?.metadata?.name ?? '').trim()
        if (!podName) continue
        const owners: any[] = Array.isArray(pod?.metadata?.ownerReferences) ? pod.metadata.ownerReferences : []
        const owner = owners.find((it) => String(it?.kind ?? '') === 'StatefulSet')
        if (!owner) continue
        const stsName = String(owner?.name ?? '').trim()
        if (!stsName) continue
        const key = getStatefulSetKey(ns, stsName)
        const target = targetMap.get(key)
        if (!target) continue
        if (target.uid && String(owner?.uid ?? '').trim() && String(owner?.uid ?? '').trim() !== target.uid) continue
        const ordinal = extractStatefulSetOrdinal(podName, stsName)
        if (ordinal < 0) continue
        if (!next[key]) next[key] = []
        next[key].push({ name: podName, ordinal, ready: getStatefulSetPodReady(pod) })
      }
    }

    for (const key of Object.keys(next)) {
      next[key].sort((a, b) => a.ordinal - b.ordinal)
    }
    statefulSetPods.value = next
  } catch {
    if (seq !== statefulSetPodsSeq) return
    statefulSetPods.value = {}
  }
}

watch(
  () => [props.workloadKind, props.clusterId, props.data] as const,
  () => {
    void loadStatefulSetPods()
  },
  { immediate: true, deep: true }
)

function getReadyTagType(row: any): 'success' | 'warning' | 'info' {
  const desired = getWorkloadDesired(row)
  const ready = getWorkloadAvailable(row)
  if (!Number.isFinite(desired) || desired <= 0) return ready > 0 ? 'warning' : 'info'
  return ready === desired && ready !== 0 ? 'success' : 'warning'
}

function getReadyStatusClass(row: any): string {
  const t = getReadyTagType(row)
  if (t === 'success') return 'k8s-status--ok'
  if (t === 'warning') return 'k8s-status--warn'
  return 'k8s-status--info'
}

function getAgeText(row: any): string {
  return getCreationAgeText(row)
}

function openDetail(row: any) {
  if (props.workloadKind === 'Deployment') props.openDeploymentDetail(row)
  else if (props.workloadKind === 'StatefulSet') props.openStatefulSetDetail(row)
  else props.openDaemonSetDetail(row)
}

function openEdit(row: any) {
  if (props.workloadKind === 'Deployment') props.openEditDeployment(row)
  else if (props.workloadKind === 'StatefulSet') props.openEditStatefulSet(row)
  else props.openEditDaemonSet(row)
}

function openUpdateImageDialog(row: any) {
  const namespace = getNamespaceValue(row)
  const name = row?.metadata?.name != null ? String(row.metadata.name).trim() : ''
  if (!namespace || !name) return

  const options = collectWorkloadImageOptions(row)
  if (options.length === 0) {
    ElMessage.warning('当前工作负载未返回容器信息，暂无法更新镜像')
    return
  }

  imageDialogOptions.value = options
  imageDialog.kind = String(row?.kind ?? props.workloadKind ?? '').trim() || props.workloadKind
  imageDialog.namespace = namespace
  imageDialog.name = name
  imageDialog.container = options[0].value
  syncImageDialogContainer(options[0].value)
  imageDialogVisible.value = true
}

function collectWorkloadImageOptions(row: any): WorkloadImageOption[] {
  const podSpec = row?.spec?.template?.spec
  const options: WorkloadImageOption[] = []
  const containers = Array.isArray(podSpec?.containers) ? podSpec.containers : []
  const initContainers = Array.isArray(podSpec?.initContainers) ? podSpec.initContainers : []

  for (const container of containers) {
    const option = buildWorkloadImageOption(container, '容器')
    if (option) options.push(option)
  }
  for (const container of initContainers) {
    const option = buildWorkloadImageOption(container, 'Init')
    if (option) options.push(option)
  }

  return options
}

function buildWorkloadImageOption(container: any, prefix: string): WorkloadImageOption | null {
  const name = container?.name != null ? String(container.name).trim() : ''
  const image = container?.image != null ? String(container.image).trim() : ''
  if (!name || !image) return null
  const { repository, tag } = splitImageReference(image)
  return {
    value: name,
    label: `${prefix} · ${name}`,
    image,
    repository,
    tag: tag || 'latest'
  }
}

function splitImageReference(image: string): { repository: string; tag: string } {
  const raw = String(image || '').trim()
  if (!raw) return { repository: '', tag: '' }
  const digestIndex = raw.indexOf('@')
  if (digestIndex >= 0) {
    return { repository: raw.slice(0, digestIndex), tag: '' }
  }
  const lastSlash = raw.lastIndexOf('/')
  const lastColon = raw.lastIndexOf(':')
  if (lastColon > lastSlash) {
    return { repository: raw.slice(0, lastColon), tag: raw.slice(lastColon + 1) }
  }
  return { repository: raw, tag: 'latest' }
}

function syncImageDialogContainer(containerName?: string) {
  const target = String(containerName ?? imageDialog.container).trim()
  const option = imageDialogOptions.value.find((item) => item.value === target)
  if (!option) return
  imageDialog.container = option.value
  imageDialog.currentImage = option.image
  imageDialog.repository = option.repository
  imageDialog.newTag = option.tag
}

async function submitImageUpdate() {
  if (!nextImagePreview.value || !imageDialog.container) {
    ElMessage.warning('请选择容器并填写新 Tag')
    return
  }
  imageDialogSubmitting.value = true
  try {
    await props.updateWorkloadImage({
      kind: imageDialog.kind,
      namespace: imageDialog.namespace,
      name: imageDialog.name,
      container: imageDialog.container,
      image: nextImagePreview.value
    })
    imageDialogVisible.value = false
  } catch {
    return
  } finally {
    imageDialogSubmitting.value = false
  }
}

function resetImageDialog() {
  imageDialogOptions.value = []
  imageDialogSubmitting.value = false
  imageDialog.kind = ''
  imageDialog.namespace = ''
  imageDialog.name = ''
  imageDialog.container = ''
  imageDialog.currentImage = ''
  imageDialog.repository = ''
  imageDialog.newTag = ''
}

function isDaemonSetRow(_row: any): boolean {
  return props.workloadKind === 'DaemonSet'
}

function isDeploymentRow(_row: any): boolean {
  return props.workloadKind === 'Deployment'
}

function isRolloutPaused(row: any): boolean {
  return Boolean(row?.spec?.paused)
}

function getRolloutPauseTooltip(row: any): string {
  return isRolloutPaused(row) ? '恢复 Rollout' : '暂停 Rollout'
}

function getRolloutPauseButtonClass(row: any): string {
  return isRolloutPaused(row) ? 'k8s-act-btn--success' : 'k8s-act-btn--warn'
}

function getRolloutPauseIcon(row: any) {
  return isRolloutPaused(row) ? VideoPlay : VideoPause
}
</script>

<style scoped>
.workloads-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.workloads-batchbar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
}

.k8s-name-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.event-warning-badge {
  display: inline-flex;
  min-width: 20px;
  height: 20px;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 0 6px;
  background: rgba(220, 38, 38, 0.12);
  color: #b91c1c;
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
}

:global(.workload-image-dialog) {
  --image-dialog-field-bg: rgba(248, 250, 252, 0.92);
  --image-dialog-field-border: rgba(148, 163, 184, 0.24);
  --image-dialog-emphasis-bg: rgba(59, 130, 246, 0.06);
  --image-dialog-emphasis-border: rgba(59, 130, 246, 0.18);
}

:global(html.dark .workload-image-dialog) {
  --image-dialog-field-bg: rgba(15, 23, 42, 0.72);
  --image-dialog-field-border: rgba(148, 163, 184, 0.2);
  --image-dialog-emphasis-bg: rgba(37, 99, 235, 0.14);
  --image-dialog-emphasis-border: rgba(96, 165, 250, 0.22);
}

:global(.workload-image-dialog .el-dialog__body) {
  padding-top: 20px;
  padding-bottom: 18px;
}

:global(.workload-image-dialog .el-form-item) {
  margin-bottom: 18px;
}

:global(.workload-image-dialog .el-form-item:last-child) {
  margin-bottom: 0;
}

:global(.workload-image-dialog .el-form-item__label) {
  color: var(--color-text-secondary);
  font-weight: 800;
}

:global(.workload-image-dialog .el-input__wrapper),
:global(.workload-image-dialog .el-select__wrapper) {
  min-height: 40px;
  border-radius: 8px;
  border: 1px solid var(--image-dialog-field-border) !important;
  background: var(--image-dialog-field-bg);
  box-shadow: none !important;
}

:global(.workload-image-dialog .el-input__wrapper.is-focus),
:global(.workload-image-dialog .el-select__wrapper.is-focused) {
  border-color: rgba(59, 130, 246, 0.52) !important;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.12) !important;
}

.image-dialog-form {
  max-width: 100%;
}

.image-dialog-target {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  padding: 8px 12px;
  border: 1px solid var(--image-dialog-emphasis-border);
  border-radius: 8px;
  background: var(--image-dialog-emphasis-bg);
  color: var(--el-text-color-primary);
  font-weight: 700;
  min-width: 0;
}

.image-dialog-target span:last-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.image-dialog-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.image-dialog-option__meta {
  max-width: 280px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-align: right;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.image-dialog-current,
.image-dialog-preview {
  width: 100%;
  min-height: 40px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--image-dialog-field-border);
  background: var(--image-dialog-field-bg);
  color: var(--el-text-color-primary);
  line-height: 1.5;
  word-break: break-all;
}

.image-dialog-preview {
  border-color: var(--image-dialog-emphasis-border);
  background: var(--image-dialog-emphasis-bg);
  color: var(--color-text-primary);
  font-weight: 700;
}

.image-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  width: 100%;
}

:global(html.dark) .image-dialog-option__meta {
  color: rgba(148, 163, 184, 0.9);
}

:global(html.dark) .event-warning-badge {
  background: rgba(248, 113, 113, 0.18);
  color: #fecaca;
}

.sts-pod-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.sts-pod-item {
  margin: 0;
}

.k8s-empty {
  color: var(--el-text-color-placeholder);
}
</style>
