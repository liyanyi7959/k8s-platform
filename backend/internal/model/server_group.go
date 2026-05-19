package model

import "time"

// ServerGroupRegion 表示服务器分组的“地域/机房”维度。
// 例如：北京机房、上海机房、AWS-cn-north-1 等。
type ServerGroupRegion struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string     `gorm:"column:name;type:varchar(120);not null;uniqueIndex:uk_server_group_regions_name"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index:idx_server_group_regions_deleted"`
}

// TableName 指定表名。
func (ServerGroupRegion) TableName() string { return "server_group_regions" }

// ServerGroupEnv 表示某个 Region 下的“环境”维度。
// 例如：prod / staging / test。
type ServerGroupEnv struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	RegionID  uint64     `gorm:"column:region_id;not null;index:idx_server_group_envs_region"`
	Name      string     `gorm:"column:name;type:varchar(120);not null"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index:idx_server_group_envs_deleted"`
}

// TableName 指定表名。
func (ServerGroupEnv) TableName() string { return "server_group_envs" }
