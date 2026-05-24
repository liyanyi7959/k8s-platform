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
    <template #cell-namespace="{ row }">
      <span class="k8s-ns" :data-ns-color="nsColorIndex(String(row?.metadata?.namespace ?? ''))">{{ String(row?.metadata?.namespace ?? '-') }}</span>
    </template>
    <template #cell-name="{ row }">
      <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
    </template>
    <template #cell-rules="{ row }">
      <span class="k8s-num">{{ Array.isArray(row?.rules) ? row.rules.length : 0 }}</span>
    </template>
    <template #cell-summary="{ row }">
      <span class="k8s-age">{{ getSummary(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--info" @click="props.openRoleDetail(row)"><el-icon><View /></el-icon></button></el-tooltip>
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--edit" @click="props.openRoleEdit(row)"><el-icon><Edit /></el-icon></button></el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--violet" @click="props.openRoleYaml(row)"><el-icon><Document /></el-icon></button></el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteRoleRow(row)"><el-icon><Delete /></el-icon></button></el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, Edit, View } from '@element-plus/icons-vue'
import { ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'rules', label: 'Rules', width: 100, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'summary', label: 'Summary', minWidth: 280, defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 160, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function getSummary(row: any): string {
  const rules: any[] = Array.isArray(row?.rules) ? row.rules : []
  const first = rules[0] ?? {}
  const verbs = Array.isArray(first?.verbs) ? first.verbs.join(',') : ''
  const resources = Array.isArray(first?.resources) ? first.resources.join(',') : ''
  const apiGroups = Array.isArray(first?.apiGroups) ? first.apiGroups.join(',') : ''
  const head = [verbs, resources].filter(Boolean).join(' ')
  const prefix = apiGroups ? `[${apiGroups}] ` : ''
  const suffix = rules.length > 1 ? ` +${rules.length - 1}` : ''
  return (prefix + head + suffix).trim() || '-'
}

const props = defineProps<{ data: any[]; persistKey: string; showTools: boolean; canWrite: boolean; openRoleDetail: (row: any) => void; openRoleYaml: (row: any) => void; openRoleEdit: (row: any) => void; deleteRoleRow: (row: any) => void }>()
const emit = defineEmits<{ (e: 'sort-change', v: any): void }>()
const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>
