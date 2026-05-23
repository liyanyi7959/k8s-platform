<template>
  <div :class="['multi-pod-log__shell', props.compact ? 'multi-pod-log__shell--compact' : '']">
    <div class="multi-pod-log__header">
      <div class="multi-pod-log__title-wrap">
        <div class="multi-pod-log__title">{{ props.title }}</div>
        <div v-if="showHeaderMeta" class="multi-pod-log__meta">{{ headerMeta }}</div>
      </div>
      <el-space wrap>
        <el-select v-model="tailLines" class="multi-pod-log__tail" @change="restartCurrentSource">
          <el-option v-for="count in tailLineOptions" :key="count" :label="`尾部 ${count} 行`" :value="count" />
        </el-select>
        <el-switch v-model="liveMode" inline-prompt active-text="实时" inactive-text="快照" @change="restartCurrentSource" />
        <el-switch v-model="autoScroll" inline-prompt active-text="自动滚动" inactive-text="手动" />
        <el-button :icon="RefreshRight" :loading="isActiveSourceBusy" @click="restartCurrentSource">刷新当前</el-button>
        <el-button :icon="Delete" :disabled="!hasBufferedLogs" @click="clearLogs">清空全部</el-button>
        <el-button :icon="CopyDocument" :disabled="!fullText.trim()" @click="copyAll">复制全部</el-button>
        <el-button :icon="Download" :disabled="!fullText.trim()" @click="downloadAll">下载 .txt</el-button>
        <el-button v-if="props.showCloseButton" :icon="Close" @click="emit('request-close')">关闭</el-button>
      </el-space>
    </div>

    <div v-if="showStatusBar" class="multi-pod-log__statusbar">
      <div class="multi-pod-log__statusbar-main">
        <el-tag size="small" :type="statusTagType(summaryTone)" :class="['multi-pod-log__status-tag', `multi-pod-log__status-tag--${summaryTone}`]">{{ statusSummary }}</el-tag>
        <span class="multi-pod-log__statusbar-hint">{{ navigationHint }}</span>
      </div>
      <span class="multi-pod-log__statusbar-meta">{{ totalBufferedLineCount }} 行</span>
    </div>

    <div v-if="sources.length === 0" class="multi-pod-log__empty">
      <el-empty description="请选择 Pod 后再打开日志工作台" />
    </div>

    <div
      v-else
      :class="[
        'multi-pod-log__body',
        props.navigationMode === 'side' ? 'multi-pod-log__body--side' : '',
        props.compact && props.navigationMode === 'top' ? 'multi-pod-log__body--wrap-tabs' : ''
      ]"
    >
      <section class="multi-pod-log__tabs-card">
        <div class="multi-pod-log__tabs-scroll">
          <div class="multi-pod-log__tabs">
            <button
              v-for="source in sources"
              :key="source.id"
              type="button"
              :class="[
                'multi-pod-log__tab',
                activeSourceId === source.id ? 'multi-pod-log__tab--active' : '',
                draggingSourceId === source.id ? 'multi-pod-log__tab--dragging' : '',
                dragOverSourceId === source.id ? 'multi-pod-log__tab--drag-over' : ''
              ]"
              :title="`${source.ns}/${source.name} · ${source.status}`"
              :draggable="sources.length > 1"
              @click="activateSource(source.id)"
              @dragstart="onTabDragStart($event, source.id)"
              @dragover.prevent="onTabDragOver($event, source.id)"
              @drop.prevent="onTabDrop($event, source.id)"
              @dragend="onTabDragEnd"
              @contextmenu.prevent="openTabContextMenu($event, source.id)"
            >
              <span :class="['multi-pod-log__tab-dot', `multi-pod-log__tab-dot--${statusTone(source.status)}`]"></span>
              <span class="multi-pod-log__tab-title">{{ source.name }}</span>
            </button>
          </div>
        </div>
      </section>

      <section v-if="activeSource" class="multi-pod-log__viewer">
        <div class="multi-pod-log__viewer-head">
          <div class="multi-pod-log__viewer-summary">
            <div class="multi-pod-log__viewer-title">{{ activeSource.name }}</div>
            <span class="multi-pod-log__viewer-divider">/</span>
            <span class="multi-pod-log__viewer-namespace">{{ activeSource.ns }}</span>
            <el-tag size="small" :type="statusTagType(activeSourceTone)" :class="['multi-pod-log__status-tag', `multi-pod-log__status-tag--${activeSourceTone}`]">{{ activeSource.status }}</el-tag>
            <span class="multi-pod-log__viewer-count">{{ activeLineCount }} 行</span>
          </div>
          <div class="multi-pod-log__viewer-toolbar">
            <el-select
              v-if="activeSource.containers.length > 0"
              v-model="activeSource.container"
              size="small"
              class="multi-pod-log__viewer-container"
              :disabled="activeSource.containers.length <= 1"
              @change="restartSource(activeSource)"
            >
              <el-option v-for="container in activeSource.containers" :key="container" :label="container" :value="container" />
            </el-select>
            <span v-else class="multi-pod-log__viewer-container-label">默认容器</span>
            <el-button size="small" :icon="RefreshRight" @click="restartSource(activeSource)">刷新当前</el-button>
            <el-button size="small" :icon="CopyDocument" :disabled="!activeSourceText.trim()" @click="copySource(activeSource)">复制当前</el-button>
            <el-button size="small" :icon="Delete" :disabled="activeLineCount === 0" @click="clearSourceLogs(activeSource)">清空当前</el-button>
          </div>
        </div>

        <div ref="activeViewportRef" class="multi-pod-log__viewer-viewport" @scroll="onActiveViewportScroll">
          <div v-if="activeLines.length === 0" class="multi-pod-log__viewer-empty">
            {{ activeSource.connecting ? (liveMode ? '正在接入实时日志…' : '正在加载日志快照…') : '当前 Pod 暂无日志内容' }}
          </div>
          <template v-else>
            <div class="multi-pod-log__viewer-spacer" :style="{ height: `${activeTotalHeight}px` }"></div>
            <div class="multi-pod-log__viewer-visible" :style="{ transform: `translateY(${activeOffsetTop}px)` }">
              <div v-for="item in visibleActiveLines" :key="`${activeSource.id}:${item.number}`" class="multi-pod-log__viewer-line">
                <span class="multi-pod-log__viewer-line-no">{{ item.number }}</span>
                <span class="multi-pod-log__viewer-line-text">{{ item.text || ' ' }}</span>
              </div>
            </div>
          </template>
        </div>
      </section>
    </div>

    <teleport to="body">
      <div v-if="contextMenuVisible" class="multi-pod-log__context-mask" @click="closeTabContextMenu" @contextmenu.prevent="closeTabContextMenu"></div>
      <div v-if="contextMenuSource" class="multi-pod-log__context-menu" :style="contextMenuStyle" @click.stop @contextmenu.prevent>
        <button type="button" class="multi-pod-log__context-item" :disabled="sources.length <= 1" @click="closeOtherTabs(contextMenuSource.id)">关闭其他</button>
        <button type="button" class="multi-pod-log__context-item multi-pod-log__context-item--danger" @click="closeSourceTab(contextMenuSource.id)">关闭当前</button>
      </div>
    </teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Close, CopyDocument, Delete, Download, RefreshRight } from '@element-plus/icons-vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import { copyToClipboard, downloadBlob, sanitizeFileName } from '@/shared/utils/text'
