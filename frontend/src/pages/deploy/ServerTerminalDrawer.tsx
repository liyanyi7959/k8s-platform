import { useEffect, useRef, useState } from 'react'
import { Alert, Button, Drawer, Space, Tag, Typography } from 'antd'
import { DisconnectOutlined, ReloadOutlined } from '@ant-design/icons'
import { createServerTerminalSession } from '@/services/deploy'
import type { DeployServer } from '@/types/deploy'

const { Text } = Typography

type TerminalStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

const WS_CONNECT_TIMEOUT_MS = 10_000

const statusMeta: Record<TerminalStatus, { color: string; text: string }> = {
  connecting: { color: 'processing', text: '正在连接' },
  connected: { color: 'success', text: '已连接' },
  disconnected: { color: 'default', text: '已断开' },
  error: { color: 'error', text: '连接失败' },
}

interface ServerTerminalDrawerProps {
  open: boolean
  server: DeployServer | null
  onClose: () => void
}

export default function ServerTerminalDrawer({ open, server, onClose }: ServerTerminalDrawerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [status, setStatus] = useState<TerminalStatus>('connecting')
  const [errorMessage, setErrorMessage] = useState('')
  const [connectionVersion, setConnectionVersion] = useState(0)

  useEffect(() => {
    if (!open || !server || !containerRef.current) return

    let disposed = false
    let socket: WebSocket | undefined
    let terminal: import('xterm').Terminal | undefined
    let removeResizeListener: (() => void) | undefined
    let connectionTimer: number | undefined

    setStatus('connecting')
    setErrorMessage('')

    const connect = async () => {
      try {
        const [{ Terminal }, { FitAddon }, { WebLinksAddon }] = await Promise.all([
          import('xterm'),
          import('xterm-addon-fit'),
          import('xterm-addon-web-links'),
          // @ts-expect-error xterm 样式包没有类型声明
          import('xterm/css/xterm.css'),
        ])
        if (disposed || !containerRef.current) return

        terminal = new Terminal({
          cursorBlink: true,
          cursorStyle: 'bar',
          fontSize: 13,
          lineHeight: 1.25,
          fontFamily: "'JetBrains Mono', 'Cascadia Code', Consolas, monospace",
          scrollback: 10000,
          allowProposedApi: false,
          theme: {
            background: '#0b1220',
            foreground: '#d7e2f0',
            cursor: '#60a5fa',
            cursorAccent: '#0b1220',
            selectionBackground: '#1d4ed880',
            black: '#111827',
            brightBlack: '#64748b',
            blue: '#60a5fa',
            brightBlue: '#93c5fd',
            green: '#34d399',
            brightGreen: '#6ee7b7',
            red: '#f87171',
            brightRed: '#fca5a5',
            yellow: '#fbbf24',
            brightYellow: '#fde68a',
          },
        })
        const fitAddon = new FitAddon()
        terminal.loadAddon(fitAddon)
        terminal.loadAddon(new WebLinksAddon())
        terminal.open(containerRef.current)
        fitAddon.fit()
        terminal.write(`\x1b[38;5;75mAIOPS Secure Shell\x1b[0m  ${server.user}@${server.ip}:${server.sshPort}\r\n`)
        terminal.write('\x1b[38;5;244m正在申请一次性终端会话…\x1b[0m\r\n')

        const resize = () => {
          if (!terminal) return
          fitAddon.fit()
          if (socket?.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }))
          }
        }
        window.addEventListener('resize', resize)
        removeResizeListener = () => window.removeEventListener('resize', resize)

        const session = await createServerTerminalSession(server.id)
        if (disposed || !terminal) return
        terminal.write('\r\n\x1b[38;5;244m会话已创建，正在建立 WebSocket 安全通道…\x1b[0m\r\n')

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const parsedUrl = new URL(session.wsUrl, window.location.origin)
        const token = localStorage.getItem('token')
        if (token) parsedUrl.searchParams.set('token', token)
        parsedUrl.protocol = protocol
        parsedUrl.host = window.location.host

        socket = new WebSocket(parsedUrl.toString())
        socket.binaryType = 'arraybuffer'
        connectionTimer = window.setTimeout(() => {
          if (disposed || socket?.readyState !== WebSocket.CONNECTING) return
          setStatus('error')
          setErrorMessage('WebSocket 连接超时，请检查前端代理或网关是否允许该终端地址升级连接。')
          terminal?.write('\r\n\x1b[31mWebSocket 连接超时。\x1b[0m\r\n')
          socket.close()
        }, WS_CONNECT_TIMEOUT_MS)
        socket.onopen = () => {
          if (!terminal || disposed) return
          if (connectionTimer) window.clearTimeout(connectionTimer)
          setStatus('connected')
          terminal.write('\x1b[38;5;71m会话已建立。\x1b[0m\r\n')
          fitAddon.fit()
          socket?.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }))
          terminal.focus()
        }
        socket.onmessage = (event) => {
          if (!terminal) return
          if (event.data instanceof ArrayBuffer) {
            const payload = new Uint8Array(event.data)
            const channel = payload[0]
            const content = new TextDecoder().decode(payload.slice(1))
            if (channel === 3) {
              terminal.write(`\r\n\x1b[31m${content}\x1b[0m\r\n`)
            } else {
              terminal.write(content)
            }
            return
          }
          terminal.write(String(event.data))
        }
        socket.onerror = () => {
          if (disposed) return
          if (connectionTimer) window.clearTimeout(connectionTimer)
          setStatus('error')
          setErrorMessage('WebSocket 连接失败，请检查网关是否允许连接升级。')
        }
        socket.onclose = (event) => {
          if (disposed) return
          if (connectionTimer) window.clearTimeout(connectionTimer)
          setStatus(event.code === 1000 ? 'disconnected' : 'error')
          if (event.reason) setErrorMessage(event.reason)
          terminal?.write(`\r\n\x1b[38;5;214m[会话已断开 code=${event.code}]\x1b[0m\r\n`)
        }

        terminal.onData((data) => {
          if (socket?.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ type: 'stdin', data }))
          }
        })
        terminal.onResize(({ cols, rows }) => {
          if (socket?.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({ type: 'resize', cols, rows }))
          }
        })
      } catch (error: any) {
        if (disposed) return
        setStatus('error')
        setErrorMessage(error?.message || '无法建立服务器终端会话')
        terminal?.write(`\r\n\x1b[31m${error?.message || '连接失败'}\x1b[0m\r\n`)
      }
    }

    void connect()
    return () => {
      disposed = true
      if (connectionTimer) window.clearTimeout(connectionTimer)
      removeResizeListener?.()
      socket?.close(1000, 'drawer closed')
      terminal?.dispose()
    }
  }, [connectionVersion, open, server])

  const meta = statusMeta[status]

  return (
    <Drawer
      title={
        <div className="app-server-terminal__title">
          <Space size={10}>
            <span className="app-server-terminal__signal" />
            <span>服务器终端</span>
            {server ? <Text type="secondary">{server.name}</Text> : null}
          </Space>
          <Tag color={meta.color}>{meta.text}</Tag>
        </div>
      }
      open={open}
      onClose={onClose}
      width="min(960px, 92vw)"
      destroyOnClose
      className="app-server-terminal"
      extra={
        <Space>
          <Button
            size="small"
            icon={<ReloadOutlined />}
            onClick={() => setConnectionVersion((current) => current + 1)}
          >
            重新连接
          </Button>
          <Button size="small" icon={<DisconnectOutlined />} onClick={onClose}>
            断开
          </Button>
        </Space>
      }
    >
      {errorMessage ? (
        <Alert
          type="error"
          showIcon
          message="终端连接异常"
          description={errorMessage}
          closable
          onClose={() => setErrorMessage('')}
          style={{ marginBottom: 12 }}
        />
      ) : null}
      <div className="app-server-terminal__meta">
        <span>目标</span>
        <code>{server ? `${server.user}@${server.ip}:${server.sshPort}` : '-'}</code>
        <span>会话</span>
        <code>一次性 · 关闭即销毁</code>
      </div>
      <div ref={containerRef} className="app-server-terminal__viewport" />
    </Drawer>
  )
}
