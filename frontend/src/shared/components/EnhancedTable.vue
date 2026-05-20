<template>
  <div v-if="!fullscreen" class="enhanced-table" ref="containerRef" @dblclick="onContainerDblclick">
    <div v-if="showTopbar" class="table-topbar">
      <div class="table-topbar-left">
        <slot name="topbar-left" :selected-rows="selectedRows" :selected-count="selectedCount" />
      </div>
      <div v-if="showTools" class="table-topbar-right">
        <el-space>
          <el-tooltip content="刷新" placement="top">
            <button class="tool-btn" type="button" @click="emit('refresh')">
              <el-icon><RefreshRight /></el-icon>
            </button>
          </el-tooltip>

          <el-dropdown trigger="click" @command="onSizeCommand">
            <button class="tool-btn" type="button" title="密度">
              <el-icon><Operation /></el-icon>
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="large">
                  <el-icon v-if="tableSize === 'large'"><Check /></el-icon>
                  宽松
                </el-dropdown-item>
                <el-dropdown-item command="default">
                  <el-icon v-if="tableSize === 'default'"><Check /></el-icon>
                  默认
                </el-dropdown-item>
                <el-dropdown-item command="small">
                  <el-icon v-if="tableSize === 'small'"><Check /></el-icon>
                  紧凑
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <el-dropdown v-if="columns.length > 0" trigger="click" :hide-on-click="false">
            <button class="tool-btn" type="button" title="列设置">
              <el-icon><Setting /></el-icon>
            </button>
            <template #dropdown>
                <div class="columns-panel">
                  <div class="columns-panel-actions">
                  <el-button size="small" :icon="Check" @click="selectAllColumns">全选</el-button>
                  <el-button size="small" :icon="CircleClose" @click="clearAllColumns">全不选</el-button>
                  <el-button size="small" :icon="RefreshRight" @click="resetColumns">重置</el-button>
                  </div>
                  <el-divider style="margin: 10px 0" />
                  <div class="columns-panel-list">
                  <el-checkbox
                    v-for="c in columns"
                    :key="c.key"
                    v-model="columnVisible[c.key]"
                    :disabled="c.disableToggle === true"
                    @change="persistColumns"
                  >
                    {{ c.label }}
                  </el-checkbox>
                </div>
              </div>
            </template>
          </el-dropdown>

          <el-tooltip :content="fullscreen ? '退出全屏' : '全屏'" placement="top">
            <button class="tool-btn" type="button" @click="toggleFullscreen">
              <el-icon><FullScreen /></el-icon>
            </button>
          </el-tooltip>
        </el-space>
      </div>
    </div>

    <el-table
      ref="tableRef"
      :data="tableData"
      :row-key="rowKey"
      :row-class-name="rowClassName"
      :stripe="stripe"
      :border="border"
      :fit="tableFit"
      :size="tableSize"
      :height="height"
      v-loading="loading"
      @selection-change="onSelectionChange"
      @sort-change="onSortChange"
      @header-dragend="onHeaderDragEnd"
      @row-contextmenu="onRowContextMenu"
    >
      <el-table-column v-if="selectable" type="selection" width="44" />
      <el-table-column v-if="showIndex" type="index" width="54" label="#" />

      <el-table-column
        v-for="col in renderedColumns"
        :key="col.key"
        :column-key="col.key"
        :prop="col.prop"
        :label="col.label"
        :width="col.width"
        :min-width="col.minWidth"
        :fixed="col.fixed"
        :sortable="col.sortable"
        :sort-method="col.sortMethod"
        :align="col.align"
        :header-align="col.headerAlign"
        :class-name="col.align === 'center' ? 'is-center-cell' : ''"
        :show-overflow-tooltip="overflowTooltipOpts(col)"
      >
        <template #default="scope">
          <slot :name="`cell-${col.key}`" v-bind="scope">
            {{ formatCell(scope.row, col) }}
          </slot>
        </template>
      </el-table-column>

      <slot />

      <template #empty>
        <div class="enhanced-table-empty">
          <svg viewBox="0 0 160 120" fill="none" xmlns="http://www.w3.org/2000/svg" class="enhanced-table-empty-svg">
            <ellipse cx="80" cy="108" rx="52" ry="7" fill="currentColor" opacity="0.04" />
            <rect x="40" y="24" width="80" height="68" rx="6" fill="currentColor" opacity="0.03" stroke="currentColor" stroke-opacity="0.08" stroke-width="1" />
            <rect x="56" y="44" width="48" height="4" rx="2" fill="currentColor" opacity="0.08" />
            <rect x="56" y="56" width="34" height="4" rx="2" fill="currentColor" opacity="0.06" />
            <rect x="56" y="68" width="42" height="4" rx="2" fill="currentColor" opacity="0.06" />
          </svg>
          <span class="enhanced-table-empty-text">暂无数据</span>
        </div>
      </template>
    </el-table>

    <div v-if="pagination" class="pager">
      <el-pagination
        v-model:current-page="pageModel"
        v-model:page-size="pageSizeModel"
        :layout="effectivePaginationLayout"
        :page-sizes="pageSizeOptions"
        :total="effectiveTotal"
        @current-change="onCurrentPageChange"
        @size-change="onPageSizeChange"
      />
    </div>
  </div>

  <teleport to="body">
    <div v-if="fullscreen" class="enhanced-table fullscreen" ref="containerRef" @dblclick="onContainerDblclick">
      <div v-if="showTopbar" class="table-topbar">
        <div class="table-topbar-left">
          <slot name="topbar-left" :selected-rows="selectedRows" :selected-count="selectedCount" />
        </div>
        <div v-if="showTools" class="table-topbar-right">
          <el-space>
            <el-tooltip content="刷新" placement="top">
              <button class="tool-btn" type="button" @click="emit('refresh')">
                <el-icon><RefreshRight /></el-icon>
              </button>
            </el-tooltip>

            <el-dropdown trigger="click" @command="onSizeCommand">
              <button class="tool-btn" type="button" title="密度">
                <el-icon><Operation /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="large">
                    <el-icon v-if="tableSize === 'large'"><Check /></el-icon>
                    宽松
                  </el-dropdown-item>
                  <el-dropdown-item command="default">
                    <el-icon v-if="tableSize === 'default'"><Check /></el-icon>
                    默认
                  </el-dropdown-item>
                  <el-dropdown-item command="small">
                    <el-icon v-if="tableSize === 'small'"><Check /></el-icon>
                    紧凑
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>

            <el-dropdown v-if="columns.length > 0" trigger="click" :hide-on-click="false">
              <button class="tool-btn" type="button" title="列设置">
                <el-icon><Setting /></el-icon>
              </button>
              <template #dropdown>
                <div class="columns-panel">
                  <div class="columns-panel-actions">
                  <el-button size="small" :icon="Check" @click="selectAllColumns">全选</el-button>
                  <el-button size="small" :icon="CircleClose" @click="clearAllColumns">全不选</el-button>
                  <el-button size="small" :icon="RefreshRight" @click="resetColumns">重置</el-button>
                  </div>
                  <el-divider style="margin: 10px 0" />
                  <div class="columns-panel-list">
                    <el-checkbox
                      v-for="c in columns"
                      :key="c.key"
                      v-model="columnVisible[c.key]"
                      :disabled="c.disableToggle === true"
                      @change="persistColumns"
                    >
                      {{ c.label }}
                    </el-checkbox>
                  </div>
                </div>
              </template>
            </el-dropdown>

            <el-tooltip :content="fullscreen ? '退出全屏' : '全屏'" placement="top">
              <button class="tool-btn" type="button" @click="toggleFullscreen">
                <el-icon><FullScreen /></el-icon>
              </button>
            </el-tooltip>
          </el-space>
        </div>
      </div>

      <div class="fullscreen-table-wrap" ref="fullscreenTableWrapRef">
        <el-table
          ref="tableRef"
          :data="tableData"
          :row-key="rowKey"
          :row-class-name="rowClassName"
          :stripe="stripe"
          :border="border"
          :fit="tableFit"
          :size="tableSize"
          :height="fullscreenTableHeight"
          v-loading="loading"
          @selection-change="onSelectionChange"
          @sort-change="onSortChange"
          @header-dragend="onHeaderDragEnd"
          @row-contextmenu="onRowContextMenu"
        >
          <el-table-column v-if="selectable" type="selection" width="44" />
          <el-table-column v-if="showIndex" type="index" width="54" label="#" />

          <el-table-column
            v-for="col in renderedColumns"
            :key="col.key"
            :column-key="col.key"
            :prop="col.prop"
            :label="col.label"
            :width="col.width"
            :min-width="col.minWidth"
            :fixed="col.fixed"
            :sortable="col.sortable"
            :sort-method="col.sortMethod"
            :align="col.align"
            :header-align="col.headerAlign"
            :class-name="col.align === 'center' ? 'is-center-cell' : ''"
            :show-overflow-tooltip="overflowTooltipOpts(col)"
          >
            <template #default="scope">
              <slot :name="`cell-${col.key}`" v-bind="scope">
                {{ formatCell(scope.row, col) }}
              </slot>
            </template>
          </el-table-column>

          <slot />

          <template #empty>
            <div class="enhanced-table-empty">
              <svg viewBox="0 0 160 120" fill="none" xmlns="http://www.w3.org/2000/svg" class="enhanced-table-empty-svg">
                <ellipse cx="80" cy="108" rx="52" ry="7" fill="currentColor" opacity="0.04" />
                <rect x="40" y="24" width="80" height="68" rx="6" fill="currentColor" opacity="0.03" stroke="currentColor" stroke-opacity="0.08" stroke-width="1" />
                <rect x="56" y="44" width="48" height="4" rx="2" fill="currentColor" opacity="0.08" />
                <rect x="56" y="56" width="34" height="4" rx="2" fill="currentColor" opacity="0.06" />
                <rect x="56" y="68" width="42" height="4" rx="2" fill="currentColor" opacity="0.06" />
              </svg>
              <span class="enhanced-table-empty-text">暂无数据</span>
            </div>
          </template>
        </el-table>
      </div>

      <div v-if="pagination" class="pager">
        <el-pagination
          v-model:current-page="pageModel"
          v-model:page-size="pageSizeModel"
          :layout="effectivePaginationLayout"
          :page-sizes="pageSizeOptions"
          :total="effectiveTotal"
          @current-change="onCurrentPageChange"
          @size-change="onPageSizeChange"
        />
      </div>
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useSlots, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Check, CircleClose, FullScreen, Operation, RefreshRight, Setting } from '@element-plus/icons-vue'

