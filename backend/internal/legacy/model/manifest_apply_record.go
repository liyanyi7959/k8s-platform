package model

import "time"

type ManifestApplyRecord struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ClusterID        uint64    `gorm:"column:cluster_id;not null;index" json:"cluster_id"`
	Status           string    `gorm:"column:status;type:varchar(32);not null;default:running;index" json:"status"`
	DryRun           bool      `gorm:"column:dry_run;not null;default:false;index" json:"dry_run"`
	DefaultNamespace string    `gorm:"column:default_namespace;type:varchar(255);not null;default:''" json:"default_namespace"`
	SourceLabel      string    `gorm:"column:source_label;type:varchar(128);not null;default:''" json:"source_label"`
	SourceResource   string    `gorm:"column:source_resource;type:varchar(64);not null;default:''" json:"source_resource"`
	WorkloadKind     string    `gorm:"column:workload_kind;type:varchar(32);not null;default:''" json:"workload_kind"`
	YAMLContent      string    `gorm:"column:yaml_content;type:longtext;not null" json:"yaml_content"`
	ResultJSON       string    `gorm:"column:result_json;type:longtext" json:"result_json"`
	ResultCount      int       `gorm:"column:result_count;not null;default:0" json:"result_count"`
	Summary          string    `gorm:"column:summary;type:varchar(512);not null;default:''" json:"summary"`
	ErrorMessage     string    `gorm:"column:error_message;type:text" json:"error_message"`
	CreatedBy        uint64    `gorm:"column:created_by;not null;default:0;index" json:"created_by"`
	CreatedByName    string    `gorm:"column:created_by_name;type:varchar(80);not null;default:''" json:"created_by_name"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ManifestApplyRecord) TableName() string { return "manifest_apply_records" }
