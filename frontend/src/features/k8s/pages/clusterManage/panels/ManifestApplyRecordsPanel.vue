<template>
  <div class="manifest-records-view">
    <div class="filter-bar">
      <div class="filter-bar__fields">
        <el-input
          v-model="keyword"
          placeholder="搜索入口、结果摘要、执行人"
          clearable
          class="manifest-records__field manifest-records__field--keyword"
          @keyup.enter="fetchData"
        />
        <el-select v-model="statusFilter" placeholder="状态" clearable class="manifest-records__field" @change="fetchData">
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="执行中" value="running" />
        </el-select>
        <el-select v-model="modeFilter" placeholder="执行方式" clearable class="manifest-records__field" @change="fetchData">
          <el-option label="Apply" value="apply" />
          <el-option label="DryRun" value="dry_run" />
        </el-select>
        <el-input
          v-model="defaultNamespace"
          placeholder="默认命名空间"
          clearable
          class="manifest-records__field"
          @keyup.enter="fetchData"
        />
      </div>

      <div class="filter-bar__actions">
        <el-button :icon="Search" @click="fetchData">查询</el-button>
        <el-button :icon="RefreshRight" @click="resetFilters">重置</el-button>
        <el-button type="primary" :icon="Upload" @click="openDeploy">通过 YAML 部署</el-button>
      </div>
    </div>

    <EnhancedTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="tableData"
      :columns="columns"
      :loading="loading"
      :total="total"
      :persist-key="persistKey"
      :show-tools="showTools"
      row-key="id"
      size="small"
      stripe
      border
      pagination
      pagination-layout="total, sizes, prev, pager, next"
      @refresh="fetchData"
    >
      <template #cell-mode="{ row }">
        <el-tag size="small" :type="row.dry_run ? 'warning' : 'success'">{{ row.dry_run ? 'DryRun' : 'Apply' }}</el-tag>
      </template>

      <template #cell-status="{ row }">
        <el-tag size="small" :type="statusTagType(row.status)">{{ statusText(row.status) }}</el-tag>
      </template>

      <template #cell-source="{ row }">
        <div class="manifest-records__source-line" :title="`${row.source_label || '通用 YAML 示例'} · ${formatSourceHint(row)}`">
          <span class="manifest-records__source-title">{{ row.source_label || '通用 YAML 示例' }}</span>
          <span class="manifest-records__source-chip">{{ formatSourceHint(row) }}</span>
        </div>
      </template>

      <template #cell-default_namespace="{ row }">
        <span>{{ row.default_namespace || '按清单原值' }}</span>
      </template>

      <template #cell-summary="{ row }">
        <div class="manifest-records__summary-line" :title="resultSummary(row)">
          <span class="manifest-records__summary" :class="row.status === 'failed' ? 'is-error' : ''">{{ resultSummary(row) }}</span>
          <span class="manifest-records__summary-chip">资源 {{ row.result_count || 0 }}</span>
        </div>
      </template>

      <template #cell-created_by_name="{ row }">
        <span>{{ row.created_by_name || '-' }}</span>
      </template>

      <template #cell-created_at="{ row }">
        <span>{{ formatDateTime(row.created_at) }}</span>
      </template>

      <template #cell-actions="{ row }">
        <div class="k8s-act-group manifest-records__actions">
          <ActionIconButton :icon="View" tooltip="查看详情" @click="openDetail(row)" />
          <ActionIconButton :icon="Upload" tooltip="再次部署" variant="success" :loading="reapplyLoadingId === row.id" @click="openDeployFromRow(row)" />
        </div>
      </template>
    </EnhancedTable>

    <el-drawer
      v-model="detailVisible"
      class="manifest-records__drawer"
      size="74%"
      destroy-on-close
      :close-on-click-modal="false"
    >
      <template #header>
        <div class="manifest-records__detail-header">
          <div>
            <div class="manifest-records__detail-title">部署记录 #{{ detail?.id || '-' }}</div>
            <div class="manifest-records__detail-sub">{{ detail?.source_label || '通用 YAML 示例' }}</div>
          </div>
          <div class="manifest-records__detail-actions">
            <el-tag size="small" :type="detail?.dry_run ? 'warning' : 'success'">{{ detail?.dry_run ? 'DryRun' : 'Apply' }}</el-tag>
            <el-tag size="small" :type="statusTagType(detail?.status || '')">{{ statusText(detail?.status || '') }}</el-tag>
            <el-button type="primary" :icon="Upload" :loading="reapplyLoadingId === detail?.id" @click="openDeployFromDetail">再次部署</el-button>
          </div>
        </div>
      </template>

      <div class="manifest-records__detail-body" v-loading="detailLoading">
        <el-descriptions v-if="detail" :column="3" border class="manifest-records__detail-meta">
          <el-descriptions-item label="执行人">{{ detail.created_by_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="执行时间">{{ formatDateTime(detail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="默认命名空间">{{ detail.default_namespace || '按清单原值' }}</el-descriptions-item>
          <el-descriptions-item label="来源资源">{{ detail.source_resource || '-' }}</el-descriptions-item>
          <el-descriptions-item label="模板 Kind">{{ detail.workload_kind || '-' }}</el-descriptions-item>
          <el-descriptions-item label="资源数量">{{ detail.result_count || 0 }}</el-descriptions-item>
        </el-descriptions>

        <el-alert
          v-if="detail"
          :title="resultSummary(detail)"
          :type="detail.status === 'failed' ? 'error' : detail.dry_run ? 'warning' : 'success'"
          :closable="false"
          show-icon
          class="manifest-records__detail-alert"
        />

        <el-tabs v-model="detailTab">
          <el-tab-pane label="执行结果" name="result">
            <el-empty v-if="!detail?.result_items?.length" :description="detail?.status === 'failed' ? '本次执行在生成资源明细前失败。' : '本次记录暂无资源结果明细。'" />
            <div v-else class="table-wrap manifest-records__detail-table">
              <el-table :data="detail?.result_items || []" class="page-table" stripe border>
                <el-table-column prop="kind" label="Kind" width="120" />
                <el-table-column prop="namespace" label="命名空间" width="140">
                  <template #default="{ row }">
                    <span>{{ row.namespace || '-' }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
                <el-table-column prop="operation" label="动作" width="100" align="center">
                  <template #default="{ row }">
                    <el-tag size="small" :type="row.operation === 'create' ? 'success' : 'warning'">{{ row.operation === 'create' ? '创建' : '更新' }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="api_version" label="API Version" width="170" show-overflow-tooltip />
                <el-table-column prop="resource" label="资源" width="150" show-overflow-tooltip />
                <el-table-column prop="scope" label="作用域" width="110" align="center" />
              </el-table>
            </div>
          </el-tab-pane>

          <el-tab-pane label="YAML 回看" name="yaml">
            <K8sYamlPanel
              :text="detail?.yaml_content || ''"
              :meta="detailYamlMeta"
              :loading="detailLoading"
              :saving="false"
              :read-only="true"
              :refreshable="false"
              :saveable="false"
              height="calc(100vh - 328px)"
            />
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RefreshRight, Search, Upload, View } from '@element-plus/icons-vue'

import K8sYamlPanel from '@/features/k8s/components/K8sYamlPanel.vue'
import {
  getManifestApplyRecord,
  getManifestApplyRecords,
  type ManifestApplyRecord,
  type ManifestApplyRecordDetail
} from '@/features/k8s/api/manifest'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import ActionIconButton from '@/shared/components/ActionIconButton.vue'
import { notifyError } from '@/shared/utils/notify'

type ManifestApplyOpenOptions = {
  defaultNamespace?: string
  initialYaml?: string
  sourceLabel?: string
  sourceResource?: string
  workloadKind?: string
}

const props = defineProps<{
  clusterId: number
  showTools?: boolean
}>()

const emit = defineEmits<{
  (e: 'open-deploy', payload: ManifestApplyOpenOptions): void
}>()

const loading = ref(false)
const tableData = ref<ManifestApplyRecord[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const statusFilter = ref('')
const modeFilter = ref('')
const defaultNamespace = ref('')

const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<ManifestApplyRecordDetail | null>(null)
const detailTab = ref<'result' | 'yaml'>('result')
const reapplyLoadingId = ref<number>(0)

const persistKey = computed(() => `k8s:cluster_manage:v2:${props.clusterId}:manifest-apply-records`)

const columns = computed<EnhancedColumn[]>(() => [
  { key: 'source', label: '部署入口', minWidth: 260, overflowTooltip: false },
  { key: 'status', label: '状态', width: 90, align: 'center' },
  { key: 'mode', label: '执行方式', width: 96, align: 'center' },
  { key: 'summary', label: '执行结果', minWidth: 360, overflowTooltip: false },
  { key: 'created_at', label: '执行时间', width: 168 },
  { key: 'id', label: '记录ID', prop: 'id', width: 90, align: 'center', defaultVisible: false },
  { key: 'default_namespace', label: '默认命名空间', prop: 'default_namespace', minWidth: 150, defaultVisible: false },
  { key: 'created_by_name', label: '执行人', prop: 'created_by_name', width: 120, defaultVisible: false },
  { key: 'actions', label: '操作', width: 106, fixed: 'right', disableToggle: true, overflowTooltip: false }
])

const detailYamlMeta = computed(() => {
  if (!detail.value) return ''
  const meta = [detail.value.source_label || '通用 YAML 示例', `cluster=${props.clusterId}`]
  if (detail.value.default_namespace) meta.push(`defaultNamespace=${detail.value.default_namespace}`)
  const source = [detail.value.source_resource, detail.value.workload_kind].filter(Boolean).join('/')
  if (source) meta.push(`template=${source}`)
  return meta.join('  ')
})

function formatDateTime(value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

function statusText(status: string) {
  switch (status) {
    case 'success':
      return '成功'
    case 'failed':
      return '失败'
    case 'running':
      return '执行中'
    default:
      return status || '-'
  }
}

function statusTagType(status: string) {
  switch (status) {
    case 'success':
      return 'success'
    case 'failed':
      return 'danger'
    default:
      return 'info'
  }
}

function formatSourceHint(row: ManifestApplyRecord) {
  const hint = [row.source_resource, row.workload_kind].filter(Boolean).join(' / ')
  return hint || '手动输入或上传 YAML'
}

function resultSummary(row: ManifestApplyRecord) {
  return row.summary || row.error_message || '暂无执行摘要'
}

async function fetchData() {
  if (!props.clusterId) {
    tableData.value = []
    total.value = 0
    return
  }
  loading.value = true
  try {
    const result = await getManifestApplyRecords(props.clusterId, {
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      status: statusFilter.value || undefined,
      mode: modeFilter.value || undefined,
      default_namespace: defaultNamespace.value.trim() || undefined
    })
    tableData.value = result.list ?? []
    total.value = result.total ?? 0
  } catch (error) {
    tableData.value = []
    total.value = 0
    notifyError(error instanceof Error && error.message ? error.message : 'YAML 部署记录加载失败')
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  keyword.value = ''
  statusFilter.value = ''
  modeFilter.value = ''
  defaultNamespace.value = ''
  page.value = 1
  void fetchData()
}

function openDeploy() {
  emit('open-deploy', {})
}

async function loadDetail(recordId: number) {
  detailLoading.value = true
  try {
    const result = await getManifestApplyRecord(props.clusterId, recordId)
    detail.value = result
    return result
  } catch (error) {
    notifyError(error instanceof Error && error.message ? error.message : '部署记录详情加载失败')
    return null
  } finally {
    detailLoading.value = false
  }
}

async function openDetail(row: ManifestApplyRecord) {
  detailVisible.value = true
  detailTab.value = 'result'
  detail.value = null
  const result = await loadDetail(row.id)
  if (!result) detailVisible.value = false
}

function buildDeployPayload(record: ManifestApplyRecordDetail): ManifestApplyOpenOptions {
  return {
    defaultNamespace: record.default_namespace || undefined,
    initialYaml: record.yaml_content,
    sourceLabel: `复用记录 #${record.id}`,
    sourceResource: record.source_resource || undefined,
    workloadKind: record.workload_kind || undefined
  }
}

async function openDeployFromRow(row: ManifestApplyRecord) {
  reapplyLoadingId.value = row.id
  try {
    const record = detail.value?.id === row.id ? detail.value : await getManifestApplyRecord(props.clusterId, row.id)
    if (!record) return
    emit('open-deploy', buildDeployPayload(record))
  } catch (error) {
    notifyError(error instanceof Error && error.message ? error.message : '复用部署记录失败')
  } finally {
    reapplyLoadingId.value = 0
  }
}

function openDeployFromDetail() {
  if (!detail.value) return
  reapplyLoadingId.value = detail.value.id
  emit('open-deploy', buildDeployPayload(detail.value))
  reapplyLoadingId.value = 0
}

watch(() => props.clusterId, () => {
  page.value = 1
  detailVisible.value = false
  detail.value = null
  void fetchData()
}, { immediate: true })

defineExpose({ reload: fetchData })
</script>

<style scoped>
.manifest-records-view {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 100%;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: nowrap;
}

.filter-bar__fields {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1 1 auto;
  min-width: 0;
  flex-wrap: nowrap;
}

.filter-bar__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  flex-wrap: nowrap;
}

.manifest-records__field {
  width: auto;
  min-width: 120px;
  flex: 0 1 132px;
}

.manifest-records__field--keyword {
  width: auto;
  min-width: 220px;
  flex: 1 1 240px;
}

.table-wrap {
  width: 100%;
}

.page-table {
  width: 100%;
}

.manifest-records__source-line,
.manifest-records__summary-line {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.manifest-records__source-title {
  min-width: 0;
  font-weight: 600;
  color: var(--app-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.manifest-records__source-chip,
.manifest-records__summary-chip {
  flex: 0 0 auto;
  padding: 3px 8px;
  border-radius: 999px;
  background: rgba(241, 245, 249, 0.95);
  color: var(--app-muted);
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
}

.manifest-records__summary {
  min-width: 0;
  flex: 1 1 auto;
  color: var(--app-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.manifest-records__summary.is-error {
  color: var(--el-color-danger);
}

.manifest-records__actions {
  justify-content: center;
}

.manifest-records__detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
}

.manifest-records__detail-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--app-text);
}

.manifest-records__detail-sub {
  margin-top: 4px;
  font-size: 13px;
  color: var(--app-muted);
}

.manifest-records__detail-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.manifest-records__detail-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: calc(100vh - 180px);
}

.manifest-records__detail-alert {
  margin-top: 4px;
}

.manifest-records__detail-table {
  padding-bottom: 4px;
}

@media (max-width: 900px) {
  .filter-bar {
    align-items: stretch;
    flex-wrap: wrap;
  }

  .manifest-records__field,
  .manifest-records__field--keyword {
    width: 100%;
    min-width: 0;
    flex: 1 1 100%;
  }

  .filter-bar__fields,
  .filter-bar__actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .manifest-records__detail-header {
    flex-direction: column;
  }

  .manifest-records__source-line,
  .manifest-records__summary-line {
    gap: 8px;
  }

  .manifest-records__source-chip,
  .manifest-records__summary-chip {
    max-width: 42%;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

:global(html.dark) .manifest-records__source-chip,
:global(html.dark) .manifest-records__summary-chip {
  background: rgba(30, 41, 59, 0.88);
}
</style>