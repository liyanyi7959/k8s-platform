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
    <template #cell-type="{ row }">
      <el-tooltip :content="String(row?.type ?? '-')" placement="top" :show-after="300">
        <span class="k8s-type">{{ String(row?.type ?? '-') }}</span>
      </el-tooltip>
    </template>
    <template #cell-immutable="{ row }">
      <span :class="['k8s-status', row?.immutable ? 'k8s-status--warn' : 'k8s-status--info']">{{ row?.immutable ? 'yes' : 'no' }}</span>
    </template>
    <template #cell-dataKeys="{ row }">
      <span class="k8s-age">{{ props.getDataKeys(row).length }} 个字段</span>
    </template>
    <template #cell-labels="{ row }">
      <span class="k8s-num">{{ getLabelsCount(row) }}</span>
    </template>
    <template #cell-keys="{ row }">
      <span class="k8s-age">
        {{ props.getDataKeys(row).slice(0, 6).join(', ') }}
        <span v-if="props.getDataKeys(row).length > 6"> …</span>
      </span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openSecretDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canRevealSecret" content="查看内容" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--warn" @click="props.openSecretReveal(row)"><el-icon><Search /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEditSecret(row)"><el-icon><Edit /></el-icon></button>
        </el-tooltip>
        <span v-if="props.canWrite" class="k8s-act-divider" />
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openSecretYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteSecretRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, Edit, Search, View } from '@element-plus/icons-vue'
import { ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'type', label: 'Type', prop: 'type', width: 200, sortable: 'custom', defaultVisible: true },
  { key: 'immutable', label: 'Immutable', prop: 'immutable', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'dataKeys', label: '数据摘要', width: 140, defaultVisible: true },
  { key: 'labels', label: 'Labels', width: 90, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'keys', label: 'Keys', minWidth: 260, defaultVisible: false },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 196, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function getLabelsCount(row: any): number {
  const labels = row?.metadata?.labels
  if (!labels || typeof labels !== 'object' || Array.isArray(labels)) return 0
  return Object.keys(labels as Record<string, unknown>).length
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  canRevealSecret: boolean
  getDataKeys: (row: any) => string[]
  openSecretDetail: (row: any) => void
  openSecretReveal: (row: any) => void
  openSecretYaml: (row: any) => void
  openEditSecret: (row: any) => void
  deleteSecretRow: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>

<style scoped>
.k8s-type {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
