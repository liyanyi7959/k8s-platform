package application

import (
	"context"
	"errors"
	"strings"
	"time"

	aidomain "k8s-platform-backend/internal/ai/domain"
	"k8s-platform-backend/internal/ai/ports"
)

type AIRouteSettingsItem struct {
	ID                            uint64           `json:"id"`
	DefaultChatModelID            *uint64          `json:"default_chat_model_id,omitempty"`
	DefaultDiagnoseModelID        *uint64          `json:"default_diagnose_model_id,omitempty"`
	DefaultVisionModelID          *uint64          `json:"default_vision_model_id,omitempty"`
	DefaultImageGenerationModelID *uint64          `json:"default_image_generation_model_id,omitempty"`
	DefaultFallbackProviderID     *uint64          `json:"default_fallback_provider_id,omitempty"`
	RoutingStrategy               string           `json:"routing_strategy"`
	AllowFallback                 bool             `json:"allow_fallback"`
	Meta                          aidomain.JSONMap `json:"meta,omitempty"`
	CreatedAt                     string           `json:"created_at"`
	UpdatedAt                     string           `json:"updated_at"`
}
type UpdateAIRouteSettingsRequest struct {
	DefaultChatModelID            *uint64           `json:"default_chat_model_id"`
	DefaultDiagnoseModelID        *uint64           `json:"default_diagnose_model_id"`
	DefaultVisionModelID          *uint64           `json:"default_vision_model_id"`
	DefaultImageGenerationModelID *uint64           `json:"default_image_generation_model_id"`
	DefaultFallbackProviderID     *uint64           `json:"default_fallback_provider_id"`
	RoutingStrategy               string            `json:"routing_strategy"`
	AllowFallback                 *bool             `json:"allow_fallback"`
	Meta                          *aidomain.JSONMap `json:"meta"`
}
type AIRouteSettingsService struct{ repository ports.RouteSettingsRepository }

func NewAIRouteSettingsService(repository ports.RouteSettingsRepository) *AIRouteSettingsService {
	return &AIRouteSettingsService{repository: repository}
}
func (s *AIRouteSettingsService) Get(ctx context.Context) (AIRouteSettingsItem, error) {
	if s == nil || s.repository == nil {
		return AIRouteSettingsItem{}, errors.New("route settings repository is required")
	}
	row, err := s.repository.GetOrCreateRouteSettings(ctx)
	if err != nil {
		return AIRouteSettingsItem{}, err
	}
	return buildAIRouteSettingsItem(row), nil
}
func (s *AIRouteSettingsService) Update(ctx context.Context, req UpdateAIRouteSettingsRequest) (AIRouteSettingsItem, error) {
	if s == nil || s.repository == nil {
		return AIRouteSettingsItem{}, errors.New("route settings repository is required")
	}
	row, err := s.repository.GetOrCreateRouteSettings(ctx)
	if err != nil {
		return AIRouteSettingsItem{}, err
	}
	strategy := normalizeAIRoutingStrategy(req.RoutingStrategy)
	if strategy == "" {
		strategy = row.RoutingStrategy
	}
	err = s.repository.UpdateRouteSettings(ctx, row.ID, ports.RouteSettingsPatch{DefaultChatModelID: req.DefaultChatModelID, DefaultDiagnoseModelID: req.DefaultDiagnoseModelID, DefaultVisionModelID: req.DefaultVisionModelID, DefaultImageGenerationModelID: req.DefaultImageGenerationModelID, DefaultFallbackProviderID: req.DefaultFallbackProviderID, RoutingStrategy: strategy, AllowFallback: req.AllowFallback, Meta: req.Meta})
	if err != nil {
		if errors.Is(err, ports.ErrModelNotFound) {
			return AIRouteSettingsItem{}, ErrWithMessage(ErrInvalidParams, "默认路由模型不存在")
		}
		if errors.Is(err, ports.ErrProviderNotFound) {
			return AIRouteSettingsItem{}, ErrWithMessage(ErrInvalidParams, "默认兜底提供商不存在")
		}
		return AIRouteSettingsItem{}, err
	}
	row, err = s.repository.GetOrCreateRouteSettings(ctx)
	if err != nil {
		return AIRouteSettingsItem{}, err
	}
	return buildAIRouteSettingsItem(row), nil
}
func buildAIRouteSettingsItem(row aidomain.AIRouteSetting) AIRouteSettingsItem {
	return AIRouteSettingsItem{ID: row.ID, DefaultChatModelID: row.DefaultChatModelID, DefaultDiagnoseModelID: row.DefaultDiagnoseModelID, DefaultVisionModelID: row.DefaultVisionModelID, DefaultImageGenerationModelID: row.DefaultImageGenerationModelID, DefaultFallbackProviderID: row.DefaultFallbackProviderID, RoutingStrategy: normalizeAIRoutingStrategy(row.RoutingStrategy), AllowFallback: row.AllowFallback, Meta: row.MetaJSON, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
}
func normalizeAIRoutingStrategy(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "priority_first":
		return "priority_first"
	case "default_model_first":
		return "default_model_first"
	case "capability_first":
		return "capability_first"
	default:
		return ""
	}
}
