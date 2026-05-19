// model 定义数据库模型（GORM）。
//
// 约定：
// - 表名与字段名遵循 migrations 中的实际建表 SQL
// - 删除采用软删除：deleted_at 非空表示已删除
// - CreatedAt/UpdatedAt 使用 GORM 的 autoCreateTime/autoUpdateTime 自动维护
package model

import "time"

type Cluster struct {
	// ID 为主键，自增。
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	// Name 为集群名称，业务上要求唯一。
	Name string `gorm:"column:name;type:varchar(120);not null;uniqueIndex:uk_clusters_name"`
	// Type 表示集群来源：例如 imported（导入）/created（平台创建）等。
	Type string `gorm:"column:type;type:varchar(16);not null;default:imported;index:idx_clusters_type"`
	// Status 表示集群状态：例如 active/disabled 等（用于管理台展示与过滤）。
	Status string `gorm:"column:status;type:varchar(16);not null;default:active;index:idx_clusters_status"`
	// KubeconfigEnc 为加密后的 kubeconfig 内容（长文本）。
	// 说明：为了避免明文泄漏，平台对 kubeconfig 进行对称加密后落库。
	KubeconfigEnc *string `gorm:"column:kubeconfig_enc;type:longtext"`
	// K8sVersion 为 K8s 集群版本号（健康检查时更新）。
	K8sVersion string `gorm:"column:k8s_version;type:varchar(32);not null;default:''"`
	// NodeCount 为集群节点总数（健康检查时更新）。
	NodeCount int `gorm:"column:node_count;not null;default:0"`
	// LastHealthAt 为最近一次健康检查时间（可为空，表示尚未检查）。
	LastHealthAt *time.Time `gorm:"column:last_health_at"`
	// CreatedAt/UpdatedAt/DeletedAt 为通用审计字段。
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index"`
}

// TableName 显式指定表名，避免 GORM 默认复数规则在不同版本中出现差异。
func (Cluster) TableName() string { return "clusters" }
