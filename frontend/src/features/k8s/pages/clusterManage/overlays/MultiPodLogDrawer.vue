<template>
  <el-drawer
    v-model="visible"
    class="multi-pod-log-drawer"
    size="94%"
    destroy-on-close
    :with-header="false"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <div class="multi-pod-log__shell">
      <div class="multi-pod-log__header">
        <div class="multi-pod-log__title-wrap">
          <div class="multi-pod-log__title">多 Pod 日志工作台</div>
          <div class="multi-pod-log__meta">{{ headerMeta }}</div>
        </div>
        <el-space wrap>
          <el-select v-model="layoutMode" class="multi-pod-log__layout">
            <el-option v-for="item in layoutOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-select v-model="tailLines" class="multi-pod-log__tail" @change="restartAll">
            <el-option v-for="count in tailLineOptions" :key="count" :label="`尾部 ${count} 行`" :value="count" />
          </el-select>
          <el-switch v-model="liveMode" inline-prompt active-text="实时" inactive-text="快照" @change="restartAll" />
          <el-switch v-model="autoScroll" inline-prompt active-text="自动滚动" inactive-text="手动" />
          <el-button @click="resetLayout">重排布局</el-button>
          <el-button :icon="RefreshRight" :loading="hasConnectingSources" @click="restartAll">刷新全部</el-button>
          <el-button :icon="Delete" :disabled="!hasBufferedLogs" @click="clearLogs">清空全部</el-button>
          <el-button :icon="CopyDocument" :disabled="!fullText.trim()" @click="copyAll">复制全部</el-button>
          <el-button :icon="Download" :disabled="!fullText.trim()" @click="downloadAll">下载 .txt</el-button>
          <el-button :icon="Close" @click="close">关闭</el-button>
        </el-space>
      </div>

      <div class="multi-pod-log__statusbar">
        <div class="multi-pod-log__statusbar-main">
          <el-tag size="small" :type="statusTagType(summaryTone)" :class="['multi-pod-log__status-tag', `multi-pod-log__status-tag--${summaryTone}`]">{{ statusSummary }}</el-tag>
          <span>{{ layoutSummary }}</span>
        </div>
        <span>当前缓存 {{ totalBufferedLineCount }} 行，每窗最多 {{ MAX_LOG_LINES_PER_SOURCE }} 行</span>
      </div>

      <div class="multi-pod-log__body">
        <aside class="multi-pod-log__sources" :style="{ width: `${sidebarWidth}px` }">
          <div class="multi-pod-log__sources-head">
            <span>日志源</span>
            <el-tag size="small" type="info">{{ sources.length }} 个</el-tag>
          </div>
          <div class="multi-pod-log__sources-hint">点击左侧卡片可定位对应日志窗，拖拽中缝可调整侧栏宽度。</div>
          <el-scrollbar class="multi-pod-log__sources-scroll">
            <div class="multi-pod-log__sources-list">
              <article
                v-for="source in sources"
                :key="source.id"
                :class="['multi-pod-log__source-item', focusedSourceId === source.id ? 'multi-pod-log__source-item--active' : '']"
                @click="focusPanel(source.id)"
              >
                <div class="multi-pod-log__source-main">
                  <div class="multi-pod-log__source-name">{{ source.ns }}/{{ source.name }}</div>
                  <el-tag size="small" :type="statusTagType(statusTone(source.status))" :class="['multi-pod-log__status-tag', `multi-pod-log__status-tag--${statusTone(source.status)}`]">{{ source.status }}</el-tag>
                </div>
                <div class="multi-pod-log__source-meta">
                  <span>{{ source.container ? `容器 ${source.container}` : '默认容器' }}</span>
                  <span>已缓存 {{ getBufferedLineCount(source) }} 行</span>
                </div>
              </article>
            </div>
          </el-scrollbar>
        </aside>

        <div class="multi-pod-log__sidebar-resizer" @mousedown.prevent="startSidebarResize" />

        <section class="multi-pod-log__workspace-wrap">
          <div v-if="sources.length === 0" class="multi-pod-log__empty">
            <el-empty description="请选择 Pod 后再打开日志工作台" />
          </div>
          <div v-else ref="workspaceScrollRef" class="multi-pod-log__workspace-scroll">
            <div class="multi-pod-log__workspace">
              <article
                v-for="source in sources"
                :key="source.id"
                :ref="(el) => setPanelRef(source.id, el as HTMLElement | null)"
                :class="['multi-pod-log__panel', focusedSourceId === source.id ? 'multi-pod-log__panel--focused' : '']"
                :style="panelStyle(source)"
                @click="focusedSourceId = source.id"
              >
                <div class="multi-pod-log__panel-head">
                  <div class="multi-pod-log__panel-title-wrap">
                    <div class="multi-pod-log__panel-title">{{ source.name }}</div>
                    <div class="multi-pod-log__panel-meta">{{ source.ns }}</div>
                  </div>
                  <div class="multi-pod-log__panel-toolbar">
                    <el-select
                      v-if="source.containers.length > 0"
                      v-model="source.container"
                      size="small"
                      class="multi-pod-log__panel-container"
                      :disabled="source.containers.length <= 1"
                      @change="restartSource(source)"
                    >
                      <el-option v-for="container in source.containers" :key="container" :label="container" :value="container" />
                    </el-select>
                    <span v-else class="multi-pod-log__panel-container-label">默认容器</span>
                    <el-tooltip content="刷新当前窗口" placement="top" :show-after="250">
                      <el-button size="small" text :icon="RefreshRight" @click.stop="restartSource(source)" />
                    </el-tooltip>
                    <el-tooltip content="复制当前窗口" placement="top" :show-after="250">
                      <el-button size="small" text :icon="CopyDocument" @click.stop="copySource(source)" />
                    </el-tooltip>
                    <el-tooltip content="清空当前窗口" placement="top" :show-after="250">
                      <el-button size="small" text :icon="Delete" @click.stop="clearSourceLogs(source)" />
                    </el-tooltip>
                  </div>
                </div>

                <div class="multi-pod-log__panel-status">
                  <el-tag size="small" :type="statusTagType(statusTone(source.status))" :class="['multi-pod-log__status-tag', `multi-pod-log__status-tag--${statusTone(source.status)}`]">{{ source.status }}</el-tag>
                  <span>{{ getBufferedLineCount(source) }} 行</span>
                </div>

                <div
                  :ref="(el) => setViewportRef(source.id, el as HTMLElement | null)"
                  class="multi-pod-log__panel-viewport"
                >
                  <div v-if="getBufferedLineCount(source) === 0" class="multi-pod-log__panel-empty">
                    {{ source.connecting ? (liveMode ? '正在接入实时日志…' : '正在加载日志快照…') : '当前窗口暂无日志内容' }}
                  </div>
                  <div v-else class="multi-pod-log__panel-lines">
                    <div v-for="(line, index) in getDisplayLines(source)" :key="`${source.id}:${index}`" class="multi-pod-log__panel-line">
                      <span class="multi-pod-log__panel-line-no">{{ index + 1 }}</span>
                      <span class="multi-pod-log__panel-line-text">{{ line || ' ' }}</span>
                    </div>
                  </div>
                </div>

                <div class="multi-pod-log__panel-footer">拖拽右下角可调整窗口大小</div>
                <div class="multi-pod-log__panel-resizer" @mousedown.stop.prevent="startPanelResize(source, $event)" />
              </article>
            </div>
          </div>
        </section>
      </div>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Close, CopyDocument, Delete, Download, RefreshRight } from '@element-plus/icons-vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import { copyToClipboard, downloadBlob, sanitizeFileName } from '@/shared/utils/text'
