<template>
  <section class="deploy-config-page">
    <el-tabs v-model="activeTab" type="border-card">
      <!-- 部署流程配置 -->
      <el-tab-pane label="部署流程配置" name="configs">
        <div class="section-header">
          <div>
            <h3>部署流程配置</h3>
            <p class="section-desc">管理 K8s 集群部署各阶段的命令模板，支持按操作系统配置不同命令</p>
          </div>
          <div class="os-filter">
            <span class="os-filter-label">操作系统：</span>
            <el-select v-model="selectedOSType" @change="loadConfigs" style="width: 160px">
              <el-option label="全部" value="" />
              <el-option v-for="os in supportedOSTypes" :key="os" :label="osTypeLabel(os)" :value="os" />
            </el-select>
          </div>
        </div>

        <el-collapse v-model="activeSteps">
          <el-collapse-item
            v-for="config in configs"
            :key="config.id"
            :name="config.step_key"
          >
            <template #title>
              <div class="step-header">
                <el-tag :type="config.enabled ? 'success' : 'info'" size="small" class="step-tag">
                  {{ config.enabled ? '启用' : '禁用' }}
                </el-tag>
                <el-tag type="warning" size="small" class="step-tag">{{ osTypeLabel(config.os_type) }}</el-tag>
                <span class="step-name">{{ config.step_name }}</span>
                <span class="step-key">{{ config.step_key }}</span>
                <span class="step-meta">超时: {{ config.timeout_seconds }}s | 重试: {{ config.retry_count }}次</span>
              </div>
            </template>

            <div class="config-detail">
              <div class="config-info">
                <p v-if="config.description" class="config-desc">{{ config.description }}</p>
              </div>

              <div class="command-section">
                <div class="command-header">
                  <span>命令模板</span>
                  <div class="command-actions">
                    <el-button size="small" @click="openEditDialog(config)">编辑</el-button>
                    <el-button size="small" @click="openVersionHistory(config)">版本历史</el-button>
                  </div>
                </div>
                <pre class="command-content">{{ config.command_template }}</pre>
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </el-tab-pane>

      <!-- 仓库配置管理 -->
      <el-tab-pane label="仓库配置管理" name="repositories">
        <div class="section-header">
          <h3>仓库配置管理</h3>
          <p class="section-desc">管理镜像仓库、yum/apt 仓库配置，支持私有仓库和公共仓库切换</p>
        </div>

        <div class="repo-toolbar">
          <el-radio-group v-model="repoTypeFilter" @change="loadRepositories">
            <el-radio-button label="">全部</el-radio-button>
            <el-radio-button label="container_mirror">容器镜像加速</el-radio-button>
            <el-radio-button label="registry">镜像仓库</el-radio-button>
            <el-radio-button label="yum">YUM 仓库</el-radio-button>
            <el-radio-button label="apt">APT 仓库</el-radio-button>
          </el-radio-group>
          <el-button type="primary" @click="openRepoDialog()">新增仓库</el-button>
        </div>

        <el-table :data="repositories" stripe>
          <el-table-column prop="name" label="仓库名称" min-width="160" />
          <el-table-column prop="repo_type" label="类型" width="120">
            <template #default="{ row }">
              <el-tag size="small">{{ repoTypeLabel(row.repo_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="url" label="仓库地址" min-width="280" show-overflow-tooltip />
          <el-table-column prop="mirror_of" label="镜像源" width="160" show-overflow-tooltip />
          <el-table-column prop="priority" label="优先级" width="80" />
          <el-table-column prop="enabled" label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
                {{ row.enabled ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="is_default" label="默认" width="70">
            <template #default="{ row }">
              <el-tag v-if="row.is_default" type="warning" size="small">默认</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openRepoDialog(row)">编辑</el-button>
              <el-button link type="danger" @click="handleDeleteRepo(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 编辑命令对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑命令模板" width="750px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item label="步骤名称">
          <el-input :model-value="editingConfig?.step_name" disabled />
        </el-form-item>
        <el-form-item label="命令模板">
          <el-input
            v-model="editForm.command_template"
            type="textarea"
            :rows="16"
            placeholder="输入命令模板，支持 {{.Variable}} 变量"
            class="code-editor"
          />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="editForm.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="超时时间">
          <el-input-number v-model="editForm.timeout_seconds" :min="10" :max="3600" />
          <span class="form-suffix">秒</span>
        </el-form-item>
        <el-form-item label="重试次数">
          <el-input-number v-model="editForm.retry_count" :min="0" :max="5" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="editForm.enabled" />
        </el-form-item>
        <el-form-item label="变更说明">
          <el-input v-model="editForm.change_summary" placeholder="简要描述本次修改内容" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSaveConfig">保存</el-button>
      </template>
    </el-dialog>

    <!-- 版本历史对话框 -->
    <el-dialog v-model="versionDialogVisible" title="配置版本历史" width="700px" destroy-on-close>
      <el-table :data="versions" stripe max-height="400">
        <el-table-column prop="changed_at" label="修改时间" width="180" />
        <el-table-column prop="change_type" label="变更类型" width="90">
          <template #default="{ row }">
            <el-tag :type="row.change_type === 'create' ? 'success' : 'warning'" size="small">
              {{ row.change_type === 'create' ? '创建' : '修改' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="change_summary" label="变更摘要" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button link type="primary" @click="viewVersionDetail(row)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 版本详情对话框 -->
    <el-dialog v-model="versionDetailVisible" title="版本详情" width="700px" destroy-on-close>
      <pre class="command-content">{{ versionDetailContent }}</pre>
    </el-dialog>

    <!-- 仓库编辑对话框 -->
    <el-dialog v-model="repoDialogVisible" :title="editingRepo ? '编辑仓库' : '新增仓库'" width="550px" destroy-on-close>
      <el-form :model="repoForm" label-width="100px">
        <el-form-item label="仓库类型" required>
          <el-select v-model="repoForm.repo_type" :disabled="!!editingRepo">
            <el-option label="容器镜像加速" value="container_mirror" />
            <el-option label="镜像仓库" value="registry" />
            <el-option label="YUM 仓库" value="yum" />
            <el-option label="APT 仓库" value="apt" />
          </el-select>
        </el-form-item>
        <el-form-item label="仓库名称" required>
          <el-input v-model="repoForm.name" />
        </el-form-item>
        <el-form-item label="仓库地址" required>
          <el-input v-model="repoForm.url" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="repoForm.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="认证类型">
          <el-select v-model="repoForm.auth_type">
            <el-option label="无需认证" value="none" />
            <el-option label="Basic 认证" value="basic" />
            <el-option label="Token 认证" value="token" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="repoForm.priority" :min="1" :max="999" />
          <span class="form-suffix">数值越小优先级越高</span>
        </el-form-item>
        <el-form-item label="镜像源">
          <el-input v-model="repoForm.mirror_of" placeholder="该仓库镜像的原始地址（可选）" />
        </el-form-item>
        <el-form-item label="默认仓库">
          <el-switch v-model="repoForm.is_default" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="repoDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingRepo" @click="handleSaveRepo">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listDeployConfigs,
  listSupportedOSTypes,
  updateDeployConfig,
  getDeployConfigVersions,
  listRepositories,
  createRepository,
  updateRepository,
  deleteRepository,
  type DeployConfigItem,
  type DeployConfigVersion,
  type RepositoryItem
} from '../api/deploy'

const activeTab = ref('configs')
const activeSteps = ref<string[]>([])
const configs = ref<DeployConfigItem[]>([])
const repositories = ref<RepositoryItem[]>([])
const repoTypeFilter = ref('')
const selectedOSType = ref('')
const supportedOSTypes = ref<string[]>([])

function osTypeLabel(os: string) {
  const map: Record<string, string> = { ubuntu: 'Ubuntu/Debian', centos: 'CentOS/Rocky', rocky: 'Rocky Linux', debian: 'Debian' }
  return map[os] || os
}
const saving = ref(false)
const savingRepo = ref(false)

// 编辑命令相关
const editDialogVisible = ref(false)
const editingConfig = ref<DeployConfigItem | null>(null)
const editForm = ref({
  command_template: '',
  description: '',
  timeout_seconds: 600,
  retry_count: 0,
  enabled: true,
  change_summary: ''
})

// 版本历史相关
const versionDialogVisible = ref(false)
const versions = ref<DeployConfigVersion[]>([])
const versionDetailVisible = ref(false)
const versionDetailContent = ref('')

// 仓库编辑相关
const repoDialogVisible = ref(false)
const editingRepo = ref<RepositoryItem | null>(null)
const repoForm = ref({
  repo_type: 'container_mirror',
  name: '',
  url: '',
  description: '',
  auth_type: 'none',
  priority: 100,
  is_default: false,
  mirror_of: ''
})

onMounted(async () => {
  await loadSupportedOSTypes()
  await loadConfigs()
  await loadRepositories()
})

async function loadSupportedOSTypes() {
  try {
    supportedOSTypes.value = await listSupportedOSTypes()
  } catch {
    // 忽略
  }
}

async function loadConfigs() {
  try {
    const params = selectedOSType.value ? { os_type: selectedOSType.value } : undefined
    configs.value = await listDeployConfigs(params)
  } catch {
    ElMessage.error('加载部署配置失败')
  }
}

async function loadRepositories() {
  try {
    repositories.value = await listRepositories(repoTypeFilter.value ? { type: repoTypeFilter.value } : undefined)
  } catch {
    ElMessage.error('加载仓库配置失败')
  }
}

function openEditDialog(config: DeployConfigItem) {
  editingConfig.value = config
  editForm.value = {
    command_template: config.command_template,
    description: config.description || '',
    timeout_seconds: config.timeout_seconds,
    retry_count: config.retry_count,
    enabled: config.enabled,
    change_summary: ''
  }
  editDialogVisible.value = true
}

async function handleSaveConfig() {
  if (!editingConfig.value) return
  if (!editForm.value.command_template.trim()) {
    ElMessage.warning('命令模板不能为空')
    return
  }
  saving.value = true
  try {
    await updateDeployConfig(editingConfig.value.id, editForm.value)
    ElMessage.success('配置已保存')
    editDialogVisible.value = false
    await loadConfigs()
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function openVersionHistory(config: DeployConfigItem) {
  try {
    versions.value = await getDeployConfigVersions(config.id)
    versionDialogVisible.value = true
  } catch {
    ElMessage.error('获取版本历史失败')
  }
}

function viewVersionDetail(version: DeployConfigVersion) {
  versionDetailContent.value = version.command_template
  versionDetailVisible.value = true
}

function openRepoDialog(repo?: RepositoryItem) {
  if (repo) {
    editingRepo.value = repo
    repoForm.value = {
      repo_type: repo.repo_type,
      name: repo.name,
      url: repo.url,
      description: repo.description || '',
      auth_type: repo.auth_type,
      priority: repo.priority,
      is_default: repo.is_default,
      mirror_of: repo.mirror_of || ''
    }
  } else {
    editingRepo.value = null
    repoForm.value = {
      repo_type: 'container_mirror',
      name: '',
      url: '',
      description: '',
      auth_type: 'none',
      priority: 100,
      is_default: false,
      mirror_of: ''
    }
  }
  repoDialogVisible.value = true
}

async function handleSaveRepo() {
  if (!repoForm.value.name.trim() || !repoForm.value.url.trim()) {
    ElMessage.warning('请填写仓库名称和地址')
    return
  }
  savingRepo.value = true
  try {
    if (editingRepo.value) {
      await updateRepository(editingRepo.value.id, repoForm.value)
      ElMessage.success('仓库配置已更新')
    } else {
      await createRepository(repoForm.value)
      ElMessage.success('仓库配置已创建')
    }
    repoDialogVisible.value = false
    await loadRepositories()
  } catch {
    ElMessage.error('保存失败')
  } finally {
    savingRepo.value = false
  }
}

async function handleDeleteRepo(row: RepositoryItem) {
  await ElMessageBox.confirm(`确认删除仓库"${row.name}"？`, '删除确认', { type: 'warning' })
  try {
    await deleteRepository(row.id)
    ElMessage.success('已删除')
    await loadRepositories()
  } catch {
    ElMessage.error('删除失败')
  }
}

function repoTypeLabel(type: string): string {
  const map: Record<string, string> = {
    container_mirror: '容器镜像加速',
    registry: '镜像仓库',
    yum: 'YUM 仓库',
    apt: 'APT 仓库'
  }
  return map[type] || type
}
</script>

<style scoped>
.deploy-config-page {
  padding: 20px;
  background: var(--el-bg-color);
  border-radius: 12px;
  min-height: 600px;
}

.section-header {
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.os-filter {
  display: flex;
  align-items: center;
  gap: 8px;
}

.os-filter-label {
  font-size: 13px;
  color: var(--color-text-regular);
  white-space: nowrap;
}

.section-header h3 {
  margin: 0 0 8px 0;
  font-size: 16px;
  color: var(--color-text-primary);
}

.section-desc {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-regular);
}

.step-header {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.step-tag {
  flex-shrink: 0;
}

.step-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.step-key {
  font-size: 12px;
  color: var(--color-text-secondary);
  font-family: monospace;
}

.step-meta {
  margin-left: auto;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.config-detail {
  padding: 12px 0;
}

.config-desc {
  margin: 0 0 16px 0;
  font-size: 13px;
  color: var(--color-text-regular);
}

.command-section {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.command-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.command-actions {
  display: flex;
  gap: 8px;
}

.command-content {
  margin: 0;
  padding: 16px;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--el-fill-color-darker);
  color: var(--color-text-primary);
  max-height: 400px;
  overflow-y: auto;
}

.repo-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.code-editor :deep(textarea) {
  font-family: monospace;
  font-size: 13px;
  line-height: 1.5;
}

.form-suffix {
  margin-left: 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
}
</style>