export type TableSize = 'large' | 'default' | 'small'

export interface EnhancedColumn {
  key: string
  label: string
  prop?: string
  width?: number | string
  minWidth?: number | string
  fixed?: boolean | 'left' | 'right'
  sortable?: boolean | 'custom'
  sortMethod?: (a: any, b: any) => number
  align?: 'left' | 'center' | 'right'
  headerAlign?: 'left' | 'center' | 'right'
  defaultVisible?: boolean
  disableToggle?: boolean
  overflowTooltip?: boolean
  formatter?: (row: any, col: EnhancedColumn, index: number) => unknown
}

const props = defineProps<{
  data: any[]
  columns: EnhancedColumn[]
  rowKey: string | ((row: any) => string)
  rowClassName?: string | ((data: { row: any; rowIndex: number }) => string)
  loading?: boolean
  selectable?: boolean
  showIndex?: boolean
  stripe?: boolean
  border?: boolean
  fit?: boolean
  height?: string | number
  size?: TableSize
  persistKey?: string
  showTopbar?: boolean
  showTools?: boolean
  pagination?: boolean
  paginationMode?: 'server' | 'client'
  refreshOnPageChange?: boolean
  total?: number
  page?: number
  pageSize?: number
  pageSizeOptions?: number[]
  paginationLayout?: string
}>()

