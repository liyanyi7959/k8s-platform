package domain

import "time"

type AIConversation struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ClusterID     uint64     `gorm:"column:cluster_id;not null;default:0;index:idx_ai_conversations_cluster_updated"`
	ProviderID    *uint64    `gorm:"column:provider_id"`
	ModelID       *uint64    `gorm:"column:model_id"`
	Title         string     `gorm:"column:title;type:varchar(255);not null;default:''"`
	Status        string     `gorm:"column:status;type:varchar(32);not null;default:open;index:idx_ai_conversations_status_updated"`
	AssistantMode string     `gorm:"column:assistant_mode;type:varchar(32);not null;default:diagnose"`
	Summary       string     `gorm:"column:summary;type:varchar(512);not null;default:''"`
	CreatedBy     uint64     `gorm:"column:created_by;not null;default:0;index:idx_ai_conversations_creator_updated"`
	CreatedByName string     `gorm:"column:created_by_name;type:varchar(80);not null;default:''"`
	LastMessageAt *time.Time `gorm:"column:last_message_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index:idx_ai_conversations_deleted"`
}

func (AIConversation) TableName() string { return "ai_conversations" }
