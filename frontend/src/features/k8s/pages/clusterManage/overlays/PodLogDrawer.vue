<template>
  <el-drawer v-model="visible" class="pod-log-drawer" size="78%" destroy-on-close :with-header="false" @closed="handleClosed">
    <div class="pod-log-drawer__shell">
      <div class="pod-log-drawer__header">
        <div class="pod-log-drawer__title-wrap">
          <div class="pod-log-drawer__title">Pod 日志</div>
          <div class="pod-log-drawer__meta">{{ headerMeta }}</div>
        </div>
        <el-space wrap>
          <el-select v-model="selectedContainer" placeholder="选择容器" class="pod-log-drawer__container" @change="restartSession">
            <el-option v-for="container in containerOptions" :key="container" :label="container" :value="container" />
          </el-select>
          <el-select v-model="tailLines" class="pod-log-drawer__tail" @change="restartSession">
            <el-option v-for="count in tailLineOptions" :key="count" :label="`尾部 ${count} 行`" :value="count" />
          </el-select>
          <el-switch v-model="liveMode" inline-prompt active-text="实时" inactive-text="快照" @change="restartSession" />
          <el-switch v-model="autoScroll" inline-prompt active-text="自动滚动" inactive-text="手动" />
          <el-button :icon="RefreshRight" :loading="connecting" @click="restartSession">刷新</el-button>
          <el-button :icon="Delete" :disabled="lineItems.length === 0 && !lineRemainder" @click="clearLogs">清空</el-button>
          <el-button :icon="CopyDocument" :disabled="!fullText.trim()" @click="copyAll">复制全部</el-button>
          <el-button :icon="Download" :disabled="!fullText.trim()" @click="downloadAll">下载 .txt</el-button>
        </el-space>
      </div>

      <div class="pod-log-drawer__statusbar">
        <el-tag size="small" :type="connecting ? 'warning' : liveMode ? 'success' : 'info'">{{ statusText }}</el-tag>
        <span>当前 {{ displayLineCount }} 行，最多保留 5000 行</span>
      </div>

      <div ref="viewportRef" class="pod-log-drawer__viewport" @scroll="onViewportScroll">
        <div class="pod-log-drawer__spacer" :style="{ height: `${totalHeight}px` }"></div>
        <div class="pod-log-drawer__visible" :style="{ transform: `translateY(${offsetTop}px)` }">
          <div v-for="item in visibleLines" :key="item.number" class="pod-log-drawer__line">
            <span class="pod-log-drawer__line-no">{{ item.number }}</span>
            <span class="pod-log-drawer__line-text">{{ item.text || ' ' }}</span>
          </div>
        </div>
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
import { buildTerminalWebSocketUrl } from '@/shared/utils/terminal'
import { copyToClipboard, downloadBlob, sanitizeFileName } from '@/shared/utils/text'

const props = defineProps<{ clusterId: number }>()

