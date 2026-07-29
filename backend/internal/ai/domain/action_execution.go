package domain

import "time"

type AIActionExecution struct {
	ID              uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ProposalID      uint64     `gorm:"column:proposal_id;not null;index:idx_ai_action_executions_status_created"`
	ConversationID  uint64     `gorm:"column:conversation_id;not null"`
	ClusterID       uint64     `gorm:"column:cluster_id;not null;default:0;index:idx_ai_action_executions_cluster_created"`
	ExecutionNo     int        `gorm:"column:execution_no;not null;default:1"`
	Status          string     `gorm:"column:status;type:varchar(32);not null;default:pending;index:idx_ai_action_executions_status_created"`
	StartedAt       *time.Time `gorm:"column:started_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at"`
	OperatorID      uint64     `gorm:"column:operator_id;not null;default:0"`
	OperatorName    string     `gorm:"column:operator_name;type:varchar(80);not null;default:''"`
	CommandSnapshot string     `gorm:"column:command_snapshot;type:text"`
	ResultJSON      JSONMap    `gorm:"column:result_json;type:json"`
	ErrorMessage    string     `gorm:"column:error_message;type:text"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIActionExecution) TableName() string { return "ai_action_executions" }
