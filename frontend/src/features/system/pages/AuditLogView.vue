<template>
  <div class="audit-log-view">
    <div class="audit-card">
      <div class="audit-card-header">
        <div class="header-left">
          <h3 class="audit-title">操作审计日志</h3>
          <span class="audit-subtitle">记录系统所有写操作，帮助追溯变更来源</span>
        </div>
      </div>

      <div class="audit-toolbar">
        <div class="toolbar-filters">
          <el-input v-model="filter.username" placeholder="用户名" clearable class="filter-input" :prefix-icon="User" @keyup.enter="fetchData" />
          <el-select v-model="filter.action" placeholder="全部动作" clearable class="filter-select">
            <el-option label="创建" value="create" />
            <el-option label="更新" value="update" />
            <el-option label="删除" value="delete" />
            <el-option label="执行" value="exec" />
            <el-option label="伸缩" value="scale" />
            <el-option label="重启" value="restart" />
            <el-option label="回滚" value="rollout" />
            <el-option label="驱逐" value="drain" />
            <el-option label="封锁" value="cordon" />
            <el-option label="认证" value="auth" />
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

      <el-table :data="tableData" v-loading="loading" class="audit-table" :header-cell-style="{ background: 'var(--table-header-bg, #f7f9fc)', color: 'var(--table-header-text, #64748b)' }">
        <el-table-column prop="created_at" label="时间" width="170" :formatter="fmtTime" />
        <el-table-column prop="username" label="用户" width="100">
          <template #default="{ row }">
            <span class="cell-user">{{ row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="动作" width="90">
          <template #default="{ row }">
            <span :class="['action-badge', `action-badge--${actionClass(row.action)}`]">{{ actionLabel(row.action) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源类型" width="120">
          <template #default="{ row }">
            <span class="cell-resource">{{ row.resource }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="resource_name" label="资源名" min-width="160" show-overflow-tooltip />
        <el-table-column prop="namespace" label="命名空间" width="120" show-overflow-tooltip />
        <el-table-column prop="cluster_id" label="集群" width="70" align="center" />
        <el-table-column prop="status_code" label="状态" width="80" align="center">
          <template #default="{ row }">
            <span :class="['status-dot', row.status_code >= 400 ? 'status-dot--error' : 'status-dot--ok']"></span>
            <span class="status-text">{{ row.status_code }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="client_ip" label="来源IP" width="130" />
        <el-table-column prop="request_id" label="请求ID" width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="cell-mono">{{ row.request_id?.slice(0, 8) }}</span>
          </template>
        </el-table-column>
      </el-table>

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
  } catch {
    tableData.value = []
    total.value = 0
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
    case 'create': return 'success'
    case 'update': case 'scale': case 'restart': case 'rollout': return 'warning'
    case 'delete': case 'drain': case 'cordon': return 'danger'
    default: return 'info'
  }
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
.audit-card-header {
  padding: 20px 22px 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.audit-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--color-text-primary, #111827);
  margin: 0;
}
.audit-subtitle {
  font-size: 13px;
  color: var(--color-text-muted, #94a3b8);
}
.audit-toolbar {
  padding: 16px 22px;
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
.audit-table {
  border-radius: 0;
}
.cell-user {
  font-weight: 600;
  color: var(--color-text-primary, #111827);
}
.cell-resource {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--color-bg-muted, #f8fafc);
  color: var(--color-text-secondary, #4b5563);
}
.cell-mono {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 12px;
  color: var(--color-text-muted, #94a3b8);
}
.action-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.6;
}
.action-badge--success {
  background: rgba(16, 185, 129, 0.1);
  color: #059669;
}
.action-badge--warning {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
}
.action-badge--danger {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
}
.action-badge--info {
  background: rgba(59, 130, 246, 0.1);
  color: #2563eb;
}
.status-dot {
  display: inline-block;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  margin-right: 5px;
  vertical-align: middle;
}
.status-dot--ok {
  background: #10b981;
  box-shadow: 0 0 4px rgba(16, 185, 129, 0.4);
}
.status-dot--error {
  background: #ef4444;
  box-shadow: 0 0 4px rgba(239, 68, 68, 0.4);
}
.status-text {
  font-size: 12px;
  color: var(--color-text-secondary, #4b5563);
}
.audit-footer {
  padding: 14px 22px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.06));
}
.total-hint {
  font-size: 13px;
  color: var(--color-text-muted, #94a3b8);
}
</style>
