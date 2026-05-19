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
        <div class="namespace-delete-dialog__lead">
          即将删除 Namespace
          <strong>{{ pendingNamespaceName || '-' }}</strong>
        </div>
        <div class="namespace-delete-dialog__hint">
          删除前将展示命名空间内资源摘要。请确认风险后，手动输入命名空间名称才能继续。
        </div>

        <div class="namespace-delete-dialog__summary">
          <div class="namespace-delete-dialog__summary-title">资源摘要</div>
          <el-skeleton v-if="summaryLoading" :rows="5" animated />
          <template v-else>
            <div class="namespace-delete-dialog__summary-total">共 {{ summaryTotal }} 个资源对象</div>
            <div v-if="nonZeroSummaryItems.length > 0" class="namespace-delete-dialog__summary-list">
              <div v-for="item in nonZeroSummaryItems" :key="item.key" class="namespace-delete-dialog__summary-item">
                <span>{{ item.label }}</span>
                <strong>{{ item.count }}</strong>
              </div>
            </div>
            <el-empty v-else description="当前命名空间下未发现常见资源对象" :image-size="72" />
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
  </div>
</template>

<script setup lang="ts">
import { Delete, Document } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

const columns: EnhancedColumn[] = [
  { key: 'name', label: 'Namespace', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'actions', label: '操作', width: 96, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

const props = defineProps<{
  clusterId: number
  data: any[]
  persistKey: string
  showTools: boolean
  openYaml: (row: any) => void
  onDeleted: () => Promise<void>
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
const summaryItems = ref<Array<{ key: string; label: string; count: number }>>([])

const summaryTotal = computed(() => summaryItems.value.reduce((sum, item) => sum + Number(item.count ?? 0), 0))
const nonZeroSummaryItems = computed(() => summaryItems.value.filter((item) => Number(item.count ?? 0) > 0))
const deleteDisabled = computed(() => deleting.value || confirmName.value.trim() !== pendingNamespaceName.value)

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

.namespace-delete-dialog__lead {
  font-size: 15px;
  color: #0f172a;
}

.namespace-delete-dialog__lead strong {
  margin-left: 6px;
}

.namespace-delete-dialog__hint {
  color: #475569;
  font-size: 13px;
}

.namespace-delete-dialog__summary {
  padding: 14px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.96), rgba(241, 245, 249, 0.96));
}

.namespace-delete-dialog__summary-title {
  font-weight: 700;
  color: #0f172a;
}

.namespace-delete-dialog__summary-total {
  margin-top: 8px;
  color: #334155;
  font-size: 13px;
}

.namespace-delete-dialog__summary-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 12px;
}

.namespace-delete-dialog__summary-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(148, 163, 184, 0.16);
}

.namespace-delete-dialog__form {
  margin-top: 2px;
}

:global(html.dark) .namespace-delete-dialog__lead {
  color: #e2e8f0;
}

:global(html.dark) .namespace-delete-dialog__hint,
:global(html.dark) .namespace-delete-dialog__summary-total {
  color: #94a3b8;
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

@media (max-width: 900px) {
  .namespace-delete-dialog__summary-list {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
