<template>
  <div class="role-manage-view">
    <div class="role-card">
      <div class="page-header">
        <h3 class="page-title">角色管理</h3>
        <el-button type="primary" @click="openCreate">新建角色</el-button>
      </div>

    <el-table :data="tableData" v-loading="loading" stripe border>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="角色名" width="160" />
      <el-table-column prop="description" label="描述" min-width="200" />
      <el-table-column prop="user_count" label="用户数" width="80" />
      <el-table-column prop="permissions" label="权限" min-width="300">
        <template #default="{ row }">
          <el-tag v-for="p in row.permissions" :key="p" size="small" class="mr-1">{{ p }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-popconfirm title="确定删除?" @confirm="handleDelete(row.id)">
            <template #reference>
              <el-button size="small" type="danger" :disabled="row.name === 'admin'">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
    </div><!-- /role-card -->

    <!-- 新建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑角色' : '新建角色'" width="560px" destroy-on-close>
      <el-form :model="form" label-width="80px">
        <el-form-item label="角色名">
          <el-input v-model="form.name" :disabled="isEdit" placeholder="英文标识" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" placeholder="角色描述" />
        </el-form-item>
        <el-form-item label="权限">
          <el-checkbox-group v-model="form.permissions">
            <el-checkbox
              v-for="p in allPermissions"
              :key="p.code"
              :label="p.code"
              :value="p.code"
            >
              {{ p.code }}<span class="perm-desc"> - {{ p.description }}</span>
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getRoles, createRole, updateRole, deleteRole,
  getPermissions,
  type RoleItem, type PermissionItem,
} from '../api/users'

const loading = ref(false)
const tableData = ref<RoleItem[]>([])
const allPermissions = ref<PermissionItem[]>([])

const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const submitting = ref(false)
const form = ref({ name: '', description: '', permissions: [] as string[] })

async function fetchData() {
  loading.value = true
  try {
    const res = await getRoles()
    tableData.value = res.data.data ?? []
  } finally {
    loading.value = false
  }
}

async function fetchPermissions() {
  const res = await getPermissions()
  allPermissions.value = res.data.data ?? []
}

function openCreate() {
  isEdit.value = false
  form.value = { name: '', description: '', permissions: [] }
  dialogVisible.value = true
}

function openEdit(row: RoleItem) {
  isEdit.value = true
  editId.value = row.id
  form.value = {
    name: row.name,
    description: row.description,
    permissions: [...row.permissions],
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  submitting.value = true
  try {
    if (isEdit.value) {
      await updateRole(editId.value, { description: form.value.description, permissions: form.value.permissions })
      ElMessage.success('更新成功')
    } else {
      if (!form.value.name) {
        ElMessage.warning('角色名不能为空')
        return
      }
      await createRole({ name: form.value.name, description: form.value.description, permissions: form.value.permissions })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: number) {
  await deleteRole(id)
  ElMessage.success('已删除')
  fetchData()
}

onMounted(() => {
  fetchData()
  fetchPermissions()
})
</script>

<style scoped>
.role-manage-view {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.role-card {
  background: var(--color-bg-card, #ffffff);
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  border-radius: 12px;
  box-shadow: var(--shadow-card, 0 1px 3px rgba(15, 23, 42, 0.05));
  overflow: hidden;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 22px;
  border-bottom: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.06));
}
.page-title {
  font-size: 17px;
  font-weight: 700;
  margin: 0;
  color: var(--color-text-primary, #111827);
}
.mr-1 {
  margin-right: 4px;
}
.perm-desc {
  color: var(--color-text-muted, #94a3b8);
  font-size: 12px;
}
</style>
