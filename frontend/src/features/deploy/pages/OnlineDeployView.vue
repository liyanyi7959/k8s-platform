<template>
  <section class="online-deploy-view">
    <header class="page-header">
      <div>
        <h1>在线部署</h1>
        <p>管理部署服务器、SSH 凭证与 Kubernetes 部署计划。</p>
      </div>
      <div class="header-actions">
        <el-button @click="openCredentialDialog">新增凭证</el-button>
        <el-button @click="openServerDialog">新增服务器</el-button>
        <el-button type="primary" @click="openPlanDialog">新建部署计划</el-button>
      </div>
    </header>

    <el-row :gutter="16">
      <el-col :span="8">
        <el-card class="summary-card" shadow="never">
          <div class="summary-value">{{ serverTotal }}</div>
          <div class="summary-label">服务器</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="summary-card" shadow="never">
          <div class="summary-value">{{ credentialTotal }}</div>
          <div class="summary-label">SSH 凭证</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="summary-card" shadow="never">
          <div class="summary-value">{{ planTotal }}</div>
          <div class="summary-label">部署计划</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="page-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>部署服务器</span>
          <el-button size="small" @click="loadServers">刷新</el-button>
        </div>
      </template>
      <el-table v-loading="serverLoading" :data="servers" border class="page-table">
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="ip" label="IP" min-width="140" />
        <el-table-column prop="ssh_port" label="SSH 端口" width="110" />
        <el-table-column prop="user" label="用户" width="110" />
        <el-table-column prop="auth_type" label="认证方式" width="110" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column label="操作" width="190" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleTestServer(row)">测试</el-button>
            <el-button link type="danger" @click="handleDeleteServer(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="page-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>SSH 凭证</span>
          <el-button size="small" @click="loadCredentials">刷新</el-button>
        </div>
      </template>
      <el-table v-loading="credentialLoading" :data="credentials" border class="page-table">
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="username" label="用户名" width="120" />
        <el-table-column prop="auth_type" label="认证方式" width="120" />
        <el-table-column prop="remark" label="备注" min-width="180" />
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" @click="handleDeleteCredential(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="page-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>部署计划</span>
          <el-button size="small" @click="loadPlans">刷新</el-button>
        </div>
      </template>
      <el-table v-loading="planLoading" :data="plans" border class="page-table">
        <el-table-column prop="name" label="计划名称" min-width="160" />
        <el-table-column prop="cluster_name" label="集群名称" min-width="160" />
        <el-table-column prop="k8s_version" label="K8s 版本" width="130" />
        <el-table-column prop="cni_type" label="CNI" width="110" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :disabled="!canExecute(row)" @click="handleExecutePlan(row)">执行</el-button>
            <el-button link :disabled="row.status !== 'running'" @click="handleCancelPlan(row)">取消</el-button>
            <el-button link type="danger" :disabled="row.status === 'running' || row.status === 'success'" @click="handleDeletePlan(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="serverDialogVisible" title="新增部署服务器" width="520px" destroy-on-close>
      <el-form :model="serverForm" label-width="96px">
        <el-form-item label="名称" required><el-input v-model="serverForm.name" /></el-form-item>
        <el-form-item label="IP" required><el-input v-model="serverForm.ip" /></el-form-item>
        <el-form-item label="SSH 端口" required><el-input-number v-model="serverForm.ssh_port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="用户" required><el-input v-model="serverForm.user" /></el-form-item>
        <el-form-item label="认证方式" required>
          <el-select v-model="serverForm.auth_type">
            <el-option label="密码" value="password" />
            <el-option label="私钥" value="key" />
          </el-select>
        </el-form-item>
        <el-form-item label="凭证" required>
          <el-input v-model="serverForm.credential" :type="serverForm.auth_type === 'password' ? 'password' : 'textarea'" :rows="4" show-password />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="serverForm.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="serverDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingServer" @click="handleCreateServer">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="credentialDialogVisible" title="新增 SSH 凭证" width="520px" destroy-on-close>
      <el-form :model="credentialForm" label-width="96px">
        <el-form-item label="名称" required><el-input v-model="credentialForm.name" /></el-form-item>
        <el-form-item label="用户名" required><el-input v-model="credentialForm.username" /></el-form-item>
        <el-form-item label="认证方式" required>
          <el-select v-model="credentialForm.auth_type">
            <el-option label="密码" value="password" />
            <el-option label="私钥" value="key" />
          </el-select>
        </el-form-item>
        <el-form-item label="凭证" required>
          <el-input v-model="credentialForm.credential" :type="credentialForm.auth_type === 'password' ? 'password' : 'textarea'" :rows="4" show-password />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="credentialForm.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="credentialDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingCredential" @click="handleCreateCredential">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="planDialogVisible" title="新建部署计划" width="720px" destroy-on-close>
      <el-form :model="planForm" label-width="110px">
        <el-form-item label="计划名称" required><el-input v-model="planForm.name" /></el-form-item>
        <el-form-item label="集群名称" required><el-input v-model="planForm.cluster_name" /></el-form-item>
        <el-form-item label="K8s 版本" required><el-input v-model="planForm.k8s_version" placeholder="v1.29.0" /></el-form-item>
        <el-form-item label="Pod 网段" required><el-input v-model="planForm.pod_cidr" /></el-form-item>
        <el-form-item label="Service 网段" required><el-input v-model="planForm.svc_cidr" /></el-form-item>
        <el-form-item label="CNI" required>
          <el-select v-model="planForm.cni_type">
            <el-option label="Flannel" value="flannel" />
            <el-option label="Calico" value="calico" />
            <el-option label="Cilium" value="cilium" />
          </el-select>
        </el-form-item>
        <el-form-item label="部署节点" required>
          <el-table :data="planForm.nodes" border class="page-table">
            <el-table-column label="服务器" min-width="220">
              <template #default="{ row }">
                <el-select v-model="row.server_id" filterable placeholder="选择服务器">
                  <el-option v-for="server in servers" :key="server.id" :label="`${server.name} (${server.ip})`" :value="server.id" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="角色" width="140">
              <template #default="{ row }">
                <el-select v-model="row.role">
                  <el-option label="Master" value="master" />
                  <el-option label="Worker" value="worker" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="90">
              <template #default="{ $index }">
                <el-button link type="danger" @click="removePlanNode($index)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button class="add-node-btn" @click="addPlanNode">添加节点</el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="planDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingPlan" @click="handleCreatePlan">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  cancelDeployPlan,
  createDeployPlan,
  createDeployServer,
  createSSHCredential,
  deleteDeployPlan,
  deleteDeployServer,
  deleteSSHCredential,
  executeDeployPlan,
  listDeployPlans,
  listDeployServers,
  listSSHCredentials,
  testDeployServerSSH,
  type CreateDeployPlanReq,
  type DeployPlanItem,
  type DeployServerItem,
  type SSHCredentialItem
} from '@/features/deploy/api/deploy'

