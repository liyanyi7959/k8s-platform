package service

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	aigateway "k8s-platform-backend/internal/ai/adapters/gateway"
	aiapp "k8s-platform-backend/internal/ai/application"
	model "k8s-platform-backend/internal/ai/domain"
)

type (
	AIChatUploadInput       = aiapp.AIChatUploadInput
	AIFileService           = aiapp.AIFileService
	AIMessageAttachmentItem = aiapp.AIMessageAttachmentItem
	AIGatewayService        = aigateway.AIGatewayService
	AIGatewayRequest        = aigateway.AIGatewayRequest
	AIGatewayResponse       = aigateway.AIGatewayResponse
	AIGatewayStreamResult   = aigateway.AIGatewayStreamResult
	AIGatewayUsage          = aigateway.AIGatewayUsage
	AIGatewayMessage        = aigateway.AIGatewayMessage
	AIGatewayImage          = aigateway.AIGatewayImage
	AIGatewayFileContext    = aigateway.AIGatewayFileContext
	AIToolSpec              = aigateway.AIToolSpec
	AIToolCall              = aigateway.AIToolCall
)

const (
	aiUploadKindImage = aiapp.UploadKindImage
	aiUploadKindText  = aiapp.UploadKindText
)

type AIChatRequest struct {
	ClusterID      uint64              `json:"-"`
	ConversationID *uint64             `json:"conversation_id"`
	Message        string              `json:"message"`
	AssistantMode  string              `json:"assistant_mode"`
	ProviderID     *uint64             `json:"provider_id"`
	ModelID        *uint64             `json:"model_id"`
	PreferModel    string              `json:"prefer_model"`
	Namespace      string              `json:"namespace"`
	ResourceKind   string              `json:"resource_kind"`
	ResourceName   string              `json:"resource_name"`
	Images         []AIChatImageInput  `json:"images"`
	Uploads        []AIChatUploadInput `json:"-"`
	UserPerms      []string            `json:"-"`
}

type AIChatImageInput struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	DataURL     string `json:"data_url"`
	Size        int64  `json:"size"`
}

type AIChatResponse struct {
	ConversationID     uint64                 `json:"conversation_id"`
	UserMessageID      uint64                 `json:"user_message_id"`
	AssistantMessageID uint64                 `json:"assistant_message_id"`
	AssistantMessage   string                 `json:"assistant_message"`
	ProviderName       string                 `json:"provider_name"`
	ModelName          string                 `json:"model_name"`
	ModelCode          string                 `json:"model_code"`
	ToolCalls          []AIToolCallItem       `json:"tool_calls"`
	ActionProposals    []AIActionProposalItem `json:"action_proposals"`
}

type AIChatService struct {
	db        *gorm.DB
	gateway   *AIGatewayService
	toolSvc   *AIToolService
	actionSvc *AIActionService
	fileSvc   *AIFileService
}

func NewAIChatService(
	db *gorm.DB,
	gateway *AIGatewayService,
	toolSvc *AIToolService,
	actionSvc *AIActionService,
	fileSvc *AIFileService,
) *AIChatService {
	return &AIChatService{db: db, gateway: gateway, toolSvc: toolSvc, actionSvc: actionSvc, fileSvc: fileSvc}
}

type aiMessageRequestScope struct {
	AssistantMode string
	Namespace     string
	ResourceKind  string
	ResourceName  string
}

type aiPreparedConversationTurn struct {
	message                    string
	conversation               model.AIConversation
	selectedProviderID         *uint64
	selectedModelID            *uint64
	normalizedImages           []model.JSONMap
	gatewayImages              []AIGatewayImage
	fileContexts               []AIGatewayFileContext
	userMessage                model.AIMessage
	assistantMessageStructured model.JSONMap
}

func prepareAIChatInputs(req AIChatRequest) ([]AIChatUploadInput, []model.JSONMap, []AIGatewayImage, []AIGatewayFileContext, error) {
	inlineUploads, err := inlineAIImagesToUploads(req.Images)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	uploads := append([]AIChatUploadInput(nil), req.Uploads...)
	uploads = append(uploads, inlineUploads...)

	normalizedImages := normalizeAIChatImages(req.Images)
	normalizedImages = append(normalizedImages, normalizeAIChatUploadImages(req.Uploads)...)
	gatewayImages := buildAIGatewayImages(normalizedImages)
	fileContexts := buildAIGatewayFileContexts(uploads)
	return uploads, normalizedImages, gatewayImages, fileContexts, nil
}

func inlineAIImagesToUploads(images []AIChatImageInput) ([]AIChatUploadInput, error) {
	if len(images) == 0 {
		return nil, nil
	}

	out := make([]AIChatUploadInput, 0, len(images))
	for _, image := range images {
		dataURL := strings.TrimSpace(image.DataURL)
		if dataURL == "" {
			continue
		}
		contentType, raw, err := decodeAIDataURL(dataURL)
		if err != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "图片附件格式无效")
		}
		if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
			return nil, ErrWithMessage(ErrInvalidParams, "仅支持图片 data url")
		}
		name := strings.TrimSpace(image.Name)
		if name == "" {
			name = "image"
		}
		out = append(out, AIChatUploadInput{
			Name:        name,
			ContentType: strings.TrimSpace(image.ContentType),
			Size:        maxInt64(image.Size, int64(len(raw))),
			FileKind:    aiUploadKindImage,
			Bytes:       raw,
			DataURL:     dataURL,
		})
	}
	return out, nil
}

func decodeAIDataURL(dataURL string) (string, []byte, error) {
	trimmed := strings.TrimSpace(dataURL)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return "", nil, ErrWithMessage(ErrInvalidParams, "invalid data url")
	}
	parts := strings.SplitN(trimmed, ",", 2)
	if len(parts) != 2 {
		return "", nil, ErrWithMessage(ErrInvalidParams, "invalid data url")
	}
	meta := strings.TrimPrefix(parts[0], "data:")
	metaParts := strings.Split(meta, ";")
	if len(metaParts) == 0 {
		return "", nil, ErrWithMessage(ErrInvalidParams, "invalid data url")
	}
	contentType := strings.TrimSpace(metaParts[0])
	if !containsAny(strings.ToLower(parts[0]), ";base64") {
		return "", nil, ErrWithMessage(ErrInvalidParams, "only base64 data url is supported")
	}
	raw, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, err
	}
	return contentType, raw, nil
}

func normalizeAIChatUploadImages(uploads []AIChatUploadInput) []model.JSONMap {
	if len(uploads) == 0 {
		return nil
	}

	out := make([]model.JSONMap, 0, len(uploads))
	for _, upload := range uploads {
		if upload.FileKind != aiUploadKindImage || strings.TrimSpace(upload.DataURL) == "" {
			continue
		}
		out = append(out, model.JSONMap{
			"name":         strings.TrimSpace(upload.Name),
			"content_type": strings.TrimSpace(upload.ContentType),
			"data_url":     strings.TrimSpace(upload.DataURL),
			"size":         maxInt64(upload.Size, int64(len(upload.Bytes))),
		})
	}
	return out
}

func buildAIGatewayImages(images []model.JSONMap) []AIGatewayImage {
	if len(images) == 0 {
		return nil
	}
	out := make([]AIGatewayImage, 0, len(images))
	for _, image := range images {
		out = append(out, AIGatewayImage{
			Name:        strings.TrimSpace(fmt.Sprint(image["name"])),
			ContentType: strings.TrimSpace(fmt.Sprint(image["content_type"])),
			DataURL:     strings.TrimSpace(fmt.Sprint(image["data_url"])),
		})
	}
	return out
}

func buildAIGatewayFileContexts(uploads []AIChatUploadInput) []AIGatewayFileContext {
	if len(uploads) == 0 {
		return nil
	}
	out := make([]AIGatewayFileContext, 0, len(uploads))
	for _, upload := range uploads {
		if upload.FileKind != aiUploadKindText {
			continue
		}
		out = append(out, AIGatewayFileContext{
			Name:        strings.TrimSpace(upload.Name),
			ContentType: strings.TrimSpace(upload.ContentType),
			Content:     strings.TrimSpace(upload.TextContent),
			Size:        maxInt64(upload.Size, int64(len(upload.Bytes))),
		})
	}
	return out
}

func validateAIChatModelInputs(aiModel model.AIModel, images []model.JSONMap, files []AIGatewayFileContext) error {
	if len(images) > 0 && !aiModel.SupportsVision {
		return ErrWithMessage(ErrInvalidParams, "当前模型不支持图片输入，请切换到支持视觉的模型")
	}
	if len(files) > 0 && !aiModel.SupportsFileInput {
		return ErrWithMessage(ErrInvalidParams, "当前模型不支持文件输入，请切换到支持文件输入的模型")
	}
	return nil
}

