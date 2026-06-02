package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type AIProviderItem struct {
	ID           uint64        `json:"id"`
	Name         string        `json:"name"`
	ProviderType string        `json:"provider_type"`
	VendorCode   string        `json:"vendor_code"`
	BaseURL      string        `json:"base_url"`
	AuthScheme   string        `json:"auth_scheme"`
	Enabled      bool          `json:"enabled"`
	Priority     int           `json:"priority"`
	HasAPIKey    bool          `json:"has_api_key"`
	Meta         model.JSONMap `json:"meta,omitempty"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
}

type CreateAIProviderRequest struct {
	Name         string        `json:"name"`
	ProviderType string        `json:"provider_type"`
	VendorCode   string        `json:"vendor_code"`
	BaseURL      string        `json:"base_url"`
	AuthScheme   string        `json:"auth_scheme"`
	APIKey       string        `json:"api_key"`
	Enabled      *bool         `json:"enabled"`
	Priority     *int          `json:"priority"`
	Meta         model.JSONMap `json:"meta"`
}

type PatchAIProviderRequest struct {
	Name         *string        `json:"name"`
	ProviderType *string        `json:"provider_type"`
	VendorCode   *string        `json:"vendor_code"`
	BaseURL      *string        `json:"base_url"`
	AuthScheme   *string        `json:"auth_scheme"`
	APIKey       *string        `json:"api_key"`
	Enabled      *bool          `json:"enabled"`
	Priority     *int           `json:"priority"`
	Meta         *model.JSONMap `json:"meta"`
}

type AIModelItem struct {
	ID              uint64        `json:"id"`
	ProviderID      uint64        `json:"provider_id"`
	ProviderName    string        `json:"provider_name"`
	Name            string        `json:"name"`
	ModelCode       string        `json:"model_code"`
	ModelType       string        `json:"model_type"`
	Enabled         bool          `json:"enabled"`
	SupportsTools   bool          `json:"supports_tools"`
	SupportsVision  bool          `json:"supports_vision"`
	SupportsStreaming        bool          `json:"supports_streaming"`
	SupportsReasoning        bool          `json:"supports_reasoning"`
	SupportsStructuredOutput bool          `json:"supports_structured_output"`
	SupportsImageGeneration  bool          `json:"supports_image_generation"`
	MaxInputTokens  int           `json:"max_input_tokens"`
	MaxOutputTokens int           `json:"max_output_tokens"`
	ContextWindow   int           `json:"context_window"`
	Meta            model.JSONMap `json:"meta,omitempty"`
	CreatedAt       string        `json:"created_at"`
	UpdatedAt       string        `json:"updated_at"`
}

type ListAIModelsRequest struct {
	ProviderID uint64
	ModelType  string
	Enabled    *bool
}

type CreateAIModelRequest struct {
	ProviderID      uint64        `json:"provider_id"`
	Name            string        `json:"name"`
	ModelCode       string        `json:"model_code"`
	ModelType       string        `json:"model_type"`
	Enabled         *bool         `json:"enabled"`
	SupportsTools   *bool         `json:"supports_tools"`
	SupportsVision  *bool         `json:"supports_vision"`
	SupportsStreaming        *bool         `json:"supports_streaming"`
	SupportsReasoning        *bool         `json:"supports_reasoning"`
	SupportsStructuredOutput *bool         `json:"supports_structured_output"`
	SupportsImageGeneration  *bool         `json:"supports_image_generation"`
	MaxInputTokens  *int          `json:"max_input_tokens"`
	MaxOutputTokens *int          `json:"max_output_tokens"`
	ContextWindow   *int          `json:"context_window"`
	Meta            model.JSONMap `json:"meta"`
}

type PatchAIModelRequest struct {
	ProviderID      *uint64        `json:"provider_id"`
	Name            *string        `json:"name"`
	ModelCode       *string        `json:"model_code"`
	ModelType       *string        `json:"model_type"`
	Enabled         *bool          `json:"enabled"`
	SupportsTools   *bool          `json:"supports_tools"`
	SupportsVision  *bool          `json:"supports_vision"`
	SupportsStreaming        *bool          `json:"supports_streaming"`
	SupportsReasoning        *bool          `json:"supports_reasoning"`
	SupportsStructuredOutput *bool          `json:"supports_structured_output"`
	SupportsImageGeneration  *bool          `json:"supports_image_generation"`
	MaxInputTokens  *int           `json:"max_input_tokens"`
	MaxOutputTokens *int           `json:"max_output_tokens"`
	ContextWindow   *int           `json:"context_window"`
	Meta            *model.JSONMap `json:"meta"`
}

type AIProviderService struct {
	db            *gorm.DB
	encryptionKey string
}

func NewAIProviderService(db *gorm.DB, encryptionKey string) *AIProviderService {
	return &AIProviderService{db: db, encryptionKey: encryptionKey}
}

func (s *AIProviderService) ListProviders(ctx context.Context) ([]AIProviderItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}

	var rows []model.AIProvider
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("priority ASC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]AIProviderItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, AIProviderItem{
			ID:           row.ID,
			Name:         row.Name,
			ProviderType: row.ProviderType,
			VendorCode:   row.VendorCode,
			BaseURL:      row.BaseURL,
			AuthScheme:   row.AuthScheme,
			Enabled:      row.Enabled,
			Priority:     row.Priority,
			HasAPIKey:    row.APIKeyEnc != nil && strings.TrimSpace(*row.APIKeyEnc) != "",
			Meta:         row.MetaJSON,
			CreatedAt:    row.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:    row.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

func (s *AIProviderService) CreateProvider(ctx context.Context, req CreateAIProviderRequest) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}

	name := strings.TrimSpace(req.Name)
	providerType := normalizeAIProviderType(req.ProviderType)
	if name == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "AI 提供商名称不能为空")
	}
	if providerType == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "AI 提供商类型不能为空")
	}

	row := model.AIProvider{
		Name:         name,
		ProviderType: providerType,
		VendorCode:   normalizeAIVendorCode(req.VendorCode),
		BaseURL:      strings.TrimSpace(req.BaseURL),
		AuthScheme:   normalizeAIAuthScheme(req.AuthScheme),
		Enabled:      boolOrDefault(req.Enabled, true),
		Priority:     intOrDefault(req.Priority, 100),
		MetaJSON:     req.Meta,
	}
	if row.AuthScheme == "" {
		row.AuthScheme = "bearer"
	}

	if strings.TrimSpace(req.APIKey) != "" {
		enc, err := encryptText(s.encryptionKey, strings.TrimSpace(req.APIKey))
		if err != nil {
			return 0, err
		}
		row.APIKeyEnc = &enc
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.AIProvider
		if err := tx.Where("deleted_at IS NULL AND name = ?", row.Name).First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&row).Error
	}); err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *AIProviderService) PatchProvider(ctx context.Context, id uint64, req PatchAIProviderRequest) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "AI 提供商 ID 无效")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.AIProvider
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		updates := map[string]any{}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return ErrWithMessage(ErrInvalidParams, "AI 提供商名称不能为空")
			}
			var existing model.AIProvider
			if err := tx.Select("id").Where("deleted_at IS NULL AND name = ? AND id <> ?", name, id).First(&existing).Error; err == nil {
				return ErrConflict
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			updates["name"] = name
		}
		if req.ProviderType != nil {
			providerType := normalizeAIProviderType(*req.ProviderType)
			if providerType == "" {
				return ErrWithMessage(ErrInvalidParams, "AI 提供商类型不能为空")
			}
			updates["provider_type"] = providerType
		}
		if req.VendorCode != nil {
			updates["vendor_code"] = normalizeAIVendorCode(*req.VendorCode)
		}
		if req.BaseURL != nil {
			updates["base_url"] = strings.TrimSpace(*req.BaseURL)
		}
		if req.AuthScheme != nil {
			authScheme := normalizeAIAuthScheme(*req.AuthScheme)
			if authScheme == "" {
				return ErrWithMessage(ErrInvalidParams, "AI provider auth scheme is invalid")
			}
			updates["auth_scheme"] = authScheme
		}
		if req.APIKey != nil {
			apiKey := strings.TrimSpace(*req.APIKey)
			if apiKey == "" {
				updates["api_key_enc"] = nil
			} else {
				enc, err := encryptText(s.encryptionKey, apiKey)
				if err != nil {
					return err
				}
				updates["api_key_enc"] = &enc
			}
		}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if req.Priority != nil {
			updates["priority"] = *req.Priority
		}
		if req.Meta != nil {
			updates["meta_json"] = *req.Meta
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&model.AIProvider{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}

func (s *AIProviderService) ListModels(ctx context.Context, req ListAIModelsRequest) ([]AIModelItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}

	type aiModelRow struct {
		model.AIModel
		ProviderName string `gorm:"column:provider_name"`
	}

	q := s.db.WithContext(ctx).
		Table("ai_models AS m").
		Select("m.*, p.name AS provider_name").
		Joins("JOIN ai_providers AS p ON p.id = m.provider_id AND p.deleted_at IS NULL").
		Where("m.deleted_at IS NULL")

	if req.ProviderID > 0 {
		q = q.Where("m.provider_id = ?", req.ProviderID)
	}
	if v := normalizeAIModelType(req.ModelType); v != "" {
		q = q.Where("m.model_type = ?", v)
	}
	if req.Enabled != nil {
		q = q.Where("m.enabled = ?", *req.Enabled)
	}

	var rows []aiModelRow
	if err := q.Order("m.id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]AIModelItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, AIModelItem{
			ID:              row.ID,
			ProviderID:      row.ProviderID,
			ProviderName:    row.ProviderName,
			Name:            row.Name,
			ModelCode:       row.ModelCode,
			ModelType:       row.ModelType,
			Enabled:         row.Enabled,
			SupportsTools:   row.SupportsTools,
			SupportsVision:  row.SupportsVision,
			SupportsStreaming:        row.SupportsStreaming,
			SupportsReasoning:        row.SupportsReasoning,
			SupportsStructuredOutput: row.SupportsStructuredOutput,
			SupportsImageGeneration:  row.SupportsImageGeneration,
			MaxInputTokens:  row.MaxInputTokens,
			MaxOutputTokens: row.MaxOutputTokens,
			ContextWindow:   row.ContextWindow,
			Meta:            row.MetaJSON,
			CreatedAt:       row.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:       row.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

func (s *AIProviderService) CreateModel(ctx context.Context, req CreateAIModelRequest) (uint64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	if req.ProviderID == 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "模型提供商不能为空")
	}

	name := strings.TrimSpace(req.Name)
	modelCode := strings.TrimSpace(req.ModelCode)
	modelType := normalizeAIModelType(req.ModelType)
	if name == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "模型名称不能为空")
	}
	if modelCode == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "模型编码不能为空")
	}
	if modelType == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "模型类型不能为空")
	}

	row := model.AIModel{
		ProviderID:      req.ProviderID,
		Name:            name,
		ModelCode:       modelCode,
		ModelType:       modelType,
		Enabled:         boolOrDefault(req.Enabled, true),
		SupportsTools:   boolOrDefault(req.SupportsTools, false),
		SupportsVision:  boolOrDefault(req.SupportsVision, false),
		SupportsStreaming:        boolOrDefault(req.SupportsStreaming, false),
		SupportsReasoning:        boolOrDefault(req.SupportsReasoning, false),
		SupportsStructuredOutput: boolOrDefault(req.SupportsStructuredOutput, false),
		SupportsImageGeneration:  boolOrDefault(req.SupportsImageGeneration, false),
		MaxInputTokens:  intOrDefault(req.MaxInputTokens, 0),
		MaxOutputTokens: intOrDefault(req.MaxOutputTokens, 0),
		ContextWindow:   intOrDefault(req.ContextWindow, 0),
		MetaJSON:        req.Meta,
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var provider model.AIProvider
		if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", row.ProviderID).First(&provider).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrWithMessage(ErrInvalidParams, "模型提供商不存在")
			}
			return err
		}

		var existing model.AIModel
		if err := tx.Where("deleted_at IS NULL AND provider_id = ? AND model_code = ?", row.ProviderID, row.ModelCode).First(&existing).Error; err == nil {
			return ErrConflict
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&row).Error
	}); err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *AIProviderService) PatchModel(ctx context.Context, id uint64, req PatchAIModelRequest) error {
	if s.db == nil {
		return errors.New("db is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "模型 ID 无效")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.AIModel
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		nextProviderID := row.ProviderID
		if req.ProviderID != nil {
			if *req.ProviderID == 0 {
				return ErrWithMessage(ErrInvalidParams, "模型提供商不能为空")
			}
			var provider model.AIProvider
			if err := tx.Select("id").Where("deleted_at IS NULL AND id = ?", *req.ProviderID).First(&provider).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrWithMessage(ErrInvalidParams, "模型提供商不存在")
				}
				return err
			}
			nextProviderID = *req.ProviderID
		}

		nextModelCode := row.ModelCode
		if req.ModelCode != nil {
			nextModelCode = strings.TrimSpace(*req.ModelCode)
			if nextModelCode == "" {
				return ErrWithMessage(ErrInvalidParams, "模型编码不能为空")
			}
		}

		if nextProviderID != row.ProviderID || nextModelCode != row.ModelCode {
			var existing model.AIModel
			if err := tx.Select("id").Where(
				"deleted_at IS NULL AND provider_id = ? AND model_code = ? AND id <> ?",
				nextProviderID, nextModelCode, id,
			).First(&existing).Error; err == nil {
				return ErrConflict
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		updates := map[string]any{}
		if req.ProviderID != nil {
			updates["provider_id"] = *req.ProviderID
		}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return ErrWithMessage(ErrInvalidParams, "模型名称不能为空")
			}
			updates["name"] = name
		}
		if req.ModelCode != nil {
			updates["model_code"] = nextModelCode
		}
		if req.ModelType != nil {
			modelType := normalizeAIModelType(*req.ModelType)
			if modelType == "" {
				return ErrWithMessage(ErrInvalidParams, "模型类型不能为空")
			}
			updates["model_type"] = modelType
		}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if req.SupportsTools != nil {
			updates["supports_tools"] = *req.SupportsTools
		}
		if req.SupportsVision != nil {
			updates["supports_vision"] = *req.SupportsVision
		}
		if req.SupportsStreaming != nil {
			updates["supports_streaming"] = *req.SupportsStreaming
		}
		if req.SupportsReasoning != nil {
			updates["supports_reasoning"] = *req.SupportsReasoning
		}
		if req.SupportsStructuredOutput != nil {
			updates["supports_structured_output"] = *req.SupportsStructuredOutput
		}
		if req.SupportsImageGeneration != nil {
			updates["supports_image_generation"] = *req.SupportsImageGeneration
		}
		if req.MaxInputTokens != nil {
			updates["max_input_tokens"] = *req.MaxInputTokens
		}
		if req.MaxOutputTokens != nil {
			updates["max_output_tokens"] = *req.MaxOutputTokens
		}
		if req.ContextWindow != nil {
			updates["context_window"] = *req.ContextWindow
		}
		if req.Meta != nil {
			updates["meta_json"] = *req.Meta
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&model.AIModel{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
	})
}

func normalizeAIProviderType(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func normalizeAIVendorCode(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func normalizeAIAuthScheme(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "bearer":
		return "bearer"
	case "api-key", "apikey":
		return "api-key"
	case "none":
		return "none"
	default:
		return ""
	}
}

func normalizeAIModelType(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func boolOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func intOrDefault(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}
