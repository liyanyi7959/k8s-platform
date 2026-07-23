package model

import "time"

// MonitorAlertRule 保存平台维护的告警规则元数据；Prometheus 规则同步由后续适配器处理。
type MonitorAlertRule struct {
	ID        uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string          `gorm:"column:name;type:varchar(160);not null"`
	ClusterID uint64          `gorm:"column:cluster_id;not null;index:idx_monitor_alert_rules_cluster"`
	Severity  string          `gorm:"column:severity;type:varchar(16);not null;default:warning"`
	Condition string          `gorm:"column:condition_text;type:text"`
	Duration  string          `gorm:"column:duration;type:varchar(32);not null;default:'5m'"`
	Receivers JSONStringSlice `gorm:"column:receivers_json;type:json"`
	Enabled   bool            `gorm:"column:enabled;not null;default:1"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time       `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time      `gorm:"column:deleted_at;index"`
}

func (MonitorAlertRule) TableName() string { return "monitor_alert_rules" }

// MonitorIncident 是从 Alertmanager 或人工创建的可处置事件。
type MonitorIncident struct {
	ID               uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Fingerprint      string     `gorm:"column:fingerprint;type:varchar(160);not null;uniqueIndex:uk_monitor_incidents_fingerprint"`
	RuleID           *uint64    `gorm:"column:rule_id;index"`
	AlertName        string     `gorm:"column:alert_name;type:varchar(255);not null"`
	ClusterID        uint64     `gorm:"column:cluster_id;not null;index:idx_monitor_incidents_cluster_status"`
	Namespace        string     `gorm:"column:namespace;type:varchar(255);not null;default:''"`
	ResourceKind     string     `gorm:"column:resource_kind;type:varchar(64);not null;default:''"`
	ResourceName     string     `gorm:"column:resource_name;type:varchar(255);not null;default:''"`
	Severity         string     `gorm:"column:severity;type:varchar(16);not null;default:warning"`
	Status           string     `gorm:"column:status;type:varchar(32);not null;default:open;index:idx_monitor_incidents_cluster_status"`
	Summary          string     `gorm:"column:summary;type:varchar(512);not null;default:''"`
	Description      string     `gorm:"column:description;type:text"`
	LabelsJSON       JSONMap    `gorm:"column:labels_json;type:json"`
	AnnotationsJSON  JSONMap    `gorm:"column:annotations_json;type:json"`
	AIConversationID *uint64    `gorm:"column:ai_conversation_id;index"`
	AIProposalID     *uint64    `gorm:"column:ai_proposal_id;index"`
	AssigneeID       *uint64    `gorm:"column:assignee_id"`
	AssigneeName     string     `gorm:"column:assignee_name;type:varchar(80);not null;default:''"`
	StartedAt        time.Time  `gorm:"column:started_at;index"`
	AcknowledgedAt   *time.Time `gorm:"column:acknowledged_at"`
	ResolvedAt       *time.Time `gorm:"column:resolved_at"`
	VerifiedAt       *time.Time `gorm:"column:verified_at"`
	VerificationNote string     `gorm:"column:verification_note;type:text"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (MonitorIncident) TableName() string { return "monitor_incidents" }

type MonitorIncidentTimeline struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	IncidentID uint64    `gorm:"column:incident_id;not null;index:idx_monitor_incident_timeline"`
	Type       string    `gorm:"column:type;type:varchar(32);not null"`
	Title      string    `gorm:"column:title;type:varchar(255);not null"`
	Detail     string    `gorm:"column:detail;type:text"`
	MetaJSON   JSONMap   `gorm:"column:meta_json;type:json"`
	OperatorID uint64    `gorm:"column:operator_id;not null;default:0"`
	Operator   string    `gorm:"column:operator;type:varchar(80);not null;default:''"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (MonitorIncidentTimeline) TableName() string { return "monitor_incident_timelines" }
