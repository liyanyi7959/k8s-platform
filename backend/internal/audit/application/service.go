package application

import (
	"context"
	"time"

	"k8s-platform-backend/internal/audit/domain"
	"k8s-platform-backend/internal/audit/ports"
)

type Service struct{ repository ports.Repository }

func NewService(repository ports.Repository) *Service { return &Service{repository: repository} }

func (s *Service) Record(ctx context.Context, entry domain.Entry) {
	if s == nil || s.repository == nil {
		return
	}
	_ = s.repository.Append(ctx, entry)
}

type EntryDTO struct {
	ID           uint64    `json:"id"`
	UserID       uint64    `json:"user_id"`
	Username     string    `json:"username"`
	Action       string    `json:"action"`
	Resource     string    `json:"resource"`
	ResourceName string    `json:"resource_name"`
	ClusterID    uint64    `json:"cluster_id"`
	Namespace    string    `json:"namespace"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"status_code"`
	Detail       string    `json:"detail,omitempty"`
	ClientIP     string    `json:"client_ip"`
	RequestID    string    `json:"request_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type Page struct {
	Total int64      `json:"total"`
	Items []EntryDTO `json:"items"`
}

func (s *Service) List(ctx context.Context, query ports.ListQuery) (Page, error) {
	query.Page, query.PageSize = normalizePage(query.Page, query.PageSize)
	entries, total, err := s.repository.List(ctx, query)
	if err != nil {
		return Page{}, err
	}
	items := make([]EntryDTO, 0, len(entries))
	for _, entry := range entries {
		items = append(items, EntryDTO{
			ID: entry.ID, UserID: entry.UserID, Username: entry.Username,
			Action: entry.Action, Resource: entry.Resource, ResourceName: entry.ResourceName,
			ClusterID: entry.ClusterID, Namespace: entry.Namespace, Path: entry.Path,
			StatusCode: entry.StatusCode, Detail: entry.Detail, ClientIP: entry.ClientIP,
			RequestID: entry.RequestID, CreatedAt: entry.CreatedAt,
		})
	}
	return Page{Total: total, Items: items}, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

// Prune 按保留策略清理过期审计记录，返回删除条数。策略无效时返回领域错误。
func (s *Service) Prune(ctx context.Context, policy domain.RetentionPolicy) (int64, error) {
	before, err := policy.CutoffBefore(time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return s.repository.DeleteBefore(ctx, before)
}
