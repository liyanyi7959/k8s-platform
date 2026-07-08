package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type AIGatewayMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIGatewayImage struct {
	Name        string
	ContentType string
	DataURL     string
}

type AIGatewayFileContext struct {
	Name        string
	ContentType string
	Content     string
	Size        int64
}

type AIGatewayRequest struct {
	ConversationID  uint64
	AssistantMode   string
	ProviderID      *uint64
	ModelID         *uint64
	PreferModelCode string
	Messages        []AIGatewayMessage
	CurrentImages   []AIGatewayImage
	CurrentFiles    []AIGatewayFileContext
	DiagnosticNotes string
	ScopeNote       string
}

type AIGatewayUsage struct {
	RequestTokens  int
	ResponseTokens int
	TotalTokens    int
	LatencyMS      int
}

type AIGatewayResponse struct {
	ProviderID       uint64
	ProviderName     string
	ModelID          uint64
	ModelName        string
	ModelCode        string
	Content          string
	SuggestedActions []AISuggestedAction
	Usage            AIGatewayUsage
}

type AIGatewayService struct {
	db            *gorm.DB
	encryptionKey string
	httpClient    *http.Client
}

func NewAIGatewayService(db *gorm.DB, encryptionKey string) *AIGatewayService {
	return &AIGatewayService{
		db:            db,
		encryptionKey: encryptionKey,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (s *AIGatewayService) Invoke(ctx context.Context, req AIGatewayRequest) (AIGatewayResponse, error) {
	if s.db == nil {
		return AIGatewayResponse{}, errors.New("db is required")
	}

	provider, aiModel, apiKey, err := s.resolveInvocationTarget(ctx, req)
	if err != nil {
		return AIGatewayResponse{}, err
	}

	if provider.ProviderType == "mock" {
		return s.invokeMock(provider, aiModel, req), nil
	}

	switch provider.ProviderType {
	case "openai", "compatible", "openai-compatible":
		return s.invokeOpenAICompatible(ctx, provider, aiModel, apiKey, req)
	default:
		return AIGatewayResponse{}, ErrWithMessage(ErrInvalidParams, "当前提供商类型暂不支持在线调用")
	}
}

func (s *AIGatewayService) resolveInvocationTarget(ctx context.Context, req AIGatewayRequest) (model.AIProvider, model.AIModel, string, error) {
	var aiModel model.AIModel
	q := s.db.WithContext(ctx).
		Table("ai_models AS m").
		Joins("JOIN ai_providers AS p ON p.id = m.provider_id").
		Where("m.deleted_at IS NULL AND p.deleted_at IS NULL AND m.enabled = 1 AND p.enabled = 1")

	switch {
	case req.ModelID != nil && *req.ModelID > 0:
		q = q.Where("m.id = ?", *req.ModelID)
	case strings.TrimSpace(req.PreferModelCode) != "":
		q = q.Where("m.model_code = ?", strings.TrimSpace(req.PreferModelCode))
	case req.ProviderID != nil && *req.ProviderID > 0:
		q = q.Where("m.provider_id = ?", *req.ProviderID)
	default:
		if routed := s.findRoutePreferredModel(ctx, req); routed != nil {
			q = q.Where("m.id = ?", *routed)
		}
	}

	if strings.TrimSpace(req.AssistantMode) == "chat" {
		q = q.Where("m.model_type IN ?", []string{"chat", "reasoning"})
	} else if false {
		q = q.Where("m.model_type IN ?", []string{"chat", "reasoning", "vision"})
	}
	if req.RequiresVision() {
		q = q.Where("m.supports_vision = 1")
	}
	if req.RequiresFileInput() {
		q = q.Where("m.supports_file_input = 1")
	}

	if err := q.Order("p.priority ASC, m.id DESC").First(&aiModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.AIProvider{}, model.AIModel{}, "", ErrWithMessage(ErrNotFound, "未找到可用的 AI 模型")
		}
		return model.AIProvider{}, model.AIModel{}, "", err
	}

	if provider, apiKey, providerErr := s.loadProviderWithKey(ctx, aiModel.ProviderID); providerErr == nil {
		return provider, aiModel, apiKey, nil
	} else if !errors.Is(providerErr, ErrNotFound) {
		return model.AIProvider{}, model.AIModel{}, "", providerErr
	} else if routeSettings, routeErr := s.loadRouteSettings(ctx); routeErr == nil && routeSettings != nil && routeSettings.AllowFallback {
		if routeSettings.DefaultFallbackProviderID != nil &&
			*routeSettings.DefaultFallbackProviderID > 0 &&
			*routeSettings.DefaultFallbackProviderID != aiModel.ProviderID {
			fallbackModel, fallbackErr := s.findFallbackModel(ctx, req, *routeSettings.DefaultFallbackProviderID)
			if fallbackErr == nil {
				fallbackProvider, fallbackKey, fallbackProviderErr := s.loadProviderWithKey(ctx, fallbackModel.ProviderID)
				if fallbackProviderErr == nil {
					return fallbackProvider, fallbackModel, fallbackKey, nil
				}
			}
		}
	}

	var provider model.AIProvider
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND enabled = 1 AND id = ?", aiModel.ProviderID).
		First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.AIProvider{}, model.AIModel{}, "", ErrWithMessage(ErrNotFound, "未找到可用的 AI 提供商")
		}
		return model.AIProvider{}, model.AIModel{}, "", err
	}

	apiKey := ""
	if provider.APIKeyEnc != nil && strings.TrimSpace(*provider.APIKeyEnc) != "" {
		plain, err := decryptText(s.encryptionKey, *provider.APIKeyEnc)
		if err != nil {
			return model.AIProvider{}, model.AIModel{}, "", err
		}
		apiKey = strings.TrimSpace(plain)
	}
	return provider, aiModel, apiKey, nil
}

