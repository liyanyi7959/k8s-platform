package application

import (
	"context"
	"errors"
	"strings"
	"time"

	aidomain "k8s-platform-backend/internal/ai/domain"
	"k8s-platform-backend/internal/ai/ports"
)

type AIProviderItem struct {
	ID           uint64           `json:"id"`
	Name         string           `json:"name"`
	ProviderType string           `json:"provider_type"`
	VendorCode   string           `json:"vendor_code"`
	BaseURL      string           `json:"base_url"`
	AuthScheme   string           `json:"auth_scheme"`
	Enabled      bool             `json:"enabled"`
	Priority     int              `json:"priority"`
	HasAPIKey    bool             `json:"has_api_key"`
	Meta         aidomain.JSONMap `json:"meta,omitempty"`
	CreatedAt    string           `json:"created_at"`
	UpdatedAt    string           `json:"updated_at"`
}
type CreateAIProviderRequest struct {
	Name         string           `json:"name"`
	ProviderType string           `json:"provider_type"`
	VendorCode   string           `json:"vendor_code"`
	BaseURL      string           `json:"base_url"`
	AuthScheme   string           `json:"auth_scheme"`
	APIKey       string           `json:"api_key"`
	Enabled      *bool            `json:"enabled"`
	Priority     *int             `json:"priority"`
	Meta         aidomain.JSONMap `json:"meta"`
}
type PatchAIProviderRequest struct {
	Name         *string           `json:"name"`
	ProviderType *string           `json:"provider_type"`
	VendorCode   *string           `json:"vendor_code"`
	BaseURL      *string           `json:"base_url"`
	AuthScheme   *string           `json:"auth_scheme"`
	APIKey       *string           `json:"api_key"`
	Enabled      *bool             `json:"enabled"`
	Priority     *int              `json:"priority"`
	Meta         *aidomain.JSONMap `json:"meta"`
}
type AIModelItem struct {
	ID                       uint64           `json:"id"`
	ProviderID               uint64           `json:"provider_id"`
	ProviderName             string           `json:"provider_name"`
	Name                     string           `json:"name"`
	ModelCode                string           `json:"model_code"`
	ModelType                string           `json:"model_type"`
	Enabled                  bool             `json:"enabled"`
	SupportsTools            bool             `json:"supports_tools"`
	SupportsVision           bool             `json:"supports_vision"`
	SupportsStreaming        bool             `json:"supports_streaming"`
	SupportsReasoning        bool             `json:"supports_reasoning"`
	SupportsStructuredOutput bool             `json:"supports_structured_output"`
	SupportsImageGeneration  bool             `json:"supports_image_generation"`
	SupportsFileInput        bool             `json:"supports_file_input"`
	MaxInputTokens           int              `json:"max_input_tokens"`
	MaxOutputTokens          int              `json:"max_output_tokens"`
	ContextWindow            int              `json:"context_window"`
	Meta                     aidomain.JSONMap `json:"meta,omitempty"`
	CreatedAt                string           `json:"created_at"`
	UpdatedAt                string           `json:"updated_at"`
}
type ListAIModelsRequest struct {
	ProviderID uint64
	ModelType  string
	Enabled    *bool
}
type CreateAIModelRequest struct {
	ProviderID               uint64           `json:"provider_id"`
	Name                     string           `json:"name"`
	ModelCode                string           `json:"model_code"`
	ModelType                string           `json:"model_type"`
	Enabled                  *bool            `json:"enabled"`
	SupportsTools            *bool            `json:"supports_tools"`
	SupportsVision           *bool            `json:"supports_vision"`
	SupportsStreaming        *bool            `json:"supports_streaming"`
	SupportsReasoning        *bool            `json:"supports_reasoning"`
	SupportsStructuredOutput *bool            `json:"supports_structured_output"`
	SupportsImageGeneration  *bool            `json:"supports_image_generation"`
	SupportsFileInput        *bool            `json:"supports_file_input"`
	MaxInputTokens           *int             `json:"max_input_tokens"`
	MaxOutputTokens          *int             `json:"max_output_tokens"`
	ContextWindow            *int             `json:"context_window"`
	Meta                     aidomain.JSONMap `json:"meta"`
}
type PatchAIModelRequest struct {
	ProviderID               *uint64           `json:"provider_id"`
	Name                     *string           `json:"name"`
	ModelCode                *string           `json:"model_code"`
	ModelType                *string           `json:"model_type"`
	Enabled                  *bool             `json:"enabled"`
	SupportsTools            *bool             `json:"supports_tools"`
	SupportsVision           *bool             `json:"supports_vision"`
	SupportsStreaming        *bool             `json:"supports_streaming"`
	SupportsReasoning        *bool             `json:"supports_reasoning"`
	SupportsStructuredOutput *bool             `json:"supports_structured_output"`
	SupportsImageGeneration  *bool             `json:"supports_image_generation"`
	SupportsFileInput        *bool             `json:"supports_file_input"`
	MaxInputTokens           *int              `json:"max_input_tokens"`
	MaxOutputTokens          *int              `json:"max_output_tokens"`
	ContextWindow            *int              `json:"context_window"`
	Meta                     *aidomain.JSONMap `json:"meta"`
}

