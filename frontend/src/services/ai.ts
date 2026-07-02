/**
 * AI 助手 API - 支持 Mock 模式
 * 对话使用 SSE 实现流式输出
 * 当后端不可用时，自动使用本地模拟数据
 */
import { request } from '@umijs/max'
import type {
  AIProvider,
  CreateAIProviderRequest,
  AIModel,
  CreateAIModelRequest,
  AIConversationItem,
  AIConversationListParams,
  AIConversationListResponse,
  AIConversationDetail,
} from '@/types'

// 保留旧接口兼容
export type { AIMessage } from '@/types/ai'

const MOCK_ENABLED = false

// ==================== AI 提供商 ====================

const MOCK_PROVIDERS: AIProvider[] = [
  { id: 1, name: 'OpenAI Production', providerType: 'openai', baseUrl: 'https://api.openai.com/v1', hasApiKey: true, priority: 1, meta: {}, enabled: true, createdAt: '2025-01-15T08:00:00Z', updatedAt: '2025-03-01T10:00:00Z' },
  { id: 2, name: 'Qwen 本地', providerType: 'qwen', baseUrl: 'http://localhost:11434', hasApiKey: false, priority: 10, meta: {}, enabled: true, createdAt: '2025-02-20T14:00:00Z', updatedAt: '2025-02-20T14:00:00Z' },
]

export function listAIProviders(signal?: AbortSignal): Promise<AIProvider[]> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve([...MOCK_PROVIDERS]), 200)
    })
  }
  return request('/api/v1/ai/providers', { signal })
}

export function createAIProvider(data: CreateAIProviderRequest): Promise<AIProvider> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const provider: AIProvider = {
        id: MOCK_PROVIDERS.length + 1,
        name: data.name,
        providerType: data.providerType,
        baseUrl: data.baseUrl,
        hasApiKey: !!data.apiKey,
        priority: data.priority || 1,
        meta: data.meta || {},
        enabled: data.enabled !== false,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }
      MOCK_PROVIDERS.push(provider)
      resolve(provider)
    })
  }
  return request('/api/v1/ai/providers', { method: 'POST', data })
}

export function updateAIProvider(id: number, data: Partial<CreateAIProviderRequest>): Promise<AIProvider> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const idx = MOCK_PROVIDERS.findIndex((p) => p.id === id)
      if (idx === -1) { reject(new Error('提供商不存在')); return }
      const existing = MOCK_PROVIDERS[idx]!
      const updated: AIProvider = {
        ...existing,
        name: data.name ?? existing.name,
        providerType: data.providerType ?? existing.providerType,
        baseUrl: data.baseUrl ?? existing.baseUrl,
        hasApiKey: data.apiKey ? true : existing.hasApiKey,
        priority: data.priority ?? existing.priority,
        meta: data.meta ?? existing.meta,
        enabled: data.enabled ?? existing.enabled,
        updatedAt: new Date().toISOString(),
      }
      MOCK_PROVIDERS[idx] = updated
      resolve(updated)
    })
  }
  return request(`/api/v1/ai/providers/${id}`, { method: 'PATCH', data })
}

// ==================== AI 模型 ====================

const MOCK_MODELS: AIModel[] = [
  { id: 1, providerId: 1, providerName: 'OpenAI Production', name: 'GPT-4.1', modelCode: 'gpt-4.1', modelType: 'chat', maxInputTokens: 128000, supportsTools: true, supportsVision: true, enabled: true, createdAt: '2025-01-15T08:00:00Z', updatedAt: '2025-03-01T10:00:00Z' },
  { id: 2, providerId: 1, providerName: 'OpenAI Production', name: 'GPT-4.1 Mini', modelCode: 'gpt-4.1-mini', modelType: 'chat', maxInputTokens: 128000, supportsTools: true, supportsVision: true, enabled: true, createdAt: '2025-03-01T10:00:00Z', updatedAt: '2025-03-01T10:00:00Z' },
  { id: 3, providerId: 2, providerName: 'Qwen 本地', name: 'Qwen2.5-72B', modelCode: 'qwen2.5-72b', modelType: 'chat', maxInputTokens: 32000, supportsTools: true, supportsVision: false, enabled: true, createdAt: '2025-02-20T14:00:00Z', updatedAt: '2025-02-20T14:00:00Z' },
]

export function listAIModels(signal?: AbortSignal): Promise<AIModel[]> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => resolve([...MOCK_MODELS]), 200)
    })
  }
  return request('/api/v1/ai/models', { signal })
}

export function createAIModel(data: CreateAIModelRequest): Promise<AIModel> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const provider = MOCK_PROVIDERS.find((p) => p.id === data.providerId)
      const model: AIModel = {
        id: MOCK_MODELS.length + 1,
        providerId: data.providerId,
        providerName: provider?.name || '',
        name: data.name,
        modelCode: data.modelCode,
        modelType: data.modelType,
        maxInputTokens: data.maxInputTokens || 4096,
        supportsTools: data.supportsTools || false,
        supportsVision: data.supportsVision || false,
        enabled: data.enabled !== false,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }
      MOCK_MODELS.push(model)
      resolve(model)
    })
  }
  return request('/api/v1/ai/models', { method: 'POST', data })
}

