package model

import "time"

// AppTemplate 应用商店模板。
// 用于预置一组可一键部署的 K8s 资源 YAML，支持 ${VAR} 变量替换。
type AppTemplate struct {
	// ID 为主键，自增。
	ID uint64 `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	// Name 为模板唯一标识（如 nginx-deploy），业务上要求唯一。
	Name string `json:"name" gorm:"column:name;type:varchar(100);not null;uniqueIndex:idx_app_templates_name"`
	// DisplayName 为展示名称（如 Nginx）。
	DisplayName string `json:"display_name" gorm:"column:display_name;type:varchar(200);default:''"`
	// Description 为模板描述（可空）。
	Description string `json:"description" gorm:"column:description;type:text"`
	// Category 为分类：database/middleware/monitoring/devtool/networking。
	Category string `json:"category" gorm:"column:category;type:varchar(50);index:idx_app_templates_category"`
	// Icon 为图标，支持 emoji 或 URL。
	Icon string `json:"icon" gorm:"column:icon;type:varchar(500);default:''"`
	// Template 为 YAML 模板，支持 ${NAMESPACE}、${NAME} 等变量。
	Template string `json:"template" gorm:"column:template;type:longtext"`
	// Variables 为 JSON 格式的变量定义数组。
	Variables string `json:"variables" gorm:"column:variables;type:text"`
	// IsBuiltin 标记是否为内置模板（内置模板不可删除）。
	IsBuiltin bool `json:"is_builtin" gorm:"column:is_builtin;default:false"`
	// DeployType 为部署类型：yaml 或 helm。helm 类型时 Template 字段存储 chart 仓库+chart名（如 bitnami/redis）。
	DeployType string `json:"deploy_type" gorm:"column:deploy_type;type:varchar(20);default:'yaml'"`
	// CreatedAt/UpdatedAt/DeletedAt 为通用审计字段。
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"column:deleted_at;index"`
}

// TableName 显式指定表名，避免 GORM 默认复数规则在不同版本中出现差异。
func (AppTemplate) TableName() string { return "app_templates" }