func (s *AIGatewayService) baseEnabledModelQuery(ctx context.Context, req AIGatewayRequest) *gorm.DB {
	q := s.db.WithContext(ctx).
		Table("ai_models AS m").
		Joins("JOIN ai_providers AS p ON p.id = m.provider_id").
		Where("m.deleted_at IS NULL AND p.deleted_at IS NULL AND m.enabled = 1 AND p.enabled = 1")

	if strings.TrimSpace(req.AssistantMode) == "chat" {
		q = q.Where("m.model_type IN ?", []string{"chat", "reasoning"})
	} else {
		q = q.Where("m.model_type IN ?", []string{"chat", "reasoning", "vision"})
	}
	if req.RequiresVision() {
		q = q.Where("m.supports_vision = 1")
	}
	if req.RequiresFileInput() {
		q = q.Where("m.supports_file_input = 1")
	}
	return q
}

func (s *AIGatewayService) findRoutePreferredModel(ctx context.Context, req AIGatewayRequest) *uint64 {
	settings, err := s.loadRouteSettings(ctx)
	if err != nil || settings == nil {
		return nil
	}

	switch normalizeAIRoutingStrategy(settings.RoutingStrategy) {
	case "default_model_first", "capability_first", "priority_first":
	default:
		return nil
	}

	var candidate *uint64
	if req.RequiresVision() && settings.DefaultVisionModelID != nil && *settings.DefaultVisionModelID > 0 {
		candidate = settings.DefaultVisionModelID
	} else if strings.TrimSpace(req.AssistantMode) == "chat" {
		candidate = settings.DefaultChatModelID
	} else {
		candidate = settings.DefaultDiagnoseModelID
	}
	if candidate == nil || *candidate == 0 {
		return nil
	}
	return candidate
}

func (s *AIGatewayService) loadRouteSettings(ctx context.Context) (*model.AIRouteSetting, error) {
	var row model.AIRouteSetting
	if err := s.db.WithContext(ctx).First(&row, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (s *AIGatewayService) findFallbackModel(ctx context.Context, req AIGatewayRequest, providerID uint64) (model.AIModel, error) {
	var row model.AIModel
	if err := s.baseEnabledModelQuery(ctx, req).
		Where("m.provider_id = ?", providerID).
		Order("m.id DESC").
		First(&row).Error; err != nil {
		return model.AIModel{}, err
	}
	return row, nil
}

func (s *AIGatewayService) loadProviderWithKey(ctx context.Context, providerID uint64) (model.AIProvider, string, error) {
	var provider model.AIProvider
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND enabled = 1 AND id = ?", providerID).
		First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.AIProvider{}, "", ErrNotFound
		}
		return model.AIProvider{}, "", err
	}

	apiKey := ""
	if provider.AuthScheme != "none" && provider.APIKeyEnc != nil && strings.TrimSpace(*provider.APIKeyEnc) != "" {
		plain, err := decryptText(s.encryptionKey, *provider.APIKeyEnc)
		if err != nil {
			return model.AIProvider{}, "", err
		}
		apiKey = strings.TrimSpace(plain)
	}
	return provider, apiKey, nil
}

