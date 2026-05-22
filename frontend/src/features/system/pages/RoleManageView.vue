<template>
  <div class="role-manage-view">
    <div class="role-card">
      <div class="role-toolbar">
        <div class="role-toolbar__summary">
          <h2 class="role-toolbar__title">角色权限矩阵</h2>
          <p class="role-toolbar__meta">{{ tableData.length }} 个角色，{{ allPermissions.length }} 个权限点，按领域分组管理。</p>
        </div>
        <div class="role-toolbar__actions">
          <el-button @click="fetchAll">刷新</el-button>
          <el-button type="primary" @click="openCreate">新建角色</el-button>
        </div>
      </div>

      <div class="table-wrap">
        <el-table :data="tableData" v-loading="loading" class="page-table" stripe border>
          <el-table-column prop="name" label="角色" width="200">
            <template #default="{ row }">
              <div class="role-name-cell">
                <span class="role-name">{{ row.name }}</span>
                <el-tag v-if="row.builtin" size="small" effect="plain">内置</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="描述" min-width="220" />
          <el-table-column prop="user_count" label="关联用户" width="96" align="center" />
          <el-table-column prop="permissions" label="权限范围" min-width="320">
            <template #default="{ row }">
              <div class="permission-preview">
                <el-tag v-for="code in row.permissions.slice(0, 4)" :key="code" size="small" class="role-tag" effect="plain">
                  {{ permissionLabel(code) }}
                </el-tag>
                <span v-if="row.permissions.length === 0" class="cell-empty">未授予权限</span>
                <span v-else-if="row.permissions.length > 4" class="permission-more">+{{ row.permissions.length - 4 }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="命名空间范围" min-width="220">
            <template #default="{ row }">
              <div v-if="row.namespace_scope" class="scope-preview">
                <span class="scope-preview__cluster">{{ row.namespace_scope.cluster_name || `集群 #${row.namespace_scope.cluster_id}` }}</span>
                <div class="scope-preview__namespaces">
                  <el-tag v-for="namespace in row.namespace_scope.namespaces.slice(0, 2)" :key="namespace" size="small" effect="plain">
                    {{ namespace }}
                  </el-tag>
                  <span v-if="row.namespace_scope.namespaces.length > 2" class="permission-more">+{{ row.namespace_scope.namespaces.length - 2 }}</span>
                </div>
              </div>
              <span v-else class="cell-empty">未限制</span>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="170" />
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="openEdit(row)">编辑</el-button>
              <el-popconfirm title="确定删除该角色?" @confirm="handleDelete(row)">
                <template #reference>
                  <el-button size="small" type="danger" :disabled="row.builtin">删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑角色' : '新建角色'" width="900px" destroy-on-close>
      <div class="role-dialog">
        <section class="dialog-section">
          <div class="section-heading">
            <div>
              <h3 class="section-heading__title">基础信息</h3>
              <p class="section-heading__meta">先确认角色名称、描述和检索条件，再配置作用域与权限。</p>
            </div>
          </div>
          <el-form :model="form" label-width="88px" class="role-form">
            <el-form-item label="角色名">
              <el-input v-model="form.name" :disabled="isEdit" placeholder="例如：platform-ops" />
            </el-form-item>
            <el-form-item label="权限检索">
              <el-input v-model="permissionKeyword" clearable placeholder="搜索权限编码、描述、分组或资源域" />
            </el-form-item>
            <el-form-item label="描述" class="role-form__full">
              <el-input v-model="form.description" type="textarea" :rows="3" placeholder="描述角色职责和适用范围" />
            </el-form-item>
          </el-form>
        </section>

        <section class="dialog-section">
          <div class="section-heading">
            <div>
              <h3 class="section-heading__title">命名空间范围</h3>
              <p class="section-heading__meta">选择命名空间权限后，可进一步限制到某个集群下的实际 namespace。</p>
            </div>
            <div class="section-heading__actions">
              <span class="section-stat">{{ namespaceScopeSummary }}</span>
            </div>
          </div>

          <div class="scope-controls">
            <div class="scope-field">
              <label class="scope-field__label">所属集群</label>
              <el-select
                v-model="form.namespace_scope.cluster_id"
                class="scope-control"
                clearable
                filterable
                placeholder="选择集群"
                :loading="clusterLoading"
                :disabled="!hasNamespacePermission"
              >
                <el-option v-for="cluster in clusterOptions" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
              </el-select>
            </div>

            <div class="scope-field">
              <label class="scope-field__label">实际 Namespace</label>
              <el-select
                v-model="form.namespace_scope.namespaces"
                class="scope-control"
                multiple
                filterable
                clearable
                collapse-tags
                collapse-tags-tooltip
                placeholder="选择实际 namespace"
                :loading="namespaceLoading"
                :disabled="namespaceSelectionDisabled"
              >
                <el-option v-for="namespace in namespaceOptions" :key="namespace" :label="namespace" :value="namespace" />
              </el-select>
            </div>
          </div>

          <p class="scope-tip">{{ namespaceScopeTip }}</p>
        </section>

        <section class="dialog-section">
          <div class="section-heading">
            <div>
              <h3 class="section-heading__title">权限选择</h3>
              <p class="section-heading__meta">点击选中/取消权限，蓝色表示已授予。</p>
            </div>
            <div class="section-heading__actions">
              <span class="section-stat section-stat--active">{{ form.permissions.length }} 已选</span>
              <span class="section-stat">{{ visiblePermissionCount }} 可选</span>
            </div>
          </div>

          <div v-if="permissionGroups.length === 0" class="permission-empty">未找到匹配的权限点。</div>
          <div v-else class="perm-grid">
            <div v-for="group in permissionGroups" :key="group.key" class="perm-category">
              <div class="perm-category__head">
                <span class="perm-category__label">{{ group.label }}</span>
                <button
                  type="button"
                  class="perm-category__toggle"
                  :class="{ 'is-all': isGroupChecked(group), 'is-partial': isGroupIndeterminate(group) }"
                  @click="handleGroupCheckChange(group, !isGroupChecked(group))"
                >
                  {{ isGroupChecked(group) ? '取消全选' : '全选' }}
                </button>
              </div>
              <div class="perm-category__chips">
                <el-tooltip
                  v-for="item in group.items"
                  :key="item.code"
                  :content="item.code"
                  placement="top"
                  :show-after="400"
                >
                  <button
                    type="button"
                    class="perm-chip"
                    :class="{ 'is-active': selectedPermissionSet.has(item.code) }"
                    @click="togglePermission(item.code, !selectedPermissionSet.has(item.code))"
                  >
                    <span class="perm-chip__dot"></span>
                    <span class="perm-chip__text">{{ item.description }}</span>
                  </button>
                </el-tooltip>
              </div>
            </div>
          </div>
        </section>
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

import { listClusters, type ClusterItem } from '@/features/clusters/api/clusters'
import { listNamespaces } from '@/features/k8s/api/namespace'

import {
  getRoles,
  createRole,
  updateRole,
  deleteRole,
  getPermissions,
  type RoleItem,
  type PermissionItem,
  type RoleNamespaceScope,
  type RoleNamespaceScopePayload,
} from '../api/users'

interface PermissionGroup {
  key: string
  label: string
  items: PermissionItem[]
}

interface NamespaceScopeForm {
  cluster_id: number | null
  namespaces: string[]
}

interface RoleFormState {
  name: string
  description: string
  permissions: string[]
  namespace_scope: NamespaceScopeForm
}

const permissionCategoryOrder: Record<string, number> = {
  system: 1,
  cluster: 2,
  namespace: 3,
  k8s: 4,
  rbac: 5,
  security: 6,
  custom: 99,
}

const loading = ref(false)
const tableData = ref<RoleItem[]>([])
const allPermissions = ref<PermissionItem[]>([])
const clusterOptions = ref<ClusterItem[]>([])
const namespaceOptions = ref<string[]>([])

const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const submitting = ref(false)
const clusterLoading = ref(false)
const clustersLoaded = ref(false)
const namespaceLoading = ref(false)
const permissionKeyword = ref('')
const form = ref<RoleFormState>(createEmptyForm())

let namespaceRequestSerial = 0

function createEmptyForm(): RoleFormState {
  return {
    name: '',
    description: '',
    permissions: [],
    namespace_scope: {
      cluster_id: null,
      namespaces: []
    }
  }
}

function normalizeScope(scope?: RoleNamespaceScope | null): NamespaceScopeForm {
  return {
    cluster_id: scope?.cluster_id ? Number(scope.cluster_id) : null,
    namespaces: Array.isArray(scope?.namespaces) ? normalizeStrings(scope.namespaces) : []
  }
}

function normalizeStrings(values: string[]): string[] {
  return Array.from(new Set(values.map((item) => String(item).trim()).filter(Boolean))).sort((left, right) => left.localeCompare(right, 'zh-CN'))
}

const permissionMap = computed<Record<string, PermissionItem>>(() => {
  return allPermissions.value.reduce<Record<string, PermissionItem>>((acc, item) => {
    acc[item.code] = item
    return acc
  }, {})
})

const permissionGroups = computed<PermissionGroup[]>(() => {
  const keyword = permissionKeyword.value.trim().toLowerCase()
  const groupMap = new Map<string, PermissionGroup>()

  allPermissions.value.forEach((item) => {
    const haystack = `${item.category_label} ${item.description} ${item.code}`.toLowerCase()
    if (keyword && !haystack.includes(keyword)) {
      return
    }

    const groupKey = item.category || 'custom'
    const existing = groupMap.get(groupKey)
    if (existing) {
      existing.items.push(item)
      return
    }

    groupMap.set(groupKey, {
      key: groupKey,
      label: item.category_label || '自定义权限',
      items: [item]
    })
  })

  return Array.from(groupMap.values())
    .map((group) => ({
      ...group,
      items: [...group.items].sort((left, right) => left.code.localeCompare(right.code))
    }))
    .sort((left, right) => {
      const leftOrder = permissionCategoryOrder[left.key] ?? 99
      const rightOrder = permissionCategoryOrder[right.key] ?? 99
      if (leftOrder !== rightOrder) {
        return leftOrder - rightOrder
      }
      return left.label.localeCompare(right.label, 'zh-CN')
    })
})

const selectedPermissionSet = computed(() => new Set(form.value.permissions))
const visiblePermissionCount = computed(() => permissionGroups.value.reduce((sum, group) => sum + group.items.length, 0))
const hasNamespacePermission = computed(() => form.value.permissions.some((code) => code.startsWith('namespace:')))
const selectedNamespaceClusterId = computed(() => Number(form.value.namespace_scope.cluster_id || 0))
const namespaceSelectionDisabled = computed(() => !hasNamespacePermission.value || !selectedNamespaceClusterId.value)

const namespaceScopeSummary = computed(() => {
  if (!hasNamespacePermission.value) {
    return '未启用命名空间权限'
  }
  if (!selectedNamespaceClusterId.value || form.value.namespace_scope.namespaces.length === 0) {
    return '未限制到具体 namespace'
  }
  const clusterName = clusterOptions.value.find((item) => item.id === selectedNamespaceClusterId.value)?.name || `集群 #${selectedNamespaceClusterId.value}`
  return `${clusterName} · ${form.value.namespace_scope.namespaces.length} 个 namespace`
})

const namespaceScopeTip = computed(() => {
  if (!hasNamespacePermission.value) {
    return '先选择“命名空间查看”或“命名空间管理”，再限制到具体 namespace。'
  }
  if (clusterLoading.value) {
    return '正在加载集群列表。'
  }
  if (clusterOptions.value.length === 0) {
    return '当前没有可用集群，暂时无法配置 namespace 范围。'
  }
  if (!selectedNamespaceClusterId.value) {
    return '未选择集群时，不会保存 namespace 限制。'
  }
  if (namespaceLoading.value) {
    return '正在加载该集群的实际 namespace 列表。'
  }
  if (namespaceOptions.value.length === 0) {
    return '该集群当前没有可选 namespace，或你还没有对应读取权限。'
  }
  if (form.value.namespace_scope.namespaces.length === 0) {
    return '从列表里选择一个或多个实际 namespace 后，角色才会保存命名空间范围。'
  }
  return `已选择 ${form.value.namespace_scope.namespaces.length} 个实际 namespace。`
})

async function fetchData() {
  loading.value = true
  try {
    tableData.value = await getRoles()
  } catch (error) {
    tableData.value = []
    ElMessage.error(getErrorMessage(error, '角色列表加载失败'))
  } finally {
    loading.value = false
  }
}

async function fetchPermissions() {
  try {
    allPermissions.value = await getPermissions()
  } catch (error) {
    allPermissions.value = []
    ElMessage.error(getErrorMessage(error, '权限列表加载失败'))
  }
}

async function fetchAll() {
  await Promise.all([fetchData(), fetchPermissions()])
}

async function ensureClusterOptions() {
  if (clustersLoaded.value || clusterLoading.value) {
    return
  }
  clusterLoading.value = true
  try {
    const result = await listClusters({ page: 1, page_size: 100 })
    clusterOptions.value = Array.isArray(result.list) ? result.list : []
    clustersLoaded.value = true
  } catch (error) {
    clusterOptions.value = []
    ElMessage.error(getErrorMessage(error, '集群列表加载失败'))
  } finally {
    clusterLoading.value = false
  }
}

async function fetchNamespaceOptions(clusterId: number) {
  const requestId = ++namespaceRequestSerial
  namespaceLoading.value = true
  try {
    const result = await listNamespaces(clusterId, { sort_by: 'metadata.name', order: 'asc' })
    if (requestId !== namespaceRequestSerial) {
      return
    }
    namespaceOptions.value = normalizeStrings(Array.isArray(result.list) ? result.list.map((item) => item?.metadata?.name ?? '') : [])
    const available = new Set(namespaceOptions.value)
    form.value.namespace_scope.namespaces = form.value.namespace_scope.namespaces.filter((item) => available.has(item))
  } catch (error) {
    if (requestId !== namespaceRequestSerial) {
      return
    }
    namespaceOptions.value = []
    ElMessage.error(getErrorMessage(error, '命名空间列表加载失败'))
  } finally {
    if (requestId === namespaceRequestSerial) {
      namespaceLoading.value = false
    }
  }
}

async function prepareNamespaceScopeOptions() {
  if (!dialogVisible.value || !hasNamespacePermission.value) {
    return
  }
  await ensureClusterOptions()
  if (!selectedNamespaceClusterId.value && clusterOptions.value.length === 1) {
    form.value.namespace_scope.cluster_id = clusterOptions.value[0].id
    return
  }
  if (selectedNamespaceClusterId.value) {
    await fetchNamespaceOptions(selectedNamespaceClusterId.value)
  }
}

async function openCreate() {
  isEdit.value = false
  editId.value = 0
  permissionKeyword.value = ''
  namespaceOptions.value = []
  form.value = createEmptyForm()
  dialogVisible.value = true
  await prepareNamespaceScopeOptions()
}

async function openEdit(row: RoleItem) {
  isEdit.value = true
  editId.value = row.id
  permissionKeyword.value = ''
  namespaceOptions.value = []
  form.value = {
    name: row.name,
    description: row.description,
    permissions: [...row.permissions],
    namespace_scope: normalizeScope(row.namespace_scope),
  }
  dialogVisible.value = true
  await prepareNamespaceScopeOptions()
}

function buildNamespaceScopePayload(): RoleNamespaceScopePayload | null {
  const clusterId = selectedNamespaceClusterId.value
  const namespaces = normalizeStrings(form.value.namespace_scope.namespaces)
  if (!hasNamespacePermission.value || clusterId === 0 || namespaces.length === 0) {
    return null
  }
  return {
    cluster_id: clusterId,
    namespaces
  }
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    ElMessage.warning('角色名不能为空')
    return
  }
  if (form.value.permissions.length === 0) {
    ElMessage.warning('请至少选择一个权限点')
    return
  }

  submitting.value = true
  try {
    const payload = {
      description: form.value.description,
      permissions: [...form.value.permissions],
      namespace_scope: buildNamespaceScopePayload(),
    }
    if (isEdit.value) {
      await updateRole(editId.value, payload)
      ElMessage.success('角色已更新')
    } else {
      await createRole({ name: form.value.name.trim(), ...payload })
      ElMessage.success('角色已创建')
    }
    dialogVisible.value = false
    await fetchData()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '角色操作失败'))
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: RoleItem) {
  try {
    await deleteRole(row.id)
    ElMessage.success(`已删除角色 ${row.name}`)
    await fetchData()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '删除失败'))
  }
}