import { buildTerminalWebSocketUrl } from '@/shared/utils/terminal'

const props = withDefaults(defineProps<{
  clusterId: number
  title?: string
  showCloseButton?: boolean
  compact?: boolean
  navigationMode?: 'top' | 'side'
}>(), {
  title: '多 Pod 日志工作台',
  showCloseButton: false,
  compact: false,
  navigationMode: 'top'
})

const emit = defineEmits<{
  (e: 'request-close'): void
}>()

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

type StatusTone = 'neutral' | 'active' | 'warning' | 'danger'

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
}

const MAX_LOG_LINES_PER_SOURCE = 3000
const LINE_HEIGHT = 22
const OVERSCAN = 16
const tailLineOptions = [100, 200, 500, 1000, 2000]

const liveMode = ref(true)
const autoScroll = ref(true)
const tailLines = ref(200)
const sources = ref<LogSource[]>([])
const activeSourceId = ref('')
const activeViewportRef = ref<HTMLElement | null>(null)
const activeScrollTop = ref(0)
const activeViewportHeight = ref(0)
const contextMenuVisible = ref(false)
const contextMenuSourceId = ref('')
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const draggingSourceId = ref('')
const dragOverSourceId = ref('')

let activeViewportObserver: ResizeObserver | null = null
let destroyed = false

