package model

import "time"

// AuditLog 操作审计日志。
type AuditLog struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID       uint64    `gorm:"column:user_id;not null" json:"user_id"`
	Username     string    `gorm:"column:username;type:varchar(80);not null" json:"username"`
	Action       string    `gorm:"column:action;type:varchar(16);not null" json:"action"`
	Resource     string    `gorm:"column:resource;type:varchar(64);not null" json:"resource"`
	ResourceName string    `gorm:"column:resource_name;type:varchar(255);not null" json:"resource_name"`
	ClusterID    uint64    `gorm:"column:cluster_id;not null" json:"cluster_id"`
	Namespace    string    `gorm:"column:namespace;type:varchar(255);not null" json:"namespace"`
	Path         string    `gorm:"column:path;type:varchar(512);not null" json:"path"`
	StatusCode   int       `gorm:"column:status_code;not null" json:"status_code"`
	Detail       string    `gorm:"column:detail;type:text" json:"detail,omitempty"`
	ClientIP     string    `gorm:"column:client_ip;type:varchar(45);not null" json:"client_ip"`
	RequestID    string    `gorm:"column:request_id;type:varchar(64);not null" json:"request_id"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
