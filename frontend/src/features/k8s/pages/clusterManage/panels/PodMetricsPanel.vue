<template>
  <div class="pod-metrics-panel">
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
      <template #cell-containers="{ row }">
        <span class="k8s-num">{{ getContainerCount(row) }}</span>
      </template>
      <template #cell-cpu="{ row }">
        <span class="pod-metrics-usage">{{ formatPodCpu(row) }}</span>
      </template>
      <template #cell-memory="{ row }">
        <span class="pod-metrics-usage">{{ formatPodMemory(row) }}</span>
      </template>
      <template #cell-window="{ row }">
        <span class="pod-metrics-window">{{ String(row?.window ?? '-') }}</span>
      </template>
      <template #cell-timestamp="{ row }">
        <span class="pod-metrics-ts">{{ formatTs(row?.timestamp) }}</span>
      </template>
      <template #cell-age="{ row }">
        <span class="k8s-age">{{ getMetricAgeText(row) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="k8s-act-group">
          <el-tooltip content="指标详情" placement="top" :show-after="300">
            <button class="k8s-act-btn k8s-act-btn--info" @click="openDetail(row)"><el-icon><View /></el-icon></button>
          </el-tooltip>
          <el-tooltip content="Pod 详情" placement="top" :show-after="300">
            <button class="k8s-act-btn k8s-act-btn--cyan" @click="props.openPodDetail(row)"><el-icon><Link /></el-icon></button>
          </el-tooltip>
        </div>
      </template>

      <template #empty>
        <div v-if="props.unsupported" class="pod-metrics-panel__unsupported">
          <MetricsApiInstallHint description="当前集群未启用 Metrics API，PodMetrics 列表暂不可用。可以先参考下方命令部署 metrics-server，然后刷新当前页面。" />
        </div>
        <EmptyState v-else class="enhanced-table-empty-state" type="no-data" description="暂无数据" />
      </template>
    </EnhancedTable>

    <el-dialog v-model="detailVisible" title="PodMetrics 详情" width="760px" destroy-on-close @closed="closeDetail">
      <div v-if="detailRow" class="pod-metrics-detail">
        <div class="pod-metrics-detail__summary">
          <div class="pod-metrics-detail__card">
            <span class="pod-metrics-detail__label">Pod</span>
            <strong class="pod-metrics-detail__value">{{ String(detailRow?.metadata?.name ?? '-') }}</strong>
          </div>
          <div class="pod-metrics-detail__card">
            <span class="pod-metrics-detail__label">Namespace</span>
            <strong class="pod-metrics-detail__value">{{ String(detailRow?.metadata?.namespace ?? '-') }}</strong>
          </div>
          <div class="pod-metrics-detail__card">
            <span class="pod-metrics-detail__label">总 CPU</span>
            <strong class="pod-metrics-detail__value pod-metrics-usage">{{ formatPodCpu(detailRow) }}</strong>
          </div>
          <div class="pod-metrics-detail__card">
            <span class="pod-metrics-detail__label">总内存</span>
            <strong class="pod-metrics-detail__value pod-metrics-usage">{{ formatPodMemory(detailRow) }}</strong>
          </div>
          <div class="pod-metrics-detail__card">
            <span class="pod-metrics-detail__label">统计周期</span>
            <strong class="pod-metrics-detail__value">{{ String(detailRow?.window ?? '-') }}</strong>
          </div>
          <div class="pod-metrics-detail__card">
            <span class="pod-metrics-detail__label">采样时间</span>
            <strong class="pod-metrics-detail__value">{{ formatTs(detailRow?.timestamp) }}</strong>
          </div>
        </div>

        <el-table :data="detailContainerRows" stripe border size="small" class="pod-metrics-detail__table">
          <el-table-column prop="name" label="容器" min-width="180" />
          <el-table-column prop="cpu" label="CPU" width="140" align="center" header-align="center" />
          <el-table-column prop="memory" label="内存" width="160" align="center" header-align="center" />
        </el-table>
      </div>

      <template #footer>
        <el-space>
          <el-button @click="detailVisible = false">关闭</el-button>
          <el-button type="primary" :disabled="!detailRow" @click="openPodDetailFromDialog">打开 Pod 详情</el-button>
        </el-space>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { Link, View } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'

import MetricsApiInstallHint from '@/features/k8s/components/MetricsApiInstallHint.vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import EmptyState from '@/shared/components/EmptyState.vue'
import { formatAgeMs, formatTs, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'
import {
  formatBytes,
  formatMillicores,
  formatPodCpu,
  formatPodMemory,
  getContainerCount,
  getMetricContainers,
  parseCpuMillicores,
  parseMemoryBytes
} from '@/features/k8s/utils/podMetrics'

const columns: EnhancedColumn[] = [
  { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 170, sortable: 'custom', defaultVisible: true },
  { key: 'name', label: 'Pod', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'containers', label: '容器数', width: 100, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'cpu', label: 'CPU', width: 130, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'memory', label: '内存', width: 140, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'window', label: '统计周期', prop: 'window', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'timestamp', label: '采样时间', prop: 'timestamp', minWidth: 176, sortable: 'custom', defaultVisible: true },
  { key: 'age', label: 'AGE', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 124, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function getMetricAgeText(row: any): string {
  const ts = new Date(String(row?.timestamp ?? '')).getTime()
  if (!Number.isFinite(ts)) return '-'
  return formatAgeMs(Math.max(0, Date.now() - ts))
}

const detailVisible = ref(false)
const detailRow = ref<any | null>(null)

const detailContainerRows = computed(() => {
  return getMetricContainers(detailRow.value).map((container: any) => ({
    name: String(container?.name ?? '-'),
    cpu: formatMillicores(parseCpuMillicores(container?.usage?.cpu)),
    memory: formatBytes(parseMemoryBytes(container?.usage?.memory))
  }))
})

function openDetail(row: any) {
  detailRow.value = row
  detailVisible.value = true
}

function closeDetail() {
  detailRow.value = null
}

function openPodDetailFromDialog() {
  if (!detailRow.value) return
  const row = detailRow.value
  detailVisible.value = false
  props.openPodDetail(row)
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  openPodDetail: (row: any) => void
  unsupported: boolean
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>

<style scoped>
.pod-metrics-panel {
  display: flex;
  flex-direction: column;
}

.pod-metrics-panel__unsupported {
  padding: 24px 18px 28px;
}

.pod-metrics-usage {
  color: #0f4aa1;
  font-weight: 700;
}

.pod-metrics-window,
.pod-metrics-ts {
  color: var(--el-text-color-regular);
}

.pod-metrics-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.pod-metrics-detail__summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.pod-metrics-detail__card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.24);
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.96), rgba(241, 245, 249, 0.92));
}

.pod-metrics-detail__label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.pod-metrics-detail__value {
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.5;
}

@media (max-width: 900px) {
  .pod-metrics-detail__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .pod-metrics-detail__summary {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>