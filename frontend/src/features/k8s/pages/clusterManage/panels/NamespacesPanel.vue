<template>
  <div class="namespaces-panel">
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
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="openDeleteDialog(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
    </EnhancedTable>

    <el-dialog v-model="deleteVisible" title="删除 Namespace" width="640px" destroy-on-close @closed="resetDeleteDialog">
      <div class="namespace-delete-dialog">
        <div class="namespace-delete-dialog__lead-row">
          <div class="namespace-delete-dialog__lead">
            即将删除 Namespace
            <strong>{{ pendingNamespaceName || '-' }}</strong>
          </div>
          <el-tooltip placement="top-start" :show-after="180" popper-class="namespace-delete-dialog__tooltip">
            <template #content>
              <div class="namespace-delete-dialog__tooltip-content">
                删除前将自动统计当前命名空间内需关注的资源摘要，已忽略默认生成的 ServiceAccount、Token Secret 和根证书 ConfigMap。请确认风险后，手动输入命名空间名称才能继续。
              </div>
            </template>
            <button type="button" class="namespace-delete-dialog__hint-trigger" aria-label="删除提示">?</button>
          </el-tooltip>
        </div>

        <div class="namespace-delete-dialog__summary">
          <div class="namespace-delete-dialog__summary-title">资源摘要</div>
          <el-skeleton v-if="summaryLoading" :rows="5" animated />
          <template v-else>
            <div class="namespace-delete-dialog__summary-total">共 {{ summaryTotal }} 个资源对象</div>
            <div v-if="nonZeroSummaryItems.length > 0" class="namespace-delete-dialog__summary-list">
              <el-tooltip
                v-for="item in nonZeroSummaryItems"
                :key="item.key"
                :disabled="canPreviewSummaryItem(item)"
                content="当前暂不支持打开该资源列表"
                placement="top"
                :show-after="180"
              >
                <span class="namespace-delete-dialog__summary-item-wrap">
                  <button
                    type="button"
                    class="namespace-delete-dialog__summary-item"
                    :class="canPreviewSummaryItem(item) ? 'is-clickable' : 'is-disabled'"
                    :disabled="!canPreviewSummaryItem(item)"
                    @click="openSummaryPreview(item)"
                  >
                    <span class="namespace-delete-dialog__summary-label" :title="item.label">{{ item.label }}</span>
                    <strong class="namespace-delete-dialog__summary-count">{{ item.count }}</strong>
                  </button>
                </span>
              </el-tooltip>
            </div>
            <el-empty v-else description="当前命名空间下未发现需关注的资源对象" :image-size="72" />
          </template>
        </div>

        <el-alert type="error" :closable="false" show-icon title="删除 Namespace 会级联移除其下所有资源，请谨慎操作。" />

        <el-form label-width="136px" class="namespace-delete-dialog__form">
          <el-form-item label="输入 Namespace 名称">
            <el-input v-model="confirmName" :placeholder="pendingNamespaceName || '请输入名称确认'" clearable />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-space>
          <el-button @click="deleteVisible = false">取消</el-button>
          <el-button type="danger" :loading="deleting" :disabled="deleteDisabled" @click="confirmDeleteNamespace">确认删除</el-button>
        </el-space>
      </template>
    </el-dialog>

    <el-dialog
      v-model="summaryPreviewVisible"
      :title="summaryPreviewTitle"
      width="860px"
      append-to-body
      destroy-on-close
      @closed="resetSummaryPreviewDialog"
    >
      <div class="namespace-summary-preview">
        <div class="namespace-summary-preview__meta">
          <div class="namespace-summary-preview__meta-item">
            <span class="namespace-summary-preview__meta-label">Namespace</span>
            <strong class="namespace-summary-preview__meta-value">{{ pendingNamespaceName || '-' }}</strong>
          </div>
          <div class="namespace-summary-preview__meta-item">
            <span class="namespace-summary-preview__meta-label">资源数</span>
            <strong class="namespace-summary-preview__meta-value">{{ summaryPreviewRows.length }}</strong>
          </div>
        </div>

        <EnhancedTable
          :data="summaryPreviewRows"
          :columns="summaryPreviewColumns"
          :row-key="getSummaryPreviewRowKey"
          :loading="summaryPreviewLoading"
          :persist-key="summaryPreviewPersistKey"
          :show-tools="false"
          :show-topbar="false"
          :height="420"
          size="small"
          stripe
          border
        >
          <template #cell-name="{ row }">
            <span class="namespace-summary-preview__name" :title="row.name">{{ row.name }}</span>
          </template>
          <template #cell-summary="{ row }">
            <span class="namespace-summary-preview__summary" :title="row.summary">{{ row.summary }}</span>
          </template>
          <template #cell-age="{ row }">
            <span class="namespace-summary-preview__age">{{ row.age }}</span>
          </template>
        </EnhancedTable>
      </div>

      <template #footer>
        <el-space>
          <el-button @click="summaryPreviewVisible = false">关闭</el-button>
          <el-button type="primary" :disabled="!activeSummaryPreviewConfig" @click="openSummaryResourcePage">
            更多详情
          </el-button>
        </el-space>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { Delete, Document } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { formatAgeMs, getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

type NamespaceSummaryItem = {
  key: string
  label: string
  count: number
}

type SummaryPreviewRow = {
  key: string
  name: string
  summary: string
  age: string
}

type SummaryPreviewConfig = {
  nodeId: string
  load: (clusterId: number, namespace: string) => Promise<{ list: any[] }>
}

const columns: EnhancedColumn[] = [
  { key: 'name', label: 'Namespace', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 96, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

const summaryPreviewColumns: EnhancedColumn[] = [
  { key: 'name', label: '名称', minWidth: 260, defaultVisible: true },
  { key: 'summary', label: '摘要', minWidth: 260, defaultVisible: true },
  { key: 'age', label: 'AGE', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true }
]

function getObjectKeyCount(...records: any[]): number {
  return records.reduce((sum, record) => {
    if (!record || typeof record !== 'object' || Array.isArray(record)) return sum
    return sum + Object.keys(record as Record<string, unknown>).length
  }, 0)
}

function getContainerReadyText(row: any): string {
  const statuses: any[] = Array.isArray(row?.status?.containerStatuses) ? row.status.containerStatuses : []
  const ready = statuses.filter((item) => item?.ready === true).length
  return `${ready}/${statuses.length}`
}

function getWorkloadReadySummary(row: any): string {
  const desired = Number(row?.spec?.replicas ?? 0)
  const ready = Number(row?.status?.readyReplicas ?? 0)
  return `Ready ${ready}/${desired}`
}

function countEndpointAddresses(row: any): number {
  const subsets: any[] = Array.isArray(row?.subsets) ? row.subsets : []
  return subsets.reduce((sum, subset) => {
    const ready = Array.isArray(subset?.addresses) ? subset.addresses.length : 0
    const notReady = Array.isArray(subset?.notReadyAddresses) ? subset.notReadyAddresses.length : 0
    return sum + ready + notReady
  }, 0)
}

function getSummaryPreviewRowKey(row: SummaryPreviewRow): string {
  return row.key
}

function getSummaryPreviewConfig(kind: string): SummaryPreviewConfig | null {
  switch (String(kind ?? '').trim()) {
    case 'ConfigMap':
      return { nodeId: 'config:configmaps', load: (clusterId, namespace) => k8sApi.listConfigMaps(clusterId, { namespace }) }
    case 'Secret':
      return { nodeId: 'config:secrets', load: (clusterId, namespace) => k8sApi.listSecrets(clusterId, { namespace }) }
    case 'ServiceAccount':
      return { nodeId: 'config:serviceaccounts', load: (clusterId, namespace) => k8sApi.listServiceAccounts(clusterId, { namespace }) }
    case 'Deployment':
      return { nodeId: 'workloads:deployments', load: (clusterId, namespace) => k8sApi.listWorkloads(clusterId, { kind: 'Deployment', namespace }) }
    case 'StatefulSet':
      return { nodeId: 'workloads:statefulsets', load: (clusterId, namespace) => k8sApi.listWorkloads(clusterId, { kind: 'StatefulSet', namespace }) }
    case 'DaemonSet':
      return { nodeId: 'workloads:daemonsets', load: (clusterId, namespace) => k8sApi.listWorkloads(clusterId, { kind: 'DaemonSet', namespace }) }
    case 'Pod':
      return { nodeId: 'workloads:pods', load: (clusterId, namespace) => k8sApi.listPods(clusterId, { namespace }) }
    case 'PodMetrics':
      return { nodeId: 'workloads:podmetrics', load: (clusterId, namespace) => k8sApi.listPodMetrics(clusterId, { namespace }) }
    case 'ReplicaSet':
      return { nodeId: 'workloads:replicasets', load: (clusterId, namespace) => k8sApi.listReplicaSets(clusterId, { namespace }) }
    case 'PodDisruptionBudget':
      return { nodeId: 'workloads:pdbs', load: (clusterId, namespace) => k8sApi.listPDBs(clusterId, { namespace }) }
    case 'HorizontalPodAutoscaler':
      return { nodeId: 'workloads:hpas', load: (clusterId, namespace) => k8sApi.listHPAs(clusterId, { namespace }) }
    case 'Job':
      return { nodeId: 'jobs:jobs', load: (clusterId, namespace) => k8sApi.listJobs(clusterId, { namespace }) }
    case 'CronJob':
      return { nodeId: 'jobs:cronjobs', load: (clusterId, namespace) => k8sApi.listCronJobs(clusterId, { namespace }) }
    case 'Service':
      return { nodeId: 'network:services', load: (clusterId, namespace) => k8sApi.listServices(clusterId, { namespace }) }
    case 'Endpoints':
      return { nodeId: 'network:endpoints', load: (clusterId, namespace) => k8sApi.listEndpoints(clusterId, { namespace }) }
    case 'EndpointSlice':
      return { nodeId: 'network:endpointslices', load: (clusterId, namespace) => k8sApi.listEndpointSlices(clusterId, { namespace }) }
    case 'Ingress':
      return { nodeId: 'network:ingresses', load: (clusterId, namespace) => k8sApi.listIngresses(clusterId, { namespace }) }
    case 'NetworkPolicy':
      return { nodeId: 'network:networkpolicies', load: (clusterId, namespace) => k8sApi.listNetworkPolicies(clusterId, { namespace }) }
    case 'Role':
      return { nodeId: 'auth:roles', load: (clusterId, namespace) => k8sApi.listRoles(clusterId, { namespace }) }
    case 'RoleBinding':
      return { nodeId: 'auth:rolebindings', load: (clusterId, namespace) => k8sApi.listRoleBindings(clusterId, { namespace }) }
    case 'PersistentVolumeClaim':
      return { nodeId: 'storage:pvcs', load: (clusterId, namespace) => k8sApi.listPVCs(clusterId, { namespace }) }
    case 'ResourceQuota':
      return { nodeId: 'storage:resourcequotas', load: (clusterId, namespace) => k8sApi.listResourceQuotas(clusterId, { namespace }) }
    case 'LimitRange':
      return { nodeId: 'storage:limitranges', load: (clusterId, namespace) => k8sApi.listLimitRanges(clusterId, { namespace }) }
    case 'CSIStorageCapacity':
      return { nodeId: 'storage:csistoragecapacities', load: (clusterId, namespace) => k8sApi.listCSIStorageCapacities(clusterId, { namespace }) }
    case 'VolumeSnapshot':
      return { nodeId: 'storage:volumesnapshots', load: (clusterId, namespace) => k8sApi.listVolumeSnapshots(clusterId, { namespace }) }
    case 'Lease':
      return { nodeId: 'cluster:leases', load: (clusterId, namespace) => k8sApi.listLeases(clusterId, { namespace }) }
    case 'Event':
      return { nodeId: 'misc:events', load: (clusterId, namespace) => k8sApi.listEvents(clusterId, { namespace }) }
    default:
      return null
  }
}

function getSummaryPreviewSummary(kind: string, row: any): string {
  switch (kind) {
    case 'ConfigMap':
      return `Keys ${getObjectKeyCount(row?.data, row?.binaryData)}`
    case 'Secret':
      return `Type ${String(row?.type ?? '-')}`
    case 'ServiceAccount':
      return `Secrets ${Array.isArray(row?.secrets) ? row.secrets.length : 0}`
    case 'Deployment':
    case 'StatefulSet':
    case 'ReplicaSet':
      return getWorkloadReadySummary(row)
    case 'DaemonSet':
      return `Ready ${Number(row?.status?.numberReady ?? 0)}/${Number(row?.status?.desiredNumberScheduled ?? 0)}`
    case 'Pod':
      return `Phase ${String(row?.status?.phase ?? '-')} / Ready ${getContainerReadyText(row)}`
    case 'PodMetrics':
      return `容器 ${Array.isArray(row?.containers) ? row.containers.length : 0} / 统计周期 ${String(row?.window ?? '-')}`
    case 'PodDisruptionBudget':
      return `Healthy ${Number(row?.status?.currentHealthy ?? 0)}`
    case 'HorizontalPodAutoscaler':
      return `Replicas ${Number(row?.status?.currentReplicas ?? 0)}/${Number(row?.status?.desiredReplicas ?? 0)}`
    case 'Job': {
      const succeeded = Number(row?.status?.succeeded ?? 0)
      const completions = Number(row?.spec?.completions ?? 1)
      return `Completed ${succeeded}/${completions}`
    }
    case 'CronJob':
      return `Schedule ${String(row?.spec?.schedule ?? '-')}`
    case 'Service':
      return `Type ${String(row?.spec?.type ?? 'ClusterIP')} / Ports ${Array.isArray(row?.spec?.ports) ? row.spec.ports.length : 0}`
    case 'Endpoints':
      return `Addresses ${countEndpointAddresses(row)}`
    case 'EndpointSlice':
      return `Endpoints ${Array.isArray(row?.endpoints) ? row.endpoints.length : 0}`
    case 'Ingress':
      return `Rules ${Array.isArray(row?.spec?.rules) ? row.spec.rules.length : 0}`
    case 'NetworkPolicy':
      return `Ingress ${Array.isArray(row?.spec?.ingress) ? row.spec.ingress.length : 0} / Egress ${Array.isArray(row?.spec?.egress) ? row.spec.egress.length : 0}`
    case 'Role':
      return `Rules ${Array.isArray(row?.rules) ? row.rules.length : 0}`
    case 'RoleBinding':
      return `Subjects ${Array.isArray(row?.subjects) ? row.subjects.length : 0}`
    case 'PersistentVolumeClaim':
      return `Phase ${String(row?.status?.phase ?? '-')} / ${String(row?.spec?.resources?.requests?.storage ?? '-')}`
    case 'ResourceQuota':
      return `Hard ${getObjectKeyCount(row?.spec?.hard)}`
    case 'LimitRange':
      return `Limits ${Array.isArray(row?.spec?.limits) ? row.spec.limits.length : 0}`
    case 'CSIStorageCapacity':
      return `Class ${String(row?.storageClassName ?? row?.spec?.storageClassName ?? '-')}`
    case 'VolumeSnapshot':
      return `Ready ${row?.status?.readyToUse ? 'Yes' : 'No'}`
    case 'Lease':
      return `Holder ${String(row?.spec?.holderIdentity ?? '-')}`
    case 'Event':
      return `${String(row?.type ?? '-')} / ${String(row?.reason ?? '-')}`
    default:
      return '-'
  }
}

function buildSummaryPreviewRows(kind: string, list: any[]): SummaryPreviewRow[] {
  return list.map((row, index) => {
    const name = String(row?.metadata?.name ?? '-').trim() || '-'
    const namespaceText = String(row?.metadata?.namespace ?? '').trim()
    const baseKey = namespaceText ? `${namespaceText}/${name}` : name
    return {
      key: `${kind}:${baseKey}:${index}`,
      name,
      summary: getSummaryPreviewSummary(kind, row),
      age: getSummaryPreviewAge(kind, row)
    }
  })
}

function getSummaryPreviewAge(kind: string, row: any): string {
  if (kind !== 'PodMetrics') return getCreationAgeText(row)
  const ts = new Date(String(row?.timestamp ?? '')).getTime()
  if (!Number.isFinite(ts)) return '-'
  return formatAgeMs(Math.max(0, Date.now() - ts))
}

const props = defineProps<{
  clusterId: number
  data: any[]
  persistKey: string
  showTools: boolean
  openYaml: (row: any) => void
  onDeleted: () => Promise<void>
  openSummaryResourcePage: (payload: { treeNodeId: string; namespace: string; kind: string }) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
const deleteVisible = ref(false)
const summaryLoading = ref(false)
const deleting = ref(false)
const pendingNamespaceName = ref('')
const confirmName = ref('')
const summaryItems = ref<NamespaceSummaryItem[]>([])
const summaryPreviewVisible = ref(false)
const summaryPreviewLoading = ref(false)
const summaryPreviewItem = ref<NamespaceSummaryItem | null>(null)
const summaryPreviewRows = ref<SummaryPreviewRow[]>([])

let summaryPreviewSeq = 0

const summaryTotal = computed(() => summaryItems.value.reduce((sum, item) => sum + Number(item.count ?? 0), 0))
const nonZeroSummaryItems = computed(() => summaryItems.value.filter((item) => Number(item.count ?? 0) > 0))
const deleteDisabled = computed(() => deleting.value || confirmName.value.trim() !== pendingNamespaceName.value)
const activeSummaryPreviewConfig = computed(() => getSummaryPreviewConfig(summaryPreviewItem.value?.label ?? ''))
const summaryPreviewTitle = computed(() => `${summaryPreviewItem.value?.label ?? '资源'} 列表`)
const summaryPreviewPersistKey = computed(() => {
  if (!summaryPreviewItem.value?.label) return undefined
  const key = summaryPreviewItem.value.label.toLowerCase().replace(/[^a-z0-9]+/g, '-')
  return `k8s:cluster_manage:v2:${props.clusterId}:namespace-summary:${key}`
})

function canPreviewSummaryItem(item: NamespaceSummaryItem): boolean {
  return Boolean(getSummaryPreviewConfig(item.label))
}

async function openDeleteDialog(row: any) {
  const name = String(row?.metadata?.name ?? '').trim()
  if (!props.clusterId || !name) return
  pendingNamespaceName.value = name
  confirmName.value = ''
  summaryItems.value = []
  deleteVisible.value = true
  summaryLoading.value = true
  try {
    const summary = await k8sApi.getNamespaceResourcesSummary(props.clusterId, name)
    summaryItems.value = Array.isArray(summary.items) ? summary.items : []
  } catch (error) {
    deleteVisible.value = false
    handleError(error)
  } finally {
    summaryLoading.value = false
  }
}

async function openSummaryPreview(item: NamespaceSummaryItem) {
  const namespaceText = pendingNamespaceName.value.trim()
  const config = getSummaryPreviewConfig(item.label)
  if (!props.clusterId || !namespaceText || !config) return
  summaryPreviewItem.value = item
  summaryPreviewRows.value = []
  summaryPreviewVisible.value = true
  summaryPreviewLoading.value = true
  const seq = ++summaryPreviewSeq
  try {
    const data = await config.load(props.clusterId, namespaceText)
    if (seq !== summaryPreviewSeq) return
    summaryPreviewRows.value = buildSummaryPreviewRows(item.label, Array.isArray(data.list) ? data.list : [])
  } catch (error) {
    if (seq !== summaryPreviewSeq) return
    summaryPreviewVisible.value = false
    handleError(error)
  } finally {
    if (seq === summaryPreviewSeq) summaryPreviewLoading.value = false
  }
}

function openSummaryResourcePage() {
  const namespaceText = pendingNamespaceName.value.trim()
  const item = summaryPreviewItem.value
  const config = activeSummaryPreviewConfig.value
  if (!item || !config || !namespaceText) return
  summaryPreviewVisible.value = false
  deleteVisible.value = false
  props.openSummaryResourcePage({ treeNodeId: config.nodeId, namespace: namespaceText, kind: item.label })
}

async function confirmDeleteNamespace() {
  const namespace = pendingNamespaceName.value.trim()
  if (!props.clusterId || !namespace || deleteDisabled.value) return
  deleting.value = true
  try {
    await k8sApi.deleteNamespace(props.clusterId, namespace)
    notifySuccess('Namespace 已删除')
    deleteVisible.value = false
    await props.onDeleted()
  } catch (error) {
    handleError(error)
  } finally {
    deleting.value = false
  }
}

function resetDeleteDialog() {
  pendingNamespaceName.value = ''
  confirmName.value = ''
  summaryItems.value = []
  summaryLoading.value = false
  deleting.value = false
  resetSummaryPreviewDialog()
}

function resetSummaryPreviewDialog() {
  summaryPreviewItem.value = null
  summaryPreviewRows.value = []
  summaryPreviewLoading.value = false
}

function handleError(error: unknown) {
  const err = error as ApiError
  notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
}

defineExpose({ getTable: () => tableRef.value })
</script>

<style scoped>
.namespaces-panel {
  display: flex;
  flex-direction: column;
}

.namespace-delete-dialog {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.namespace-delete-dialog__lead-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.namespace-delete-dialog__lead {
  font-size: 15px;
  color: #0f172a;
}

.namespace-delete-dialog__lead strong {
  margin-left: 6px;
}

.namespace-delete-dialog__hint-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.26);
  background: rgba(248, 250, 252, 0.96);
  color: #475569;
  font-size: 14px;
  font-weight: 700;
  line-height: 1;
  cursor: help;
  transition: border-color 0.18s ease, color 0.18s ease, background 0.18s ease;
}

.namespace-delete-dialog__hint-trigger:hover {
  border-color: rgba(59, 130, 246, 0.28);
  background: rgba(239, 246, 255, 0.96);
  color: #2563eb;
}

.namespace-delete-dialog__summary {
  padding: 12px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.96), rgba(241, 245, 249, 0.96));
}

.namespace-delete-dialog__summary-title {
  font-weight: 700;
  color: #0f172a;
}

.namespace-delete-dialog__summary-total {
  margin-top: 6px;
  color: #334155;
  font-size: 12px;
}

.namespace-delete-dialog__summary-list {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-top: 10px;
}

.namespace-delete-dialog__summary-item-wrap {
  display: block;
}

.namespace-delete-dialog__summary-item {
  appearance: none;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  min-width: 0;
  padding: 8px 10px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(148, 163, 184, 0.16);
  text-align: left;
  opacity: 1;
}

.namespace-delete-dialog__summary-item.is-clickable {
  cursor: pointer;
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
}

.namespace-delete-dialog__summary-item.is-clickable:hover {
  transform: translateY(-1px);
  border-color: rgba(37, 99, 235, 0.24);
  box-shadow: 0 10px 18px rgba(15, 23, 42, 0.05);
}

.namespace-delete-dialog__summary-item.is-clickable:focus-visible {
  outline: none;
  border-color: rgba(37, 99, 235, 0.34);
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12);
}

