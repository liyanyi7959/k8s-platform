<template>
  <div class="user-manage-view">
    <div class="user-card">
      <div class="filter-bar">
        <div class="filter-bar__fields">
          <el-input v-model="keyword" placeholder="搜索用户名" clearable class="w-input-sm" @keyup.enter="fetchData" />
          <el-select v-model="statusFilter" placeholder="状态" clearable class="w-input-sm" @change="fetchData">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </div>

        <div class="filter-bar__actions">
          <el-button @click="fetchData">查询</el-button>
          <el-button type="primary" @click="openCreate">新建用户</el-button>
        </div>
      </div>

      <div class="table-wrap">
        <el-table :data="tableData" v-loading="loading" class="page-table" stripe border>
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="username" label="用户名" width="160" />
          <el-table-column prop="status" label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
                {{ row.status === 'active' ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="roles" label="角色" min-width="180">
            <template #default="{ row }">
              <el-tag v-for="r in row.roles" :key="r" size="small" class="mr-1">{{ r }}</el-tag>
              <span v-if="row.roles.length === 0" class="cell-empty">未分配</span>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="170" />
          <el-table-column label="操作" width="240" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="openEdit(row)">编辑</el-button>
              <el-button size="small" type="warning" @click="handleResetPwd(row)">重置密码</el-button>
              <el-popconfirm title="确定删除?" @confirm="handleDelete(row.id)">
                <template #reference>
                  <el-button size="small" type="danger" :disabled="row.username === 'admin'">删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="fetchData"
        />
      </div>
    </div><!-- /user-card -->

    <!-- 新建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑用户' : '新建用户'" width="480px" destroy-on-close>
      <el-form :model="form" label-width="80px">
        <el-form-item label="用户名">
          <el-input v-model="form.username" :disabled="isEdit" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item v-if="!isEdit" label="密码">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.role_ids" multiple placeholder="选择角色" class="w-full">
            <el-option v-for="r in roleOptions" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="isEdit" label="状态">
          <el-select v-model="form.status">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog v-model="resetPwdVisible" title="重置密码" width="400px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item label="新密码">
          <el-input v-model="newPassword" type="password" placeholder="输入新密码" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetPwdVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="confirmResetPwd">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getUsers, createUser, updateUser, deleteUser, resetPassword,
  getRoles,
  type UserItem, type RoleItem,
} from '../api/users'

const keyword = ref('')
const statusFilter = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const tableData = ref<UserItem[]>([])

const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const submitting = ref(false)
const form = ref({ username: '', password: '', role_ids: [] as number[], status: 'active' })
const roleOptions = ref<RoleItem[]>([])

const resetPwdVisible = ref(false)
const resetPwdUserId = ref(0)
const newPassword = ref('')

async function fetchData() {
  loading.value = true
  try {
    const result = await getUsers({ page: page.value, page_size: pageSize.value, keyword: keyword.value, status: statusFilter.value })
    tableData.value = result.items ?? []
    total.value = result.total ?? 0
  } finally {
    loading.value = false
  }
}

async function fetchRoles() {
  roleOptions.value = await getRoles()
}

function openCreate() {
  isEdit.value = false
  form.value = { username: '', password: '', role_ids: [], status: 'active' }
  dialogVisible.value = true
}

function openEdit(row: UserItem) {
  isEdit.value = true
  editId.value = row.id
  form.value = {
    username: row.username,
    password: '',
    role_ids: roleOptions.value.filter(r => row.roles.includes(r.name)).map(r => r.id),
    status: row.status,
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    if (isEdit.value) {
      await updateUser(editId.value, { status: form.value.status, role_ids: form.value.role_ids })
      ElMessage.success('更新成功')
    } else {
      if (!form.value.username || !form.value.password) {
        ElMessage.warning('用户名和密码不能为空')
        return
      }
      await createUser({ username: form.value.username, password: form.value.password, role_ids: form.value.role_ids })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await fetchData()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '操作失败'))
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await deleteUser(id)
    ElMessage.success('已删除')
    await fetchData()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '删除失败'))
  }
}

function handleResetPwd(row: UserItem) {
  resetPwdUserId.value = row.id
  newPassword.value = ''
  resetPwdVisible.value = true
}

async function confirmResetPwd() {
  if (!newPassword.value) {
    ElMessage.warning('请输入新密码')
    return
  }
  submitting.value = true
  try {
    await resetPassword(resetPwdUserId.value, newPassword.value)
    ElMessage.success('密码已重置')
    resetPwdVisible.value = false
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '重置失败'))
  } finally {
    submitting.value = false
  }
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

onMounted(() => {
  fetchData()
  fetchRoles()
})
</script>

<style scoped>
.user-manage-view {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.user-card {
  background: var(--color-bg-card, #ffffff);
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  border-radius: 12px;
  box-shadow: var(--shadow-card, 0 1px 3px rgba(15, 23, 42, 0.05));
  overflow: hidden;
}
.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 18px 20px 14px;
}
.filter-bar__fields {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
.filter-bar__actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.w-input-sm {
  width: 160px;
}
.w-full {
  width: 100%;
}
.mr-1 {
  margin-right: 4px;
}
.cell-empty {
  font-size: 12px;
  color: var(--color-text-muted, #94a3b8);
}
.table-wrap {
  padding: 0 16px;
}
.page-table {
  width: 100%;
}
.pagination-wrap {
  padding: 14px 20px 18px;
  display: flex;
  justify-content: flex-end;
}
</style>
