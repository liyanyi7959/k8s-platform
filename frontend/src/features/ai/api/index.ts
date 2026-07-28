import { request } from '@umijs/max'
import type {
  AIActionProposal,
  AIChatRequest,
  AIChatResponse,
  AIConversationDetail,
  AIConversationItem,
  AIConversationListParams,
  AIConversationListResponse,
  AIMessage,
  AIMessageAttachment,
  AIModel,
  AIProvider,
  AIRouteSettings,
  AIToolCall,
  CreateAIConversationRequest,
  CreateAIModelRequest,
  CreateAIProviderRequest,
  UpdateAIRouteSettingsRequest,
} from '@/shared/types'

type AnyRecord = Record<string, unknown>
const AI_CHAT_REQUEST_TIMEOUT = 180_000

function toCamelCase(input: string): string {
  return input.replace(/_([a-z])/g, (_, char: string) => char.toUpperCase())
}

function camelize<T>(input: unknown): T {
  if (Array.isArray(input)) {
    return input.map((item) => camelize(item)) as T
  }
  if (input && typeof input === 'object' && !(input instanceof Date)) {
    return Object.entries(input as AnyRecord).reduce<AnyRecord>((acc, [key, value]) => {
      acc[toCamelCase(key)] = camelize(value)
      return acc
    }, {}) as T
  }
  return input as T
}

function mapAIProvider(raw: unknown): AIProvider {
  const data = camelize<any>(raw)
  return {
    id: Number(data.id || 0),
    name: data.name || '',
    providerType: data.providerType || '',
    vendorCode: data.vendorCode || '',
    baseUrl: data.baseUrl || '',
    authScheme: data.authScheme || 'bearer',
    hasApiKey: Boolean(data.hasApiKey),
    priority: Number(data.priority || 0),
    meta: (data.meta || {}) as Record<string, unknown>,
    enabled: data.enabled !== false,
    createdAt: data.createdAt || '',
    updatedAt: data.updatedAt || '',
  }
}

function mapAIModel(raw: unknown): AIModel {
  const data = camelize<any>(raw)
  return {
    id: Number(data.id || 0),
    providerId: Number(data.providerId || 0),
    providerName: data.providerName || '',
    name: data.name || '',
    modelCode: data.modelCode || '',
    modelType: data.modelType || '',
    maxInputTokens: Number(data.maxInputTokens || 0),
    maxOutputTokens: Number(data.maxOutputTokens || 0),
    contextWindow: Number(data.contextWindow || 0),
    supportsTools: Boolean(data.supportsTools),
    supportsVision: Boolean(data.supportsVision),
    supportsStreaming: Boolean(data.supportsStreaming),
    supportsReasoning: Boolean(data.supportsReasoning),
    supportsStructuredOutput: Boolean(data.supportsStructuredOutput),
    supportsImageGeneration: Boolean(data.supportsImageGeneration),
    supportsFileInput: Boolean(data.supportsFileInput),
    enabled: data.enabled !== false,
    meta: (data.meta || {}) as Record<string, unknown>,
    createdAt: data.createdAt || '',
    updatedAt: data.updatedAt || '',
  }
}

function mapAIToolCall(raw: unknown): AIToolCall {
  const data = camelize<any>(raw)
  return {
    id: Number(data.id || 0),
    messageId: data.messageId ? Number(data.messageId) : undefined,
    toolName: data.toolName || '',
    toolKind: data.toolKind || '',
    status: data.status || '',
    riskLevel: data.riskLevel || '',
    confirmLevel: data.confirmLevel || '',
    resultSummary: data.resultSummary || '',
    result: data.result || undefined,
    errorMessage: data.errorMessage || '',
    createdAt: data.createdAt || '',
  }
}

