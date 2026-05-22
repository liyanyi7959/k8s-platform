<template>
  <el-drawer v-model="visible" class="multi-pod-log-drawer" size="86%" destroy-on-close :with-header="false" @closed="handleClosed">
    <div class="multi-pod-log__shell">
      <div class="multi-pod-log__header">
        <div class="multi-pod-log__title-wrap">
          <div class="multi-pod-log__title">多 Pod 日志</div>
          <div class="multi-pod-log__meta">{{ headerMeta }}</div>
        </div>
        <el-space wrap>
          <el-select v-model="tailLines" class="multi-pod-log__tail" @change="restartAll">
            <el-option v-for="count in tailLineOptions" :key="count" :label="`尾部 ${count} 行`" :value="count" />
          </el-select>
          <el-switch v-model="liveMode" inline-prompt active-text="实时" inactive-text="快照" @change="restartAll" />
          <el-switch v-model="autoScroll" inline-prompt active-text="自动滚动" inactive-text="手动" />
          <el-button :icon="RefreshRight" :loading="hasConnectingSources" @click="restartAll">刷新</el-button>
          <el-button :icon="Delete" :disabled="lineItems.length === 0" @click="clearLogs">清空</el-button>
          <el-button :icon="CopyDocument" :disabled="!fullText.trim()" @click="copyAll">复制全部</el-button>
          <el-button :icon="Download" :disabled="!fullText.trim()" @click="downloadAll">下载 .txt</el-button>
        </el-space>
      </div>

      <div class="multi-pod-log__statusbar">
        <el-tag size="small" :type="hasConnectingSources ? 'warning' : failedSourceCount > 0 ? 'danger' : liveMode ? 'success' : 'info'">{{ statusSummary }}</el-tag>
        <span>当前 {{ lineItems.length }} 行，最多保留 8000 行</span>
      </div>

      <div class="multi-pod-log__body">
        <aside class="multi-pod-log__sources">
          <div class="multi-pod-log__sources-head">
            <span>日志源</span>
            <el-tag size="small" type="info">{{ sources.length }} 个</el-tag>
          </div>
          <el-scrollbar class="multi-pod-log__sources-scroll">
            <div class="multi-pod-log__sources-list">
              <article v-for="source in sources" :key="source.id" class="multi-pod-log__source-item">
                <div class="multi-pod-log__source-main">
                  <div class="multi-pod-log__source-name">{{ source.ns }}/{{ source.name }}</div>
                  <el-tag size="small" :type="sourceTagType(source.status)">{{ source.status }}</el-tag>
                </div>
                <div class="multi-pod-log__source-meta">已收集 {{ source.lineCount }} 行</div>
                <el-select
                  v-if="source.containers.length > 0"
                  v-model="source.container"
                  size="small"
                  placeholder="容器"
                  class="multi-pod-log__source-container"
                  :disabled="source.containers.length <= 1"
                  @change="restartAll"
                >
                  <el-option v-for="container in source.containers" :key="container" :label="container" :value="container" />
                </el-select>
              </article>
            </div>
          </el-scrollbar>
        </aside>

        <section class="multi-pod-log__viewport-wrap">
          <div v-if="sources.length === 0" class="multi-pod-log__empty">
            <el-empty description="请选择 Pod 后再打开多 Pod 日志工作台" />
          </div>
          <template v-else>
            <div ref="viewportRef" class="multi-pod-log__viewport" @scroll="onViewportScroll">
              <div class="multi-pod-log__spacer" :style="{ height: `${totalHeight}px` }"></div>
              <div class="multi-pod-log__visible" :style="{ transform: `translateY(${offsetTop}px)` }">
                <div v-for="item in visibleLines" :key="item.id" class="multi-pod-log__line">
                  <span class="multi-pod-log__line-no">{{ item.id }}</span>
                  <span class="multi-pod-log__line-source">{{ item.source }}</span>
                  <span class="multi-pod-log__line-text">{{ item.text || ' ' }}</span>
                </div>
              </div>
            </div>
          </template>
        </section>
      </div>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { CopyDocument, Delete, Download, RefreshRight } from '@element-plus/icons-vue'

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

type AggregatedLine = {
  id: number
  sourceId: string
  source: string
  text: string
}

type LogSource = {
  id: string
  ns: string
  name: string
  containers: string[]
  container: string
  connecting: boolean
  status: string
  remainder: string
  lineCount: number
  ws: WebSocket | null
  seq: number
}