const activeSource = computed<LogSource | null>(() => {
  if (sources.value.length === 0) return null
  return sources.value.find((source) => source.id === activeSourceId.value) ?? sources.value[0] ?? null
})
const contextMenuSource = computed<LogSource | null>(() => {
  if (!contextMenuSourceId.value) return null
  return sources.value.find((source) => source.id === contextMenuSourceId.value) ?? null
})
const contextMenuStyle = computed(() => {
  if (typeof window === 'undefined') return {}
  const menuWidth = 176
  const menuHeight = 112
  const left = Math.max(12, Math.min(contextMenuX.value, window.innerWidth - menuWidth - 12))
  const top = Math.max(12, Math.min(contextMenuY.value, window.innerHeight - menuHeight - 12))
  return {
    left: `${left}px`,
    top: `${top}px`
  }
})

const headerMeta = computed(() => {
  if (sources.value.length === 0) return '-'
  const modeText = liveMode.value ? '实时模式' : '快照模式'
  const navigationText = props.navigationMode === 'side' ? '侧边切换' : (props.compact ? '顶部折叠切换' : '顶部切换')
  return `${sources.value.length} 个 Pod · ${navigationText} · ${modeText}`
})
const showHeaderMeta = computed(() => !props.compact || sources.value.length > 0)
const showStatusBar = computed(() => !props.compact || sources.value.length > 0)

const hasConnectingSources = computed(() => sources.value.some((source) => source.connecting))
const failedSourceCount = computed(() => sources.value.filter((source) => /失败|异常/.test(source.status)).length)
const totalBufferedLineCount = computed(() => sources.value.reduce((sum, source) => sum + getBufferedLineCount(source), 0))
const hasBufferedLogs = computed(() => totalBufferedLineCount.value > 0)
const isActiveSourceBusy = computed(() => Boolean(activeSource.value?.connecting))
const summaryTone = computed<StatusTone>(() => {
  if (hasConnectingSources.value) return 'warning'
  if (failedSourceCount.value > 0) return 'danger'
  if (sources.value.length === 0) return 'neutral'
  return 'active'
})
const navigationHint = computed(() => {
  return sources.value.length > 1 ? '点击切换，拖动排序，右键关闭' : '单 Pod 宽视图'
})
const statusSummary = computed(() => {
  if (sources.value.length === 0) return '未选择'
  const activeCount = sources.value.filter((source) => /实时中|快照中|已加载|已结束/.test(source.status)).length
  if (hasConnectingSources.value) return `连接 ${activeCount}/${sources.value.length}`
  if (failedSourceCount.value > 0) return `${activeCount}/${sources.value.length} 可用 · ${failedSourceCount.value} 异常`
  return liveMode.value ? `${activeCount}/${sources.value.length} 实时` : `${activeCount}/${sources.value.length} 快照`
})
const activeSourceTone = computed<StatusTone>(() => (activeSource.value ? statusTone(activeSource.value.status) : 'neutral'))
const activeLines = computed(() => (activeSource.value ? getDisplayLines(activeSource.value) : []))
const activeLineCount = computed(() => (activeSource.value ? getBufferedLineCount(activeSource.value) : 0))
const activeSourceText = computed(() => (activeSource.value ? getSourceText(activeSource.value) : ''))
const activeTotalHeight = computed(() => activeLineCount.value * LINE_HEIGHT)
const activeStartIndex = computed(() => Math.max(0, Math.floor(activeScrollTop.value / LINE_HEIGHT) - OVERSCAN))
const activeVisibleCount = computed(() => Math.ceil((activeViewportHeight.value || 0) / LINE_HEIGHT) + OVERSCAN * 2)
const visibleActiveLines = computed(() => {
  return activeLines.value.slice(activeStartIndex.value, activeStartIndex.value + activeVisibleCount.value).map((text, index) => ({
    number: activeStartIndex.value + index + 1,
    text
  }))
})
const activeOffsetTop = computed(() => activeStartIndex.value * LINE_HEIGHT)
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
  clearDragState()
  closeTabContextMenu()
  closeAllSockets('replace_targets')
  sources.value = normalizeTargets(nextTargets)
  ensureActiveSource()
  if (sources.value.length === 0) {
    activeScrollTop.value = 0
    activeViewportHeight.value = 0
    return
  }
  void nextTick(async () => {
    initActiveViewportObserver()
    activeScrollTop.value = activeViewportRef.value?.scrollTop ?? 0
    await restartCurrentSource()
    scrollActiveSourceToBottom()
  })
}