type AIProviderService struct {
	repository    ports.ProviderRepository
	encryptionKey string
}

func NewAIProviderService(repository ports.ProviderRepository, encryptionKey string) *AIProviderService {
	return &AIProviderService{repository: repository, encryptionKey: encryptionKey}
}
func (s *AIProviderService) ListProviders(ctx context.Context) ([]AIProviderItem, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("provider repository is required")
	}
	rows, err := s.repository.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AIProviderItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, buildAIProviderItem(row))
	}
	return out, nil
}
func buildAIProviderItem(row aidomain.AIProvider) AIProviderItem {
	return AIProviderItem{ID: row.ID, Name: row.Name, ProviderType: row.ProviderType, VendorCode: row.VendorCode, BaseURL: row.BaseURL, AuthScheme: row.AuthScheme, Enabled: row.Enabled, Priority: row.Priority, HasAPIKey: row.APIKeyEnc != nil && strings.TrimSpace(*row.APIKeyEnc) != "", Meta: row.MetaJSON, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
}
func (s *AIProviderService) CreateProvider(ctx context.Context, req CreateAIProviderRequest) (uint64, error) {
	if s == nil || s.repository == nil {
		return 0, errors.New("provider repository is required")
	}
	name := strings.TrimSpace(req.Name)
	kind := normalizeAIProviderType(req.ProviderType)
	if name == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "AI 提供商名称不能为空")
	}
	if kind == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "AI 提供商类型不能为空")
	}
	row := aidomain.AIProvider{Name: name, ProviderType: kind, VendorCode: normalizeAIVendorCode(req.VendorCode), BaseURL: strings.TrimSpace(req.BaseURL), AuthScheme: normalizeAIAuthScheme(req.AuthScheme), Enabled: boolOrDefault(req.Enabled, true), Priority: intOrDefault(req.Priority, 100), MetaJSON: req.Meta}
	if row.AuthScheme == "" {
		row.AuthScheme = "bearer"
	}
	if key := strings.TrimSpace(req.APIKey); key != "" {
		encoded, err := encryptText(s.encryptionKey, key)
		if err != nil {
			return 0, err
		}
		row.APIKeyEnc = &encoded
	}
	if err := s.repository.CreateProvider(ctx, &row); err != nil {
		if errors.Is(err, ports.ErrConflict) {
			return 0, ErrConflict
		}
		return 0, err
	}
	return row.ID, nil
}
func (s *AIProviderService) PatchProvider(ctx context.Context, id uint64, req PatchAIProviderRequest) error {
	if s == nil || s.repository == nil {
		return errors.New("provider repository is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "AI 提供商 ID 无效")
	}
	patch := ports.ProviderPatch{Enabled: req.Enabled, Priority: req.Priority, Meta: req.Meta}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return ErrWithMessage(ErrInvalidParams, "AI 提供商名称不能为空")
		}
		patch.Name = &name
	}
	if req.ProviderType != nil {
		kind := normalizeAIProviderType(*req.ProviderType)
		if kind == "" {
			return ErrWithMessage(ErrInvalidParams, "AI 提供商类型不能为空")
		}
		patch.ProviderType = &kind
	}
	if req.VendorCode != nil {
		value := normalizeAIVendorCode(*req.VendorCode)
		patch.VendorCode = &value
	}
	if req.BaseURL != nil {
		value := strings.TrimSpace(*req.BaseURL)
		patch.BaseURL = &value
	}
	if req.AuthScheme != nil {
		value := normalizeAIAuthScheme(*req.AuthScheme)
		if value == "" {
			return ErrWithMessage(ErrInvalidParams, "AI provider auth scheme is invalid")
		}
		patch.AuthScheme = &value
	}
	if req.APIKey != nil {
		var encrypted *string
		if key := strings.TrimSpace(*req.APIKey); key != "" {
			value, err := encryptText(s.encryptionKey, key)
			if err != nil {
				return err
			}
			encrypted = &value
		}
		patch.APIKeyEnc = &encrypted
	}
	err := s.repository.PatchProvider(ctx, id, patch)
	if errors.Is(err, ports.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, ports.ErrConflict) {
		return ErrConflict
	}
	return err
}
func (s *AIProviderService) DeleteProvider(ctx context.Context, id uint64) error {
	if s == nil || s.repository == nil {
		return errors.New("provider repository is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "AI 提供商 ID 无效")
	}
	err := s.repository.DeleteProvider(ctx, id, time.Now().UTC())
	if errors.Is(err, ports.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, ports.ErrProviderInUse) {
		return ErrWithMessage(ErrConflict, "请先删除该提供商下的模型配置")
	}
	return err
}
func (s *AIProviderService) ListModels(ctx context.Context, req ListAIModelsRequest) ([]AIModelItem, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("provider repository is required")
	}
	rows, err := s.repository.ListModels(ctx, ports.ModelFilter{ProviderID: req.ProviderID, ModelType: normalizeAIModelType(req.ModelType), Enabled: req.Enabled})
	if err != nil {
		return nil, err
	}
	out := make([]AIModelItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, buildAIModelItem(row.Model, row.ProviderName))
	}
	return out, nil
}
func buildAIModelItem(row aidomain.AIModel, providerName string) AIModelItem {
	return AIModelItem{ID: row.ID, ProviderID: row.ProviderID, ProviderName: providerName, Name: row.Name, ModelCode: row.ModelCode, ModelType: row.ModelType, Enabled: row.Enabled, SupportsTools: row.SupportsTools, SupportsVision: row.SupportsVision, SupportsStreaming: row.SupportsStreaming, SupportsReasoning: row.SupportsReasoning, SupportsStructuredOutput: row.SupportsStructuredOutput, SupportsImageGeneration: row.SupportsImageGeneration, SupportsFileInput: row.SupportsFileInput, MaxInputTokens: row.MaxInputTokens, MaxOutputTokens: row.MaxOutputTokens, ContextWindow: row.ContextWindow, Meta: row.MetaJSON, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
}
func (s *AIProviderService) CreateModel(ctx context.Context, req CreateAIModelRequest) (uint64, error) {
	if s == nil || s.repository == nil {
		return 0, errors.New("provider repository is required")
	}
	if req.ProviderID == 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "模型提供商不能为空")
	}
	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.ModelCode)
	kind := normalizeAIModelType(req.ModelType)
	if name == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "模型名称不能为空")
	}
	if code == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "模型编码不能为空")
	}
	if kind == "" {
		return 0, ErrWithMessage(ErrInvalidParams, "模型类型不能为空")
	}
	row := aidomain.AIModel{ProviderID: req.ProviderID, Name: name, ModelCode: code, ModelType: kind, Enabled: boolOrDefault(req.Enabled, true), SupportsTools: boolOrDefault(req.SupportsTools, false), SupportsVision: boolOrDefault(req.SupportsVision, false), SupportsStreaming: boolOrDefault(req.SupportsStreaming, false), SupportsReasoning: boolOrDefault(req.SupportsReasoning, false), SupportsStructuredOutput: boolOrDefault(req.SupportsStructuredOutput, false), SupportsImageGeneration: boolOrDefault(req.SupportsImageGeneration, false), SupportsFileInput: boolOrDefault(req.SupportsFileInput, false), MaxInputTokens: intOrDefault(req.MaxInputTokens, 0), MaxOutputTokens: intOrDefault(req.MaxOutputTokens, 0), ContextWindow: intOrDefault(req.ContextWindow, 0), MetaJSON: req.Meta}
	err := s.repository.CreateModel(ctx, &row)
	if errors.Is(err, ports.ErrProviderNotFound) {
		return 0, ErrWithMessage(ErrInvalidParams, "模型提供商不存在")
	}
	if errors.Is(err, ports.ErrConflict) {
		return 0, ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}
