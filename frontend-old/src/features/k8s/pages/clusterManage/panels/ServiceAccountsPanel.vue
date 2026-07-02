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
    <template #cell-secretsCount="{ row }">
      <span class="k8s-num">{{ Array.isArray(row?.secrets) ? row.secrets.length : 0 }}</span>
    </template>
    <template #cell-imagePullSecretsCount="{ row }">
      <span class="k8s-num">{{ Array.isArray(row?.imagePullSecrets) ? row.imagePullSecrets.length : 0 }}</span>
    </template>
    <template #cell-automount="{ row }">
      <span :class="['k8s-status', getAutomountTagClass(row)]">{{ getAutomountText(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openServiceAccountDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEditServiceAccount(row)"><el-icon><Edit /></el-icon></button>
        </el-tooltip>
        <span v-if="props.canWrite" class="k8s-act-divider" />
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openServiceAccountYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteServiceAccountRow(row)"><el-icon><Delete /></el-icon></button>
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
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 240, sortable: 'custom', defaultVisible: true },
  { key: 'secretsCount', label: 'Secrets', width: 110, defaultVisible: true, align: 'center', headerAlign: 'center' },
  { key: 'imagePullSecretsCount', label: 'PullSecrets', width: 120, defaultVisible: true, align: 'center', headerAlign: 'center' },
  { key: 'automount', label: 'Automount', width: 120, defaultVisible: true, align: 'center', headerAlign: 'center' },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 160, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function getAutomountText(row: any): string {
  const value = row?.automountServiceAccountToken
  if (value === true) return 'yes'
  if (value === false) return 'no'
  return 'default'
}

function getAutomountTagClass(row: any): string {
  const value = row?.automountServiceAccountToken
  if (value === true) return 'k8s-status--ok'
  if (value === false) return 'k8s-status--warn'
  return 'k8s-status--info'
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  openServiceAccountDetail: (row: any) => void
  openServiceAccountYaml: (row: any) => void
  openEditServiceAccount: (row: any) => void
  deleteServiceAccountRow: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>

