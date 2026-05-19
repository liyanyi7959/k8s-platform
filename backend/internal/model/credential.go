package model

import "time"

type Credential struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string     `gorm:"column:name;type:varchar(120);not null;uniqueIndex:uk_credentials_provider_name,priority:2"`
	Provider  string     `gorm:"column:provider;type:varchar(32);not null;uniqueIndex:uk_credentials_provider_name,priority:1;index:idx_credentials_provider"`
	AuthType  string     `gorm:"column:auth_type;type:varchar(32);not null;index:idx_credentials_auth_type"`
	Desc      *string    `gorm:"column:desc;type:varchar(255)"`
	Meta      JSONMap    `gorm:"column:meta;type:json"`
	DataEnc   string     `gorm:"column:data_enc;type:longtext;not null"`
	CreatedBy uint64     `gorm:"column:created_by;not null;default:0;index:idx_credentials_created_by"`
	UpdatedBy uint64     `gorm:"column:updated_by;not null;default:0"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index:idx_credentials_deleted"`
}

func (Credential) TableName() string { return "credentials" }
