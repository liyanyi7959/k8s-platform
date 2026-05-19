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
    <el-table-column v-if="showResourceQuotaUsage" type="expand" width="44">
      <template #default="{ row }">
        <div class="quota-expand">
          <div v-if="getResourceQuotaDetails(row).length" class="quota-expand-table">
            <div class="quota-expand-head">
              <span>资源</span>
              <span>已使用</span>
              <span>上限</span>
              <span>进度</span>
            </div>
            <div v-for="detail in getResourceQuotaDetails(row)" :key="detail.key" class="quota-expand-row">
              <span class="quota-resource">{{ detail.key }}</span>
              <span class="quota-value">{{ detail.usedText }}</span>
              <span class="quota-value">{{ detail.hardText }}</span>
              <div class="quota-progress-cell">
                <div class="quota-progress-meta">
                  <span class="quota-progress-text">{{ formatQuotaPercent(detail.percent) }}</span>
                </div>
                <el-progress
                  :percentage="capQuotaPercent(detail.percent)"
                  :stroke-width="8"
                  :show-text="false"
                  :color="getQuotaProgressColor(detail.percent)"
                />
              </div>
            </div>
          </div>
          <div v-else class="quota-empty">当前未返回可计算的配额使用量</div>
        </div>
      </template>
    </el-table-column>
    <template #cell-namespace="{ row }">
      <span class="k8s-ns" :data-ns-color="nsColorIndex(String(row?.metadata?.namespace ?? ''))">{{ String(row?.metadata?.namespace ?? '-') }}</span>
    </template>
    <template #cell-name="{ row }">
      <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
    </template>
    <template v-if="showResourceQuotaUsage" #cell-utilization="{ row }">
      <div v-if="getResourceQuotaPeak(row)" class="quota-peak">
        <div class="quota-peak-head">
          <span class="quota-peak-key">{{ getResourceQuotaPeak(row)?.key }}</span>
          <span class="quota-peak-text">{{ formatQuotaPercent(getResourceQuotaPeak(row)?.percent ?? null) }}</span>
        </div>
        <el-progress
          :percentage="capQuotaPercent(getResourceQuotaPeak(row)?.percent ?? null)"
          :stroke-width="8"
          :show-text="false"
          :color="getQuotaProgressColor(getResourceQuotaPeak(row)?.percent ?? null)"
        />
      </div>
      <span v-else class="k8s-age">-</span>
    </template>
    <template #cell-summary="{ row }">
      <span class="k8s-age">{{ getSummary(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="YAML" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--violet" @click="openYaml(row)"><el-icon><Document /></el-icon></button></el-tooltip>
        <el-tooltip v-if="canWrite" content="编辑" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--edit" @click="openEdit(row)"><el-icon><Edit /></el-icon></button></el-tooltip>
        <el-tooltip v-if="canWrite" content="删除" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--danger" @click="deleteRow(row)"><el-icon><Delete /></el-icon></button></el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, Edit } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'
import { parseSiValue } from '@/shared/utils/parseSiValue'

type ResourceQuotaDetail = {
  key: string
  usedText: string
  hardText: string
  percent: number | null
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  getSummary: (row: any) => string
  openYaml: (row: any) => void
  openEdit: (row: any) => void
  deleteRow: (row: any) => void
  showResourceQuotaUsage?: boolean
}>()

const columns = computed<EnhancedColumn[]>(() => {
  const base: EnhancedColumn[] = [
    { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
    { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true }
  ]
  if (props.showResourceQuotaUsage) {
    base.push({ key: 'utilization', label: '最高使用率', minWidth: 220, defaultVisible: true })
  }
  base.push(
    { key: 'summary', label: '摘要', minWidth: props.showResourceQuotaUsage ? 280 : 360, defaultVisible: true },
    { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
    { key: 'actions', label: '操作', width: 128, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
  )
  return base
})

function getResourceQuotaDetails(row: any): ResourceQuotaDetail[] {
  if (!props.showResourceQuotaUsage) return []
  const hard = row?.status?.hard ?? row?.spec?.hard ?? {}
  const used = row?.status?.used ?? {}
  return Object.keys(hard)
    .map((key) => {
      const hardText = String(hard?.[key] ?? '')
      const usedText = String(used?.[key] ?? '0')
      const hardValue = parseSiValue(hardText)
      const usedValue = parseSiValue(usedText)
      const percent = Number.isFinite(hardValue) && hardValue > 0 && Number.isFinite(usedValue)
        ? (usedValue / hardValue) * 100
        : null
      return { key, usedText, hardText, percent }
    })
    .sort((a, b) => (b.percent ?? -1) - (a.percent ?? -1) || a.key.localeCompare(b.key, 'zh-Hans-CN'))
}

function getResourceQuotaPeak(row: any): ResourceQuotaDetail | null {
  const [first] = getResourceQuotaDetails(row)
  return first ?? null
}

function capQuotaPercent(percent: number | null): number {
  if (percent == null || !Number.isFinite(percent)) return 0
  return Math.max(0, Math.min(percent, 100))
}

function formatQuotaPercent(percent: number | null): string {
  if (percent == null || !Number.isFinite(percent)) return '-'
  return `${percent >= 100 ? percent.toFixed(0) : percent.toFixed(1)}%`
}

function getQuotaProgressColor(percent: number | null): string {
  if (percent != null && percent > 95) return '#dc2626'
  if (percent != null && percent > 80) return '#f59e0b'
  return '#16a34a'
}

const emit = defineEmits<{ (e: 'sort-change', v: any): void }>()
const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>

<style scoped>
.quota-peak {
  min-width: 0;
}

.quota-peak-head,
.quota-progress-meta,
.quota-expand-head,
.quota-expand-row {
  display: grid;
  gap: 12px;
}

.quota-peak-head {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  margin-bottom: 6px;
}

.quota-peak-key,
.quota-resource,
.quota-value,
.quota-peak-text,
.quota-progress-text {
  font-size: 12px;
}

.quota-peak-key,
.quota-resource {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.quota-peak-text,
.quota-progress-text {
  color: var(--el-text-color-regular);
  font-variant-numeric: tabular-nums;
}

.quota-expand {
  padding: 6px 8px 2px;
}

.quota-expand-table {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  overflow: hidden;
  background: var(--el-bg-color-overlay);
}

.quota-expand-head,
.quota-expand-row {
  grid-template-columns: minmax(180px, 1.2fr) minmax(120px, 0.8fr) minmax(120px, 0.8fr) minmax(220px, 1.4fr);
  align-items: center;
  padding: 10px 14px;
}

.quota-expand-head {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.quota-expand-row {
  border-top: 1px solid var(--el-border-color-lighter);
}

.quota-progress-cell {
  min-width: 0;
}

.quota-progress-meta {
  grid-template-columns: auto;
  justify-content: end;
  margin-bottom: 4px;
}

.quota-empty {
  padding: 10px 14px;
  border: 1px dashed var(--el-border-color);
  border-radius: 10px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
}

@media (max-width: 960px) {
  .quota-expand-head,
  .quota-expand-row {
    grid-template-columns: minmax(120px, 1fr) minmax(90px, 0.8fr) minmax(90px, 0.8fr) minmax(140px, 1.2fr);
    gap: 10px;
    padding: 10px 12px;
  }
}
</style>
