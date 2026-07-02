<template>
  <div class="pod-exec-terminal-wrapper">
    <div ref="terminalRef" class="terminal-viewport" />
  </div>
</template>

<script setup lang="ts">
import { ref, onBeforeUnmount, onMounted, watch } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import { buildTerminalWebSocketUrl, resolveTerminalBaseUrl, getTerminalTheme, type TerminalUiTheme } from '@/shared/utils/terminal'

const props = withDefaults(defineProps<{
  clusterId: number
  namespace: string
  pod: string
  container?: string
  command?: string[]
  theme?: 'light' | 'dark'
  autoConnect?: boolean
}>(), {
  container: '',
  command: () => ['sh'],
  theme: 'dark',
  autoConnect: true
})

const emit = defineEmits<{
  (e: 'status', v: { connected: boolean; connecting: boolean }): void
}>()

const terminalRef = ref<HTMLElement | null>(null)
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let windowResizeHandler: (() => void) | null = null
let wsSeq = 0
const connecting = ref(false)
const connected = ref(false)
let connectPromise: Promise<void> | null = null
let receivedTerminalError = false
let receivedSocketError = false

// Terminal Theme Configuration
function initTerminal() {
  if (!terminalRef.value) return

  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    theme: getTerminalTheme(props.theme as TerminalUiTheme),
    allowProposedApi: true,
    drawBoldTextInBrightColors: true,
    minimumContrastRatio: 4.5
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(terminalRef.value)
  fitAddon.fit()

  term.onData((data) => {
    sendTerminalData(data)
  })

  term.onResize((size) => {
    sendResize(size.cols, size.rows)
  })

  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      fitAddon?.fit()
      if (term) {
        sendResize(term.cols, term.rows)
      }
    })
    resizeObserver.observe(terminalRef.value)
  } else {
    windowResizeHandler = () => {
      fitAddon?.fit()
      if (term) {
        sendResize(term.cols, term.rows)
      }
    }
    window.addEventListener('resize', windowResizeHandler)
  }
}

function buildPodExecWsUrl(raw: string): string | null {
  const v = String(raw ?? '').trim()
  if (!v) return null

  let u: URL
  if (/^wss?:\/\//.test(v)) {
    u = new URL(v)
  } else if (/^https?:\/\//.test(v)) {
    u = new URL(v)
    u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  } else {
    // 相对路径（可能含 ?query），用 URL 构造器正确分离 path 与 query
    u = new URL(v.startsWith('/') ? v : `/${v}`, resolveTerminalBaseUrl())
  }

  return buildTerminalWebSocketUrl(u.pathname, Object.fromEntries(u.searchParams.entries()))
}

function setStatus(next: { connected?: boolean; connecting?: boolean }) {
  if (typeof next.connected === 'boolean') connected.value = next.connected
  if (typeof next.connecting === 'boolean') connecting.value = next.connecting
  emit('status', { connected: connected.value, connecting: connecting.value })
}

function disconnect() {
  wsSeq++
  connectPromise = null
  receivedTerminalError = false
  setStatus({ connected: false, connecting: false })
  if (ws) {
    try {
      ws.onopen = null
      ws.onclose = null
      ws.onerror = null
      ws.onmessage = null
      ws.close(1000, 'client_close')
    } catch {
      // ignore
    } finally {
      ws = null
    }
  }
}