type PodLogTarget = {
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

const MAX_LOG_LINES = 5000
const LINE_HEIGHT = 20
const OVERSCAN = 16
const tailLineOptions = [100, 200, 500, 1000, 2000]

const visible = ref(false)
const connecting = ref(false)
const liveMode = ref(true)
const autoScroll = ref(true)
const tailLines = ref(200)
const statusText = ref('未连接')
const target = ref<PodLogTarget | null>(null)
const containerOptions = ref<string[]>([])
const selectedContainer = ref('')
const viewportRef = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewportHeight = ref(0)
const lineItems = ref<string[]>([])
const lineRemainder = ref('')

let ws: WebSocket | null = null
let wsSeq = 0
let viewportObserver: ResizeObserver | null = null

const headerMeta = computed(() => {
  if (!target.value) return '-'
  return selectedContainer.value
    ? `${target.value.ns}/${target.value.name} · ${selectedContainer.value}`
    : `${target.value.ns}/${target.value.name}`
})

const displayLineCount = computed(() => lineItems.value.length + (lineRemainder.value ? 1 : 0))
const fullText = computed(() => {
  const lines = lineItems.value.slice()
  if (lineRemainder.value) lines.push(lineRemainder.value)
  return lines.join('\n')
})
const totalHeight = computed(() => displayLineCount.value * LINE_HEIGHT)
const startIndex = computed(() => Math.max(0, Math.floor(scrollTop.value / LINE_HEIGHT) - OVERSCAN))
const visibleCount = computed(() => Math.ceil((viewportHeight.value || 0) / LINE_HEIGHT) + OVERSCAN * 2)
const visibleLines = computed(() => {
  const lines = lineItems.value.slice()
  if (lineRemainder.value) lines.push(lineRemainder.value)
  return lines.slice(startIndex.value, startIndex.value + visibleCount.value).map((text, index) => ({
    number: startIndex.value + index + 1,
    text
  }))
})
const offsetTop = computed(() => startIndex.value * LINE_HEIGHT)

function open(next: PodLogTarget) {
  target.value = { ...next }
  containerOptions.value = normalizeContainers(next)
  selectedContainer.value = pickContainer(next, containerOptions.value)
  visible.value = true
  void nextTick(() => {
    initViewportObserver()
    void restartSession()
  })
}

function close() {
  visible.value = false
}

defineExpose({ open, close })

function normalizeContainers(next: PodLogTarget): string[] {
  const values = Array.isArray(next.containers) ? next.containers.map((item) => String(item || '').trim()).filter(Boolean) : []
  if (values.length > 0) return Array.from(new Set(values))
  return next.container ? [String(next.container).trim()].filter(Boolean) : []
}

function pickContainer(next: PodLogTarget, options: string[]): string {
  const preferred = String(next.container || '').trim()
  if (preferred && options.includes(preferred)) return preferred
  return options[0] || ''
}

function clearLogs() {
  lineItems.value = []
  lineRemainder.value = ''
  scrollTop.value = 0
  if (viewportRef.value) viewportRef.value.scrollTop = 0
}

async function copyAll() {
  if (!fullText.value.trim()) return
  try {
    await copyToClipboard(fullText.value)
    notifySuccess('日志已复制')
  } catch {
    notifyError('复制日志失败')
  }
}

function downloadAll() {
  if (!fullText.value.trim()) return
  const meta = target.value ? `${target.value.ns}_${target.value.name}` : 'pod'
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-')
  downloadBlob(`pod_logs_${sanitizeFileName(meta)}_${timestamp}.txt`, new Blob([fullText.value], { type: 'text/plain;charset=utf-8' }))
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

function appendChunk(chunk: string) {
  const normalized = String(chunk || '').replace(/\r\n/g, '\n').replace(/\r/g, '\n')
  if (!normalized) return
  const combined = `${lineRemainder.value}${normalized}`
  const endsWithNewline = combined.endsWith('\n')
  const parts = combined.split('\n')
  if (endsWithNewline && parts.length > 0 && parts[parts.length - 1] === '') {
    parts.pop()
    lineRemainder.value = ''
  } else {
    lineRemainder.value = parts.pop() ?? ''
  }
  if (parts.length > 0) {
    lineItems.value = trimLines(lineItems.value.concat(parts))
  }
  scrollToBottom()
}

function flushRemainder() {
  if (!lineRemainder.value) return
  lineItems.value = trimLines(lineItems.value.concat(lineRemainder.value))
  lineRemainder.value = ''
  scrollToBottom()
}

function trimLines(lines: string[]): string[] {
  if (lines.length <= MAX_LOG_LINES) return lines
  return lines.slice(lines.length - MAX_LOG_LINES)
}

function closeSocket(reason = 'client_close') {
  wsSeq += 1
  if (!ws) return
  try {
    ws.onopen = null
    ws.onclose = null
    ws.onerror = null
    ws.onmessage = null
    ws.close(1000, reason)
  } catch {
    // ignore
  } finally {
    ws = null
  }
}

function buildPodLogWsUrl(raw: string): string | null {
  const value = String(raw || '').trim()
  if (!value) return null
  const parsed = /^https?:\/\//.test(value) || /^wss?:\/\//.test(value)
    ? new URL(value)
    : new URL(value.startsWith('/') ? value : `/${value}`, window.location.origin)
  return buildTerminalWebSocketUrl(parsed.pathname, Object.fromEntries(parsed.searchParams.entries()))
}

async function restartSession() {
  if (!visible.value || !props.clusterId || !target.value) return
  closeSocket('restart')
  clearLogs()
  connecting.value = true
  statusText.value = liveMode.value ? '连接实时日志中' : '加载日志快照中'
  const seq = wsSeq
  try {
    const session = await k8sApi.createPodLogSession(props.clusterId, target.value.ns, target.value.name, {
      container: selectedContainer.value || undefined,
      follow: liveMode.value,
      tail_lines: tailLines.value
    })
    if (seq !== wsSeq) return
    const wsUrl = buildPodLogWsUrl(session.ws_url)
    if (!wsUrl) throw new Error('日志 WebSocket 地址无效')

    const currentWs = new WebSocket(wsUrl)
    ws = currentWs

    currentWs.onopen = () => {
      if (ws !== currentWs) return
      connecting.value = false
      statusText.value = liveMode.value ? '实时流已连接' : '快照加载中'
    }

    currentWs.onmessage = (event) => {
      if (ws !== currentWs) return
      let frame: PodLogFrame | null = null
      try {
        frame = JSON.parse(String(event.data || '')) as PodLogFrame
      } catch {
        frame = { type: 'chunk', data: String(event.data || '') }
      }
      const frameType = String(frame?.type || '').toLowerCase()
      if (frameType === 'chunk') {
        appendChunk(String(frame?.data || ''))
        return
      }
      if (frameType === 'eof') {
        flushRemainder()
        connecting.value = false
        statusText.value = liveMode.value ? '日志流结束' : '快照已加载'
        return
      }
      if (frameType === 'error') {
        connecting.value = false
        statusText.value = '日志流异常'
        notifyError(String(frame?.message || '日志流连接失败'))
      }
    }

    currentWs.onerror = () => {
      if (ws !== currentWs) return
      connecting.value = false
      statusText.value = '日志流连接失败'
    }

    currentWs.onclose = () => {
      if (ws === currentWs) ws = null
      if (!visible.value) return
      connecting.value = false
      if (statusText.value === '连接实时日志中' || statusText.value === '加载日志快照中') {
        statusText.value = liveMode.value ? '连接已关闭' : '快照已关闭'
      }
    }
  } catch (error) {
    connecting.value = false
    statusText.value = '日志流创建失败'
    const err = error as ApiError
    notifyError(err?.message ? (err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message) : '创建日志会话失败')
  }
}

function handleClosed() {
  closeSocket('drawer_closed')
  destroyViewportObserver()
  clearLogs()
  target.value = null
  containerOptions.value = []
  selectedContainer.value = ''
  statusText.value = '未连接'
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
  closeSocket('component_unmount')
  destroyViewportObserver()
})
</script>

<style scoped>
.pod-log-drawer__shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  padding: 16px 18px 18px;
  background:
    radial-gradient(circle at top left, rgba(14, 165, 233, 0.14), transparent 26%),
    radial-gradient(circle at top right, rgba(16, 185, 129, 0.12), transparent 24%),
    linear-gradient(180deg, rgba(248, 250, 252, 0.98), rgba(241, 245, 249, 0.98));
}

