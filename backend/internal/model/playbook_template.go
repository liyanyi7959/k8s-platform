package model

import "time"

type PlaybookTemplate struct {
	ID             uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name           string     `gorm:"column:name;type:varchar(120);not null;uniqueIndex:uk_playbook_templates_name"`
	Scenario       string     `gorm:"column:scenario;type:varchar(32);not null;default:service_install;index:idx_playbook_templates_scenario"`
	Description    *string    `gorm:"column:description;type:varchar(255)"`
	CurrentVersion string     `gorm:"column:current_version;type:varchar(32);not null;default:''"`
	CreatedBy      uint64     `gorm:"column:created_by;not null;default:0"`
	UpdatedBy      uint64     `gorm:"column:updated_by;not null;default:0"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index:idx_playbook_templates_deleted"`
}

func (PlaybookTemplate) TableName() string { return "playbook_templates" }

type PlaybookTemplateVersion struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	TemplateID uint64    `gorm:"column:template_id;not null;uniqueIndex:uk_ptv_template_version,priority:1;index:idx_ptv_template_id"`
	Version    string    `gorm:"column:version;type:varchar(32);not null;uniqueIndex:uk_ptv_template_version,priority:2"`
	Source     JSONMap   `gorm:"column:source;type:json"`
	ParamsSchema JSONMap `gorm:"column:params_schema;type:json"`
	Defaults   JSONMap   `gorm:"column:defaults;type:json"`
	CreatedBy  uint64    `gorm:"column:created_by;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (PlaybookTemplateVersion) TableName() string { return "playbook_template_versions" }

