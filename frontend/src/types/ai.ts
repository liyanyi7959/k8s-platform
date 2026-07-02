/**
 * AI 助手类型 — 扁平结构
 */
import type { PageResult } from './common'

/** AI 提供商 */
export interface AIProvider {
  id: number
  name: string
  providerType: string    // openai / azure-openai / ollama / qwen
  baseUrl: string
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
  baseUrl: string
  apiKey?: string
  priority?: number
  meta?: Record<string, unknown>
  enabled?: boolean
}

/** AI 模型 */
export interface AIModel {
  id: number
  providerId: number
  providerName: string
  name: string
  modelCode: string
  modelType: string       // chat / vision / embedding / image
  maxInputTokens: number
  supportsTools: boolean
  supportsVision: boolean
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateAIModelRequest {
  providerId: number
  name: string
  modelCode: string
  modelType: string
  maxInputTokens?: number
  supportsTools?: boolean
  supportsVision?: boolean
  enabled?: boolean
}

/** AI 对话消息 */
export interface AIMessage {
  id: string
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  status: 'sent' | 'pending' | 'failed' | 'cancelled'
  toolCallCount: number
  tokenInput: number
  tokenOutput: number
  createdAt: string
}

/** AI 工具调用 */
export interface AIToolCall {
  id: string
  toolName: string
  resultSummary: string
  errorMessage: string
  createdAt: string
}

/** AI 建议动作 */
export interface AISuggestedAction {
  actionType: string
  title: string
  reason: string
  targetKind: string
  targetNamespace: string
  targetName: string
  replicas: number
  riskLevel: 'low' | 'medium' | 'high' | 'critical'
  manifestYaml: string
  defaultNamespace: string
}

/** AI 操作提案 */
export interface AIActionProposal {
  id: number
  title: string
  actionType: string
  summary: string
  targetKind: string
  targetNamespace: string
  targetName: string
  riskLevel: string
  status: 'pending_confirm' | 'confirmed' | 'executing' | 'succeeded' | 'failed' | 'cancelled'
  confirmLevel: 'single' | 'double'
  manifestYaml: string
  latestExecution: {
    status: string
    createdAt: string
  } | null
  createdAt: string
}

/** AI 对话详情 */
export interface AIConversationDetail {
  id: string
  title: string
  clusterId: number
  assistantMode: string
  status: string
  createdByName: string
  createdBy: number
  messages: AIMessage[]
  toolCalls: AIToolCall[]
  suggestedActions: AISuggestedAction[]
  actionProposals: AIActionProposal[]
  createdAt: string
  updatedAt: string
}

/** AI 对话列表项 */
export interface AIConversationItem {
  id: string
  title: string
  messageCount: number
  updatedAt: string
  status: string
  assistantMode: string
}

export interface AIConversationListParams {
  page?: number
  pageSize?: number
  keyword?: string
  clusterId?: number
}

export type AIConversationListResponse = PageResult<AIConversationItem>