const slots = useSlots()
const hasTopbarLeftSlot = computed(() => Boolean(slots['topbar-left']))
const toolsEnabled = computed(() => props.showTools !== false)
const showTopbar = computed(() => {
  if (props.showTopbar !== undefined) return props.showTopbar
  return hasTopbarLeftSlot.value || toolsEnabled.value
})
const showTools = computed(() => showTopbar.value && toolsEnabled.value)
const tableFit = computed(() => props.fit !== false)
const stripe = computed(() => props.stripe !== false)
const border = computed(() => props.border !== false)

/**
 * Element Plus 表格溢出 tooltip 默认会挂到 table wrapper。
 * 当前页面大量使用带 backdrop-filter 的卡片容器，这会让 fixed popper 的定位参考系漂移。
 * 统一挂到 body，避免各类资源列表出现悬浮提示偏位。
 */
function overflowTooltipOpts(col: EnhancedColumn): boolean | Record<string, any> {
  const enabled = col.overflowTooltip ?? (col.key === 'actions' ? false : true)
  if (!enabled) return false
  return {
    appendTo: 'body'
  }
}

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'selection-change', rows: any[]): void
  (e: 'sort-change', v: { prop?: string; order?: 'ascending' | 'descending' | null }): void
  (e: 'row-contextmenu', row: any, event: MouseEvent): void
  (e: 'update:page', v: number): void
  (e: 'update:pageSize', v: number): void
  (e: 'update:size', v: TableSize): void
}>()

