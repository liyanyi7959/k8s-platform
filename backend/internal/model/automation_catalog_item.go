package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type JSONStringArray []string

func (j *JSONStringArray) Scan(value interface{}) error {
	if value == nil {
		*j = JSONStringArray{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONStringArray) Value() (driver.Value, error) {
	if j == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(j)
}

type AutomationCatalogItem struct {
	ID                 uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name               string          `gorm:"column:name;type:varchar(160);not null;index:idx_automation_catalog_items_name"`
	Category           string          `gorm:"column:category;type:varchar(64);not null;index:idx_automation_catalog_items_category"`
	Description        *string         `gorm:"column:description;type:varchar(255)"`
	RecommendedVersion string          `gorm:"column:recommended_version;type:varchar(32);not null;default:latest"`
	Tags               JSONStringArray `gorm:"column:tags;type:json"`
	Published          bool            `gorm:"column:published;not null;default:true;index:idx_automation_catalog_items_published"`
	Visibility         string          `gorm:"column:visibility;type:varchar(16);not null;default:public;index:idx_automation_catalog_items_visibility"`
	IconURL            *string         `gorm:"column:icon_url;type:text"`
	TemplateID         uint64          `gorm:"column:template_id;not null;index:idx_automation_catalog_items_template_id"`
	TemplateVersion    string          `gorm:"column:template_version;type:varchar(32);not null"`
	SortOrder          int             `gorm:"column:sort_order;not null;default:0"`
	CreatedBy          uint64          `gorm:"column:created_by;not null;default:0"`
	UpdatedBy          uint64          `gorm:"column:updated_by;not null;default:0"`
	CreatedAt          time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time       `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt          *time.Time      `gorm:"column:deleted_at;index:idx_automation_catalog_items_deleted"`
}

func (AutomationCatalogItem) TableName() string { return "automation_catalog_items" }