func (s *AIProviderService) PatchModel(ctx context.Context, id uint64, req PatchAIModelRequest) error {
	if s == nil || s.repository == nil {
		return errors.New("provider repository is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "模型 ID 无效")
	}
	patch := ports.ModelPatch{ProviderID: req.ProviderID, Enabled: req.Enabled, SupportsTools: req.SupportsTools, SupportsVision: req.SupportsVision, SupportsStreaming: req.SupportsStreaming, SupportsReasoning: req.SupportsReasoning, SupportsStructuredOutput: req.SupportsStructuredOutput, SupportsImageGeneration: req.SupportsImageGeneration, SupportsFileInput: req.SupportsFileInput, MaxInputTokens: req.MaxInputTokens, MaxOutputTokens: req.MaxOutputTokens, ContextWindow: req.ContextWindow, Meta: req.Meta}
	if req.ProviderID != nil && *req.ProviderID == 0 {
		return ErrWithMessage(ErrInvalidParams, "模型提供商不能为空")
	}
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			return ErrWithMessage(ErrInvalidParams, "模型名称不能为空")
		}
		patch.Name = &value
	}
	if req.ModelCode != nil {
		value := strings.TrimSpace(*req.ModelCode)
		if value == "" {
			return ErrWithMessage(ErrInvalidParams, "模型编码不能为空")
		}
		patch.ModelCode = &value
	}
	if req.ModelType != nil {
		value := normalizeAIModelType(*req.ModelType)
		if value == "" {
			return ErrWithMessage(ErrInvalidParams, "模型类型不能为空")
		}
		patch.ModelType = &value
	}
	err := s.repository.PatchModel(ctx, id, patch)
	if errors.Is(err, ports.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, ports.ErrProviderNotFound) {
		return ErrWithMessage(ErrInvalidParams, "模型提供商不存在")
	}
	if errors.Is(err, ports.ErrConflict) {
		return ErrConflict
	}
	return err
}
func (s *AIProviderService) DeleteModel(ctx context.Context, id uint64) error {
	if s == nil || s.repository == nil {
		return errors.New("provider repository is required")
	}
	if id == 0 {
		return ErrWithMessage(ErrInvalidParams, "模型 ID 无效")
	}
	err := s.repository.DeleteModel(ctx, id, time.Now().UTC())
	if errors.Is(err, ports.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
func normalizeAIProviderType(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
func normalizeAIVendorCode(v string) string   { return strings.ToLower(strings.TrimSpace(v)) }
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
func normalizeAIModelType(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
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
