import { http } from '@/shared/http/http'
import type { ApiResponse, PageResult } from '@/shared/types/api'

async function unwrap<T>(request: Promise<unknown>): Promise<T> {
  const resp = (await request) as ApiResponse<T>
  return resp.data
}

export interface AIProviderItem {
  id: number
  name: string
  provider_type: string
  base_url: string
  enabled: boolean
  priority: number
  has_api_key: boolean
  meta?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateAIProviderRequest {
  name: string
  provider_type: string
  base_url: string
  api_key?: string
  enabled?: boolean
  priority?: number
  meta?: Record<string, unknown>
}

export interface PatchAIProviderRequest extends Partial<CreateAIProviderRequest> {}

export interface AIModelItem {
  id: number
  provider_id: number
  provider_name: string
  name: string
  model_code: string
  model_type: string
  enabled: boolean
  supports_tools: boolean
  supports_vision: boolean
  max_input_tokens: number
  max_output_tokens: number
  meta?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateAIModelRequest {
  provider_id: number
  name: string
  model_code: string
  model_type: string
  enabled?: boolean
  supports_tools?: boolean
  supports_vision?: boolean
  max_input_tokens?: number
  max_output_tokens?: number
  meta?: Record<string, unknown>
}

export interface PatchAIModelRequest extends Partial<CreateAIModelRequest> {}

export interface AIConversationItem {
  id: number
  cluster_id: number
  provider_id?: number
  model_id?: number
  title: string
  status: string
  assistant_mode: string
  summary: string
  created_by: number
  created_by_name: string
  message_count: number
  last_message_at?: string
  created_at: string
  updated_at: string
}

export interface AIMessageItem {
  id: number
  conversation_id: number
  role: string
  message_type: string
  content: string
  structured?: Record<string, unknown>
  status: string
  tool_call_count: number
  token_input: number
  token_output: number
  created_by: number
  created_at: string
}

export interface AIToolCallItem {
  id: number
  message_id?: number
  tool_name: string
  tool_kind: string
  status: string
  risk_level: string
  confirm_level: string
  result_summary: string
  result?: Record<string, unknown>
  error_message?: string
  created_at: string
}

export interface AIToolCatalogItem {
  name: string
  category: string
  description: string
  required_permissions: string[]
  missing_permissions?: string[]
  risk_level: string
  confirm_level: string
  timeout_seconds: number
  redaction_policy?: string
  input_schema?: Record<string, unknown>
  output_schema?: Record<string, unknown>
  available: boolean
}

export interface AIActionExecutionItem {
  id: number
  proposal_id: number
  status: string
  execution_no: number
  operator_id: number
  operator_name: string
  command_snapshot: string
  result?: Record<string, unknown>
  error_message?: string
  started_at?: string
  finished_at?: string
  created_at: string
}

export interface AIActionProposalItem {
  id: number
  conversation_id: number
  message_id?: number
  tool_call_id?: number
  cluster_id: number
  action_type: string
  target_kind: string
  target_namespace: string
  target_name: string
  risk_level: string
  confirm_level: string
  status: string
  title: string
  summary: string
  change?: Record<string, unknown>
  created_by: number
  created_by_name: string
  approved_by?: number
  approved_by_name: string
  approved_at?: string
  second_approved_by?: number
  second_approved_name: string
  second_approved_at?: string
  required_confirmation_text: string
  latest_execution?: AIActionExecutionItem
  executions?: AIActionExecutionItem[]
  created_at: string
  updated_at: string
}

export interface AIConversationDetail extends AIConversationItem {
  messages: AIMessageItem[]
  tool_calls: AIToolCallItem[]
  action_proposals: AIActionProposalItem[]
}

export interface CreateAIConversationRequest {
  title?: string
  assistant_mode?: 'diagnose' | 'chat'
  provider_id?: number
  model_id?: number
  opening_message?: string
}

export interface AIChatImagePayload {
  name: string
  content_type: string
  data_url: string
  size?: number
}

export interface SendAIChatRequest {
  conversation_id?: number
  message: string
  assistant_mode?: 'diagnose' | 'chat'
  provider_id?: number
  model_id?: number
  prefer_model?: string
  namespace?: string
  resource_kind?: string
  resource_name?: string
  images?: AIChatImagePayload[]
}

export interface SendAIChatResponse {
  conversation_id: number
  user_message_id: number
  assistant_message_id: number
  assistant_message: string
  provider_name: string
  model_name: string
  model_code: string
  tool_calls: AIToolCallItem[]
  action_proposals: AIActionProposalItem[]
}

export interface CreateAIActionProposalRequest {
  conversation_id: number
  message_id?: number
  proposal_type: 'restart_workload' | 'scale_workload' | 'apply_manifest'
  target_resource: {
    kind: string
    namespace: string
    name: string
  }
  payload?: Record<string, unknown>
  reason?: string
}

export interface CreateAIActionProposalResponse {
  proposal_id: number
  status: string
  risk_level: string
  need_second_confirm: boolean
  required_confirmation_text: string
  preview: string
  diff: string
  proposal: AIActionProposalItem
}

export interface ConfirmAIActionProposalRequest {
  confirmation_text?: string
  confirm_risk?: boolean
  operator_comment?: string
}

export interface ConfirmAIActionProposalResponse {
  proposal_id: number
  execution_status: string
  result_summary: string
  proposal: AIActionProposalItem
  execution?: AIActionExecutionItem
}

export function getAIProviders() {
  return unwrap<AIProviderItem[]>(http.get('/api/v1/ai/providers'))
}

export function createAIProvider(data: CreateAIProviderRequest) {
  return unwrap<{ id: number }>(http.post('/api/v1/ai/providers', data))
}

export function patchAIProvider(id: number, data: PatchAIProviderRequest) {
  return unwrap<null>(http.patch(`/api/v1/ai/providers/${id}`, data))
}

export function getAIModels(params: { provider_id?: number; model_type?: string; enabled?: boolean } = {}) {
  return unwrap<AIModelItem[]>(http.get('/api/v1/ai/models', { params }))
}

export function getAITools() {
  return unwrap<AIToolCatalogItem[]>(http.get('/api/v1/ai/tools'))
}

export function createAIModel(data: CreateAIModelRequest) {
  return unwrap<{ id: number }>(http.post('/api/v1/ai/models', data))
}

export function patchAIModel(id: number, data: PatchAIModelRequest) {
  return unwrap<null>(http.patch(`/api/v1/ai/models/${id}`, data))
}

export function getAIConversations(params: {
  page?: number
  page_size?: number
  cluster_id?: number
  status?: string
  assistant_mode?: string
  keyword?: string
} = {}) {
  return unwrap<PageResult<AIConversationItem>>(http.get('/api/v1/ai/conversations', { params }))
}

export function getAIConversationDetail(id: number) {
  return unwrap<AIConversationDetail>(http.get(`/api/v1/ai/conversations/${id}`))
}

export function deleteAIConversation(id: number) {
  return unwrap<null>(http.delete(`/api/v1/ai/conversations/${id}`))
}

export function createAIConversation(clusterId: number, data: CreateAIConversationRequest) {
  return unwrap<{ id: number }>(http.post(`/api/v1/clusters/${clusterId}/ai/conversations`, data))
}

export function sendAIChat(clusterId: number, data: SendAIChatRequest, options: { signal?: AbortSignal } = {}) {
  return unwrap<SendAIChatResponse>(http.post(`/api/v1/clusters/${clusterId}/ai/chat`, data, {
    timeout: 120_000,
    signal: options.signal
  }))
}

export function createAIActionProposal(clusterId: number, data: CreateAIActionProposalRequest) {
  return unwrap<CreateAIActionProposalResponse>(http.post(`/api/v1/clusters/${clusterId}/ai/actions/propose`, data))
}

export function confirmAIActionProposal(clusterId: number, proposalId: number, data: ConfirmAIActionProposalRequest = {}) {
  return unwrap<ConfirmAIActionProposalResponse>(http.post(`/api/v1/clusters/${clusterId}/ai/actions/${proposalId}/confirm`, data))
}