.namespace-delete-dialog__summary-item.is-disabled {
  cursor: not-allowed;
  opacity: 0.62;
  filter: saturate(0.82);
}

.namespace-delete-dialog__summary-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #475569;
  font-size: 12px;
  font-weight: 400;
}

.namespace-delete-dialog__summary-count {
  flex: 0 0 auto;
  color: #2563eb;
  font-size: 15px;
  line-height: 1;
}

.namespace-summary-preview {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.namespace-summary-preview__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.namespace-summary-preview__meta-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(248, 250, 252, 0.88);
}

.namespace-summary-preview__meta-label {
  color: #64748b;
  font-size: 12px;
}

.namespace-summary-preview__meta-value {
  color: #0f172a;
  font-size: 13px;
}

.namespace-summary-preview__name,
.namespace-summary-preview__summary {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.namespace-summary-preview__name {
  color: #0f172a;
}

.namespace-summary-preview__summary,
.namespace-summary-preview__age {
  color: #64748b;
}

.namespace-delete-dialog__form {
  margin-top: 2px;
}

:global(html.dark) .namespace-delete-dialog__lead {
  color: #e2e8f0;
}

:global(html.dark) .namespace-delete-dialog__summary-total {
  color: #94a3b8;
}

:global(html.dark) .namespace-delete-dialog__hint-trigger {
  border-color: rgba(148, 163, 184, 0.2);
  background: rgba(15, 23, 42, 0.88);
  color: #cbd5e1;
}

:global(html.dark) .namespace-delete-dialog__hint-trigger:hover {
  border-color: rgba(96, 165, 250, 0.3);
  background: rgba(30, 41, 59, 0.92);
  color: #93c5fd;
}

:global(html.dark) .namespace-delete-dialog__summary {
  border-color: rgba(148, 163, 184, 0.16);
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.9), rgba(15, 23, 42, 0.78));
}