function reset() {
  clearDragState()
  closeTabContextMenu()
  closeAllSockets('workbench_reset')
  destroyActiveViewportObserver()
  clearLogs()
  sources.value = []
  activeSourceId.value = ''
  activeViewportRef.value = null
  activeScrollTop.value = 0
  activeViewportHeight.value = 0
}

defineExpose({ open, reset, restartCurrentSource })

function ensureActiveSource() {
  if (sources.value.length === 0) {
    activeSourceId.value = ''
    return
  }
  if (!sources.value.some((source) => source.id === activeSourceId.value)) {
    activeSourceId.value = sources.value[0].id
  }
}

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
      seq: 0
    })
  }
  return Array.from(map.values())
}

function activateSource(id: string) {
  if (draggingSourceId.value) return
  closeTabContextMenu()
  if (activeSourceId.value === id) return
  activeSourceId.value = id
  void ensureActiveSourceSession(false)
}

function openTabContextMenu(event: MouseEvent, sourceId: string) {
  contextMenuSourceId.value = sourceId
  contextMenuX.value = event.clientX
  contextMenuY.value = event.clientY
  contextMenuVisible.value = true
}

function closeTabContextMenu() {
  contextMenuVisible.value = false
  contextMenuSourceId.value = ''
}

function reorderSource(sourceId: string, targetId: string, placeAfter: boolean) {
  const currentIndex = sources.value.findIndex((source) => source.id === sourceId)
  const targetIndex = sources.value.findIndex((source) => source.id === targetId)
  if (currentIndex < 0 || targetIndex < 0 || currentIndex === targetIndex) return
  const next = sources.value.slice()
  const [item] = next.splice(currentIndex, 1)
  const adjustedTargetIndex = currentIndex < targetIndex ? targetIndex - 1 : targetIndex
  const insertionIndex = placeAfter ? adjustedTargetIndex + 1 : adjustedTargetIndex
  next.splice(Math.max(0, Math.min(insertionIndex, next.length)), 0, item)
  sources.value = next
}

function onTabDragStart(event: DragEvent, sourceId: string) {
  if (sources.value.length <= 1) return
  closeTabContextMenu()
  draggingSourceId.value = sourceId
  dragOverSourceId.value = ''
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', sourceId)
  }
}

function onTabDragOver(event: DragEvent, sourceId: string) {
  if (!draggingSourceId.value || draggingSourceId.value === sourceId) {
    dragOverSourceId.value = ''
    return
  }
  dragOverSourceId.value = sourceId
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
}

function onTabDrop(event: DragEvent, targetId: string) {
  const sourceId = draggingSourceId.value || event.dataTransfer?.getData('text/plain') || ''
  if (!sourceId || sourceId === targetId) {
    clearDragState()
    return
  }
  const targetEl = event.currentTarget instanceof HTMLElement ? event.currentTarget : null
  const placeAfter = targetEl ? event.clientX > targetEl.getBoundingClientRect().left + targetEl.getBoundingClientRect().width / 2 : false
  reorderSource(sourceId, targetId, placeAfter)
  clearDragState()
}

function onTabDragEnd() {
  clearDragState()
}

