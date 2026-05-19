package model

import "time"

// InspectionTemplate 巡检模板（含检查项 JSON）。
type InspectionTemplate struct {
	ID          uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string          `gorm:"column:name;type:varchar(120);not null"`
	Description *string         `gorm:"column:description;type:varchar(512)"`
	Category    string          `gorm:"column:category;type:varchar(32);not null;default:baseline"`
	Version     string          `gorm:"column:version;type:varchar(32);not null;default:v1.0"`
	Tags        JSONStringArray `gorm:"column:tags;type:json"`
	Recommended bool            `gorm:"column:recommended;not null;default:false"`
	Checks      *string         `gorm:"column:checks;type:longtext"` // JSON array
	CreatedBy   uint64          `gorm:"column:created_by;not null;default:0"`
	UpdatedBy   uint64          `gorm:"column:updated_by;not null;default:0"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   *time.Time      `gorm:"column:deleted_at;index:idx_inspection_templates_deleted"`
}

func (InspectionTemplate) TableName() string { return "inspection_templates" }

// InspectionSchedule 定时巡检计划。
type InspectionSchedule struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string     `gorm:"column:name;type:varchar(160);not null"`
	TemplateID    uint64     `gorm:"column:template_id;not null;index:idx_inspection_schedules_template"`
	Cron          string     `gorm:"column:cron;type:varchar(64);not null"`
	Status        string     `gorm:"column:status;type:varchar(16);not null;default:enabled"`
	ServerIDs     *string    `gorm:"column:server_ids;type:json"` // JSON array of uint64
	TargetCount   int        `gorm:"column:target_count;not null;default:0"`
	LastRunStatus *string    `gorm:"column:last_run_status;type:varchar(16)"`
	LastRunAt     *time.Time `gorm:"column:last_run_at"`
	NextRunAt     *time.Time `gorm:"column:next_run_at"`
	CreatedBy     uint64     `gorm:"column:created_by;not null;default:0"`
	UpdatedBy     uint64     `gorm:"column:updated_by;not null;default:0"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index:idx_inspection_schedules_deleted"`
}

func (InspectionSchedule) TableName() string { return "inspection_schedules" }

// InspectionReport 巡检报告（含主机详情与问题列表 JSON）。
type InspectionReport struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ReportNo      string     `gorm:"column:report_no;type:varchar(64);not null"`
	TemplateID    uint64     `gorm:"column:template_id;not null;index:idx_inspection_reports_template"`
	TemplateName  string     `gorm:"column:template_name;type:varchar(120);not null;default:''"`
	ScopeLabel    string     `gorm:"column:scope_label;type:varchar(255);not null;default:''"`
	TargetCount   int        `gorm:"column:target_count;not null;default:0"`
	HealthScore   int        `gorm:"column:health_score;not null;default:100"`
	AbnormalCount int        `gorm:"column:abnormal_count;not null;default:0"`
	HighRiskCount int        `gorm:"column:high_risk_count;not null;default:0"`
	RiskLevel     string     `gorm:"column:risk_level;type:varchar(8);not null;default:p3;index:idx_inspection_reports_risk"`
	Status        string     `gorm:"column:status;type:varchar(16);not null;default:success"`
	TopIssues     *string    `gorm:"column:top_issues;type:json"` // JSON array of string
	TaskID        *uint64    `gorm:"column:task_id"`
	HostsJSON     *string    `gorm:"column:hosts_json;type:longtext"`  // JSON
	IssuesJSON    *string    `gorm:"column:issues_json;type:longtext"` // JSON
	GeneratedAt   *time.Time `gorm:"column:generated_at"`
	CreatedBy     string     `gorm:"column:created_by;type:varchar(64);not null;default:''"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_inspection_reports_created"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (InspectionReport) TableName() string { return "inspection_reports" }