async function connect() {
  if (!props.clusterId || !props.namespace || !props.pod) return
  if (connecting.value || connected.value) return

  if (connectPromise) {
    await connectPromise
    return
  }

  disconnect()
  setStatus({ connecting: true })
  receivedTerminalError = false
  receivedSocketError = false
  term?.write('Connecting...\r\n')
  const seq = wsSeq

  const attempt = (async () => {
    const WS_TIMEOUT_MS = 15000
    try {
      const reqBody = {
        container: props.container || undefined,
        command: props.command && props.command.length ? props.command : ['/bin/bash'],
        tty: true
      }
      console.log('[PodExecTerminal] Creating session:', { clusterId: props.clusterId, namespace: props.namespace, pod: props.pod, body: reqBody })
      const session = await k8sApi.createPodExecSession(props.clusterId, props.namespace, props.pod, reqBody)

      if (wsSeq !== seq) return

      if (!session?.ws_url) {
        term?.write('\r\nError: 服务端未返回 WebSocket 地址，请检查集群连接和权限。\r\n')
        setStatus({ connected: false, connecting: false })
        return
      }

      const wsUrl = buildPodExecWsUrl(session.ws_url)
      if (!wsUrl) {
        term?.write('\r\nError: 无法构建 WebSocket 连接地址。\r\n')
        setStatus({ connected: false, connecting: false })
        return
      }

      console.log('[PodExecTerminal] Connecting to:', wsUrl)
      const currentWs = new WebSocket(wsUrl)
      currentWs.binaryType = 'arraybuffer'
      ws = currentWs

      await new Promise<void>((resolve) => {
        let settled = false
        const finish = () => {
          if (!settled) {
            settled = true
            resolve()
          }
        }

        const timeout = setTimeout(() => {
          if (wsSeq === seq && ws === currentWs) {
            term?.write('\r\n连接超时，请检查网络和集群状态。\r\n')
            try { currentWs.close(1000, 'timeout') } catch { /* ignore */ }
          }
          finish()
        }, WS_TIMEOUT_MS)

        currentWs.onopen = () => {
          clearTimeout(timeout)
          if (wsSeq !== seq || ws !== currentWs) {
            finish()
            return
          }
          term?.write('Connected.\r\n')
          setStatus({ connected: true, connecting: false })
          fitAddon?.fit()
          if (term) {
            sendResize(term.cols, term.rows)
          }
          term?.focus()
          finish()
        }

        currentWs.onclose = (ev) => {
          clearTimeout(timeout)
          console.warn(`[PodExecTerminal] WebSocket closed: code=${ev.code} reason="${ev.reason || ''}"`)
          if (ws === currentWs) {
            ws = null
          }
          if (wsSeq === seq) {
            const reason = String(ev.reason ?? '').trim()
            if (!receivedTerminalError && ev.code !== 1000) {
              if (reason) {
                receivedTerminalError = true
                term?.write(`\r\nError: ${reason}\r\n`)
              } else if (receivedSocketError) {
                term?.write('\r\nWebSocket 连接失败，请检查后端服务是否运行、集群是否可达。\r\n')
              } else if (ev.code === 1006) {
                term?.write('\r\n连接异常断开 (code=1006)：后端可能未启动、集群不可达或 WebSocket 握手失败。\r\n')
              } else {
                term?.write(`\r\n连接关闭 (code=${ev.code})。\r\n`)
              }
            }
            setStatus({ connected: false, connecting: false })
          }
          finish()
        }

        currentWs.onerror = (ev) => {
          console.error('[PodExecTerminal] WebSocket error:', ev)
          if (wsSeq === seq && ws === currentWs) {
            receivedSocketError = true
            setStatus({ connected: false, connecting: false })
          }
        }

        const decoder = typeof TextDecoder !== 'undefined' ? new TextDecoder() : null
        currentWs.onmessage = (ev) => {
          if (wsSeq !== seq || ws !== currentWs) return
          const data = ev.data
          if (typeof data === 'string') {
            handlePayload(data)
            return
          }
          if (data instanceof ArrayBuffer) {
            const u8 = new Uint8Array(data)
            // K8s exec channel multiplexing: 1=stdout, 2=stderr, 3=error
            if (u8.length >= 2 && (u8[0] === 1 || u8[0] === 2)) {
              term?.write(decoder ? decoder.decode(u8.slice(1)) : String.fromCharCode(...u8.slice(1)))
            } else if (u8.length >= 2 && u8[0] === 3) {
              receivedTerminalError = true
              term?.write(`\r\nError: ${decoder ? decoder.decode(u8.slice(1)) : String.fromCharCode(...u8.slice(1))}\r\n`)
            } else {
              handlePayload(decoder ? decoder.decode(u8) : String.fromCharCode(...u8))
            }
          }
        }
      })
    } catch (e) {
      if (wsSeq !== seq) return
      const err = e as ApiError
      receivedTerminalError = true
      const msg = err.message || '创建终端会话失败'
      term?.write(`\r\nError: ${msg}\r\n`)
      notifyError(msg)
      setStatus({ connected: false, connecting: false })
    }

  })()

  const attemptPromise = attempt.finally(() => {
    if (connectPromise === attemptPromise) {
      connectPromise = null
    }
  })

  connectPromise = attemptPromise

  await connectPromise
}

function handlePayload(text: string) {
  try {
    const v = JSON.parse(text)
    if (v.type === 'stdout' || v.type === 'stderr') {
      term?.write(v.data ?? '')
    } else if (v.type === 'error') {
		receivedTerminalError = true
      term?.write(`\r\nError: ${v.data}\r\n`)
    } else if (typeof v.data === 'string') {
      term?.write(v.data)
    }
  } catch {
    // Fallback for raw text
    term?.write(text)
  }
}

function sendTerminalData(data: string) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return
  try {
    ws.send(JSON.stringify({ type: 'stdin', data }))
  } catch {
    try {
      ws.send(data)
    } catch {
      // ignore
    }
  }
}

function sendResize(cols: number, rows: number) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return
  try {
    ws.send(JSON.stringify({ type: 'resize', cols, rows }))
  } catch {
    // ignore
  }
}

function dispose() {
  resizeObserver?.disconnect()
  resizeObserver = null
  if (windowResizeHandler) {
    window.removeEventListener('resize', windowResizeHandler)
    windowResizeHandler = null
  }
  
  disconnect()
  if (term) {
    term.dispose()
    term = null
  }
  fitAddon = null
}

watch(() => props.theme, (v) => {
  if (term) {
    term.options.theme = getTerminalTheme(v as TerminalUiTheme)
  }
})

watch(
  () => [props.clusterId, props.namespace, props.pod, props.container, (props.command ?? []).join('\u0000'), props.autoConnect],
  () => {
    if (!props.autoConnect) return
    if (!term) return
    void connect()
  }
)

function clear() {
  term?.clear()
}

function focus() {
  term?.focus()
}

function sendCtrlC() {
  sendTerminalData('\u0003')
}

function reconnect() {
  disconnect()
  void connect()
}

defineExpose({ connect, reconnect, disconnect, clear, focus, sendCtrlC })

onMounted(() => {
  initTerminal()
  if (props.autoConnect) void connect()
})

onBeforeUnmount(() => {
  dispose()
})
</script>

<style scoped>
.pod-exec-terminal-wrapper {
  width: 100%;
  height: 100%;
  background: v-bind("props.theme === 'light' ? 'var(--c-white)' : 'var(--c-slate-950)'");
  overflow: hidden;
  border-radius: 8px;
}

.terminal-viewport {
  width: 100%;
  height: 100%;
  padding: 8px 0 0 8px; /* Slight padding */
}
</style>
