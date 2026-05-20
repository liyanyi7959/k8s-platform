<template>
  <div class="pods-panel">
    <div v-if="props.canWrite && selectedRows.length > 0" class="pods-batchbar">
      <div class="pods-batchbar__meta">已选 {{ selectedRows.length }} 个 Pod</div>
      <el-space wrap>
        <el-button size="small" type="danger" @click="props.bulkDeletePods(selectedRows)">删除选中</el-button>
        <el-button size="small" type="warning" plain @click="props.bulkDeletePods(selectedRows, { force: true })">强制删除</el-button>
      </el-space>
    </div>

    <div class="pods-phasebar">
      <button
        v-for="item in phaseCards"
        :key="item.key"
        type="button"
        :class="['pods-phasebar__item', `pods-phasebar__item--${item.tone}`, props.activePhaseFilter === item.key ? 'is-active' : '']"
        @click="togglePhaseFilter(item.key)"
      >
        <span class="pods-phasebar__label">{{ item.label }}</span>
        <strong class="pods-phasebar__value">{{ item.count }}</strong>
      </button>
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
import { computed, ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { nsColorIndex, getRestartsClass, getReadyNumClass } from '@/features/k8s/pages/ClusterManageView.utils'

type PodPhaseQuickFilter = '' | 'Running' | 'Pending' | 'Failed' | 'Completed'

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
  podPhaseSummary: Record<Exclude<PodPhaseQuickFilter, ''>, number>
  activePhaseFilter: PodPhaseQuickFilter
  getWarningEventCount: (row: any) => number
  getPodRowKey: (row: any) => string
  getPodPhaseTagType: (row: any) => string
  getPodPhaseText: (row: any) => string
  getPodReadyText: (row: any) => string
  getPodRestarts: (row: any) => number
  getPodAge: (row: any) => string
  openPodDetail: (row: any) => void
  openPodLogs: (row: any) => void
  openPodExec: (row: any) => void
  openPodYaml: (row: any) => void
  deletePodRow: (row: any) => void
  bulkDeletePods: (rows: any[], options?: { force?: boolean }) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
  (e: 'selection-change', rows: any[]): void
  (e: 'phase-filter-change', value: PodPhaseQuickFilter): void
}>()

const tableRef = ref<any>(null)
const selectedRows = ref<any[]>([])
const phaseCards = computed(() => [
  { key: 'Running' as const, label: 'Running', count: props.podPhaseSummary.Running ?? 0, tone: 'running' },
  { key: 'Pending' as const, label: 'Pending', count: props.podPhaseSummary.Pending ?? 0, tone: 'pending' },
  { key: 'Failed' as const, label: 'Failed', count: props.podPhaseSummary.Failed ?? 0, tone: 'failed' },
  { key: 'Completed' as const, label: 'Completed', count: props.podPhaseSummary.Completed ?? 0, tone: 'completed' }
])

function clearSelection() {
  selectedRows.value = []
  tableRef.value?.clearSelection?.()
}

function togglePhaseFilter(value: PodPhaseQuickFilter) {
  emit('phase-filter-change', props.activePhaseFilter === value ? '' : value)
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

.pods-phasebar {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.pods-phasebar__item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: flex-start;
  gap: 10px;
  min-height: 92px;
  padding: 14px 16px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(255, 255, 255, 0.94);
  color: #0f172a;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
}

.pods-phasebar__item::before {
  content: '';
  position: absolute;
  inset: 0 0 auto;
  height: 3px;
  background: var(--phase-tone, rgba(148, 163, 184, 0.7));
}

.pods-phasebar__item:hover {
  transform: translateY(-1px);
  border-color: rgba(59, 130, 246, 0.14);
  box-shadow: 0 10px 22px rgba(15, 23, 42, 0.05);
}

.pods-phasebar__item.is-active {
  border-color: rgba(59, 130, 246, 0.28);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.08);
}

.pods-phasebar__item--running {
  --phase-tone: #059669;
}

.pods-phasebar__item--pending {
  --phase-tone: #d97706;
}

.pods-phasebar__item--failed {
  --phase-tone: #dc2626;
}

.pods-phasebar__item--completed {
  --phase-tone: #64748b;
}

.pods-phasebar__label {
  font-size: 12px;
  font-weight: 700;
  color: #64748b;
}

.pods-phasebar__value {
  font-size: 28px;
  line-height: 1;
  letter-spacing: -0.02em;
}

:global(html.dark) .pods-phasebar__item {
  border-color: rgba(148, 163, 184, 0.16);
  background: rgba(15, 23, 42, 0.88);
  color: #e2e8f0;
}

:global(html.dark) .pods-batchbar {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.72);
}

:global(html.dark) .pods-batchbar__meta {
  color: #cbd5e1;
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
  .pods-phasebar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
