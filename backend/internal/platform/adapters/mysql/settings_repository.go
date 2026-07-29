package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type SettingsRepository struct{ db *gorm.DB }

func NewSettingsRepository(db *gorm.DB) *SettingsRepository { return &SettingsRepository{db: db} }

func (r *SettingsRepository) Load(ctx context.Context) (map[string]string, error) {
	var rows []systemSettingRow
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	return values, nil
}

func (r *SettingsRepository) Save(ctx context.Context, values map[string]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			var row systemSettingRow
			err := tx.Where("`key` = ?", key).First(&row).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if row.ID == 0 {
				if err := tx.Create(&systemSettingRow{Key: key, Value: value}).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Model(&systemSettingRow{}).Where("id = ?", row.ID).Update("value", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type systemSettingRow struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Key       string    `gorm:"column:key;type:varchar(64);not null;uniqueIndex:uk_system_settings_key"`
	Value     string    `gorm:"column:value;type:text;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (systemSettingRow) TableName() string { return "system_settings" }
