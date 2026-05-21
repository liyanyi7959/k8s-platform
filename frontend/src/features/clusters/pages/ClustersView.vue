<template>
  <section class="clusters-page">
    <section class="clusters-panel clusters-panel--filters" aria-label="集群筛选条件">
      <div class="clusters-overview" aria-label="集群概览">
        <div class="clusters-overview__item clusters-overview__item--total">
          <div class="clusters-overview__meta">
            <span class="clusters-overview__label">可见集群</span>
            <span class="clusters-overview__hint">当前筛选结果</span>
          </div>
          <strong class="clusters-overview__value">{{ total }}</strong>
        </div>
        <div class="clusters-overview__item clusters-overview__item--ok">
          <div class="clusters-overview__meta">
            <span class="clusters-overview__label">正常</span>
            <span class="clusters-overview__hint">在线且健康</span>
          </div>
          <strong class="clusters-overview__value">{{ visibleStatusSummary.active }}</strong>
        </div>
        <div class="clusters-overview__item clusters-overview__item--warn">
          <div class="clusters-overview__meta">
            <span class="clusters-overview__label">需关注</span>
            <span class="clusters-overview__hint">降级或处理中</span>
          </div>
          <strong class="clusters-overview__value">{{ visibleAttentionCount }}</strong>
        </div>
        <div class="clusters-overview__item clusters-overview__item--nodes">
          <div class="clusters-overview__meta">
            <span class="clusters-overview__label">节点总数</span>
            <span class="clusters-overview__hint">当前可见资源</span>
          </div>
          <strong class="clusters-overview__value">{{ visibleNodeCount }}</strong>
        </div>
      </div>

      <div class="clusters-filters">
        <div class="clusters-filters__fields">
          <el-input
            v-model="query.keyword"
            class="clusters-filter clusters-filter--search"
            size="default"
            placeholder="搜索集群名称"
            clearable
            @keyup.enter="onSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>

          <el-select v-model="query.status" class="clusters-filter" size="default" clearable placeholder="状态">
            <el-option label="正常" value="active" />
            <el-option label="已禁用" value="disabled" />
            <el-option label="创建中" value="creating" />
            <el-option label="降级" value="degraded" />
            <el-option label="删除中" value="deleting" />
          </el-select>
          <el-select v-model="query.type" class="clusters-filter" size="default" clearable placeholder="类型">
            <el-option label="导入" value="imported" />
            <el-option label="创建" value="created" />
          </el-select>
        </div>

        <div class="clusters-filters__actions">
          <el-button type="primary" size="default" @click="openImport">
            <el-icon><Upload /></el-icon>
            <span>导入集群</span>
          </el-button>
          <el-button type="primary" size="default" @click="onSearch">
            <el-icon><Search /></el-icon>
            <span>查询</span>
          </el-button>
          <el-button size="default" @click="onResetFilters">
              <el-icon><RefreshRight /></el-icon>
            <span>重置</span>
          </el-button>
          <el-button size="default" :disabled="selectedCount === 0" :loading="bulkHealthLoading" @click="bulkHealth">
              <el-icon><CircleCheck /></el-icon>
            <span>批量检查</span>
          </el-button>
          <el-button v-if="selectedCount > 0" size="default" @click="clearSelection">
              <el-icon><CircleClose /></el-icon>
              <span>清空选择</span>
            </el-button>
        </div>
      </div>
    </section>

    <el-card class="page-card page-card--clusters">
      <EnhancedTable
        ref="tableUiRef"
        v-model:page="page"
        v-model:page-size="pageSize"
        :total="total"
        pagination
        persist-key="clusters:clusters"
        :data="list"
        :columns="columns"
        :row-key="'id'"
        :loading="loading"
        selectable
        stripe
        border
        @refresh="load"
        @sort-change="onSortChange"
        @selection-change="onSelectionChange"
        @row-contextmenu="onRowContextMenu"
      >
        <template #topbar-left="{ selectedCount: tableSelectedCount }">
          <div class="table-summary">
            <span class="table-summary__title">集群列表</span>
            <span class="table-summary__meta">共 {{ total }} 个集群</span>
            <span v-if="tableSelectedCount > 0" class="table-summary__badge">已选 {{ tableSelectedCount }}</span>
          </div>
        </template>

        <template #cell-name="{ row }">
          <el-button
            v-if="canOpenK8s"
            link
            type="primary"
            @click="openCluster(row)"
            @contextmenu.prevent.stop="openClusterContextMenu($event, row)"
          >
            {{ row.name }}
          </el-button>
          <span v-else>{{ row.name }}</span>
        </template>

        <template #cell-k8s_version="{ row }">
          <el-tag v-if="row.k8s_version" type="info" size="small">{{ row.k8s_version }}</el-tag>
          <span v-else class="text-slate-400 text-xs">—</span>
        </template>

        <template #cell-node_count="{ row }">
          <span v-if="row.node_count != null && row.node_count > 0" class="font-semibold">{{ row.node_count }}</span>
          <span v-else class="text-slate-400 text-xs">—</span>
        </template>

        <template #cell-status="{ row }">
          <el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
        </template>

        <template #cell-actions="{ row }">
          <div class="k8s-act-group">
            <el-tooltip content="健康检查" placement="top" :show-after="400">
              <button class="k8s-act-btn k8s-act-btn--success" :disabled="healthChecking[String(row.id)]" @click="onHealth(row.id)"><el-icon><CircleCheck /></el-icon></button>
            </el-tooltip>
            <el-tooltip v-if="canOpenK8s" content="进入 K8s 管理" placement="top" :show-after="400">
              <button class="k8s-act-btn k8s-act-btn--info" @click="openCluster(row)"><K8sClusterIcon class="cluster-enter-icon" /></button>
            </el-tooltip>
            <el-tooltip v-if="canOpenK8s" :content="isClusterPinned(row.id) ? '取消左侧快捷入口' : '固定到左侧快捷入口'" placement="top" :show-after="400">
              <button :class="['k8s-act-btn', 'k8s-act-btn--warn', isClusterPinned(row.id) ? 'k8s-act-btn--pinned' : '']" @click="toggleClusterShortcut(row)">
                <el-icon><StarFilled v-if="isClusterPinned(row.id)" /><Star v-else /></el-icon>
              </button>
            </el-tooltip>
            <el-tooltip content="编辑" placement="top" :show-after="400">
              <button class="k8s-act-btn k8s-act-btn--edit" :disabled="row.status === 'creating' || row.status === 'deleting'" @click="openEdit(row)"><el-icon><EditPen /></el-icon></button>
            </el-tooltip>
            <el-tooltip content="删除" placement="top" :show-after="400">
              <button class="k8s-act-btn k8s-act-btn--danger" :disabled="row.status === 'creating' || row.status === 'deleting'" @click="onDelete(row)"><el-icon><Delete /></el-icon></button>
            </el-tooltip>
          </div>
        </template>
      </EnhancedTable>
    </el-card>

    <el-dialog v-model="importVisible" title="导入集群" class="dialog-lg">
      <el-form :model="importForm" label-width="120px">
        <el-form-item label="名称">
          <el-input v-model="importForm.name" />
        </el-form-item>
        <el-form-item label="kubeconfig">
          <el-input v-model="importForm.kubeconfig" type="textarea" :rows="12" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="importVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="doImport">导入</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" title="编辑集群" class="dialog-md">
      <el-form :model="editForm" label-width="120px">
        <el-form-item label="集群名称">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item v-if="editForm.type === 'imported'" label="kubeconfig(可选)">
          <el-input v-model="editForm.kubeconfig" type="textarea" :rows="8" placeholder="不修改可留空；需要更新时粘贴完整 kubeconfig 内容" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="editSaving" @click="doEdit">保存</el-button>
      </template>
    </el-dialog>

    <Teleport to="body">
      <div
        v-if="contextMenuVisible"
        class="cluster-context-backdrop"
        @click="closeClusterContextMenu"
        @contextmenu.prevent="closeClusterContextMenu"
      />
      <div
        v-if="contextMenuVisible && contextMenuData"
        class="cluster-context-menu"
        :style="contextMenuStyle"
        @click.stop
      >
        <button class="cluster-context-item" type="button" @click="openContextCluster">
          <K8sClusterIcon class="cluster-context-icon" />
          <span>进入 K8s 管理</span>
        </button>
        <button :class="['cluster-context-item', isClusterPinned(contextMenuData.id) ? 'cluster-context-item--active' : '']" type="button" @click="toggleContextShortcut">
          <el-icon><StarFilled v-if="isClusterPinned(contextMenuData.id)" /><Star v-else /></el-icon>
          <span>{{ isClusterPinned(contextMenuData.id) ? '取消左侧快捷入口' : '固定到左侧快捷入口' }}</span>
        </button>
        <button class="cluster-context-item" type="button" @click="healthContextCluster">
          <el-icon><CircleCheck /></el-icon>
          <span>健康检查</span>
        </button>
        <button class="cluster-context-item" type="button" @click="editContextCluster">
          <el-icon><EditPen /></el-icon>
          <span>编辑集群</span>
        </button>
        <button class="cluster-context-item cluster-context-item--danger" type="button" @click="deleteContextCluster">
          <el-icon><Delete /></el-icon>
          <span>删除集群</span>
        </button>
      </div>
    </Teleport>

  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/app/store/user'