.pod-log-drawer__header {
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

.pod-log-drawer__title-wrap {
  min-width: 0;
}

.pod-log-drawer__title {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

.pod-log-drawer__meta {
  margin-top: 4px;
  color: #475569;
  font-size: 13px;
  word-break: break-all;
}

.pod-log-drawer__container {
  width: 220px;
}

.pod-log-drawer__tail {
  width: 128px;
}

.pod-log-drawer__statusbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 4px;
  color: #475569;
  font-size: 13px;
}

.pod-log-drawer__viewport {
  position: relative;
  flex: 1;
  overflow: auto;
  border-radius: 18px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: linear-gradient(180deg, #07121f, #0f172a);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.pod-log-drawer__spacer {
  width: 1px;
  opacity: 0;
}

.pod-log-drawer__visible {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
}

.pod-log-drawer__line {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 12px;
  min-height: 20px;
  padding: 0 16px;
  font-family: 'Cascadia Code', 'JetBrains Mono', 'Fira Code', Consolas, monospace;
  font-size: 12px;
  line-height: 20px;
}

.pod-log-drawer__line:nth-child(odd) {
  background: rgba(15, 23, 42, 0.18);
}

.pod-log-drawer__line-no {
  color: rgba(148, 163, 184, 0.9);
  text-align: right;
  user-select: none;
}

.pod-log-drawer__line-text {
  color: rgba(226, 232, 240, 0.96);
  white-space: pre;
}

:global(html.dark) .pod-log-drawer__shell {
  background:
    radial-gradient(circle at top left, rgba(14, 165, 233, 0.12), transparent 26%),
    radial-gradient(circle at top right, rgba(16, 185, 129, 0.08), transparent 24%),
    linear-gradient(180deg, rgba(2, 6, 23, 0.98), rgba(15, 23, 42, 0.96));
}

:global(html.dark) .pod-log-drawer__header {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.78);
}

:global(html.dark) .pod-log-drawer__title {
  color: #e2e8f0;
}

:global(html.dark) .pod-log-drawer__meta,
:global(html.dark) .pod-log-drawer__statusbar {
  color: #94a3b8;
}

@media (max-width: 960px) {
  .pod-log-drawer__header {
    flex-direction: column;
  }

  .pod-log-drawer__container,
  .pod-log-drawer__tail {
    width: 100%;
  }

  .pod-log-drawer__line {
    grid-template-columns: 56px minmax(0, 1fr);
    padding: 0 12px;
  }
}
</style>