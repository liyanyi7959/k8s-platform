package model

import "time"

type User struct {
	// ID 为用户主键。
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	// Username 为登录名，全局唯一。
	Username string `gorm:"column:username;type:varchar(80);not null;uniqueIndex:uk_users_username"`
	// PasswordHash 存储 bcrypt hash，不存储明文密码。
	PasswordHash string `gorm:"column:password_hash;type:varchar(255);not null"`
	// Status 代表用户状态：active/disabled。
	Status    string    `gorm:"column:status;type:varchar(16);not null;default:active;index:idx_users_status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
	// DeletedAt 软删除字段，当前查询通常以 `deleted_at IS NULL` 过滤。
	DeletedAt *time.Time `gorm:"column:deleted_at;index"`
}

func (User) TableName() string { return "users" }

type Role struct {
	// Role 用于聚合权限点（Permission）。
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	// Name 为角色名称，全局唯一（例如 admin）。
	Name string `gorm:"column:name;type:varchar(80);not null;uniqueIndex:uk_roles_name"`
	// Desc 为可选描述。
	Desc      *string    `gorm:"column:desc;type:varchar(255)"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index"`
}

func (Role) TableName() string { return "roles" }

type Permission struct {
	// Permission 表示一个“权限点”，通常以 code 作为唯一标识（例如 sys:user_admin）。
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	// Code 为权限点编码，全局唯一。
	Code string `gorm:"column:code;type:varchar(120);not null;uniqueIndex:uk_permissions_code"`
	// Desc 为可选描述。
	Desc      *string    `gorm:"column:desc;type:varchar(255)"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index"`
}

func (Permission) TableName() string { return "permissions" }

type UserRole struct {
	// UserRole 为用户与角色的多对多关联表（联合主键）。
	UserID    uint64    `gorm:"column:user_id;primaryKey"`
	RoleID    uint64    `gorm:"column:role_id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (UserRole) TableName() string { return "user_roles" }

type RolePermission struct {
	// RolePermission 为角色与权限点的多对多关联表（联合主键）。
	RoleID       uint64    `gorm:"column:role_id;primaryKey"`
	PermissionID uint64    `gorm:"column:permission_id;primaryKey"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (RolePermission) TableName() string { return "role_permissions" }