const servers = ref<DeployServerItem[]>([])
const credentials = ref<SSHCredentialItem[]>([])
const plans = ref<DeployPlanItem[]>([])
const serverTotal = ref(0)
const credentialTotal = ref(0)
const planTotal = ref(0)
const serverLoading = ref(false)
const credentialLoading = ref(false)
const planLoading = ref(false)
const savingServer = ref(false)
const savingCredential = ref(false)
const savingPlan = ref(false)
const serverDialogVisible = ref(false)
const credentialDialogVisible = ref(false)
const planDialogVisible = ref(false)

const serverForm = reactive({ name: '', ip: '', ssh_port: 22, user: 'root', auth_type: 'password' as 'password' | 'key', credential: '', remark: '' })
const credentialForm = reactive({ name: '', username: 'root', auth_type: 'password' as 'password' | 'key', credential: '', remark: '' })
const planForm = reactive<CreateDeployPlanReq>({
  name: '',
  cluster_name: '',
  k8s_version: 'v1.29.0',
  pod_cidr: '10.244.0.0/16',
  svc_cidr: '10.96.0.0/12',
  cni_type: 'flannel',
  nodes: []
})

async function loadServers() {
  serverLoading.value = true
  try {
    const data = await listDeployServers({ page: 1, page_size: 20 })
    servers.value = data.list || []
    serverTotal.value = data.total || 0
  } finally {
    serverLoading.value = false
  }
}

async function loadCredentials() {
  credentialLoading.value = true
  try {
    const data = await listSSHCredentials({ page: 1, page_size: 20 })
    credentials.value = data.list || []
    credentialTotal.value = data.total || 0
  } finally {
    credentialLoading.value = false
  }
}

async function loadPlans() {
  planLoading.value = true
  try {
    const data = await listDeployPlans({ page: 1, page_size: 20 })
    plans.value = data.list || []
    planTotal.value = data.total || 0
  } finally {
    planLoading.value = false
  }
}