const MAX_LOG_LINES = 8000
const LINE_HEIGHT = 22
const OVERSCAN = 16
const tailLineOptions = [100, 200, 500, 1000, 2000]

const visible = ref(false)
const liveMode = ref(true)
const autoScroll = ref(true)
const tailLines = ref(200)
const sources = ref<LogSource[]>([])
const lineItems = ref<AggregatedLine[]>([])
const viewportRef = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportHeight = ref(0)

let viewportObserver: ResizeObserver | null = null
let globalLineSeq = 0

const headerMeta = computed(() => {
  if (sources.value.length === 0) return '-'
  const mode = liveMode.value ? '实时聚合模式' : '快照聚合模式'
  return `${sources.value.length} 个 Pod · ${mode}`
})

const hasConnectingSources = computed(() => sources.value.some((source) => source.connecting))
const failedSourceCount = computed(() => sources.value.filter((source) => /失败|异常/.test(source.status)).length)
const statusSummary = computed(() => {
  if (sources.value.length === 0) return '未选择日志源'
  const activeCount = sources.value.filter((source) => /实时中|快照中|已加载|已结束/.test(source.status)).length
  if (hasConnectingSources.value) return `正在连接 ${activeCount}/${sources.value.length}`
  if (failedSourceCount.value > 0) return `${activeCount}/${sources.value.length} 已连接，${failedSourceCount.value} 路异常`
  return liveMode.value ? `${activeCount}/${sources.value.length} 路实时跟随` : `${activeCount}/${sources.value.length} 路快照已加载`
})
const fullText = computed(() => lineItems.value.map((item) => `[${item.source}] ${item.text}`).join('\n'))
const totalHeight = computed(() => lineItems.value.length * LINE_HEIGHT)
const startIndex = computed(() => Math.max(0, Math.floor(scrollTop.value / LINE_HEIGHT) - OVERSCAN))
const visibleCount = computed(() => Math.ceil((viewportHeight.value || 0) / LINE_HEIGHT) + OVERSCAN * 2)
const visibleLines = computed(() => lineItems.value.slice(startIndex.value, startIndex.value + visibleCount.value))
const offsetTop = computed(() => startIndex.value * LINE_HEIGHT)

function open(nextTargets: MultiPodLogTarget[]) {
  sources.value = normalizeTargets(nextTargets)
  lineItems.value = []
  globalLineSeq = 0
  visible.value = true
  void nextTick(() => {
    initViewportObserver()
    void restartAll()
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
      lineCount: 0,
      ws: null,
      seq: 0
    })
  }
  return Array.from(map.values())
}

function sourceLabel(source: Pick<LogSource, 'ns' | 'name' | 'container'>) {
  return source.container ? `${source.ns}/${source.name} · ${source.container}` : `${source.ns}/${source.name}`
}

function sourceTagType(status: string): 'info' | 'success' | 'warning' | 'danger' {
  if (/失败|异常/.test(status)) return 'danger'
  if (/连接中|加载中|快照中/.test(status)) return 'warning'
  if (/实时中|已加载|已结束/.test(status)) return 'success'
  return 'info'
}

function clearLogs() {
  lineItems.value = []
  globalLineSeq = 0
  scrollTop.value = 0
  if (viewportRef.value) viewportRef.value.scrollTop = 0
  sources.value.forEach((source) => {
    source.remainder = ''
    source.lineCount = 0
  })
}

async function copyAll() {
  if (!fullText.value.trim()) return
  try {
    await copyToClipboard(fullText.value)
    notifySuccess('聚合日志已复制')
  } catch {
    notifyError('复制日志失败')
  }
}

function downloadAll() {
  if (!fullText.value.trim()) return
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-')
  downloadBlob(`multi_pod_logs_${sanitizeFileName(String(props.clusterId))}_${timestamp}.txt`, new Blob([fullText.value], { type: 'text/plain;charset=utf-8' }))
}

function onViewportScroll() {
  scrollTop.value = viewportRef.value?.scrollTop ?? 0
}

function initViewportObserver() {
  if (!viewportRef.value) return
  viewportHeight.value = viewportRef.value.clientHeight
  if (viewportObserver || typeof ResizeObserver === 'undefined') return
  viewportObserver = new ResizeObserver(() => {
    viewportHeight.value = viewportRef.value?.clientHeight ?? 0
  })
  viewportObserver.observe(viewportRef.value)
}