function clearDragState() {
  draggingSourceId.value = ''
  dragOverSourceId.value = ''
}

function closeSourceTab(sourceId: string) {
  const currentIndex = sources.value.findIndex((source) => source.id === sourceId)
  if (currentIndex < 0) return
  const [removed] = sources.value.splice(currentIndex, 1)
  if (removed) {
    removed.seq += 1
    closeSourceSocket(removed, 'tab_close')
    removed.connecting = false
  }
  if (activeSourceId.value === sourceId) {
    const fallback = sources.value[currentIndex] ?? sources.value[currentIndex - 1] ?? sources.value[0] ?? null
    activeSourceId.value = fallback?.id ?? ''
    if (fallback) {
      void ensureActiveSourceSession(false)
    } else {
      activeScrollTop.value = 0
      activeViewportHeight.value = 0
    }
  }
  closeTabContextMenu()
}

function closeOtherTabs(sourceId: string) {
  const keep = sources.value.find((source) => source.id === sourceId)
  if (!keep) return
  sources.value.forEach((source) => {
    if (source.id === sourceId) return
    source.seq += 1
    closeSourceSocket(source, 'tab_close_other')
    source.connecting = false
  })
  sources.value = [keep]
  activeSourceId.value = keep.id
  closeTabContextMenu()
  void ensureActiveSourceSession(false)
}

function sourceLabel(source: Pick<LogSource, 'ns' | 'name' | 'container'>) {
  return source.container ? `${source.ns}/${source.name} · ${source.container}` : `${source.ns}/${source.name}`
}

function statusTone(status: string): StatusTone {
  if (/失败|异常/.test(status)) return 'danger'
  if (/连接中|加载中|快照中/.test(status)) return 'warning'
  if (/实时中|已加载|已结束/.test(status)) return 'active'
  return 'neutral'
}

