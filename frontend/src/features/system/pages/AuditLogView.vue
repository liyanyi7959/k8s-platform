<template>
  <div class="audit-log-view">
    <div class="audit-card">
      <div class="audit-toolbar">
        <div class="toolbar-filters">
          <el-input v-model="filter.username" placeholder="用户名" clearable class="filter-input" :prefix-icon="User" @keyup.enter="fetchData" />
          <el-select v-model="filter.action" placeholder="全部动作" clearable class="filter-select">
            <el-option v-for="item in actionOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-input v-model="filter.resource" placeholder="资源类型" clearable class="filter-input" @keyup.enter="fetchData" />
          <el-date-picker
            v-model="filter.timeRange"
            type="datetimerange"
            range-separator="—"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            class="filter-date"
          />
        </div>
        <div class="toolbar-actions">
          <el-button type="primary" @click="fetchData">
            <el-icon><Search /></el-icon>
            <span>查询</span>
          </el-button>
          <el-button @click="resetFilter">重置</el-button>
        </div>
      </div>

      <div class="audit-table-wrap">
        <el-table :data="tableData" v-loading="loading" class="audit-table" :header-cell-style="{ background: 'var(--table-header-bg, #f7f9fc)', color: 'var(--table-header-text, #64748b)' }">
          <el-table-column prop="created_at" label="时间" width="170" :formatter="fmtTime" />
          <el-table-column prop="username" label="用户" width="100">
            <template #default="{ row }">
              <span class="cell-user">{{ row.username || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="action" label="动作" width="90">
            <template #default="{ row }">
              <span :class="['action-badge', `action-badge--${actionClass(row.action)}`]">{{ actionLabel(row.action) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="resource" label="资源类型" width="120">
            <template #default="{ row }">
              <span :class="['cell-resource', `cell-resource--${resourceClass(row.resource)}`]">{{ row.resource || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="resource_name" label="资源名" min-width="160" show-overflow-tooltip />
          <el-table-column prop="namespace" label="命名空间" width="120" show-overflow-tooltip />
          <el-table-column prop="cluster_id" label="集群" width="70" align="center" />
          <el-table-column prop="status_code" label="结果" width="110" align="center">
            <template #default="{ row }">
              <span :class="['result-pill', isAuditSuccess(row.status_code) ? 'result-pill--ok' : 'result-pill--error']">
                {{ resultLabel(row.status_code) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="detail" label="详情" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="detail-text">{{ row.detail || row.path || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="client_ip" label="来源IP" width="130" />
          <el-table-column prop="request_id" label="请求ID" width="140" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="cell-mono">{{ row.request_id?.slice(0, 8) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="audit-footer">
        <span class="total-hint">共 {{ total }} 条记录</span>
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[20, 50, 100]"
          layout="sizes, prev, pager, next"
          @current-change="fetchData"
          @size-change="fetchData"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Search, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getAuditLogs, type AuditLog } from '@/features/system/api/audit'

const loading = ref(false)
const tableData = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const filter = reactive({
  username: '',
  action: '',
  resource: '',
  timeRange: null as [string, string] | null
})

const actionOptions = [
  { label: '创建', value: 'create' },
  { label: '更新', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '执行', value: 'exec' },
  { label: '伸缩', value: 'scale' },
  { label: '重启', value: 'restart' },
  { label: '回滚', value: 'rollout' },
  { label: '驱逐', value: 'drain' },
  { label: '封锁', value: 'cordon' },
  { label: '认证', value: 'auth' }
] as const

function resetFilter() {
  filter.username = ''
  filter.action = ''
  filter.resource = ''
  filter.timeRange = null
  page.value = 1
  fetchData()
}

async function fetchData() {
  loading.value = true
  try {
    const result = await getAuditLogs({
      page: page.value,
      page_size: pageSize.value,
      username: filter.username || undefined,
      action: filter.action || undefined,
      resource: filter.resource || undefined,
      start_time: filter.timeRange?.[0] || undefined,
      end_time: filter.timeRange?.[1] || undefined
    })
    tableData.value = result.items ?? []
    total.value = result.total ?? 0
  } catch (error) {
    tableData.value = []
    total.value = 0
    ElMessage.error(getErrorMessage(error, '审计日志加载失败'))
  } finally {
    loading.value = false
  }
}

function fmtTime(_row: AuditLog, _col: unknown, value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

const actionMap: Record<string, string> = {
  create: '创建', update: '更新', delete: '删除', exec: '执行',
  scale: '伸缩', restart: '重启', rollout: '回滚', drain: '驱逐',
  cordon: '封锁', auth: '认证',
}

function actionLabel(action: string) {
  return actionMap[action] || action
}

function actionClass(action: string): string {
  switch (action) {
    case 'create': return 'create'
    case 'update': return 'update'
    case 'delete': return 'delete'
    case 'exec': return 'exec'
    case 'scale': return 'scale'
    case 'restart': return 'restart'
    case 'rollout': return 'rollout'
    case 'drain': return 'drain'
    case 'cordon': return 'cordon'
    case 'auth': return 'auth'
    default: return 'unknown'
  }
}

function resourceClass(resource: string): string {
  switch (resource) {
    case 'user':
    case 'role':
    case 'permission':
      return 'system'
    case 'cluster':
      return 'cluster'
    case 'pod':
    case 'deployment':
    case 'statefulset':
    case 'daemonset':
    case 'service':
    case 'namespace':
      return 'workload'
    case 'session':
      return 'auth'
    default:
      return 'default'
  }
}

function isAuditSuccess(code: number) {
  return code === 0 || (code >= 200 && code < 300)
}

function resultLabel(code: number) {
  if (isAuditSuccess(code)) {
    return '成功'
  }
  return `失败 ${code}`
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

onMounted(fetchData)
</script>

<style scoped>
.audit-log-view {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.audit-card {
  background: var(--color-bg-card, #ffffff);
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  border-radius: 12px;
  box-shadow: var(--shadow-card, 0 1px 3px rgba(15, 23, 42, 0.05));
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.audit-toolbar {
  padding: 18px 20px 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.toolbar-filters {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.toolbar-actions {
  display: flex;
  gap: 8px;
}
.filter-input {
  width: 140px;
}
.filter-select {
  width: 130px;
}
.filter-date {
  max-width: 320px;
}
.audit-table-wrap {
  padding: 0 16px;
}
.audit-table {
  width: 100%;
}
.cell-user {
  font-weight: 600;
  color: var(--color-text-primary, #111827);
}
.cell-resource {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 12px;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--color-bg-muted, #f8fafc);
  color: var(--color-text-secondary, #4b5563);
}
.cell-resource--system {
  background: rgba(59, 130, 246, 0.12);
  color: #1d4ed8;
}
.cell-resource--cluster {
  background: rgba(16, 185, 129, 0.12);
  color: #059669;
}
.cell-resource--workload {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}
.cell-resource--auth {
  background: rgba(20, 184, 166, 0.14);
  color: #0f766e;
}
.cell-mono {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 12px;
  color: var(--color-text-muted, #94a3b8);
}
.action-badge {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.4;
}
.action-badge--create {
  background: rgba(16, 185, 129, 0.1);
  color: #059669;
}
.action-badge--update {
  background: rgba(59, 130, 246, 0.12);
  color: #2563eb;
}
.action-badge--delete {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
}
.action-badge--exec {
  background: rgba(168, 85, 247, 0.12);
  color: #7c3aed;
}
.action-badge--scale {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}
.action-badge--restart {
  background: rgba(249, 115, 22, 0.14);
  color: #c2410c;
}
.action-badge--rollout {
  background: rgba(99, 102, 241, 0.12);
  color: #4338ca;
}
.action-badge--drain {
  background: rgba(220, 38, 38, 0.12);
  color: #b91c1c;
}
.action-badge--cordon {
  background: rgba(71, 85, 105, 0.12);
  color: #334155;
}
.action-badge--auth {
  background: rgba(20, 184, 166, 0.12);
  color: #0f766e;
}
.action-badge--unknown {
  background: rgba(148, 163, 184, 0.16);
  color: #64748b;
}
.result-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 72px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
}
.result-pill--ok {
  background: rgba(16, 185, 129, 0.12);
  color: #059669;
}
.result-pill--error {
  background: rgba(239, 68, 68, 0.12);
  color: #dc2626;
}
.detail-text {
  color: var(--color-text-secondary, #4b5563);
}
.audit-footer {
  padding: 14px 20px 18px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.total-hint {
  font-size: 13px;
  color: var(--color-text-muted, #94a3b8);
}
</style>