import { ElMessageBox } from 'element-plus'
import { CircleCheck, CircleClose, Delete, EditPen, RefreshRight, Search, Star, StarFilled, Upload } from '@element-plus/icons-vue'
import * as clustersApi from '@/features/clusters/api/clusters'
import { getClusterUnavailableMessage, useClusterShortcuts } from '@/app/composables/useClusterShortcuts'
import { K8sClusterIcon } from '@/shared/icons/appIcons'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import { useContextMenu } from '@/shared/utils/contextMenu'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'

const router = useRouter()
const userStore = useUserStore()
const {
  isPinned: isClusterPinned,
  toggleCluster: toggleClusterPin,
  unpinCluster,
  syncShortcutsFromClusters
} = useClusterShortcuts()
const {
  visible: contextMenuVisible,
  data: contextMenuData,
  style: contextMenuStyle,
  open: openContextMenu,
  close: closeClusterContextMenu,
  bumpViewport: bumpContextMenuViewport
} = useContextMenu<clustersApi.ClusterItem>({ width: 228, height: 266 })

const canOpenK8s = computed(() => {
  const perms = userStore.permissions ?? []
  return perms.includes('cluster:read') || perms.includes('k8s:read') || perms.includes('k8s:rbac_read') || perms.includes('k8s:permission_audit')
})

