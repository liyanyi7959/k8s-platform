/**
 * WebSocket 连接 Hook
 * 支持心跳、重连、熔断
 */
import { useRef, useEffect, useCallback, useState } from 'react'

interface UseWebSocketOptions {
  url: string | null
  onMessage?: (data: string) => void
  onError?: (event: Event) => void
  onClose?: (event: CloseEvent) => void
  heartbeatInterval?: number
  reconnectMaxRetries?: number
  reconnectBaseDelay?: number
}

export function useWebSocket({
  url,
  onMessage,
  onError,
  onClose,
  heartbeatInterval = 30_000,
  reconnectMaxRetries = 5,
  reconnectBaseDelay = 1_000,
}: UseWebSocketOptions) {
  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const retryCountRef = useRef(0)
  const heartbeatTimerRef = useRef<ReturnType<typeof setInterval>>()
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout>>()

  const cleanup = useCallback(() => {
    if (heartbeatTimerRef.current) clearInterval(heartbeatTimerRef.current)
    if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
    wsRef.current?.close()
    wsRef.current = null
    setIsConnected(false)
  }, [])

  const connect = useCallback(() => {
    if (!url) return
    cleanup()

    const ws = new WebSocket(url)
    wsRef.current = ws

    heartbeatTimerRef.current = setInterval(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, heartbeatInterval)

    ws.onopen = () => {
      setIsConnected(true)
      retryCountRef.current = 0
    }

    ws.onmessage = (event) => {
      onMessage?.(event.data)
    }

    ws.onerror = (event) => {
      onError?.(event)
    }

    ws.onclose = (event) => {
      setIsConnected(false)
      onClose?.(event)

      // 指数退避重连
      if (retryCountRef.current < reconnectMaxRetries) {
        const delay = reconnectBaseDelay * Math.pow(2, retryCountRef.current)
        reconnectTimerRef.current = setTimeout(() => {
          retryCountRef.current++
          connect()
        }, delay)
      }
    }
  }, [url, heartbeatInterval, reconnectMaxRetries, reconnectBaseDelay, cleanup, onMessage, onError, onClose])

  useEffect(() => {
    connect()
    return cleanup
  }, [connect, cleanup])

  const send = useCallback((data: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(data)
    }
  }, [])

  return { isConnected, send, close: cleanup, reconnect: connect }
}
