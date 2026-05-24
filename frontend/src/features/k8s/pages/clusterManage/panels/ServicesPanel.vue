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
    <template #cell-ports="{ row }">
      <span class="k8s-num">{{ props.formatPorts(row?.spec?.ports ?? []) }}</span>
    </template>
    <template #cell-externalIp="{ row }">
      <span class="k8s-age" :title="getExternalIpText(row)">{{ getExternalIpText(row) }}</span>
    </template>
    <template #cell-sessionAffinity="{ row }">
      <span class="k8s-status k8s-status--info">{{ getSessionAffinityText(row) }}</span>
    </template>
    <template #cell-selector="{ row }">
      <span class="k8s-age">{{ props.formatSelector(row?.spec?.selector ?? {}) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openServiceDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEditService(row)"><el-icon><Edit /></el-icon></button>
        </el-tooltip>
        <span v-if="props.canWrite" class="k8s-act-divider" />
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openServiceYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteServiceRow(row)"><el-icon><Delete /></el-icon></button>
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
  { key: 'type', label: 'Type', prop: 'spec.type', width: 140, sortable: 'custom', defaultVisible: true },
  { key: 'clusterIP', label: 'ClusterIP', prop: 'spec.clusterIP', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'externalIp', label: 'External IP', minWidth: 220, defaultVisible: true },
  { key: 'ports', label: 'Ports', minWidth: 220, defaultVisible: true },
  { key: 'sessionAffinity', label: 'Affinity', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'selector', label: 'Selector', minWidth: 220, defaultVisible: false },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 160, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function getExternalIpText(row: any): string {
  const values: string[] = []
  const specExternalIps: any[] = Array.isArray(row?.spec?.externalIPs) ? row.spec.externalIPs : []
  for (const item of specExternalIps) {
    const text = String(item ?? '').trim()
    if (text) values.push(text)
  }
  const lbIngress: any[] = Array.isArray(row?.status?.loadBalancer?.ingress) ? row.status.loadBalancer.ingress : []
  for (const item of lbIngress) {
    const text = String(item?.ip ?? item?.hostname ?? '').trim()
    if (text) values.push(text)
  }
  if (String(row?.spec?.type ?? '').trim() === 'ExternalName') {
    const externalName = String(row?.spec?.externalName ?? '').trim()
    if (externalName) values.push(externalName)
  }
  const uniq = Array.from(new Set(values))
  return uniq.join(', ') || '-'
}

function getSessionAffinityText(row: any): string {
  const value = String(row?.spec?.sessionAffinity ?? '').trim()
  return value || 'None'
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  formatPorts: (ports: any[]) => string
  formatSelector: (selector: Record<string, any>) => string
  openServiceDetail: (row: any) => void
  openServiceYaml: (row: any) => void
  openEditService: (row: any) => void
  deleteServiceRow: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>