const loading = ref(false)
const saving = ref(false)

const query = reactive<{ keyword: string; status: string | undefined; type: string | undefined }>({ keyword: '', status: undefined, type: undefined })
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const list = ref<clustersApi.ClusterItem[]>([])
const sortBy = ref<string | undefined>(undefined)
const order = ref<'asc' | 'desc' | undefined>(undefined)

const tableUiRef = ref<InstanceType<typeof EnhancedTable> | null>(null)
const selectedRows = ref<clustersApi.ClusterItem[]>([])
const selectedCount = computed(() => selectedRows.value.length)
const visibleStatusSummary = computed(() => {
  const summary = {
    active: 0,
    degraded: 0,
    disabled: 0,
    creating: 0,
    deleting: 0,
  }

  for (const item of list.value) {
    switch (item.status) {
      case 'active':
        summary.active += 1
        break
      case 'degraded':
        summary.degraded += 1
        break
      case 'disabled':
        summary.disabled += 1
        break
      case 'creating':
        summary.creating += 1
        break
      case 'deleting':
        summary.deleting += 1
        break
    }
  }

  return summary
})
const visibleAttentionCount = computed(() => (
  visibleStatusSummary.value.degraded +
  visibleStatusSummary.value.disabled +
  visibleStatusSummary.value.creating +
  visibleStatusSummary.value.deleting
))
const visibleNodeCount = computed(() => list.value.reduce((sum, item) => sum + (item.node_count ?? 0), 0))