import { buildTerminalWebSocketUrl } from '@/shared/utils/terminal'

const props = defineProps<{ clusterId: number }>()

type MultiPodLogTarget = {
  ns: string
  name: string
  container?: string
  containers?: string[]
}

type PodLogFrame = {
  type?: string
  data?: string
  message?: string
}

type LayoutMode = 'auto' | 'columns-2' | 'columns-3' | 'stack'

type LogSource = {
  id: string
  ns: string
  name: string
  containers: string[]
  container: string
  connecting: boolean
  status: string
  remainder: string
  lines: string[]
  ws: WebSocket | null
  seq: number
  width: number
  height: number
  manualSize: boolean
}

const MAX_LOG_LINES_PER_SOURCE = 3000
const PANEL_GAP = 12
const PANEL_MIN_WIDTH = 360
const PANEL_MIN_HEIGHT = 240
const PANEL_MAX_HEIGHT = 760
const SIDEBAR_MIN_WIDTH = 236
const SIDEBAR_MAX_WIDTH = 420
const tailLineOptions = [100, 200, 500, 1000, 2000]
const layoutOptions: Array<{ label: string; value: LayoutMode }> = [
  { label: '自适应', value: 'auto' },
  { label: '双列并排', value: 'columns-2' },
  { label: '三列并排', value: 'columns-3' },
  { label: '纵向堆叠', value: 'stack' }
]

