package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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

// ChatRuntime coordinates gateway invocation, persisted conversation state,
// tool calls and optional action proposals for the AI HTTP runtime. The AI
// application package owns the request policy and DTOs; this adapter owns the
// concrete gateway and persistence interactions.
type ChatRuntime struct {
	db        *gorm.DB
	gateway   *AIGatewayService
	toolSvc   *AIToolService
	actionSvc *ActionRuntime
	fileSvc   *AIFileService
}

func NewChatRuntime(
	db *gorm.DB,
	gateway *AIGatewayService,
	toolSvc *AIToolService,
	actionSvc *ActionRuntime,
	fileSvc *AIFileService,
) *ChatRuntime {
	return &ChatRuntime{db: db, gateway: gateway, toolSvc: toolSvc, actionSvc: actionSvc, fileSvc: fileSvc}
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

func buildAIGatewayImages(images []aiapp.ChatImage) []AIGatewayImage {
	if len(images) == 0 {
		return nil
	}
	out := make([]AIGatewayImage, 0, len(images))
	for _, image := range images {
		out = append(out, AIGatewayImage{
			Name:        image.Name,
			ContentType: image.ContentType,
			DataURL:     image.DataURL,
		})
	}
	return out
}

func buildAIGatewayFileContexts(files []aiapp.ChatFileContext) []AIGatewayFileContext {
	if len(files) == 0 {
		return nil
	}
	out := make([]AIGatewayFileContext, 0, len(files))
	for _, file := range files {
		out = append(out, AIGatewayFileContext{
			Name:        file.Name,
			ContentType: file.ContentType,
			Content:     file.Content,
			Size:        file.Size,
		})
	}
	return out
}

func (s *ChatRuntime) startConversationTurn(
	ctx context.Context,
	userID uint64,
	username string,
	req aiapp.RuntimeChatRequest,
) (_ aiPreparedConversationTurn, err error) {
	message := strings.TrimSpace(req.Message)
	if req.ClusterID == 0 {
		return aiPreparedConversationTurn{}, aiapp.ErrorWithMessage(aiapp.ErrInvalidParams, "集群 ID 无效")
	}
	if message == "" {
		return aiPreparedConversationTurn{}, aiapp.ErrorWithMessage(aiapp.ErrInvalidParams, "消息内容不能为空")
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

	selectedProviderID, selectedModelID := aiapp.ResolveChatInvocationTarget(aiapp.ChatInvocationTargetInput{
		ConversationProviderID: conversation.ProviderID,
		ConversationModelID:    conversation.ModelID,
		RequestProviderID:      req.ProviderID,
		RequestModelID:         req.ModelID,
	})
	preparedInputs, err := aiapp.PrepareChatInputs(req.Images, req.Uploads)
	if err != nil {
		return aiPreparedConversationTurn{}, err
	}
	uploads := preparedInputs.Uploads
	normalizedImages := aiapp.ChatImageSnapshots(preparedInputs.Images)
	gatewayImages := buildAIGatewayImages(preparedInputs.Images)
	fileContexts := buildAIGatewayFileContexts(preparedInputs.Files)
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
		if err := aiapp.ValidateChatModelInputs(aiModel, preparedInputs.Images, preparedInputs.Files); err != nil {
			return aiPreparedConversationTurn{}, err
		}
	}

	userMessageStructured := aiapp.BuildChatMessageScopeSnapshot(aiapp.ChatMessageScopeSnapshotInput{
		ClusterID:      req.ClusterID,
		ConversationID: conversation.ID,
		Scope:          chatRequestScope(conversation, req),
		PreferModel:    req.PreferModel,
		ProviderID:     selectedProviderID,
		ModelID:        selectedModelID,
		Images:         normalizedImages,
	})
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
		userMessageStructured = aiapp.BuildChatMessageScopeSnapshot(aiapp.ChatMessageScopeSnapshotInput{
			ClusterID:      req.ClusterID,
			ConversationID: conversation.ID,
			Scope:          chatRequestScope(conversation, req),
			PreferModel:    req.PreferModel,
			ProviderID:     selectedProviderID,
			ModelID:        selectedModelID,
			Images:         normalizedImages,
			Attachments:    attachments,
		})
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
		assistantMessageStructured: aiapp.BuildChatMessageScopeSnapshot(aiapp.ChatMessageScopeSnapshotInput{
			ClusterID:      req.ClusterID,
			ConversationID: conversation.ID,
			Scope:          chatRequestScope(conversation, req),
			PreferModel:    req.PreferModel,
			ProviderID:     selectedProviderID,
			ModelID:        selectedModelID,
			Attachments:    attachments,
		}),
	}, nil
}

