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
    <template #cell-name="{ row }"><span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span></template>
    <template #cell-roleRef="{ row }"><span class="k8s-age">{{ getRoleRef(row) }}</span></template>
    <template #cell-subjects="{ row }"><span class="k8s-age">{{ getSubjects(row) }}</span></template>
    <template #cell-age="{ row }"><span class="k8s-age">{{ getCreationAgeText(row) }}</span></template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--info" @click="props.openClusterRoleBindingDetail(row)"><el-icon><View /></el-icon></button></el-tooltip>
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--edit" @click="props.openClusterRoleBindingEdit(row)"><el-icon><Edit /></el-icon></button></el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--violet" @click="props.openClusterRoleBindingYaml(row)"><el-icon><Document /></el-icon></button></el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteClusterRoleBindingRow(row)"><el-icon><Delete /></el-icon></button></el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, Edit, View } from '@element-plus/icons-vue'
import { ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 240, sortable: 'custom', defaultVisible: true },
  { key: 'roleRef', label: 'RoleRef', minWidth: 220, defaultVisible: true },
  { key: 'subjects', label: 'Subjects', minWidth: 320, defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 160, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function getRoleRef(row: any): string {
  const kind = String(row?.roleRef?.kind ?? '').trim()
  const name = String(row?.roleRef?.name ?? '').trim()
  return kind && name ? `${kind}/${name}` : '-'
}

function getSubjects(row: any): string {
  const items: any[] = Array.isArray(row?.subjects) ? row.subjects : []
  const text = items.slice(0, 2).map((it) => {
    const kind = String(it?.kind ?? '').trim()
    const name = String(it?.name ?? '').trim()
    const ns = String(it?.namespace ?? '').trim()
    return [kind && name ? `${kind}/${name}` : name || kind, ns ? `ns=${ns}` : ''].filter(Boolean).join(' ')
  }).filter(Boolean).join(', ')
  return items.length > 2 ? `${text} +${items.length - 2}` : text || '-'
}

const props = defineProps<{ data: any[]; persistKey: string; showTools: boolean; canWrite: boolean; openClusterRoleBindingDetail: (row: any) => void; openClusterRoleBindingYaml: (row: any) => void; openClusterRoleBindingEdit: (row: any) => void; deleteClusterRoleBindingRow: (row: any) => void }>()
const emit = defineEmits<{ (e: 'sort-change', v: any): void }>()
const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>