const tableRef = ref()
const containerRef = ref<HTMLElement | null>(null)
const fullscreenTableWrapRef = ref<HTMLElement | null>(null)
const fullscreen = ref(false)
const tableViewportWidth = ref(0)
const bodyOverflowBeforeFullscreen = ref<string | null>(null)
const selectedRows = ref<any[]>([])
const selectedCount = computed(() => selectedRows.value.length)
const fullscreenTableHeight = ref<number | undefined>(undefined)

const pageModel = computed({
  get: () => props.page ?? 1,
  set: (v: number) => emit('update:page', v)
})

const pageSizeModel = computed({
  get: () => props.pageSize ?? 20,
  set: (v: number) => emit('update:pageSize', v)
})

const tableSize = ref<TableSize>(props.size ?? 'default')

const pageSizeOptions = computed(() => props.pageSizeOptions ?? [20, 50, 100, 200])
const effectivePaginationLayout = computed(() => props.paginationLayout ?? 'total, sizes, prev, pager, next, jumper')

const effectivePaginationMode = computed<'server' | 'client'>(() => {
  if (props.paginationMode) return props.paginationMode
  return props.total != null ? 'server' : 'client'
})

const effectiveTotal = computed(() => {
  if (!props.pagination) return 0
  if (effectivePaginationMode.value === 'client') return props.data.length
  return Number(props.total ?? 0)
})

const tableData = computed(() => {
  if (!props.pagination || effectivePaginationMode.value !== 'client') return props.data
  const p = Math.max(1, Number(pageModel.value) || 1)
  const ps = Math.max(1, Number(pageSizeModel.value) || 20)
  const start = (p - 1) * ps
  return props.data.slice(start, start + ps)
})

const shouldRefreshOnPageChange = computed(() => {
  if (props.refreshOnPageChange != null) return props.refreshOnPageChange
  return effectivePaginationMode.value === 'server'
})

function sizeStorageKey() {
  if (!props.persistKey) return ''
  return `${props.persistKey}:size`
}

function columnsStorageKey() {
  if (!props.persistKey) return ''
  return `${props.persistKey}:columns`
}

function columnWidthsStorageKey() {
  if (!props.persistKey) return ''
  return `${props.persistKey}:column-widths`
}

const columnVisible = ref<Record<string, boolean>>({})
const columnWidths = ref<Record<string, number>>({})

const visibleColumns = computed(() => props.columns.filter((c) => columnVisible.value[c.key] !== false))

const renderedColumns = computed(() => {
  const columns = visibleColumns.value.map((col) => {
    const normalizedCol: EnhancedColumn = {
      ...col,
      align: col.align ?? (col.key === 'actions' ? 'center' : undefined),
      headerAlign: col.headerAlign ?? (col.key === 'actions' ? 'center' : col.align === 'center' ? 'center' : undefined)
    }
    const manualWidth = columnWidths.value[col.key]
    if (!Number.isFinite(manualWidth)) return normalizedCol
    return {
      ...normalizedCol,
      width: manualWidth,
      minWidth: manualWidth
    }
  })
  return adaptColumnsForViewport(columns, tableViewportWidth.value)
})

function initColumnsVisibility() {
  const defaults: Record<string, boolean> = {}
  for (const c of props.columns) defaults[c.key] = c.defaultVisible !== false
  columnVisible.value = defaults
}

