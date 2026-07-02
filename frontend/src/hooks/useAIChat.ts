/**
 * AI 对话 Hook
 * 支持 SSE 流式输出和手动停止，Mock 模式下使用本地模拟
 */
import { useState, useCallback, useRef } from 'react'

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  isStreaming?: boolean
}

const MOCK_ENABLED = false

/** Mock 回复内容 */
const MOCK_RESPONSES: Record<string, string> = {
  default:
    'Kubernetes 常用运维命令：\n\n' +
    '1. 查看 Pod 状态：`kubectl get pods -A`\n' +
    '2. 查看日志：`kubectl logs <pod-name> -n <namespace>`\n' +
    '3. 进入容器：`kubectl exec -it <pod-name> -- /bin/sh`\n' +
    '4. 查看事件：`kubectl get events --sort-by=.metadata.creationTimestamp`\n\n' +
    '您可以在 K8s 运维页面中直接操作这些资源。',
  restart:
    'Pod 频繁重启的排查步骤：\n\n' +
    '1. 查看 Pod 事件：`kubectl describe pod <pod-name>`\n' +
    '2. 查看容器日志：`kubectl logs <pod-name> --previous`\n' +
    '3. 检查资源限制：确认 requests/limits 是否合理\n' +
    '4. 检查健康检查：liveness probe 配置是否正确\n' +
    '5. 检查依赖服务：数据库、缓存等是否可用\n\n' +
    '常见原因：OOMKilled、健康检查失败、依赖服务不可用。',
  deploy:
    'Deployment 滚动更新最佳实践：\n\n' +
    '1. 设置合理的 `maxUnavailable` 和 `maxSurge`\n' +
    '2. 配置 `readinessProbe` 确保新 Pod 就绪后再切流量\n' +
    '3. 使用 `kubectl rollout status` 监控更新进度\n' +
    '4. 准备回滚方案：`kubectl rollout undo deployment/<name>`\n\n' +
    '建议在生产环境使用 Canary 或 Blue-Green 策略。',
}

function getMockResponse(input: string): string {
  const lower = input.toLowerCase()
  if (lower.includes('重启') || lower.includes('restart')) return MOCK_RESPONSES.restart || ''
  if (lower.includes('部署') || lower.includes('deploy')) return MOCK_RESPONSES.deploy || ''
  return MOCK_RESPONSES.default || ''
}

export function useAIChat(clusterId?: string) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const abortControllerRef = useRef<AbortController | null>(null)
  const mockTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const sendMessage = useCallback(
    async (userInput: string) => {
      if (isStreaming) return

      const userMsg: ChatMessage = {
        id: crypto.randomUUID(),
        role: 'user',
        content: userInput,
      }

      const assistantMsgId = crypto.randomUUID()
      const assistantMsg: ChatMessage = {
        id: assistantMsgId,
        role: 'assistant',
        content: '',
        isStreaming: true,
      }

      setMessages((prev) => [...prev, userMsg, assistantMsg])
      setIsStreaming(true)

      if (MOCK_ENABLED) {
        // Mock 模式：逐字输出模拟流式响应
        const fullText = getMockResponse(userInput)
        let charIndex = 0

        const tick = () => {
          if (charIndex >= fullText.length) {
            setMessages((prev) =>
              prev.map((m) => (m.id === assistantMsgId ? { ...m, isStreaming: false } : m)),
            )
            setIsStreaming(false)
            mockTimerRef.current = null
            return
          }

          // 每次输出 2-4 个字符，模拟打字效果
          const chunkSize = Math.floor(Math.random() * 3) + 2
          const chunk = fullText.slice(charIndex, charIndex + chunkSize)
          charIndex += chunkSize

          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantMsgId ? { ...m, content: m.content + chunk } : m,
            ),
          )

          mockTimerRef.current = setTimeout(tick, 30 + Math.random() * 40)
        }

        mockTimerRef.current = setTimeout(tick, 500)
        return
      }

      // 真实模式：SSE 流式请求
      const controller = new AbortController()
      abortControllerRef.current = controller

      try {
        const response = await fetch(clusterId ? `/api/v1/clusters/${clusterId}/ai/chat/stream` : '/api/v1/ai/chat', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
          credentials: 'include',
          body: JSON.stringify({ message: userInput }),
          signal: controller.signal,
        })

        if (!response.ok) throw new Error(`AI 服务响应异常: ${response.status}`)

        const reader = response.body?.getReader()
        if (!reader) throw new Error('不支持流式响应')

        const decoder = new TextDecoder()
        let buffer = ''

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n\n')
          buffer = lines.pop() || ''

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              const jsonStr = line.slice(6)
              if (jsonStr === '[DONE]') continue
              try {
                const data = JSON.parse(jsonStr)
                setMessages((prev) =>
                  prev.map((m) =>
                    m.id === assistantMsgId
                      ? { ...m, content: m.content + (data.content || '') }
                      : m,
                  ),
                )
              } catch {
                // 跳过非 JSON 行
              }
            }
          }
        }
      } catch (err: unknown) {
        if (err instanceof DOMException && err.name === 'AbortError') {
          // 用户手动停止
        } else {
          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantMsgId
                ? { ...m, content: 'AI 响应异常，请重试', isStreaming: false }
                : m,
            ),
          )
        }
      } finally {
        setMessages((prev) =>
          prev.map((m) => (m.id === assistantMsgId ? { ...m, isStreaming: false } : m)),
        )
        setIsStreaming(false)
        abortControllerRef.current = null
      }
    },
    [isStreaming],
  )

  const stopGeneration = useCallback(() => {
    if (mockTimerRef.current) {
      clearTimeout(mockTimerRef.current)
      mockTimerRef.current = null
    }
    abortControllerRef.current?.abort()
    setMessages((prev) => prev.map((m) => (m.isStreaming ? { ...m, isStreaming: false } : m)))
    setIsStreaming(false)
  }, [])

  const clearMessages = useCallback(() => {
    setMessages([])
  }, [])

  return { messages, isStreaming, sendMessage, stopGeneration, clearMessages }
}
