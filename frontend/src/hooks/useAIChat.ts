import { useCallback, useRef, useState } from 'react'
import {
  buildAIChatFetchRequest,
  getAIChatStreamUrl,
  sendAIChatMessage,
} from '@/features/ai/api'
import type {
  AIActionProposal,
  AIChatRequest,
  AIConversationDetail,
  AIMessage,
  AIMessageAttachment,
  AIStreamDoneData,
  AIToolCall,
} from '@/shared/types'

export interface UIChatMessage extends AIMessage {
  localId: string
  isStreaming?: boolean
  isPending?: boolean
}

interface UseAIChatResult {
  messages: UIChatMessage[]
  conversationId?: number
  toolCalls: AIToolCall[]
  actionProposals: AIActionProposal[]
  isStreaming: boolean
  isResponding: boolean
  progress: string
  activeModelName: string
  activeProviderName: string
  hydrateConversation: (detail?: AIConversationDetail | null) => void
  clearConversation: () => void
  stopGeneration: () => void
  sendMessage: (clusterId: number, payload: AIChatRequest, stream: boolean) => Promise<number | undefined>
}

function buildLocalAttachments(files?: File[]): AIMessageAttachment[] {
  return (files || []).map((file, index) => ({
    id: -(Date.now() + index),
    originalName: file.name,
    contentType: file.type || 'application/octet-stream',
    fileSize: file.size,
    purpose: 'chat',
    status: 'local',
    fileKind: file.type.startsWith('image/') ? 'image' : 'text',
    downloadUrl: '',
    createdAt: new Date().toISOString(),
  }))
}

function toUIMessage(message: AIMessage): UIChatMessage {
  return {
    ...message,
    attachments: message.attachments || [],
    localId: `server-${message.id}`,
  }
}

function getAuthHeaders(extra?: HeadersInit): Headers {
  const headers = new Headers(extra || {})
  const token = localStorage.getItem('token')
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  headers.set('Accept', 'text/event-stream')
  return headers
}

