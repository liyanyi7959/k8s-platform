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
      <span class="k8s-name-wrap">
        <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
        <el-tag v-if="isDefaultClass(row)" size="small" type="warning" effect="light" class="k8s-default-tag">★ 默认</el-tag>
      </span>
    </template>
    <template #cell-allowExpansion="{ row }">
      <span :class="['k8s-status', row?.allowVolumeExpansion ? 'k8s-status--ok' : 'k8s-status--info']">{{ row?.allowVolumeExpansion ? 'yes' : 'no' }}</span>
    </template>
    <template #cell-default="{ row }">
      <span :class="['k8s-status', isDefaultClass(row) ? 'k8s-status--ok' : 'k8s-status--info']">{{ isDefaultClass(row) ? 'yes' : 'no' }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEditStorageClass(row)"><el-icon><Edit /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openStorageClassYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteStorageClassRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, Edit } from '@element-plus/icons-vue'
import { ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'provisioner', label: 'Provisioner', prop: 'provisioner', minWidth: 260, sortable: 'custom', defaultVisible: true },
  { key: 'reclaimPolicy', label: 'ReclaimPolicy', prop: 'reclaimPolicy', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'default', label: 'Default', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'bindingMode', label: 'BindingMode', prop: 'volumeBindingMode', width: 160, sortable: 'custom', defaultVisible: false },
  { key: 'allowExpansion', label: 'Expansion', prop: 'allowVolumeExpansion', width: 130, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 128, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function isDefaultClass(row: any): boolean {
  const ann = row?.metadata?.annotations
  if (!ann || typeof ann !== 'object') return false
  const v =
    ann['storageclass.kubernetes.io/is-default-class'] ??
    ann['storageclass.beta.kubernetes.io/is-default-class'] ??
    ann['is-default-class']
  return String(v ?? '').toLowerCase() === 'true'
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  openEditStorageClass: (row: any) => void
  openStorageClassYaml: (row: any) => void
  deleteStorageClassRow: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>

<style scoped>
.k8s-name-wrap {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.k8s-default-tag {
  flex-shrink: 0;
}
</style>