export function updateAIModel(id: number, data: Partial<CreateAIModelRequest>): Promise<AIModel> {
  if (MOCK_ENABLED) {
    return new Promise((resolve, reject) => {
      const idx = MOCK_MODELS.findIndex((m) => m.id === id)
      if (idx !== -1) {
        const existing = MOCK_MODELS[idx]!
        const updated: AIModel = {
          ...existing,
          providerId: data.providerId ?? existing.providerId,
          name: data.name ?? existing.name,
          modelCode: data.modelCode ?? existing.modelCode,
          modelType: data.modelType ?? existing.modelType,
          maxInputTokens: data.maxInputTokens ?? existing.maxInputTokens,
          supportsTools: data.supportsTools ?? existing.supportsTools,
          supportsVision: data.supportsVision ?? existing.supportsVision,
          enabled: data.enabled ?? existing.enabled,
          updatedAt: new Date().toISOString(),
        }
        MOCK_MODELS[idx] = updated
        resolve(updated)
      } else {
        reject(new Error('模型不存在'))
      }
    })
  }
  return request(`/api/v1/ai/models/${id}`, { method: 'PATCH', data })
}

// ==================== AI 对话 ====================

/** 获取 SSE 流式对话 URL */
export function getChatUrl(clusterId: string): string {
  return `/api/v1/clusters/${clusterId}/ai/chat/stream`
}

/** 获取对话历史列表 */
export function listConversations(
  params?: AIConversationListParams,
  signal?: AbortSignal,
): Promise<AIConversationListResponse> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      const items: AIConversationItem[] = [
        { id: 'conv-1', title: '如何查看 Pod 日志？', messageCount: 4, updatedAt: '2025-03-20T10:00:05Z', status: 'completed', assistantMode: 'troubleshoot' },
        { id: 'conv-2', title: 'Deployment 滚动更新失败怎么办？', messageCount: 6, updatedAt: '2025-03-19T14:30:10Z', status: 'completed', assistantMode: 'troubleshoot' },
        { id: 'conv-3', title: '集群资源不足如何扩容？', messageCount: 8, updatedAt: '2025-03-18T09:15:00Z', status: 'completed', assistantMode: 'optimize' },
      ]
      setTimeout(() => resolve({ items, total: items.length, page: params?.page || 1, pageSize: params?.pageSize || 10 }), 200)
    })
  }
  return request('/api/v1/ai/conversations', { params, signal })
}

/** 获取对话详情 */
export function getConversation(id: string, signal?: AbortSignal): Promise<AIConversationDetail> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          id,
          title: '如何查看 Pod 日志？',
          clusterId: 1,
          assistantMode: 'troubleshoot',
          status: 'completed',
          createdByName: 'admin',
          createdBy: 1,
          messages: [
            { id: 'msg-1', role: 'user', content: '如何查看 Pod 日志？', status: 'sent', toolCallCount: 0, tokenInput: 0, tokenOutput: 0, createdAt: '2025-03-20T10:00:00Z' },
            { id: 'msg-2', role: 'assistant', content: '查看 Pod 日志可以使用以下命令：\n\n```bash\nkubectl logs <pod-name> -n <namespace>\n```\n\n也可以使用 `-f` 参数实时跟踪日志。', status: 'sent', toolCallCount: 1, tokenInput: 150, tokenOutput: 80, createdAt: '2025-03-20T10:00:05Z' },
          ],
          toolCalls: [{ id: 'tc-1', toolName: 'kubectl_get_pods', resultSummary: '获取到 3 个运行中的 Pod', errorMessage: '', createdAt: '2025-03-20T10:00:03Z' }],
          suggestedActions: [],
          actionProposals: [],
          createdAt: '2025-03-20T10:00:00Z',
          updatedAt: '2025-03-20T10:00:05Z',
        })
      }, 200)
    })
  }
  return request(`/api/v1/ai/conversations/${id}`, { signal })
}

/** 删除对话 */
export function deleteConversation(id: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { resolve() })
  }
  return request(`/api/v1/ai/conversations/${id}`, { method: 'DELETE' })
}

/** 创建新对话 */
export function createConversation(data: { title: string; clusterId: number; assistantMode?: string }): Promise<AIConversationDetail> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => {
      resolve({
        id: `conv-${Date.now()}`,
        title: data.title,
        clusterId: data.clusterId,
        assistantMode: data.assistantMode || 'troubleshoot',
        status: 'active',
        createdByName: 'admin',
        createdBy: 1,
        messages: [],
        toolCalls: [],
        suggestedActions: [],
        actionProposals: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      })
    })
  }
  return request(`/api/v1/clusters/${data.clusterId}/ai/conversations`, { method: 'POST', data })
}

/** 发送消息（非流式，备用） */
export function sendMessage(clusterId: string, conversationId: string, content: string): Promise<void> {
  if (MOCK_ENABLED) {
    return new Promise((resolve) => { resolve() })
  }
  return request(`/api/v1/clusters/${clusterId}/ai/chat`, { method: 'POST', data: { conversationId, content } })
}
