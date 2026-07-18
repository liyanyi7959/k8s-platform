package model

import "time"

// SystemSetting 存储全局系统配置，key-value 形式。
type SystemSetting struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Key       string    `gorm:"column:key;type:varchar(64);not null;uniqueIndex:uk_system_settings_key"`
	Value     string    `gorm:"column:value;type:text;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SystemSetting) TableName() string { return "system_settings" }
