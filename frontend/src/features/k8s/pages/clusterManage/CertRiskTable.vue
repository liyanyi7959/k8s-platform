<template>
  <div v-if="!clusterOverview" class="dash-empty">
    <EmptyState type="no-data" description="证书风险数据未加载">
      <el-button type="primary" size="small" :loading="loading" @click="$emit('reload')">重新加载</el-button>
    </EmptyState>
  </div>
  <div v-else-if="certRows.length === 0" class="dash-empty">
    <EmptyState type="no-data" description="暂无证书风险数据" />
  </div>
  <el-table v-else :data="certRows" size="small" class="cert-table">
    <el-table-column prop="name" label="证书" min-width="200" />
    <el-table-column prop="component" label="所属组件 / 用途" min-width="220">
      <template #default="{ row }">
        <div class="cert-comp">{{ row.component }}</div>
        <div class="cert-purpose">{{ row.purpose }}</div>
      </template>
    </el-table-column>
    <el-table-column prop="not_before" label="生效时间" width="170">
      <template #default="{ row }">{{ fmtTime(row.not_before) }}</template>
    </el-table-column>
    <el-table-column prop="not_after" label="过期时间" width="170">
      <template #default="{ row }">{{ fmtTime(row.not_after) }}</template>
    </el-table-column>
    <el-table-column prop="days_left" label="剩余天数" width="110" align="center" header-align="center">
      <template #default="{ row }">
        <span :class="certDaysClass(row)">{{ fmtDays(row.days_left) }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="status" label="风险" width="100" align="center" header-align="center">
      <template #default="{ row }">
        <el-tag size="small" :type="certStatusTagType(row.status)">{{ certStatusText(row.status) }}</el-tag>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import EmptyState from '@/shared/components/EmptyState.vue'

type CertStatus = 'ok' | 'warn' | 'critical' | 'unknown'
type CertRow = {
  key: string; name: string; component: string; purpose: string
  not_before?: string; not_after?: string; days_left?: number; status: CertStatus
}

defineProps<{
  clusterOverview: any
  certRows: CertRow[]
  loading: boolean
}>()

defineEmits<{
  (e: 'reload'): void
}>()

function fmtTime(v?: string): string {
  const s = String(v ?? '').trim()
  if (!s) return '—'
  const ms = Date.parse(s)
  if (!Number.isFinite(ms)) return s
  return new Date(ms).toLocaleString()
}

function fmtDays(v?: number): string {
  if (v == null || !Number.isFinite(v)) return '—'
  return String(Math.round(v))
}

function certDaysClass(row: CertRow): string {
  if (row.status === 'critical') return 'cert-days cert-days--critical'
  if (row.status === 'warn') return 'cert-days cert-days--warn'
  if (row.status === 'ok') return 'cert-days cert-days--ok'
  return 'cert-days'
}

function certStatusTagType(status: CertStatus): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'critical') return 'danger'
  if (status === 'warn') return 'warning'
  if (status === 'ok') return 'success'
  return 'info'
}

function certStatusText(status: CertStatus): string {
  if (status === 'critical') return '紧急'
  if (status === 'warn') return '预警'
  if (status === 'ok') return '正常'
  return '未知'
}
</script>

<style scoped>
.dash-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  min-height: 120px;
}

.cert-table {
  width: 100%;
}

.cert-table :deep(.el-table__header-wrapper th) {
  background: var(--color-bg-muted) !important;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
}

:global(html.dark) .cert-table :deep(.el-table__header-wrapper th) {
  background: rgba(15, 23, 42, 0.4) !important;
}

.cert-comp {
  font-weight: 900;
  color: rgba(2, 6, 23, 0.88);
}

:global(html.dark) .cert-comp {
  color: rgba(226, 232, 240, 0.9);
}

.cert-purpose {
  margin-top: 2px;
  font-size: 12px;
  color: var(--app-muted);
}

.cert-days {
  font-weight: 900;
}

.cert-days--critical {
  color: var(--c-red-600);
}

.cert-days--warn {
  color: var(--c-amber-600);
}

.cert-days--ok {
  color: var(--c-emerald-600);
}
</style>