const healthChecking = reactive<Record<string, boolean>>({})
const bulkHealthLoading = ref(false)

const columns = computed<EnhancedColumn[]>(() => [
  { key: 'name', label: '名称', prop: 'name', sortable: 'custom', minWidth: 200 },
  { key: 'k8s_version', label: 'K8s 版本', prop: 'k8s_version', width: 140, sortable: 'custom' },
  { key: 'node_count', label: '节点数', prop: 'node_count', width: 100, sortable: 'custom', align: 'center', headerAlign: 'center' },
  { key: 'status', label: '状态', prop: 'status', width: 160, sortable: 'custom' },
  { key: 'created_at', label: '创建时间', prop: 'created_at', minWidth: 180, sortable: 'custom' },
  { key: 'actions', label: '操作', width: 180, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false }
])

const importVisible = ref(false)
const importForm = reactive<{ name: string; kubeconfig: string }>({ name: '', kubeconfig: '' })

const editVisible = ref(false)
const editSaving = ref(false)
const editForm = reactive<{ id: number | null; name: string; type: clustersApi.ClusterItem['type'] | null; kubeconfig: string }>({
  id: null,
  name: '',
  type: null,
  kubeconfig: ''
})

async function load() {
  loading.value = true
  try {
    const data = await clustersApi.listClusters({
      page: page.value,
      page_size: pageSize.value,
      keyword: query.keyword || undefined,
      status: query.status,
      type: query.type,
      sort_by: sortBy.value,
      order: order.value
    })
    list.value = data.list
    total.value = data.total
    syncShortcutsFromClusters(data.list)
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loading.value = false
  }

  window.dispatchEvent(new Event('k8s-platform:clusters-changed'))
}

function onSortChange(v: { prop?: string; order?: 'ascending' | 'descending' | null }) {
  sortBy.value = v?.prop || undefined
  order.value = v?.order === 'ascending' ? 'asc' : v?.order === 'descending' ? 'desc' : undefined
  page.value = 1
  void load()
}

function onSelectionChange(rows: clustersApi.ClusterItem[]) {
  selectedRows.value = rows
}

function clearSelection() {
  tableUiRef.value?.clearSelection()
}

async function bulkHealth() {
  if (selectedRows.value.length === 0) return
  try {
    bulkHealthLoading.value = true
    const rows = [...selectedRows.value]
    const tasks = rows.map(async (c) => {
      healthChecking[String(c.id)] = true
      try {
        const data = await clustersApi.checkClusterHealth(c.id)
        updateClusterStatusByHealth(c.id, data.api_ok)
        return { ok: true, apiOk: data.api_ok }
      } catch (e) {
        updateClusterStatusByHealth(c.id, false)
        throw e
      } finally {
        healthChecking[String(c.id)] = false
      }
    })
    const settled = await Promise.allSettled(tasks)
    const okCount = settled.filter((s) => s.status === 'fulfilled').length
    const failCount = settled.length - okCount
    if (failCount === 0) notifySuccess(`健康检查完成：成功 ${okCount} 个`)
    else notifyError(`健康检查部分失败：成功 ${okCount} 个，失败 ${failCount} 个`)
    // 重新加载列表，以获取健康检查后回填的 k8s_version / node_count
    await load()
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    bulkHealthLoading.value = false
  }
}

function onResetFilters() {
  query.keyword = ''
  query.status = undefined
  query.type = undefined
  page.value = 1
  void load()
}

function onSearch() {
  page.value = 1
  void load()
}

function statusTagType(status: clustersApi.ClusterItem['status']) {
  if (status === 'active') return 'success'
  if (status === 'degraded') return 'warning'
  if (status === 'disabled') return 'info'
  if (status === 'creating') return 'info'
  if (status === 'deleting') return 'danger'
  return 'info'
}

function statusLabel(status: clustersApi.ClusterItem['status']) {
  if (status === 'active') return '正常'
  if (status === 'degraded') return '降级'
  if (status === 'disabled') return '已禁用'
  if (status === 'creating') return '创建中'
  if (status === 'deleting') return '删除中'
  return status
}

