package domain

import "time"

type AIUsageRecord struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ConversationID *uint64   `gorm:"column:conversation_id"`
	MessageID      *uint64   `gorm:"column:message_id"`
	ProviderID     *uint64   `gorm:"column:provider_id;index:idx_ai_usage_records_provider_created"`
	ModelID        *uint64   `gorm:"column:model_id;index:idx_ai_usage_records_model_created"`
	UsageType      string    `gorm:"column:usage_type;type:varchar(32);not null;default:chat;index:idx_ai_usage_records_type_created"`
	RequestTokens  int       `gorm:"column:request_tokens;not null;default:0"`
	ResponseTokens int       `gorm:"column:response_tokens;not null;default:0"`
	TotalTokens    int       `gorm:"column:total_tokens;not null;default:0"`
	ImageCount     int       `gorm:"column:image_count;not null;default:0"`
	EstimatedCost  float64   `gorm:"column:estimated_cost;type:decimal(16,6);not null;default:0"`
	Currency       string    `gorm:"column:currency;type:varchar(16);not null;default:USD"`
	LatencyMS      int       `gorm:"column:latency_ms;not null;default:0"`
	MetaJSON       JSONMap   `gorm:"column:meta_json;type:json"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (AIUsageRecord) TableName() string { return "ai_usage_records" }
