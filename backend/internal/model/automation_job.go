package model

import "time"

type AutomationJob struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string     `gorm:"column:name;type:varchar(160);not null;index:idx_automation_jobs_name"`
	Mode          string     `gorm:"column:mode;type:varchar(16);not null;default:manual;index:idx_automation_jobs_mode"`
	JobType       string     `gorm:"column:job_type;type:varchar(32);not null;default:inspection;index:idx_automation_jobs_type"`
	Env           string     `gorm:"column:env;type:varchar(16);not null;default:prod;index:idx_automation_jobs_env"`
	Status        string     `gorm:"column:status;type:varchar(16);not null;default:enabled;index:idx_automation_jobs_status"`
	RiskLevel     string     `gorm:"column:risk_level;type:varchar(16);not null;default:medium;index:idx_automation_jobs_risk"`
	ApprovalMode  string     `gorm:"column:approval_mode;type:varchar(16);not null;default:manual;index:idx_automation_jobs_approval"`
	Strategy      string     `gorm:"column:strategy;type:varchar(16);not null;default:batch"`
	Concurrency   int        `gorm:"column:concurrency;not null;default:5"`
	TimeoutSec    int        `gorm:"column:timeout_sec;not null;default:1800"`
	TemplateID    *uint64    `gorm:"column:template_id;index:idx_automation_jobs_template_id"`
	Cron          *string    `gorm:"column:cron;type:varchar(64)"`
	Targets       *string    `gorm:"column:targets;type:text"`
	LimitSpec     *string    `gorm:"column:limit_spec;type:varchar(255)"`
	Vars          JSONMap    `gorm:"column:vars;type:json"`
	ChangeWindow  *string    `gorm:"column:change_window;type:varchar(255)"`
	RollbackPlan  *string    `gorm:"column:rollback_plan;type:text"`
	LastRunTaskID *uint64    `gorm:"column:last_run_task_id"`
	LastRunStatus *string    `gorm:"column:last_run_status;type:varchar(32)"`
	LastRunAt     *time.Time `gorm:"column:last_run_at"`
	CreatedBy     uint64     `gorm:"column:created_by;not null;default:0"`
	UpdatedBy     uint64     `gorm:"column:updated_by;not null;default:0"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index:idx_automation_jobs_deleted"`
}

func (AutomationJob) TableName() string { return "automation_jobs" }
