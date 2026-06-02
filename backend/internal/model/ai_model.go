package model

import "time"

type AIModel struct {
	ID              uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ProviderID      uint64     `gorm:"column:provider_id;not null;uniqueIndex:uk_ai_models_provider_code"`
	Name            string     `gorm:"column:name;type:varchar(120);not null"`
	ModelCode       string     `gorm:"column:model_code;type:varchar(160);not null;uniqueIndex:uk_ai_models_provider_code"`
	ModelType       string     `gorm:"column:model_type;type:varchar(32);not null;default:chat;index:idx_ai_models_type_enabled"`
	Enabled         bool       `gorm:"column:enabled;not null;default:1;index:idx_ai_models_type_enabled"`
	SupportsTools   bool       `gorm:"column:supports_tools;not null;default:0"`
	SupportsVision  bool       `gorm:"column:supports_vision;not null;default:0"`
	SupportsStreaming bool     `gorm:"column:supports_streaming;not null;default:0"`
	SupportsReasoning bool     `gorm:"column:supports_reasoning;not null;default:0"`
	SupportsStructuredOutput bool `gorm:"column:supports_structured_output;not null;default:0"`
	SupportsImageGeneration bool `gorm:"column:supports_image_generation;not null;default:0"`
	MaxInputTokens  int        `gorm:"column:max_input_tokens;not null;default:0"`
	MaxOutputTokens int        `gorm:"column:max_output_tokens;not null;default:0"`
	ContextWindow   int        `gorm:"column:context_window;not null;default:0"`
	MetaJSON        JSONMap    `gorm:"column:meta_json;type:json"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt       *time.Time `gorm:"column:deleted_at;index:idx_ai_models_deleted"`
}

func (AIModel) TableName() string { return "ai_models" }