function openImport() {
  importForm.name = ''
  importForm.kubeconfig = ''
  importVisible.value = true
}

async function openCluster(row: clustersApi.ClusterItem) {
  if (!canOpenK8s.value) {
    notifyError('无权限进入 K8s 管理')
    return
  }
  const msg = clusterBlockedMessage(row.status)
  if (msg) {
    notifyError(msg)
    return
  }
  await router.push({ name: 'K8sClusterManage', params: { clusterId: String(row.id) } })
}

function clusterBlockedMessage(status: clustersApi.ClusterItem['status']): string | null {
  return getClusterUnavailableMessage(status)
}

function toggleClusterShortcut(row: clustersApi.ClusterItem) {
  const pinned = toggleClusterPin(row)
  notifySuccess(pinned ? `已固定到左侧快捷入口：${row.name}` : `已从左侧快捷入口移除：${row.name}`)
}

function openClusterContextMenu(event: MouseEvent, row: clustersApi.ClusterItem) {
  openContextMenu(event, row)
}

function onRowContextMenu(row: clustersApi.ClusterItem, event: MouseEvent) {
  openClusterContextMenu(event, row)
}

async function openContextCluster() {
  const row = contextMenuData.value
  closeClusterContextMenu()
  if (row) await openCluster(row)
}

function toggleContextShortcut() {
  const row = contextMenuData.value
  closeClusterContextMenu()
  if (row) toggleClusterShortcut(row)
}

function healthContextCluster() {
  const row = contextMenuData.value
  closeClusterContextMenu()
  if (row) void onHealth(row.id)
}

function editContextCluster() {
  const row = contextMenuData.value
  closeClusterContextMenu()
  if (row) openEdit(row)
}

function deleteContextCluster() {
  const row = contextMenuData.value
  closeClusterContextMenu()
  if (row) void onDelete(row)
}

function onContextMenuKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') closeClusterContextMenu()
}

function updateClusterStatusByHealth(id: number, apiOk: boolean) {
  const status: clustersApi.ClusterItem['status'] = apiOk ? 'active' : 'degraded'
  const target = String(id)
  const idx = list.value.findIndex((c) => String(c.id) === target)
  if (idx < 0) return
  const cur = list.value[idx]
  list.value = [...list.value.slice(0, idx), { ...cur, status }, ...list.value.slice(idx + 1)]
}

async function doImport() {
  saving.value = true
  try {
    const data = await clustersApi.importCluster({ name: importForm.name, kubeconfig: importForm.kubeconfig })
    notifySuccess(`已导入：cluster_id=${data.cluster_id}`)
    importVisible.value = false
    await load()
    window.dispatchEvent(new Event('k8s-platform:clusters-changed'))
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    saving.value = false
  }
}

function openEdit(row: clustersApi.ClusterItem) {
  if (row.status === 'creating') {
    notifyError('集群正在创建中，暂不可编辑')
    return
  }
  if (row.status === 'deleting') {
    notifyError('集群正在删除中，暂不可编辑')
    return
  }
  editForm.id = row.id
  editForm.name = row.name
  editForm.type = row.type
  editForm.kubeconfig = ''
  editVisible.value = true
}

async function doEdit() {
  if (!editForm.id) return
  const id = editForm.id
  const name = editForm.name.trim()
  if (!name) {
    notifyError('集群名称不能为空')
    return
  }
  editSaving.value = true
  try {
    const kubeconfig = editForm.kubeconfig.trim()
    await clustersApi.patchCluster(id, { name, kubeconfig: kubeconfig ? kubeconfig : undefined })
    notifySuccess('已保存')
    editVisible.value = false
    await load()
    window.dispatchEvent(new Event('k8s-platform:clusters-changed'))
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    editSaving.value = false
  }
}