function applyPersistedColumns() {
  const key = columnsStorageKey()
  if (!key) return
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return
    const parsed = JSON.parse(raw) as Record<string, boolean>
    columnVisible.value = { ...columnVisible.value, ...parsed }
  } catch (e) {
    void e
  }
}

function applyPersistedColumnWidths() {
  const key = columnWidthsStorageKey()
  if (!key) return
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return
    const parsed = JSON.parse(raw) as Record<string, number>
    const next: Record<string, number> = {}
    for (const [columnKey, width] of Object.entries(parsed)) {
      const numericWidth = Number(width)
      if (Number.isFinite(numericWidth) && numericWidth > 0) {
        next[columnKey] = numericWidth
      }
    }
    columnWidths.value = next
  } catch (e) {
    void e
  }
}

function persistColumns() {
  const key = columnsStorageKey()
  if (!key) return
  try {
    localStorage.setItem(key, JSON.stringify(columnVisible.value))
  } catch (e) {
    void e
  }
}

function persistColumnWidths() {
  const key = columnWidthsStorageKey()
  if (!key) return
  try {
    localStorage.setItem(key, JSON.stringify(columnWidths.value))
  } catch (e) {
    void e
  }
}

function selectAllColumns() {
  const m = { ...columnVisible.value }
  for (const c of props.columns) m[c.key] = true
  columnVisible.value = m
  persistColumns()
}

function clearAllColumns() {
  const m = { ...columnVisible.value }
  for (const c of props.columns) {
    if (c.disableToggle === true) continue
    m[c.key] = false
  }
  columnVisible.value = m
  persistColumns()
}

function resetColumns() {
  initColumnsVisibility()
  columnWidths.value = {}
  persistColumns()
  persistColumnWidths()
}

function applyPersistedSize() {
  const key = sizeStorageKey()
  if (!key) return
  const v = localStorage.getItem(key) as TableSize | null
  if (!v) return
  if (v === 'large' || v === 'default' || v === 'small') tableSize.value = v
}

function persistSize(v: TableSize) {
  const key = sizeStorageKey()
  if (!key) return
  try {
    localStorage.setItem(key, v)
  } catch (e) {
    void e
  }
}

function onSizeCommand(cmd: string) {
  const v = cmd as TableSize
  if (v !== 'large' && v !== 'default' && v !== 'small') return
  tableSize.value = v
  persistSize(v)
  emit('update:size', v)
}

function onSelectionChange(rows: any[]) {
  selectedRows.value = rows
  emit('selection-change', rows)
}

function onSortChange(v: unknown) {
  emit('sort-change', v as { prop?: string; order?: 'ascending' | 'descending' | null })
}

function onRowContextMenu(row: any, _column: unknown, event: MouseEvent) {
  emit('row-contextmenu', row, event)
}

function onCurrentPageChange(v: number) {
  emit('update:page', v)
  if (shouldRefreshOnPageChange.value) emit('refresh')
}

function onPageSizeChange(v: number) {
  emit('update:pageSize', v)
  if (shouldRefreshOnPageChange.value) emit('refresh')
}

async function copyText(text: string) {
  if (!text) return false
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch (e) {
    void e
  }

  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', 'true')
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    textarea.style.pointerEvents = 'none'
    document.body.appendChild(textarea)
    textarea.select()
    textarea.setSelectionRange(0, text.length)
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    return ok
  } catch (e) {
    void e
    return false
  }
}

async function onContainerDblclick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  const nameEl = target?.closest?.('.k8s-name') as HTMLElement | null
  if (!nameEl) return
  const text = String(nameEl.textContent ?? '').trim()
  if (!text || text === '-') return
  const ok = await copyText(text)
  if (ok) {
    ElMessage.success(`已复制：${text}`)
  }
}

function getByPath(obj: any, path: string) {
  const segs = path.split('.').filter(Boolean)
  let cur = obj
  for (const s of segs) {
    if (cur == null) return undefined
    cur = cur[s]
  }
  return cur
}

function formatCell(row: any, col: EnhancedColumn) {
  if (col.formatter) return col.formatter(row, col, 0)
  if (!col.prop) return ''
  const v = getByPath(row, col.prop)
  return v == null ? '' : String(v)
}

function escapeHtml(v: string) {
  return v
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function downloadBlob(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 500)
}

