<template>
  <EnhancedTable
    ref="tableRef"
    :data="data"
    :columns="columns"
    :persist-key="persistKey"
    :show-tools="showTools"
    :row-key="getNamespacedRowKey"
    size="small"
    stripe
    border
    @sort-change="emit('sort-change', $event)"
  >
    <template #cell-namespace="{ row }">
      <span class="k8s-ns" :data-ns-color="nsColorIndex(String(row?.metadata?.namespace ?? ''))">{{ String(row?.metadata?.namespace ?? '-') }}</span>
    </template>
    <template #cell-name="{ row }">
      <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
    </template>
    <template #cell-schedule="{ row }">
      <el-tooltip
        placement="top"
        :show-after="250"
        effect="dark"
        :disabled="getScheduleText(row) === '-'"
      >
        <template #content>
          <div class="max-w-[320px] leading-6">
            <div class="text-[13px] text-white">{{ describeCronSchedule(row?.spec?.schedule) }}</div>
            <div class="mt-1 font-mono text-[12px] text-white/75">{{ getScheduleText(row) }}</div>
          </div>
        </template>
        <span class="cursor-help font-mono text-[13px] text-slate-900">{{ getScheduleText(row) }}</span>
      </el-tooltip>
    </template>
    <template #cell-suspend="{ row }">
      <span :class="['k8s-status', row?.spec?.suspend ? 'k8s-status--warn' : 'k8s-status--ok']">{{ row?.spec?.suspend ? 'true' : 'false' }}</span>
    </template>
    <template #cell-active="{ row }">
      <span class="k8s-num">{{ formatNumber(row?.status?.active) }}</span>
    </template>
    <template #cell-lastScheduleTime="{ row }">
      <span class="k8s-age">{{ formatTs(row?.status?.lastScheduleTime) }}</span>
    </template>
    <template #cell-lastSuccessfulTime="{ row }">
      <span class="k8s-age">{{ formatTs(row?.status?.lastSuccessfulTime) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip content="详情" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--info" @click="props.openCronJobDetail(row)"><el-icon><View /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="立即执行" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--ok" @click="props.triggerCronJob(row)"><el-icon><VideoPlay /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" :content="row?.spec?.suspend ? '恢复调度' : '暂停调度'" placement="top" :show-after="300">
          <button class="k8s-act-btn" :class="row?.spec?.suspend ? 'k8s-act-btn--ok' : 'k8s-act-btn--warn'" @click="props.toggleCronJobSuspend(row)"><el-icon><VideoPause /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEditCronJob(row)"><el-icon><EditPen /></el-icon></button>
        </el-tooltip>
        <span class="k8s-act-divider" />
        <el-tooltip content="YAML" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--violet" @click="props.openCronJobYaml(row)"><el-icon><Document /></el-icon></button>
        </el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300">
          <button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteCronJobRow(row)"><el-icon><Delete /></el-icon></button>
        </el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, EditPen, VideoPause, VideoPlay, View } from '@element-plus/icons-vue'
import { ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText, getNamespacedRowKey, nsColorIndex } from '@/features/k8s/pages/ClusterManageView.utils'

const columns: EnhancedColumn[] = [
  { key: 'namespace', label: 'Namespace', prop: 'metadata.namespace', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'name', label: '名称', prop: 'metadata.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'schedule', label: 'Schedule', prop: 'spec.schedule', minWidth: 200, sortable: 'custom', overflowTooltip: false, defaultVisible: true },
  { key: 'suspend', label: 'Suspend', prop: 'spec.suspend', width: 120, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'active', label: 'Active', prop: 'status.active', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'lastScheduleTime', label: 'Last Schedule', prop: 'status.lastScheduleTime', width: 168, sortable: 'custom', defaultVisible: true },
  { key: 'lastSuccessfulTime', label: 'Last Successful', prop: 'status.lastSuccessfulTime', width: 200, sortable: 'custom', defaultVisible: false },
  { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
  { key: 'concurrencyPolicy', label: 'Concurrency', prop: 'spec.concurrencyPolicy', width: 150, sortable: 'custom', defaultVisible: false },
  { key: 'actions', label: '操作', width: 228, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

function formatNumber(v: any): string {
  const n = Number(v)
  return Number.isFinite(n) ? String(n) : '-'
}

function formatTs(ts: any): string {
  if (!ts) return '-'
  const t = new Date(String(ts)).getTime()
  if (!Number.isFinite(t)) return '-'
  const date = new Date(t)
  const yyyy = date.getFullYear()
  const mm = String(date.getMonth() + 1).padStart(2, '0')
  const dd = String(date.getDate()).padStart(2, '0')
  const hh = String(date.getHours()).padStart(2, '0')
  const mi = String(date.getMinutes()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd} ${hh}:${mi}`
}

function getScheduleText(row: any): string {
  const value = String(row?.spec?.schedule ?? '').trim()
  return value || '-'
}

function describeCronSchedule(expr: any): string {
  const cron = String(expr ?? '').trim()
  if (!cron) return '未设置执行计划'

  const fields = cron.split(/\s+/)
  if (fields.length !== 5) return `Cron 表达式：${cron}`

  const [minute, hour, dayOfMonth, month, dayOfWeek] = fields
  const timeText = formatCronTime(hour, minute)
  const weekText = formatWeekday(dayOfWeek)
  const monthDayText = formatMonthDay(dayOfMonth)

  if (minute === '*' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return '每分钟执行一次'
  }

  const minuteStep = readStep(minute)
  if (minuteStep && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return `每 ${minuteStep} 分钟执行一次`
  }

  const hourStep = readStep(hour)
  if (isNumber(minute) && hourStep && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return `每 ${hourStep} 小时的第 ${Number(minute)} 分执行`
  }

  if (timeText && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return `每天 ${timeText} 执行`
  }

  const dayStep = readStep(dayOfMonth)
  if (timeText && dayStep && month === '*' && dayOfWeek === '*') {
    return `每 ${dayStep} 天 ${timeText} 执行一次`
  }

  if (timeText && weekText && dayOfMonth === '*' && month === '*') {
    return `每周${weekText} ${timeText} 执行`
  }

  if (timeText && monthDayText && month === '*' && dayOfWeek === '*') {
    return `每月${monthDayText} ${timeText} 执行`
  }

  const monthStep = readStep(month)
  if (timeText && monthDayText && monthStep && dayOfWeek === '*') {
    return `每 ${monthStep} 个月的${monthDayText} ${timeText} 执行`
  }

  return `自定义 Cron 计划：${cron}`
}

function readStep(value: string): number | null {
  const match = /^\*\/(\d+)$/.exec(value)
  if (!match) return null
  const step = Number(match[1])
  return Number.isFinite(step) && step > 0 ? step : null
}

function isNumber(value: string): boolean {
  return /^\d+$/.test(value)
}

function formatCronTime(hour: string, minute: string): string | null {
  if (!isNumber(hour) || !isNumber(minute)) return null
  return `${String(Number(hour)).padStart(2, '0')}:${String(Number(minute)).padStart(2, '0')}`
}

function formatMonthDay(value: string): string | null {
  if (isNumber(value)) return `${Number(value)} 日`
  const step = readStep(value)
  if (step) return `每 ${step} 天`
  return null
}

function formatWeekday(value: string): string | null {
  if (value === '*') return null
  const mapWeekday = (raw: string): string | null => {
    const normalized = raw === '7' ? '0' : raw
    const labels: Record<string, string> = {
      '0': '日',
      '1': '一',
      '2': '二',
      '3': '三',
      '4': '四',
      '5': '五',
      '6': '六'
    }
    return labels[normalized] ?? null
  }

  if (/^\d(?:,\d)*$/.test(value)) {
    const labels = value.split(',').map(mapWeekday).filter(Boolean)
    return labels.length ? labels.join('、周') : null
  }

  if (/^\d-\d$/.test(value)) {
    const [start, end] = value.split('-')
    const startText = mapWeekday(start)
    const endText = mapWeekday(end)
    return startText && endText ? `${startText}至周${endText}` : null
  }

  const one = mapWeekday(value)
  return one ?? null
}

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  openCronJobDetail: (row: any) => void
  openEditCronJob: (row: any) => void
  openCronJobYaml: (row: any) => void
  deleteCronJobRow: (row: any) => void
  triggerCronJob: (row: any) => void
  toggleCronJobSuspend: (row: any) => void
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
}>()

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>