async function onDelete(row: clustersApi.ClusterItem) {
  if (row.status === 'creating') {
    notifyError('集群正在创建中，暂不可删除')
    return
  }
  if (row.status === 'deleting') {
    notifyError('集群正在删除中，暂不可删除')
    return
  }
  try {
    await ElMessageBox.confirm(`确认删除集群：${row.name}？`, '提示', { type: 'warning' })
    await clustersApi.deleteCluster(row.id)
    unpinCluster(row.id)
    notifySuccess('已删除')
    await load()
    window.dispatchEvent(new Event('k8s-platform:clusters-changed'))
  } catch (e) {
    if (e === 'cancel') return
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  }
}

async function onHealth(id: number) {
  if (healthChecking[String(id)]) return
  healthChecking[String(id)] = true
  try {
    const data = await clustersApi.checkClusterHealth(id)
    updateClusterStatusByHealth(id, data.api_ok)
    const msg = `api_ok=${data.api_ok}, nodes=${data.node_ready}/${data.node_total} (${data.checked_at})`
    if (data.api_ok) notifySuccess(msg)
    else notifyError(msg)
    // 重新加载列表，以获取健康检查后回填的 k8s_version / node_count
    await load()
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    healthChecking[String(id)] = false
  }
}

onMounted(() => {
  void load()
  window.addEventListener('resize', bumpContextMenuViewport)
  window.addEventListener('keydown', onContextMenuKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', bumpContextMenuViewport)
  window.removeEventListener('keydown', onContextMenuKeydown)
})
</script>

<style scoped>
.clusters-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.clusters-panel {
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
}

.clusters-panel--filters {
  padding: 16px 18px;
}

.clusters-overview {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 14px;
  padding: 4px;
  border: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.06));
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.96), rgba(241, 245, 249, 0.72));
}

.clusters-overview__item {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 68px;
  padding: 12px 16px 12px 18px;
  border: 1px solid rgba(255, 255, 255, 0.82);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.94);
  overflow: hidden;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72);
}

