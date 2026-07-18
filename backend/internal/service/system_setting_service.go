package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// 系统设置 key 常量。
const (
	SettingKeySiteName               = "site_name"
	SettingKeySessionTimeout         = "session_timeout"
	SettingKeyMaxLoginAttempts       = "max_login_attempts"
	SettingKeyPasswordExpirationDays = "password_expiration_days"
	SettingKeyEnableAuditLog         = "enable_audit_log"
	SettingKeyEnableTwoFactorAuth    = "enable_two_factor_auth"
)

// SystemSettings 系统设置 DTO（与前端约定字段名）。
type SystemSettings struct {
	SiteName               string `json:"siteName"`
	SessionTimeout         int    `json:"sessionTimeout"`
	MaxLoginAttempts       int    `json:"maxLoginAttempts"`
	PasswordExpirationDays int    `json:"passwordExpirationDays"`
	EnableAuditLog         bool   `json:"enableAuditLog"`
	EnableTwoFactorAuth    bool   `json:"enableTwoFactorAuth"`
}

// SystemSettingsService 提供系统设置的读写能力。
type SystemSettingsService struct {
	db *gorm.DB
}

func NewSystemSettingsService(db *gorm.DB) *SystemSettingsService {
	return &SystemSettingsService{db: db}
}

// defaultSettings 返回默认设置，用于数据库中不存在对应 key 时兜底。
func defaultSettings() map[string]string {
	return map[string]string{
		SettingKeySiteName:               "AIOPS 智能运维平台",
		SettingKeySessionTimeout:         "3600",
		SettingKeyMaxLoginAttempts:       "5",
		SettingKeyPasswordExpirationDays: "90",
		SettingKeyEnableAuditLog:         "true",
		SettingKeyEnableTwoFactorAuth:    "false",
	}
}

// Get 读取系统设置。
func (s *SystemSettingsService) Get(ctx context.Context) (*SystemSettings, error) {
	rows := make([]model.SystemSetting, 0)
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	values := defaultSettings()
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	return parseSystemSettings(values), nil
}

// Update 更新系统设置。
func (s *SystemSettingsService) Update(ctx context.Context, settings *SystemSettings) error {
	if settings == nil {
		return ErrInvalidParams
	}
	if err := validateSystemSettings(settings); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]string{
			SettingKeySiteName:               strings.TrimSpace(settings.SiteName),
			SettingKeySessionTimeout:         strconv.Itoa(settings.SessionTimeout),
			SettingKeyMaxLoginAttempts:       strconv.Itoa(settings.MaxLoginAttempts),
			SettingKeyPasswordExpirationDays: strconv.Itoa(settings.PasswordExpirationDays),
			SettingKeyEnableAuditLog:         strconv.FormatBool(settings.EnableAuditLog),
			SettingKeyEnableTwoFactorAuth:    strconv.FormatBool(settings.EnableTwoFactorAuth),
		}
		for key, value := range updates {
			var existing model.SystemSetting
			err := tx.Where("`key` = ?", key).First(&existing).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if existing.ID > 0 {
				if err := tx.Model(&model.SystemSetting{}).Where("id = ?", existing.ID).Update("value", value).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Create(&model.SystemSetting{Key: key, Value: value}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func parseSystemSettings(values map[string]string) *SystemSettings {
	s := &SystemSettings{
		SiteName:               values[SettingKeySiteName],
		SessionTimeout:         3600,
		MaxLoginAttempts:       5,
		PasswordExpirationDays: 90,
		EnableAuditLog:         true,
		EnableTwoFactorAuth:    false,
	}
	if v, err := strconv.Atoi(values[SettingKeySessionTimeout]); err == nil {
		s.SessionTimeout = v
	}
	if v, err := strconv.Atoi(values[SettingKeyMaxLoginAttempts]); err == nil {
		s.MaxLoginAttempts = v
	}
	if v, err := strconv.Atoi(values[SettingKeyPasswordExpirationDays]); err == nil {
		s.PasswordExpirationDays = v
	}
	if v, err := strconv.ParseBool(values[SettingKeyEnableAuditLog]); err == nil {
		s.EnableAuditLog = v
	}
	if v, err := strconv.ParseBool(values[SettingKeyEnableTwoFactorAuth]); err == nil {
		s.EnableTwoFactorAuth = v
	}
	if s.SiteName == "" {
		s.SiteName = "AIOPS 智能运维平台"
	}
	return s
}

func validateSystemSettings(s *SystemSettings) error {
	if strings.TrimSpace(s.SiteName) == "" {
		return &ServiceError{Kind: ErrInvalidParams, Message: "站点名称不能为空"}
	}
	if s.SessionTimeout < 300 || s.SessionTimeout > 86400 {
		return &ServiceError{Kind: ErrInvalidParams, Message: fmt.Sprintf("会话超时需在 %d-%d 秒之间", 300, 86400)}
	}
	if s.MaxLoginAttempts < 3 || s.MaxLoginAttempts > 10 {
		return &ServiceError{Kind: ErrInvalidParams, Message: fmt.Sprintf("最大登录尝试次数需在 %d-%d 之间", 3, 10)}
	}
	if s.PasswordExpirationDays < 30 || s.PasswordExpirationDays > 365 {
		return &ServiceError{Kind: ErrInvalidParams, Message: fmt.Sprintf("密码过期天数需在 %d-%d 之间", 30, 365)}
	}
	return nil
}
