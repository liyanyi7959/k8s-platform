/**
 * Pod Shell 终端页面
 * 通过 URL 参数接收 ws_url 和 token，建立 WebSocket 连接
 */
import { useEffect, useRef } from 'react'
import { useSearchParams } from '@umijs/max'

export default function TerminalPage() {
  const [searchParams] = useSearchParams()
  const containerRef = useRef<HTMLDivElement>(null)
  const doneRef = useRef(false)

  useEffect(() => {
    if (doneRef.current || !containerRef.current) return
    doneRef.current = true

    const wsUrl = searchParams.get('ws_url')
    const token = searchParams.get('token')

    if (!wsUrl) {
      document.title = '终端 - 参数错误'
      return
    }

    // 构建完整 WebSocket URL（同源，代理/网关负责转发）
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const base = `${protocol}//${window.location.host}`
    const parsed = new URL(wsUrl, 'http://placeholder')
    const search = new URLSearchParams(parsed.search)
    if (token) search.set('token', token)
    const fullWsUrl = base + parsed.pathname + '?' + search.toString()

    document.title = 'Pod Shell 终端'

    const initTerminal = async () => {
      const { Terminal } = await import('xterm')
      const { FitAddon } = await import('xterm-addon-fit')
      // @ts-expect-error 样式副作用导入，无类型声明
      await import('xterm/css/xterm.css')

      const term = new Terminal({
        fontSize: 14,
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
        theme: { background: '#1a1b26' },
        cursorBlink: true,
        scrollback: 10000,
      })

      const fitAddon = new FitAddon()
      term.loadAddon(fitAddon)
      term.open(containerRef.current!)
      fitAddon.fit()

      const onResize = () => fitAddon.fit()
      window.addEventListener('resize', onResize)

      term.write('Connecting...\r\n')

      const ws = new WebSocket(fullWsUrl)
      ws.binaryType = 'arraybuffer'

      ws.onopen = () => {
        term.focus()
        fitAddon.fit()
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }

      ws.onmessage = (event) => {
        if (event.data instanceof ArrayBuffer) {
          const u8 = new Uint8Array(event.data)
          const firstByte = u8[0] ?? 0
          if (u8.length > 1 && firstByte >= 1 && firstByte <= 3) {
            const channel = firstByte
            const text = new TextDecoder().decode(u8.slice(1))
            if (channel === 3) {
              term.write(`\r\n\x1b[31mError: ${text}\x1b[0m\r\n`)
            } else {
              term.write(text)
            }
          } else {
            term.write(new TextDecoder().decode(u8))
          }
        } else {
          term.write(String(event.data))
        }
      }

      ws.onclose = (ev) => {
        term.write(`\r\n\x1b[33m[disconnected code=${ev.code}]\x1b[0m\r\n`)
      }

      ws.onerror = () => {
        term.write('\r\n\x1b[31mWebSocket error\x1b[0m\r\n')
      }

      term.onData((data: string) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'stdin', data }))
        }
      })

      term.onResize(({ cols, rows }) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'resize', cols, rows }))
        }
      })
    }

    initTerminal()
  }, [searchParams])

  return (
    <div
      ref={containerRef}
      style={{ width: '100vw', height: '100vh', background: '#1a1b26', padding: 4 }}
    />
  )
}
