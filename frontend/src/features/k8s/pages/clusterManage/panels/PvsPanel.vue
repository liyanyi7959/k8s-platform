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
    <template #cell-capacity="{ row }">
      <span class="k8s-num">{{ formatStorage(row?.spec?.capacity?.storage) }}</span>
    </template>
    <template #cell-accessModes="{ row }">
      <span class="k8s-age">{{ formatAccessModes(row?.spec?.accessModes) }}</span>
    </template>
    <template #cell-claim="{ row }">
      <span class="k8s-ns">{{ props.formatClaimRef(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openPVDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openPVYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deletePVRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, View } from '@element-plus/icons-vue'
import { ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'phase', label: 'Phase', prop: 'status.phase', width: 140, sortable: 'custom', defaultVisible: true },
  { key: 'storageClass', label: 'StorageClass', prop: 'spec.storageClassName', minWidth: 200, sortable: 'custom', defaultVisible: true },
  { key: 'capacity', label: 'Capacity', prop: 'spec.capacity.storage', width: 130, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'reclaim', label: 'Reclaim', prop: 'spec.persistentVolumeReclaimPolicy', width: 130, sortable: 'custom', defaultVisible: true },
  { key: 'claim', label: 'Claim', prop: 'spec.claimRef.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'accessModes', label: 'AccessModes', minWidth: 170, defaultVisible: false },
  { key: 'actions', label: '操作', width: 128, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function formatAccessModes(modes: any): string {
  const arr: any[] = Array.isArray(modes) ? modes : []
  const parts = arr.map((it) => String(it ?? '').trim()).filter(Boolean)
  return parts.join(', ') || '-'
}

function formatStorage(v: any): string {
  const s = v != null ? String(v).trim() : ''
  return s || '-'
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  formatClaimRef: (row: any) => string
  openPVDetail: (row: any) => void
  openPVYaml: (row: any) => void
  deletePVRow: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>
