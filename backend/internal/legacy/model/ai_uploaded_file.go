package model

import "time"

type AIUploadedFile struct {
	ID             uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ConversationID *uint64    `gorm:"column:conversation_id"`
	MessageID      *uint64    `gorm:"column:message_id"`
	StoragePath    string     `gorm:"column:storage_path;type:varchar(255);not null"`
	OriginalName   string     `gorm:"column:original_name;type:varchar(255);not null;default:''"`
	ContentType    string     `gorm:"column:content_type;type:varchar(120);not null;default:''"`
	FileSize       int64      `gorm:"column:file_size;not null;default:0"`
	Purpose        string     `gorm:"column:purpose;type:varchar(32);not null;default:chat;index:idx_ai_uploaded_files_purpose_created"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;default:active"`
	SHA256         string     `gorm:"column:sha256;type:char(64);not null;default:'';index:idx_ai_uploaded_files_sha256"`
	CreatedBy      uint64     `gorm:"column:created_by;not null;default:0"`
	CreatedByName  string     `gorm:"column:created_by_name;type:varchar(80);not null;default:''"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index:idx_ai_uploaded_files_deleted"`
}

func (AIUploadedFile) TableName() string { return "ai_uploaded_files" }