func (s *AIGatewayService) invokeMock(provider model.AIProvider, aiModel model.AIModel, req AIGatewayRequest) AIGatewayResponse {
	lastUser := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if strings.EqualFold(req.Messages[i].Role, "user") {
			lastUser = strings.TrimSpace(req.Messages[i].Content)
			break
		}
	}
	notes := strings.TrimSpace(req.DiagnosticNotes)
	if len([]rune(notes)) > 360 {
		notes = string([]rune(notes)[:360]) + "..."
	}
	content := "当前使用的是 mock AI 提供商，已打通会话、自动取证和回复链路。\n\n"
	if lastUser != "" {
		content += "你的问题摘要: " + lastUser + "\n"
	}
	if notes != "" {
		content += "\n已收集诊断上下文:\n" + notes + "\n"
	}
	content += "\n建议下一步: 如需接入真实模型，请在 AI 模型配置页启用兼容 OpenAI 协议的提供商。"
	return AIGatewayResponse{
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ModelID:      aiModel.ID,
		ModelName:    aiModel.Name,
		ModelCode:    aiModel.ModelCode,
		Content:      content,
		Usage: AIGatewayUsage{
			RequestTokens:  0,
			ResponseTokens: 0,
			TotalTokens:    0,
			LatencyMS:      1,
		},
	}
}

func (s *AIGatewayService) invokeOpenAICompatible(
	ctx context.Context,
	provider model.AIProvider,
	aiModel model.AIModel,
	apiKey string,
	req AIGatewayRequest,
) (AIGatewayResponse, error) {
	if apiKey == "" {
		return AIGatewayResponse{}, ErrWithMessage(ErrInvalidParams, "当前 AI 提供商未配置 API Key")
	}

	messages := buildOpenAICompatibleMessages(req)
	if notes := strings.TrimSpace(req.DiagnosticNotes); false {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "以下是平台自动收集的只读诊断信息，请优先基于这些证据分析，不要编造缺失事实。\n" + notes,
		})
	} else {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "当前没有拿到任何平台诊断证据。不要声称已经看到集群状态、Pod 状态、事件、日志或具体资源异常；请明确说明证据不足，并只给出下一步排查建议。",
		})
	}
	if false {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "No platform diagnostic evidence was collected for this round. Do not pretend you saw cluster state. Say that this round lacks evidence, ask to narrow scope or retry collection, and only suggest manual commands when the requested data is outside the platform's current read capabilities.",
		})
	} else if false {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "Treat the provided diagnostic notes as platform-collected evidence. If the notes already contain concrete counts, lists, states, logs, metrics, or events, answer with them directly and do not fall back to generic kubectl instructions.",
		})
	}
	for _, item := range []AIGatewayMessage{} {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role == "" {
			role = "user"
		}
		messages = append(messages, map[string]any{
			"role":    role,
			"content": item.Content,
		})
	}

	payload := map[string]any{
		"model":       aiModel.ModelCode,
		"messages":    messages,
		"temperature": 0.2,
		"stream":      false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return AIGatewayResponse{}, err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := baseURL + "/chat/completions"

	reqStart := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return AIGatewayResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return AIGatewayResponse{}, context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return AIGatewayResponse{}, context.DeadlineExceeded
		}
		return AIGatewayResponse{}, ErrWithMessage(ErrK8sNetwork, "AI 提供商连接失败")
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return AIGatewayResponse{}, context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return AIGatewayResponse{}, context.DeadlineExceeded
		}
		return AIGatewayResponse{}, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return AIGatewayResponse{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("AI 提供商返回异常状态: %d", httpResp.StatusCode))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return AIGatewayResponse{}, ErrWithMessage(ErrInvalidParams, "AI 提供商响应格式无法解析")
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return AIGatewayResponse{}, ErrWithMessage(ErrInvalidParams, "AI 提供商未返回有效回答")
	}

	return AIGatewayResponse{
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ModelID:      aiModel.ID,
		ModelName:    aiModel.Name,
		ModelCode:    aiModel.ModelCode,
		Content:      strings.TrimSpace(parsed.Choices[0].Message.Content),
		Usage: AIGatewayUsage{
			RequestTokens:  parsed.Usage.PromptTokens,
			ResponseTokens: parsed.Usage.CompletionTokens,
			TotalTokens:    parsed.Usage.TotalTokens,
			LatencyMS:      int(time.Since(reqStart).Milliseconds()),
		},
	}, nil
}

// AIGatewayStreamResult holds the final result after streaming completes.
type AIGatewayStreamResult struct {
	ProviderID   uint64
	ProviderName string
	ModelID      uint64
	ModelName    string
	ModelCode    string
	Usage        AIGatewayUsage
}

