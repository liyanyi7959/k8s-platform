package model

import "time"

type Server struct {
	// Server 为服务器资产表模型（对应 migrations/004_servers.sql 的 servers 表）。
	//
	// 约定：
	// - Name 全局唯一，用于前端展示与快速定位
	// - AuthType 表示认证方式：password | key
	// - PasswordEnc/PrivateKeyEnc 存储加密后的密文，不存储明文
	// - Tags 使用 JSON 数组字符串存储（例如 ["prod","cn"]），便于简单筛选
	// - DeletedAt 为软删除标记，查询时通常以 deleted_at IS NULL 过滤
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string     `gorm:"column:name;type:varchar(120);not null;uniqueIndex:uk_servers_name"`
	IP            string     `gorm:"column:ip;type:varchar(64);not null;index:idx_servers_ip"`
	Port          int        `gorm:"column:port;not null;default:22"`
	AuthType      string     `gorm:"column:auth_type;type:varchar(16);not null;default:password"`
	Username      string     `gorm:"column:username;type:varchar(80);not null"`
	PasswordEnc   *string    `gorm:"column:password_enc;type:longtext"`
	PrivateKeyEnc *string    `gorm:"column:private_key_enc;type:longtext"`
	Tags          *string    `gorm:"column:tags;type:longtext"`
	Status        string     `gorm:"column:status;type:varchar(16);not null;default:active;index:idx_servers_status"`
	LastCheckAt   *time.Time `gorm:"column:last_check_at;index:idx_servers_last_check_at"`
	CreatedBy     uint64     `gorm:"column:created_by;not null;default:0;index:idx_servers_created_by"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index:idx_servers_deleted"`
}

func (Server) TableName() string { return "servers" }