function statusTagType(tone: StatusTone): 'info' | 'warning' | 'danger' {
  if (tone === 'danger') return 'danger'
  if (tone === 'warning') return 'warning'
  return 'info'
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

function hasSourceBuffer(source: LogSource): boolean {
  return source.lines.length > 0 || Boolean(source.remainder)
}

function closeInactiveSources(activeID: string) {
  sources.value.forEach((source) => {
    if (source.id === activeID) return
    const wasBusy = source.connecting || Boolean(source.ws)
    source.seq += 1
    closeSourceSocket(source, 'inactive')
    source.connecting = false
    if (!wasBusy) return
    source.status = hasSourceBuffer(source) && liveMode.value ? '已暂停' : hasSourceBuffer(source) ? '已加载' : '未连接'
  })
}

async function ensureActiveSourceSession(force = false) {
  if (!props.clusterId) return
  ensureActiveSource()
  const source = activeSource.value
  if (!source) return
  closeInactiveSources(source.id)
  if (!force) {
    if (liveMode.value && (source.connecting || Boolean(source.ws))) {
      scrollActiveSourceToBottom(source.id)
      return
    }
    if (!liveMode.value && !source.connecting && hasSourceBuffer(source)) {
      scrollActiveSourceToBottom(source.id)
      return
    }
  }
  await startSource(source)
}

function onActiveViewportScroll() {
  activeScrollTop.value = activeViewportRef.value?.scrollTop ?? 0
}

function initActiveViewportObserver() {
  if (!activeViewportRef.value) return
  activeViewportHeight.value = activeViewportRef.value.clientHeight
  if (activeViewportObserver || typeof ResizeObserver === 'undefined') return
  activeViewportObserver = new ResizeObserver(() => {
    activeViewportHeight.value = activeViewportRef.value?.clientHeight ?? 0
  })
  activeViewportObserver.observe(activeViewportRef.value)
}

function destroyActiveViewportObserver() {
  activeViewportObserver?.disconnect()
  activeViewportObserver = null
}

function clearSourceLogs(source: LogSource) {
  source.lines = []
  source.remainder = ''
  if (activeSourceId.value === source.id && activeViewportRef.value) {
    activeViewportRef.value.scrollTop = 0
    activeScrollTop.value = 0
  }
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

function scrollActiveSourceToBottom(sourceId = activeSourceId.value) {
  if (!autoScroll.value || !sourceId || sourceId !== activeSourceId.value) return
  void nextTick(() => {
    if (!activeViewportRef.value) return
    activeViewportRef.value.scrollTop = activeViewportRef.value.scrollHeight
    activeScrollTop.value = activeViewportRef.value.scrollTop
  })
}

function trimSourceLines(lines: string[]): string[] {
  if (lines.length <= MAX_LOG_LINES_PER_SOURCE) return lines
  return lines.slice(lines.length - MAX_LOG_LINES_PER_SOURCE)
}

function appendLines(source: LogSource, lines: string[]) {
  if (lines.length === 0) return
  source.lines = trimSourceLines(source.lines.concat(lines))
  if (source.id === activeSourceId.value) {
    scrollActiveSourceToBottom(source.id)
  }
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
    if (destroyed || source.seq !== seq) return
    const wsUrl = buildPodLogWsUrl(session.ws_url)
    if (!wsUrl) throw new Error('日志 WebSocket 地址无效')

    const currentWs = new WebSocket(wsUrl)
    source.ws = currentWs

    currentWs.onopen = () => {
      if (source.ws !== currentWs || source.seq !== seq || destroyed) return
      source.connecting = false
      source.status = liveMode.value ? '实时中' : '快照中'
    }

    currentWs.onmessage = (evt) => {
      if (source.ws !== currentWs || source.seq !== seq || destroyed) return
      handleSourceFrame(source, evt.data)
    }

    currentWs.onerror = () => {
      if (source.ws !== currentWs || source.seq !== seq || destroyed) return
      source.connecting = false
      source.status = '连接失败'
    }

    currentWs.onclose = () => {
      if (source.ws === currentWs) source.ws = null
      if (source.seq !== seq || destroyed) return
      flushRemainder(source)
      source.connecting = false
      if (source.status === '连接中' || source.status === '加载中') {
        source.status = liveMode.value ? '连接已关闭' : '快照已关闭'
      }
    }
  } catch (error) {
    if (destroyed || source.seq !== seq) return
    source.connecting = false
    source.status = '连接失败'
    const err = error as ApiError
    const message = err?.message ? (err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) : '创建日志会话失败'
    notifyError(`${source.ns}/${source.name}：${message}`)
  }
}

async function restartSource(source: LogSource | null) {
  if (!props.clusterId || !source) return
  await startSource(source)
}

async function restartCurrentSource() {
  if (!props.clusterId || sources.value.length === 0) return
  await ensureActiveSourceSession(true)
}

watch(autoScroll, (value) => {
  if (value) scrollActiveSourceToBottom()
})

watch(activeSourceId, () => {
  void nextTick(() => {
    initActiveViewportObserver()
    activeScrollTop.value = activeViewportRef.value?.scrollTop ?? 0
    if (autoScroll.value) scrollActiveSourceToBottom()
  })
})

onBeforeUnmount(() => {
  destroyed = true
  closeAllSockets('component_unmount')
  destroyActiveViewportObserver()
})
</script>

<style scoped>
.multi-pod-log__shell {
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
  padding: 16px 18px 18px;
  background:
    radial-gradient(circle at top left, rgba(14, 165, 233, 0.14), transparent 24%),
    radial-gradient(circle at top right, rgba(59, 130, 246, 0.1), transparent 26%),
    linear-gradient(180deg, rgba(248, 250, 252, 0.98), rgba(241, 245, 249, 0.98));
}

.multi-pod-log__shell--compact {
  gap: 8px;
  padding: 10px 12px 12px;
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

.multi-pod-log__shell--compact .multi-pod-log__header {
  gap: 12px;
  padding: 9px 10px;
  border-radius: 14px;
}

.multi-pod-log__title-wrap {
  min-width: 0;
}

.multi-pod-log__title {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.multi-pod-log__shell--compact .multi-pod-log__title {
  font-size: 16px;
}

.multi-pod-log__meta {
  margin-top: 4px;
  color: #475569;
  font-size: 13px;
}

.multi-pod-log__shell--compact .multi-pod-log__meta {
  margin-top: 2px;
  font-size: 12px;
}

.multi-pod-log__tail {
  width: 128px;
}

.multi-pod-log__statusbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 2px;
  color: #64748b;
  font-size: 12px;
}

.multi-pod-log__shell--compact .multi-pod-log__statusbar {
  font-size: 11px;
}

.multi-pod-log__statusbar-main {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.multi-pod-log__statusbar-hint,
.multi-pod-log__statusbar-meta {
  white-space: nowrap;
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

.multi-pod-log__empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 0;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(255, 255, 255, 0.78);
}

.multi-pod-log__body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.multi-pod-log__body--side {
  flex-direction: row;
  align-items: stretch;
}

.multi-pod-log__body--wrap-tabs {
  gap: 8px;
}

.multi-pod-log__tabs-card,
.multi-pod-log__viewer {
  min-height: 0;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(255, 255, 255, 0.84);
  backdrop-filter: blur(8px);
}

.multi-pod-log__tabs-card {
  padding: 8px 10px;
}

.multi-pod-log__shell--compact .multi-pod-log__tabs-card {
  padding: 6px 8px;
}

.multi-pod-log__body--side .multi-pod-log__tabs-card {
  width: 196px;
  flex: 0 0 196px;
}

.multi-pod-log__tabs-scroll {
  overflow-x: auto;
  overflow-y: hidden;
  padding-bottom: 2px;
}

.multi-pod-log__body--wrap-tabs .multi-pod-log__tabs-scroll {
  overflow-x: hidden;
  overflow-y: auto;
  max-height: 92px;
  padding-bottom: 0;
}

.multi-pod-log__body--side .multi-pod-log__tabs-scroll {
  height: 100%;
  overflow-x: hidden;
  overflow-y: auto;
  padding-bottom: 0;
}

.multi-pod-log__tabs {
  display: flex;
  gap: 8px;
  min-width: max-content;
}

.multi-pod-log__body--wrap-tabs .multi-pod-log__tabs {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(168px, 1fr));
  min-width: 0;
}

.multi-pod-log__body--side .multi-pod-log__tabs {
  flex-direction: column;
  min-width: 0;
}

.multi-pod-log__tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  appearance: none;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(248, 250, 252, 0.92);
  border-radius: 999px;
  padding: 7px 13px;
  width: auto;
  max-width: 240px;
  flex: 0 0 auto;
  cursor: grab;
  user-select: none;
  transition: border-color 0.18s ease, background-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
}

.multi-pod-log__body--wrap-tabs .multi-pod-log__tab {
  width: 100%;
  max-width: none;
  border-radius: 14px;
  padding: 6px 10px;
}

.multi-pod-log__body--side .multi-pod-log__tab {
  width: 100%;
  max-width: none;
  border-radius: 14px;
}

.multi-pod-log__tab:hover {
  border-color: rgba(59, 130, 246, 0.26);
  background: rgba(239, 246, 255, 0.74);
}

.multi-pod-log__tab--active {
  border-color: rgba(37, 99, 235, 0.32);
  background: rgba(219, 234, 254, 0.88);
}

.multi-pod-log__tab--dragging {
  opacity: 0.56;
  transform: scale(0.98);
  cursor: grabbing;
}

.multi-pod-log__tab--drag-over {
  border-color: rgba(14, 165, 233, 0.52);
  box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.14);
}