// InvokeStream starts a streaming chat completion request. It returns:
//   - an io.ReadCloser for the SSE event stream (caller must close)
//   - the resolved provider/model metadata
//   - an error if the request setup or HTTP call fails
//
// The caller is responsible for reading and parsing SSE lines from the reader,
// and for closing it when done.
func (s *AIGatewayService) InvokeStream(ctx context.Context, req AIGatewayRequest) (io.ReadCloser, AIGatewayStreamResult, error) {
	var result AIGatewayStreamResult
	if s.db == nil {
		return nil, result, errors.New("db is required")
	}

	provider, aiModel, apiKey, err := s.resolveInvocationTarget(ctx, req)
	if err != nil {
		return nil, result, err
	}

	if apiKey == "" {
		return nil, result, ErrWithMessage(ErrInvalidParams, "当前 AI 提供商未配置 API Key")
	}

	result.ProviderID = provider.ID
	result.ProviderName = provider.Name
	result.ModelID = aiModel.ID
	result.ModelName = aiModel.Name
	result.ModelCode = aiModel.ModelCode

	messages := buildOpenAICompatibleMessages(req)
	if notes := strings.TrimSpace(req.DiagnosticNotes); false {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "以下是平台自动收集的只读诊断信息，请优先基于这些证据分析，不要编造缺失事实。\n" + notes,
		})
	} else if false {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "当前没有拿到任何平台诊断证据。不要声称已经看到集群状态、Pod 状态、事件、日志或具体资源异常；请明确说明证据不足，并只给出下一步排查建议。",
		})
	}
	for _, item := range []AIGatewayMessage{} {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role == "" {
			role = "user"
		}
		messages = append(messages, map[string]any{"role": role, "content": item.Content})
	}

	payload := map[string]any{
		"model":       aiModel.ModelCode,
		"messages":    messages,
		"temperature": 0.2,
		"stream":      true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, result, err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := baseURL + "/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, result, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return nil, result, context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, result, context.DeadlineExceeded
		}
		return nil, result, ErrWithMessage(ErrK8sNetwork, "AI 提供商连接失败")
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		_ = httpResp.Body.Close()
		return nil, result, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("AI 提供商返回异常状态: %d", httpResp.StatusCode))
	}

	return httpResp.Body, result, nil
}

func buildAISystemPromptV2(mode string) string {
	base := "You are the built-in AI assistant of a Kubernetes management platform. The platform backend can directly collect live, read-only cluster evidence for the current scope. When evidence is provided, treat it as current platform data. State confirmed facts directly, separate them from inference, and do not say that you cannot access the cluster. Do not ask the user to run kubectl for data that the platform has already collected. If counts, lists, states, logs, metrics, events, or rollout details appear in evidence, answer with them directly. Only say evidence is insufficient when the evidence for this round is truly missing, partial, or failed."
	if strings.TrimSpace(mode) == "chat" {
		return base + " Current mode is general assistance. Keep answers concise but evidence-based. You may explain, compare, and summarize, but you must still respect platform permissions and must not imply direct write execution."
	}
	return base + " Current mode is fault diagnosis. Prefer an answer structure of issue summary, key evidence, likely causes, impact scope, and next step. For write actions, only provide recommendations or proposals and never imply that a risky change has already been executed."
}

func (r AIGatewayRequest) RequiresVision() bool {
	return len(r.CurrentImages) > 0
}

func (r AIGatewayRequest) RequiresFileInput() bool {
	return len(r.CurrentFiles) > 0
}

func buildOpenAICompatibleMessages(req AIGatewayRequest) []map[string]any {
	systemPrompt := buildAISystemPromptV2(req.AssistantMode)
	messages := make([]map[string]any, 0, len(req.Messages)+4)
	messages = append(messages, map[string]any{
		"role":    "system",
		"content": systemPrompt,
	})
	if scopeNote := strings.TrimSpace(req.ScopeNote); scopeNote != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": scopeNote,
		})
	}
	if notes := strings.TrimSpace(req.DiagnosticNotes); notes != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "浠ヤ笅鏄钩鍙拌嚜鍔ㄦ敹闆嗙殑鍙璇婃柇淇℃伅锛岃浼樺厛鍩轰簬杩欎簺璇佹嵁鍒嗘瀽锛屼笉瑕佺紪閫犵己澶变簨瀹炪€俓n" + notes,
		})
	} else {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "褰撳墠娌℃湁鎷垮埌浠讳綍骞冲彴璇婃柇璇佹嵁銆備笉瑕佸０绉板凡缁忕湅鍒伴泦缇ょ姸鎬併€丳od 鐘舵€併€佷簨浠躲€佹棩蹇楁垨鍏蜂綋璧勬簮寮傚父锛涜鏄庣‘璇存槑璇佹嵁涓嶈冻锛屽苟鍙粰鍑轰笅涓€姝ユ帓鏌ュ缓璁€?,
		})
	}
	if strings.TrimSpace(req.DiagnosticNotes) == "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "No platform diagnostic evidence was collected for this round. Do not pretend you saw cluster state. Say that this round lacks evidence, ask to narrow scope or retry collection, and only suggest manual commands when the requested data is outside the platform's current read capabilities.",
		})
	} else {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "Treat the provided diagnostic notes as platform-collected evidence. If the notes already contain concrete counts, lists, states, logs, metrics, or events, answer with them directly and do not fall back to generic kubectl instructions.",
		})
	}

	lastUserIndex := len(req.Messages) - 1
	for index, item := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role == "" {
			role = "user"
		}
		if role == "user" && index == lastUserIndex {
			messages = append(messages, buildOpenAIUserMessage(role, item.Content, req.CurrentFiles, req.CurrentImages))
			continue
		}
		messages = append(messages, map[string]any{
			"role":    role,
			"content": item.Content,
		})
	}
	return messages
}

