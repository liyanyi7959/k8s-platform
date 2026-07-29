package domain

import "time"

type AIMessage struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ConversationID uint64    `gorm:"column:conversation_id;not null;index:idx_ai_messages_conversation_created"`
	Role           string    `gorm:"column:role;type:varchar(32);not null;default:user;index:idx_ai_messages_role_created"`
	MessageType    string    `gorm:"column:message_type;type:varchar(32);not null;default:text"`
	Content        string    `gorm:"column:content;type:longtext;not null"`
	StructuredJSON JSONMap   `gorm:"column:structured_json;type:json"`
	Status         string    `gorm:"column:status;type:varchar(32);not null;default:created"`
	ToolCallCount  int       `gorm:"column:tool_call_count;not null;default:0"`
	TokenInput     int       `gorm:"column:token_input;not null;default:0"`
	TokenOutput    int       `gorm:"column:token_output;not null;default:0"`
	CreatedBy      uint64    `gorm:"column:created_by;not null;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIMessage) TableName() string { return "ai_messages" }