function destroyViewportObserver() {
  viewportObserver?.disconnect()
  viewportObserver = null
}

function scrollToBottom() {
  if (!autoScroll.value) return
  void nextTick(() => {
    if (!viewportRef.value) return
    viewportRef.value.scrollTop = viewportRef.value.scrollHeight
    scrollTop.value = viewportRef.value.scrollTop
  })
}

function appendLines(source: LogSource, lines: string[]) {
  if (lines.length === 0) return
  source.lineCount += lines.length
  const nextItems = lines.map((text) => ({
    id: ++globalLineSeq,
    sourceId: source.id,
    source: sourceLabel(source),
    text
  }))
  const merged = lineItems.value.concat(nextItems)
  lineItems.value = merged.length <= MAX_LOG_LINES ? merged : merged.slice(merged.length - MAX_LOG_LINES)
  scrollToBottom()
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
  source.connecting = true
  source.status = liveMode.value ? '连接中' : '加载中'
  source.remainder = ''
  source.lineCount = 0

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

    currentWs.onmessage = (event) => {
      if (source.ws !== currentWs || source.seq !== seq) return
      handleSourceFrame(source, event.data)
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

async function restartAll() {
  if (!visible.value || !props.clusterId || sources.value.length === 0) return
  closeAllSockets('restart')
  clearLogs()
  await Promise.all(sources.value.map((source) => startSource(source)))
}

function handleClosed() {
  closeAllSockets('drawer_closed')
  destroyViewportObserver()
  clearLogs()
  sources.value = []
}

watch(visible, (value) => {
  if (!value) return
  void nextTick(() => {
    initViewportObserver()
    scrollToBottom()
  })
})

watch(autoScroll, (value) => {
  if (value) scrollToBottom()
})

onBeforeUnmount(() => {
  closeAllSockets('component_unmount')
  destroyViewportObserver()
})
</script>

<style scoped>
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

.multi-pod-log__body {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 14px;
}

.multi-pod-log__sources,
.multi-pod-log__viewport-wrap {
  min-height: 0;
}

.multi-pod-log__sources {
  display: flex;
  flex-direction: column;
  gap: 10px;
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
  font-size: 12px;
  color: #64748b;
}

.multi-pod-log__source-container {
  margin-top: 10px;
  width: 100%;
}

.multi-pod-log__viewport-wrap {
  display: flex;
  flex-direction: column;
  border-radius: 18px;
  overflow: hidden;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: #0f172a;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04), 0 24px 60px rgba(15, 23, 42, 0.18);
}

.multi-pod-log__empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.96), rgba(2, 6, 23, 0.98));
}

.multi-pod-log__viewport {
  position: relative;
  flex: 1;
  overflow: auto;
  min-height: 0;
  font-family: "JetBrains Mono", "Cascadia Code", Consolas, monospace;
  font-size: 12px;
  line-height: 22px;
  color: #e2e8f0;
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.96), rgba(2, 6, 23, 0.98)),
    radial-gradient(circle at top left, rgba(14, 165, 233, 0.16), transparent 24%);
}

.multi-pod-log__spacer {
  width: 100%;
}

.multi-pod-log__visible {
  position: absolute;
  left: 0;
  top: 0;
  right: 0;
}

.multi-pod-log__line {
  display: grid;
  grid-template-columns: 64px 260px minmax(0, 1fr);
  gap: 12px;
  padding: 0 14px;
  min-height: 22px;
  align-items: start;
  white-space: pre-wrap;
}

.multi-pod-log__line:nth-child(2n) {
  background: rgba(255, 255, 255, 0.02);
}

.multi-pod-log__line-no {
  color: #64748b;
  text-align: right;
  user-select: none;
}

.multi-pod-log__line-source {
  color: #38bdf8;
  user-select: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.multi-pod-log__line-text {
  min-width: 0;
  word-break: break-word;
}

@media (max-width: 1200px) {
  .multi-pod-log__body {
    grid-template-columns: 1fr;
  }

  .multi-pod-log__line {
    grid-template-columns: 52px 220px minmax(0, 1fr);
  }
}

@media (max-width: 900px) {
  .multi-pod-log__header {
    flex-direction: column;
  }

  .multi-pod-log__line {
    grid-template-columns: 52px minmax(0, 1fr);
  }

  .multi-pod-log__line-source {
    grid-column: 2;
  }

  .multi-pod-log__line-text {
    grid-column: 2;
  }
}
</style>