<template>
  
    <el-card class="page-card">
      <div class="srv-query-bar">
        <!-- 搜索区 -->
        <div class="qb-search">
          <el-icon class="qb-search-icon"><Search /></el-icon>
          <el-input
            v-model="query.keyword"
            class="qb-keyword"
            size="default"
            placeholder="搜索集群名称…"
            clearable
            @keyup.enter="onSearch"
          />
        </div>

        <!-- 筛选区 -->
        <div class="qb-filters">
          <el-select v-model="query.status" class="qb-select" size="default" clearable placeholder="状态">
            <el-option label="正常" value="active" />
            <el-option label="已禁用" value="disabled" />
            <el-option label="创建中" value="creating" />
            <el-option label="降级" value="degraded" />
            <el-option label="删除中" value="deleting" />
          </el-select>
          <el-select v-model="query.type" class="qb-select" size="default" clearable placeholder="类型">
            <el-option label="导入" value="imported" />
            <el-option label="创建" value="created" />
          </el-select>
        </div>

        <!-- 操作区 -->
        <div class="qb-actions">
          <el-tooltip content="查询" placement="top" :show-after="300">
            <el-button class="qb-btn" size="default" @click="onSearch">
              <el-icon><Search /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="重置" placement="top" :show-after="300">
            <el-button class="qb-btn" size="default" @click="onResetFilters">
              <el-icon><RefreshRight /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="导入集群" placement="top" :show-after="300">
            <el-button class="qb-btn" size="default" @click="openImport">
              <el-icon><Upload /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip content="批量检查" placement="top" :show-after="300">
            <el-button class="qb-btn" type="warning" size="default" :disabled="selectedCount === 0" :loading="bulkHealthLoading" @click="bulkHealth">
              <el-icon><CircleCheck /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip v-if="selectedCount > 0" content="清空选择" placement="top" :show-after="300">
            <el-button class="qb-btn" size="default" @click="clearSelection">
              <el-icon><CircleClose /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </div>

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
              <button class="k8s-act-btn k8s-act-btn--warn" @click="toggleClusterShortcut(row)">
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
        <button class="cluster-context-item" type="button" @click="toggleContextShortcut">
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
} = useContextMenu<clustersApi.ClusterItem>({ width: 228, height: 228 })

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
/* 查询栏样式由 enterprise.css 全局提供 (.srv-query-bar / .qb-*) */

/* 操作区按钮：图标模式 */
.qb-actions .qb-btn.el-button {
  width: 34px;
  padding: 0;
}

.k8s-act-btn .cluster-enter-icon {
  width: 18px;
  height: 18px;
  display: block;
  transform: scale(1.04);
  transform-origin: center;
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
  padding: 6px;
  border-radius: 8px;
  border: 1px solid var(--color-border-subtle, rgba(226, 232, 240, 0.9));
  background: var(--color-bg-card, #fff);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.18);
}

.cluster-context-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 34px;
  padding: 0 10px;
  border: none;
  border-radius: 6px;
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

.cluster-context-item--danger:hover {
  background: rgba(239, 68, 68, 0.08);
  color: #ef4444;
}

.cluster-context-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

:global(html.dark) .cluster-context-menu {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.98);
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.36);
}
</style>