const visible = ref(false)
const liveMode = ref(true)
const autoScroll = ref(true)
const tailLines = ref(200)
const layoutMode = ref<LayoutMode>('auto')
const sources = ref<LogSource[]>([])
const focusedSourceId = ref('')
const sidebarWidth = ref(276)
const workspaceScrollRef = ref<HTMLElement | null>(null)
const workspaceWidth = ref(0)

const panelRefs = new Map<string, HTMLElement>()
const viewportRefs = new Map<string, HTMLElement>()
let workspaceObserver: ResizeObserver | null = null
let cleanupDragListeners: (() => void) | null = null

const headerMeta = computed(() => {
  if (sources.value.length === 0) return '-'
  const modeText = liveMode.value ? '实时模式' : '快照模式'
  return `${sources.value.length} 个 Pod · 独立分屏工作台 · ${modeText}`
})

const hasConnectingSources = computed(() => sources.value.some((source) => source.connecting))
const failedSourceCount = computed(() => sources.value.filter((source) => /失败|异常/.test(source.status)).length)
const totalBufferedLineCount = computed(() => sources.value.reduce((sum, source) => sum + getBufferedLineCount(source), 0))
const hasBufferedLogs = computed(() => totalBufferedLineCount.value > 0)
const summaryTone = computed(() => {
  if (hasConnectingSources.value) return 'warning'
  if (failedSourceCount.value > 0) return 'danger'
  if (sources.value.length === 0) return 'neutral'
  return 'active'
})
const resolvedColumnCount = computed(() => {
  const count = sources.value.length
  const width = workspaceWidth.value || 1280
  if (layoutMode.value === 'stack') return 1
  if (layoutMode.value === 'columns-2') return width < 860 ? 1 : 2
  if (layoutMode.value === 'columns-3') {
    if (width < 860) return 1
    if (width < 1360) return 2
    return 3
  }
  if (count <= 1) return 1
  if (width < 860) return 1
  if (count === 2) return 2
  return width >= 1480 ? 3 : 2
})
const layoutSummary = computed(() => `${layoutLabel(layoutMode.value)} · 当前 ${resolvedColumnCount.value} 列 · 可拖拽中缝和窗体右下角调整尺寸`)
const statusSummary = computed(() => {
  if (sources.value.length === 0) return '未选择日志源'
  const activeCount = sources.value.filter((source) => /实时中|快照中|已加载|已结束/.test(source.status)).length
  if (hasConnectingSources.value) return `正在连接 ${activeCount}/${sources.value.length}`
  if (failedSourceCount.value > 0) return `${activeCount}/${sources.value.length} 个窗口可用，${failedSourceCount.value} 个窗口异常`
  return liveMode.value ? `${activeCount}/${sources.value.length} 个窗口实时跟随` : `${activeCount}/${sources.value.length} 个窗口快照已加载`
})
const fullText = computed(() => {
  return sources.value
    .map((source) => {
      const text = getSourceText(source).trim()
      if (!text) return ''
      return `===== ${sourceLabel(source)} =====\n${text}`
    })
    .filter(Boolean)
    .join('\n\n')
})

function open(nextTargets: MultiPodLogTarget[]) {
  sources.value = normalizeTargets(nextTargets)
  focusedSourceId.value = sources.value[0]?.id ?? ''
  visible.value = true
  void nextTick(async () => {
    initWorkspaceObserver()
    resetLayout()
    await restartAll()
  })
}

function close() {
  visible.value = false
}

defineExpose({ open, close })

