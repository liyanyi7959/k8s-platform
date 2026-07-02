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
    <el-table-column v-if="showResourceQuotaUsage" type="expand" width="44">
      <template #default="{ row }">
        <div class="quota-expand">
          <div v-if="getResourceQuotaDetails(row).length" class="quota-expand-table">
            <div class="quota-expand-head">
              <span>资源</span>
              <span>已使用</span>
              <span>上限</span>
              <span>进度</span>
            </div>
            <div v-for="detail in getResourceQuotaDetails(row)" :key="detail.key" class="quota-expand-row">
              <span class="quota-resource">{{ detail.key }}</span>
              <span class="quota-value">{{ detail.usedText }}</span>
              <span class="quota-value">{{ detail.hardText }}</span>
              <div class="quota-progress-cell">
                <div class="quota-progress-meta">
                  <span class="quota-progress-text">{{ formatQuotaPercent(detail.percent) }}</span>
                </div>
                <el-progress
                  :percentage="capQuotaPercent(detail.percent)"
                  :stroke-width="8"
                  :show-text="false"
                  :color="getQuotaProgressColor(detail.percent)"
                />
              </div>
            </div>
          </div>
          <div v-else class="quota-empty">当前未返回可计算的配额使用量</div>
        </div>
      </template>
    </el-table-column>
    <template #cell-namespace="{ row }">
      <span class="k8s-ns" :data-ns-color="nsColorIndex(String(row?.metadata?.namespace ?? ''))">{{ String(row?.metadata?.namespace ?? '-') }}</span>
    </template>
    <template #cell-name="{ row }">
      <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
    </template>
    <template v-if="showNetworkPolicyColumns" #cell-policyTypes="{ row }">
      <span class="k8s-age">{{ getNetworkPolicyTypesText(row) }}</span>
    </template>
    <template v-if="showNetworkPolicyColumns" #cell-selector="{ row }">
      <span class="k8s-age" :title="getNetworkPolicySelectorText(row)">{{ getNetworkPolicySelectorText(row) }}</span>
    </template>
    <template v-if="showNetworkPolicyColumns" #cell-ingressRules="{ row }">
      <span class="k8s-num">{{ getNetworkPolicyRuleCount(row, 'ingress') }}</span>
    </template>
    <template v-if="showNetworkPolicyColumns" #cell-egressRules="{ row }">
      <span class="k8s-num">{{ getNetworkPolicyRuleCount(row, 'egress') }}</span>
    </template>
    <template v-if="showEndpointsColumns || showEndpointSliceColumns" #cell-service="{ row }">
      <span class="k8s-age" :title="getServiceNameText(row)">{{ getServiceNameText(row) }}</span>
    </template>
    <template v-if="showEndpointsColumns" #cell-ready="{ row }">
      <span class="k8s-num">{{ getEndpointsAddressCount(row, 'addresses') }}</span>
    </template>
    <template v-if="showEndpointsColumns" #cell-notReady="{ row }">
      <span class="k8s-num">{{ getEndpointsAddressCount(row, 'notReadyAddresses') }}</span>
    </template>
    <template v-if="showEndpointsColumns || showEndpointSliceColumns" #cell-ports="{ row }">
      <span class="k8s-num">{{ getPortsCount(row) }}</span>
    </template>
    <template v-if="showEndpointSliceColumns" #cell-addressType="{ row }">
      <span class="k8s-age">{{ getAddressTypeText(row) }}</span>
    </template>
    <template v-if="showEndpointSliceColumns" #cell-endpointsCount="{ row }">
      <span class="k8s-num">{{ getEndpointSliceCount(row) }}</span>
    </template>
    <template v-if="showReplicaSetColumns" #cell-desired="{ row }">
      <span class="k8s-num">{{ getReplicaSetDesired(row) }}</span>
    </template>
    <template v-if="showReplicaSetColumns" #cell-readyReplicas="{ row }">
      <span class="k8s-num">{{ getReplicaSetReady(row) }}</span>
    </template>
    <template v-if="showReplicaSetColumns" #cell-availableReplicas="{ row }">
      <span class="k8s-num">{{ getReplicaSetAvailable(row) }}</span>
    </template>
    <template v-if="showVolumeSnapshotColumns" #cell-readyToUse="{ row }">
      <span :class="['k8s-status', getVolumeSnapshotReady(row) ? 'k8s-status--ok' : 'k8s-status--warn']">{{ getVolumeSnapshotReadyText(row) }}</span>
    </template>
    <template v-if="showVolumeSnapshotColumns" #cell-sourcePvc="{ row }">
      <span class="k8s-age" :title="getVolumeSnapshotSourcePvcText(row)">{{ getVolumeSnapshotSourcePvcText(row) }}</span>
    </template>
    <template v-if="showVolumeSnapshotColumns" #cell-snapshotClass="{ row }">
      <span class="k8s-age" :title="getVolumeSnapshotClassText(row)">{{ getVolumeSnapshotClassText(row) }}</span>
    </template>
    <template v-if="showVolumeSnapshotColumns" #cell-boundContent="{ row }">
      <span class="k8s-age" :title="getVolumeSnapshotContentText(row)">{{ getVolumeSnapshotContentText(row) }}</span>
    </template>
    <template v-if="showResourceQuotaColumns" #cell-quotaResources="{ row }">
      <span class="k8s-age" :title="getResourceQuotaKeysText(row)">{{ getResourceQuotaKeysCount(row) }} 项</span>
    </template>
    <template v-if="showResourceQuotaColumns" #cell-quotaScopes="{ row }">
      <span class="k8s-age" :title="getResourceQuotaScopesText(row)">{{ getResourceQuotaScopesText(row) }}</span>
    </template>
    <template v-if="showLimitRangeColumns" #cell-limitEntries="{ row }">
      <span class="k8s-num">{{ getLimitRangeEntriesCount(row) }}</span>
    </template>
    <template v-if="showLimitRangeColumns" #cell-limitTypes="{ row }">
      <span class="k8s-age" :title="getLimitRangeTypesText(row)">{{ getLimitRangeTypesText(row) }}</span>
    </template>
    <template v-if="showResourceQuotaUsage" #cell-utilization="{ row }">
      <div v-if="getResourceQuotaPeak(row)" class="quota-peak">
        <div class="quota-peak-head">
          <span class="quota-peak-key">{{ getResourceQuotaPeak(row)?.key }}</span>
          <span class="quota-peak-text">{{ formatQuotaPercent(getResourceQuotaPeak(row)?.percent ?? null) }}</span>
        </div>
        <el-progress
          :percentage="capQuotaPercent(getResourceQuotaPeak(row)?.percent ?? null)"
          :stroke-width="8"
          :show-text="false"
          :color="getQuotaProgressColor(getResourceQuotaPeak(row)?.percent ?? null)"
        />
      </div>
      <span v-else class="k8s-age">-</span>
    </template>
    <template #cell-summary="{ row }">
      <span class="k8s-age">{{ getSummary(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip v-if="props.openDetail" content="详情" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--info" @click="props.openDetail(row)"><el-icon><View /></el-icon></button></el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--violet" @click="openYaml(row)"><el-icon><Document /></el-icon></button></el-tooltip>
        <el-tooltip v-if="canWrite" content="编辑" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--edit" @click="openEdit(row)"><el-icon><Edit /></el-icon></button></el-tooltip>
        <el-tooltip v-if="canWrite" content="删除" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--danger" @click="deleteRow(row)"><el-icon><Delete /></el-icon></button></el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, Edit, View } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'
import { parseSiValue } from '@/shared/utils/parseSiValue'

type ResourceQuotaDetail = {
  key: string
  usedText: string
  hardText: string
  percent: number | null
}

type GovernanceResourceKind = 'networkpolicies' | 'endpoints' | 'endpointslices' | 'replicasets' | 'volumesnapshots' | 'resourcequotas' | 'limitranges'

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  getSummary: (row: any) => string
  openYaml: (row: any) => void
  openEdit: (row: any) => void
  deleteRow: (row: any) => void
  openDetail?: (row: any) => void
  showResourceQuotaUsage?: boolean
  resourceKind?: GovernanceResourceKind
}>()

const showNetworkPolicyColumns = computed(() => props.resourceKind === 'networkpolicies')
const showEndpointsColumns = computed(() => props.resourceKind === 'endpoints')
const showEndpointSliceColumns = computed(() => props.resourceKind === 'endpointslices')
const showReplicaSetColumns = computed(() => props.resourceKind === 'replicasets')
const showVolumeSnapshotColumns = computed(() => props.resourceKind === 'volumesnapshots')
const showResourceQuotaColumns = computed(() => props.resourceKind === 'resourcequotas')
const showLimitRangeColumns = computed(() => props.resourceKind === 'limitranges')
const showStructuredSummary = computed(() => !(
  showNetworkPolicyColumns.value ||
  showEndpointsColumns.value ||
  showEndpointSliceColumns.value ||
  showReplicaSetColumns.value ||
  showVolumeSnapshotColumns.value ||
  showResourceQuotaColumns.value ||
  showLimitRangeColumns.value
))

const columns = computed<EnhancedColumn[]>(() => {
  const base: EnhancedColumn[] = [
    { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
    { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true }
  ]
  if (showNetworkPolicyColumns.value) {
    base.push(
      { key: 'policyTypes', label: 'PolicyTypes', minWidth: 160, defaultVisible: true },
      { key: 'selector', label: 'Pod Selector', minWidth: 220, defaultVisible: true },
      { key: 'ingressRules', label: 'Ingress', width: 96, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'egressRules', label: 'Egress', width: 96, align: 'center', headerAlign: 'center', defaultVisible: true }
    )
  }
  if (showEndpointsColumns.value) {
    base.push(
      { key: 'service', label: 'Service', minWidth: 200, defaultVisible: true },
      { key: 'ready', label: 'Ready', width: 96, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'notReady', label: 'NotReady', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'ports', label: 'Ports', width: 90, align: 'center', headerAlign: 'center', defaultVisible: true }
    )
  }
  if (showEndpointSliceColumns.value) {
    base.push(
      { key: 'service', label: 'Service', minWidth: 200, defaultVisible: true },
      { key: 'addressType', label: 'AddressType', width: 130, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'endpointsCount', label: 'Endpoints', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'ports', label: 'Ports', width: 90, align: 'center', headerAlign: 'center', defaultVisible: true }
    )
  }
  if (showReplicaSetColumns.value) {
    base.push(
      { key: 'desired', label: 'Desired', width: 96, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'readyReplicas', label: 'Ready', width: 96, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'availableReplicas', label: 'Available', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true }
    )
  }
  if (showVolumeSnapshotColumns.value) {
    base.push(
      { key: 'readyToUse', label: 'Ready', width: 96, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'sourcePvc', label: 'Source PVC', minWidth: 180, defaultVisible: true },
      { key: 'snapshotClass', label: 'Class', minWidth: 180, defaultVisible: true },
      { key: 'boundContent', label: 'Bound Content', minWidth: 220, defaultVisible: true }
    )
  }
  if (props.showResourceQuotaUsage) {
    if (showResourceQuotaColumns.value) {
      base.push(
        { key: 'quotaResources', label: '配额项', width: 100, align: 'center', headerAlign: 'center', defaultVisible: true },
        { key: 'quotaScopes', label: 'Scopes', minWidth: 180, defaultVisible: true }
      )
    }
    base.push({ key: 'utilization', label: '最高使用率', minWidth: 220, defaultVisible: true })
  }
  if (showLimitRangeColumns.value) {
    base.push(
      { key: 'limitEntries', label: 'Entries', width: 96, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'limitTypes', label: 'Types', minWidth: 220, defaultVisible: true }
    )
  }
  base.push({
    key: 'summary',
    label: '摘要',
    minWidth: props.showResourceQuotaUsage ? 280 : 360,
    defaultVisible: showStructuredSummary.value
  })
  base.push(
    { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
    { key: 'actions', label: '操作', width: props.openDetail ? 160 : 128, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
  )
  return base
})

function formatPreviewList(values: string[], limit = 3): string {
  const items = values.map((item) => item.trim()).filter(Boolean)
  if (!items.length) return '-'
  if (items.length <= limit) return items.join(', ')
  return `${items.slice(0, limit).join(', ')} +${items.length - limit}`
}

function formatSelectorText(record: unknown, emptyText = '-'): string {
  if (!record || typeof record !== 'object' || Array.isArray(record)) return emptyText
  const parts = Object.entries(record as Record<string, unknown>)
    .map(([key, value]) => `${key}=${String(value ?? '')}`)
    .filter((item) => !item.endsWith('='))
  return parts.length ? parts.join(', ') : emptyText
}

function getNetworkPolicyTypesText(row: any): string {
  const types = Array.isArray(row?.spec?.policyTypes)
    ? row.spec.policyTypes.map((item: unknown) => String(item ?? '').trim()).filter(Boolean)
    : []
  if (types.length) return types.join(', ')
  const inferred: string[] = []
  if (Array.isArray(row?.spec?.ingress)) inferred.push('Ingress')
  if (Array.isArray(row?.spec?.egress)) inferred.push('Egress')
  return inferred.length ? inferred.join(', ') : 'Ingress'
}

function getNetworkPolicySelectorText(row: any): string {
  return formatSelectorText(row?.spec?.podSelector?.matchLabels, 'all pods')
}

function getNetworkPolicyRuleCount(row: any, key: 'ingress' | 'egress'): number {
  return Array.isArray(row?.spec?.[key]) ? row.spec[key].length : 0
}

function getServiceNameText(row: any): string {
  const serviceName = String(row?.metadata?.labels?.['kubernetes.io/service-name'] ?? '').trim()
  if (serviceName) return serviceName
  if (showEndpointsColumns.value) {
    return String(row?.metadata?.name ?? '-').trim() || '-'
  }
  return '-'
}

function getEndpointsAddressCount(row: any, key: 'addresses' | 'notReadyAddresses'): number {
  const subsets = Array.isArray(row?.subsets) ? row.subsets : []
  return subsets.reduce((sum: number, subset: any) => sum + (Array.isArray(subset?.[key]) ? subset[key].length : 0), 0)
}

function getPortsCount(row: any): number {
  if (showEndpointSliceColumns.value) {
    return Array.isArray(row?.ports) ? row.ports.length : 0
  }
  const subsets = Array.isArray(row?.subsets) ? row.subsets : []
  return subsets.reduce((sum: number, subset: any) => sum + (Array.isArray(subset?.ports) ? subset.ports.length : 0), 0)
}

function getAddressTypeText(row: any): string {
  return String(row?.addressType ?? '-').trim() || '-'
}

function getEndpointSliceCount(row: any): number {
  return Array.isArray(row?.endpoints) ? row.endpoints.length : 0
}

function getReplicaSetDesired(row: any): number {
  const desired = row?.spec?.replicas ?? row?.status?.replicas
  return Number.isFinite(Number(desired)) ? Number(desired) : 0
}

function getReplicaSetReady(row: any): number {
  const ready = row?.status?.readyReplicas
  return Number.isFinite(Number(ready)) ? Number(ready) : 0
}

function getReplicaSetAvailable(row: any): number {
  const available = row?.status?.availableReplicas
  return Number.isFinite(Number(available)) ? Number(available) : 0
}

function getVolumeSnapshotReady(row: any): boolean | null {
  if (typeof row?.status?.readyToUse === 'boolean') return row.status.readyToUse
  return null
}

function getVolumeSnapshotReadyText(row: any): string {
  const ready = getVolumeSnapshotReady(row)
  if (ready == null) return '-'
  return ready ? 'yes' : 'no'
}

function getVolumeSnapshotSourcePvcText(row: any): string {
  return String(row?.spec?.source?.persistentVolumeClaimName ?? '-').trim() || '-'
}

function getVolumeSnapshotClassText(row: any): string {
  return String(row?.spec?.volumeSnapshotClassName ?? '-').trim() || '-'
}

function getVolumeSnapshotContentText(row: any): string {
  return String(row?.status?.boundVolumeSnapshotContentName ?? '-').trim() || '-'
}

function getResourceQuotaHard(row: any): Record<string, unknown> {
  const hard = row?.status?.hard ?? row?.spec?.hard
  if (!hard || typeof hard !== 'object' || Array.isArray(hard)) return {}
  return hard as Record<string, unknown>
}

function getResourceQuotaKeysCount(row: any): number {
  return Object.keys(getResourceQuotaHard(row)).length
}

function getResourceQuotaKeysText(row: any): string {
  return formatPreviewList(Object.keys(getResourceQuotaHard(row)), 4)
}

function getResourceQuotaScopesText(row: any): string {
  const scopes = Array.isArray(row?.spec?.scopes)
    ? row.spec.scopes.map((item: unknown) => String(item ?? '').trim()).filter(Boolean)
    : []
  if (scopes.length) return formatPreviewList(scopes, 3)
  const expressions = Array.isArray(row?.spec?.scopeSelector?.matchExpressions) ? row.spec.scopeSelector.matchExpressions : []
  if (expressions.length) return `selector ${expressions.length} 条`
  return '-'
}

function getLimitRangeEntries(row: any): any[] {
  return Array.isArray(row?.spec?.limits) ? row.spec.limits : []
}

function getLimitRangeEntriesCount(row: any): number {
  return getLimitRangeEntries(row).length
}

function getLimitRangeTypesText(row: any): string {
  const types = Array.from(new Set(getLimitRangeEntries(row)
    .map((item: any) => String(item?.type ?? '').trim())
    .filter(Boolean)))
  return formatPreviewList(types, 3)
}

function getResourceQuotaDetails(row: any): ResourceQuotaDetail[] {
  if (!props.showResourceQuotaUsage) return []
  const hard = getResourceQuotaHard(row)
  const used = row?.status?.used ?? {}
  return Object.keys(hard)
    .map((key) => {
      const hardText = String(hard?.[key] ?? '')
      const usedText = String(used?.[key] ?? '0')
      const hardValue = parseSiValue(hardText)
      const usedValue = parseSiValue(usedText)
      const percent = Number.isFinite(hardValue) && hardValue > 0 && Number.isFinite(usedValue)
        ? (usedValue / hardValue) * 100
        : null
      return { key, usedText, hardText, percent }
    })
    .sort((a, b) => (b.percent ?? -1) - (a.percent ?? -1) || a.key.localeCompare(b.key, 'zh-Hans-CN'))
}

function getResourceQuotaPeak(row: any): ResourceQuotaDetail | null {
  const [first] = getResourceQuotaDetails(row)
  return first ?? null
}

function capQuotaPercent(percent: number | null): number {
  if (percent == null || !Number.isFinite(percent)) return 0
  return Math.max(0, Math.min(percent, 100))
}

function formatQuotaPercent(percent: number | null): string {
  if (percent == null || !Number.isFinite(percent)) return '-'
  return `${percent >= 100 ? percent.toFixed(0) : percent.toFixed(1)}%`
}

function getQuotaProgressColor(percent: number | null): string {
  if (percent != null && percent > 95) return '#dc2626'
  if (percent != null && percent > 80) return '#f59e0b'
  return '#16a34a'
}

const emit = defineEmits<{ (e: 'sort-change', v: any): void }>()
const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>

<style scoped>
.quota-peak {
  min-width: 0;
}

.quota-peak-head,
.quota-progress-meta,
.quota-expand-head,
.quota-expand-row {
  display: grid;
  gap: 12px;
}

.quota-peak-head {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  margin-bottom: 6px;
}

.quota-peak-key,
.quota-resource,
.quota-value,
.quota-peak-text,
.quota-progress-text {
  font-size: 12px;
}

.quota-peak-key,
.quota-resource {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.quota-peak-text,
.quota-progress-text {
  color: var(--el-text-color-regular);
  font-variant-numeric: tabular-nums;
}

.quota-expand {
  padding: 6px 8px 2px;
}

.quota-expand-table {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  overflow: hidden;
  background: var(--el-bg-color-overlay);
}

.quota-expand-head,
.quota-expand-row {
  grid-template-columns: minmax(180px, 1.2fr) minmax(120px, 0.8fr) minmax(120px, 0.8fr) minmax(220px, 1.4fr);
  align-items: center;
  padding: 10px 14px;
}

.quota-expand-head {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.quota-expand-row {
  border-top: 1px solid var(--el-border-color-lighter);
}

.quota-progress-cell {
  min-width: 0;
}

.quota-progress-meta {
  grid-template-columns: auto;
  justify-content: end;
  margin-bottom: 4px;
}

.quota-empty {
  padding: 10px 14px;
  border: 1px dashed var(--el-border-color);
  border-radius: 10px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
}

@media (max-width: 960px) {
  .quota-expand-head,
  .quota-expand-row {
    grid-template-columns: minmax(120px, 1fr) minmax(90px, 0.8fr) minmax(90px, 0.8fr) minmax(140px, 1.2fr);
    gap: 10px;
    padding: 10px 12px;
  }
}
</style>
