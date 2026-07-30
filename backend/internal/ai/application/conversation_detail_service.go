package application

import (
	"context"
	"errors"
	"time"

	"k8s-platform-backend/internal/ai/domain"
	"k8s-platform-backend/internal/ai/ports"
)

// ConversationDetailProjectionPort supplies read-only projections owned by
// other AI use cases. The port keeps the conversation query independent of
// tool execution and change-approval implementations.
type ConversationDetailProjectionPort interface {
	ListToolCallItems(context.Context, uint64) ([]any, error)
	ListActionProposalItems(context.Context, uint64) ([]any, error)
}

type ConversationDetailRepository interface {
	ports.ConversationRepository
	ports.FileRepository
}

type ConversationMessageItem struct {
	ID             uint64                    `json:"id"`
	ConversationID uint64                    `json:"conversation_id"`
	Role           string                    `json:"role"`
	MessageType    string                    `json:"message_type"`
	Content        string                    `json:"content"`
	Attachments    []AIMessageAttachmentItem `json:"attachments,omitempty"`
	Structured     domain.JSONMap            `json:"structured,omitempty"`
	Status         string                    `json:"status"`
	ToolCallCount  int                       `json:"tool_call_count"`
	TokenInput     int                       `json:"token_input"`
	TokenOutput    int                       `json:"token_output"`
	CreatedBy      uint64                    `json:"created_by"`
	CreatedAt      string                    `json:"created_at"`
}

type ConversationDetail struct {
	ConversationItem
	Messages        []ConversationMessageItem `json:"messages"`
	ToolCalls       []any                     `json:"tool_calls"`
	ActionProposals []any                     `json:"action_proposals"`
}

type ConversationDetailService struct {
	repository  ConversationDetailRepository
	projections ConversationDetailProjectionPort
}

func NewConversationDetailService(repository ConversationDetailRepository, projections ConversationDetailProjectionPort) *ConversationDetailService {
	return &ConversationDetailService{repository: repository, projections: projections}
}

func (s *ConversationDetailService) Get(ctx context.Context, id uint64) (ConversationDetail, error) {
	if s == nil || s.repository == nil || s.projections == nil {
		return ConversationDetail{}, errors.New("conversation detail service is required")
	}
	if id == 0 {
		return ConversationDetail{}, ErrorWithMessage(ErrInvalidParams, "会话 ID 无效")
	}

	conversation, err := s.repository.FindConversation(ctx, id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ConversationDetail{}, ErrNotFound
		}
		return ConversationDetail{}, err
	}

	messages, err := s.repository.ListConversationMessages(ctx, id)
	if err != nil {
		return ConversationDetail{}, err
	}
	attachmentsByMessage, err := ListConversationAttachments(ctx, s.repository, id)
	if err != nil {
		return ConversationDetail{}, err
	}
	messageItems := make([]ConversationMessageItem, 0, len(messages))
	for _, message := range messages {
		messageItems = append(messageItems, ConversationMessageItem{
			ID: message.ID, ConversationID: message.ConversationID, Role: message.Role, MessageType: message.MessageType,
			Content: message.Content, Attachments: attachmentsByMessage[message.ID], Structured: message.StructuredJSON,
			Status: message.Status, ToolCallCount: message.ToolCallCount, TokenInput: message.TokenInput, TokenOutput: message.TokenOutput,
			CreatedBy: message.CreatedBy, CreatedAt: message.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	toolCalls, err := s.projections.ListToolCallItems(ctx, id)
	if err != nil {
		return ConversationDetail{}, err
	}
	actionProposals, err := s.projections.ListActionProposalItems(ctx, id)
	if err != nil {
		return ConversationDetail{}, err
	}
	return ConversationDetail{
		ConversationItem: BuildConversationItem(conversation, len(messageItems)),
		Messages:         messageItems, ToolCalls: toolCalls, ActionProposals: actionProposals,
	}, nil
}
