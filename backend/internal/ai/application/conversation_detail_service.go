package application

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/ai/domain"
)

// ConversationDetailProjectionPort supplies read-only projections owned by
// other AI use cases. The port keeps the conversation query independent of
// tool execution and change-approval implementations.
type ConversationDetailProjectionPort interface {
	ListToolCallItems(context.Context, uint64) ([]any, error)
	ListActionProposalItems(context.Context, uint64) ([]any, error)
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
	db          *gorm.DB
	projections ConversationDetailProjectionPort
}

func NewConversationDetailService(db *gorm.DB, projections ConversationDetailProjectionPort) *ConversationDetailService {
	return &ConversationDetailService{db: db, projections: projections}
}

func (s *ConversationDetailService) Get(ctx context.Context, id uint64) (ConversationDetail, error) {
	if s == nil || s.db == nil || s.projections == nil {
		return ConversationDetail{}, errors.New("conversation detail service is required")
	}
	if id == 0 {
		return ConversationDetail{}, ErrorWithMessage(ErrInvalidParams, "会话 ID 无效")
	}

	var conversation domain.AIConversation
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ConversationDetail{}, ErrNotFound
		}
		return ConversationDetail{}, err
	}

	var messages []domain.AIMessage
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", id).Order("created_at ASC, id ASC").Find(&messages).Error; err != nil {
		return ConversationDetail{}, err
	}
	attachmentsByMessage, err := ListConversationAttachments(ctx, s.db, id)
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