// buildFunctionCallingTools 从工具注册表构建 LLM function calling 工具列表
// 只暴露只读工具（query/inspect/export），不暴露变更提案工具（proposal）
func (s *ChatRuntime) buildFunctionCallingTools(userPerms []string) []AIToolSpec {
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
		if !aiapp.HasAllToolPermissions(userPerms, def.RequiredPermissions) {
			continue
		}
		tools = append(tools, AIToolSpec{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  aiapp.BuildToolInputJSONSchema(def.InputSchema),
		})
	}
	return tools
}

// runFunctionCallingLoop 执行 LLM function calling 工具调用循环
// gatewayReq 应已设置 Messages 和 Tools；toolReq 提供工具执行上下文
// 返回：更新后的消息历史、工具调用记录、最后一次网关响应
func (s *ChatRuntime) runFunctionCallingLoop(
	ctx context.Context,
	gatewayReq AIGatewayRequest,
	toolReq AIToolContextRequest,
	onChunk func(aiapp.RuntimeStreamChunk),
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
				onChunk(aiapp.RuntimeStreamChunk{Type: "progress", Progress: "正在执行 " + tc.Name + "..."})
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

func (s *ChatRuntime) SendMessage(ctx context.Context, userID uint64, username string, req aiapp.RuntimeChatRequest) (aiapp.RuntimeChatResponse, error) {
	if s.db == nil {
		return aiapp.RuntimeChatResponse{}, errors.New("db is required")
	}
	if s.gateway == nil || s.toolSvc == nil {
		return aiapp.RuntimeChatResponse{}, errors.New("ai dependencies are required")
	}
	turn, err := s.startConversationTurn(ctx, userID, username, req)
	if err != nil {
		return aiapp.RuntimeChatResponse{}, err
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
		history, err := s.buildGatewayMessages(ctx, turn.conversation.ID, chatRequestScope(turn.conversation, req), turn.userMessage.ID)
		if err != nil {
			return aiapp.RuntimeChatResponse{}, err
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
					Status:            aiapp.ConversationRunStatusFromError(diagErr),
					Summary:           aiapp.UserFacingError(diagErr),
					AssistantContent:  aiapp.BuildAssistantFailureReply(diagErr),
					AssistantStatus:   aiapp.MessageStatusFromError(diagErr),
					StructuredPayload: turn.assistantMessageStructured,
					ToolCallCount:     len(toolCalls),
					ProviderID:        turn.selectedProviderID,
					ModelID:           turn.selectedModelID,
				})
				runCompleted = true
				return aiapp.RuntimeChatResponse{}, diagErr
			}
		}
		diagnosticSummary = aiapp.BuildChatDiagnosticEvidenceSummary(toolCalls)
		diagnosticNotes = aiapp.BuildChatDiagnosticEvidenceDigest(toolCalls, diagnosticNotes)
		gatewayMode := aiapp.EffectiveChatGatewayMode(turn.conversation.AssistantMode, toolCalls, diagnosticNotes)

		history, err := s.buildGatewayMessages(ctx, turn.conversation.ID, chatRequestScope(turn.conversation, req), turn.userMessage.ID)
		if err != nil {
			return aiapp.RuntimeChatResponse{}, err
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
				Status:            aiapp.ConversationRunStatusFromError(err),
				Summary:           aiapp.UserFacingError(err),
				AssistantContent:  aiapp.BuildAssistantFailureReply(err),
				AssistantStatus:   aiapp.MessageStatusFromError(err),
				StructuredPayload: turn.assistantMessageStructured,
				ToolCallCount:     len(toolCalls),
				ProviderID:        turn.selectedProviderID,
				ModelID:           turn.selectedModelID,
			})
			runCompleted = true
			return aiapp.RuntimeChatResponse{}, err
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
		return aiapp.RuntimeChatResponse{}, err
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
		return aiapp.RuntimeChatResponse{}, err
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

	summary := aiapp.BuildConversationTitle(assistantContent)
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
		return aiapp.RuntimeChatResponse{}, err
	}
	runCompleted = true

	return aiapp.RuntimeChatResponse{
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

// SendChatStream performs the same business logic as SendMessage but streams
// the AI response via the onChunk callback. The callback is guaranteed to be
// called sequentially: zero or more "chunk" events, then exactly one "done" or "error" event.
func (s *ChatRuntime) SendChatStream(ctx context.Context, userID uint64, username string, req aiapp.RuntimeChatRequest, onChunk func(aiapp.RuntimeStreamChunk)) error {
	if s.db == nil || s.gateway == nil || s.toolSvc == nil {
		onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: "服务依赖未就绪"})
		return errors.New("ai dependencies are required")
	}
	turn, err := s.startConversationTurn(ctx, userID, username, req)
	if err != nil {
		onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: aiapp.UserFacingError(err)})
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
		onChunk(aiapp.RuntimeStreamChunk{Type: "progress", Progress: "正在收集集群诊断证据..."})

		history, err := s.buildGatewayMessages(ctx, turn.conversation.ID, chatRequestScope(turn.conversation, req), turn.userMessage.ID)
		if err != nil {
			onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: "构建消息历史失败"})
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
					onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: "保存助手消息失败"})
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

				summary := aiapp.BuildConversationTitle(assistantContent)
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

				onChunk(aiapp.RuntimeStreamChunk{Type: "chunk", Content: assistantContent})
				onChunk(aiapp.RuntimeStreamChunk{
					Type: "done",
					Content: mustMarshalJSON(aiapp.RuntimeStreamDoneData{
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
			onChunk(aiapp.RuntimeStreamChunk{Type: "progress", Progress: "正在收集集群诊断证据..."})

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
					Status:            aiapp.ConversationRunStatusFromError(diagErr),
					Summary:           aiapp.UserFacingError(diagErr),
					AssistantContent:  aiapp.BuildAssistantFailureReply(diagErr),
					AssistantStatus:   aiapp.MessageStatusFromError(diagErr),
					StructuredPayload: turn.assistantMessageStructured,
					ToolCallCount:     len(toolCalls),
					ProviderID:        turn.selectedProviderID,
					ModelID:           turn.selectedModelID,
				})
				runCompleted = true
				onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: aiapp.UserFacingError(diagErr)})
				return diagErr
			}
		}
		diagnosticSummary = aiapp.BuildChatDiagnosticEvidenceSummary(toolCalls)
		diagnosticNotes = aiapp.BuildChatDiagnosticEvidenceDigest(toolCalls, diagnosticNotes)
		gatewayMode := aiapp.EffectiveChatGatewayMode(turn.conversation.AssistantMode, toolCalls, diagnosticNotes)

		history, err := s.buildGatewayMessages(ctx, turn.conversation.ID, chatRequestScope(turn.conversation, req), turn.userMessage.ID)
		if err != nil {
			onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: "构建消息历史失败"})
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
				Status:            aiapp.ConversationRunStatusFromError(err),
				Summary:           aiapp.UserFacingError(err),
				AssistantContent:  aiapp.BuildAssistantFailureReply(err),
				AssistantStatus:   aiapp.MessageStatusFromError(err),
				StructuredPayload: turn.assistantMessageStructured,
				ToolCallCount:     len(toolCalls),
				ProviderID:        turn.selectedProviderID,
				ModelID:           turn.selectedModelID,
			})
			runCompleted = true
			onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: aiapp.UserFacingError(err)})
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
			onChunk(aiapp.RuntimeStreamChunk{Type: "chunk", Content: delta})

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
			onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: "流式读取中断"})
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
			onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: "保存助手消息失败"})
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

		summary := aiapp.BuildConversationTitle(assistantContent)
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

		onChunk(aiapp.RuntimeStreamChunk{
			Type: "done",
			Content: mustMarshalJSON(aiapp.RuntimeStreamDoneData{
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
func (s *ChatRuntime) streamFunctionCallingResponse(
	ctx context.Context,
	turn aiPreparedConversationTurn,
	req aiapp.RuntimeChatRequest,
	userID uint64,
	username string,
	toolCalls []AIToolCallItem,
	streamReader io.ReadCloser,
	streamResult AIGatewayStreamResult,
	onChunk func(aiapp.RuntimeStreamChunk),
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
		onChunk(aiapp.RuntimeStreamChunk{Type: "chunk", Content: delta})

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
		onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: "流式读取中断"})
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
		onChunk(aiapp.RuntimeStreamChunk{Type: "error", Error: "保存助手消息失败"})
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

	summary := aiapp.BuildConversationTitle(assistantContent)
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

	onChunk(aiapp.RuntimeStreamChunk{
		Type: "done",
		Content: mustMarshalJSON(aiapp.RuntimeStreamDoneData{
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

func (s *ChatRuntime) beginConversationRun(ctx context.Context, conversationID uint64, lastMessageAt time.Time) error {
	if s == nil || s.db == nil {
		return errors.New("db is required")
	}
	if conversationID == 0 {
		return aiapp.ErrorWithMessage(aiapp.ErrInvalidParams, "会话 ID 无效")
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
			return aiapp.ErrNotFound
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(row.Status), "running") {
		return aiapp.ErrorWithMessage(aiapp.ErrConflict, "当前会话仍在处理中，请先等待完成或取消当前回答")
	}
	return aiapp.ErrConflict
}

func chatRequestScope(conversation model.AIConversation, req aiapp.RuntimeChatRequest) aiapp.ChatRequestScope {
	return aiapp.NewChatRequestScope(conversation.AssistantMode, req.Namespace, req.ResourceKind, req.ResourceName)
}

func (s *ChatRuntime) finishConversationRun(conversationID uint64, result aiConversationRunResult) {
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

func (s *ChatRuntime) resetConversationRun(conversationID uint64) {
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

func (s *ChatRuntime) ensureConversation(ctx context.Context, userID uint64, username string, req aiapp.RuntimeChatRequest) (model.AIConversation, error) {
	if req.ConversationID != nil && *req.ConversationID > 0 {
		var conversation model.AIConversation
		if err := s.db.WithContext(ctx).
			Where("deleted_at IS NULL AND id = ? AND cluster_id = ?", *req.ConversationID, req.ClusterID).
			First(&conversation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.AIConversation{}, aiapp.ErrNotFound
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

func (s *ChatRuntime) buildGatewayMessages(
	ctx context.Context,
	conversationID uint64,
	currentScope aiapp.ChatRequestScope,
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

	historyRows := make([]aiapp.ChatHistoryMessage, 0, len(rows))
	for _, row := range rows {
		historyRows = append(historyRows, aiapp.ChatHistoryMessage{
			ID:         row.ID,
			Role:       row.Role,
			Content:    row.Content,
			Structured: model.JSONMap(row.StructuredJSON),
		})
	}
	filteredRows := aiapp.FilterScopedChatHistory(historyRows, currentScope)
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

func (s *ChatRuntime) autoCreateActionProposals(
	ctx context.Context,
	userID uint64,
	username string,
	req aiapp.RuntimeChatRequest,
	conversationID uint64,
	messageID uint64,
	message string,
) []aiapp.ActionProposalItem {
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

	items := make([]aiapp.ActionProposalItem, 0, len(specs))
	for _, spec := range specs {
		if s.hasOpenProposal(ctx, conversationID, req.ClusterID, spec) {
			continue
		}
		result, err := s.actionSvc.CreateProposal(ctx, req.ClusterID, userID, username, aiapp.CreateActionProposalRequest{
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

func (s *ChatRuntime) hasOpenProposal(
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
	if spec.ProposalType != aiapp.ActionTypeScaleWorkload {
		return true
	}

	targetReplicas, ok := aiapp.IntegerInputValue(spec.Payload["replicas"])
	if !ok {
		return false
	}
	for _, row := range rows {
		if replicas, ok := aiapp.IntegerInputValue(aiapp.ActionPayload(row.ChangeJSON)["replicas"]); ok && replicas == targetReplicas {
			return true
		}
		if replicas, ok := aiapp.IntegerInputValue(row.ChangeJSON["target_replicas"]); ok && replicas == targetReplicas {
			return true
		}
	}
	return false
}
