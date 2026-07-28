import type { PageResult } from '@/shared/types/common'

export type AIAssistantMode = 'chat' | 'diagnose'

export interface AIProvider {
  id: number
  name: string
  providerType: string
  vendorCode?: string
  baseUrl: string
  authScheme?: string
  hasApiKey: boolean
  priority: number
  meta: Record<string, unknown>
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateAIProviderRequest {
  name: string
  providerType: string
  vendorCode?: string
  baseUrl: string
  authScheme?: string
  apiKey?: string
  priority?: number
  meta?: Record<string, unknown>
  enabled?: boolean
}

export interface AIModel {
  id: number
  providerId: number
  providerName: string
  name: string
  modelCode: string
  modelType: string
  maxInputTokens: number
  maxOutputTokens: number
  contextWindow: number
  supportsTools: boolean
  supportsVision: boolean
  supportsStreaming: boolean
  supportsReasoning: boolean
  supportsStructuredOutput: boolean
  supportsImageGeneration: boolean
  supportsFileInput: boolean
  enabled: boolean
  meta?: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface CreateAIModelRequest {
  providerId: number
  name: string
  modelCode: string
  modelType: string
  maxInputTokens?: number
  maxOutputTokens?: number
  contextWindow?: number
  supportsTools?: boolean
  supportsVision?: boolean
  supportsStreaming?: boolean
  supportsReasoning?: boolean
  supportsStructuredOutput?: boolean
  supportsImageGeneration?: boolean
  supportsFileInput?: boolean
  enabled?: boolean
  meta?: Record<string, unknown>
}

export interface AIRouteSettings {
  id: number
  defaultChatModelId?: number
  defaultDiagnoseModelId?: number
  defaultVisionModelId?: number
  defaultImageGenerationModelId?: number
  defaultFallbackProviderId?: number
  routingStrategy: string
  allowFallback: boolean
  meta: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface UpdateAIRouteSettingsRequest {
  defaultChatModelId?: number | null
  defaultDiagnoseModelId?: number | null
  defaultVisionModelId?: number | null
  defaultImageGenerationModelId?: number | null
  defaultFallbackProviderId?: number | null
  routingStrategy: string
  allowFallback?: boolean
  meta?: Record<string, unknown>
}

export interface AIMessageAttachment {
  id: number
  conversationId?: number
  messageId?: number
  originalName: string
  contentType: string
  fileSize: number
  purpose: string
  status: string
  fileKind: string
  downloadUrl: string
  createdAt: string
}

export interface AIMessage {
  id: number
  conversationId: number
  role: 'user' | 'assistant' | 'system' | 'tool'
  messageType: string
  content: string
  attachments?: AIMessageAttachment[]
  structured?: Record<string, unknown>
  status: string
  toolCallCount: number
  tokenInput: number
  tokenOutput: number
  createdBy: number
  createdAt: string
}

export interface AIToolCall {
  id: number
  messageId?: number
  toolName: string
  toolKind: string
  status: string
  riskLevel: string
  confirmLevel: string
  resultSummary: string
  result?: Record<string, unknown>
  errorMessage: string
  createdAt: string
}

export interface AIActionExecution {
  id: number
  proposalId: number
  status: string
  executionNo: number
  operatorId: number
  operatorName: string
  commandSnapshot: string
  result?: Record<string, unknown>
  errorMessage?: string
  startedAt?: string
  finishedAt?: string
  createdAt: string
}

export interface AIActionProposal {
  id: number
  conversationId: number
  messageId?: number
  toolCallId?: number
  clusterId: number
  actionType: string
  targetKind: string
  targetNamespace: string
  targetName: string
  riskLevel: string
  confirmLevel: string
  status: string
  title: string
  summary: string
  change?: Record<string, unknown>
  createdBy: number
  createdByName: string
  approvedBy?: number
  approvedByName: string
  approvedAt?: string
  secondApprovedBy?: number
  secondApprovedName: string
  secondApprovedAt?: string
  requiredConfirmationText: string
  latestExecution?: AIActionExecution | null
  executions?: AIActionExecution[]
  createdAt: string
  updatedAt: string
}

export interface AIConversationItem {
  id: number
  clusterId: number
  providerId?: number
  modelId?: number
  title: string
  status: string
  assistantMode: string
  summary: string
  createdBy: number
  createdByName: string
  messageCount: number
  lastMessageAt?: string
  createdAt: string
  updatedAt: string
}

export interface AIConversationDetail extends AIConversationItem {
  messages: AIMessage[]
  toolCalls: AIToolCall[]
  actionProposals: AIActionProposal[]
}

export interface AIConversationListParams {
  page?: number
  pageSize?: number
  keyword?: string
  clusterId?: number
  status?: string
  assistantMode?: string
}

export type AIConversationListResponse = PageResult<AIConversationItem>

export interface CreateAIConversationRequest {
  title?: string
  clusterId: number
  assistantMode?: AIAssistantMode
  providerId?: number
  modelId?: number
  openingMessage?: string
}

export interface AIChatImageInput {
  name: string
  contentType: string
  dataUrl: string
  size: number
}

export interface AIChatRequest {
  conversationId?: number
  message: string
  assistantMode?: AIAssistantMode
  providerId?: number
  modelId?: number
  preferModel?: string
  namespace?: string
  resourceKind?: string
  resourceName?: string
  images?: AIChatImageInput[]
  files?: File[]
}

export interface AIChatResponse {
  conversationId: number
  userMessageId: number
  assistantMessageId: number
  assistantMessage: string
  providerName: string
  modelName: string
  modelCode: string
  toolCalls: AIToolCall[]
  actionProposals: AIActionProposal[]
}

export interface AIStreamDoneData {
  conversationId: number
  userMessageId: number
  assistantMessageId: number
  providerName: string
  modelName: string
  modelCode: string
  toolCalls: AIToolCall[]
  actionProposals: AIActionProposal[]
}