func buildOpenAIUserMessage(role, content string, files []AIGatewayFileContext, images []AIGatewayImage) map[string]any {
	textContent := buildAIGatewayUserText(content, files)
	if len(images) == 0 {
		return map[string]any{
			"role":    role,
			"content": textContent,
		}
	}

	parts := make([]map[string]any, 0, len(images)+1)
	parts = append(parts, map[string]any{
		"type": "text",
		"text": textContent,
	})
	for _, image := range images {
		if strings.TrimSpace(image.DataURL) == "" {
			continue
		}
		parts = append(parts, map[string]any{
			"type": "image_url",
			"image_url": map[string]any{
				"url": image.DataURL,
			},
		})
	}
	return map[string]any{
		"role":    role,
		"content": parts,
	}
}

func buildAIGatewayUserText(content string, files []AIGatewayFileContext) string {
	content = strings.TrimSpace(content)
	if len(files) == 0 {
		return content
	}

	var builder strings.Builder
	builder.WriteString(content)
	builder.WriteString("\n\nAttached file excerpts:")
	for _, file := range files {
		builder.WriteString("\n\n[")
		builder.WriteString(strings.TrimSpace(file.Name))
		builder.WriteString("]")
		if contentType := strings.TrimSpace(file.ContentType); contentType != "" {
			builder.WriteString(" (")
			builder.WriteString(contentType)
			builder.WriteString(")")
		}
		builder.WriteString("\n")
		builder.WriteString(truncateAIGatewayFileText(strings.TrimSpace(file.Content)))
	}
	return strings.TrimSpace(builder.String())
}

func truncateAIGatewayFileText(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "(empty file)"
	}
	runes := []rune(content)
	if len(runes) <= aiChatMaxFileTextRunes {
		return content
	}
	return string(runes[:aiChatMaxFileTextRunes]) + "\n...[truncated]"
}

func buildAIGatewayScopeNote(namespace, kind, name string) string {
	ns := strings.TrimSpace(namespace)
	resKind := strings.TrimSpace(kind)
	resName := strings.TrimSpace(name)
	if ns == "" && resKind == "" && resName == "" {
		return ""
	}

	parts := make([]string, 0, 3)
	if ns != "" {
		parts = append(parts, "namespace="+ns)
	}
	if resKind != "" {
		parts = append(parts, "kind="+resKind)
	}
	if resName != "" {
		parts = append(parts, "name="+resName)
	}

	base := "Current request scope is fixed to " + strings.Join(parts, ", ") + "."
	if resKind != "" && resName != "" {
		return base + " Answer only for this target resource. Do not list sibling resources, namespace-wide inventories, or cluster-wide summaries unless the user explicitly asks to broaden scope."
	}
	if ns != "" {
		return base + " Keep the answer inside this namespace scope unless the user explicitly asks to broaden scope."
	}
	return base + " Keep the answer inside this current scope unless the user explicitly asks to broaden scope."
}

func buildAISystemPrompt(mode string) string {
	base := "你是 Kubernetes 平台内置的 AI 助手。请基于提供的集群证据回答，严格区分“已确认事实”和“推测判断”。如果证据中已经包含明确数量、列表或状态，必须直接给出结论，不要回答“无法确定”。只有证据完全缺失时才说明不足。"
	if strings.TrimSpace(mode) == "chat" {
		return base + "当前模式为通用协助，但仍然不允许绕过平台权限或直接执行写操作。"
	}
	return base + "当前模式为故障诊断，请优先输出问题摘要、可能原因、关键证据、影响范围和建议下一步。涉及写操作时只给建议，不可默认执行。"
}
