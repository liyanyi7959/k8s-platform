<template>
  <EnhancedTable
    ref="tableRef"
    :data="data"
    :columns="columns"
    :persist-key="persistKey"
    :show-tools="showTools"
    row-key="metadata.name"
    size="small"
    stripe
    border
    @sort-change="emit('sort-change', $event)"
  >
    <template #cell-name="{ row }">
      <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
    </template>
    <template #cell-ready="{ row }">
      <span :class="['k8s-status', props.getReady(row) ? 'k8s-status--ok' : 'k8s-status--err']">{{ props.getReady(row) ? 'True' : 'False' }}</span>
    </template>
    <template #cell-roles="{ row }">
      <span class="k8s-age" :title="getNodeRolesText(row)">{{ getNodeRolesText(row) }}</span>
    </template>
    <template #cell-scheduling="{ row }">
      <span :class="['k8s-status', row?.spec?.unschedulable ? 'k8s-status--warn' : 'k8s-status--ok']">{{ row?.spec?.unschedulable ? '已停止' : '可调度' }}</span>
    </template>
    <template #cell-internalIP="{ row }">
      <span class="k8s-num" :title="getInternalIpText(row)">{{ getInternalIpText(row) }}</span>
    </template>
    <template #cell-cpu="{ row }">
      <span class="k8s-num">{{ getCpuText(row) }}</span>
    </template>
    <template #cell-memory="{ row }">
      <span class="k8s-num">{{ getMemoryText(row) }}</span>
    </template>
    <template #cell-taints="{ row }">
      <span class="k8s-num">{{ getTaintsCount(row) }}</span>
    </template>
    <template #cell-pods="{ row }">
      <span class="k8s-num">{{ getPodsText(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="openDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="openYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <template v-if="props.canWrite">
          <span class="k8s-act-divider" />
          <el-tooltip v-if="row?.spec?.unschedulable" content="恢复调度" placement="top" :show-after="300">
            <button class="k8s-act-btn k8s-act-btn--success" @click="handleUncordon(row)"><el-icon><CircleCheck /></el-icon></button>
          </el-tooltip>
          <el-tooltip v-else content="停止调度" placement="top" :show-after="300">
            <button class="k8s-act-btn k8s-act-btn--warn" @click="handleCordon(row)"><el-icon><CircleClose /></el-icon></button>
          </el-tooltip>
          <el-tooltip content="驱逐" placement="top" :show-after="300">
            <button class="k8s-act-btn k8s-act-btn--warn" @click="handleDrain(row)"><el-icon><RemoveFilled /></el-icon></button>
          </el-tooltip>
          <el-tooltip content="删除" placement="top" :show-after="300">
            <button class="k8s-act-btn k8s-act-btn--danger" @click="handleDelete(row)"><el-icon><Delete /></el-icon></button>
          </el-tooltip>
        </template>
      </div>
    </template>
  </EnhancedTable>

  <el-dialog v-model="drainVisible" title="驱逐节点" class="dialog-sm">
    <el-form label-width="120px">
      <el-form-item label="节点">
        <div class="meta">{{ drainMeta }}</div>
      </el-form-item>
      <el-form-item label="超时(秒)">
        <el-input-number v-model="drainTimeoutSeconds" :min="60" :max="3600" />
      </el-form-item>
      <el-form-item label="强制">
        <el-switch v-model="drainForce" inline-prompt active-text="是" inactive-text="否" />
      </el-form-item>
      <el-form-item label="忽略 DaemonSet">
        <el-switch v-model="drainIgnoreDaemonSets" :disabled="!drainForce" inline-prompt active-text="是" inactive-text="否" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-space>
        <el-tooltip content="取消" placement="top">
          <el-button :icon="CircleClose" circle :disabled="draining" @click="drainVisible = false" />
        </el-tooltip>
        <el-tooltip content="确认" placement="top">
          <el-button type="primary" :icon="CircleCheck" circle :disabled="!drainTarget" :loading="draining" @click="submitDrain" />
        </el-tooltip>
      </el-space>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { CircleCheck, CircleClose, Delete, Document, RemoveFilled, View } from '@element-plus/icons-vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { computed, ref, watch } from 'vue'
import { getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'
import { cordonNode, deleteNode, drainNode, uncordonNode } from '@/features/k8s/api/k8s'
import { ElMessage, ElMessageBox } from 'element-plus'

const columns: EnhancedColumn[] = [
  { key: 'name', label: '节点', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'ready', label: 'Ready', prop: 'ready', width: 120, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'roles', label: 'Roles', minWidth: 180, defaultVisible: true },
  { key: 'scheduling', label: 'Scheduling', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'internalIP', label: 'Internal IP', minWidth: 160, defaultVisible: true },
  { key: 'cpu', label: 'CPU', prop: 'status.allocatable.cpu', width: 140, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'memory', label: '内存', prop: 'status.allocatable.memory', width: 160, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'taints', label: 'Taints', width: 90, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'pods', label: 'Pods', prop: 'status.allocatable.pods', width: 120, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: false },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'kubelet', label: 'kubelet', prop: 'status.nodeInfo.kubeletVersion', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'osImage', label: 'OS Image', prop: 'status.nodeInfo.osImage', minWidth: 260, sortable: 'custom', defaultVisible: false },
  { key: 'podCIDR', label: 'PodCIDR', prop: 'spec.podCIDR', width: 160, sortable: 'custom', defaultVisible: false },
  { key: 'actions', label: '操作', width: 224, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

const props = defineProps<{
  clusterId: number
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  getReady: (row: any) => boolean
  openNodeDetail: (row: any) => void
  openNodeYaml: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
  (e: 'refresh'): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })

const drainVisible = ref(false)
const draining = ref(false)
const drainTarget = ref<any>(null)
const drainForce = ref(false)
const drainIgnoreDaemonSets = ref(true)
const drainTimeoutSeconds = ref(600)

const drainMeta = computed(() => {
  const n = String(drainTarget.value?.metadata?.name ?? '').trim()
  if (!n) return '-'
  return `cluster=${props.clusterId}  ${n}`
})

watch(
  () => drainForce.value,
  (v) => {
    if (!v) drainIgnoreDaemonSets.value = true
  }
)

watch(
  () => drainVisible.value,
  (v) => {
    if (v) return
    drainTarget.value = null
    drainForce.value = false
    drainIgnoreDaemonSets.value = true
    drainTimeoutSeconds.value = 600
  }
)

function getCpuText(row: any): string {
  const cap = row?.status?.capacity?.cpu
  const alloc = row?.status?.allocatable?.cpu
  return formatAllocCap(formatCpu(alloc), formatCpu(cap))
}

function getMemoryText(row: any): string {
  const cap = row?.status?.capacity?.memory
  const alloc = row?.status?.allocatable?.memory
  return formatAllocCap(formatMemory(alloc), formatMemory(cap))
}

function getPodsText(row: any): string {
  const cap = row?.status?.capacity?.pods
  const alloc = row?.status?.allocatable?.pods
  return formatAllocCap(formatPlain(alloc), formatPlain(cap))
}

function getNodeRolesText(row: any): string {
  const labels = row?.metadata?.labels
  if (!labels || typeof labels !== 'object' || Array.isArray(labels)) return '-'
  const roles: string[] = []
  for (const [key, value] of Object.entries(labels as Record<string, unknown>)) {
    if (key === 'kubernetes.io/role') {
      const role = String(value ?? '').trim()
      if (role) roles.push(role)
      continue
    }
    if (!key.startsWith('node-role.kubernetes.io/')) continue
    const suffix = key.slice('node-role.kubernetes.io/'.length).trim()
    roles.push(suffix || 'default')
  }
  const uniq = Array.from(new Set(roles.filter(Boolean)))
  return uniq.length ? uniq.join(', ') : '-'
}

function getInternalIpText(row: any): string {
  const items: any[] = Array.isArray(row?.status?.addresses) ? row.status.addresses : []
  const values = items
    .filter((item) => String(item?.type ?? '').trim() === 'InternalIP')
    .map((item) => String(item?.address ?? '').trim())
    .filter(Boolean)
  return values.join(', ') || '-'
}

function getTaintsCount(row: any): number {
  return Array.isArray(row?.spec?.taints) ? row.spec.taints.length : 0
}

function formatAllocCap(alloc: string, cap: string): string {
  if (alloc === '-' && cap === '-') return '-'
  if (alloc === '-' && cap !== '-') return `-/${cap}`
  if (alloc !== '-' && cap === '-') return `${alloc}/-`
  return `${alloc}/${cap}`
}

function formatPlain(v: any): string {
  const s = v != null ? String(v).trim() : ''
  return s || '-'
}

function formatCpu(v: any): string {
  const s = v != null ? String(v).trim() : ''
  if (!s) return '-'
  const m = s.match(/^(\d+(?:\.\d+)?)m$/)
  if (!m) return s
  const mc = Number(m[1])
  if (!Number.isFinite(mc)) return s
  const cores = mc / 1000
  const text = cores >= 10 ? cores.toFixed(0) : cores >= 1 ? cores.toFixed(2) : cores.toFixed(3)
  return text.replace(/0+$/, '').replace(/\.$/, '')
}

function formatMemory(v: any): string {
  const s = v != null ? String(v).trim() : ''
  if (!s) return '-'
  const bytes = parseBinaryQuantityBytes(s)
  if (bytes == null) return s
  return formatBinaryBytes(bytes)
}

function parseBinaryQuantityBytes(s: string): number | null {
  const m = s.match(/^(\d+(?:\.\d+)?)([KMGTEP]i)?$/)
  if (!m) return null
  const n = Number(m[1])
  if (!Number.isFinite(n)) return null
  const unit = m[2] ?? ''
  const p =
    unit === 'Ki'
      ? 1
      : unit === 'Mi'
        ? 2
        : unit === 'Gi'
          ? 3
          : unit === 'Ti'
            ? 4
            : unit === 'Pi'
              ? 5
              : unit === 'Ei'
                ? 6
                : 0
  return n * 1024 ** p
}

function formatBinaryBytes(bytes: number): string {
  if (!Number.isFinite(bytes)) return '-'
  const abs = Math.abs(bytes)
  const gi = 1024 ** 3
  const mi = 1024 ** 2
  if (abs >= gi) {
    const v = bytes / gi
    const text = v >= 10 ? v.toFixed(0) : v.toFixed(1)
    return text.replace(/\.0$/, '') + 'Gi'
  }
  if (abs >= mi) {
    const v = bytes / mi
    const text = v >= 10 ? v.toFixed(0) : v.toFixed(1)
    return text.replace(/\.0$/, '') + 'Mi'
  }
  const ki = 1024
  if (abs >= ki) {
    const v = bytes / ki
    const text = v >= 10 ? v.toFixed(0) : v.toFixed(1)
    return text.replace(/\.0$/, '') + 'Ki'
  }
  return String(Math.round(bytes)) + 'B'
}

const handleCordon = async (row: any) => {
  if (!props.canWrite) return
  try {
    await ElMessageBox.confirm(`确定要停止节点 ${row.metadata.name} 调度吗？`, '提示', { type: 'warning' })
    await cordonNode(props.clusterId, row.metadata.name)
    ElMessage.success('操作成功')
    emit('refresh')
  } catch (e) {
    // ignore
  }
}

const handleUncordon = async (row: any) => {
  if (!props.canWrite) return
  try {
    await ElMessageBox.confirm(`确定要恢复节点 ${row.metadata.name} 调度吗？`, '提示', { type: 'warning' })
    await uncordonNode(props.clusterId, row.metadata.name)
    ElMessage.success('操作成功')
    emit('refresh')
  } catch (e) {
    // ignore
  }
}

type RowAction = {
  key: string
  label: string
  type: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  class?: string
  icon?: any
  divided?: boolean
  onClick: (row: any) => void
}

const ACTION_COLLAPSE_THRESHOLD = 3
const INLINE_ACTION_COUNT_WHEN_COLLAPSED = 3

const handleDrain = async (row: any) => {
  if (!props.canWrite) return
  drainTarget.value = row
  drainForce.value = false
  drainIgnoreDaemonSets.value = true
  drainTimeoutSeconds.value = 600
  drainVisible.value = true
}

const handleDelete = async (row: any) => {
  if (!props.canWrite) return
  try {
    await ElMessageBox.confirm(`确定要删除节点 ${row.metadata.name} 吗？`, '提示', { type: 'warning' })
    await deleteNode(props.clusterId, row.metadata.name)
    ElMessage.success('操作成功')
    emit('refresh')
  } catch (e) {
    // ignore
  }
}

async function submitDrain() {
  if (!props.canWrite) return
  const name = String(drainTarget.value?.metadata?.name ?? '').trim()
  if (!name) return
  draining.value = true
  try {
    await drainNode(props.clusterId, name, {
      force: drainForce.value,
      timeout_seconds: drainTimeoutSeconds.value,
      ignore_daemonsets: drainForce.value ? drainIgnoreDaemonSets.value : true
    })
    ElMessage.success('操作成功')
    drainVisible.value = false
    emit('refresh')
  } catch {
    // ignore
  } finally {
    draining.value = false
  }
}

function getActions(row: any): RowAction[] {
  // kept for compat but no longer used by template
  return []
}

function openYaml(row: any) {
  props.openNodeYaml(row)
}

function openDetail(row: any) {
  props.openNodeDetail(row)
}
</script>
