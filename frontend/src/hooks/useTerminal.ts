/**
 * 终端连接 Hook
 * 管理 xterm.js 终端实例和 WebSocket 连接
 */
import { useRef, useCallback, useEffect } from 'react'
import type { Terminal as XTerminal } from 'xterm'

interface UseTerminalOptions {
  fontSize?: number
  fontFamily?: string
  theme?: Record<string, string>
}

export function useTerminal(options: UseTerminalOptions = {}) {
  const termRef = useRef<XTerminal>()
  const wsRef = useRef<WebSocket>()
  const cleanupRef = useRef<() => void>()

  const init = useCallback(
    async (container: HTMLDivElement) => {
      const { Terminal } = await import('xterm')
      const { FitAddon } = await import('xterm-addon-fit')
      const { WebLinksAddon } = await import('xterm-addon-web-links')

      const term = new Terminal({
        fontSize: options.fontSize || 13,
        fontFamily: options.fontFamily || 'ui-monospace, SFMono-Regular, Menlo, monospace',
        theme: (options.theme as Record<string, string>) || { background: '#1a1b26' },
        cursorBlink: true,
        scrollback: 10000,
      })

      const fitAddon = new FitAddon()
      term.loadAddon(fitAddon)
      term.loadAddon(new WebLinksAddon())
      term.open(container)
      fitAddon.fit()

      termRef.current = term

      const handleResize = () => fitAddon.fit()
      window.addEventListener('resize', handleResize)

      cleanupRef.current = () => {
        window.removeEventListener('resize', handleResize)
        wsRef.current?.close()
        term.dispose()
      }

      return term
    },
    [options],
  )

  const connect = useCallback((wsUrl: string) => {
    if (!termRef.current) return

    const ws = new WebSocket(wsUrl)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => termRef.current?.focus()

    ws.onmessage = (event) => {
      const data =
        event.data instanceof ArrayBuffer ? new TextDecoder().decode(event.data) : event.data
      termRef.current?.write(data)
    }

    ws.onclose = () => {
      termRef.current?.write('\r\n\x1b[31m[连接已断开]\x1b[0m\r\n')
    }

    termRef.current.onData((data: string) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(data)
    })

    termRef.current.onResize(({ cols, rows }: { cols: number; rows: number }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols, rows }))
      }
    })
  }, [])

  useEffect(() => {
    return () => cleanupRef.current?.()
  }, [])

  return { init, connect, term: termRef }
}