function normalizeTargets(nextTargets: MultiPodLogTarget[]): LogSource[] {
  const map = new Map<string, LogSource>()
  for (const item of Array.isArray(nextTargets) ? nextTargets : []) {
    const ns = String(item?.ns ?? '').trim()
    const name = String(item?.name ?? '').trim()
    if (!ns || !name) continue
    const id = `${ns}/${name}`
    const containers = Array.from(new Set((Array.isArray(item?.containers) ? item.containers : []).map((value) => String(value ?? '').trim()).filter(Boolean)))
    const preferredContainer = String(item?.container ?? '').trim()
    const container = preferredContainer && containers.includes(preferredContainer) ? preferredContainer : containers[0] ?? ''
    map.set(id, {
      id,
      ns,
      name,
      containers,
      container,
      connecting: false,
      status: '未连接',
      remainder: '',
      lines: [],
      ws: null,
      seq: 0,
      width: PANEL_MIN_WIDTH,
      height: 320,
      manualSize: false
    })
  }
  return Array.from(map.values())
}

function layoutLabel(value: LayoutMode): string {
  return layoutOptions.find((item) => item.value === value)?.label ?? '自适应'
}

function sourceLabel(source: Pick<LogSource, 'ns' | 'name' | 'container'>) {
  return source.container ? `${source.ns}/${source.name} · ${source.container}` : `${source.ns}/${source.name}`
}

function statusTone(status: string): 'neutral' | 'active' | 'warning' | 'danger' {
  if (/失败|异常/.test(status)) return 'danger'
  if (/连接中|加载中|快照中/.test(status)) return 'warning'
  if (/实时中|已加载|已结束/.test(status)) return 'active'
  return 'neutral'
}

function statusTagType(tone: ReturnType<typeof statusTone>): 'info' | 'warning' | 'danger' {
  if (tone === 'danger') return 'danger'
  if (tone === 'warning') return 'warning'
  return 'info'
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function getBufferedLineCount(source: LogSource): number {
  return source.lines.length + (source.remainder ? 1 : 0)
}

function getDisplayLines(source: LogSource): string[] {
  return source.remainder ? source.lines.concat(source.remainder) : source.lines
}

function getSourceText(source: LogSource): string {
  return getDisplayLines(source).join('\n')
}

function setPanelRef(id: string, el: HTMLElement | null) {
  if (el) panelRefs.set(id, el)
  else panelRefs.delete(id)
}

function setViewportRef(id: string, el: HTMLElement | null) {
  if (el) viewportRefs.set(id, el)
  else viewportRefs.delete(id)
}

function focusPanel(id: string) {
  focusedSourceId.value = id
  void nextTick(() => {
    panelRefs.get(id)?.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' })
  })
}

function initWorkspaceObserver() {
  if (!workspaceScrollRef.value) return
  workspaceWidth.value = workspaceScrollRef.value.clientWidth
  if (workspaceObserver || typeof ResizeObserver === 'undefined') return
  workspaceObserver = new ResizeObserver(() => {
    workspaceWidth.value = workspaceScrollRef.value?.clientWidth ?? 0
  })
  workspaceObserver.observe(workspaceScrollRef.value)
}

function destroyWorkspaceObserver() {
  workspaceObserver?.disconnect()
  workspaceObserver = null
}

function destroyDragListeners() {
  cleanupDragListeners?.()
}

function withWindowDrag(onMove: (event: MouseEvent) => void) {
  destroyDragListeners()
  const handleMove = (event: MouseEvent) => onMove(event)
  const handleUp = () => destroyDragListeners()
  window.addEventListener('mousemove', handleMove)
  window.addEventListener('mouseup', handleUp, { once: true })
  cleanupDragListeners = () => {
    window.removeEventListener('mousemove', handleMove)
    window.removeEventListener('mouseup', handleUp)
    cleanupDragListeners = null
  }
}

function getDefaultPanelWidth(columns = resolvedColumnCount.value): number {
  const available = Math.max(PANEL_MIN_WIDTH, (workspaceWidth.value || 1280) - PANEL_GAP * Math.max(0, columns - 1))
  return Math.max(PANEL_MIN_WIDTH, Math.floor(available / columns))
}

function getDefaultPanelHeight(columns = resolvedColumnCount.value): number {
  if (columns <= 1) return 420
  if (columns === 2) return 340
  return 300
}

function maxPanelWidth(): number {
  return Math.max(PANEL_MIN_WIDTH, (workspaceWidth.value || 1280) - 24)
}

function normalizePanelSize(source: LogSource) {
  source.width = clamp(source.width || getDefaultPanelWidth(), PANEL_MIN_WIDTH, maxPanelWidth())
  source.height = clamp(source.height || getDefaultPanelHeight(), PANEL_MIN_HEIGHT, PANEL_MAX_HEIGHT)
}

function applyLayoutPreset(force = true) {
  const width = getDefaultPanelWidth()
  const height = getDefaultPanelHeight()
  sources.value.forEach((source) => {
    if (!force && source.manualSize) {
      normalizePanelSize(source)
      return
    }
    source.width = width
    source.height = height
    source.manualSize = false
  })
}

function resetLayout() {
  applyLayoutPreset(true)
}

function panelStyle(source: LogSource) {
  return {
    width: `${clamp(source.width || getDefaultPanelWidth(), PANEL_MIN_WIDTH, maxPanelWidth())}px`,
    height: `${clamp(source.height || getDefaultPanelHeight(), PANEL_MIN_HEIGHT, PANEL_MAX_HEIGHT)}px`
  }
}

function startSidebarResize(event: MouseEvent) {
  if (window.innerWidth <= 1080) return
  const startX = event.clientX
  const startWidth = sidebarWidth.value
  withWindowDrag((moveEvent) => {
    const maxWidth = Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, window.innerWidth * 0.36))
    sidebarWidth.value = clamp(startWidth + (moveEvent.clientX - startX), SIDEBAR_MIN_WIDTH, maxWidth)
  })
}

