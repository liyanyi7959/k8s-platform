package model

import "time"

type ServerTerminalFavorite struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    uint64    `gorm:"column:user_id;not null;uniqueIndex:uk_terminal_favorites_user_server"`
	ServerID  uint64    `gorm:"column:server_id;not null;uniqueIndex:uk_terminal_favorites_user_server"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ServerTerminalFavorite) TableName() string { return "server_terminal_favorites" }

type ServerTerminalAudit struct {
	ID                 uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	SessionID          string     `gorm:"column:session_id;type:varchar(64);not null;uniqueIndex:uk_terminal_audits_session"`
	UserID             uint64     `gorm:"column:user_id;not null;index:idx_terminal_audits_user"`
	ServerID           uint64     `gorm:"column:server_id;not null;index:idx_terminal_audits_server"`
	ServerName         string     `gorm:"column:server_name;type:varchar(120);not null"`
	ServerIP           string     `gorm:"column:server_ip;type:varchar(64);not null"`
	Status             string     `gorm:"column:status;type:varchar(24);not null;index:idx_terminal_audits_status"`
	CloseReason        *string    `gorm:"column:close_reason;type:varchar(255)"`
	RiskLevel          string     `gorm:"column:risk_level;type:varchar(24);not null"`
	RiskCount          int        `gorm:"column:risk_count;not null"`
	LastCommand        *string    `gorm:"column:last_command;type:varchar(255)"`
	StartedAt          time.Time  `gorm:"column:started_at;index:idx_terminal_audits_started_at"`
	LastActiveAt       time.Time  `gorm:"column:last_active_at"`
	EndedAt            *time.Time `gorm:"column:ended_at"`
	IdleTimeoutSec     int        `gorm:"column:idle_timeout_sec;not null"`
	AbsoluteTimeoutSec int        `gorm:"column:absolute_timeout_sec;not null"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (ServerTerminalAudit) TableName() string { return "server_terminal_audits" }
