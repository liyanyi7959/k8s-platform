package model

import "time"

type AIProvider struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name         string     `gorm:"column:name;type:varchar(120);not null;uniqueIndex:uk_ai_providers_name"`
	ProviderType string     `gorm:"column:provider_type;type:varchar(32);not null;default:openai;index:idx_ai_providers_type_enabled"`
	VendorCode   string     `gorm:"column:vendor_code;type:varchar(32);not null;default:''"`
	BaseURL      string     `gorm:"column:base_url;type:varchar(255);not null;default:''"`
	AuthScheme   string     `gorm:"column:auth_scheme;type:varchar(32);not null;default:bearer"`
	APIKeyEnc    *string    `gorm:"column:api_key_enc;type:longtext"`
	Enabled      bool       `gorm:"column:enabled;not null;default:1;index:idx_ai_providers_type_enabled"`
	Priority     int        `gorm:"column:priority;not null;default:100"`
	MetaJSON     JSONMap    `gorm:"column:meta_json;type:json"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt    *time.Time `gorm:"column:deleted_at;index:idx_ai_providers_deleted"`
}

func (AIProvider) TableName() string { return "ai_providers" }
