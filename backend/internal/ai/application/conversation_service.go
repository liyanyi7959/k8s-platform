package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/ai/domain"
)

type ConversationListRequest struct {
	Page          int
	PageSize      int
	ClusterID     uint64
	Status        string
	AssistantMode string
	Keyword       string
}

type ConversationItem struct {
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

type CreateConversationRequest struct {
	Title          string  `json:"title"`
	AssistantMode  string  `json:"assistant_mode"`
	ProviderID     *uint64 `json:"provider_id"`
	ModelID        *uint64 `json:"model_id"`
	OpeningMessage *string `json:"opening_message"`
}

type PageResult[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type ConversationService struct {
	db *gorm.DB
}

func NewConversationService(db *gorm.DB) *ConversationService {
	return &ConversationService{db: db}
}

func (s *ConversationService) List(ctx context.Context, req ConversationListRequest) (PageResult[ConversationItem], error) {
	if s == nil || s.db == nil {
		return PageResult[ConversationItem]{}, errors.New("db is required")
	}
	page, pageSize := normalizeConversationPage(req.Page, req.PageSize)
	query := s.db.WithContext(ctx).Model(&domain.AIConversation{}).Where("deleted_at IS NULL")
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if mode := NormalizeAssistantMode(req.AssistantMode); mode != "" {
		query = query.Where("assistant_mode = ?", mode)
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		query = query.Where("title LIKE ? OR summary LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return PageResult[ConversationItem]{}, err
	}
	var rows []domain.AIConversation
	if err := query.Order("updated_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[ConversationItem]{}, err
	}
	counts, err := s.messageCounts(ctx, rows)
	if err != nil {
		return PageResult[ConversationItem]{}, err
	}
	items := make([]ConversationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, BuildConversationItem(row, counts[row.ID]))
	}
	return PageResult[ConversationItem]{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *ConversationService) Create(ctx context.Context, clusterID, userID uint64, username string, req CreateConversationRequest) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("db is required")
	}
	if clusterID == 0 {
		return 0, ErrorWithMessage(ErrInvalidParams, "集群 ID 无效")
	}

	mode := NormalizeAssistantMode(req.AssistantMode)
	if mode == "" {
		mode = "diagnose"
	}
	openingMessage := ""
	if req.OpeningMessage != nil {
		openingMessage = strings.TrimSpace(*req.OpeningMessage)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		switch {
		case openingMessage != "":
			title = BuildConversationTitle(openingMessage)
		case mode == "chat":
			title = "新的 AI 对话"
		default:
			title = "新的故障诊断会话"
		}
	}

	now := time.Now().UTC()
	conversation := domain.AIConversation{
		ClusterID:     clusterID,
		ProviderID:    req.ProviderID,
		ModelID:       req.ModelID,
		Title:         title,
		Status:        "open",
		AssistantMode: mode,
		CreatedBy:     userID,
		CreatedByName: strings.TrimSpace(username),
	}
	if openingMessage != "" {
		conversation.LastMessageAt = &now
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cluster clusterReference
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", clusterID).First(&cluster).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrorWithMessage(ErrInvalidParams, "目标集群不存在")
			}
			return err
		}
		if conversation.ProviderID != nil {
			var provider domain.AIProvider
			if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *conversation.ProviderID).First(&provider).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrorWithMessage(ErrInvalidParams, "AI 提供商不存在")
				}
				return err
			}
		}
		if conversation.ModelID != nil {
			var model domain.AIModel
			if err := tx.Where("deleted_at IS NULL AND id = ?", *conversation.ModelID).First(&model).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrorWithMessage(ErrInvalidParams, "AI 模型不存在")
				}
				return err
			}
			if conversation.ProviderID != nil && model.ProviderID != *conversation.ProviderID {
				return ErrorWithMessage(ErrInvalidParams, "AI 模型与提供商不匹配")
			}
		}
		if err := tx.Create(&conversation).Error; err != nil {
			return err
		}
		if openingMessage == "" {
			return nil
		}
		return tx.Create(&domain.AIMessage{
			ConversationID: conversation.ID,
			Role:           "user",
			MessageType:    "text",
			Content:        openingMessage,
			Status:         "created",
			CreatedBy:      userID,
		}).Error
	}); err != nil {
		return 0, err
	}
	return conversation.ID, nil
}

func (s *ConversationService) Delete(ctx context.Context, id uint64) error {
	if s == nil || s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrorWithMessage(ErrInvalidParams, "会话 ID 无效")
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&domain.AIConversation{}).Where("deleted_at IS NULL AND id = ?", id).Update("deleted_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ConversationService) messageCounts(ctx context.Context, rows []domain.AIConversation) (map[uint64]int, error) {
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
	if err := s.db.WithContext(ctx).Model(&domain.AIMessage{}).Select("conversation_id, COUNT(*) AS cnt").Where("conversation_id IN ?", ids).Group("conversation_id").Scan(&result).Error; err != nil {
		return nil, err
	}
	for _, row := range result {
		counts[row.ConversationID] = row.Count
	}
	return counts, nil
}

func BuildConversationItem(row domain.AIConversation, messageCount int) ConversationItem {
	var lastMessageAt *string
	if row.LastMessageAt != nil {
		formatted := row.LastMessageAt.UTC().Format(time.RFC3339)
		lastMessageAt = &formatted
	}
	return ConversationItem{
		ID: row.ID, ClusterID: row.ClusterID, ProviderID: row.ProviderID, ModelID: row.ModelID,
		Title: row.Title, Status: row.Status, AssistantMode: row.AssistantMode, Summary: row.Summary,
		CreatedBy: row.CreatedBy, CreatedByName: row.CreatedByName, MessageCount: messageCount,
		LastMessageAt: lastMessageAt, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func NormalizeAssistantMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "diagnose":
		return "diagnose"
	case "chat":
		return "chat"
	default:
		return ""
	}
}

func BuildConversationTitle(content string) string {
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

func normalizeConversationPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

type clusterReference struct{ ID uint64 }

func (clusterReference) TableName() string { return "clusters" }