function exportExcel(fileName = 'export.xls') {
  const cols = visibleColumns.value.filter((c) => c.prop)
  const header = cols.map((c) => `<th>${escapeHtml(c.label)}</th>`).join('')
  const rows = props.data
    .map((r) => {
      const tds = cols
        .map((c) => {
          const v = c.formatter ? c.formatter(r, c, 0) : getByPath(r, c.prop || '')
          return `<td>${escapeHtml(v == null ? '' : String(v))}</td>`
        })
        .join('')
      return `<tr>${tds}</tr>`
    })
    .join('')

  const html =
    `<html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel" xmlns="http://www.w3.org/TR/REC-html40">` +
    `<head><meta charset="UTF-8" /></head><body><table>${`<tr>${header}</tr>`}${rows}</table></body></html>`

  const blob = new Blob([html], { type: 'application/vnd.ms-excel' })
  downloadBlob(blob, fileName)
}

function exportCsv(fileName = 'export.csv') {
  const cols = visibleColumns.value.filter((c) => c.prop)
  const header = cols.map((c) => `"${String(c.label).replaceAll('"', '""')}"`).join(',')
  const lines = props.data.map((r) => {
    return cols
      .map((c) => {
        const v = c.formatter ? c.formatter(r, c, 0) : getByPath(r, c.prop || '')
        return `"${String(v ?? '').replaceAll('"', '""')}"`
      })
      .join(',')
  })
  const csv = [header, ...lines].join('\r\n')
  const blob = new Blob([`\ufeff${csv}`], { type: 'text/csv;charset=utf-8' })
  downloadBlob(blob, fileName)
}

function clearSelection() {
  tableRef.value?.clearSelection?.()
}

function doLayout() {
  tableRef.value?.doLayout?.()
}

function onHeaderDragEnd(newWidth?: number, _oldWidth?: number, column?: { columnKey?: string; property?: string; label?: string }) {
  const nextWidth = Number(newWidth)
  const columnKey = column?.columnKey
    || props.columns.find((item) => item.prop && item.prop === column?.property)?.key
    || props.columns.find((item) => item.label === column?.label)?.key
  if (columnKey && Number.isFinite(nextWidth) && nextWidth > 0) {
    columnWidths.value = {
      ...columnWidths.value,
      [columnKey]: Math.round(nextWidth)
    }
    persistColumnWidths()
  }
  nextTick(() => {
    doLayout()
  })
}

function updateFullscreenTableHeight() {
  if (!fullscreen.value) {
    fullscreenTableHeight.value = undefined
    return
  }
  const h = fullscreenTableWrapRef.value?.clientHeight
  if (!h) return
  fullscreenTableHeight.value = Math.max(120, h)
}

function updateTableViewportWidth() {
  const host = fullscreen.value ? fullscreenTableWrapRef.value : containerRef.value
  const width = host?.clientWidth ?? 0
  tableViewportWidth.value = Math.max(0, width)
}

function parseColumnWidth(value: number | string | undefined): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value !== 'string') return null
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : null
}

function getColumnBaseWidth(col: EnhancedColumn): number {
  return parseColumnWidth(col.width) ?? parseColumnWidth(col.minWidth) ?? estimateColumnWidth(col)
}

function getColumnFloorWidth(col: EnhancedColumn): number {
  const labelWidth = Math.max(64, String(col.label ?? '').trim().length * 14 + 28)
  const key = String(col.key ?? '').toLowerCase()

  if (col.key === 'actions') return getColumnBaseWidth(col)
  if (col.align === 'center') return Math.max(72, Math.min(96, labelWidth))
  if (/(name|pod|object|message|rules|hosts|selector|schedule|controller|provisioner|clusterip|storageclass|service|volume|claim|node|namespace|address|target|keys|osimage)/.test(key)) {
    return Math.max(92, Math.min(148, labelWidth + 18))
  }
  return Math.max(68, Math.min(116, labelWidth + 6))
}

function estimateColumnWidth(col: EnhancedColumn): number {
  const labelWidth = Math.max(88, String(col.label ?? '').trim().length * 15 + 36)
  const key = String(col.key ?? '').toLowerCase()

  if (col.key === 'actions') return parseColumnWidth(col.width) ?? 140
  if (/(name|pod|object)/.test(key)) return Math.max(180, labelWidth + 44)
  if (/(namespace|node|storageclass|clusterip|schedule|controller|provisioner|service|volume|claim|address|target|keys|osimage|message|rules|hosts)/.test(key)) {
    return Math.max(136, labelWidth + 28)
  }
  if (col.align === 'center') return Math.max(88, labelWidth)
  return Math.max(110, labelWidth + 16)
}

