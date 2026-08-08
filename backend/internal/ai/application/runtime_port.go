package application

import "context"

// RuntimePort isolates the AI HTTP interaction surface from the implementation
// that coordinates gateways, tools and change execution.
type RuntimePort interface {
	ListTools([]string) []any
	Conversation(context.Context, uint64) (any, error)
	SendMessage(context.Context, uint64, string, RuntimeChatRequest) (any, error)
	SendChatStream(context.Context, uint64, string, RuntimeChatRequest, func(RuntimeStreamChunk)) error
	CreateProposal(context.Context, uint64, uint64, string, CreateActionProposalRequest) (any, error)
	ConfirmProposal(context.Context, uint64, uint64, uint64, string, ConfirmActionProposalRequest) (any, error)
}

type RuntimeChatRequest struct {
	ClusterID      uint64              `json:"-"`
	ConversationID *uint64             `json:"conversation_id"`
	Message        string              `json:"message"`
	AssistantMode  string              `json:"assistant_mode"`
	ProviderID     *uint64             `json:"provider_id"`
	ModelID        *uint64             `json:"model_id"`
	PreferModel    string              `json:"prefer_model"`
	Namespace      string              `json:"namespace"`
	ResourceKind   string              `json:"resource_kind"`
	ResourceName   string              `json:"resource_name"`
	Images         []RuntimeChatImage  `json:"images"`
	Uploads        []AIChatUploadInput `json:"-"`
	UserPerms      []string            `json:"-"`
}

// RuntimeChatImage is retained as the runtime-port name while the shared
// input policy owns the transport-neutral image contract.
type RuntimeChatImage = ChatImageInput

// RuntimeChatResponse is the application-owned result contract for a
// completed chat turn. Runtime adapters may enrich the response internally,
// but HTTP callers no longer depend on legacy service DTOs.
type RuntimeChatResponse struct {
	ConversationID     uint64             `json:"conversation_id"`
	UserMessageID      uint64             `json:"user_message_id"`
	AssistantMessageID uint64             `json:"assistant_message_id"`
	AssistantMessage   string             `json:"assistant_message"`
	ProviderName       string             `json:"provider_name"`
	ModelName          string             `json:"model_name"`
	ModelCode          string             `json:"model_code"`
	ToolCalls          []ToolCallItem     `json:"tool_calls"`
	ActionProposals    []ActionProposalItem `json:"action_proposals"`
}

type RuntimeStreamChunk struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	Error    string `json:"error"`
	Progress string `json:"progress,omitempty"`
}

// RuntimeStreamDoneData is encoded in the terminal SSE event payload.
// Keeping it alongside RuntimeStreamChunk makes the streaming contract
// explicit at the application boundary instead of in a legacy coordinator.
type RuntimeStreamDoneData struct {
	ConversationID     uint64               `json:"conversation_id"`
	UserMessageID      uint64               `json:"user_message_id"`
	AssistantMessageID uint64               `json:"assistant_message_id"`
	ProviderName       string               `json:"provider_name"`
	ModelName          string               `json:"model_name"`
	ModelCode          string               `json:"model_code"`
	ToolCalls          []ToolCallItem       `json:"tool_calls"`
	ActionProposals    []ActionProposalItem `json:"action_proposals"`
}

type ActionTargetResource struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type CreateActionProposalRequest struct {
	ConversationID uint64               `json:"conversation_id"`
	MessageID      *uint64              `json:"message_id"`
	ProposalType   string               `json:"proposal_type"`
	TargetResource ActionTargetResource `json:"target_resource"`
	Payload        map[string]any       `json:"payload"`
	Reason         string               `json:"reason"`
}

type ConfirmActionProposalRequest struct {
	ConfirmationText string `json:"confirmation_text"`
	ConfirmRisk      bool   `json:"confirm_risk"`
	OperatorComment  string `json:"operator_comment"`
	// IdempotencyKey 可选：重复确认时携带相同值，服务端返回快照而不重复执行。
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}