function startPanelResize(source: LogSource, event: MouseEvent) {
  focusedSourceId.value = source.id
  const startX = event.clientX
  const startY = event.clientY
  const startWidth = source.width || getDefaultPanelWidth()
  const startHeight = source.height || getDefaultPanelHeight()
  withWindowDrag((moveEvent) => {
    source.manualSize = true
    source.width = clamp(startWidth + (moveEvent.clientX - startX), PANEL_MIN_WIDTH, maxPanelWidth())
    source.height = clamp(startHeight + (moveEvent.clientY - startY), PANEL_MIN_HEIGHT, PANEL_MAX_HEIGHT)
  })
}

function clearSourceLogs(source: LogSource) {
  source.lines = []
  source.remainder = ''
  const viewport = viewportRefs.get(source.id)
  if (viewport) viewport.scrollTop = 0
}

function clearLogs() {
  sources.value.forEach((source) => clearSourceLogs(source))
}

async function copyAll() {
  if (!fullText.value.trim()) return
  try {
    await copyToClipboard(fullText.value)
    notifySuccess('工作台日志已复制')
  } catch {
    notifyError('复制日志失败')
  }
}

async function copySource(source: LogSource) {
  const text = getSourceText(source)
  if (!text.trim()) return
  try {
    await copyToClipboard(text)
    notifySuccess(`${source.name} 日志已复制`)
  } catch {
    notifyError('复制日志失败')
  }
}

function downloadAll() {
  if (!fullText.value.trim()) return
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-')
  downloadBlob(`pod_log_workbench_${sanitizeFileName(String(props.clusterId))}_${timestamp}.txt`, new Blob([fullText.value], { type: 'text/plain;charset=utf-8' }))
}

function scrollPanelToBottom(sourceId: string) {
  if (!autoScroll.value) return
  void nextTick(() => {
    const viewport = viewportRefs.get(sourceId)
    if (!viewport) return
    viewport.scrollTop = viewport.scrollHeight
  })
}

function trimSourceLines(lines: string[]): string[] {
  if (lines.length <= MAX_LOG_LINES_PER_SOURCE) return lines
  return lines.slice(lines.length - MAX_LOG_LINES_PER_SOURCE)
}

function appendLines(source: LogSource, lines: string[]) {
  if (lines.length === 0) return
  source.lines = trimSourceLines(source.lines.concat(lines))
  scrollPanelToBottom(source.id)
}

function appendChunk(source: LogSource, chunk: string) {
  const normalized = String(chunk || '').replace(/\r\n/g, '\n').replace(/\r/g, '\n')
  if (!normalized) return
  const combined = `${source.remainder}${normalized}`
  const endsWithNewLine = combined.endsWith('\n')
  const parts = combined.split('\n')
  if (endsWithNewLine && parts.length > 0 && parts[parts.length - 1] === '') {
    parts.pop()
    source.remainder = ''
  } else {
    source.remainder = parts.pop() ?? ''
  }
  appendLines(source, parts)
}

function flushRemainder(source: LogSource) {
  if (!source.remainder) return
  const remainder = source.remainder
  source.remainder = ''
  appendLines(source, [remainder])
}