function adaptColumnsForViewport(columns: EnhancedColumn[], viewportWidth: number): EnhancedColumn[] {
  if (!columns.length) return columns

  const reserved = (props.selectable ? 44 : 0) + (props.showIndex ? 54 : 0) + 8
  const available = Math.max(320, viewportWidth - reserved)
  if (!viewportWidth || available <= 0) return columns

  const baseWidths = columns.map((col) => getColumnBaseWidth(col))
  const floorWidths = columns.map((col, index) => Math.min(baseWidths[index], getColumnFloorWidth(col)))
  const totalBase = baseWidths.reduce((sum, width) => sum + width, 0)

  if (totalBase <= available) return columns

  const shrinkable = baseWidths.reduce((sum, width, index) => {
    if (columns[index]?.key === 'actions') return sum
    return sum + Math.max(0, width - floorWidths[index])
  }, 0)
  if (shrinkable <= 0) return columns

  const needShrink = totalBase - available

  return columns.map((col, index) => {
    const width = baseWidths[index]
    const floor = floorWidths[index]
    if (col.key === 'actions') {
      return {
        ...col,
        width,
        minWidth: width
      }
    }
    const shrinkRoom = Math.max(0, width - floor)
    const nextWidth = needShrink >= shrinkable
      ? floor
      : Math.max(floor, Math.round(width - (needShrink * shrinkRoom) / shrinkable))

    return {
      ...col,
      width: nextWidth,
      minWidth: nextWidth
    }
  })
}

function toggleFullscreen() {
  fullscreen.value = !fullscreen.value
}

onMounted(() => {
  initColumnsVisibility()
  applyPersistedColumns()
  applyPersistedColumnWidths()
  applyPersistedSize()
  nextTick(() => {
    updateTableViewportWidth()
    updateFullscreenTableHeight()
    doLayout()
  })
})

function onWindowResize() {
  updateTableViewportWidth()
  updateFullscreenTableHeight()
  doLayout()
}

watch(
  fullscreen,
  async (v) => {
    if (v) {
      if (bodyOverflowBeforeFullscreen.value == null) {
        bodyOverflowBeforeFullscreen.value = document.body.style.overflow || ''
      }
      document.body.style.overflow = 'hidden'
      window.addEventListener('resize', onWindowResize)
    } else {
      window.removeEventListener('resize', onWindowResize)
      if (bodyOverflowBeforeFullscreen.value != null) {
        document.body.style.overflow = bodyOverflowBeforeFullscreen.value
        bodyOverflowBeforeFullscreen.value = null
      }
    }
    await nextTick()
    updateTableViewportWidth()
    updateFullscreenTableHeight()
    doLayout()
  },
  { flush: 'post' }
)

watch(
  () => [effectivePaginationMode.value, props.data.length, pageSizeModel.value].join('|'),
  () => {
    if (!props.pagination) return
    if (effectivePaginationMode.value !== 'client') return
    const ps = Math.max(1, Number(pageSizeModel.value) || 20)
    const maxPage = Math.max(1, Math.ceil(props.data.length / ps))
    const p = Math.max(1, Number(pageModel.value) || 1)
    if (p > maxPage) emit('update:page', maxPage)
  }
)

onBeforeUnmount(() => {
  window.removeEventListener('resize', onWindowResize)
  if (bodyOverflowBeforeFullscreen.value != null) {
    document.body.style.overflow = bodyOverflowBeforeFullscreen.value
    bodyOverflowBeforeFullscreen.value = null
  }
})

watch(
  () => props.columns,
  () => {
    initColumnsVisibility()
    applyPersistedColumns()
    applyPersistedColumnWidths()
  },
  { deep: true }
)

watch(
  () => props.size,
  (v) => {
    if (!v) return
    tableSize.value = v
  }
)

