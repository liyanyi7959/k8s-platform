package application

import (
	"context"

	"k8s-platform-backend/internal/platform/domain"
	"k8s-platform-backend/internal/platform/ports"
)

type Service struct{ repository ports.SettingsRepository }

func NewService(repository ports.SettingsRepository) *Service {
	return &Service{repository: repository}
}

type SettingsDTO struct {
	SiteName               string `json:"siteName"`
	SessionTimeout         int    `json:"sessionTimeout"`
	MaxLoginAttempts       int    `json:"maxLoginAttempts"`
	PasswordExpirationDays int    `json:"passwordExpirationDays"`
	EnableAuditLog         bool   `json:"enableAuditLog"`
	EnableTwoFactorAuth    bool   `json:"enableTwoFactorAuth"`
}

func (s *Service) Get(ctx context.Context) (SettingsDTO, error) {
	values, err := s.repository.Load(ctx)
	if err != nil {
		return SettingsDTO{}, err
	}
	return toDTO(domain.FromValues(values)), nil
}

func (s *Service) Update(ctx context.Context, request SettingsDTO) error {
	settings := domain.Settings{
		SiteName: request.SiteName, SessionTimeout: request.SessionTimeout,
		MaxLoginAttempts: request.MaxLoginAttempts, PasswordExpirationDays: request.PasswordExpirationDays,
		EnableAuditLog: request.EnableAuditLog, EnableTwoFactorAuth: request.EnableTwoFactorAuth,
	}
	if err := settings.Validate(); err != nil {
		return err
	}
	return s.repository.Save(ctx, settings.Values())
}

func toDTO(settings domain.Settings) SettingsDTO {
	return SettingsDTO{
		SiteName: settings.SiteName, SessionTimeout: settings.SessionTimeout,
		MaxLoginAttempts: settings.MaxLoginAttempts, PasswordExpirationDays: settings.PasswordExpirationDays,
		EnableAuditLog: settings.EnableAuditLog, EnableTwoFactorAuth: settings.EnableTwoFactorAuth,
	}
}
