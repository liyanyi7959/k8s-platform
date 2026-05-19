<template>
  <div class="pvcs-panel">
    <div v-if="props.canWrite" class="pvcs-toolbar">
      <el-button type="primary" size="small" @click="createVisible = true">创建 PVC</el-button>
    </div>

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
    <template #cell-accessModes="{ row }">
      <span class="k8s-age">{{ formatAccessModes(row?.spec?.accessModes) }}</span>
    </template>
    <template #cell-request="{ row }">
      <span class="k8s-num">{{ formatStorage(row?.spec?.resources?.requests?.storage) }}</span>
    </template>
    <template #cell-capacity="{ row }">
      <span class="k8s-num">{{ formatStorage(row?.status?.capacity?.storage) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openPVCDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openPVCYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deletePVCRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
    </EnhancedTable>

    <CreatePvcDialog
      v-model="createVisible"
      :cluster-id="props.clusterId"
      :namespaces="props.namespaces"
      :default-namespace="props.defaultNamespace"
      @created="props.onCreated"
    />
  </div>
</template>

<script setup lang="ts">
import { Delete, Document, View } from '@element-plus/icons-vue'
import { ref } from 'vue'
import CreatePvcDialog from '@/features/k8s/pages/clusterManage/overlays/CreatePvcDialog.vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'phase', label: 'Phase', prop: 'status.phase', width: 140, sortable: 'custom', defaultVisible: true },
  { key: 'storageClass', label: 'StorageClass', prop: 'spec.storageClassName', minWidth: 200, sortable: 'custom', defaultVisible: true },
  { key: 'request', label: 'Request', prop: 'spec.resources.requests.storage', width: 130, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'capacity', label: 'Capacity', prop: 'status.capacity.storage', width: 130, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'accessModes', label: 'AccessModes', minWidth: 170, defaultVisible: false },
  { key: 'volume', label: 'Volume', prop: 'spec.volumeName', minWidth: 200, sortable: 'custom', defaultVisible: false },
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
  clusterId: number
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  namespaces: string[]
  defaultNamespace?: string
  openPVCDetail: (row: any) => void
  openPVCYaml: (row: any) => void
  deletePVCRow: (row: any) => void
  onCreated: () => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
const createVisible = ref(false)
defineExpose({ getTable: () => tableRef.value })
</script>

<style scoped>
.pvcs-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pvcs-toolbar {
  display: flex;
  justify-content: flex-end;
}
</style>