func buildAIMessageAttachmentRefs(attachments []AIMessageAttachmentItem) []model.JSONMap {
	if len(attachments) == 0 {
		return nil
	}
	out := make([]model.JSONMap, 0, len(attachments))
	for _, item := range attachments {
		out = append(out, model.JSONMap{
			"id":            item.ID,
			"original_name": item.OriginalName,
			"content_type":  item.ContentType,
			"file_size":     item.FileSize,
			"file_kind":     item.FileKind,
			"download_url":  item.DownloadURL,
		})
	}
	return out
}

func (s *AIChatService) startConversationTurn(
	ctx context.Context,
	userID uint64,
	username string,
	req AIChatRequest,
) (_ aiPreparedConversationTurn, err error) {
	message := strings.TrimSpace(req.Message)
	if req.ClusterID == 0 {
		return aiPreparedConversationTurn{}, ErrWithMessage(ErrInvalidParams, "集群 ID 无效")
	}
	if message == "" {
		return aiPreparedConversationTurn{}, ErrWithMessage(ErrInvalidParams, "消息内容不能为空")
	}

	conversation, err := s.ensureConversation(ctx, userID, username, req)
	if err != nil {
		return aiPreparedConversationTurn{}, err
	}

	now := time.Now().UTC()
	if err := s.beginConversationRun(ctx, conversation.ID, now); err != nil {
		return aiPreparedConversationTurn{}, err
	}
	runStarted := true
	defer func() {
		if err != nil && runStarted {
			s.resetConversationRun(conversation.ID)
		}
	}()

	selectedProviderID, selectedModelID := resolveAIInvocationTarget(conversation, req)
	uploads, normalizedImages, gatewayImages, fileContexts, err := prepareAIChatInputs(req)
	if err != nil {
		return aiPreparedConversationTurn{}, err
	}
	if len(uploads) > 0 && s.fileSvc == nil {
		return aiPreparedConversationTurn{}, errors.New("ai file service is required")
	}
	if len(gatewayImages) > 0 || len(fileContexts) > 0 {
		_, aiModel, _, err := s.gateway.ResolveInvocationTarget(ctx, AIGatewayRequest{
			AssistantMode:   conversation.AssistantMode,
			ProviderID:      selectedProviderID,
			ModelID:         selectedModelID,
			PreferModelCode: req.PreferModel,
			CurrentImages:   gatewayImages,
			CurrentFiles:    fileContexts,
		})
		if err != nil {
			return aiPreparedConversationTurn{}, err
		}
		if err := validateAIChatModelInputs(aiModel, normalizedImages, fileContexts); err != nil {
			return aiPreparedConversationTurn{}, err
		}
	}

	userMessageStructured := buildAIMessageScopeSnapshot(
		conversation,
		req,
		selectedProviderID,
		selectedModelID,
		normalizedImages,
		nil,
	)
	userMessage := model.AIMessage{
		ConversationID: conversation.ID,
		Role:           "user",
		MessageType:    "text",
		Content:        message,
		Status:         "created",
		StructuredJSON: model.JSONMap(userMessageStructured),
		CreatedBy:      userID,
	}
	if err := s.db.WithContext(ctx).Create(&userMessage).Error; err != nil {
		return aiPreparedConversationTurn{}, err
	}

	var attachments []AIMessageAttachmentItem
	if len(uploads) > 0 {
		attachments, err = s.fileSvc.SaveChatUploads(ctx, conversation.ID, userMessage.ID, userID, username, uploads)
		if err != nil {
			return aiPreparedConversationTurn{}, err
		}
	}
	if len(attachments) > 0 {
		userMessageStructured = buildAIMessageScopeSnapshot(
			conversation,
			req,
			selectedProviderID,
			selectedModelID,
			normalizedImages,
			attachments,
		)
		if err := s.db.WithContext(ctx).
			Model(&model.AIMessage{}).
			Where("id = ?", userMessage.ID).
			Update("structured_json", userMessageStructured).Error; err != nil {
			return aiPreparedConversationTurn{}, err
		}
		userMessage.StructuredJSON = model.JSONMap(userMessageStructured)
	}

	runStarted = false
	return aiPreparedConversationTurn{
		message:            message,
		conversation:       conversation,
		selectedProviderID: selectedProviderID,
		selectedModelID:    selectedModelID,
		normalizedImages:   normalizedImages,
		gatewayImages:      gatewayImages,
		fileContexts:       fileContexts,
		userMessage:        userMessage,
		assistantMessageStructured: buildAIMessageScopeSnapshot(
			conversation,
			req,
			selectedProviderID,
			selectedModelID,
			nil,
			attachments,
		),
	}, nil
}

// buildFunctionCallingTools 从工具注册表构建 LLM function calling 工具列表
// 只暴露只读工具（query/inspect/export），不暴露变更提案工具（proposal）
func (s *AIChatService) buildFunctionCallingTools(userPerms []string) []AIToolSpec {
	if s == nil || s.toolSvc == nil {
		return nil
	}
	defs := s.toolSvc.ListDefinitions()
	tools := make([]AIToolSpec, 0, len(defs))
	for _, def := range defs {
		// 排除变更提案类工具
		if def.Category == "proposal" {
			continue
		}
		// 检查用户权限
		if !hasAllPermissions(userPerms, def.RequiredPermissions) {
			continue
		}
		tools = append(tools, AIToolSpec{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  aiToolSpecToJSONSchema(def.InputSchema),
		})
	}
	return tools
}