.multi-pod-log__tab-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  flex: 0 0 auto;
  background: rgba(148, 163, 184, 0.72);
}

.multi-pod-log__tab-dot--neutral {
  background: rgba(148, 163, 184, 0.72);
}

.multi-pod-log__tab-dot--active {
  background: rgba(14, 165, 233, 0.96);
}

.multi-pod-log__tab-dot--warning {
  background: rgba(245, 158, 11, 0.96);
}

.multi-pod-log__tab-dot--danger {
  background: rgba(239, 68, 68, 0.96);
}

.multi-pod-log__tab-title {
  min-width: 0;
  font-size: 13px;
  font-weight: 700;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.multi-pod-log__body--wrap-tabs .multi-pod-log__tab-title {
  display: -webkit-box;
  overflow: hidden;
  text-overflow: initial;
  white-space: normal;
  line-height: 1.25;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.multi-pod-log__viewer {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.multi-pod-log__body--side .multi-pod-log__viewer {
  min-width: 0;
}

.multi-pod-log__context-mask {
  position: fixed;
  inset: 0;
  z-index: 4100;
}

.multi-pod-log__context-menu {
  position: fixed;
  z-index: 4101;
  min-width: 168px;
  padding: 6px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.18);
  backdrop-filter: blur(12px);
}

.multi-pod-log__context-item {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  border: none;
  border-radius: 10px;
  background: transparent;
  padding: 9px 10px;
  color: #0f172a;
  font-size: 13px;
  cursor: pointer;
}

.multi-pod-log__context-item:hover:not(:disabled) {
  background: rgba(239, 246, 255, 0.9);
  color: #075985;
}

.multi-pod-log__context-item:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.multi-pod-log__context-item--danger:hover:not(:disabled) {
  background: rgba(254, 242, 242, 0.92);
  color: #b91c1c;
}

.multi-pod-log__viewer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.16);
}

