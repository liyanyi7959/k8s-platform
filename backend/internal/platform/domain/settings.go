package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrValidation = errors.New("system settings validation failed")

const (
	KeySiteName               = "site_name"
	KeySessionTimeout         = "session_timeout"
	KeyMaxLoginAttempts       = "max_login_attempts"
	KeyPasswordExpirationDays = "password_expiration_days"
	KeyEnableAuditLog         = "enable_audit_log"
	KeyEnableTwoFactorAuth    = "enable_two_factor_auth"
)

type Settings struct {
	SiteName               string
	SessionTimeout         int
	MaxLoginAttempts       int
	PasswordExpirationDays int
	EnableAuditLog         bool
	EnableTwoFactorAuth    bool
}

func Defaults() Settings {
	return Settings{SiteName: "AIOPS 智能运维平台", SessionTimeout: 3600, MaxLoginAttempts: 5, PasswordExpirationDays: 90, EnableAuditLog: true}
}

func FromValues(values map[string]string) Settings {
	settings := Defaults()
	if value := strings.TrimSpace(values[KeySiteName]); value != "" {
		settings.SiteName = value
	}
	parseInt(values[KeySessionTimeout], &settings.SessionTimeout)
	parseInt(values[KeyMaxLoginAttempts], &settings.MaxLoginAttempts)
	parseInt(values[KeyPasswordExpirationDays], &settings.PasswordExpirationDays)
	parseBool(values[KeyEnableAuditLog], &settings.EnableAuditLog)
	parseBool(values[KeyEnableTwoFactorAuth], &settings.EnableTwoFactorAuth)
	return settings
}

func (s Settings) Values() map[string]string {
	return map[string]string{
		KeySiteName: strings.TrimSpace(s.SiteName), KeySessionTimeout: strconv.Itoa(s.SessionTimeout),
		KeyMaxLoginAttempts: strconv.Itoa(s.MaxLoginAttempts), KeyPasswordExpirationDays: strconv.Itoa(s.PasswordExpirationDays),
		KeyEnableAuditLog: strconv.FormatBool(s.EnableAuditLog), KeyEnableTwoFactorAuth: strconv.FormatBool(s.EnableTwoFactorAuth),
	}
}

func (s *Settings) Validate() error {
	if s == nil {
		return fmt.Errorf("%w: settings are required", ErrValidation)
	}
	s.SiteName = strings.TrimSpace(s.SiteName)
	if s.SiteName == "" {
		return fmt.Errorf("%w: 站点名称不能为空", ErrValidation)
	}
	if s.SessionTimeout < 300 || s.SessionTimeout > 86400 {
		return fmt.Errorf("%w: 会话超时需在 300-86400 秒之间", ErrValidation)
	}
	if s.MaxLoginAttempts < 3 || s.MaxLoginAttempts > 10 {
		return fmt.Errorf("%w: 最大登录尝试次数需在 3-10 之间", ErrValidation)
	}
	if s.PasswordExpirationDays < 30 || s.PasswordExpirationDays > 365 {
		return fmt.Errorf("%w: 密码过期天数需在 30-365 之间", ErrValidation)
	}
	return nil
}

func parseInt(value string, target *int) {
	if parsed, err := strconv.Atoi(value); err == nil {
		*target = parsed
	}
}

func parseBool(value string, target *bool) {
	if parsed, err := strconv.ParseBool(value); err == nil {
		*target = parsed
	}
}