// aiToolSpecToJSONSchema 将简化的 InputSchema 转换为 OpenAI function calling 所需的 JSON Schema
// cluster_id 由平台注入，不暴露给 LLM
func aiToolSpecToJSONSchema(input model.JSONMap) model.JSONMap {
	if len(input) == 0 {
		return model.JSONMap{
			"type":       "object",
			"properties": model.JSONMap{},
		}
	}

	properties := make(model.JSONMap, len(input))
	required := make([]string, 0, len(input))

	for key, value := range input {
		// cluster_id 由平台自动注入，不暴露给 LLM
		if key == "cluster_id" {
			continue
		}

		var prop model.JSONMap
		isOptional := false

		switch v := value.(type) {
		case string:
			typeStr := v
			if strings.HasSuffix(typeStr, "?") {
				isOptional = true
				typeStr = strings.TrimSuffix(typeStr, "?")
			}
			if strings.Contains(typeStr, "|") {
				// 枚举类型："Deployment|StatefulSet|DaemonSet"
				options := strings.Split(typeStr, "|")
				prop = model.JSONMap{
					"type": "string",
					"enum": options,
				}
			} else {
				prop = model.JSONMap{"type": typeStr}
			}
		case []string:
			if len(v) > 0 {
				prop = model.JSONMap{
					"type":  "array",
					"items": model.JSONMap{"type": v[0]},
				}
			} else {
				prop = model.JSONMap{"type": "string"}
			}
		case []any:
			if len(v) > 0 {
				if s, ok := v[0].(string); ok {
					prop = model.JSONMap{
						"type":  "array",
						"items": model.JSONMap{"type": s},
					}
				} else {
					prop = model.JSONMap{"type": "array"}
				}
			} else {
				prop = model.JSONMap{"type": "array"}
			}
		default:
			prop = model.JSONMap{"type": "string"}
		}

		properties[key] = prop

		if !isOptional {
			required = append(required, key)
		}
	}

	schema := model.JSONMap{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// runFunctionCallingLoop 执行 LLM function calling 工具调用循环
// gatewayReq 应已设置 Messages 和 Tools；toolReq 提供工具执行上下文
// 返回：更新后的消息历史、工具调用记录、最后一次网关响应
func (s *AIChatService) runFunctionCallingLoop(
	ctx context.Context,
	gatewayReq AIGatewayRequest,
	toolReq AIToolContextRequest,
	onChunk func(AIStreamChunk),
) ([]AIGatewayMessage, []AIToolCallItem, AIGatewayResponse, error) {
	history := gatewayReq.Messages
	toolCalls := make([]AIToolCallItem, 0, 6)
	var lastResp AIGatewayResponse

	maxToolRounds := 5
	for round := 0; round < maxToolRounds; round++ {
		resp, err := s.gateway.Invoke(ctx, gatewayReq)
		if err != nil {
			return history, toolCalls, lastResp, err
		}
		lastResp = resp

		// LLM 没有请求工具调用，直接使用文本回答
		if len(resp.ToolCalls) == 0 {
			return history, toolCalls, resp, nil
		}

		// 追加 assistant 的 tool_calls 消息到历史
		history = append(history, AIGatewayMessage{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// 执行 LLM 请求的每个工具
		for _, tc := range resp.ToolCalls {
			// 推送工具执行进度
			if onChunk != nil {
				onChunk(AIStreamChunk{Type: "progress", Progress: "正在执行 " + tc.Name + "..."})
			}

			var input map[string]any
			if strings.TrimSpace(tc.Arguments) != "" {
				_ = json.Unmarshal([]byte(tc.Arguments), &input)
			}

			item, block := s.toolSvc.Execute(ctx, toolReq, tc.Name, input)
			if item.ID > 0 {
				toolCalls = append(toolCalls, item)
			}

			// 构建工具结果消息内容
			resultContent := strings.TrimSpace(block)
			if resultContent == "" {
				resultContent = strings.TrimSpace(item.ResultSummary)
			}
			if resultContent == "" && item.ErrorMessage != "" {
				resultContent = "Tool " + tc.Name + " failed: " + item.ErrorMessage
			}
			if resultContent == "" {
				resultContent = "{}"
			}

			// 工具结果截断到 4000 字符，防止撑爆上下文窗口
			if len(resultContent) > 4000 {
				resultContent = resultContent[:4000] + "\n... (truncated, total " + strconv.Itoa(len(resultContent)) + " chars)"
			}

			history = append(history, AIGatewayMessage{
				Role:       "tool",
				Content:    resultContent,
				ToolCallID: tc.ID,
			})
		}

		// 更新请求消息历史，进入下一轮（跳过 last-user 增强和诊断注入）
		gatewayReq.Messages = history
		gatewayReq.FunctionCallingMode = true
	}

	// 达到最大轮次，做最后一次无工具调用以生成总结回答
	gatewayReq.Tools = nil
	finalResp, err := s.gateway.Invoke(ctx, gatewayReq)
	if err != nil {
		return history, toolCalls, lastResp, nil
	}
	return history, toolCalls, finalResp, nil
}

func (s *AIChatService) SendMessage(ctx context.Context, userID uint64, username string, req AIChatRequest) (AIChatResponse, error) {
	if s.db == nil {
		return AIChatResponse{}, errors.New("db is required")
	}
	if s.gateway == nil || s.toolSvc == nil {
		return AIChatResponse{}, errors.New("ai dependencies are required")
	}
	turn, err := s.startConversationTurn(ctx, userID, username, req)
	if err != nil {
		return AIChatResponse{}, err
	}

	runCompleted := false
	defer func() {
		if !runCompleted {
			s.resetConversationRun(turn.conversation.ID)
		}
	}()

	toolCalls := make([]AIToolCallItem, 0, 6)
	diagnosticNotes := ""
	diagnosticSummary := ""

	// 检查当前模型是否支持 function calling
	modelSupportsTools := s.gateway.ModelSupportsTools(ctx, AIGatewayRequest{
		AssistantMode:   turn.conversation.AssistantMode,
		ProviderID:      turn.selectedProviderID,
		ModelID:         turn.selectedModelID,
		PreferModelCode: req.PreferModel,
	})

	var assistantResp AIGatewayResponse

	if modelSupportsTools {
		// function calling 模式：LLM 自主决定调用什么工具
		history, err := s.buildGatewayMessages(ctx, turn.conversation.ID, aiMessageRequestScope{
			AssistantMode: turn.conversation.AssistantMode,
			Namespace:     strings.TrimSpace(req.Namespace),
			ResourceKind:  strings.TrimSpace(req.ResourceKind),
			ResourceName:  strings.TrimSpace(req.ResourceName),
		}, turn.userMessage.ID)
		if err != nil {
			return AIChatResponse{}, err
		}
		history = append(history, AIGatewayMessage{
			Role:    "user",
			Content: turn.message,
		})

		tools := s.buildFunctionCallingTools(req.UserPerms)
		toolReq := AIToolContextRequest{
			ConversationID: turn.conversation.ID,
			MessageID:      turn.userMessage.ID,
			ClusterID:      req.ClusterID,
			UserID:         userID,
			Username:       username,
			UserPerms:      req.UserPerms,
			Query:          turn.message,
			Namespace:      strings.TrimSpace(req.Namespace),
			ResourceKind:   strings.TrimSpace(req.ResourceKind),
			ResourceName:   strings.TrimSpace(req.ResourceName),
		}

		gatewayReq := AIGatewayRequest{
			ConversationID:  turn.conversation.ID,
			AssistantMode:   turn.conversation.AssistantMode,
			ProviderID:      turn.selectedProviderID,
			ModelID:         turn.selectedModelID,
			PreferModelCode: req.PreferModel,
			Messages:        history,
			CurrentImages:   turn.gatewayImages,
			CurrentFiles:    turn.fileContexts,
			ScopeNote:       aigateway.BuildScopeNote(req.Namespace, req.ResourceKind, req.ResourceName),
			Tools:           tools,
		}

		_, toolCalls, assistantResp, err = s.runFunctionCallingLoop(ctx, gatewayReq, toolReq, nil)
		if err != nil {
			// function calling 失败（如模型不支持 tools 返回 400），降级到规则引擎
			toolCalls = make([]AIToolCallItem, 0, 6)
			modelSupportsTools = false
		}
	}

	if !modelSupportsTools {
		// 降级模式：使用现有的规则引擎进行关键词匹配诊断
		diagPlan := aiapp.BuildChatDiagnosticsPlan(aiapp.ChatDiagnosticsRequest{
			AssistantMode: turn.conversation.AssistantMode,
			Namespace:     req.Namespace,
			ResourceKind:  req.ResourceKind,
			ResourceName:  req.ResourceName,
			Message:       turn.message,
		})
		if diagPlan.Enabled {
			diagCtx := ctx
			cancel := func() {}
			if diagPlan.Timeout > 0 {
				diagCtx, cancel = context.WithTimeout(ctx, diagPlan.Timeout)
			}
			defer cancel()

			var diagErr error
			toolCalls, diagnosticNotes, diagErr = s.toolSvc.RunAutoDiagnostics(diagCtx, AIToolContextRequest{
				ConversationID: turn.conversation.ID,
				MessageID:      turn.userMessage.ID,
				ClusterID:      req.ClusterID,
				UserID:         userID,
				Username:       username,
				UserPerms:      req.UserPerms,
				Query:          turn.message,
				Namespace:      strings.TrimSpace(req.Namespace),
				ResourceKind:   strings.TrimSpace(req.ResourceKind),
				ResourceName:   strings.TrimSpace(req.ResourceName),
			})
			if diagErr != nil && !diagPlan.Optional {
				s.finishConversationRun(turn.conversation.ID, aiConversationRunResult{
					Status:            aiConversationRunStatusFromError(diagErr),
					Summary:           firstUserFacingError(diagErr),
					AssistantContent:  buildAIAssistantFailureReply(diagErr),
					AssistantStatus:   aiMessageStatusFromError(diagErr),
					StructuredPayload: turn.assistantMessageStructured,
					ToolCallCount:     len(toolCalls),
					ProviderID:        turn.selectedProviderID,
					ModelID:           turn.selectedModelID,
				})
				runCompleted = true
				return AIChatResponse{}, diagErr
			}
		}
		diagnosticSummary = buildAIDiagnosticEvidenceSummary(toolCalls)
		diagnosticNotes = buildAIDiagnosticEvidenceDigest(toolCalls, diagnosticNotes)
		gatewayMode := effectiveAIGatewayMode(turn.conversation.AssistantMode, toolCalls, diagnosticNotes)

		history, err := s.buildGatewayMessages(ctx, turn.conversation.ID, aiMessageRequestScope{
			AssistantMode: turn.conversation.AssistantMode,
			Namespace:     strings.TrimSpace(req.Namespace),
			ResourceKind:  strings.TrimSpace(req.ResourceKind),
			ResourceName:  strings.TrimSpace(req.ResourceName),
		}, turn.userMessage.ID)
		if err != nil {
			return AIChatResponse{}, err
		}
		history = append(history, AIGatewayMessage{
			Role:    "user",
			Content: turn.message,
		})
		assistantResp, err = s.gateway.Invoke(ctx, AIGatewayRequest{
			ConversationID:    turn.conversation.ID,
			AssistantMode:     gatewayMode,
			ProviderID:        turn.selectedProviderID,
			ModelID:           turn.selectedModelID,
			PreferModelCode:   req.PreferModel,
			Messages:          history,
			CurrentImages:     turn.gatewayImages,
			CurrentFiles:      turn.fileContexts,
			DiagnosticSummary: diagnosticSummary,
			DiagnosticNotes:   diagnosticNotes,
			ScopeNote:         aigateway.BuildScopeNote(req.Namespace, req.ResourceKind, req.ResourceName),
		})
		if err != nil {
			s.finishConversationRun(turn.conversation.ID, aiConversationRunResult{
				Status:            aiConversationRunStatusFromError(err),
				Summary:           firstUserFacingError(err),
				AssistantContent:  buildAIAssistantFailureReply(err),
				AssistantStatus:   aiMessageStatusFromError(err),
				StructuredPayload: turn.assistantMessageStructured,
				ToolCallCount:     len(toolCalls),
				ProviderID:        turn.selectedProviderID,
				ModelID:           turn.selectedModelID,
			})
			runCompleted = true
			return AIChatResponse{}, err
		}
	}

	assistantContent, assistantStructured := aiapp.NormalizeModelAnswer(assistantResp.Content)
	assistantStructured = aiapp.MergeStructuredPayload(assistantStructured, model.JSONMap(turn.assistantMessageStructured))
	assistantResp.Content = assistantContent

	assistantMessage := model.AIMessage{
		ConversationID: turn.conversation.ID,
		Role:           "assistant",
		MessageType:    "text",
		Content:        assistantContent,
		Status:         "created",
		StructuredJSON: model.JSONMap(assistantStructured),
		ToolCallCount:  len(toolCalls),
		TokenInput:     assistantResp.Usage.RequestTokens,
		TokenOutput:    assistantResp.Usage.ResponseTokens,
	}
	if err := s.db.WithContext(ctx).Create(&assistantMessage).Error; err != nil {
		return AIChatResponse{}, err
	}

	usage := model.AIUsageRecord{
		ConversationID: ptrUint64(turn.conversation.ID),
		MessageID:      ptrUint64(assistantMessage.ID),
		ProviderID:     ptrUint64(assistantResp.ProviderID),
		ModelID:        ptrUint64(assistantResp.ModelID),
		UsageType:      "chat",
		RequestTokens:  assistantResp.Usage.RequestTokens,
		ResponseTokens: assistantResp.Usage.ResponseTokens,
		TotalTokens:    assistantResp.Usage.TotalTokens,
		ImageCount:     len(turn.normalizedImages),
		LatencyMS:      assistantResp.Usage.LatencyMS,
	}
	if err := s.db.WithContext(ctx).Create(&usage).Error; err != nil {
		return AIChatResponse{}, err
	}

	actionProposals := s.autoCreateActionProposals(
		ctx,
		userID,
		username,
		req,
		turn.conversation.ID,
		assistantMessage.ID,
		turn.message,
	)

	summary := buildConversationSummary(assistantContent)
	conversationStatus := "open"
	if len(actionProposals) > 0 {
		conversationStatus = "waiting_confirm"
	}
	if err := s.db.WithContext(ctx).Model(&model.AIConversation{}).
		Where("id = ?", turn.conversation.ID).
		Updates(map[string]any{
			"provider_id":     assistantResp.ProviderID,
			"model_id":        assistantResp.ModelID,
			"summary":         summary,
			"status":          conversationStatus,
			"last_message_at": time.Now().UTC(),
		}).Error; err != nil {
		return AIChatResponse{}, err
	}
	runCompleted = true

	return AIChatResponse{
		ConversationID:     turn.conversation.ID,
		UserMessageID:      turn.userMessage.ID,
		AssistantMessageID: assistantMessage.ID,
		AssistantMessage:   assistantContent,
		ProviderName:       assistantResp.ProviderName,
		ModelName:          assistantResp.ModelName,
		ModelCode:          assistantResp.ModelCode,
		ToolCalls:          toolCalls,
		ActionProposals:    actionProposals,
	}, nil
}

// AIStreamChunk represents a single event in the SSE stream.
type AIStreamChunk struct {
	Type     string `json:"type"`               // "chunk", "done", "error", "progress"
	Content  string `json:"content"`            // delta text for "chunk"
	Error    string `json:"error"`              // error message for "error"
	Progress string `json:"progress,omitempty"` // 进度提示文案 for "progress"
}

// AIStreamDoneData is sent as the final "done" event payload.
type AIStreamDoneData struct {
	ConversationID     uint64                 `json:"conversation_id"`
	UserMessageID      uint64                 `json:"user_message_id"`
	AssistantMessageID uint64                 `json:"assistant_message_id"`
	ProviderName       string                 `json:"provider_name"`
	ModelName          string                 `json:"model_name"`
	ModelCode          string                 `json:"model_code"`
	ToolCalls          []AIToolCallItem       `json:"tool_calls"`
	ActionProposals    []AIActionProposalItem `json:"action_proposals"`
}

// SendChatStream performs the same business logic as SendMessage but streams
// the AI response via the onChunk callback. The callback is guaranteed to be
// called sequentially: zero or more "chunk" events, then exactly one "done" or "error" event.
func (s *AIChatService) SendChatStream(ctx context.Context, userID uint64, username string, req AIChatRequest, onChunk func(AIStreamChunk)) error {
	if s.db == nil || s.gateway == nil || s.toolSvc == nil {
		onChunk(AIStreamChunk{Type: "error", Error: "服务依赖未就绪"})
		return errors.New("ai dependencies are required")
	}
	turn, err := s.startConversationTurn(ctx, userID, username, req)
	if err != nil {
		onChunk(AIStreamChunk{Type: "error", Error: firstUserFacingError(err)})
		return err
	}

	runCompleted := false
	defer func() {
		if !runCompleted {
			s.resetConversationRun(turn.conversation.ID)
		}
	}()

	toolCalls := make([]AIToolCallItem, 0, 6)
	diagnosticNotes := ""
	diagnosticSummary := ""

	// 检查当前模型是否支持 function calling
	modelSupportsTools := s.gateway.ModelSupportsTools(ctx, AIGatewayRequest{
		AssistantMode:   turn.conversation.AssistantMode,
		ProviderID:      turn.selectedProviderID,
		ModelID:         turn.selectedModelID,
		PreferModelCode: req.PreferModel,
	})

	// function calling 模式：先非流式执行工具调用循环，再用流式模式生成最终回答
	if modelSupportsTools {
		onChunk(AIStreamChunk{Type: "progress", Progress: "正在收集集群诊断证据..."})

		history, err := s.buildGatewayMessages(ctx, turn.conversation.ID, aiMessageRequestScope{
			AssistantMode: turn.conversation.AssistantMode,
			Namespace:     strings.TrimSpace(req.Namespace),
			ResourceKind:  strings.TrimSpace(req.ResourceKind),
			ResourceName:  strings.TrimSpace(req.ResourceName),
		}, turn.userMessage.ID)
		if err != nil {
			onChunk(AIStreamChunk{Type: "error", Error: "构建消息历史失败"})
			return err
		}
		history = append(history, AIGatewayMessage{
			Role:    "user",
			Content: turn.message,
		})

		tools := s.buildFunctionCallingTools(req.UserPerms)
		toolReq := AIToolContextRequest{
			ConversationID: turn.conversation.ID,
			MessageID:      turn.userMessage.ID,
			ClusterID:      req.ClusterID,
			UserID:         userID,
			Username:       username,
			UserPerms:      req.UserPerms,
			Query:          turn.message,
			Namespace:      strings.TrimSpace(req.Namespace),
			ResourceKind:   strings.TrimSpace(req.ResourceKind),
			ResourceName:   strings.TrimSpace(req.ResourceName),
		}

		gatewayReq := AIGatewayRequest{
			ConversationID:  turn.conversation.ID,
			AssistantMode:   turn.conversation.AssistantMode,
			ProviderID:      turn.selectedProviderID,
			ModelID:         turn.selectedModelID,
			PreferModelCode: req.PreferModel,
			Messages:        history,
			CurrentImages:   turn.gatewayImages,
			CurrentFiles:    turn.fileContexts,
			ScopeNote:       aigateway.BuildScopeNote(req.Namespace, req.ResourceKind, req.ResourceName),
			Tools:           tools,
		}

		updatedHistory, toolCallItems, assistantResp, err := s.runFunctionCallingLoop(ctx, gatewayReq, toolReq, onChunk)
		toolCalls = toolCallItems
		if err != nil {
			// function calling 失败，降级到规则引擎
			toolCalls = make([]AIToolCallItem, 0, 6)
			modelSupportsTools = false
		}

		if modelSupportsTools {
			// 工具调用循环已生成最终回答，用流式模式重新生成以便前端逐字展示
			streamReader, streamResult, streamErr := s.gateway.InvokeStream(ctx, AIGatewayRequest{
				ConversationID:      turn.conversation.ID,
				AssistantMode:       turn.conversation.AssistantMode,
				ProviderID:          turn.selectedProviderID,
				ModelID:             turn.selectedModelID,
				PreferModelCode:     req.PreferModel,
				Messages:            updatedHistory,
				CurrentImages:       turn.gatewayImages,
				CurrentFiles:        turn.fileContexts,
				ScopeNote:           aigateway.BuildScopeNote(req.Namespace, req.ResourceKind, req.ResourceName),
				FunctionCallingMode: true,
			})
			if streamErr != nil {
				// 流式调用失败时降级使用非流式结果
				fullContent := assistantResp.Content
				assistantContent, assistantStructured := aiapp.NormalizeModelAnswer(fullContent)
				assistantStructured = aiapp.MergeStructuredPayload(assistantStructured, model.JSONMap(turn.assistantMessageStructured))

				assistantMessage := model.AIMessage{
					ConversationID: turn.conversation.ID,
					Role:           "assistant",
					MessageType:    "text",
					Content:        assistantContent,
					Status:         "created",
					StructuredJSON: model.JSONMap(assistantStructured),
					ToolCallCount:  len(toolCalls),
					TokenInput:     assistantResp.Usage.RequestTokens,
					TokenOutput:    assistantResp.Usage.ResponseTokens,
				}
				if dbErr := s.db.WithContext(ctx).Create(&assistantMessage).Error; dbErr != nil {
					onChunk(AIStreamChunk{Type: "error", Error: "保存助手消息失败"})
					return dbErr
				}

				usage := model.AIUsageRecord{
					ConversationID: ptrUint64(turn.conversation.ID),
					MessageID:      ptrUint64(assistantMessage.ID),
					ProviderID:     ptrUint64(assistantResp.ProviderID),
					ModelID:        ptrUint64(assistantResp.ModelID),
					UsageType:      "chat",
					RequestTokens:  assistantResp.Usage.RequestTokens,
					ResponseTokens: assistantResp.Usage.ResponseTokens,
					TotalTokens:    assistantResp.Usage.TotalTokens,
					ImageCount:     len(turn.normalizedImages),
					LatencyMS:      assistantResp.Usage.LatencyMS,
				}
				_ = s.db.WithContext(ctx).Create(&usage).Error

				actionProposals := s.autoCreateActionProposals(ctx, userID, username, req, turn.conversation.ID, assistantMessage.ID, turn.message)

				summary := buildConversationSummary(assistantContent)
				conversationStatus := "open"
				if len(actionProposals) > 0 {
					conversationStatus = "waiting_confirm"
				}
				_ = s.db.WithContext(ctx).Model(&model.AIConversation{}).
					Where("id = ?", turn.conversation.ID).
					Updates(map[string]any{
						"provider_id":     assistantResp.ProviderID,
						"model_id":        assistantResp.ModelID,
						"summary":         summary,
						"status":          conversationStatus,
						"last_message_at": time.Now().UTC(),
					})
				runCompleted = true

				onChunk(AIStreamChunk{Type: "chunk", Content: assistantContent})
				onChunk(AIStreamChunk{
					Type: "done",
					Content: mustMarshalJSON(AIStreamDoneData{
						ConversationID:     turn.conversation.ID,
						UserMessageID:      turn.userMessage.ID,
						AssistantMessageID: assistantMessage.ID,
						ProviderName:       assistantResp.ProviderName,
						ModelName:          assistantResp.ModelName,
						ModelCode:          assistantResp.ModelCode,
						ToolCalls:          toolCalls,
						ActionProposals:    actionProposals,
					}),
				})
				return nil
			}

			// 成功建立流式连接，走正常的流式读取逻辑
			// 复用下方的流式读取代码
			return s.streamFunctionCallingResponse(ctx, turn, req, userID, username, toolCalls, streamReader, streamResult, onChunk, &runCompleted)
		} // end if modelSupportsTools
	}

	if !modelSupportsTools {
		// 降级模式：使用规则引擎进行诊断
		diagPlan := aiapp.BuildChatDiagnosticsPlan(aiapp.ChatDiagnosticsRequest{
			AssistantMode: turn.conversation.AssistantMode,
			Namespace:     req.Namespace,
			ResourceKind:  req.ResourceKind,
			ResourceName:  req.ResourceName,
			Message:       turn.message,
		})
		if diagPlan.Enabled {
			diagCtx := ctx
			cancel := func() {}
			if diagPlan.Timeout > 0 {
				diagCtx, cancel = context.WithTimeout(ctx, diagPlan.Timeout)
			}
			defer cancel()

			// 诊断前推送进度，提示用户正在收集集群证据
			onChunk(AIStreamChunk{Type: "progress", Progress: "正在收集集群诊断证据..."})

			var diagErr error
			toolCalls, diagnosticNotes, diagErr = s.toolSvc.RunAutoDiagnostics(diagCtx, AIToolContextRequest{
				ConversationID: turn.conversation.ID,
				MessageID:      turn.userMessage.ID,
				ClusterID:      req.ClusterID,
				UserID:         userID,
				Username:       username,
				UserPerms:      req.UserPerms,
				Query:          turn.message,
				Namespace:      strings.TrimSpace(req.Namespace),
				ResourceKind:   strings.TrimSpace(req.ResourceKind),
				ResourceName:   strings.TrimSpace(req.ResourceName),
			})
			if diagErr != nil && !diagPlan.Optional {
				s.finishConversationRun(turn.conversation.ID, aiConversationRunResult{
					Status:            aiConversationRunStatusFromError(diagErr),
					Summary:           firstUserFacingError(diagErr),
					AssistantContent:  buildAIAssistantFailureReply(diagErr),
					AssistantStatus:   aiMessageStatusFromError(diagErr),
					StructuredPayload: turn.assistantMessageStructured,
					ToolCallCount:     len(toolCalls),
					ProviderID:        turn.selectedProviderID,
					ModelID:           turn.selectedModelID,
				})
				runCompleted = true
				onChunk(AIStreamChunk{Type: "error", Error: firstUserFacingError(diagErr)})
				return diagErr
			}
		}
		diagnosticSummary = buildAIDiagnosticEvidenceSummary(toolCalls)
		diagnosticNotes = buildAIDiagnosticEvidenceDigest(toolCalls, diagnosticNotes)
		gatewayMode := effectiveAIGatewayMode(turn.conversation.AssistantMode, toolCalls, diagnosticNotes)

		history, err := s.buildGatewayMessages(ctx, turn.conversation.ID, aiMessageRequestScope{
			AssistantMode: turn.conversation.AssistantMode,
			Namespace:     strings.TrimSpace(req.Namespace),
			ResourceKind:  strings.TrimSpace(req.ResourceKind),
			ResourceName:  strings.TrimSpace(req.ResourceName),
		}, turn.userMessage.ID)
		if err != nil {
			onChunk(AIStreamChunk{Type: "error", Error: "构建消息历史失败"})
			return err
		}

		history = append(history, AIGatewayMessage{
			Role:    "user",
			Content: turn.message,
		})

		streamReader, streamResult, err := s.gateway.InvokeStream(ctx, AIGatewayRequest{
			ConversationID:    turn.conversation.ID,
			AssistantMode:     gatewayMode,
			ProviderID:        turn.selectedProviderID,
			ModelID:           turn.selectedModelID,
			PreferModelCode:   req.PreferModel,
			Messages:          history,
			CurrentImages:     turn.gatewayImages,
			CurrentFiles:      turn.fileContexts,
			DiagnosticSummary: diagnosticSummary,
			DiagnosticNotes:   diagnosticNotes,
			ScopeNote:         aigateway.BuildScopeNote(req.Namespace, req.ResourceKind, req.ResourceName),
		})
		if err != nil {
			s.finishConversationRun(turn.conversation.ID, aiConversationRunResult{
				Status:            aiConversationRunStatusFromError(err),
				Summary:           firstUserFacingError(err),
				AssistantContent:  buildAIAssistantFailureReply(err),
				AssistantStatus:   aiMessageStatusFromError(err),
				StructuredPayload: turn.assistantMessageStructured,
				ToolCallCount:     len(toolCalls),
				ProviderID:        turn.selectedProviderID,
				ModelID:           turn.selectedModelID,
			})
			runCompleted = true
			onChunk(AIStreamChunk{Type: "error", Error: firstUserFacingError(err)})
			return err
		}
		defer func() { _ = streamReader.Close() }()

		// Read SSE lines, accumulate content, forward deltas
		var fullContent strings.Builder
		reqStart := time.Now()
		scanner := bufio.NewScanner(streamReader)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			delta := chunk.Choices[0].Delta.Content
			if delta == "" {
				continue
			}
			fullContent.WriteString(delta)
			onChunk(AIStreamChunk{Type: "chunk", Content: delta})

			if chunk.Usage != nil {
				streamResult.Usage.RequestTokens = chunk.Usage.PromptTokens
				streamResult.Usage.ResponseTokens = chunk.Usage.CompletionTokens
				streamResult.Usage.TotalTokens = chunk.Usage.TotalTokens
			}
		}

		if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
			s.finishConversationRun(turn.conversation.ID, aiConversationRunResult{
				Status:            "failed",
				Summary:           "流式读取中断",
				AssistantContent:  fullContent.String(),
				AssistantStatus:   "failed",
				StructuredPayload: turn.assistantMessageStructured,
				ToolCallCount:     len(toolCalls),
				ProviderID:        turn.selectedProviderID,
				ModelID:           turn.selectedModelID,
			})
			runCompleted = true
			onChunk(AIStreamChunk{Type: "error", Error: "流式读取中断"})
			return err
		}

		latencyMS := int(time.Since(reqStart).Milliseconds())
		streamResult.Usage.LatencyMS = latencyMS
		if streamResult.Usage.TotalTokens == 0 {
			streamResult.Usage.TotalTokens = streamResult.Usage.RequestTokens + streamResult.Usage.ResponseTokens
		}

		assistantContent, assistantStructured := aiapp.NormalizeModelAnswer(fullContent.String())
		assistantStructured = aiapp.MergeStructuredPayload(assistantStructured, model.JSONMap(turn.assistantMessageStructured))

		assistantMessage := model.AIMessage{
			ConversationID: turn.conversation.ID,
			Role:           "assistant",
			MessageType:    "text",
			Content:        assistantContent,
			Status:         "created",
			StructuredJSON: model.JSONMap(assistantStructured),
			ToolCallCount:  len(toolCalls),
			TokenInput:     streamResult.Usage.RequestTokens,
			TokenOutput:    streamResult.Usage.ResponseTokens,
		}
		if err := s.db.WithContext(ctx).Create(&assistantMessage).Error; err != nil {
			onChunk(AIStreamChunk{Type: "error", Error: "保存助手消息失败"})
			return err
		}

		usage := model.AIUsageRecord{
			ConversationID: ptrUint64(turn.conversation.ID),
			MessageID:      ptrUint64(assistantMessage.ID),
			ProviderID:     ptrUint64(streamResult.ProviderID),
			ModelID:        ptrUint64(streamResult.ModelID),
			UsageType:      "chat",
			RequestTokens:  streamResult.Usage.RequestTokens,
			ResponseTokens: streamResult.Usage.ResponseTokens,
			TotalTokens:    streamResult.Usage.TotalTokens,
			ImageCount:     len(turn.normalizedImages),
			LatencyMS:      latencyMS,
		}
		if err := s.db.WithContext(ctx).Create(&usage).Error; err != nil {
			// Non-fatal: log but don't fail the response
			_ = err
		}

		actionProposals := s.autoCreateActionProposals(ctx, userID, username, req, turn.conversation.ID, assistantMessage.ID, turn.message)

		summary := buildConversationSummary(assistantContent)
		conversationStatus := "open"
		if len(actionProposals) > 0 {
			conversationStatus = "waiting_confirm"
		}
		_ = s.db.WithContext(ctx).Model(&model.AIConversation{}).
			Where("id = ?", turn.conversation.ID).
			Updates(map[string]any{
				"provider_id":     streamResult.ProviderID,
				"model_id":        streamResult.ModelID,
				"summary":         summary,
				"status":          conversationStatus,
				"last_message_at": time.Now().UTC(),
			})
		runCompleted = true

		onChunk(AIStreamChunk{
			Type: "done",
			Content: mustMarshalJSON(AIStreamDoneData{
				ConversationID:     turn.conversation.ID,
				UserMessageID:      turn.userMessage.ID,
				AssistantMessageID: assistantMessage.ID,
				ProviderName:       streamResult.ProviderName,
				ModelName:          streamResult.ModelName,
				ModelCode:          streamResult.ModelCode,
				ToolCalls:          toolCalls,
				ActionProposals:    actionProposals,
			}),
		})
		return nil
	} // end if !modelSupportsTools
	return nil
}

func mustMarshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// streamFunctionCallingResponse 处理 function calling 模式下的流式 SSE 读取和消息持久化
// 复用与降级模式相同的 SSE 读取逻辑
func (s *AIChatService) streamFunctionCallingResponse(
	ctx context.Context,
	turn aiPreparedConversationTurn,
	req AIChatRequest,
	userID uint64,
	username string,
	toolCalls []AIToolCallItem,
	streamReader io.ReadCloser,
	streamResult AIGatewayStreamResult,
	onChunk func(AIStreamChunk),
	runCompleted *bool,
) error {
	defer func() { _ = streamReader.Close() }()

	var fullContent strings.Builder
	reqStart := time.Now()
	scanner := bufio.NewScanner(streamReader)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		fullContent.WriteString(delta)
		onChunk(AIStreamChunk{Type: "chunk", Content: delta})

		if chunk.Usage != nil {
			streamResult.Usage.RequestTokens = chunk.Usage.PromptTokens
			streamResult.Usage.ResponseTokens = chunk.Usage.CompletionTokens
			streamResult.Usage.TotalTokens = chunk.Usage.TotalTokens
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		s.finishConversationRun(turn.conversation.ID, aiConversationRunResult{
			Status:            "failed",
			Summary:           "流式读取中断",
			AssistantContent:  fullContent.String(),
			AssistantStatus:   "failed",
			StructuredPayload: turn.assistantMessageStructured,
			ToolCallCount:     len(toolCalls),
			ProviderID:        turn.selectedProviderID,
			ModelID:           turn.selectedModelID,
		})
		*runCompleted = true
		onChunk(AIStreamChunk{Type: "error", Error: "流式读取中断"})
		return err
	}

	latencyMS := int(time.Since(reqStart).Milliseconds())
	streamResult.Usage.LatencyMS = latencyMS
	if streamResult.Usage.TotalTokens == 0 {
		streamResult.Usage.TotalTokens = streamResult.Usage.RequestTokens + streamResult.Usage.ResponseTokens
	}

	assistantContent, assistantStructured := aiapp.NormalizeModelAnswer(fullContent.String())
	assistantStructured = aiapp.MergeStructuredPayload(assistantStructured, model.JSONMap(turn.assistantMessageStructured))

	assistantMessage := model.AIMessage{
		ConversationID: turn.conversation.ID,
		Role:           "assistant",
		MessageType:    "text",
		Content:        assistantContent,
		Status:         "created",
		StructuredJSON: model.JSONMap(assistantStructured),
		ToolCallCount:  len(toolCalls),
		TokenInput:     streamResult.Usage.RequestTokens,
		TokenOutput:    streamResult.Usage.ResponseTokens,
	}
	if err := s.db.WithContext(ctx).Create(&assistantMessage).Error; err != nil {
		onChunk(AIStreamChunk{Type: "error", Error: "保存助手消息失败"})
		return err
	}

	usage := model.AIUsageRecord{
		ConversationID: ptrUint64(turn.conversation.ID),
		MessageID:      ptrUint64(assistantMessage.ID),
		ProviderID:     ptrUint64(streamResult.ProviderID),
		ModelID:        ptrUint64(streamResult.ModelID),
		UsageType:      "chat",
		RequestTokens:  streamResult.Usage.RequestTokens,
		ResponseTokens: streamResult.Usage.ResponseTokens,
		TotalTokens:    streamResult.Usage.TotalTokens,
		ImageCount:     len(turn.normalizedImages),
		LatencyMS:      latencyMS,
	}
	_ = s.db.WithContext(ctx).Create(&usage).Error

	actionProposals := s.autoCreateActionProposals(ctx, userID, username, req, turn.conversation.ID, assistantMessage.ID, turn.message)

	summary := buildConversationSummary(assistantContent)
	conversationStatus := "open"
	if len(actionProposals) > 0 {
		conversationStatus = "waiting_confirm"
	}
	_ = s.db.WithContext(ctx).Model(&model.AIConversation{}).
		Where("id = ?", turn.conversation.ID).
		Updates(map[string]any{
			"provider_id":     streamResult.ProviderID,
			"model_id":        streamResult.ModelID,
			"summary":         summary,
			"status":          conversationStatus,
			"last_message_at": time.Now().UTC(),
		})
	*runCompleted = true

	onChunk(AIStreamChunk{
		Type: "done",
		Content: mustMarshalJSON(AIStreamDoneData{
			ConversationID:     turn.conversation.ID,
			UserMessageID:      turn.userMessage.ID,
			AssistantMessageID: assistantMessage.ID,
			ProviderName:       streamResult.ProviderName,
			ModelName:          streamResult.ModelName,
			ModelCode:          streamResult.ModelCode,
			ToolCalls:          toolCalls,
			ActionProposals:    actionProposals,
		}),
	})
	return nil
}

type aiConversationRunResult struct {
	Status            string
	Summary           string
	AssistantContent  string
	AssistantStatus   string
	StructuredPayload model.JSONMap
	ToolCallCount     int
	ProviderID        *uint64
	ModelID           *uint64
}

func (s *AIChatService) beginConversationRun(ctx context.Context, conversationID uint64, lastMessageAt time.Time) error {
	if s == nil || s.db == nil {
		return errors.New("db is required")
	}
	if conversationID == 0 {
		return ErrWithMessage(ErrInvalidParams, "会话 ID 无效")
	}
	result := s.db.WithContext(ctx).
		Model(&model.AIConversation{}).
		Where("deleted_at IS NULL AND id = ? AND status <> ?", conversationID, "running").
		Updates(map[string]any{
			"status":          "running",
			"last_message_at": lastMessageAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	var row model.AIConversation
	if err := s.db.WithContext(ctx).
		Select("id", "status").
		Where("deleted_at IS NULL AND id = ?", conversationID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(row.Status), "running") {
		return ErrWithMessage(ErrConflict, "当前会话仍在处理中，请先等待完成或取消当前回答")
	}
	return ErrConflict
}

func resolveAIInvocationTarget(conversation model.AIConversation, req AIChatRequest) (*uint64, *uint64) {
	providerID := conversation.ProviderID
	if req.ProviderID != nil && *req.ProviderID > 0 {
		providerID = req.ProviderID
	}
	modelID := conversation.ModelID
	if req.ModelID != nil && *req.ModelID > 0 {
		modelID = req.ModelID
	}
	return providerID, modelID
}

func buildAIMessageScopeSnapshot(
	conversation model.AIConversation,
	req AIChatRequest,
	providerID,
	modelID *uint64,
	images []model.JSONMap,
	attachments []AIMessageAttachmentItem,
) model.JSONMap {
	scope := model.JSONMap{
		"request_scope": model.JSONMap{
			"cluster_id":      req.ClusterID,
			"conversation_id": conversation.ID,
			"assistant_mode":  conversation.AssistantMode,
			"namespace":       strings.TrimSpace(req.Namespace),
			"resource_kind":   strings.TrimSpace(req.ResourceKind),
			"resource_name":   strings.TrimSpace(req.ResourceName),
			"prefer_model":    strings.TrimSpace(req.PreferModel),
		},
	}
	requestScope, _ := scope["request_scope"].(model.JSONMap)
	if providerID != nil && *providerID > 0 {
		requestScope["provider_id"] = *providerID
	}
	if modelID != nil && *modelID > 0 {
		requestScope["model_id"] = *modelID
	}
	if len(images) > 0 {
		requestScope["image_count"] = len(images)
		scope["request_images"] = images
	}
	if len(attachments) > 0 {
		requestScope["attachment_count"] = len(attachments)
		scope["request_attachments"] = buildAIMessageAttachmentRefs(attachments)
	}
	return scope
}

func normalizeAIChatImages(images []AIChatImageInput) []model.JSONMap {
	if len(images) == 0 {
		return nil
	}

	out := make([]model.JSONMap, 0, minInt(len(images), 6))
	for _, item := range images {
		if len(out) >= 6 {
			break
		}
		dataURL := strings.TrimSpace(item.DataURL)
		contentType := strings.TrimSpace(item.ContentType)
		name := strings.TrimSpace(item.Name)
		if dataURL == "" || !strings.HasPrefix(strings.ToLower(dataURL), "data:image/") {
			continue
		}
		if contentType == "" {
			contentType = inferAIChatImageContentType(dataURL)
		}
		if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
			continue
		}
		if len(dataURL) > 8*1024*1024 {
			continue
		}
		if name == "" {
			name = "image"
		}
		out = append(out, model.JSONMap{
			"name":         name,
			"content_type": contentType,
			"data_url":     dataURL,
			"size":         maxInt64(item.Size, 0),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func inferAIChatImageContentType(dataURL string) string {
	trimmed := strings.TrimSpace(dataURL)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return ""
	}
	trimmed = strings.TrimPrefix(trimmed, "data:")
	parts := strings.SplitN(trimmed, ";", 2)
	return strings.TrimSpace(parts[0])
}

func maxInt64(value, floor int64) int64 {
	if value < floor {
		return floor
	}
	return value
}

func aiConversationRunStatusFromError(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "failed"
	default:
		return "failed"
	}
}

func aiMessageStatusFromError(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "failed"
	default:
		return "failed"
	}
}

func buildAIAssistantFailureReply(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "本轮回答已取消，平台已停止继续生成结果。你可以调整问题或范围后重新发送。"
	case errors.Is(err, context.DeadlineExceeded):
		return "本轮回答处理超时，AI 未能在限定时间内完成。建议缩小排查范围后重试。"
	default:
		message := firstUserFacingError(err)
		if strings.TrimSpace(message) == "" {
			message = "本轮回答失败，请稍后重试。"
		}
		return "本轮回答未成功完成。原因：" + message
	}
}

func (s *AIChatService) finishConversationRun(conversationID uint64, result aiConversationRunResult) {
	if s == nil || s.db == nil || conversationID == 0 {
		return
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	updates := map[string]any{
		"status":          strings.TrimSpace(result.Status),
		"summary":         strings.TrimSpace(result.Summary),
		"last_message_at": time.Now().UTC(),
	}
	if result.ProviderID != nil && *result.ProviderID > 0 {
		updates["provider_id"] = *result.ProviderID
	}
	if result.ModelID != nil && *result.ModelID > 0 {
		updates["model_id"] = *result.ModelID
	}
	_ = s.db.WithContext(bgCtx).
		Model(&model.AIConversation{}).
		Where("id = ?", conversationID).
		Updates(updates).Error

	content := strings.TrimSpace(result.AssistantContent)
	if content == "" {
		return
	}
	messageStatus := strings.TrimSpace(result.AssistantStatus)
	if messageStatus == "" {
		messageStatus = "failed"
	}
	_ = s.db.WithContext(bgCtx).Create(&model.AIMessage{
		ConversationID: conversationID,
		Role:           "assistant",
		MessageType:    "text",
		Content:        content,
		Status:         messageStatus,
		StructuredJSON: model.JSONMap(result.StructuredPayload),
		ToolCallCount:  result.ToolCallCount,
	}).Error
}

func (s *AIChatService) resetConversationRun(conversationID uint64) {
	if s == nil || s.db == nil || conversationID == 0 {
		return
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_ = s.db.WithContext(bgCtx).
		Model(&model.AIConversation{}).
		Where("id = ? AND status = ?", conversationID, "running").
		Updates(map[string]any{
			"status":          "open",
			"last_message_at": time.Now().UTC(),
		}).Error
}

func (s *AIChatService) ensureConversation(ctx context.Context, userID uint64, username string, req AIChatRequest) (model.AIConversation, error) {
	if req.ConversationID != nil && *req.ConversationID > 0 {
		var conversation model.AIConversation
		if err := s.db.WithContext(ctx).
			Where("deleted_at IS NULL AND id = ? AND cluster_id = ?", *req.ConversationID, req.ClusterID).
			First(&conversation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.AIConversation{}, ErrNotFound
			}
			return model.AIConversation{}, err
		}
		return conversation, nil
	}

	mode := aiapp.NormalizeAssistantMode(req.AssistantMode)
	if mode == "" {
		mode = "diagnose"
	}
	title := aiapp.BuildConversationTitle(req.Message)
	conversation := model.AIConversation{
		ClusterID:     req.ClusterID,
		ProviderID:    req.ProviderID,
		ModelID:       req.ModelID,
		Title:         title,
		Status:        "open",
		AssistantMode: mode,
		CreatedBy:     userID,
		CreatedByName: strings.TrimSpace(username),
	}
	if err := s.db.WithContext(ctx).Create(&conversation).Error; err != nil {
		return model.AIConversation{}, err
	}
	return conversation, nil
}

func (s *AIChatService) buildGatewayMessages(
	ctx context.Context,
	conversationID uint64,
	currentScope aiMessageRequestScope,
	excludeMessageID uint64,
) ([]AIGatewayMessage, error) {
	var rows []model.AIMessage
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC, id DESC").
		Limit(20).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for left, right := 0, len(rows)-1; left < right; left, right = left+1, right-1 {
		rows[left], rows[right] = rows[right], rows[left]
	}

	filteredRows := buildScopedGatewayHistory(rows, currentScope)
	messages := make([]AIGatewayMessage, 0, len(filteredRows))
	for _, row := range filteredRows {
		if excludeMessageID > 0 && row.ID == excludeMessageID {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(row.Role))
		if role == "" {
			role = "user"
		}
		messages = append(messages, AIGatewayMessage{
			Role:    role,
			Content: row.Content,
		})
	}
	return messages, nil
}

func buildScopedGatewayHistory(rows []model.AIMessage, currentScope aiMessageRequestScope) []model.AIMessage {
	if len(rows) == 0 {
		return nil
	}
	if !currentScope.hasExplicitScope() {
		return append([]model.AIMessage(nil), rows...)
	}

	filteredReversed := make([]model.AIMessage, 0, len(rows))
	seenScopedCurrent := false
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		rowScope := extractAIMessageRequestScope(model.JSONMap(row.StructuredJSON))
		if rowScope.hasExplicitScope() {
			if !currentScope.matches(rowScope) {
				break
			}
			seenScopedCurrent = true
			filteredReversed = append(filteredReversed, row)
			continue
		}
		if seenScopedCurrent && strings.EqualFold(strings.TrimSpace(row.Role), "system") {
			filteredReversed = append(filteredReversed, row)
		}
	}

	if len(filteredReversed) == 0 {
		return append([]model.AIMessage(nil), rows[len(rows)-1])
	}

	filtered := make([]model.AIMessage, 0, len(filteredReversed))
	for i := len(filteredReversed) - 1; i >= 0; i-- {
		filtered = append(filtered, filteredReversed[i])
	}
	return filtered
}

func extractAIMessageRequestScope(structured model.JSONMap) aiMessageRequestScope {
	if len(structured) == 0 {
		return aiMessageRequestScope{}
	}
	raw, _ := structured["request_scope"].(model.JSONMap)
	if raw == nil {
		if mapValue, ok := structured["request_scope"].(map[string]any); ok {
			raw = model.JSONMap(mapValue)
		}
	}
	if raw == nil {
		return aiMessageRequestScope{}
	}
	return aiMessageRequestScope{
		AssistantMode: strings.TrimSpace(fmt.Sprint(raw["assistant_mode"])),
		Namespace:     strings.TrimSpace(fmt.Sprint(raw["namespace"])),
		ResourceKind:  strings.TrimSpace(fmt.Sprint(raw["resource_kind"])),
		ResourceName:  strings.TrimSpace(fmt.Sprint(raw["resource_name"])),
	}
}

func (s aiMessageRequestScope) hasExplicitScope() bool {
	return strings.TrimSpace(s.Namespace) != "" || strings.TrimSpace(s.ResourceKind) != "" || strings.TrimSpace(s.ResourceName) != ""
}

func (s aiMessageRequestScope) matches(other aiMessageRequestScope) bool {
	return strings.EqualFold(strings.TrimSpace(s.AssistantMode), strings.TrimSpace(other.AssistantMode)) &&
		strings.EqualFold(strings.TrimSpace(s.Namespace), strings.TrimSpace(other.Namespace)) &&
		strings.EqualFold(strings.TrimSpace(s.ResourceKind), strings.TrimSpace(other.ResourceKind)) &&
		strings.EqualFold(strings.TrimSpace(s.ResourceName), strings.TrimSpace(other.ResourceName))
}

func buildConversationSummary(content string) string {
	return aiapp.BuildConversationTitle(content)
}

func (s *AIChatService) autoCreateActionProposals(
	ctx context.Context,
	userID uint64,
	username string,
	req AIChatRequest,
	conversationID uint64,
	messageID uint64,
	message string,
) []AIActionProposalItem {
	if s.actionSvc == nil {
		return nil
	}

	specs := aiapp.BuildAutoActionProposalSpecs(aiapp.ChatActionIntentRequest{
		ResourceKind: req.ResourceKind,
		Namespace:    req.Namespace,
		ResourceName: req.ResourceName,
	}, message)
	if len(specs) == 0 {
		return nil
	}

	items := make([]AIActionProposalItem, 0, len(specs))
	for _, spec := range specs {
		if s.hasOpenProposal(ctx, conversationID, req.ClusterID, spec) {
			continue
		}
		result, err := s.actionSvc.CreateProposal(ctx, req.ClusterID, userID, username, CreateAIActionProposalRequest{
			ConversationID: conversationID,
			MessageID:      ptrUint64(messageID),
			ProposalType:   spec.ProposalType,
			TargetResource: spec.TargetResource,
			Payload:        spec.Payload,
			Reason:         spec.Reason,
		})
		if err != nil {
			continue
		}
		items = append(items, result.Proposal)
	}

	return items
}

func (s *AIChatService) hasOpenProposal(
	ctx context.Context,
	conversationID uint64,
	clusterID uint64,
	spec aiapp.AutoActionProposalSpec,
) bool {
	if s.db == nil {
		return false
	}

	var rows []model.AIActionProposal
	if err := s.db.WithContext(ctx).
		Where(
			`conversation_id = ? AND cluster_id = ? AND action_type = ? AND target_kind = ? AND target_namespace = ? AND target_name = ? AND status IN ?`,
			conversationID,
			clusterID,
			spec.ProposalType,
			spec.TargetResource.Kind,
			spec.TargetResource.Namespace,
			spec.TargetResource.Name,
			[]string{"pending_confirm", "approved", "executing"},
		).
		Find(&rows).Error; err != nil {
		return false
	}
	if len(rows) == 0 {
		return false
	}
	if spec.ProposalType != aiActionTypeScaleWorkload {
		return true
	}

	targetReplicas, ok := jsonIntValue(spec.Payload["replicas"])
	if !ok {
		return false
	}
	for _, row := range rows {
		if replicas, ok := jsonIntValue(aiActionPayload(model.JSONMap(row.ChangeJSON))["replicas"]); ok && replicas == targetReplicas {
			return true
		}
		if replicas, ok := jsonIntValue(row.ChangeJSON["target_replicas"]); ok && replicas == targetReplicas {
			return true
		}
	}
	return false
}

func hasTentativeActionIntent(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if strings.Contains(lower, "?") || strings.Contains(message, "\uff1f") {
		return true
	}
	return containsAny(
		lower,
		"should restart",
		"should scale",
		"can we restart",
		"can we scale",
		"could we restart",
		"could we scale",
		"need to restart",
		"need to scale",
	) || containsAny(
		message,
		"\u662f\u5426\u91cd\u542f",
		"\u8981\u4e0d\u8981\u91cd\u542f",
		"\u662f\u4e0d\u662f\u8981\u91cd\u542f",
		"\u9700\u4e0d\u9700\u8981\u91cd\u542f",
		"\u662f\u5426\u6269\u5bb9",
		"\u662f\u5426\u7f29\u5bb9",
		"\u8981\u4e0d\u8981\u6269\u5bb9",
		"\u8981\u4e0d\u8981\u7f29\u5bb9",
		"\u9700\u4e0d\u9700\u8981\u6269\u5bb9",
		"\u9700\u4e0d\u9700\u8981\u7f29\u5bb9",
		"\u53ef\u4ee5\u91cd\u542f\u5417",
		"\u53ef\u4ee5\u6269\u5bb9\u5417",
		"\u53ef\u4ee5\u7f29\u5bb9\u5417",
	)
}

// isExplicitDeleteIntent 判断是否为删除工作负载的明确意图
func isExplicitDeleteIntent(msg string) bool {
	lower := strings.ToLower(msg)
	keywords := []string{"delete", "remove", "删除", "移除"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			// 排除试探性表述
			if hasTentativeActionIntent(msg) {
				return false
			}
			return true
		}
	}
	return false
}

// isExplicitDeletePodIntent 判断是否为删除 Pod 的明确意图
func isExplicitDeletePodIntent(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "delete") || strings.Contains(lower, "删除") || strings.Contains(lower, "移除")
}

// isExplicitCordonIntent 判断是否为节点封锁意图
func isExplicitCordonIntent(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "cordon") || strings.Contains(lower, "封锁") || strings.Contains(lower, "停止调度")
}