.multi-pod-log__shell--compact .multi-pod-log__viewer-head {
  gap: 8px;
  padding: 6px 10px;
}

.multi-pod-log__viewer-summary {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.multi-pod-log__viewer-title {
  min-width: 0;
  font-size: 15px;
  font-weight: 800;
  color: #0f172a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.multi-pod-log__shell--compact .multi-pod-log__viewer-title {
  font-size: 14px;
}

.multi-pod-log__viewer-divider,
.multi-pod-log__viewer-namespace,
.multi-pod-log__viewer-count {
  color: #64748b;
  font-size: 12px;
  white-space: nowrap;
}

.multi-pod-log__viewer-toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.multi-pod-log__viewer-container {
  width: 168px;
}

.multi-pod-log__viewer-container-label {
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  padding: 0 6px;
}

.multi-pod-log__viewer-viewport {
  position: relative;
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

.multi-pod-log__viewer-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100%;
  padding: 24px;
  color: #94a3b8;
  font-size: 13px;
}

.multi-pod-log__viewer-lines {
  display: flex;
  flex-direction: column;
}

.multi-pod-log__viewer-spacer {
  width: 100%;
}

.multi-pod-log__viewer-visible {
  position: absolute;
  left: 0;
  top: 0;
  right: 0;
}

.multi-pod-log__viewer-line {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr);
  gap: 12px;
  padding: 0 14px;
  min-height: 22px;
  align-items: start;
  white-space: pre-wrap;
}

.multi-pod-log__viewer-line:nth-child(2n) {
  background: rgba(255, 255, 255, 0.02);
}

.multi-pod-log__viewer-line-no {
  color: #64748b;
  text-align: right;
  user-select: none;
}

.multi-pod-log__viewer-line-text {
  min-width: 0;
  word-break: break-word;
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

@media (max-width: 1180px) {
  .multi-pod-log__body--side {
    flex-direction: column;
  }

  .multi-pod-log__body--side .multi-pod-log__tabs-card {
    width: auto;
    flex: 0 0 auto;
  }

  .multi-pod-log__body--side .multi-pod-log__tabs-scroll {
    height: auto;
    overflow-x: auto;
    overflow-y: hidden;
    padding-bottom: 2px;
  }

  .multi-pod-log__body--side .multi-pod-log__tabs {
    flex-direction: row;
    min-width: max-content;
  }

  .multi-pod-log__body--side .multi-pod-log__tab {
    width: auto;
    max-width: 220px;
    border-radius: 999px;
  }
}

@media (max-width: 900px) {
  .multi-pod-log__body--wrap-tabs .multi-pod-log__tabs {
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  }

  .multi-pod-log__viewer-head,
  .multi-pod-log__viewer-summary {
    flex-direction: column;
    align-items: flex-start;
  }

  .multi-pod-log__tab {
    max-width: 200px;
  }
}

@media (max-width: 720px) {
  .multi-pod-log__viewer-toolbar {
    width: 100%;
  }

  .multi-pod-log__viewer-container {
    width: 100%;
  }
}
</style>