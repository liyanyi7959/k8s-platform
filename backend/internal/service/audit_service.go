package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// AuditService 提供操作审计日志的写入和查询能力。
type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// AuditEntry 中间件传入的审计记录。
type AuditEntry struct {
	UserID       uint64
	Username     string
	Action       string
	Resource     string
	ResourceName string
	ClusterID    uint64
	Namespace    string
	Path         string
	StatusCode   int
	Detail       string
	ClientIP     string
	RequestID    string
}

// Record 写入一条审计日志（异步友好，不阻塞请求）。
func (s *AuditService) Record(ctx context.Context, entry AuditEntry) {
	if s == nil || s.db == nil {
		return
	}
	row := model.AuditLog{
		UserID:       entry.UserID,
		Username:     entry.Username,
		Action:       entry.Action,
		Resource:     entry.Resource,
		ResourceName: entry.ResourceName,
		ClusterID:    entry.ClusterID,
		Namespace:    entry.Namespace,
		Path:         entry.Path,
		StatusCode:   entry.StatusCode,
		Detail:       entry.Detail,
		ClientIP:     entry.ClientIP,
		RequestID:    entry.RequestID,
	}
	_ = s.db.WithContext(ctx).Create(&row).Error
}

// AuditListParams 查询参数。
type AuditListParams struct {
	Page      int
	PageSize  int
	Username  string
	Action    string
	Resource  string
	ClusterID uint64
	StartTime *time.Time
	EndTime   *time.Time
}

// AuditListResult 分页查询结果。
type AuditListResult struct {
	Total int64            `json:"total"`
	Items []model.AuditLog `json:"items"`
}

// List 分页查询审计日志。
func (s *AuditService) List(ctx context.Context, p AuditListParams) (*AuditListResult, error) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 || p.PageSize > 100 {
		p.PageSize = 20
	}

	q := s.db.WithContext(ctx).Model(&model.AuditLog{})
	if v := strings.TrimSpace(p.Username); v != "" {
		q = q.Where("username = ?", v)
	}
	if v := strings.TrimSpace(p.Action); v != "" {
		q = q.Where("action = ?", v)
	}
	if v := strings.TrimSpace(p.Resource); v != "" {
		q = q.Where("resource = ?", v)
	}
	if p.ClusterID > 0 {
		q = q.Where("cluster_id = ?", p.ClusterID)
	}
	if p.StartTime != nil {
		q = q.Where("created_at >= ?", *p.StartTime)
	}
	if p.EndTime != nil {
		q = q.Where("created_at <= ?", *p.EndTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []model.AuditLog
	offset := (p.Page - 1) * p.PageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(p.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}
	return &AuditListResult{Total: total, Items: items}, nil
}
