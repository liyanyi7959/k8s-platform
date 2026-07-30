/**
 * Pod Shell terminal page.
 *
 * The opener transfers a short-lived, one-time stream ticket through a
 * same-origin postMessage handshake. No token or ticket is placed in this
 * window's address bar.
 */
import { useEffect, useRef, useState } from 'react'

function validTerminalStreamPath(value: unknown): string {
  if (typeof value !== 'string' || !value) return ''
  try {
    const url = new URL(value, window.location.origin)
    if (
      url.origin !== window.location.origin ||
      url.search ||
      !url.pathname.startsWith('/streams/v2/pod-exec/')
    ) {
      return ''
    }
    return url.pathname
  } catch {
    return ''
  }
}

export default function TerminalPage() {
  const containerRef = useRef<HTMLDivElement>(null)
  const initializedRef = useRef(false)
  const [streamPath, setStreamPath] = useState('')
  const [bootstrapError, setBootstrapError] = useState('')

  useEffect(() => {
    const origin = window.location.origin
    const opener = window.opener
    if (!opener) {
      setBootstrapError('This terminal must be opened from the Pod page.')
      return undefined
    }
    const receiveTicket = (event: MessageEvent) => {
      if (
        event.origin !== origin ||
        event.source !== opener ||
        event.data?.type !== 'aiops-terminal-connect'
      )
        return
      const path = validTerminalStreamPath(event.data.wsUrl)
      if (!path) {
        setBootstrapError('Invalid terminal session.')
        return
      }
      setStreamPath(path)
    }
    window.addEventListener('message', receiveTicket)
    opener.postMessage({ type: 'aiops-terminal-ready' }, origin)
    return () => window.removeEventListener('message', receiveTicket)
  }, [])

  useEffect(() => {
    if (!streamPath || initializedRef.current || !containerRef.current) return undefined
    initializedRef.current = true
    document.title = 'Pod Shell terminal'

    let disposed = false
    let ws: WebSocket | undefined
    let terminal: import('xterm').Terminal | undefined
    let removeResizeListener: (() => void) | undefined
    let inputSubscription: { dispose: () => void } | undefined
    let terminalResizeSubscription: { dispose: () => void } | undefined

    const initTerminal = async () => {
      const { Terminal } = await import('xterm')
      const { FitAddon } = await import('xterm-addon-fit')
      // @ts-expect-error CSS side-effect import has no declaration.
      await import('xterm/css/xterm.css')
      if (disposed || !containerRef.current) return

      terminal = new Terminal({
        fontSize: 14,
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
        theme: { background: '#1a1b26' },
        cursorBlink: true,
        scrollback: 10000,
      })
      const fitAddon = new FitAddon()
      terminal.loadAddon(fitAddon)
      terminal.open(containerRef.current)
      fitAddon.fit()

      const resize = () => fitAddon.fit()
      window.addEventListener('resize', resize)
      removeResizeListener = () => window.removeEventListener('resize', resize)
      terminal.write('Connecting...\r\n')

      const endpoint = new URL(streamPath, window.location.origin)
      endpoint.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      endpoint.host = window.location.host
      ws = new WebSocket(endpoint.toString())
      ws.binaryType = 'arraybuffer'

      ws.onopen = () => {
        if (!terminal) return
        terminal.focus()
        fitAddon.fit()
        ws?.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }))
      }
      ws.onmessage = (event) => {
        if (!terminal) return
        if (event.data instanceof ArrayBuffer) {
          const payload = new Uint8Array(event.data)
          const channel = payload[0] ?? 0
          const text = new TextDecoder().decode(
            payload.length > 1 && channel >= 1 && channel <= 3 ? payload.slice(1) : payload,
          )
          terminal.write(channel === 3 ? `\r\n\x1b[31mError: ${text}\x1b[0m\r\n` : text)
          return
        }
        terminal.write(String(event.data))
      }
      ws.onclose = (event) =>
        terminal?.write(`\r\n\x1b[33m[disconnected code=${event.code}]\x1b[0m\r\n`)
      ws.onerror = () => terminal?.write('\r\n\x1b[31mWebSocket error\x1b[0m\r\n')

      inputSubscription = terminal.onData((data: string) => {
        if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'stdin', data }))
      })
      terminalResizeSubscription = terminal.onResize(({ cols, rows }) => {
        if (ws?.readyState === WebSocket.OPEN)
          ws.send(JSON.stringify({ type: 'resize', cols, rows }))
      })
    }
    void initTerminal()

    return () => {
      disposed = true
      removeResizeListener?.()
      inputSubscription?.dispose()
      terminalResizeSubscription?.dispose()
      if (ws && ws.readyState < WebSocket.CLOSING) ws.close()
      terminal?.dispose()
    }
  }, [streamPath])

  return (
    <div
      ref={containerRef}
      style={{
        width: '100vw',
        height: '100vh',
        background: '#1a1b26',
        padding: 4,
        color: '#f8f8f2',
      }}
    >
      {bootstrapError && <div style={{ padding: 16 }}>{bootstrapError}</div>}
    </div>
  )
}