function mapAIActionProposal(raw: unknown): AIActionProposal {
  const data = camelize<any>(raw)
  return {
    id: Number(data.id || 0),
    conversationId: Number(data.conversationId || 0),
    messageId: data.messageId ? Number(data.messageId) : undefined,
    toolCallId: data.toolCallId ? Number(data.toolCallId) : undefined,
    clusterId: Number(data.clusterId || 0),
    actionType: data.actionType || '',
    targetKind: data.targetKind || '',
    targetNamespace: data.targetNamespace || '',
    targetName: data.targetName || '',
    riskLevel: data.riskLevel || '',
    confirmLevel: data.confirmLevel || '',
    status: data.status || '',
    title: data.title || '',
    summary: data.summary || '',
    change: data.change || undefined,
    createdBy: Number(data.createdBy || 0),
    createdByName: data.createdByName || '',
    approvedBy: data.approvedBy ? Number(data.approvedBy) : undefined,
    approvedByName: data.approvedByName || '',
    approvedAt: data.approvedAt || undefined,
    secondApprovedBy: data.secondApprovedBy ? Number(data.secondApprovedBy) : undefined,
    secondApprovedName: data.secondApprovedName || '',
    secondApprovedAt: data.secondApprovedAt || undefined,
    requiredConfirmationText: data.requiredConfirmationText || '',
    latestExecution: data.latestExecution || null,
    executions: data.executions || [],
    createdAt: data.createdAt || '',
    updatedAt: data.updatedAt || '',
  }
}

function mapAIMessageAttachment(raw: unknown): AIMessageAttachment {
  const data = camelize<any>(raw)
  return {
    id: Number(data.id || 0),
    conversationId: data.conversationId ? Number(data.conversationId) : undefined,
    messageId: data.messageId ? Number(data.messageId) : undefined,
    originalName: data.originalName || '',
    contentType: data.contentType || '',
    fileSize: Number(data.fileSize || 0),
    purpose: data.purpose || '',
    status: data.status || '',
    fileKind: data.fileKind || '',
    downloadUrl: data.downloadUrl || '',
    createdAt: data.createdAt || '',
  }
}

function mapAIMessage(raw: unknown): AIMessage {
  const data = camelize<any>(raw)
  return {
    id: Number(data.id || 0),
    conversationId: Number(data.conversationId || 0),
    role: data.role || 'assistant',
    messageType: data.messageType || 'text',
    content: data.content || '',
    attachments: Array.isArray(data.attachments) ? data.attachments.map(mapAIMessageAttachment) : [],
    structured: data.structured || undefined,
    status: data.status || '',
    toolCallCount: Number(data.toolCallCount || 0),
    tokenInput: Number(data.tokenInput || 0),
    tokenOutput: Number(data.tokenOutput || 0),
    createdBy: Number(data.createdBy || 0),
    createdAt: data.createdAt || '',
  }
}

function mapAIConversationItem(raw: unknown): AIConversationItem {
  const data = camelize<any>(raw)
  return {
    id: Number(data.id || 0),
    clusterId: Number(data.clusterId || 0),
    providerId: data.providerId ? Number(data.providerId) : undefined,
    modelId: data.modelId ? Number(data.modelId) : undefined,
    title: data.title || '',
    status: data.status || '',
    assistantMode: data.assistantMode || 'diagnose',
    summary: data.summary || '',
    createdBy: Number(data.createdBy || 0),
    createdByName: data.createdByName || '',
    messageCount: Number(data.messageCount || 0),
    lastMessageAt: data.lastMessageAt || undefined,
    createdAt: data.createdAt || '',
    updatedAt: data.updatedAt || '',
  }
}

function mapAIConversationDetail(raw: unknown): AIConversationDetail {
  const data = camelize<any>(raw)
  return {
    ...mapAIConversationItem(data),
    messages: Array.isArray(data.messages) ? data.messages.map(mapAIMessage) : [],
    toolCalls: Array.isArray(data.toolCalls) ? data.toolCalls.map(mapAIToolCall) : [],
    actionProposals: Array.isArray(data.actionProposals) ? data.actionProposals.map(mapAIActionProposal) : [],
  }
}

function mapAIRouteSettings(raw: unknown): AIRouteSettings {
  const data = camelize<any>(raw)
  return {
    id: Number(data.id || 0),
    defaultChatModelId: data.defaultChatModelId ? Number(data.defaultChatModelId) : undefined,
    defaultDiagnoseModelId: data.defaultDiagnoseModelId ? Number(data.defaultDiagnoseModelId) : undefined,
    defaultVisionModelId: data.defaultVisionModelId ? Number(data.defaultVisionModelId) : undefined,
    defaultImageGenerationModelId: data.defaultImageGenerationModelId ? Number(data.defaultImageGenerationModelId) : undefined,
    defaultFallbackProviderId: data.defaultFallbackProviderId ? Number(data.defaultFallbackProviderId) : undefined,
    routingStrategy: data.routingStrategy || 'priority_first',
    allowFallback: Boolean(data.allowFallback),
    meta: (data.meta || {}) as Record<string, unknown>,
    createdAt: data.createdAt || '',
    updatedAt: data.updatedAt || '',
  }
}

