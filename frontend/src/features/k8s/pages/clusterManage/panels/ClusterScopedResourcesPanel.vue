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
    <template #cell-summary="{ row }">
      <span class="k8s-age">{{ props.getSummary(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEdit(row)"><el-icon><Edit /></el-icon></button></el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--violet" @click="props.openYaml(row)"><el-icon><Document /></el-icon></button></el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteRow(row)"><el-icon><Delete /></el-icon></button></el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, Edit } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  summaryLabel?: string
  getSummary: (row: any) => string
  openYaml: (row: any) => void
  openEdit: (row: any) => void
  deleteRow: (row: any) => void
}>()

const emit = defineEmits<{ (e: 'sort-change', v: any): void }>()

const columns = computed<EnhancedColumn[]>(() => [
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 280, sortable: 'custom', defaultVisible: true },
  { key: 'summary', label: props.summaryLabel || '摘要', minWidth: 420, defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 128, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
])

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>
