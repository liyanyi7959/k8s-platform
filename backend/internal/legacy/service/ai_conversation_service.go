package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	aiapp "k8s-platform-backend/internal/ai/application"
	"k8s-platform-backend/internal/legacy/model"
)

type AIConversationItem = aiapp.ConversationItem

type AIMessageItem struct {
	ID             uint64                    `json:"id"`
	ConversationID uint64                    `json:"conversation_id"`
	Role           string                    `json:"role"`
	MessageType    string                    `json:"message_type"`
	Content        string                    `json:"content"`
	Attachments    []AIMessageAttachmentItem `json:"attachments,omitempty"`
	Structured     model.JSONMap             `json:"structured,omitempty"`
	Status         string                    `json:"status"`
	ToolCallCount  int                       `json:"tool_call_count"`
	TokenInput     int                       `json:"token_input"`
	TokenOutput    int                       `json:"token_output"`
	CreatedBy      uint64                    `json:"created_by"`
	CreatedAt      string                    `json:"created_at"`
}

type AIConversationDetail struct {
	AIConversationItem
	Messages        []AIMessageItem        `json:"messages"`
	ToolCalls       []AIToolCallItem       `json:"tool_calls"`
	ActionProposals []AIActionProposalItem `json:"action_proposals"`
}

// AIConversationDetailService composes legacy tool/action projections into a
// conversation detail response. Conversation lifecycle use cases live in AI
// application and are intentionally not duplicated here.
type AIConversationDetailService struct {
	db *gorm.DB
}

func NewAIConversationDetailService(db *gorm.DB) *AIConversationDetailService {
	return &AIConversationDetailService{db: db}
}

func (s *AIConversationDetailService) GetConversation(ctx context.Context, id uint64) (AIConversationDetail, error) {
	if s == nil || s.db == nil {
		return AIConversationDetail{}, errors.New("db is required")
	}
	if id == 0 {
		return AIConversationDetail{}, ErrWithMessage(ErrInvalidParams, "会话 ID 无效")
	}

	var conversation model.AIConversation
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AIConversationDetail{}, ErrNotFound
		}
		return AIConversationDetail{}, err
	}

	var messages []model.AIMessage
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", id).Order("created_at ASC, id ASC").Find(&messages).Error; err != nil {
		return AIConversationDetail{}, err
	}
	attachmentsByMessage, err := aiapp.ListConversationAttachments(ctx, s.db, id)
	if err != nil {
		return AIConversationDetail{}, err
	}
	messageItems := make([]AIMessageItem, 0, len(messages))
	for _, message := range messages {
		messageItems = append(messageItems, AIMessageItem{
			ID: message.ID, ConversationID: message.ConversationID, Role: message.Role, MessageType: message.MessageType,
			Content: message.Content, Attachments: attachmentsByMessage[message.ID], Structured: model.JSONMap(message.StructuredJSON),
			Status: message.Status, ToolCallCount: message.ToolCallCount, TokenInput: message.TokenInput, TokenOutput: message.TokenOutput,
			CreatedBy: message.CreatedBy, CreatedAt: message.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	var toolRows []model.AIToolCall
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", id).Order("created_at ASC, id ASC").Find(&toolRows).Error; err != nil {
		return AIConversationDetail{}, err
	}
	toolCalls := make([]AIToolCallItem, 0, len(toolRows))
	for _, row := range toolRows {
		toolCalls = append(toolCalls, buildAIToolCallItem(row))
	}
	actionProposals, err := listConversationActionProposals(ctx, s.db, id)
	if err != nil {
		return AIConversationDetail{}, err
	}

	return AIConversationDetail{
		AIConversationItem: aiapp.BuildConversationItem(conversation, len(messageItems)),
		Messages:           messageItems,
		ToolCalls:          toolCalls,
		ActionProposals:    actionProposals,
	}, nil
}

func normalizeAIAssistantMode(value string) string {
	return aiapp.NormalizeAssistantMode(value)
}

func buildConversationTitle(content string) string {
	return aiapp.BuildConversationTitle(content)
}
