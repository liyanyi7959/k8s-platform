package mysql

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/audit/domain"
	"k8s-platform-backend/internal/audit/ports"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Append(ctx context.Context, entry domain.Entry) error {
	return r.db.WithContext(ctx).Create(&auditLogRow{
		UserID: entry.UserID, Username: entry.Username, Action: entry.Action,
		Resource: entry.Resource, ResourceName: entry.ResourceName, ClusterID: entry.ClusterID,
		Namespace: entry.Namespace, Path: entry.Path, StatusCode: entry.StatusCode,
		Detail: entry.Detail, ClientIP: entry.ClientIP, RequestID: entry.RequestID,
	}).Error
}

func (r *Repository) List(ctx context.Context, query ports.ListQuery) ([]domain.Entry, int64, error) {
	dbQuery := r.db.WithContext(ctx).Model(&auditLogRow{})
	if value := strings.TrimSpace(query.Keyword); value != "" {
		like := "%" + value + "%"
		dbQuery = dbQuery.Where("username LIKE ? OR resource LIKE ? OR resource_name LIKE ? OR detail LIKE ? OR client_ip LIKE ?", like, like, like, like, like)
	}
	if value := strings.TrimSpace(query.Username); value != "" {
		dbQuery = dbQuery.Where("username = ?", value)
	}
	if value := strings.TrimSpace(query.Action); value != "" {
		dbQuery = dbQuery.Where("action = ?", value)
	}
	if value := strings.TrimSpace(query.Resource); value != "" {
		dbQuery = dbQuery.Where("resource = ?", value)
	}
	if query.ClusterID > 0 {
		dbQuery = dbQuery.Where("cluster_id = ?", query.ClusterID)
	}
	switch strings.ToLower(strings.TrimSpace(query.Status)) {
	case "success":
		dbQuery = dbQuery.Where("status_code >= ? AND status_code < ?", 200, 300)
	case "failure":
		dbQuery = dbQuery.Where("status_code >= ?", 400)
	}
	if query.StartTime != nil {
		dbQuery = dbQuery.Where("created_at >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		dbQuery = dbQuery.Where("created_at <= ?", *query.EndTime)
	}
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []auditLogRow
	if err := dbQuery.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	entries := make([]domain.Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, domain.Entry{
			ID: row.ID, UserID: row.UserID, Username: row.Username, Action: row.Action,
			Resource: row.Resource, ResourceName: row.ResourceName, ClusterID: row.ClusterID,
			Namespace: row.Namespace, Path: row.Path, StatusCode: row.StatusCode,
			Detail: row.Detail, ClientIP: row.ClientIP, RequestID: row.RequestID, CreatedAt: row.CreatedAt,
		})
	}
	return entries, total, nil
}

type auditLogRow struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       uint64    `gorm:"column:user_id;not null"`
	Username     string    `gorm:"column:username;type:varchar(80);not null"`
	Action       string    `gorm:"column:action;type:varchar(16);not null"`
	Resource     string    `gorm:"column:resource;type:varchar(64);not null"`
	ResourceName string    `gorm:"column:resource_name;type:varchar(255);not null"`
	ClusterID    uint64    `gorm:"column:cluster_id;not null"`
	Namespace    string    `gorm:"column:namespace;type:varchar(255);not null"`
	Path         string    `gorm:"column:path;type:varchar(512);not null"`
	StatusCode   int       `gorm:"column:status_code;not null"`
	Detail       string    `gorm:"column:detail;type:text"`
	ClientIP     string    `gorm:"column:client_ip;type:varchar(45);not null"`
	RequestID    string    `gorm:"column:request_id;type:varchar(64);not null"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (auditLogRow) TableName() string { return "audit_logs" }
