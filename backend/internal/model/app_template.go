package model

import "time"

// AppTemplate 应用模板（Helm Chart / YAML / GitOps 编排清单）。
type AppTemplate struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string     `gorm:"column:name;type:varchar(120);not null;index:idx_app_templates_name"`
	Category      string     `gorm:"column:category;type:varchar(64);not null;default:'其他'"`
	Version       string     `gorm:"column:version;type:varchar(32);not null;default:'1.0.0'"`
	Engine        string     `gorm:"column:engine;type:varchar(16);not null;default:helm;index:idx_app_templates_engine"`
	Status        string     `gorm:"column:status;type:varchar(32);not null;default:ready;index:idx_app_templates_status"`
	Summary       string     `gorm:"column:summary;type:varchar(512);not null;default:''"`
	Tags          *string    `gorm:"column:tags;type:json"`           // JSON array of string
	Manifest      *string    `gorm:"column:manifest;type:longtext"`   // YAML manifest text
	ValuesSchema  *string    `gorm:"column:values_schema;type:json"`  // JSON object
	DefaultValues *string    `gorm:"column:default_values;type:json"` // JSON object
	Source        string     `gorm:"column:source;type:varchar(128);not null;default:''"`
	SourceType    string     `gorm:"column:source_type;type:varchar(32);not null;default:custom;index:idx_app_templates_source_type"`
	SourceURL     string     `gorm:"column:source_url;type:varchar(512);not null;default:''"`
	SourceRef     *string    `gorm:"column:source_ref;type:json"`
	Owner         string     `gorm:"column:owner;type:varchar(64);not null;default:''"`
	ChartName     string     `gorm:"column:chart_name;type:varchar(120);not null;default:''"`
	AppVersion    string     `gorm:"column:app_version;type:varchar(64);not null;default:''"`
	Readme        *string    `gorm:"column:readme;type:longtext"`
	EnvExample    *string    `gorm:"column:env_example;type:longtext"`
	ProjectName   string     `gorm:"column:project_name_default;type:varchar(128);not null;default:''"`
	InstallDir    string     `gorm:"column:install_dir_default;type:varchar(256);not null;default:''"`
	ExtraFiles    *string    `gorm:"column:extra_files;type:json"`
	CreatedBy     uint64     `gorm:"column:created_by;not null;default:0"`
	UpdatedBy     uint64     `gorm:"column:updated_by;not null;default:0"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index:idx_app_templates_deleted"`
}

func (AppTemplate) TableName() string { return "app_templates" }
