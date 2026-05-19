<template>
  <EnhancedTable
    ref="tableRef"
    :data="data"
    :columns="columns"
    :persist-key="persistKey"
    :show-tools="showTools"
    :row-key="getNamespacedRowKey"
    size="small"
    stripe
    border
    @sort-change="emit('sort-change', $event)"
  >
    <template #topbar-left>
      <div class="jobs-topbar-meta">
        <el-tag type="info">已完成 {{ props.completedCount }} 项</el-tag>
        <el-button v-if="props.canWrite" size="small" type="danger" plain :disabled="props.completedCount <= 0" @click="props.cleanCompletedJobs()">批量清理已完成</el-button>
      </div>
    </template>
    <template #cell-namespace="{ row }">
      <span class="k8s-ns" :data-ns-color="nsColorIndex(String(row?.metadata?.namespace ?? ''))">{{ String(row?.metadata?.namespace ?? '-') }}</span>
    </template>
    <template #cell-name="{ row }">
      <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
    </template>
    <template #cell-cronJob="{ row }">
      <span class="jobs-owner">{{ getOwnerCronJob(row) }}</span>
    </template>
    <template #cell-completions="{ row }">
      <span class="k8s-num">{{ formatCompletions(row) }}</span>
    </template>
    <template #cell-status="{ row }">
      <span :class="['k8s-status', getStatusClass(row)]">{{ getStatusText(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openJobDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEditJob(row)"><el-icon><EditPen /></el-icon></button>
        </el-tooltip>
        <span class="k8s-act-divider" />
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openJobYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteJobRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, EditPen, View } from '@element-plus/icons-vue'
import { ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'cronJob', label: '关联 CronJob', minWidth: 180, defaultVisible: true },
  { key: 'completions', label: 'Completions', width: 150, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'active', label: 'Active', prop: 'status.active', width: 120, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'status', label: 'Status', width: 140, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'succeeded', label: 'Succeeded', prop: 'status.succeeded', width: 120, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: false },
  { key: 'failed', label: 'Failed', prop: 'status.failed', width: 120, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: false },
  { key: 'actions', label: '操作', width: 160, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function getOwnerCronJob(row: any): string {
  const refs = Array.isArray(row?.metadata?.ownerReferences) ? row.metadata.ownerReferences : []
  const owner = refs.find((item: any) => String(item?.kind ?? '') === 'CronJob')
  const name = String(owner?.name ?? '').trim()
  return name || '-'
}

function formatCompletions(row: any): string {
  const succeeded = Number(row?.status?.succeeded ?? 0)
  const desired = row?.spec?.completions != null ? Number(row.spec.completions ?? 0) : null
  if (desired == null || !Number.isFinite(desired) || desired <= 0) return String(Number.isFinite(succeeded) ? succeeded : 0)
  return `${Number.isFinite(succeeded) ? succeeded : 0}/${desired}`
}

function getStatusText(row: any): string {
  const failed = Number(row?.status?.failed ?? 0)
  const succeeded = Number(row?.status?.succeeded ?? 0)
  const active = Number(row?.status?.active ?? 0)
  if (Number.isFinite(failed) && failed > 0) return 'Failed'
  if (Number.isFinite(succeeded) && succeeded > 0 && (!Number.isFinite(active) || active <= 0)) return 'Succeeded'
  if (Number.isFinite(active) && active > 0) return 'Running'
  return 'Pending'
}

function getStatusTagType(row: any): 'success' | 'warning' | 'danger' | 'info' {
  const t = getStatusText(row)
  if (t === 'Succeeded') return 'success'
  if (t === 'Failed') return 'danger'
  if (t === 'Running') return 'warning'
  return 'info'
}

function getStatusClass(row: any): string {
  const t = getStatusTagType(row)
  if (t === 'success') return 'k8s-status--ok'
  if (t === 'danger') return 'k8s-status--err'
  if (t === 'warning') return 'k8s-status--warn'
  return 'k8s-status--info'
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  completedCount: number
  openJobDetail: (row: any) => void
  openEditJob: (row: any) => void
  openJobYaml: (row: any) => void
  deleteJobRow: (row: any) => void
  cleanCompletedJobs: () => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>

<style scoped>
.jobs-topbar-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.jobs-owner {
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}
</style>
