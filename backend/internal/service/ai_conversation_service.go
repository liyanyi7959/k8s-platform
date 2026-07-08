package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type ListAIConversationsRequest struct {
	Page          int
	PageSize      int
	ClusterID     uint64
	Status        string
	AssistantMode string
	Keyword       string
}

type AIConversationItem struct {
	ID            uint64  `json:"id"`
	ClusterID     uint64  `json:"cluster_id"`
	ProviderID    *uint64 `json:"provider_id,omitempty"`
	ModelID       *uint64 `json:"model_id,omitempty"`
	Title         string  `json:"title"`
	Status        string  `json:"status"`
	AssistantMode string  `json:"assistant_mode"`
	Summary       string  `json:"summary"`
	CreatedBy     uint64  `json:"created_by"`
	CreatedByName string  `json:"created_by_name"`
	MessageCount  int     `json:"message_count"`
	LastMessageAt *string `json:"last_message_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type AIMessageItem struct {
	ID             uint64        `json:"id"`
	ConversationID uint64        `json:"conversation_id"`
	Role           string        `json:"role"`
	MessageType    string        `json:"message_type"`
	Content        string        `json:"content"`
	Attachments    []AIMessageAttachmentItem `json:"attachments,omitempty"`
	Structured     model.JSONMap `json:"structured,omitempty"`
	Status         string        `json:"status"`
	ToolCallCount  int           `json:"tool_call_count"`
	TokenInput     int           `json:"token_input"`
	TokenOutput    int           `json:"token_output"`
	CreatedBy      uint64        `json:"created_by"`
	CreatedAt      string        `json:"created_at"`
}

type AIConversationDetail struct {
	AIConversationItem
	Messages        []AIMessageItem        `json:"messages"`
	ToolCalls       []AIToolCallItem       `json:"tool_calls"`
	ActionProposals []AIActionProposalItem `json:"action_proposals"`
}

type CreateAIConversationRequest struct {
	Title          string  `json:"title"`
	AssistantMode  string  `json:"assistant_mode"`
	ProviderID     *uint64 `json:"provider_id"`
	ModelID        *uint64 `json:"model_id"`
	OpeningMessage *string `json:"opening_message"`
}

type AIConversationService struct {
	db *gorm.DB
}

func NewAIConversationService(db *gorm.DB) *AIConversationService {
	return &AIConversationService{db: db}
}

func (s *AIConversationService) ListConversations(ctx context.Context, req ListAIConversationsRequest) (PageResult[AIConversationItem], error) {
	if s.db == nil {
		return PageResult[AIConversationItem]{}, errors.New("db is required")
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	q := s.db.WithContext(ctx).Model(&model.AIConversation{}).Where("deleted_at IS NULL")
	if req.ClusterID > 0 {
		q = q.Where("cluster_id = ?", req.ClusterID)
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		q = q.Where("status = ?", status)
	}
	if mode := normalizeAIAssistantMode(req.AssistantMode); mode != "" {
		q = q.Where("assistant_mode = ?", mode)
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		q = q.Where("title LIKE ? OR summary LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return PageResult[AIConversationItem]{}, err
	}

	var rows []model.AIConversation
	if err := q.Order("updated_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[AIConversationItem]{}, err
	}

	counts, err := s.conversationMessageCounts(ctx, rows)
	if err != nil {
		return PageResult[AIConversationItem]{}, err
	}

	list := make([]AIConversationItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, buildAIConversationItem(row, counts[row.ID]))
	}
	return PageResult[AIConversationItem]{
		List:     list,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AIConversationService) GetConversation(ctx context.Context, id uint64) (AIConversationDetail, error) {
	if s.db == nil {
		return AIConversationDetail{}, errors.New("db is required")
	}
	if id == 0 {
		return AIConversationDetail{}, ErrWithMessage(ErrInvalidParams, "会话 ID 无效")
	}

	var row model.AIConversation
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AIConversationDetail{}, ErrNotFound
		}
		return AIConversationDetail{}, err
	}

	var messages []model.AIMessage
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", id).
		Order("created_at ASC, id ASC").
		Find(&messages).Error; err != nil {
		return AIConversationDetail{}, err
	}

	attachmentsByMessage, err := listConversationAttachments(ctx, s.db, id)
	if err != nil {
		return AIConversationDetail{}, err
	}

	items := make([]AIMessageItem, 0, len(messages))
	for _, msg := range messages {
		items = append(items, AIMessageItem{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			Role:           msg.Role,
			MessageType:    msg.MessageType,
			Content:        msg.Content,
			Attachments:    attachmentsByMessage[msg.ID],
			Structured:     msg.StructuredJSON,
			Status:         msg.Status,
			ToolCallCount:  msg.ToolCallCount,
			TokenInput:     msg.TokenInput,
			TokenOutput:    msg.TokenOutput,
			CreatedBy:      msg.CreatedBy,
			CreatedAt:      msg.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	var toolRows []model.AIToolCall
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", id).
		Order("created_at ASC, id ASC").
		Find(&toolRows).Error; err != nil {
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
		AIConversationItem: buildAIConversationItem(row, len(items)),
		Messages:           items,
		ToolCalls:          toolCalls,
		ActionProposals:    actionProposals,
	}, nil
}

func (s *AIConversationService) DeleteConversation(ctx context.Context, id uint64) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "会话 ID 无效")
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).
		Model(&model.AIConversation{}).
		Where("deleted_at IS NULL AND id = ?", id).
		Update("deleted_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *AIConversationService) CreateConversation(
	ctx context.Context,
	clusterID uint64,
	userID uint64,
	username string,
	req CreateAIConversationRequest,
) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	if clusterID == 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "集群 ID 无效")
	}

	title := strings.TrimSpace(req.Title)
	mode := normalizeAIAssistantMode(req.AssistantMode)
	if mode == "" {
		mode = "diagnose"
	}
	openingMessage := ""
	if req.OpeningMessage != nil {
		openingMessage = strings.TrimSpace(*req.OpeningMessage)
	}
	if title == "" {
		switch {
		case openingMessage != "":
			title = buildConversationTitle(openingMessage)
		case mode == "chat":
			title = "新的 AI 对话"
		default:
			title = "新的故障诊断会话"
		}
	}

	now := time.Now().UTC()
	conversation := model.AIConversation{
		ClusterID:     clusterID,
		ProviderID:    req.ProviderID,
		ModelID:       req.ModelID,
		Title:         title,
		Status:        "open",
		AssistantMode: mode,
		CreatedBy:     userID,
		CreatedByName: strings.TrimSpace(username),
		LastMessageAt: nil,
	}
	if openingMessage != "" {
		conversation.LastMessageAt = &now
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cluster model.Cluster
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", clusterID).First(&cluster).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWithMessage(ErrInvalidParams, "目标集群不存在")
			}
			return err
		}

		if conversation.ProviderID != nil {
			var provider model.AIProvider
			if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *conversation.ProviderID).First(&provider).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrWithMessage(ErrInvalidParams, "AI 提供商不存在")
				}
				return err
			}
		}
		if conversation.ModelID != nil {
			var aiModel model.AIModel
			if err := tx.Where("deleted_at IS NULL AND id = ?", *conversation.ModelID).First(&aiModel).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrWithMessage(ErrInvalidParams, "AI 模型不存在")
				}
				return err
			}
			if conversation.ProviderID != nil && aiModel.ProviderID != *conversation.ProviderID {
				return ErrWithMessage(ErrInvalidParams, "AI 模型与提供商不匹配")
			}
		}

		if err := tx.Create(&conversation).Error; err != nil {
			return err
		}
		if openingMessage != "" {
			message := model.AIMessage{
				ConversationID: conversation.ID,
				Role:           "user",
				MessageType:    "text",
				Content:        openingMessage,
				Status:         "created",
				CreatedBy:      userID,
			}
			if err := tx.Create(&message).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return conversation.ID, nil
}

func (s *AIConversationService) conversationMessageCounts(ctx context.Context, rows []model.AIConversation) (map[uint64]int, error) {
	counts := make(map[uint64]int, len(rows))
	if len(rows) == 0 {
		return counts, nil
	}

	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	type countRow struct {
		ConversationID uint64 `gorm:"column:conversation_id"`
		Count          int    `gorm:"column:cnt"`
	}

	var result []countRow
	if err := s.db.WithContext(ctx).
		Model(&model.AIMessage{}).
		Select("conversation_id, COUNT(*) AS cnt").
		Where("conversation_id IN ?", ids).
		Group("conversation_id").
		Scan(&result).Error; err != nil {
		return nil, err
	}
	for _, row := range result {
		counts[row.ConversationID] = row.Count
	}
	return counts, nil
}

func buildAIConversationItem(row model.AIConversation, messageCount int) AIConversationItem {
	var lastMessageAt *string
	if row.LastMessageAt != nil {
		v := row.LastMessageAt.UTC().Format(time.RFC3339)
		lastMessageAt = &v
	}
	return AIConversationItem{
		ID:            row.ID,
		ClusterID:     row.ClusterID,
		ProviderID:    row.ProviderID,
		ModelID:       row.ModelID,
		Title:         row.Title,
		Status:        row.Status,
		AssistantMode: row.AssistantMode,
		Summary:       row.Summary,
		CreatedBy:     row.CreatedBy,
		CreatedByName: row.CreatedByName,
		MessageCount:  messageCount,
		LastMessageAt: lastMessageAt,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func normalizeAIAssistantMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "diagnose":
		return "diagnose"
	case "chat":
		return "chat"
	default:
		return ""
	}
}

func buildConversationTitle(content string) string {
	title := strings.TrimSpace(content)
	runes := []rune(title)
	if len(runes) > 24 {
		return string(runes[:24]) + "..."
	}
	if title == "" {
		return "新的 AI 会话"
	}
	return title
}
