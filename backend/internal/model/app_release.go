package model

import "time"

// AppRelease 应用发布实例（绑定模板的具体部署记录）。
type AppRelease struct {
	ID                 uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name               string     `gorm:"column:name;type:varchar(160);not null;index:idx_app_releases_name"`
	TemplateEngine     string     `gorm:"column:template_engine;type:varchar(16);not null;default:helm"`
	ClusterID          uint64     `gorm:"column:cluster_id;not null;index:idx_app_releases_cluster"`
	ClusterName        string     `gorm:"column:cluster_name;type:varchar(120);not null;default:''"`
	Namespace          string     `gorm:"column:namespace;type:varchar(64);not null;index:idx_app_releases_ns"`
	TargetType         string     `gorm:"column:target_type;type:varchar(16);not null;default:'';index:idx_app_releases_target_type"`
	TargetID           uint64     `gorm:"column:target_id;not null;default:0;index:idx_app_releases_target_id"`
	TargetName         string     `gorm:"column:target_name;type:varchar(120);not null;default:''"`
	ProjectName        string     `gorm:"column:project_name;type:varchar(128);not null;default:''"`
	InstallDir         string     `gorm:"column:install_dir;type:varchar(256);not null;default:''"`
	EnvOverride        *string    `gorm:"column:env_override;type:longtext"`
	PullImages         bool       `gorm:"column:pull_images;not null;default:true"`
	AutoInstallDocker  bool       `gorm:"column:auto_install_docker;not null;default:true"`
	AutoInstallCompose bool       `gorm:"column:auto_install_compose;not null;default:true"`
	LastTaskID         *uint64    `gorm:"column:last_task_id;index:idx_app_releases_last_task_id"`
	TemplateID         uint64     `gorm:"column:template_id;not null;index:idx_app_releases_template"`
	TemplateName       string     `gorm:"column:template_name;type:varchar(120);not null;default:''"`
	TemplateVersion    string     `gorm:"column:template_version;type:varchar(32);not null;default:''"`
	Status             string     `gorm:"column:status;type:varchar(16);not null;default:progressing;index:idx_app_releases_status"`
	CurrentRevision    int        `gorm:"column:current_revision;not null;default:1"`
	DesiredRevision    int        `gorm:"column:desired_revision;not null;default:1"`
	Strategy           string     `gorm:"column:strategy;type:varchar(16);not null;default:rolling"`
	Replicas           string     `gorm:"column:replicas;type:varchar(16);not null;default:'0/1'"`
	HealthScore        int        `gorm:"column:health_score;not null;default:70"`
	Values             *string    `gorm:"column:values;type:json"` // JSON object
	LastEvent          string     `gorm:"column:last_event;type:varchar(512);not null;default:''"`
	Operator           string     `gorm:"column:operator;type:varchar(64);not null;default:''"`
	Paused             bool       `gorm:"column:paused;not null;default:false"`
	CreatedBy          uint64     `gorm:"column:created_by;not null;default:0"`
	UpdatedBy          uint64     `gorm:"column:updated_by;not null;default:0"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt          *time.Time `gorm:"column:deleted_at;index:idx_app_releases_deleted"`
}

func (AppRelease) TableName() string { return "app_releases" }