.clusters-overview__item::before {
  content: '';
  position: absolute;
  top: 10px;
  bottom: 10px;
  left: 0;
  width: 3px;
  border-radius: 0 999px 999px 0;
  background: var(--overview-accent, #94a3b8);
}

.clusters-overview__item::after {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at right top, color-mix(in srgb, var(--overview-accent, #94a3b8) 10%, transparent), transparent 48%);
  pointer-events: none;
}

.clusters-overview__item--total {
  --overview-accent: #2563eb;
}

.clusters-overview__item--ok {
  --overview-accent: #10b981;
}

.clusters-overview__item--warn {
  --overview-accent: #f59e0b;
}

.clusters-overview__item--nodes {
  --overview-accent: #64748b;
}

.clusters-overview__meta {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
}

.clusters-overview__label {
  color: var(--color-text-secondary, #64748b);
  font-size: 12px;
  font-weight: 700;
}

.clusters-overview__hint {
  color: var(--color-text-muted, #94a3b8);
  font-size: 11px;
  font-weight: 600;
  line-height: 1.2;
}

.clusters-overview__value {
  color: var(--color-text-title, rgba(15, 23, 42, 0.92));
  font-size: 28px;
  font-weight: 800;
  line-height: 1;
}

.clusters-overview__item--total .clusters-overview__value {
  color: #1d4ed8;
}

.clusters-overview__item--ok .clusters-overview__value {
  color: #047857;
}

.clusters-overview__item--warn .clusters-overview__value {
  color: #b45309;
}

.clusters-overview__item--nodes .clusters-overview__value {
  color: #334155;
}

.clusters-filters {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.clusters-filters__fields {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  flex-wrap: wrap;
  gap: 12px;
}

.clusters-filters__actions {
  display: flex;
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.clusters-filter {
  width: 148px;
}

.clusters-filter--search {
  min-width: 220px;
  flex: 0 1 280px;
}

:deep(.clusters-filter .el-input__wrapper),
:deep(.clusters-filter .el-select__wrapper) {
  min-height: 40px;
  border-radius: 12px;
}

.page-card--clusters {
  position: relative;
}

.table-summary {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.table-summary__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-title, rgba(15, 23, 42, 0.92));
}

.table-summary__meta {
  color: var(--color-text-muted, #94a3b8);
  font-size: 12px;
  font-weight: 600;
}

.table-summary__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(59, 130, 246, 0.1);
  color: #2563eb;
  font-size: 12px;
  font-weight: 700;
}

.k8s-act-btn .cluster-enter-icon {
  width: 18px;
  height: 18px;
  display: block;
  transform: scale(1.04);
  transform-origin: center;
}

.k8s-act-btn--pinned {
  color: #d97706;
  border-color: rgba(245, 158, 11, 0.26);
  background: rgba(245, 158, 11, 0.14);
  box-shadow: 0 8px 18px rgba(245, 158, 11, 0.14);
}

.k8s-act-btn--pinned:hover {
  background: rgba(245, 158, 11, 0.18);
  box-shadow: 0 12px 22px rgba(245, 158, 11, 0.18);
}

.cluster-context-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1990;
  background: transparent;
}

.cluster-context-menu {
  position: fixed;
  z-index: 1991;
  width: 228px;
  padding: 8px;
  border-radius: 14px;
  border: 1px solid var(--color-border-subtle, rgba(226, 232, 240, 0.9));
  background: var(--color-glass-bg-strong, rgba(255, 255, 255, 0.96));
  box-shadow: var(--shadow-dropdown, 0 18px 48px rgba(15, 23, 42, 0.18));
  backdrop-filter: blur(16px);
}

.cluster-context-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 38px;
  padding: 0 10px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--color-text-secondary, #475569);
  font-size: 13px;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
}

.cluster-context-item:hover {
  background: rgba(59, 130, 246, 0.08);
  color: var(--color-accent-primary, #2563eb);
}

.cluster-context-item--active {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
}

.cluster-context-item--active:hover {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}

.cluster-context-item--danger:hover {
  background: rgba(239, 68, 68, 0.08);
  color: #ef4444;
}

.cluster-context-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

:global(html.dark) .clusters-panel {
  border-color: rgba(148, 163, 184, 0.16);
  background: rgba(15, 23, 42, 0.9);
  box-shadow: 0 18px 36px rgba(2, 6, 23, 0.22);
}

:global(html.dark) .clusters-overview {
  border-color: rgba(148, 163, 184, 0.14);
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.76), rgba(15, 23, 42, 0.58));
}

:global(html.dark) .clusters-overview__item {
  border-color: rgba(148, 163, 184, 0.14);
  background: rgba(15, 23, 42, 0.82);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

:global(html.dark) .clusters-overview__label,
:global(html.dark) .clusters-overview__hint,
:global(html.dark) .table-summary__meta {
  color: var(--color-text-muted, #7c8aa0);
}

:global(html.dark) .clusters-overview__item--total .clusters-overview__value {
  color: #93c5fd;
}

:global(html.dark) .clusters-overview__item--ok .clusters-overview__value {
  color: #6ee7b7;
}

:global(html.dark) .clusters-overview__item--warn .clusters-overview__value {
  color: #fcd34d;
}

:global(html.dark) .clusters-overview__item--nodes .clusters-overview__value {
  color: #cbd5e1;
}

:global(html.dark) .table-summary__badge {
  background: rgba(59, 130, 246, 0.2);
  color: #bfdbfe;
}

:global(html.dark) .k8s-act-btn--pinned {
  color: #fcd34d;
  border-color: rgba(245, 158, 11, 0.3);
  background: rgba(245, 158, 11, 0.18);
  box-shadow: 0 10px 20px rgba(245, 158, 11, 0.14);
}

:global(html.dark) .cluster-context-item--active {
  background: rgba(245, 158, 11, 0.16);
  color: #fcd34d;
}

:global(html.dark) .cluster-context-menu {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.98);
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.36);
}

@media (max-width: 960px) {
  .clusters-filters {
    flex-direction: column;
    align-items: stretch;
  }

  .clusters-overview {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .clusters-filters__actions {
    justify-content: flex-start;
  }
}

@media (max-width: 640px) {
  .clusters-overview {
    grid-template-columns: 1fr;
  }

  .clusters-filter,
  .clusters-filter--search {
    width: 100%;
    min-width: 0;
  }
}
</style>