function closeSourceSocket(source: LogSource, reason = 'client_close') {
  const currentWs = source.ws
  source.ws = null
  if (!currentWs) return
  try {
    currentWs.onopen = null
    currentWs.onclose = null
    currentWs.onerror = null
    currentWs.onmessage = null
    currentWs.close(1000, reason)
  } catch {
    // ignore
  }
}

function closeAllSockets(reason = 'client_close') {
  sources.value.forEach((source) => {
    source.seq += 1
    closeSourceSocket(source, reason)
    source.connecting = false
  })
}

function buildPodLogWsUrl(raw: string): string | null {
  const value = String(raw || '').trim()
  if (!value) return null
  const parsed = /^https?:\/\//.test(value) || /^wss?:\/\//.test(value)
    ? new URL(value)
    : new URL(value.startsWith('/') ? value : `/${value}`, window.location.origin)
  return buildTerminalWebSocketUrl(parsed.pathname, Object.fromEntries(parsed.searchParams.entries()))
}

function handleSourceFrame(source: LogSource, rawData: unknown) {
  let frame: PodLogFrame | null = null
  try {
    frame = JSON.parse(String(rawData || '')) as PodLogFrame
  } catch {
    frame = { type: 'chunk', data: String(rawData || '') }
  }
  const frameType = String(frame?.type || '').toLowerCase()
  if (frameType === 'chunk') {
    appendChunk(source, String(frame?.data || ''))
    return
  }
  if (frameType === 'eof') {
    flushRemainder(source)
    source.connecting = false
    source.status = liveMode.value ? '已结束' : '已加载'
    return
  }
  if (frameType === 'error') {
    source.connecting = false
    source.status = '日志异常'
    notifyError(`${source.ns}/${source.name}：${String(frame?.message || '日志流连接失败')}`)
  }
}

async function startSource(source: LogSource) {
  source.seq += 1
  const seq = source.seq
  closeSourceSocket(source, 'restart')
  clearSourceLogs(source)
  source.connecting = true
  source.status = liveMode.value ? '连接中' : '加载中'

  try {
    const session = await k8sApi.createPodLogSession(props.clusterId, source.ns, source.name, {
      container: source.container || undefined,
      follow: liveMode.value,
      tail_lines: tailLines.value
    })
    if (source.seq !== seq || !visible.value) return
    const wsUrl = buildPodLogWsUrl(session.ws_url)
    if (!wsUrl) throw new Error('日志 WebSocket 地址无效')

    const currentWs = new WebSocket(wsUrl)
    source.ws = currentWs

    currentWs.onopen = () => {
      if (source.ws !== currentWs || source.seq !== seq) return
      source.connecting = false
      source.status = liveMode.value ? '实时中' : '快照中'
    }

    currentWs.onmessage = (evt) => {
      if (source.ws !== currentWs || source.seq !== seq) return
      handleSourceFrame(source, evt.data)
    }

    currentWs.onerror = () => {
      if (source.ws !== currentWs || source.seq !== seq) return
      source.connecting = false
      source.status = '连接失败'
    }

    currentWs.onclose = () => {
      if (source.ws === currentWs) source.ws = null
      if (!visible.value || source.seq !== seq) return
      flushRemainder(source)
      source.connecting = false
      if (source.status === '连接中' || source.status === '加载中') {
        source.status = liveMode.value ? '连接已关闭' : '快照已关闭'
      }
    }
  } catch (error) {
    source.connecting = false
    source.status = '连接失败'
    const err = error as ApiError
    const message = err?.message ? (err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) : '创建日志会话失败'
    notifyError(`${source.ns}/${source.name}：${message}`)
  }
}

async function restartSource(source: LogSource) {
  if (!visible.value || !props.clusterId) return
  await startSource(source)
}

async function restartAll() {
  if (!visible.value || !props.clusterId || sources.value.length === 0) return
  closeAllSockets('restart')
  clearLogs()
  await Promise.all(sources.value.map((source) => startSource(source)))
}

function handleClosed() {
  closeAllSockets('drawer_closed')
  destroyWorkspaceObserver()
  destroyDragListeners()
  clearLogs()
  sources.value = []
  focusedSourceId.value = ''
  panelRefs.clear()
  viewportRefs.clear()
}

