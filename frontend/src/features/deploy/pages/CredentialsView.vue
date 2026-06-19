<template>
  <div class="cred-page">
    <!-- Hero 区域 -->
    <section class="cred-hero">
      <div class="cred-hero__main">
        <div class="hero-mark">
          <span class="hero-mark__icon"><el-icon><Key /></el-icon></span>
          <div>
            <p class="eyebrow">Credential Vault</p>
            <h1>SSH 凭据管理</h1>
          </div>
        </div>
        <p>集中管理 SSH 登录凭据，支持密码与私钥两种认证方式，可关联至多台部署服务器，统一维护访问凭证安全。</p>
        <div class="hero-chips">
          <span class="hero-chip"><el-icon><Lock /></el-icon> 加密存储</span>
          <span class="hero-chip"><el-icon><Connection /></el-icon> 多服务器关联</span>
          <span class="hero-chip"><el-icon><Document /></el-icon> 密码 / 私钥双模式</span>
        </div>
      </div>
      <div class="cred-hero__side">
        <div class="hero-stat">
          <span class="hero-stat__icon hero-stat__icon--primary"><el-icon><Key /></el-icon></span>
          <div>
            <span class="hero-stat__label">凭据总数</span>
            <strong class="hero-stat__value">{{ total }}</strong>
          </div>
        </div>
        <div class="hero-stat">
          <span class="hero-stat__icon hero-stat__icon--warning"><el-icon><Lock /></el-icon></span>
          <div>
            <span class="hero-stat__label">密码认证</span>
            <strong class="hero-stat__value">{{ passwordCount }}</strong>
          </div>
        </div>
        <div class="hero-stat">
          <span class="hero-stat__icon hero-stat__icon--success"><el-icon><Document /></el-icon></span>
          <div>
            <span class="hero-stat__label">私钥认证</span>
            <strong class="hero-stat__value">{{ keyCount }}</strong>
          </div>
        </div>
      </div>
    </section>

    <!-- 工具栏 -->
    <div class="page-toolbar">
      <div class="page-toolbar-left">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索名称 / 用户名"
          clearable
          class="toolbar-input"
          @clear="handleSearch"
          @keyup.enter="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="searchAuthType" placeholder="认证方式" clearable class="toolbar-select" @change="handleSearch">
          <el-option label="密码" value="password" />
          <el-option label="私钥" value="key" />
        </el-select>
        <el-button @click="handleSearch"><el-icon><Search /></el-icon><span>搜索</span></el-button>
        <el-button @click="handleReset"><el-icon><RefreshRight /></el-icon><span>重置</span></el-button>
      </div>
      <div class="page-toolbar-right">
        <el-button
          v-if="selectedIds.length > 0"
          type="danger"
          plain
          @click="handleBatchDelete"
        >
          <el-icon><Delete /></el-icon>
          <span>批量删除 ({{ selectedIds.length }})</span>
        </el-button>
        <el-button @click="loadCredentials"><el-icon><Refresh /></el-icon><span>刷新</span></el-button>
        <el-button type="primary" @click="openDialog(null)"><el-icon><Plus /></el-icon><span>新增凭据</span></el-button>
      </div>
    </div>

    <!-- 凭据列表 -->
    <el-card class="page-card" shadow="never">
      <el-table
        :data="credentials"
        v-loading="loading"
        stripe
        border
        class="cred-table"
        @selection-change="handleSelectionChange"
        empty-text="暂无凭据，点击「新增凭据」添加第一条"
      >
        <el-table-column type="selection" width="42" align="center" />
        <el-table-column prop="name" label="凭据名称" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="cred-name-cell">
              <el-icon class="cred-name-cell__icon" :class="row.auth_type === 'password' ? 'type-pwd' : 'type-key'">
                <component :is="row.auth_type === 'password' ? Lock : Key" />
              </el-icon>
              <span class="cred-name-cell__text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="username" label="用户名" width="130">
          <template #default="{ row }">
            <el-tag size="small" effect="plain" type="info">{{ row.username }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="认证方式" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="row.auth_type === 'password' ? 'warning' : 'success'" size="small" effect="light">
              {{ row.auth_type === 'password' ? '密码' : '私钥' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="关联服务器" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.server_count > 0" type="primary" size="small" effect="plain">
              {{ row.server_count }} 台
            </el-tag>
            <span v-else class="cell-empty">未关联</span>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="cell-text">{{ row.remark || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">
            <span class="cell-text">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">
            <span class="cell-text">{{ formatTime(row.updated_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <div class="cell-actions">
              <el-button link type="primary" size="small" @click="openDetail(row)">详情</el-button>
              <el-button link type="primary" size="small" @click="openDialog(row)">编辑</el-button>
              <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrap" v-if="total > 0">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="loadCredentials"
          @size-change="handleSizeChange"
        />
      </div>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? '编辑凭据' : '新增 SSH 凭据'"
      width="580px"
      destroy-on-close
      @closed="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px" class="cred-form">
        <el-divider content-position="left">基本信息</el-divider>
        <el-form-item label="凭据名称" prop="name">
          <el-input v-model="form.name" placeholder="如：root-password-01" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="认证方式" prop="auth_type">
          <el-radio-group v-model="form.auth_type">
            <el-radio-button value="password">密码认证</el-radio-button>
            <el-radio-button value="key">私钥认证</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-divider content-position="left">凭证内容</el-divider>
        <el-form-item :label="form.auth_type === 'password' ? 'SSH 密码' : '私钥内容'" prop="credential">
          <el-input
            v-if="form.auth_type === 'password'"
            v-model="form.credential"
            type="password"
            show-password
            :placeholder="isEditing ? '留空则不修改当前密码' : '请输入 SSH 密码'"
          />
          <el-input
            v-else
            v-model="form.credential"
            type="textarea"
            :rows="6"
            :placeholder="isEditing ? '留空则不修改当前私钥' : '请粘贴私钥内容（BEGIN RSA PRIVATE KEY ...）'"
          />
          <div class="form-hint" v-if="isEditing">留空表示不修改现有凭证内容</div>
        </el-form-item>

        <el-divider content-position="left">扩展信息</el-divider>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="可选，用于备注凭据用途、关联服务器等" maxlength="200" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">{{ isEditing ? '保存修改' : '创建凭据' }}</el-button>
      </template>
    </el-dialog>

    <!-- 详情抽屉 -->
    <el-drawer
      v-model="detailVisible"
      title="凭据详情"
      size="420px"
      destroy-on-close
    >
      <template v-if="detailData">
        <div class="detail-section">
          <div class="detail-header">
            <span class="detail-header__icon" :class="detailData.auth_type === 'password' ? 'type-pwd' : 'type-key'">
              <el-icon><component :is="detailData.auth_type === 'password' ? Lock : Key" /></el-icon>
            </span>
            <div>
              <h3 class="detail-header__name">{{ detailData.name }}</h3>
              <el-tag :type="detailData.auth_type === 'password' ? 'warning' : 'success'" size="small" effect="light">
                {{ detailData.auth_type === 'password' ? '密码认证' : '私钥认证' }}
              </el-tag>
            </div>
          </div>
        </div>

        <el-descriptions :column="1" border class="detail-desc">
          <el-descriptions-item label="凭据 ID">{{ detailData.id }}</el-descriptions-item>
          <el-descriptions-item label="用户名">
            <el-tag size="small" effect="plain" type="info">{{ detailData.username }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="认证方式">{{ detailData.auth_type === 'password' ? '密码' : '私钥' }}</el-descriptions-item>
          <el-descriptions-item label="关联服务器">
            <el-tag v-if="detailData.server_count > 0" type="primary" size="small" effect="plain">
              {{ detailData.server_count }} 台
            </el-tag>
            <span v-else class="cell-empty">未关联</span>
          </el-descriptions-item>
          <el-descriptions-item label="备注">{{ detailData.remark || '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(detailData.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatTime(detailData.updated_at) }}</el-descriptions-item>
        </el-descriptions>

        <div class="detail-actions">
          <el-button type="primary" @click="detailVisible = false; openDialog(detailData!)">
            <el-icon><EditPen /></el-icon>编辑
          </el-button>
          <el-button type="danger" plain @click="detailVisible = false; handleDelete(detailData!)">
            <el-icon><Delete /></el-icon>删除
          </el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Connection, Delete, Document, EditPen, Key, Lock, Plus, Refresh, RefreshRight, Search } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'

import {
  batchDeleteSSHCredentials,
  createSSHCredential,
  deleteSSHCredential,
  listSSHCredentials,
  updateSSHCredential,
  type SSHCredentialItem
} from '../api/deploy'

// ── 搜索与分页 ──
const searchKeyword = ref('')
const searchAuthType = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// ── 列表 ──
const credentials = ref<SSHCredentialItem[]>([])
const loading = ref(false)
const selectedIds = ref<number[]>([])

// ── 统计 ──
const passwordCount = computed(() => credentials.value.filter(c => c.auth_type === 'password').length)
const keyCount = computed(() => credentials.value.filter(c => c.auth_type === 'key').length)

// ── 对话框 ──
const dialogVisible = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const isEditing = computed(() => editingId.value !== null)

const formRef = ref<FormInstance>()
const form = reactive({
  name: '',
  username: 'root',
  auth_type: 'password' as 'password' | 'key',
  credential: '',
  remark: ''
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入凭据名称', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  auth_type: [{ required: true, message: '请选择认证方式', trigger: 'change' }],
  credential: [{
    validator: (_rule: any, _value: any, callback: any) => {
      if (isEditing.value) {
        callback()
      } else if (!form.credential) {
        callback(new Error(form.auth_type === 'password' ? '请输入密码' : '请输入私钥'))
      } else {
        callback()
      }
    },
    trigger: 'blur'
  }]
}

// ── 详情抽屉 ──
const detailVisible = ref(false)
const detailData = ref<SSHCredentialItem | null>(null)

// ── 工具函数 ──
function formatTime(s?: string) {
  if (!s) return '-'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleString()
}

// ── 加载列表 ──
async function loadCredentials() {
  loading.value = true
  try {
    const data = await listSSHCredentials({
      page: currentPage.value,
      page_size: pageSize.value,
      keyword: searchKeyword.value || undefined,
      auth_type: searchAuthType.value || undefined
    })
    credentials.value = data.list || []
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  currentPage.value = 1
  loadCredentials()
}

function handleReset() {
  searchKeyword.value = ''
  searchAuthType.value = ''
  currentPage.value = 1
  loadCredentials()
}

function handleSizeChange() {
  currentPage.value = 1
  loadCredentials()
}

function handleSelectionChange(rows: SSHCredentialItem[]) {
  selectedIds.value = rows.map(r => r.id)
}

// ── 对话框操作 ──
function openDialog(row: SSHCredentialItem | null) {
  editingId.value = row ? row.id : null
  if (row) {
    Object.assign(form, {
      name: row.name,
      username: row.username,
      auth_type: row.auth_type,
      credential: '',
      remark: row.remark || ''
    })
  } else {
    Object.assign(form, { name: '', username: 'root', auth_type: 'password', credential: '', remark: '' })
  }
  dialogVisible.value = true
  nextTick(() => formRef.value?.clearValidate())
}

function resetForm() {
  editingId.value = null
  Object.assign(form, { name: '', username: 'root', auth_type: 'password', credential: '', remark: '' })
}

async function handleSave() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    if (isEditing.value) {
      await updateSSHCredential(editingId.value!, {
        name: form.name,
        username: form.username,
        auth_type: form.auth_type,
        credential: form.credential || undefined,
        remark: form.remark || undefined
      })
      ElMessage.success('凭据更新成功')
    } else {
      await createSSHCredential({
        name: form.name,
        username: form.username,
        auth_type: form.auth_type,
        credential: form.credential,
        remark: form.remark
      })
      ElMessage.success('凭据创建成功')
    }
    dialogVisible.value = false
    await loadCredentials()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  } finally {
    saving.value = false
  }
}

// ── 详情 ──
function openDetail(row: SSHCredentialItem) {
  detailData.value = row
  detailVisible.value = true
}

// ── 删除 ──
async function handleDelete(row: SSHCredentialItem) {
  try {
    await ElMessageBox.confirm(
      `确定要删除凭据「${row.name}」吗？${row.server_count > 0 ? `当前有 ${row.server_count} 台服务器关联此凭据，删除后将无法通过该凭据连接。` : ''}`,
      '删除确认',
      { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await deleteSSHCredential(row.id)
    ElMessage.success('凭据已删除')
    await loadCredentials()
  } catch (e: any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

// ── 批量删除 ──
async function handleBatchDelete() {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedIds.value.length} 条凭据吗？删除后关联的服务器将无法通过这些凭据连接。`,
      '批量删除确认',
      { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await batchDeleteSSHCredentials(selectedIds.value)
    ElMessage.success(`已删除 ${selectedIds.value.length} 条凭据`)
    selectedIds.value = []
    await loadCredentials()
  } catch (e: any) {
    ElMessage.error(e?.message || '批量删除失败')
  }
}

onMounted(() => {
  loadCredentials()
})
</script>

<style scoped>
/* ── 页面容器 ── */
.cred-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px;
  background:
    radial-gradient(circle at top right, rgba(59, 130, 246, 0.06), transparent 24%),
    radial-gradient(circle at top left, rgba(250, 204, 21, 0.06), transparent 22%),
    linear-gradient(180deg, #f7f9fc 0%, #f4f7fb 100%);
  min-height: 100%;
}

/* ── Hero 区域 ── */
.cred-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 20px 22px;
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background:
    radial-gradient(circle at top right, rgba(191, 219, 254, 0.18), transparent 28%),
    linear-gradient(180deg, #ffffff, #f9fbff);
  box-shadow: 0 14px 32px rgba(15, 23, 42, 0.06);
}

.cred-hero__main {
  min-width: 0;
  display: grid;
  gap: 12px;
}

.hero-mark {
  display: flex;
  align-items: center;
  gap: 12px;
}

.hero-mark__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.14), rgba(96, 165, 250, 0.2));
  color: #2563eb;
  font-size: 20px;
}

.cred-hero h1 {
  margin: 0;
  color: #1f2937;
  font-size: 18px;
  line-height: 1.35;
}

.cred-hero p {
  max-width: 580px;
  margin: 0;
  font-size: 14px;
  line-height: 1.65;
  color: #4b5563;
}

.eyebrow {
  margin: 0 0 8px;
  font-size: 12px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #2563eb;
}

.hero-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.hero-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-radius: 999px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.86);
  color: #1f2937;
  font-size: 11px;
  font-weight: 600;
}

.cred-hero__side {
  display: grid;
  grid-template-columns: repeat(3, minmax(110px, 1fr));
  gap: 10px;
  min-width: 380px;
  flex-shrink: 0;
}

.hero-stat {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid rgba(15, 23, 42, 0.08);
}

.hero-stat__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  font-size: 16px;
}

.hero-stat__icon--primary {
  background: rgba(59, 130, 246, 0.1);
  color: #2563eb;
}

.hero-stat__icon--warning {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
}

.hero-stat__icon--success {
  background: rgba(16, 185, 129, 0.1);
  color: #059669;
}

.hero-stat__label {
  display: block;
  font-size: 11px;
  color: #6b7280;
  line-height: 1.2;
}

.hero-stat__value {
  display: block;
  font-size: 20px;
  font-weight: 700;
  color: #111827;
  line-height: 1.3;
}

/* ── 工具栏 ── */
.page-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.86);
  backdrop-filter: blur(8px);
}

.page-toolbar-left,
.page-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.toolbar-input {
  width: 220px;
}

.toolbar-select {
  width: 130px;
}

/* ── 表格 ── */
.cred-table {
  border-radius: 8px;
}

.cred-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cred-name-cell__icon {
  font-size: 16px;
  flex-shrink: 0;
}

.cred-name-cell__icon.type-pwd {
  color: #d97706;
}

.cred-name-cell__icon.type-key {
  color: #059669;
}

.cred-name-cell__text {
  font-weight: 500;
  color: #1f2937;
}

.cell-text {
  color: #4b5563;
  font-size: 13px;
}

.cell-empty {
  color: #9ca3af;
  font-size: 13px;
}

.cell-actions {
  display: inline-flex;
  align-items: center;
  gap: 0;
}

.cell-actions .el-button {
  padding: 0 8px;
  height: 24px;
  font-size: 13px;
  font-weight: 500;
  border-radius: 0;
  position: relative;
}

.cell-actions .el-button + .el-button::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 1px;
  height: 14px;
  background: #e5e7eb;
}

/* ── 分页 ── */
.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}

/* ── 表单 ── */
.cred-form :deep(.el-divider) {
  margin: 16px 0 12px;
}

.cred-form :deep(.el-divider__text) {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
}

.form-hint {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 4px;
  line-height: 1.4;
}

/* ── 详情抽屉 ── */
.detail-section {
  margin-bottom: 20px;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.06), rgba(96, 165, 250, 0.1));
  border: 1px solid rgba(59, 130, 246, 0.12);
}

.detail-header__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  font-size: 20px;
}

.detail-header__icon.type-pwd {
  background: rgba(245, 158, 11, 0.12);
  color: #d97706;
}

.detail-header__icon.type-key {
  background: rgba(16, 185, 129, 0.12);
  color: #059669;
}

.detail-header__name {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.detail-desc {
  margin-top: 16px;
}

.detail-desc :deep(.el-descriptions__label) {
  width: 100px;
  font-weight: 500;
  color: #6b7280;
}

.detail-actions {
  display: flex;
  gap: 10px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #f3f4f6;
}

/* ── Dark Mode ── */
html.dark .cred-page {
  background:
    radial-gradient(circle at top right, rgba(59, 130, 246, 0.04), transparent 24%),
    radial-gradient(circle at top left, rgba(250, 204, 21, 0.04), transparent 22%),
    linear-gradient(180deg, #0f172a 0%, #1e293b 100%);
}

html.dark .cred-hero {
  border-color: rgba(255, 255, 255, 0.08);
  background:
    radial-gradient(circle at top right, rgba(59, 130, 246, 0.08), transparent 28%),
    linear-gradient(180deg, #1e293b, #1a2332);
  box-shadow: 0 14px 32px rgba(0, 0, 0, 0.3);
}

html.dark .cred-hero h1 { color: #f1f5f9; }
html.dark .cred-hero p { color: #94a3b8; }
html.dark .hero-chip { background: rgba(255, 255, 255, 0.06); border-color: rgba(255, 255, 255, 0.1); color: #e2e8f0; }
html.dark .hero-stat { background: rgba(255, 255, 255, 0.04); border-color: rgba(255, 255, 255, 0.08); }
html.dark .hero-stat__label { color: #94a3b8; }
html.dark .hero-stat__value { color: #f1f5f9; }
html.dark .page-toolbar { background: rgba(30, 41, 59, 0.86); border-color: rgba(255, 255, 255, 0.08); }
html.dark .cred-name-cell__text { color: #e2e8f0; }
html.dark .cell-text { color: #94a3b8; }
html.dark .detail-header { background: linear-gradient(135deg, rgba(59, 130, 246, 0.08), rgba(96, 165, 250, 0.12)); border-color: rgba(59, 130, 246, 0.15); }
html.dark .detail-header__name { color: #f1f5f9; }
html.dark .detail-actions { border-top-color: #334155; }
html.dark .cred-form :deep(.el-divider__text) { color: #cbd5e1; }
</style>