function toAIProviderPayload(data: Partial<CreateAIProviderRequest>) {
  return {
    name: data.name,
    provider_type: data.providerType,
    vendor_code: data.vendorCode,
    base_url: data.baseUrl,
    auth_scheme: data.authScheme,
    api_key: data.apiKey,
    priority: data.priority,
    enabled: data.enabled,
    meta: data.meta,
  }
}

function toAIModelPayload(data: Partial<CreateAIModelRequest>) {
  return {
    provider_id: data.providerId,
    name: data.name,
    model_code: data.modelCode,
    model_type: data.modelType,
    max_input_tokens: data.maxInputTokens,
    max_output_tokens: data.maxOutputTokens,
    context_window: data.contextWindow,
    supports_tools: data.supportsTools,
    supports_vision: data.supportsVision,
    supports_streaming: data.supportsStreaming,
    supports_reasoning: data.supportsReasoning,
    supports_structured_output: data.supportsStructuredOutput,
    supports_image_generation: data.supportsImageGeneration,
    supports_file_input: data.supportsFileInput,
    enabled: data.enabled,
    meta: data.meta,
  }
}

function toRouteSettingsPayload(data: UpdateAIRouteSettingsRequest) {
  return {
    default_chat_model_id: data.defaultChatModelId ?? null,
    default_diagnose_model_id: data.defaultDiagnoseModelId ?? null,
    default_vision_model_id: data.defaultVisionModelId ?? null,
    default_image_generation_model_id: data.defaultImageGenerationModelId ?? null,
    default_fallback_provider_id: data.defaultFallbackProviderId ?? null,
    routing_strategy: data.routingStrategy,
    allow_fallback: data.allowFallback,
    meta: data.meta,
  }
}

function toAIChatPayload(data: AIChatRequest) {
  return {
    conversation_id: data.conversationId,
    message: data.message,
    assistant_mode: data.assistantMode,
    provider_id: data.providerId,
    model_id: data.modelId,
    prefer_model: data.preferModel,
    namespace: data.namespace,
    resource_kind: data.resourceKind,
    resource_name: data.resourceName,
    images:
      data.images?.map((image) => ({
        name: image.name,
        content_type: image.contentType,
        data_url: image.dataUrl,
        size: image.size,
      })) || [],
  }
}

export function buildAIChatFormData(data: AIChatRequest): FormData {
  const formData = new FormData()
  formData.append('message', data.message)
  if (data.conversationId) formData.append('conversation_id', String(data.conversationId))
  if (data.assistantMode) formData.append('assistant_mode', data.assistantMode)
  if (data.providerId) formData.append('provider_id', String(data.providerId))
  if (data.modelId) formData.append('model_id', String(data.modelId))
  if (data.preferModel) formData.append('prefer_model', data.preferModel)
  if (data.namespace) formData.append('namespace', data.namespace)
  if (data.resourceKind) formData.append('resource_kind', data.resourceKind)
  if (data.resourceName) formData.append('resource_name', data.resourceName)
  if (data.images?.length) {
    formData.append('images', JSON.stringify(toAIChatPayload(data).images))
  }
  data.files?.forEach((file) => {
    formData.append('files', file)
  })
  return formData
}

export function getAIChatStreamUrl(clusterId: number | string): string {
  return `/api/v1/clusters/${clusterId}/ai/chat/stream`
}

export function getChatUrl(clusterId: number | string): string {
  return getAIChatStreamUrl(clusterId)
}

export function buildAIChatFetchRequest(data: AIChatRequest): Pick<RequestInit, 'body' | 'headers'> {
  const hasFiles = Boolean(data.files?.length)
  if (hasFiles) {
    return { body: buildAIChatFormData(data) }
  }
  return {
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    },
    body: JSON.stringify(toAIChatPayload(data)),
  }
}

export function listAIProviders(signal?: AbortSignal): Promise<AIProvider[]> {
  return request<unknown[]>('/api/v1/ai/providers', { signal }).then((items) => (items || []).map(mapAIProvider))
}

export function createAIProvider(data: CreateAIProviderRequest): Promise<{ id: number }> {
  return request<any>('/api/v1/ai/providers', {
    method: 'POST',
    data: toAIProviderPayload(data),
  }).then((result) => ({ id: Number(result?.id || 0) }))
}

