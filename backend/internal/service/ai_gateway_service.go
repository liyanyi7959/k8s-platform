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
	DiagnosticSummary string
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
	} else {
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
	base := "You are the built-in AI assistant of a Kubernetes management platform. The platform backend can directly collect live, read-only cluster evidence for the current scope. When evidence is provided, treat it as current platform data. State confirmed facts directly, separate them from inference, and do not say that you cannot access the cluster. Do not ask the user to run kubectl for data that the platform has already collected. If counts, lists, states, logs, metrics, events, or rollout details appear in evidence, answer with them directly. When evidence exists, do not lead with a generic checklist; lead with the confirmed findings from the evidence first. Only say evidence is insufficient when the evidence for this round is truly missing, partial, or failed. Format the final answer in Markdown, using short headings, lists, tables, and fenced code blocks whenever they improve readability."

	// 安全规则：强制人工确认
	safetyRules := `
## SAFETY RULES (MANDATORY)
1. NEVER execute write operations (create, update, delete, scale, restart) without explicit human confirmation.
2. ALL destructive operations (delete resource, delete pod, drain node) are HIGH RISK and require DOUBLE confirmation from two different operators.
3. When recommending a change, always state:
   - The exact action and target resource (kind/namespace/name)
   - The risk level (low/medium/high)
   - That human confirmation is required before execution
   - Potential side effects and rollback considerations
4. NEVER claim an operation has been executed. Use phrases like "proposed", "recommended", "pending confirmation".
5. For namespace-wide or cluster-wide operations, warn about blast radius.
6. When uncertain about safety, default to recommending caution and manual review.
7. Secret values are masked. Never attempt to reconstruct or expose masked data.`

	modeRule := ""
	if strings.TrimSpace(mode) == "chat" {
		modeRule = " Current mode is general assistance. Keep answers concise but evidence-based. You may explain, compare, and summarize, but you must still respect platform permissions and must not imply direct write execution."
	} else {
		modeRule = " Current mode is fault diagnosis. Prefer an answer structure of issue summary, key evidence, likely causes, impact scope, and next step. For write actions, only provide recommendations or proposals and never imply that a risky change has already been executed."
	}

	return base + safetyRules + modeRule
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

	lastUserIndex := -1
	for index := len(req.Messages) - 1; index >= 0; index-- {
		if strings.EqualFold(strings.TrimSpace(req.Messages[index].Role), "user") {
			lastUserIndex = index
			break
		}
	}
	for index, item := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role == "" {
			role = "user"
		}
		if role == "user" && index == lastUserIndex {
			continue
		}
		messages = append(messages, map[string]any{
			"role":    role,
			"content": item.Content,
		})
	}

	messages = appendOpenAIContextMessages(messages, req)
	if lastUserIndex >= 0 && lastUserIndex < len(req.Messages) {
		currentUser := req.Messages[lastUserIndex]
		messages = append(messages, buildOpenAIUserMessage("user", buildOpenAICurrentTurnContent(currentUser.Content, req.DiagnosticSummary), req.CurrentFiles, req.CurrentImages))
	}
	return messages
}

func buildOpenAICurrentTurnContent(content, diagnosticSummary string) string {
	userContent := strings.TrimSpace(content)
	summary := strings.TrimSpace(diagnosticSummary)
	if summary == "" {
		return userContent
	}
	if userContent == "" {
		return summary
	}
	return summary + "\n\nTreat the above evidence as confirmed current platform data for this round. If earlier conversation turns conflict with it, trust this evidence.\n\nCurrent user request:\n" + userContent
}

func appendOpenAIContextMessages(messages []map[string]any, req AIGatewayRequest) []map[string]any {
	if notes := strings.TrimSpace(req.DiagnosticNotes); notes != "" {
		messages = append(messages, map[string]any{
			"role": "system",
			"content": "Fresh platform diagnostic evidence exists for this round. It overrides any earlier assistant guesswork " +
				"or earlier missing-evidence statements in the conversation history.",
		})
		messages = append(messages, map[string]any{
			"role": "system",
			"content": "The following diagnostic notes were collected by the platform for this round. " +
				"Treat them as current evidence, cite confirmed facts directly, and clearly label any inference.\n" + notes,
		})
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "When evidence is partial, first summarize the confirmed facts from that evidence, then explicitly call out only the missing dimensions. Do not say that the platform provided no evidence when these notes are present.",
		})
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": "When the evidence already contains concrete counts, states, events, logs, or metrics, answer with them directly instead of falling back to generic kubectl guidance or a generic inspection checklist.",
		})
		return messages
	}

	return append(messages, map[string]any{
		"role":    "system",
		"content": "No platform diagnostic evidence was collected for this round. Do not pretend you saw live cluster state. Say evidence is insufficient, keep the answer bounded, and only suggest manual commands when the requested data is outside the platform's current read capabilities.",
	})
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
