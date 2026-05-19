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
    <template #cell-parameters="{ row }">
      <span class="k8s-age">{{ formatParameters(row) }}</span>
    </template>
    <template #cell-default="{ row }">
      <span :class="['k8s-status', isDefaultClass(row) ? 'k8s-status--ok' : 'k8s-status--info']">{{ isDefaultClass(row) ? 'yes' : 'no' }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openIngressClassDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEditIngressClass(row)"><el-icon><Edit /></el-icon></button>
        </el-tooltip>
        <span class="k8s-act-divider" />
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openIngressClassYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteIngressClassRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
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
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'controller', label: 'Controller', prop: 'spec.controller', minWidth: 260, sortable: 'custom', defaultVisible: true },
  { key: 'default', label: 'Default', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'parameters', label: 'Parameters', minWidth: 220, defaultVisible: false },
  { key: 'actions', label: '操作', width: 160, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function formatParameters(row: any): string {
  const p = row?.spec?.parameters
  if (!p || typeof p !== 'object') return '-'
  const kind = p?.kind != null ? String(p.kind) : ''
  const name = p?.name != null ? String(p.name) : ''
  const scope = p?.scope != null ? String(p.scope) : ''
  const ns = p?.namespace != null ? String(p.namespace) : ''
  const head = kind && name ? `${kind}/${name}` : kind || name
  const tail = [scope, ns ? `ns=${ns}` : ''].filter(Boolean).join(' ')
  const text = [head, tail].filter(Boolean).join(' ')
  return text || '-'
}

function isDefaultClass(row: any): boolean {
  const ann = row?.metadata?.annotations
  if (!ann || typeof ann !== 'object') return false
  const v =
    ann['ingressclass.kubernetes.io/is-default-class'] ??
    ann['ingressclass.k8s.io/is-default-class'] ??
    ann['is-default-class']
  return String(v ?? '').toLowerCase() === 'true'
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  openIngressClassDetail: (row: any) => void
  openIngressClassYaml: (row: any) => void
  openEditIngressClass: (row: any) => void
  deleteIngressClassRow: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>
