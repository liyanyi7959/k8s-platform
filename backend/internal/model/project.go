package model

import "time"

// Project 项目（多租户/命名空间分组）。
// 用于把一组命名空间归属到同一逻辑单元下，并附带 CPU/内存/Pod 配额信息。
type Project struct {
	// ID 为主键，自增。
	ID uint64 `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	// Name 为项目名称，业务上要求唯一。
	Name string `json:"name" gorm:"column:name;type:varchar(100);not null;uniqueIndex:idx_projects_name"`
	// Description 为项目描述（可空）。
	Description string `json:"description" gorm:"column:description;type:varchar(500);default:''"`
	// ClusterID 为关联的集群 ID（0 表示未绑定具体集群）。
	ClusterID uint64 `json:"cluster_id" gorm:"column:cluster_id;index:idx_projects_cluster_id"`
	// Namespaces 为逗号分隔的命名空间列表，例如 "default,kube-system"。
	Namespaces string `json:"namespaces" gorm:"column:namespaces;type:text"`
	// QuotaCPU 为 CPU 配额，例如 "10"。
	QuotaCPU string `json:"quota_cpu" gorm:"column:quota_cpu;type:varchar(20);default:''"`
	// QuotaMemory 为内存配额，例如 "16Gi"。
	QuotaMemory string `json:"quota_memory" gorm:"column:quota_memory;type:varchar(20);default:''"`
	// QuotaPods 为 Pod 配额，例如 "100"。
	QuotaPods string `json:"quota_pods" gorm:"column:quota_pods;type:varchar(20);default:''"`
	// CreatorID 为创建者用户 ID。
	CreatorID uint64 `json:"creator_id" gorm:"column:creator_id;index:idx_projects_creator_id"`
	// CreatedAt/UpdatedAt/DeletedAt 为通用审计字段。
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"column:deleted_at;index"`
}

// TableName 显式指定表名，避免 GORM 默认复数规则在不同版本中出现差异。
func (Project) TableName() string { return "projects" }