export function updateAIProvider(id: number, data: Partial<CreateAIProviderRequest>): Promise<void> {
  return request(`/api/v1/ai/providers/${id}`, {
    method: 'PATCH',
    data: toAIProviderPayload(data),
  })
}

export function deleteAIProvider(id: number): Promise<void> {
  return request(`/api/v1/ai/providers/${id}`, { method: 'DELETE' })
}

export function listAIModels(signal?: AbortSignal): Promise<AIModel[]> {
  return request<unknown[]>('/api/v1/ai/models', { signal }).then((items) => (items || []).map(mapAIModel))
}

export function createAIModel(data: CreateAIModelRequest): Promise<{ id: number }> {
  return request<any>('/api/v1/ai/models', {
    method: 'POST',
    data: toAIModelPayload(data),
  }).then((result) => ({ id: Number(result?.id || 0) }))
}

export function updateAIModel(id: number, data: Partial<CreateAIModelRequest>): Promise<void> {
  return request(`/api/v1/ai/models/${id}`, {
    method: 'PATCH',
    data: toAIModelPayload(data),
  })
}

export function deleteAIModel(id: number): Promise<void> {
  return request(`/api/v1/ai/models/${id}`, { method: 'DELETE' })
}

export function getAIRouteSettings(signal?: AbortSignal): Promise<AIRouteSettings> {
  return request('/api/v1/ai/route-settings', { signal }).then(mapAIRouteSettings)
}

export function updateAIRouteSettings(data: UpdateAIRouteSettingsRequest): Promise<AIRouteSettings> {
  return request('/api/v1/ai/route-settings', {
    method: 'PUT',
    data: toRouteSettingsPayload(data),
  }).then(mapAIRouteSettings)
}

export function listConversations(
  params?: AIConversationListParams,
  signal?: AbortSignal,
): Promise<AIConversationListResponse> {
  return request<any>('/api/v1/ai/conversations', {
    params: {
      page: params?.page,
      page_size: params?.pageSize,
      keyword: params?.keyword,
      cluster_id: params?.clusterId,
      status: params?.status,
      assistant_mode: params?.assistantMode,
    },
    signal,
  }).then((result) => ({
    items: Array.isArray(result?.list) ? result.list.map(mapAIConversationItem) : [],
    total: Number(result?.total || 0),
    page: Number(result?.page || 1),
    pageSize: Number(result?.page_size || result?.pageSize || 20),
  }))
}

export function getConversation(id: number | string, signal?: AbortSignal): Promise<AIConversationDetail> {
  return request(`/api/v1/ai/conversations/${id}`, { signal }).then(mapAIConversationDetail)
}

/** 更新会话标题（后端可能未实现 PATCH，调用失败时由前端本地回退） */
export function updateConversation(id: number | string, data: { title?: string }): Promise<any> {
  return request(`/api/v1/ai/conversations/${id}`, {
    method: 'PATCH',
    data,
  })
}

export function deleteConversation(id: number | string): Promise<void> {
  return request(`/api/v1/ai/conversations/${id}`, { method: 'DELETE' })
}

export function createConversation(data: CreateAIConversationRequest): Promise<{ id: number }> {
  return request<any>(`/api/v1/clusters/${data.clusterId}/ai/conversations`, {
    method: 'POST',
    data: {
      title: data.title,
      assistant_mode: data.assistantMode,
      provider_id: data.providerId,
      model_id: data.modelId,
      opening_message: data.openingMessage,
    },
  }).then((result) => ({ id: Number(result?.id || 0) }))
}

export function sendAIChatMessage(clusterId: number, data: AIChatRequest): Promise<AIChatResponse> {
  if (data.files?.length) {
    return request(`/api/v1/clusters/${clusterId}/ai/chat`, {
      method: 'POST',
      timeout: AI_CHAT_REQUEST_TIMEOUT,
      data: buildAIChatFormData(data),
    }).then((result: any) => camelize<AIChatResponse>(result))
  }

  return request(`/api/v1/clusters/${clusterId}/ai/chat`, {
    method: 'POST',
    timeout: AI_CHAT_REQUEST_TIMEOUT,
    data: toAIChatPayload(data),
  }).then((result: any) => camelize<AIChatResponse>(result))
}

/** 确认执行变更提案 */
export function confirmActionProposal(clusterId: number, actionId: number, data: {
  confirmation_text: string
  confirm_risk: boolean
}): Promise<any> {
  return request(`/api/v1/clusters/${clusterId}/ai/actions/${actionId}/confirm`, {
    method: 'POST',
    data,
  })
}
