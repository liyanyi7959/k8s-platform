package domain

import "time"

type DeployConfig struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	StepKey         string    `gorm:"column:step_key" json:"step_key"`
	OSType          string    `gorm:"column:os_type;default:ubuntu" json:"os_type"`
	StepName        string    `gorm:"column:step_name" json:"step_name"`
	StepOrder       int       `gorm:"column:step_order" json:"step_order"`
	CommandTemplate string    `gorm:"column:command_template;type:text" json:"command_template"`
	Description     *string   `gorm:"column:description;type:text" json:"description"`
	Enabled         bool      `gorm:"column:enabled;default:true" json:"enabled"`
	TimeoutSeconds  int       `gorm:"column:timeout_seconds;default:600" json:"timeout_seconds"`
	RetryCount      int       `gorm:"column:retry_count;default:0" json:"retry_count"`
	CreatedBy       uint64    `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (DeployConfig) TableName() string { return "deploy_configs" }

type DeployConfigVersion struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ConfigID        uint64    `gorm:"column:config_id" json:"config_id"`
	StepKey         string    `gorm:"column:step_key" json:"step_key"`
	CommandTemplate string    `gorm:"column:command_template;type:text" json:"command_template"`
	Description     *string   `gorm:"column:description;type:text" json:"description"`
	ChangeType      string    `gorm:"column:change_type" json:"change_type"`
	ChangedBy       uint64    `gorm:"column:changed_by" json:"changed_by"`
	ChangedAt       time.Time `gorm:"column:changed_at;autoCreateTime" json:"changed_at"`
	ChangeSummary   *string   `gorm:"column:change_summary" json:"change_summary"`
}

func (DeployConfigVersion) TableName() string { return "deploy_config_versions" }

type DeployRepository struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RepoType    string     `gorm:"column:repo_type" json:"repo_type"`
	Name        string     `gorm:"column:name" json:"name"`
	URL         string     `gorm:"column:url" json:"url"`
	Description *string    `gorm:"column:description;type:text" json:"description"`
	AuthType    string     `gorm:"column:auth_type;default:none" json:"auth_type"`
	AuthConfig  JSONMap    `gorm:"column:auth_config;type:json" json:"auth_config"`
	Priority    int        `gorm:"column:priority;default:100" json:"priority"`
	Enabled     bool       `gorm:"column:enabled;default:true" json:"enabled"`
	IsDefault   bool       `gorm:"column:is_default;default:false" json:"is_default"`
	MirrorOf    *string    `gorm:"column:mirror_of" json:"mirror_of"`
	CreatedBy   uint64     `gorm:"column:created_by" json:"created_by"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (DeployRepository) TableName() string { return "deploy_repositories" }