// isExplicitDrainIntent 判断是否为节点驱逐意图
func isExplicitDrainIntent(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "drain") || strings.Contains(lower, "驱逐") || strings.Contains(lower, "排空")
}

// isExplicitUpdateImageIntent 判断是否为更新镜像意图
func isExplicitUpdateImageIntent(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "update image") || strings.Contains(lower, "change image") ||
		strings.Contains(lower, "更新镜像") || strings.Contains(lower, "修改镜像") ||
		strings.Contains(lower, "rollout image")
}

// detectImageFromMessage 从消息中提取镜像名称（匹配 image:xxx 格式）
func detectImageFromMessage(msg string) string {
	lower := strings.ToLower(msg)
	idx := strings.Index(lower, "image:")
	if idx >= 0 {
		rest := msg[idx+6:]
		// 截取到空白或行尾
		end := strings.IndexAny(rest, " \n\t,，")
		if end > 0 {
			return strings.TrimSpace(rest[:end])
		}
		return strings.TrimSpace(rest)
	}
	return ""
}

func containsAny(text string, candidates ...string) bool {
	for _, candidate := range candidates {
		if candidate != "" && strings.Contains(text, candidate) {
			return true
		}
	}
	return false
}

func buildAIDiagnosticEvidenceSummary(toolCalls []AIToolCallItem) string {
	summaryLines := make([]string, 0, len(toolCalls))
	for _, item := range toolCalls {
		if !strings.EqualFold(strings.TrimSpace(item.Status), "succeeded") {
			continue
		}
		toolName := strings.TrimSpace(item.ToolName)
		resultSummary := strings.TrimSpace(item.ResultSummary)
		if toolName == "" && resultSummary == "" {
			continue
		}
		if resultSummary == "" {
			summaryLines = append(summaryLines, "- "+toolName)
			continue
		}
		if toolName == "" {
			summaryLines = append(summaryLines, "- "+resultSummary)
			continue
		}
		summaryLines = append(summaryLines, "- "+toolName+": "+resultSummary)
	}

	if len(summaryLines) == 0 {
		return ""
	}
	return "Confirmed platform evidence for this round:\n" + strings.Join(summaryLines, "\n")
}

func buildAIDiagnosticEvidenceDigest(toolCalls []AIToolCallItem, diagnosticNotes string) string {
	digest := buildAIDiagnosticEvidenceSummary(toolCalls)
	notes := strings.TrimSpace(diagnosticNotes)
	if digest == "" {
		return notes
	}
	if notes == "" {
		return digest
	}
	return digest + "\n\nDetailed platform evidence:\n" + notes
}

func effectiveAIGatewayMode(mode string, toolCalls []AIToolCallItem, diagnosticNotes string) string {
	normalizedMode := aiapp.NormalizeAssistantMode(mode)
	if normalizedMode == "" {
		normalizedMode = "diagnose"
	}
	if normalizedMode == "chat" && (len(toolCalls) > 0 || strings.TrimSpace(diagnosticNotes) != "") {
		return "diagnose"
	}
	return normalizedMode
}

func ptrUint64(v uint64) *uint64 {
	return &v
}
