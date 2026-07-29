package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	aidomain "k8s-platform-backend/internal/ai/domain"
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

type AIRouteSettingsService struct {
	db *gorm.DB
}

func NewAIRouteSettingsService(db *gorm.DB) *AIRouteSettingsService {
	return &AIRouteSettingsService{db: db}
}

func (s *AIRouteSettingsService) Get(ctx context.Context) (AIRouteSettingsItem, error) {
	if s.db == nil {
		return AIRouteSettingsItem{}, errors.New("db is required")
	}
	row, err := s.ensureRow(ctx)
	if err != nil {
		return AIRouteSettingsItem{}, err
	}
	return buildAIRouteSettingsItem(row), nil
}

func (s *AIRouteSettingsService) Update(ctx context.Context, req UpdateAIRouteSettingsRequest) (AIRouteSettingsItem, error) {
	if s.db == nil {
		return AIRouteSettingsItem{}, errors.New("db is required")
	}

	row, err := s.ensureRow(ctx)
	if err != nil {
		return AIRouteSettingsItem{}, err
	}

	strategy := normalizeAIRoutingStrategy(req.RoutingStrategy)
	if strategy == "" {
		strategy = row.RoutingStrategy
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.validateModelRef(tx, req.DefaultChatModelID); err != nil {
			return err
		}
		if err := s.validateModelRef(tx, req.DefaultDiagnoseModelID); err != nil {
			return err
		}
		if err := s.validateModelRef(tx, req.DefaultVisionModelID); err != nil {
			return err
		}
		if err := s.validateModelRef(tx, req.DefaultImageGenerationModelID); err != nil {
			return err
		}
		if err := s.validateProviderRef(tx, req.DefaultFallbackProviderID); err != nil {
			return err
		}

		updates := map[string]any{
			"routing_strategy": strategy,
		}
		if req.AllowFallback != nil {
			updates["allow_fallback"] = *req.AllowFallback
		}
		if req.Meta != nil {
			updates["meta_json"] = *req.Meta
		}
		if req.DefaultChatModelID != nil {
			updates["default_chat_model_id"] = nullableUint64Value(req.DefaultChatModelID)
		}
		if req.DefaultDiagnoseModelID != nil {
			updates["default_diagnose_model_id"] = nullableUint64Value(req.DefaultDiagnoseModelID)
		}
		if req.DefaultVisionModelID != nil {
			updates["default_vision_model_id"] = nullableUint64Value(req.DefaultVisionModelID)
		}
		if req.DefaultImageGenerationModelID != nil {
			updates["default_image_generation_model_id"] = nullableUint64Value(req.DefaultImageGenerationModelID)
		}
		if req.DefaultFallbackProviderID != nil {
			updates["default_fallback_provider_id"] = nullableUint64Value(req.DefaultFallbackProviderID)
		}

		return tx.Model(&aidomain.AIRouteSetting{}).Where("id = ?", row.ID).Updates(updates).Error
	}); err != nil {
		return AIRouteSettingsItem{}, err
	}

	row, err = s.ensureRow(ctx)
	if err != nil {
		return AIRouteSettingsItem{}, err
	}
	return buildAIRouteSettingsItem(row), nil
}

func (s *AIRouteSettingsService) ensureRow(ctx context.Context) (aidomain.AIRouteSetting, error) {
	var row aidomain.AIRouteSetting
	if err := s.db.WithContext(ctx).First(&row, 1).Error; err == nil {
		return row, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return aidomain.AIRouteSetting{}, err
	}

	row = aidomain.AIRouteSetting{
		ID:              1,
		RoutingStrategy: "priority_first",
		AllowFallback:   true,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return aidomain.AIRouteSetting{}, err
	}
	return row, nil
}

func (s *AIRouteSettingsService) validateModelRef(tx *gorm.DB, value *uint64) error {
	if value == nil || *value == 0 {
		return nil
	}
	var row aidomain.AIModel
	if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *value).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWithMessage(ErrInvalidParams, "默认路由模型不存在")
		}
		return err
	}
	return nil
}

func (s *AIRouteSettingsService) validateProviderRef(tx *gorm.DB, value *uint64) error {
	if value == nil || *value == 0 {
		return nil
	}
	var row aidomain.AIProvider
	if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *value).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWithMessage(ErrInvalidParams, "默认兜底提供商不存在")
		}
		return err
	}
	return nil
}

func buildAIRouteSettingsItem(row aidomain.AIRouteSetting) AIRouteSettingsItem {
	return AIRouteSettingsItem{
		ID:                            row.ID,
		DefaultChatModelID:            row.DefaultChatModelID,
		DefaultDiagnoseModelID:        row.DefaultDiagnoseModelID,
		DefaultVisionModelID:          row.DefaultVisionModelID,
		DefaultImageGenerationModelID: row.DefaultImageGenerationModelID,
		DefaultFallbackProviderID:     row.DefaultFallbackProviderID,
		RoutingStrategy:               normalizeAIRoutingStrategy(row.RoutingStrategy),
		AllowFallback:                 row.AllowFallback,
		Meta:                          row.MetaJSON,
		CreatedAt:                     row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                     row.UpdatedAt.UTC().Format(time.RFC3339),
	}
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

func nullableUint64Value(value *uint64) any {
	if value == nil || *value == 0 {
		return nil
	}
	return *value
}