watch(visible, (value) => {
  if (!value) return
  void nextTick(() => {
    initWorkspaceObserver()
    applyLayoutPreset(false)
    sources.value.forEach((source) => scrollPanelToBottom(source.id))
  })
})

watch(autoScroll, (value) => {
  if (!value) return
  sources.value.forEach((source) => scrollPanelToBottom(source.id))
})

watch(layoutMode, () => {
  if (!visible.value) return
  resetLayout()
})

watch(workspaceWidth, (value, oldValue) => {
  if (!visible.value || value === oldValue) return
  applyLayoutPreset(false)
})

onBeforeUnmount(() => {
  closeAllSockets('component_unmount')
  destroyWorkspaceObserver()
  destroyDragListeners()
})
</script>

<style scoped>
.multi-pod-log-drawer :deep(.el-drawer__body) {
  padding: 0;
}

.multi-pod-log__shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  padding: 16px 18px 18px;
  background:
    radial-gradient(circle at top left, rgba(14, 165, 233, 0.14), transparent 24%),
    radial-gradient(circle at top right, rgba(59, 130, 246, 0.1), transparent 26%),
    linear-gradient(180deg, rgba(248, 250, 252, 0.98), rgba(241, 245, 249, 0.98));
}

.multi-pod-log__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.82);
  backdrop-filter: blur(10px);
}

.multi-pod-log__title-wrap {
  min-width: 0;
}

.multi-pod-log__title {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.multi-pod-log__meta {
  margin-top: 4px;
  color: #475569;
  font-size: 13px;
}

.multi-pod-log__layout {
  width: 136px;
}

.multi-pod-log__tail {
  width: 128px;
}

.multi-pod-log__statusbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 4px;
  color: #475569;
  font-size: 13px;
}

.multi-pod-log__statusbar-main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.multi-pod-log__status-tag {
  border: none;
}

.multi-pod-log__status-tag--neutral {
  color: #475569;
  background: rgba(148, 163, 184, 0.16);
}

.multi-pod-log__status-tag--active {
  color: #075985;
  background: rgba(14, 165, 233, 0.16);
}

.multi-pod-log__status-tag--warning {
  color: #9a3412;
  background: rgba(251, 191, 36, 0.2);
}

.multi-pod-log__status-tag--danger {
  color: #b91c1c;
  background: rgba(248, 113, 113, 0.16);
}

.multi-pod-log__body {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: stretch;
  gap: 0;
}

.multi-pod-log__sources,
.multi-pod-log__workspace-wrap {
  min-height: 0;
}

.multi-pod-log__sources {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 14px;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  background: rgba(255, 255, 255, 0.84);
  backdrop-filter: blur(8px);
}

.multi-pod-log__sources-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
}

.multi-pod-log__sources-hint {
  color: #64748b;
  font-size: 12px;
  line-height: 1.5;
}

.multi-pod-log__sources-scroll {
  flex: 1;
  min-height: 0;
}

.multi-pod-log__sources-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.multi-pod-log__source-item {
  padding: 12px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.96), rgba(241, 245, 249, 0.92));
  cursor: pointer;
  transition: border-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
}

.multi-pod-log__source-item:hover {
  transform: translateY(-1px);
  border-color: rgba(59, 130, 246, 0.26);
  box-shadow: 0 10px 22px rgba(15, 23, 42, 0.08);
}

.multi-pod-log__source-item--active {
  border-color: rgba(37, 99, 235, 0.32);
  background: linear-gradient(180deg, rgba(239, 246, 255, 0.98), rgba(226, 232, 240, 0.92));
  box-shadow: 0 14px 28px rgba(37, 99, 235, 0.12);
}

.multi-pod-log__source-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.multi-pod-log__source-name {
  min-width: 0;
  font-size: 13px;
  font-weight: 700;
  color: #0f172a;
  word-break: break-all;
}

.multi-pod-log__source-meta {
  margin-top: 6px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: #64748b;
}

.multi-pod-log__sidebar-resizer {
  flex: 0 0 12px;
  position: relative;
  cursor: col-resize;
}

.multi-pod-log__sidebar-resizer::before {
  content: '';
  position: absolute;
  left: 50%;
  top: 10px;
  bottom: 10px;
  width: 2px;
  border-radius: 999px;
  background: linear-gradient(180deg, rgba(148, 163, 184, 0.08), rgba(148, 163, 184, 0.42), rgba(148, 163, 184, 0.08));
  transform: translateX(-50%);
}

