package model

import "time"

type AIActionProposal struct {
	ID                 uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ConversationID     uint64     `gorm:"column:conversation_id;not null;index:idx_ai_action_proposals_status_created"`
	MessageID          *uint64    `gorm:"column:message_id"`
	ToolCallID         *uint64    `gorm:"column:tool_call_id"`
	ClusterID          uint64     `gorm:"column:cluster_id;not null;default:0;index:idx_ai_action_proposals_cluster_created"`
	ActionType         string     `gorm:"column:action_type;type:varchar(64);not null"`
	TargetKind         string     `gorm:"column:target_kind;type:varchar(64);not null;default:''"`
	TargetNamespace    string     `gorm:"column:target_namespace;type:varchar(255);not null;default:''"`
	TargetName         string     `gorm:"column:target_name;type:varchar(255);not null;default:''"`
	RiskLevel          string     `gorm:"column:risk_level;type:varchar(32);not null;default:low;index:idx_ai_action_proposals_risk_status"`
	ConfirmLevel       string     `gorm:"column:confirm_level;type:varchar(32);not null;default:single"`
	Status             string     `gorm:"column:status;type:varchar(32);not null;default:draft;index:idx_ai_action_proposals_status_created;index:idx_ai_action_proposals_risk_status"`
	Title              string     `gorm:"column:title;type:varchar(255);not null;default:''"`
	Summary            string     `gorm:"column:summary;type:varchar(512);not null;default:''"`
	ChangeJSON         JSONMap    `gorm:"column:change_json;type:json"`
	CreatedBy          uint64     `gorm:"column:created_by;not null;default:0"`
	CreatedByName      string     `gorm:"column:created_by_name;type:varchar(80);not null;default:''"`
	ApprovedBy         *uint64    `gorm:"column:approved_by"`
	ApprovedByName     string     `gorm:"column:approved_by_name;type:varchar(80);not null;default:''"`
	ApprovedAt         *time.Time `gorm:"column:approved_at"`
	SecondApprovedBy   *uint64    `gorm:"column:second_approved_by"`
	SecondApprovedName string     `gorm:"column:second_approved_name;type:varchar(80);not null;default:''"`
	SecondApprovedAt   *time.Time `gorm:"column:second_approved_at"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AIActionProposal) TableName() string { return "ai_action_proposals" }