watch(
  () => [props.data?.length, visibleColumns.value.length, tableSize.value, fullscreen.value],
  async () => {
    await nextTick()
    updateTableViewportWidth()
    updateFullscreenTableHeight()
    doLayout()
  }
)

defineExpose({
  getColumns: () => props.columns,
  getColumnVisibleMap: () => ({ ...columnVisible.value }),
  getColumnWidthsMap: () => ({ ...columnWidths.value }),
  setColumnVisibleMap: (m: Record<string, boolean>) => {
    columnVisible.value = { ...columnVisible.value, ...m }
    persistColumns()
  },
  setColumnWidthsMap: (m: Record<string, number>) => {
    columnWidths.value = { ...columnWidths.value, ...m }
    persistColumnWidths()
  },
  selectAllColumns,
  clearAllColumns,
  resetColumns,
  exportExcel,
  exportCsv,
  clearSelection,
  getSelectedRows: () => selectedRows.value
})
</script>

<style scoped>
.table-topbar {
  @apply mb-1 flex items-center justify-between gap-3;
  position: sticky;
  top: 0;
  z-index: 6;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  padding: 6px 8px;
}

.table-topbar-left,
.table-topbar-right {
  @apply flex min-w-0 items-center gap-2;
}

:global(html.dark) .table-topbar {
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
}

.tool-btn {
  @apply inline-flex h-9 w-9 items-center justify-center rounded-xl border border-slate-200/70 bg-white/70 text-slate-700 backdrop-blur-xl transition;
}

.tool-btn:hover {
  @apply bg-white text-slate-900;
}

:global(html.dark) .tool-btn {
  @apply border-slate-600/30 bg-slate-900/40 text-slate-200;
}

:global(html.dark) .tool-btn:hover {
  @apply bg-slate-900/60 text-slate-50;
}

.columns-panel {
  @apply w-[260px] p-3;
}

.columns-panel-actions {
  @apply flex flex-wrap gap-2;
}

.columns-panel-list {
  @apply flex flex-col gap-2;
}

.pager {
  @apply mt-2 flex justify-center text-xs;
}

.pager :deep(.el-pagination) {
  font-size: 12px;
}

.pager :deep(.el-pagination__jump) {
  font-size: 12px;
}

.pager :deep(.el-pagination__jump .el-input) {
  width: 48px;
}

.pager :deep(.el-pagination__jump .el-input__wrapper) {
  padding: 0 6px;
}

.pager :deep(.el-pagination__jump .el-input__inner) {
  height: 22px;
  line-height: 22px;
  font-size: 12px;
}

.fullscreen {
  @apply fixed inset-0 z-[9999] flex flex-col overflow-hidden bg-white/85 p-4 backdrop-blur-xl;
}

:global(html.dark) .fullscreen {
  @apply bg-slate-950/85;
}

.fullscreen-table-wrap {
  @apply flex-1 min-h-0 overflow-auto;
}

/* ── 居中列单元格内容居中 ── */
.enhanced-table :deep(.is-center-cell .cell) {
  display: flex;
  justify-content: center;
  align-items: center;
}

/* ── 消除 el-table 右侧空白 gutter 占位 ── */
.enhanced-table :deep(.el-table__header-wrapper) {
  overflow: hidden;
}

.enhanced-table :deep(.el-table .gutter) {
  display: none !important;
  width: 0 !important;
}

.enhanced-table :deep(.el-table colgroup col[name='gutter']) {
  display: none !important;
  width: 0 !important;
}

.enhanced-table :deep(.el-table__body) {
  width: 100% !important;
}

.enhanced-table :deep(.el-table__header) {
  width: 100% !important;
}

.enhanced-table :deep(.el-table--border),
.enhanced-table :deep(.el-table--group) {
  border-radius: 12px;
  overflow: hidden;
}

.enhanced-table :deep(.k8s-act-group) {
  width: 100%;
  justify-content: center;
  flex-wrap: nowrap;
}

/* ── 空状态 ── */
.enhanced-table-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  user-select: none;
  color: var(--color-text-muted, #94a3b8);
}

.enhanced-table-empty-svg {
  width: 120px;
  height: 90px;
  margin-bottom: 12px;
  color: var(--color-text-muted, #94a3b8);
}

.enhanced-table-empty-text {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-muted, #94a3b8);
  letter-spacing: 0.02em;
}
</style>