.multi-pod-log__workspace-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  border-radius: 18px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(248, 250, 252, 0.78);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72), 0 24px 60px rgba(15, 23, 42, 0.08);
}

.multi-pod-log__workspace-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.multi-pod-log__workspace {
  display: flex;
  flex-wrap: wrap;
  align-content: flex-start;
  gap: 12px;
  min-width: 0;
}

.multi-pod-log__empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.multi-pod-log__panel {
  position: relative;
  flex: 0 0 auto;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 18px;
  border: 1px solid rgba(15, 23, 42, 0.1);
  background: rgba(255, 255, 255, 0.98);
  box-shadow: 0 18px 42px rgba(15, 23, 42, 0.08);
}

.multi-pod-log__panel--focused {
  border-color: rgba(37, 99, 235, 0.32);
  box-shadow: 0 20px 46px rgba(37, 99, 235, 0.12);
}

.multi-pod-log__panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px 10px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.16);
}

.multi-pod-log__panel-title-wrap {
  min-width: 0;
}

.multi-pod-log__panel-title {
  font-size: 14px;
  font-weight: 800;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.multi-pod-log__panel-meta {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.multi-pod-log__panel-toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
}

.multi-pod-log__panel-container {
  width: 140px;
}

.multi-pod-log__panel-container-label {
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  padding: 0 6px;
}

.multi-pod-log__panel-status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 14px;
  background: rgba(248, 250, 252, 0.92);
  color: #64748b;
  font-size: 12px;
}

.multi-pod-log__panel-viewport {
  flex: 1;
  min-height: 0;
  overflow: auto;
  font-family: "JetBrains Mono", "Cascadia Code", Consolas, monospace;
  font-size: 12px;
  line-height: 22px;
  color: #e2e8f0;
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.96), rgba(2, 6, 23, 0.98)),
    radial-gradient(circle at top left, rgba(14, 165, 233, 0.14), transparent 24%);
}

.multi-pod-log__panel-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100%;
  padding: 24px;
  color: #94a3b8;
  font-size: 13px;
}

.multi-pod-log__panel-lines {
  display: flex;
  flex-direction: column;
}

.multi-pod-log__panel-line {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr);
  gap: 12px;
  padding: 0 14px;
  min-height: 22px;
  align-items: start;
  white-space: pre-wrap;
}

.multi-pod-log__panel-line:nth-child(2n) {
  background: rgba(255, 255, 255, 0.02);
}

.multi-pod-log__panel-line-no {
  color: #64748b;
  text-align: right;
  user-select: none;
}

.multi-pod-log__panel-line-text {
  min-width: 0;
  word-break: break-word;
}

.multi-pod-log__panel-footer {
  padding: 7px 14px 8px;
  border-top: 1px solid rgba(148, 163, 184, 0.12);
  color: #64748b;
  font-size: 11px;
  background: rgba(248, 250, 252, 0.92);
}

.multi-pod-log__panel-resizer {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 18px;
  height: 18px;
  cursor: nwse-resize;
}

.multi-pod-log__panel-resizer::before {
  content: '';
  position: absolute;
  right: 4px;
  bottom: 4px;
  width: 10px;
  height: 10px;
  border-right: 2px solid rgba(148, 163, 184, 0.72);
  border-bottom: 2px solid rgba(148, 163, 184, 0.72);
  border-bottom-right-radius: 3px;
}

@media (max-width: 1280px) {
  .multi-pod-log__header {
    flex-direction: column;
  }

  .multi-pod-log__statusbar {
    flex-direction: column;
    align-items: flex-start;
  }
}

@media (max-width: 1080px) {
  .multi-pod-log__body {
    flex-direction: column;
  }

  .multi-pod-log__sources {
    width: 100% !important;
    max-height: 260px;
  }

  .multi-pod-log__sidebar-resizer {
    display: none;
  }

  .multi-pod-log__panel {
    width: 100% !important;
    min-width: 0;
  }

  .multi-pod-log__panel-resizer {
    display: none;
  }
}

@media (max-width: 720px) {
  .multi-pod-log__panel-head {
    flex-direction: column;
  }

  .multi-pod-log__panel-toolbar {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
  }

  .multi-pod-log__panel-container {
    width: 100%;
  }
}
</style>