<template>
  <EnhancedTable
    ref="tableRef"
    :data="data"
    :columns="columns"
    :persist-key="persistKey"
    :show-tools="showTools"
    :row-class-name="getRowClassName"
    row-key="metadata.uid"
    size="small"
    stripe
    border
    @sort-change="emit('sort-change', $event)"
  >
    <template #topbar-left>
      <div class="events-toolbar">
        <el-select v-model="typeFilterModel" class="events-filter" clearable placeholder="事件类型">
          <el-option v-for="item in typeOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="reasonFilterModel" class="events-filter events-filter--reason" clearable filterable placeholder="Reason">
          <el-option v-for="item in reasonOptions" :key="item" :label="item" :value="item" />
        </el-select>
      </div>
    </template>

    <template #cell-time="{ row }">
      <span :class="['events-time', isWarningRow(row) ? 'events-time--warning' : '']">{{ getLastTimestamp(row) }}</span>
    </template>
    <template #cell-type="{ row }">
      <el-tag size="small" effect="plain" :type="isWarningRow(row) ? 'danger' : 'info'">{{ String(row?.type ?? '-') }}</el-tag>
    </template>
    <template #cell-reason="{ row }">
      <span :class="['events-reason', isWarningRow(row) ? 'events-reason--warning' : '']">{{ String(row?.reason ?? '-') }}</span>
    </template>
    <template #cell-namespace="{ row }">
      <span>{{ getInvolvedNamespace(row) }}</span>
    </template>
    <template #cell-object="{ row }">
      <div class="events-object">
        <span class="events-object__kind">{{ getInvolvedKind(row) }}</span>
        <span class="events-object__name">{{ getInvolvedName(row) }}</span>
      </div>
    </template>
    <template #cell-message="{ row }">
      <el-popover
        v-if="shouldUseMessagePopover(row)"
        placement="top-start"
        :width="520"
        trigger="hover"
        :show-after="180"
        popper-class="events-message-popper"
      >
        <template #reference>
          <button type="button" :class="['events-message-trigger', isWarningRow(row) ? 'events-message-trigger--warning' : '']">
            <span :class="['events-message', 'events-message--truncate', isWarningRow(row) ? 'events-message--warning' : '']">{{ getMessageText(row) }}</span>
            <span class="events-message__more">展开</span>
          </button>
        </template>

        <div class="events-message-card">
          <div class="events-message-card__meta">
            <span>{{ getInvolvedKind(row) }}</span>
            <span>{{ getInvolvedName(row) }}</span>
            <span v-if="Number(row?.count ?? 0) > 1">累计 {{ Number(row?.count ?? 0) }} 次</span>
          </div>
          <div :class="['events-message-card__body', isWarningRow(row) ? 'events-message-card__body--warning' : '']">{{ getMessageText(row) }}</div>
        </div>
      </el-popover>

      <span v-else :class="['events-message', isWarningRow(row) ? 'events-message--warning' : '']">{{ getMessageText(row) }}</span>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'

const columns: EnhancedColumn[] = [
  { key: 'time', label: 'Time', prop: 'lastTimestamp', width: 180, sortable: 'custom', defaultVisible: true },
  { key: 'type', label: 'Type', prop: 'type', width: 120, sortable: 'custom', defaultVisible: true },
  { key: 'reason', label: 'Reason', prop: 'reason', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'namespace', label: 'Namespace', prop: 'involvedObject.namespace', width: 160, sortable: 'custom', defaultVisible: true },
  { key: 'object', label: 'Object', prop: 'involvedObject.name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'message', label: 'Message', prop: 'message', minWidth: 360, overflowTooltip: false, defaultVisible: true },
  { key: 'count', label: 'Count', prop: 'count', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: false }
]

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  typeFilter: string
  reasonFilter: string
  typeOptions: string[]
  reasonOptions: string[]
}>()

const emit = defineEmits<{
  (e: 'sort-change', v: any): void
  (e: 'update:typeFilter', value: string): void
  (e: 'update:reasonFilter', value: string): void
}>()

const typeFilterModel = computed({
  get: () => props.typeFilter,
  set: (value: string) => emit('update:typeFilter', value)
})

const reasonFilterModel = computed({
  get: () => props.reasonFilter,
  set: (value: string) => emit('update:reasonFilter', value)
})

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })

function getInvolvedObject(row: any) {
  return row?.involvedObject ?? row?.regarding ?? {}
}

function getInvolvedNamespace(row: any): string {
  const namespace = String(getInvolvedObject(row)?.namespace ?? '').trim()
  return namespace || '-'
}

function getInvolvedKind(row: any): string {
  const kind = String(getInvolvedObject(row)?.kind ?? '').trim()
  return kind || 'Unknown'
}

function getInvolvedName(row: any): string {
  const name = String(getInvolvedObject(row)?.name ?? '').trim()
  return name || '-'
}

function getLastTimestamp(row: any): string {
  const value = String(row?.lastTimestamp ?? row?.eventTime ?? row?.metadata?.creationTimestamp ?? '').trim()
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function getMessageText(row: any): string {
  const value = String(row?.message ?? '').trim()
  return value || '-'
}

function shouldUseMessagePopover(row: any): boolean {
  const message = getMessageText(row)
  return message.length > 96 || message.includes('\n')
}

function isWarningRow(row: any): boolean {
  return String(row?.type ?? '').trim().toLowerCase() === 'warning'
}

function getRowClassName({ row }: { row: any; rowIndex: number }): string {
  return isWarningRow(row) ? 'event-row--warning' : ''
}
</script>

<style scoped>
.events-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.events-filter {
  width: 160px;
}

.events-filter--reason {
  width: 220px;
}

.events-time,
.events-reason,
.events-message {
  color: var(--el-text-color-regular);
}

.events-message-trigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.events-message--truncate {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.events-message__more {
  flex: none;
  color: var(--el-color-primary);
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
}

.events-message-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.events-message-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.events-message-card__body {
  color: var(--el-text-color-primary);
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.events-time--warning,
.events-reason--warning,
.events-message--warning {
  color: #b91c1c;
  font-weight: 600;
}

.events-message-card__body--warning {
  color: #991b1b;
}

.events-object {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.events-object__kind {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.events-object__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.event-row--warning > td.el-table__cell) {
  background: rgba(254, 242, 242, 0.92);
}

:global(html.dark) :deep(.event-row--warning > td.el-table__cell) {
  background: rgba(127, 29, 29, 0.18);
}

:global(html.dark) .events-time--warning,
:global(html.dark) .events-reason--warning,
:global(html.dark) .events-message--warning {
  color: #fecaca;
}

:global(html.dark) .events-message__more {
  color: #93c5fd;
}

:global(.events-message-popper.el-popover) {
  max-width: min(560px, calc(100vw - 56px));
  padding: 14px 16px;
  border-radius: 16px;
}
</style>