export function useAIChat(): UseAIChatResult {
  const [messages, setMessages] = useState<UIChatMessage[]>([])
  const [conversationId, setConversationId] = useState<number | undefined>(undefined)
  const [toolCalls, setToolCalls] = useState<AIToolCall[]>([])
  const [actionProposals, setActionProposals] = useState<AIActionProposal[]>([])
  const [activeModelName, setActiveModelName] = useState('')
  const [activeProviderName, setActiveProviderName] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [isResponding, setIsResponding] = useState(false)
  const [progress, setProgress] = useState('')
  const abortControllerRef = useRef<AbortController | null>(null)

  const hydrateConversation = useCallback((detail?: AIConversationDetail | null) => {
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    setIsStreaming(false)
    setIsResponding(false)
    setConversationId(detail?.id)
    setMessages((detail?.messages || []).map(toUIMessage))
    setToolCalls(detail?.toolCalls || [])
    setActionProposals(detail?.actionProposals || [])
  }, [])

  const clearConversation = useCallback(() => {
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    setIsStreaming(false)
    setIsResponding(false)
    setProgress('')
    setConversationId(undefined)
    setMessages([])
    setToolCalls([])
    setActionProposals([])
    setActiveModelName('')
    setActiveProviderName('')
  }, [])

  const stopGeneration = useCallback(() => {
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    setMessages((current) =>
      current.map((item) =>
        item.isStreaming ? { ...item, isStreaming: false, isPending: false } : item,
      ),
    )
    setIsStreaming(false)
    setIsResponding(false)
    setProgress('')
  }, [])

  const sendMessage = useCallback(async (clusterId: number, payload: AIChatRequest, stream: boolean) => {
    if (isStreaming || isResponding) {
      return conversationId
    }

    const currentConversationId = payload.conversationId || conversationId
    const userMessageLocalId = `user-${Date.now()}`
    const assistantMessageLocalId = `assistant-${Date.now()}`
    const now = new Date().toISOString()

    const userMessage: UIChatMessage = {
      id: 0,
      localId: userMessageLocalId,
      conversationId: currentConversationId || 0,
      role: 'user',
      messageType: 'text',
      content: payload.message,
      attachments: buildLocalAttachments(payload.files),
      status: 'created',
      toolCallCount: 0,
      tokenInput: 0,
      tokenOutput: 0,
      createdBy: 0,
      createdAt: now,
    }
    const assistantMessage: UIChatMessage = {
      id: 0,
      localId: assistantMessageLocalId,
      conversationId: currentConversationId || 0,
      role: 'assistant',
      messageType: 'text',
      content: '',
      attachments: [],
      status: 'created',
      toolCallCount: 0,
      tokenInput: 0,
      tokenOutput: 0,
      createdBy: 0,
      createdAt: now,
      isStreaming: stream,
      isPending: true,
    }

    setMessages((current) => [...current, userMessage, assistantMessage])
    setIsResponding(true)
    setIsStreaming(stream)
    setProgress('')

    if (!stream) {
      try {
        const response = await sendAIChatMessage(clusterId, {
          ...payload,
          conversationId: currentConversationId,
        })
        setConversationId(response.conversationId)
        setActiveModelName(response.modelName || response.modelCode || '')
        setActiveProviderName(response.providerName || '')
        setToolCalls(response.toolCalls || [])
        setActionProposals(response.actionProposals || [])
        setMessages((current) =>
          current.map((item) => {
            if (item.localId === assistantMessageLocalId) {
              return {
                ...item,
                conversationId: response.conversationId,
                content: response.assistantMessage,
                isStreaming: false,
                isPending: false,
              }
            }
            if (item.localId === userMessageLocalId) {
              return {
                ...item,
                conversationId: response.conversationId,
              }
            }
            return item
          }),
        )
        return response.conversationId
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'AI 响应异常'
        setMessages((current) =>
          current.map((item) =>
            item.localId === assistantMessageLocalId
              ? { ...item, content: errorMessage, isStreaming: false, isPending: false, status: 'failed' }
              : item,
          ),
        )
        return currentConversationId
      } finally {
        setIsStreaming(false)
        setIsResponding(false)
      }
    }

    const controller = new AbortController()
    abortControllerRef.current = controller
    try {
      const streamRequest = buildAIChatFetchRequest({
        ...payload,
        conversationId: currentConversationId,
      })
      const response = await fetch(getAIChatStreamUrl(clusterId), {
        method: 'POST',
        credentials: 'include',
        signal: controller.signal,
        body: streamRequest.body,
        headers: getAuthHeaders(streamRequest.headers),
      })

      const contentType = response.headers.get('content-type') || ''
      if (contentType.includes('application/json')) {
        const result = await response.json()
        throw new Error(result?.message || '请求失败')
      }
      if (!response.ok) {
        throw new Error(`AI 服务响应异常: ${response.status}`)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('当前浏览器不支持流式响应')
      }

      const decoder = new TextDecoder()
      let buffer = ''
      let doneData: AIStreamDoneData | null = null

      while (true) {
        const { done, value } = await reader.read()
        if (done) {
          break
        }

        buffer += decoder.decode(value, { stream: true })
        const events = buffer.split('\n\n')
        buffer = events.pop() || ''

        for (const eventText of events) {
          const dataLine = eventText
            .split('\n')
            .find((line) => line.startsWith('data: '))
          if (!dataLine) {
            continue
          }
          const payloadText = dataLine.slice(6)
          const chunk = JSON.parse(payloadText) as {
            type: 'chunk' | 'done' | 'error' | 'progress'
            content?: string
            error?: string
            progress?: string
          }

          if (chunk.type === 'progress') {
            setProgress(chunk.progress || '')
            continue
          }
          if (chunk.type === 'chunk') {
            setMessages((current) =>
              current.map((item) =>
                item.localId === assistantMessageLocalId
                  ? { ...item, content: item.content + (chunk.content || '') }
                  : item,
              ),
            )
            continue
          }
          if (chunk.type === 'error') {
            throw new Error(chunk.error || 'AI 响应异常')
          }
          if (chunk.type === 'done') {
            doneData = chunk.content ? (JSON.parse(chunk.content) as AIStreamDoneData) : null
          }
        }
      }

      if (doneData) {
        setConversationId(doneData.conversationId)
        setActiveModelName(doneData.modelName || doneData.modelCode || '')
        setActiveProviderName(doneData.providerName || '')
        setToolCalls(doneData.toolCalls || [])
        setActionProposals(doneData.actionProposals || [])
        setMessages((current) =>
          current.map((item) => {
            if (item.localId === assistantMessageLocalId) {
              return {
                ...item,
                conversationId: doneData!.conversationId,
                isStreaming: false,
                isPending: false,
              }
            }
            if (item.localId === userMessageLocalId) {
              return {
                ...item,
                conversationId: doneData!.conversationId,
              }
            }
            return item
          }),
        )
        return doneData.conversationId
      }

      return currentConversationId
    } catch (error) {
      if (!(error instanceof DOMException && error.name === 'AbortError')) {
        const errorMessage = error instanceof Error ? error.message : 'AI 响应异常'
        setMessages((current) =>
          current.map((item) =>
            item.localId === assistantMessageLocalId
              ? { ...item, content: errorMessage, isStreaming: false, isPending: false, status: 'failed' }
              : item,
          ),
        )
      }
      return currentConversationId
    } finally {
      abortControllerRef.current = null
      setMessages((current) =>
        current.map((item) =>
          item.localId === assistantMessageLocalId
            ? { ...item, isStreaming: false, isPending: false }
            : item,
        ),
      )
      setIsStreaming(false)
      setIsResponding(false)
    }
  }, [conversationId, isResponding, isStreaming])

  return {
    messages,
    conversationId,
    toolCalls,
    actionProposals,
    isStreaming,
    isResponding,
    progress,
    activeModelName,
    activeProviderName,
    hydrateConversation,
    clearConversation,
    stopGeneration,
    sendMessage,
  }
}
