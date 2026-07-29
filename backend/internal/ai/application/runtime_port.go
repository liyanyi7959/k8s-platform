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

type RuntimeChatImage struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	DataURL     string `json:"data_url"`
	Size        int64  `json:"size"`
}

type RuntimeStreamChunk struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	Error    string `json:"error"`
	Progress string `json:"progress,omitempty"`
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
}