:global(html.dark) .namespace-delete-dialog__summary-title {
  color: #e2e8f0;
}

:global(html.dark) .namespace-delete-dialog__summary-item {
  background: rgba(30, 41, 59, 0.72);
  border-color: rgba(148, 163, 184, 0.14);
  color: #e2e8f0;
}

:global(html.dark) .namespace-delete-dialog__summary-item.is-clickable:hover {
  border-color: rgba(96, 165, 250, 0.3);
  box-shadow: 0 12px 22px rgba(2, 6, 23, 0.28);
  background: rgba(30, 41, 59, 0.86);
}

:global(html.dark) .namespace-delete-dialog__summary-label {
  color: #cbd5e1;
}

:global(html.dark) .namespace-delete-dialog__summary-count {
  color: #93c5fd;
}

:global(.namespace-delete-dialog__tooltip) {
  max-width: 360px;
}

:global(.namespace-delete-dialog__tooltip .namespace-delete-dialog__tooltip-content) {
  white-space: normal;
  line-height: 1.6;
}

:global(html.dark) .namespace-summary-preview__meta-item {
  border-color: rgba(148, 163, 184, 0.16);
  background: rgba(15, 23, 42, 0.84);
}

:global(html.dark) .namespace-summary-preview__meta-label,
:global(html.dark) .namespace-summary-preview__summary,
:global(html.dark) .namespace-summary-preview__age {
  color: #94a3b8;
}

:global(html.dark) .namespace-summary-preview__meta-value,
:global(html.dark) .namespace-summary-preview__name {
  color: #e2e8f0;
}

@media (max-width: 900px) {
  .namespace-delete-dialog__summary-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .namespace-delete-dialog__lead-row {
    align-items: flex-start;
  }
}

@media (max-width: 640px) {
  .namespace-delete-dialog__summary-list {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
