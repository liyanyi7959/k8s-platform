package model

import "time"

type AIToolCall struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ConversationID uint64    `gorm:"column:conversation_id;not null;index:idx_ai_tool_calls_status_created"`
	MessageID      *uint64   `gorm:"column:message_id;index:idx_ai_tool_calls_message"`
	ClusterID      uint64    `gorm:"column:cluster_id;not null;default:0;index:idx_ai_tool_calls_cluster_created"`
	ToolName       string    `gorm:"column:tool_name;type:varchar(120);not null"`
	ToolKind       string    `gorm:"column:tool_kind;type:varchar(32);not null;default:query"`
	CommandText    string    `gorm:"column:command_text;type:text"`
	ParamsJSON     JSONMap   `gorm:"column:params_json;type:json"`
	ExecutionMode  string    `gorm:"column:execution_mode;type:varchar(32);not null;default:proposal"`
	Status         string    `gorm:"column:status;type:varchar(32);not null;default:pending;index:idx_ai_tool_calls_status_created"`
	RiskLevel      string    `gorm:"column:risk_level;type:varchar(32);not null;default:low"`
	ConfirmLevel   string    `gorm:"column:confirm_level;type:varchar(32);not null;default:single"`
	ResultSummary  string    `gorm:"column:result_summary;type:varchar(512);not null;default:''"`
	ResultJSON     JSONMap   `gorm:"column:result_json;type:json"`
	ErrorMessage   string    `gorm:"column:error_message;type:text"`
	CreatedBy      uint64    `gorm:"column:created_by;not null;default:0"`
	CreatedByName  string    `gorm:"column:created_by_name;type:varchar(80);not null;default:''"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIToolCall) TableName() string { return "ai_tool_calls" }
