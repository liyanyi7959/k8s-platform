package model

import "time"

type AIRouteSetting struct {
	ID                            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	DefaultChatModelID            *uint64    `gorm:"column:default_chat_model_id"`
	DefaultDiagnoseModelID        *uint64    `gorm:"column:default_diagnose_model_id"`
	DefaultVisionModelID          *uint64    `gorm:"column:default_vision_model_id"`
	DefaultImageGenerationModelID *uint64    `gorm:"column:default_image_generation_model_id"`
	DefaultFallbackProviderID     *uint64    `gorm:"column:default_fallback_provider_id"`
	RoutingStrategy               string     `gorm:"column:routing_strategy;type:varchar(32);not null;default:priority_first"`
	AllowFallback                 bool       `gorm:"column:allow_fallback;not null;default:1"`
	MetaJSON                      JSONMap    `gorm:"column:meta_json;type:json"`
	CreatedAt                     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIRouteSetting) TableName() string { return "ai_route_settings" }