function openServerDialog() {
  Object.assign(serverForm, { name: '', ip: '', ssh_port: 22, user: 'root', auth_type: 'password', credential: '', remark: '' })
  serverDialogVisible.value = true
}

function openCredentialDialog() {
  Object.assign(credentialForm, { name: '', username: 'root', auth_type: 'password', credential: '', remark: '' })
  credentialDialogVisible.value = true
}

function openPlanDialog() {
  Object.assign(planForm, {
    name: '',
    cluster_name: '',
    k8s_version: 'v1.29.0',
    pod_cidr: '10.244.0.0/16',
    svc_cidr: '10.96.0.0/12',
    cni_type: 'flannel',
    nodes: []
  })
  addPlanNode()
  planDialogVisible.value = true
}

function addPlanNode() {
  planForm.nodes.push({ server_id: 0, role: planForm.nodes.length === 0 ? 'master' : 'worker', sort_order: planForm.nodes.length })
}

function removePlanNode(index: number) {
  planForm.nodes.splice(index, 1)
  planForm.nodes.forEach((node, i) => {
    node.sort_order = i
  })
}

async function handleCreateServer() {
  savingServer.value = true
  try {
    await createDeployServer(serverForm)
    ElMessage.success('服务器已创建')
    serverDialogVisible.value = false
    await loadServers()
  } finally {
    savingServer.value = false
  }
}

async function handleCreateCredential() {
  savingCredential.value = true
  try {
    await createSSHCredential(credentialForm)
    ElMessage.success('凭证已创建')
    credentialDialogVisible.value = false
    await loadCredentials()
  } finally {
    savingCredential.value = false
  }
}

async function handleCreatePlan() {
  savingPlan.value = true
  try {
    await createDeployPlan({ ...planForm, nodes: planForm.nodes.map((node, index) => ({ ...node, sort_order: index })) })
    ElMessage.success('部署计划已创建')
    planDialogVisible.value = false
    await loadPlans()
  } finally {
    savingPlan.value = false
  }
}

async function handleTestServer(row: DeployServerItem) {
  const data = await testDeployServerSSH(row.id)
  ElMessage.success(data.message || '测试成功')
  await loadServers()
}

async function handleDeleteServer(row: DeployServerItem) {
  await ElMessageBox.confirm(`确认删除服务器“${row.name}”？`, '删除服务器', { type: 'warning' })
  await deleteDeployServer(row.id)
  ElMessage.success('服务器已删除')
  await loadServers()
}

async function handleDeleteCredential(row: SSHCredentialItem) {
  await ElMessageBox.confirm(`确认删除凭证“${row.name}”？`, '删除凭证', { type: 'warning' })
  await deleteSSHCredential(row.id)
  ElMessage.success('凭证已删除')
  await loadCredentials()
}

function canExecute(row: DeployPlanItem) {
  return row.status === 'draft' || row.status === 'failed' || row.status === 'cancelled'
}

async function handleExecutePlan(row: DeployPlanItem) {
  await ElMessageBox.confirm(`确认执行部署计划“${row.name}”？`, '执行部署', { type: 'warning' })
  const data = await executeDeployPlan(row.id)
  ElMessage.success(`部署任务已创建：#${data.task_id}`)
  await loadPlans()
}

async function handleCancelPlan(row: DeployPlanItem) {
  await ElMessageBox.confirm(`确认取消部署计划“${row.name}”？`, '取消部署', { type: 'warning' })
  await cancelDeployPlan(row.id)
  ElMessage.success('部署计划已取消')
  await loadPlans()
}

async function handleDeletePlan(row: DeployPlanItem) {
  await ElMessageBox.confirm(`确认删除部署计划“${row.name}”？`, '删除部署计划', { type: 'warning' })
  await deleteDeployPlan(row.id)
  ElMessage.success('部署计划已删除')
  await loadPlans()
}

onMounted(() => {
  loadServers()
  loadCredentials()
  loadPlans()
})
</script>

<style scoped>
.online-deploy-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header,
.header-actions,
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.page-header h1 {
  margin: 0;
  font-size: 22px;
  color: var(--color-text-primary);
}

.page-header p {
  margin: 6px 0 0;
  color: var(--color-text-secondary);
}

.summary-card,
.page-card {
  border-radius: 16px;
}

.summary-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.summary-label {
  margin-top: 6px;
  color: var(--color-text-secondary);
}

.page-table {
  width: 100%;
}

.add-node-btn {
  margin-top: 10px;
}
</style>