function permissionLabel(code: string) {
  return permissionMap.value[code]?.description ?? code
}

function groupSelectedCount(group: PermissionGroup) {
  return group.items.filter((item) => selectedPermissionSet.value.has(item.code)).length
}

function isGroupChecked(group: PermissionGroup) {
  return group.items.length > 0 && group.items.every((item) => selectedPermissionSet.value.has(item.code))
}

function isGroupIndeterminate(group: PermissionGroup) {
  const selectedCount = groupSelectedCount(group)
  return selectedCount > 0 && selectedCount < group.items.length
}

function togglePermission(code: string, checked: string | number | boolean) {
  const next = new Set(form.value.permissions)
  if (Boolean(checked)) {
    next.add(code)
  } else {
    next.delete(code)
  }
  form.value.permissions = Array.from(next).sort((left, right) => left.localeCompare(right))
}

function toggleGroup(group: PermissionGroup, checked: string | number | boolean) {
  const next = new Set(form.value.permissions)
  if (Boolean(checked)) {
    group.items.forEach((item) => next.add(item.code))
  } else {
    group.items.forEach((item) => next.delete(item.code))
  }
  form.value.permissions = Array.from(next).sort((left, right) => left.localeCompare(right))
}

function handleGroupCheckChange(group: PermissionGroup, checked: string | number | boolean) {
  toggleGroup(group, checked)
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

watch(
  () => hasNamespacePermission.value,
  (enabled) => {
    if (!dialogVisible.value) {
      return
    }
    if (!enabled) {
      namespaceOptions.value = []
      return
    }
    void prepareNamespaceScopeOptions()
  }
)

watch(
  () => selectedNamespaceClusterId.value,
  (clusterId, previousClusterId) => {
    if (!dialogVisible.value || !hasNamespacePermission.value) {
      return
    }
    if (!clusterId) {
      namespaceOptions.value = []
      form.value.namespace_scope.namespaces = []
      return
    }
    if (previousClusterId && previousClusterId !== clusterId) {
      form.value.namespace_scope.namespaces = []
    }
    void fetchNamespaceOptions(clusterId)
  }
)

onMounted(() => {
  fetchAll()
})
</script>

<style scoped>
.role-manage-view {
  display: flex;
  flex-direction: column;
}

.role-card {
  background: #ffffff;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 14px;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.role-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px 14px;
  flex-wrap: wrap;
}

.role-toolbar__summary {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.role-toolbar__title {
  margin: 0;
  font-size: 18px;
  line-height: 1.3;
  color: #111827;
}

.role-toolbar__meta {
  margin: 0;
  font-size: 13px;
  color: #6b7280;
}

.role-toolbar__actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.table-wrap {
  padding: 0 16px 18px;
}

.page-table {
  width: 100%;
}

.role-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.role-name {
  font-weight: 600;
  color: #111827;
}

.permission-preview,
.scope-preview__namespaces {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.scope-preview {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.scope-preview__cluster {
  font-size: 12px;
  color: #374151;
  font-weight: 600;
}

.role-tag {
  margin-right: 0;
}

.permission-more,
.cell-empty {
  font-size: 12px;
  color: #94a3b8;
}

.role-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.dialog-section {
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 12px;
  background: #ffffff;
  padding: 16px;
}

.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.section-heading__title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #111827;
}

.section-heading__meta {
  margin: 4px 0 0;
  font-size: 13px;
  color: #6b7280;
}

.section-heading__actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.section-stat {
  display: inline-flex;
  align-items: center;
  min-height: 32px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: #f8fafc;
  font-size: 12px;
  color: #475569;
}

.role-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 16px;
}

.role-form__full {
  grid-column: 1 / -1;
}

.scope-controls {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.scope-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.scope-field__label {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
}

.scope-control {
  width: 100%;
}

.scope-tip {
  margin: 12px 0 0;
  font-size: 12px;
  color: #6b7280;
}

.permission-empty {
  padding: 32px 12px;
  text-align: center;
  color: #94a3b8;
}

.perm-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.perm-category {
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 10px;
  padding: 12px;
  background: #fafbfc;
}

.perm-category__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.perm-category__label {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}

.perm-category__toggle {
  all: unset;
  cursor: pointer;
  font-size: 12px;
  color: #64748b;
  padding: 2px 8px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}

.perm-category__toggle:hover {
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.06);
}

.perm-category__toggle.is-all {
  color: #3b82f6;
}

.perm-category__toggle.is-partial {
  color: #f59e0b;
}

.perm-category__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.perm-chip {
  all: unset;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 6px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  background: #ffffff;
  font-size: 13px;
  color: #475569;
  transition: all 0.15s ease;
  user-select: none;
}

.perm-chip:hover {
  border-color: #93c5fd;
  background: rgba(59, 130, 246, 0.04);
}

.perm-chip.is-active {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.08);
  color: #1d4ed8;
  font-weight: 500;
}

.perm-chip__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #cbd5e1;
  flex-shrink: 0;
  transition: background 0.15s, box-shadow 0.15s;
}

.perm-chip.is-active .perm-chip__dot {
  background: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
}

.perm-chip__text {
  white-space: nowrap;
}

.section-stat--active {
  background: rgba(59, 130, 246, 0.08);
  border-color: rgba(59, 130, 246, 0.2);
  color: #2563eb;
}

@media (max-width: 900px) {
  .role-form,
  .scope-controls {
    grid-template-columns: 1fr;
  }

  .perm-grid {
    grid-template-columns: 1fr;
  }

  .section-heading {
    flex-direction: column;
    align-items: stretch;
  }

  .section-heading__actions {
    width: 100%;
  }

  .section-stat {
    justify-content: center;
  }
}
</style>
