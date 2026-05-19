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
    <template #cell-class="{ row }">
      <span class="k8s-num">{{ String(row?.spec?.ingressClassName ?? '-') }}</span>
    </template>
    <template #cell-hosts="{ row }">
      <span class="k8s-age">{{ props.getHosts(row).join(', ') || '-' }}</span>
    </template>
    <template #cell-address="{ row }">
      <span class="k8s-age">{{ getAddress(row) }}</span>
    </template>
    <template #cell-rules="{ row }">
      <span class="k8s-age">{{ props.formatRules(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openIngressDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEditIngress(row)"><el-icon><Edit /></el-icon></button>
        </el-tooltip>
        <span class="k8s-act-divider" />
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openIngressYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteIngressRow(row)"><el-icon><Delete /></el-icon></button>
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
import { getCreationAgeText, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'class', label: 'Class', prop: 'spec.ingressClassName', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'hosts', label: 'Hosts', minWidth: 260, defaultVisible: true },
  { key: 'address', label: 'Address', minWidth: 200, defaultVisible: true },
  { key: 'rules', label: 'Rules', minWidth: 360, defaultVisible: false },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 160, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function getAddress(row: any): string {
  const ing = row?.status?.loadBalancer?.ingress
  if (!Array.isArray(ing) || ing.length === 0) return '-'
  const parts = ing
    .map((x: any) => String(x?.hostname ?? x?.ip ?? '').trim())
    .filter((x: string) => Boolean(x))
  return parts.join(', ') || '-'
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  getHosts: (row: any) => string[]
  formatRules: (row: any) => string
  openIngressDetail: (row: any) => void
  openIngressYaml: (row: any) => void
  openEditIngress: (row: any) => void
  deleteIngressRow: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>
