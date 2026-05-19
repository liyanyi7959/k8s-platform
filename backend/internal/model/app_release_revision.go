package model

import "time"

// AppReleaseRevision 记录 Compose 发布每次可回放的部署快照。
type AppReleaseRevision struct {
	ID                 uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ReleaseID          uint64    `gorm:"column:release_id;not null;index:idx_app_release_revisions_release,priority:1"`
	Revision           int       `gorm:"column:revision;not null;index:idx_app_release_revisions_release,priority:2"`
	TemplateID         uint64    `gorm:"column:template_id;not null;default:0"`
	TemplateName       string    `gorm:"column:template_name;type:varchar(120);not null;default:''"`
	TemplateVersion    string    `gorm:"column:template_version;type:varchar(32);not null;default:''"`
	ComposeManifest    string    `gorm:"column:compose_manifest;type:longtext;not null"`
	EnvContent         *string   `gorm:"column:env_content;type:longtext"`
	EnvOverride        *string   `gorm:"column:env_override;type:longtext"`
	Values             *string   `gorm:"column:values;type:json"`
	ProjectName        string    `gorm:"column:project_name;type:varchar(128);not null;default:''"`
	InstallDir         string    `gorm:"column:install_dir;type:varchar(256);not null;default:''"`
	PullImages         bool      `gorm:"column:pull_images;not null;default:true"`
	AutoInstallDocker  bool      `gorm:"column:auto_install_docker;not null;default:true"`
	AutoInstallCompose bool      `gorm:"column:auto_install_compose;not null;default:true"`
	Operator           string    `gorm:"column:operator;type:varchar(64);not null;default:''"`
	CreatedBy          uint64    `gorm:"column:created_by;not null;default:0"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (AppReleaseRevision) TableName() string { return "app_release_revisions" }
