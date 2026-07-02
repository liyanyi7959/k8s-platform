<template>
  <section class="online-deploy-view">
    <header class="page-header">
      <div>
        <h1>在线部署</h1>
        <p>管理部署服务器与 Kubernetes 部署计划。</p>
      </div>
      <div class="header-actions">
        <el-button @click="openServerDialog"><el-icon><Plus /></el-icon><span>新增服务器</span></el-button>
        <el-button type="primary" @click="openPlanDialog"><el-icon><CirclePlus /></el-icon><span>新建部署计划</span></el-button>
      </div>
    </header>

    <el-row :gutter="16">
      <el-col :span="12">
        <el-card class="summary-card" shadow="never">
          <div class="summary-value">{{ serverTotal }}</div>
          <div class="summary-label">服务器</div>
        </el-card>
      </el-col>
      <el-col :span="12">
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
          <el-button size="small" @click="loadServers"><el-icon><Refresh /></el-icon><span>刷新</span></el-button>
        </div>
      </template>
      <el-table v-loading="serverLoading" :data="servers" border class="page-table" empty-text="暂无服务器，请点击「新增服务器」添加">
        <el-table-column prop="name" label="名称" min-width="160" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" min-width="140" />
        <el-table-column prop="ssh_port" label="SSH 端口" width="100" align="center" />
        <el-table-column prop="user" label="用户" width="100" />
        <el-table-column label="认证方式" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.auth_type === 'password' ? 'warning' : 'success'" size="small">
              {{ row.auth_type === 'password' ? '密码' : '私钥' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="凭据来源" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.credential_id" type="info" size="small">已有凭据</el-tag>
            <el-tag v-else type="info" size="small" effect="plain">手动输入</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作系统" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ row.os || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="硬件信息" width="160">
          <template #default="{ row }">
            <span v-if="row.cpu_cores">{{ row.cpu_cores }}核 / {{ row.memory_mb ? Math.round(row.memory_mb / 1024) + 'GB' : '-' }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">
            <span>{{ formatTime(row.updated_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <div class="k8s-act-group">
              <ActionIconButton :icon="EditPen" tooltip="编辑" variant="edit" @click="handleEditServer(row)" />
              <ActionIconButton :icon="Connection" tooltip="测试连接" variant="success" @click="handleTestServer(row)" />
              <ActionIconButton :icon="Delete" tooltip="删除" variant="danger" @click="handleDeleteServer(row)" />
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="page-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>部署计划</span>
          <el-button size="small" @click="loadPlans"><el-icon><Refresh /></el-icon><span>刷新</span></el-button>
        </div>
      </template>
      <el-table v-loading="planLoading" :data="plans" border class="page-table" empty-text="暂无部署计划，请点击「新建部署计划」创建">
        <el-table-column prop="name" label="计划名称" min-width="160" show-overflow-tooltip />
        <el-table-column prop="cluster_name" label="集群名称" min-width="160" show-overflow-tooltip />
        <el-table-column prop="k8s_version" label="K8s 版本" width="110" align="center" />
        <el-table-column prop="cni_type" label="CNI" width="100" align="center" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="planStatusType(row.status)" size="small">{{ planStatusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">
            <span>{{ formatTime(row.updated_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template #default="{ row }">
            <div class="k8s-act-group">
              <ActionIconButton :icon="View" tooltip="模拟运行" variant="info" @click="handleDryRun(row)" />
              <ActionIconButton :icon="CaretRight" tooltip="执行" variant="success" :disabled="!canExecute(row)" @click="handleExecutePlan(row)" />
              <ActionIconButton :icon="Close" tooltip="取消" variant="warn" :disabled="row.status !== 'running'" @click="handleCancelPlan(row)" />
              <ActionIconButton :icon="Document" tooltip="日志" variant="info" :disabled="!row.task_id" @click="handleViewLogs(row)" />
              <ActionIconButton :icon="Delete" tooltip="删除" variant="danger" :disabled="row.status === 'running' || row.status === 'success'" @click="handleDeletePlan(row)" />
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="serverDialogVisible" :title="editingServerId ? '编辑部署服务器' : '新增部署服务器'" width="520px" destroy-on-close>
      <el-form ref="serverFormRef" :model="serverForm" :rules="serverRules" label-width="96px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="serverForm.name" placeholder="如：master-01" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="IP" prop="ip">
          <el-input v-model="serverForm.ip" placeholder="如：192.168.1.10" />
        </el-form-item>
        <el-form-item label="SSH 端口">
          <el-input-number v-model="serverForm.ssh_port" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
        <el-form-item label="用户" prop="user">
          <el-input v-model="serverForm.user" placeholder="root" />
        </el-form-item>
        <el-form-item label="凭据来源">
          <el-radio-group v-model="serverForm.credential_mode">
            <el-radio value="manual">手动输入</el-radio>
            <el-radio value="credential">选择已有凭据</el-radio>
          </el-radio-group>
        </el-form-item>
        <template v-if="serverForm.credential_mode === 'credential'">
          <el-form-item label="选择凭据" prop="credential_id">
            <el-select v-model="serverForm.credential_id" placeholder="请选择已保存的凭据" style="width: 100%" clearable>
              <el-option v-for="cred in credentials" :key="cred.id" :label="`${cred.name} (${cred.username} / ${cred.auth_type === 'password' ? '密码' : '私钥'})`" :value="cred.id" />
            </el-select>
            <div v-if="credentials.length === 0" class="form-tip">
              暂无已保存的凭据，请先在「SSH 凭据」标签页添加
            </div>
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="认证方式" prop="auth_type">
            <el-select v-model="serverForm.auth_type" style="width: 100%">
              <el-option label="密码" value="password" />
              <el-option label="私钥" value="key" />
            </el-select>
          </el-form-item>
          <el-form-item :label="serverForm.auth_type === 'password' ? '密码' : '私钥'" prop="credential">
            <el-input v-model="serverForm.credential" :type="serverForm.auth_type === 'password' ? 'password' : 'textarea'" :rows="4" :placeholder="editingServerId ? '留空则不修改' : (serverForm.auth_type === 'password' ? '请输入 SSH 密码' : '请粘贴私钥内容（BEGIN RSA PRIVATE KEY ...）')" show-password />
          </el-form-item>
        </template>
        <el-form-item label="备注">
          <el-input v-model="serverForm.remark" type="textarea" :rows="2" placeholder="可选，用于备注服务器用途" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="serverDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingServer" @click="handleSaveServer">{{ editingServerId ? '更新' : '保存' }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="planDialogVisible" title="新建部署计划" width="780px" destroy-on-close>
      <el-steps :active="wizardStep" finish-status="success" align-center style="margin-bottom: 24px">
        <el-step title="基础信息" />
        <el-step title="节点配置" />
        <el-step title="组件选型" />
        <el-step title="确认提交" />
      </el-steps>

      <!-- Step 1: 基础信息 -->
      <el-form v-if="wizardStep === 0" :model="planForm" label-width="110px">
        <el-form-item label="计划名称" required><el-input v-model="planForm.name" placeholder="如：生产集群部署" /></el-form-item>
        <el-form-item label="集群名称" required><el-input v-model="planForm.cluster_name" placeholder="如：prod-k8s-01" /></el-form-item>
        <el-form-item label="K8s 版本" required>
          <el-select v-model="planForm.k8s_version" style="width: 100%">
            <el-option label="v1.31.0 (推荐)" value="v1.31.0" />
            <el-option label="v1.30.0" value="v1.30.0" />
            <el-option label="v1.29.0" value="v1.29.0" />
            <el-option label="v1.28.0" value="v1.28.0" />
          </el-select>
        </el-form-item>
        <el-form-item label="Pod 网段" required><el-input v-model="planForm.pod_cidr" placeholder="10.244.0.0/16" /></el-form-item>
        <el-form-item label="Service 网段" required><el-input v-model="planForm.svc_cidr" placeholder="10.96.0.0/12" /></el-form-item>
        <el-form-item label="CNI" required>
          <el-select v-model="planForm.cni_type" style="width: 100%">
            <el-option label="Flannel (轻量级，适合测试)" value="flannel" />
            <el-option label="Calico (功能丰富，适合生产)" value="calico" />
            <el-option label="Cilium (高性能，eBPF)" value="cilium" />
          </el-select>
        </el-form-item>
      </el-form>

      <!-- Step 2: 节点配置 -->
      <div v-if="wizardStep === 1">
        <el-alert type="info" :closable="false" style="margin-bottom: 16px">
          Master 节点数量必须为 1 或 3（奇数），Worker 节点至少 1 个。
        </el-alert>
        <el-table :data="planForm.nodes" border class="page-table">
          <el-table-column label="服务器" min-width="220">
            <template #default="{ row }">
              <el-select v-model="row.server_id" filterable placeholder="选择服务器">
                <el-option v-for="server in availableServers" :key="server.id" :label="`${server.name} (${server.ip})`" :value="server.id" />
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
          <el-table-column label="操作" width="90" align="center">
            <template #default="{ $index }">
              <div class="k8s-act-group">
                <ActionIconButton :icon="Delete" tooltip="移除" variant="danger" @click="removePlanNode($index)" />
              </div>
            </template>
          </el-table-column>
        </el-table>
        <el-button class="add-node-btn" @click="addPlanNode"><el-icon><Plus /></el-icon><span>添加节点</span></el-button>
        <div v-if="nodeValidationMsg" style="color: var(--el-color-danger); margin-top: 8px; font-size: 13px">{{ nodeValidationMsg }}</div>
      </div>

      <!-- Step 3: 组件选型 -->
      <div v-if="wizardStep === 2">
        <el-checkbox-group v-model="planForm.addons">
          <el-checkbox label="metrics-server" style="display: block; margin-bottom: 12px">
            <div>
              <div style="font-weight: 500">Metrics Server</div>
              <div style="font-size: 12px; color: var(--el-text-color-secondary)">提供资源指标采集，支持 kubectl top 和 HPA</div>
            </div>
          </el-checkbox>
          <el-checkbox label="ingress-nginx" style="display: block; margin-bottom: 12px">
            <div>
              <div style="font-weight: 500">Ingress-Nginx Controller</div>
              <div style="font-size: 12px; color: var(--el-text-color-secondary)">提供 HTTP/HTTPS 入口流量管理</div>
            </div>
          </el-checkbox>
          <el-checkbox label="local-storage" style="display: block; margin-bottom: 12px">
            <div>
              <div style="font-weight: 500">Local Storage Provisioner</div>
              <div style="font-size: 12px; color: var(--el-text-color-secondary)">提供本地存储动态供给</div>
            </div>
          </el-checkbox>
        </el-checkbox-group>
      </div>

      <!-- Step 4: 确认提交 -->
      <div v-if="wizardStep === 3">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="计划名称">{{ planForm.name }}</el-descriptions-item>
          <el-descriptions-item label="集群名称">{{ planForm.cluster_name }}</el-descriptions-item>
          <el-descriptions-item label="K8s 版本">{{ planForm.k8s_version }}</el-descriptions-item>
          <el-descriptions-item label="CNI">{{ planForm.cni_type }}</el-descriptions-item>
          <el-descriptions-item label="Pod 网段">{{ planForm.pod_cidr }}</el-descriptions-item>
          <el-descriptions-item label="Service 网段">{{ planForm.svc_cidr }}</el-descriptions-item>
          <el-descriptions-item label="节点" :span="2">
            <el-tag v-for="(node, i) in planForm.nodes" :key="i" :type="node.role === 'master' ? 'danger' : 'info'" style="margin-right: 8px">
              {{ getServerLabel(node.server_id) }} ({{ node.role }})
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="附加组件" :span="2">
            <el-tag v-for="addon in planForm.addons" :key="addon" style="margin-right: 8px">{{ addon }}</el-tag>
            <span v-if="!planForm.addons.length" style="color: var(--el-text-color-secondary)">无</span>
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <template #footer>
        <el-button @click="planDialogVisible = false">取消</el-button>
        <el-button v-if="wizardStep > 0" @click="wizardStep--">上一步</el-button>
        <el-button v-if="wizardStep < 3" type="primary" @click="nextWizardStep">下一步</el-button>
        <el-button v-if="wizardStep === 3" type="primary" :loading="savingPlan" @click="handleCreatePlan">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="logDialogVisible" title="部署日志 (实时)" width="700px" destroy-on-close @close="closeLogDialog">
      <div class="log-container">
        <pre v-if="taskLogs.length" class="log-content">{{ taskLogs.join('\n') }}</pre>
        <el-empty v-else description="等待日志..." />
      </div>
      <template #footer>
        <el-button @click="closeLogDialog">关闭</el-button>
      </template>
    </el-dialog>

    <DryRunDialog v-model="dryRunDialogVisible" :plan-id="dryRunPlanId" />
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CaretRight, CirclePlus, Close, Connection, Delete, Document, EditPen, Plus, Refresh, View } from '@element-plus/icons-vue'
import ActionIconButton from '@/shared/components/ActionIconButton.vue'
import DryRunDialog from '../components/DryRunDialog.vue'

import {
  cancelDeployPlan,
  createDeployPlan,
  createDeployServer,
  deleteDeployPlan,
  deleteDeployServer,
  executeDeployPlan,
  getDeployTaskLogs,
  listDeployPlans,
  listDeployServers,
  listSSHCredentials,
  subscribeDeployLogs,
  testDeployServerSSH,
  updateDeployServer,
  type CreateDeployPlanReq,
  type DeployPlanItem,
  type DeployServerItem,
  type SSHCredentialItem
} from '@/features/deploy/api/deploy'

const servers = ref<DeployServerItem[]>([])
const credentials = ref<SSHCredentialItem[]>([])
const plans = ref<DeployPlanItem[]>([])
const serverTotal = ref(0)
const planTotal = ref(0)
const serverLoading = ref(false)
const planLoading = ref(false)
const savingServer = ref(false)
const savingPlan = ref(false)
const serverDialogVisible = ref(false)
const editingServerId = ref<number | null>(null)
const planDialogVisible = ref(false)
const logDialogVisible = ref(false)
const dryRunDialogVisible = ref(false)
const dryRunPlanId = ref<number | null>(null)
const taskLogs = ref<string[]>([])
let currentTaskId = 0

const serverFormRef = ref()
const serverForm = reactive({ name: '', ip: '', ssh_port: 22, user: 'root', auth_type: 'password' as 'password' | 'key', credential_mode: 'manual' as 'manual' | 'credential', credential_id: null as number | null, credential: '', remark: '' })
const serverRules = {
  name: [{ required: true, message: '请输入服务器名称', trigger: 'blur' }],
  ip: [
    { required: true, message: '请输入 IP 地址', trigger: 'blur' },
    { pattern: /^(\d{1,3}\.){3}\d{1,3}$/, message: 'IP 格式不正确', trigger: 'blur' }
  ],
  user: [{ required: true, message: '请输入 SSH 用户', trigger: 'blur' }],
  credential_id: [{
    validator: (_rule: any, value: any, callback: any) => {
      if (serverForm.credential_mode === 'credential' && !value) {
        callback(new Error('请选择已有凭据'))
      } else {
        callback()
      }
    },
    trigger: 'change'
  }],
  credential: [{
    validator: (_rule: any, value: any, callback: any) => {
      if (serverForm.credential_mode !== 'manual') {
        callback()
      } else if (!editingServerId.value && !value) {
        callback(new Error(serverForm.auth_type === 'password' ? '请输入密码' : '请输入私钥'))
      } else {
        callback()
      }
    },
    trigger: 'blur'
  }]
}
const planForm = reactive<CreateDeployPlanReq & { addons: string[] }>({
  name: '',
  cluster_name: '',
  k8s_version: 'v1.31.0',
  pod_cidr: '10.244.0.0/16',
  svc_cidr: '10.96.0.0/12',
  cni_type: 'flannel',
  nodes: [],
  addons: []
})

const wizardStep = ref(0)
const nodeValidationMsg = ref('')

// 可用服务器列表（排除已选中的）
const availableServers = computed(() => {
  const selectedIds = new Set(planForm.nodes.map(n => n.server_id).filter(id => id > 0))
  return servers.value.filter(s => !selectedIds.has(s.id))
})

function getServerLabel(serverId: number) {
  const server = servers.value.find(s => s.id === serverId)
  return server ? `${server.name} (${server.ip})` : '未知'
}

function formatTime(t: string) {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return t
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function statusType(s: string) {
  const map: Record<string, string> = { available: 'success', unavailable: 'danger', registered: 'info', checking: 'warning', failed: 'danger' }
  return map[s] || 'info'
}

function statusLabel(s: string) {
  const map: Record<string, string> = { available: '可用', unavailable: '不可用', registered: '待检查', checking: '检查中', failed: '失败' }
  return map[s] || s
}

function planStatusType(s: string) {
  const map: Record<string, string> = { draft: 'info', running: 'warning', success: 'success', failed: 'danger', cancelled: 'info' }
  return map[s] || 'info'
}

function planStatusText(s: string) {
  const map: Record<string, string> = { draft: '草稿', running: '运行中', success: '成功', failed: '失败', cancelled: '已取消' }
  return map[s] || s
}

function nextWizardStep() {
  nodeValidationMsg.value = ''
  if (wizardStep.value === 0) {
    // 验证基础信息
    if (!planForm.name || !planForm.cluster_name || !planForm.k8s_version || !planForm.pod_cidr || !planForm.svc_cidr || !planForm.cni_type) {
      ElMessage.warning('请填写所有必填项')
      return
    }
  } else if (wizardStep.value === 1) {
    // 验证节点配置
    if (planForm.nodes.length === 0) {
      nodeValidationMsg.value = '请至少添加一个节点'
      return
    }
    const masters = planForm.nodes.filter(n => n.role === 'master')
    const workers = planForm.nodes.filter(n => n.role === 'worker')
    if (masters.length !== 1 && masters.length !== 3) {
      nodeValidationMsg.value = 'Master 节点数量必须为 1 或 3'
      return
    }
    if (workers.length < 1) {
      nodeValidationMsg.value = 'Worker 节点至少 1 个'
      return
    }
    const unselected = planForm.nodes.some(n => n.server_id === 0)
    if (unselected) {
      nodeValidationMsg.value = '请为每个节点选择服务器'
      return
    }
  }
  wizardStep.value++
}

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
  try {
    const data = await listSSHCredentials({ page: 1, page_size: 20 })
    credentials.value = data.list || []
  } catch {
    // 静默处理，不影响主流程
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
  editingServerId.value = null
  Object.assign(serverForm, { name: '', ip: '', ssh_port: 22, user: 'root', auth_type: 'password', credential_mode: 'manual', credential_id: null, credential: '', remark: '' })
  serverDialogVisible.value = true
  nextTick(() => serverFormRef.value?.clearValidate())
}

function handleEditServer(row: DeployServerItem) {
  editingServerId.value = row.id
  const hasCredId = !!row.credential_id
  Object.assign(serverForm, {
    name: row.name,
    ip: row.ip,
    ssh_port: row.ssh_port,
    user: row.user,
    auth_type: row.auth_type,
    credential_mode: hasCredId ? 'credential' : 'manual',
    credential_id: row.credential_id || null,
    credential: '',
    remark: row.remark || ''
  })
  serverDialogVisible.value = true
  nextTick(() => serverFormRef.value?.clearValidate())
}

function openPlanDialog() {
  Object.assign(planForm, {
    name: '',
    cluster_name: '',
    k8s_version: 'v1.31.0',
    pod_cidr: '10.244.0.0/16',
    svc_cidr: '10.96.0.0/12',
    cni_type: 'flannel',
    nodes: [],
    addons: []
  })
  wizardStep.value = 0
  nodeValidationMsg.value = ''
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

async function handleSaveServer() {
  if (!serverFormRef.value) return
  try {
    await serverFormRef.value.validate()
  } catch {
    return
  }
  savingServer.value = true
  try {
    const useCredentialMode = serverForm.credential_mode === 'credential'
    if (editingServerId.value) {
      const data: Record<string, unknown> = {
        name: serverForm.name,
        ip: serverForm.ip,
        ssh_port: serverForm.ssh_port,
        user: serverForm.user,
        remark: serverForm.remark
      }
      if (useCredentialMode) {
        data.credential_id = serverForm.credential_id
      } else {
        data.auth_type = serverForm.auth_type
        if (serverForm.credential) {
          data.credential = serverForm.credential
        }
      }
      await updateDeployServer(editingServerId.value, data as any)
      ElMessage.success('服务器已更新')
    } else {
      const data: Record<string, unknown> = {
        name: serverForm.name,
        ip: serverForm.ip,
        ssh_port: serverForm.ssh_port,
        user: serverForm.user,
        remark: serverForm.remark
      }
      if (useCredentialMode) {
        data.credential_id = serverForm.credential_id
        data.auth_type = 'password' // 会被后端覆盖
        data.credential = '' // 不需要
      } else {
        data.auth_type = serverForm.auth_type
        data.credential = serverForm.credential
      }
      await createDeployServer(data as any)
      ElMessage.success('服务器已创建')
    }
    serverDialogVisible.value = false
    await loadServers()
  } catch (e: any) {
    ElMessage.error(e?.message || (editingServerId.value ? '更新服务器失败' : '创建服务器失败'))
  } finally {
    savingServer.value = false
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
  try {
    const data = await testDeployServerSSH(row.id)
    const detail = [data.message, data.os && `系统: ${data.os}`, data.kernel && `内核: ${data.kernel}`].filter(Boolean).join(' | ')
    ElMessage.success(detail || 'SSH 连接成功')
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : ''
    ElMessage.error(msg || 'SSH 连接失败')
  }
  await loadServers()
}

async function handleDeleteServer(row: DeployServerItem) {
  await ElMessageBox.confirm(`确认删除服务器“${row.name}”？`, '删除服务器', { type: 'warning' })
  await deleteDeployServer(row.id)
  ElMessage.success('服务器已删除')
  await loadServers()
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

let currentEventSource: EventSource | null = null

function handleDryRun(row: DeployPlanItem) {
  dryRunPlanId.value = row.id
  dryRunDialogVisible.value = true
}

async function handleViewLogs(row: DeployPlanItem) {
  if (!row.task_id) {
    ElMessage.warning('该计划暂无关联任务')
    return
  }
  currentTaskId = row.task_id
  taskLogs.value = []
  logDialogVisible.value = true

  // 先加载已有日志
  try {
    const data = await getDeployTaskLogs(currentTaskId)
    taskLogs.value = data.logs || []
  } catch {
    // 忽略错误，SSE 会继续推送
  }

  // 建立 SSE 连接接收新日志
  if (currentEventSource) {
    currentEventSource.close()
  }
  currentEventSource = subscribeDeployLogs(
    currentTaskId,
    (log) => {
      taskLogs.value.push(log)
      // 自动滚动到底部
      nextTick(() => {
        const container = document.querySelector('.log-container')
        if (container) {
          container.scrollTop = container.scrollHeight
        }
      })
    },
    (status, message) => {
      // 任务完成
      ElMessage.info(`部署任务已完成: ${status}`)
      loadPlans()
    }
  )
}

function closeLogDialog() {
  if (currentEventSource) {
    currentEventSource.close()
    currentEventSource = null
  }
  logDialogVisible.value = false
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

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

/* ── 固定列不透明背景（与 EnhancedTable 一致） ── */
.online-deploy-view :deep(.el-table__body-wrapper td.el-table__cell.el-table-fixed-column--right),
.online-deploy-view :deep(.el-table__body-wrapper td.el-table__cell.el-table-fixed-column--left) {
  background-color: var(--el-bg-color, #ffffff) !important;
}

.online-deploy-view :deep(.el-table__body-wrapper tr:hover > td.el-table__cell.el-table-fixed-column--right),
.online-deploy-view :deep(.el-table__body-wrapper tr:hover > td.el-table__cell.el-table-fixed-column--left),
.online-deploy-view :deep(.el-table__body-wrapper tr.hover-row > td.el-table__cell.el-table-fixed-column--right),
.online-deploy-view :deep(.el-table__body-wrapper tr.hover-row > td.el-table__cell.el-table-fixed-column--left) {
  background: linear-gradient(0deg, var(--el-table-row-hover-bg-color, #f5f7fa), var(--el-table-row-hover-bg-color, #f5f7fa)),
              var(--el-bg-color, #ffffff) !important;
}

.online-deploy-view :deep(.el-table__body-wrapper tr.current-row > td.el-table__cell.el-table-fixed-column--right),
.online-deploy-view :deep(.el-table__body-wrapper tr.current-row > td.el-table__cell.el-table-fixed-column--left) {
  background: linear-gradient(0deg, var(--el-table-current-row-bg-color, #ecf5ff), var(--el-table-current-row-bg-color, #ecf5ff)),
              var(--el-bg-color, #ffffff) !important;
}

.add-node-btn {
  margin-top: 10px;
}

.log-container {
  max-height: 500px;
  overflow-y: auto;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  padding: 12px;
}

.log-content {
  margin: 0;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--color-text-primary);
}
</style>
