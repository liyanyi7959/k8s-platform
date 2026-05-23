<template>
  <div class="pods-panel">
    <div v-if="selectedRows.length > 0" class="pods-batchbar">
      <div class="pods-batchbar__meta">已选 {{ selectedRows.length }} 个 Pod</div>
      <div class="pods-batchbar__actions">
        <el-button size="small" class="pods-batchbar__action pods-batchbar__action--logs" @click="props.openMultiPodLogs(selectedRows)">日志工作台</el-button>
        <template v-if="props.canWrite">
          <el-button size="small" class="pods-batchbar__action pods-batchbar__action--danger" @click="props.bulkDeletePods(selectedRows)">删除选中</el-button>
          <el-button size="small" class="pods-batchbar__action pods-batchbar__action--warning" @click="props.bulkDeletePods(selectedRows, { force: true })">强制删除</el-button>
        </template>
      </div>
    </div>

    <EnhancedTable
      ref="tableRef"
      :data="data"
      :columns="columns"
      :persist-key="persistKey"
      :show-tools="showTools"
      :row-key="props.getPodRowKey"
      size="small"
      selectable
      stripe
      border
      @sort-change="emit('sort-change', $event)"
      @selection-change="onSelectionChange"
    >
    <template #cell-namespace="{ row }">
      <span class="k8s-ns" :data-ns-color="nsColorIndex(String(row?.metadata?.namespace ?? ''))">{{ String(row?.metadata?.namespace ?? '-') }}</span>
    </template>
    <template #cell-name="{ row }">
      <div class="k8s-name-cell">
        <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
        <el-tooltip v-if="getWarningEventCount(row) > 0" :content="`关联 Warning 事件 ${getWarningEventCount(row)} 条`" placement="top">
          <span class="event-warning-badge">{{ getWarningEventCount(row) }}</span>
        </el-tooltip>
      </div>
    </template>
    <template #cell-phase="{ row }">
      <span :class="['k8s-status', getPhaseStatusClass(row)]">{{ props.getPodPhaseText(row) }}</span>
    </template>
    <template #cell-ready="{ row }">
      <span :class="getPodReadyClass(row)">{{ props.getPodReadyText(row) }}</span>
    </template>
    <template #cell-restarts="{ row }">
      <span :class="getRestartsClass(props.getPodRestarts(row))">{{ props.getPodRestarts(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ props.getPodAge(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openPodDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="日志流" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--cyan" @click="props.openPodLogs(row)"><el-icon><Collection /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="Shell" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--success" @click="props.openPodExec(row)"><el-icon><Link /></el-icon></button>
        </el-tooltip>
        <span class="k8s-act-divider" />
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openPodYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deletePodRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
    </EnhancedTable>
  </div>
</template>

<script setup lang="ts">
import { Collection, Delete, Document, Link, View } from '@element-plus/icons-vue'
import { ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { nsColorIndex, getRestartsClass, getReadyNumClass } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'name', label: 'Pod', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'phase', label: 'Phase', prop: 'status.phase', width: 140, sortable: 'custom', defaultVisible: true },
  { key: 'ready', label: 'READY', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'restarts', label: 'RESTARTS', prop: 'restarts', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'ageMs', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'node', label: 'Node', prop: 'spec.nodeName', minWidth: 180, sortable: 'custom', defaultVisible: true },
  { key: 'actions', label: '操作', width: 192, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  getWarningEventCount: (row: any) => number
  getPodRowKey: (row: any) => string
  getPodPhaseTagType: (row: any) => string
  getPodPhaseText: (row: any) => string
  getPodReadyText: (row: any) => string
  getPodRestarts: (row: any) => number
  getPodAge: (row: any) => string
  openPodDetail: (row: any) => void
  openPodLogs: (row: any) => void
  openMultiPodLogs: (rows: any[]) => void
  openPodExec: (row: any) => void
  openPodYaml: (row: any) => void
  deletePodRow: (row: any) => void
  bulkDeletePods: (rows: any[], options?: { force?: boolean }) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
  (e: 'selection-change', rows: any[]): void
}>()

const tableRef = ref<any>(null)
const selectedRows = ref<any[]>([])

function clearSelection() {
  selectedRows.value = []
  tableRef.value?.clearSelection?.()
}

function onSelectionChange(rows: any[]) {
  selectedRows.value = Array.isArray(rows) ? rows : []
  emit('selection-change', selectedRows.value)
}

defineExpose({ clearSelection, getTable: () => tableRef.value })

function getPhaseStatusClass(row: any): string {
  const t = String(props.getPodPhaseTagType(row))
  if (t === 'success') return 'k8s-status--ok'
  if (t === 'danger') return 'k8s-status--err'
  if (t === 'warning') return 'k8s-status--warn'
  return 'k8s-status--info'
}

function getPodReadyClass(row: any): string {
  const text = String(props.getPodReadyText(row) ?? '')
  const [currentRaw, desiredRaw] = text.split('/')
  const current = Number(currentRaw ?? 0)
  const desired = Number(desiredRaw ?? 0)
  return getReadyNumClass(current, desired)
}

function getWarningEventCount(row: any): number {
	const count = Number(props.getWarningEventCount(row) ?? 0)
	return Number.isFinite(count) && count > 0 ? Math.trunc(count) : 0
}

</script>

<style scoped>
.pods-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.k8s-name-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.event-warning-badge {
  display: inline-flex;
  min-width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 0 7px;
  border: 1px solid rgba(248, 113, 113, 0.18);
  background: rgba(248, 250, 252, 0.94);
  color: #b91c1c;
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
}

.pods-batchbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(248, 250, 252, 0.94);
}

.pods-batchbar__meta {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}

.pods-batchbar__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.pods-batchbar__action {
  --el-button-bg-color: rgba(255, 255, 255, 0.96);
  --el-button-border-color: rgba(148, 163, 184, 0.24);
  --el-button-text-color: #334155;
  --el-button-hover-bg-color: #fff;
  --el-button-hover-border-color: rgba(100, 116, 139, 0.28);
  --el-button-hover-text-color: #0f172a;
  --el-button-active-bg-color: rgba(248, 250, 252, 0.98);
  --el-button-active-border-color: rgba(100, 116, 139, 0.32);
  --el-button-active-text-color: #0f172a;
  --el-button-disabled-bg-color: rgba(255, 255, 255, 0.68);
  --el-button-disabled-border-color: rgba(148, 163, 184, 0.16);
  --el-button-disabled-text-color: #94a3b8;
  border-radius: 10px;
  box-shadow: none;
}

.pods-batchbar__action--logs {
  --el-button-border-color: rgba(59, 130, 246, 0.2);
  --el-button-text-color: #2563eb;
  --el-button-hover-bg-color: rgba(239, 246, 255, 0.96);
  --el-button-hover-border-color: rgba(59, 130, 246, 0.28);
  --el-button-hover-text-color: #1d4ed8;
  --el-button-active-bg-color: rgba(219, 234, 254, 0.9);
  --el-button-active-border-color: rgba(37, 99, 235, 0.34);
  --el-button-active-text-color: #1e40af;
}

.pods-batchbar__action--danger {
  --el-button-border-color: rgba(248, 113, 113, 0.24);
  --el-button-text-color: #dc2626;
  --el-button-hover-bg-color: rgba(254, 242, 242, 0.96);
  --el-button-hover-border-color: rgba(248, 113, 113, 0.32);
  --el-button-hover-text-color: #b91c1c;
  --el-button-active-bg-color: rgba(254, 226, 226, 0.92);
  --el-button-active-border-color: rgba(239, 68, 68, 0.34);
  --el-button-active-text-color: #991b1b;
}

.pods-batchbar__action--warning {
  --el-button-border-color: rgba(251, 191, 36, 0.28);
  --el-button-text-color: #d97706;
  --el-button-hover-bg-color: rgba(255, 247, 237, 0.96);
  --el-button-hover-border-color: rgba(245, 158, 11, 0.34);
  --el-button-hover-text-color: #b45309;
  --el-button-active-bg-color: rgba(254, 243, 199, 0.92);
  --el-button-active-border-color: rgba(217, 119, 6, 0.34);
  --el-button-active-text-color: #92400e;
}

:global(html.dark) .pods-batchbar {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.72);
}

:global(html.dark) .pods-batchbar__meta {
  color: #cbd5e1;
}

:global(html.dark) .pods-batchbar__action {
  --el-button-bg-color: rgba(15, 23, 42, 0.82);
  --el-button-border-color: rgba(148, 163, 184, 0.18);
  --el-button-disabled-border-color: rgba(148, 163, 184, 0.12);
  --el-button-disabled-text-color: rgba(148, 163, 184, 0.78);
}

:global(html.dark) .pods-batchbar__action--logs {
  --el-button-border-color: rgba(96, 165, 250, 0.26);
  --el-button-text-color: rgba(147, 197, 253, 0.96);
  --el-button-hover-bg-color: rgba(30, 41, 59, 0.96);
  --el-button-hover-border-color: rgba(96, 165, 250, 0.38);
  --el-button-hover-text-color: #dbeafe;
  --el-button-active-bg-color: rgba(37, 99, 235, 0.16);
  --el-button-active-border-color: rgba(96, 165, 250, 0.42);
  --el-button-active-text-color: #eff6ff;
}

:global(html.dark) .pods-batchbar__action--danger {
  --el-button-border-color: rgba(248, 113, 113, 0.26);
  --el-button-text-color: rgba(252, 165, 165, 0.96);
  --el-button-hover-bg-color: rgba(69, 10, 10, 0.38);
  --el-button-hover-border-color: rgba(248, 113, 113, 0.4);
  --el-button-hover-text-color: #fecaca;
  --el-button-active-bg-color: rgba(127, 29, 29, 0.42);
  --el-button-active-border-color: rgba(248, 113, 113, 0.44);
  --el-button-active-text-color: #fee2e2;
}

:global(html.dark) .pods-batchbar__action--warning {
  --el-button-border-color: rgba(251, 191, 36, 0.28);
  --el-button-text-color: rgba(253, 224, 71, 0.96);
  --el-button-hover-bg-color: rgba(120, 53, 15, 0.34);
  --el-button-hover-border-color: rgba(251, 191, 36, 0.4);
  --el-button-hover-text-color: #fde68a;
  --el-button-active-bg-color: rgba(146, 64, 14, 0.42);
  --el-button-active-border-color: rgba(251, 191, 36, 0.44);
  --el-button-active-text-color: #fef3c7;
}

:global(html.dark) .pods-phasebar__label {
  color: #94a3b8;
}

:global(html.dark) .pods-phasebar__value {
  color: #f8fafc;
}

:global(html.dark) .event-warning-badge {
  border-color: rgba(248, 113, 113, 0.18);
  background: rgba(127, 29, 29, 0.24);
  color: #fecaca;
}

@media (max-width: 960px) {
  .pods-batchbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .pods-batchbar__actions {
    justify-content: flex-start;
  }
}
</style>
