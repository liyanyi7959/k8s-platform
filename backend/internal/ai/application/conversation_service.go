package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"k8s-platform-backend/internal/ai/domain"
	"k8s-platform-backend/internal/ai/ports"
)

type ConversationListRequest struct {
	Page, PageSize                 int
	ClusterID                      uint64
	Status, AssistantMode, Keyword string
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

type ConversationService struct{ repository ports.ConversationRepository }

func NewConversationService(repository ports.ConversationRepository) *ConversationService {
	return &ConversationService{repository: repository}
}
func (s *ConversationService) List(ctx context.Context, req ConversationListRequest) (PageResult[ConversationItem], error) {
	if s == nil || s.repository == nil {
		return PageResult[ConversationItem]{}, errors.New("conversation repository is required")
	}
	page, pageSize := normalizeConversationPage(req.Page, req.PageSize)
	status := strings.TrimSpace(req.Status)
	mode := NormalizeAssistantMode(req.AssistantMode)
	keyword := strings.TrimSpace(req.Keyword)
	rows, total, err := s.repository.ListConversations(ctx, ports.ConversationFilter{Page: page, PageSize: pageSize, ClusterID: req.ClusterID, Status: status, AssistantMode: mode, Keyword: keyword})
	if err != nil {
		return PageResult[ConversationItem]{}, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	counts, err := s.repository.CountMessages(ctx, ids)
	if err != nil {
		return PageResult[ConversationItem]{}, err
	}
	items := make([]ConversationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, BuildConversationItem(row, counts[row.ID]))
	}
	return PageResult[ConversationItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}
func (s *ConversationService) Create(ctx context.Context, clusterID, userID uint64, username string, req CreateConversationRequest) (uint64, error) {
	if s == nil || s.repository == nil {
		return 0, errors.New("conversation repository is required")
	}
	if clusterID == 0 {
		return 0, ErrorWithMessage(ErrInvalidParams, "集群 ID 无效")
	}
	mode := NormalizeAssistantMode(req.AssistantMode)
	if mode == "" {
		mode = "diagnose"
	}
	opening := ""
	if req.OpeningMessage != nil {
		opening = strings.TrimSpace(*req.OpeningMessage)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		switch {
		case opening != "":
			title = BuildConversationTitle(opening)
		case mode == "chat":
			title = "新的 AI 对话"
		default:
			title = "新的故障诊断会话"
		}
	}
	conversation := domain.AIConversation{ClusterID: clusterID, ProviderID: req.ProviderID, ModelID: req.ModelID, Title: title, Status: "open", AssistantMode: mode, CreatedBy: userID, CreatedByName: strings.TrimSpace(username)}
	if opening != "" {
		now := time.Now().UTC()
		conversation.LastMessageAt = &now
	}
	id, err := s.repository.CreateConversation(ctx, ports.ConversationCreate{Conversation: conversation, OpeningMessage: opening, UserID: userID})
	if err == nil {
		return id, nil
	}
	switch {
	case errors.Is(err, ports.ErrClusterNotFound):
		return 0, ErrorWithMessage(ErrInvalidParams, "目标集群不存在")
	case errors.Is(err, ports.ErrProviderNotFound):
		return 0, ErrorWithMessage(ErrInvalidParams, "AI 提供商不存在")
	case errors.Is(err, ports.ErrModelNotFound):
		return 0, ErrorWithMessage(ErrInvalidParams, "AI 模型不存在")
	case errors.Is(err, ports.ErrModelProviderMismatch):
		return 0, ErrorWithMessage(ErrInvalidParams, "AI 模型与提供商不匹配")
	default:
		return 0, err
	}
}
func (s *ConversationService) Delete(ctx context.Context, id uint64) error {
	if s == nil || s.repository == nil {
		return errors.New("conversation repository is required")
	}
	if id == 0 {
		return ErrorWithMessage(ErrInvalidParams, "会话 ID 无效")
	}
	err := s.repository.DeleteConversation(ctx, id, time.Now().UTC())
	if errors.Is(err, ports.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
func BuildConversationItem(row domain.AIConversation, messageCount int) ConversationItem {
	var lastMessageAt *string
	if row.LastMessageAt != nil {
		formatted := row.LastMessageAt.UTC().Format(time.RFC3339)
		lastMessageAt = &formatted
	}
	return ConversationItem{ID: row.ID, ClusterID: row.ClusterID, ProviderID: row.ProviderID, ModelID: row.ModelID, Title: row.Title, Status: row.Status, AssistantMode: row.AssistantMode, Summary: row.Summary, CreatedBy: row.CreatedBy, CreatedByName: row.CreatedByName, MessageCount: messageCount, LastMessageAt: lastMessageAt, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
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
