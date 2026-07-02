/**
 * 日志流 Hook
 * 支持暂停/继续、自动滚动
 */
import { useState, useCallback, useRef, useEffect } from 'react'

interface UseLogStreamOptions {
  maxLines?: number
  autoConnect?: boolean
}

export function useLogStream(wsUrl: string | null, options: UseLogStreamOptions = {}) {
  const { maxLines = 1000, autoConnect = true } = options
  const [logs, setLogs] = useState<string[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [isPaused, setIsPaused] = useState(false)
  const wsRef = useRef<WebSocket>()
  const logsRef = useRef<string[]>([])

  const connect = useCallback(() => {
    if (!wsUrl) return

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => setIsConnected(true)

    ws.onmessage = (event) => {
      if (isPaused) return

      const log = event.data
      logsRef.current = [...logsRef.current.slice(-(maxLines - 1)), log]
      setLogs([...logsRef.current])
    }

    ws.onclose = () => setIsConnected(false)
    ws.onerror = () => setIsConnected(false)
  }, [wsUrl, isPaused, maxLines])

  const disconnect = useCallback(() => {
    wsRef.current?.close()
    wsRef.current = undefined
    setIsConnected(false)
  }, [])

  const clear = useCallback(() => {
    logsRef.current = []
    setLogs([])
  }, [])

  const togglePause = useCallback(() => {
    setIsPaused((prev) => !prev)
  }, [])

  useEffect(() => {
    if (autoConnect && wsUrl) connect()
    return () => disconnect()
  }, [autoConnect, wsUrl, connect, disconnect])

  return {
    logs,
    isConnected,
    isPaused,
    connect,
    disconnect,
    clear,
    togglePause,
  }
}
