package service

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

var (
	scaleReplicaPatterns = []*regexp.Regexp{
		regexp.MustCompile("(?i)(?:replicas?|副本(?:数)?)[^\\d]{0,8}(\\d{1,4})"),
		regexp.MustCompile("(?i)(?:调整到|改到|改成|改为|设为|设置为|to|=|为|到)[^\\d]{0,4}(\\d{1,4})\\s*(?:replicas?|副本)?"),
		regexp.MustCompile("(?i)(?:scale|扩容|缩容|扩缩容|横向扩展|横向缩容)[^\\d]{0,10}(\\d{1,4})"),
	}
)

type AIChatRequest struct {
	ClusterID      uint64   `json:"-"`
	ConversationID *uint64  `json:"conversation_id"`
	Message        string   `json:"message"`
	AssistantMode  string   `json:"assistant_mode"`
	ProviderID     *uint64  `json:"provider_id"`
	ModelID        *uint64  `json:"model_id"`
	PreferModel    string   `json:"prefer_model"`
	Namespace      string   `json:"namespace"`
	ResourceKind   string   `json:"resource_kind"`
	ResourceName   string   `json:"resource_name"`
	UserPerms      []string `json:"-"`
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
}

func NewAIChatService(
	db *gorm.DB,
	gateway *AIGatewayService,
	toolSvc *AIToolService,
	actionSvc *AIActionService,
) *AIChatService {
	return &AIChatService{db: db, gateway: gateway, toolSvc: toolSvc, actionSvc: actionSvc}
}

type aiAutoDiagnosticsPlan struct {
	Enabled  bool
	Optional bool
	Timeout  time.Duration
}

func (s *AIChatService) SendMessage(ctx context.Context, userID uint64, username string, req AIChatRequest) (AIChatResponse, error) {
	if s.db == nil {
		return AIChatResponse{}, errors.New("db is required")
	}
	if s.gateway == nil || s.toolSvc == nil {
		return AIChatResponse{}, errors.New("ai dependencies are required")
	}
	if req.ClusterID == 0 {
		return AIChatResponse{}, ErrWithMessage(ErrInvalidParams, "集群 ID 无效")
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return AIChatResponse{}, ErrWithMessage(ErrInvalidParams, "消息内容不能为空")
	}

	conversation, err := s.ensureConversation(ctx, userID, username, req)
	if err != nil {
		return AIChatResponse{}, err
	}

	now := time.Now().UTC()
	userMessage := model.AIMessage{
		ConversationID: conversation.ID,
		Role:           "user",
		MessageType:    "text",
		Content:        message,
		Status:         "created",
		CreatedBy:      userID,
	}
	if err := s.db.WithContext(ctx).Create(&userMessage).Error; err != nil {
		return AIChatResponse{}, err
	}
	if err := s.db.WithContext(ctx).Model(&model.AIConversation{}).
		Where("id = ?", conversation.ID).
		Updates(map[string]any{"status": "running", "last_message_at": &now}).Error; err != nil {
		return AIChatResponse{}, err
	}

	toolCalls := make([]AIToolCallItem, 0, 6)
	diagnosticNotes := ""
	diagPlan := buildAIAutoDiagnosticsPlan(conversation.AssistantMode, req, message)
	if diagPlan.Enabled {
		diagCtx := ctx
		cancel := func() {}
		if diagPlan.Timeout > 0 {
			diagCtx, cancel = context.WithTimeout(ctx, diagPlan.Timeout)
		}
		defer cancel()

		var diagErr error
		toolCalls, diagnosticNotes, diagErr = s.toolSvc.RunAutoDiagnostics(diagCtx, AIToolContextRequest{
			ConversationID: conversation.ID,
			MessageID:      userMessage.ID,
			ClusterID:      req.ClusterID,
			UserID:         userID,
			Username:       username,
			UserPerms:      req.UserPerms,
			Query:          message,
			Namespace:      strings.TrimSpace(req.Namespace),
			ResourceKind:   strings.TrimSpace(req.ResourceKind),
			ResourceName:   strings.TrimSpace(req.ResourceName),
		})
		if diagErr != nil && !diagPlan.Optional {
			return AIChatResponse{}, diagErr
		}
	}

	history, err := s.buildGatewayMessages(ctx, conversation.ID)
	if err != nil {
		return AIChatResponse{}, err
	}
	providerID := conversation.ProviderID
	if req.ProviderID != nil {
		providerID = req.ProviderID
	}
	modelID := conversation.ModelID
	if req.ModelID != nil {
		modelID = req.ModelID
	}
	assistantResp, err := s.gateway.Invoke(ctx, AIGatewayRequest{
		ConversationID:  conversation.ID,
		AssistantMode:   conversation.AssistantMode,
		ProviderID:      providerID,
		ModelID:         modelID,
		PreferModelCode: req.PreferModel,
		Messages:        history,
		DiagnosticNotes: diagnosticNotes,
	})
	if err != nil {
		_ = s.db.WithContext(ctx).Model(&model.AIConversation{}).
			Where("id = ?", conversation.ID).
			Update("status", "failed").Error
		return AIChatResponse{}, err
	}

	assistantMessage := model.AIMessage{
		ConversationID: conversation.ID,
		Role:           "assistant",
		MessageType:    "text",
		Content:        assistantResp.Content,
		Status:         "created",
		ToolCallCount:  len(toolCalls),
		TokenInput:     assistantResp.Usage.RequestTokens,
		TokenOutput:    assistantResp.Usage.ResponseTokens,
	}
	if err := s.db.WithContext(ctx).Create(&assistantMessage).Error; err != nil {
		return AIChatResponse{}, err
	}

	usage := model.AIUsageRecord{
		ConversationID: ptrUint64(conversation.ID),
		MessageID:      ptrUint64(assistantMessage.ID),
		ProviderID:     ptrUint64(assistantResp.ProviderID),
		ModelID:        ptrUint64(assistantResp.ModelID),
		UsageType:      "chat",
		RequestTokens:  assistantResp.Usage.RequestTokens,
		ResponseTokens: assistantResp.Usage.ResponseTokens,
		TotalTokens:    assistantResp.Usage.TotalTokens,
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
		conversation.ID,
		assistantMessage.ID,
		message,
	)

	summary := buildConversationSummary(assistantResp.Content)
	conversationStatus := "open"
	if len(actionProposals) > 0 {
		conversationStatus = "waiting_confirm"
	}
	if err := s.db.WithContext(ctx).Model(&model.AIConversation{}).
		Where("id = ?", conversation.ID).
		Updates(map[string]any{
			"provider_id":     assistantResp.ProviderID,
			"model_id":        assistantResp.ModelID,
			"summary":         summary,
			"status":          conversationStatus,
			"last_message_at": time.Now().UTC(),
		}).Error; err != nil {
		return AIChatResponse{}, err
	}

	return AIChatResponse{
		ConversationID:     conversation.ID,
		UserMessageID:      userMessage.ID,
		AssistantMessageID: assistantMessage.ID,
		AssistantMessage:   assistantResp.Content,
		ProviderName:       assistantResp.ProviderName,
		ModelName:          assistantResp.ModelName,
		ModelCode:          assistantResp.ModelCode,
		ToolCalls:          toolCalls,
		ActionProposals:    actionProposals,
	}, nil
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

	mode := normalizeAIAssistantMode(req.AssistantMode)
	if mode == "" {
		mode = "diagnose"
	}
	title := buildConversationTitle(req.Message)
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

func (s *AIChatService) buildGatewayMessages(ctx context.Context, conversationID uint64) ([]AIGatewayMessage, error) {
	var rows []model.AIMessage
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC, id ASC").
		Limit(20).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	messages := make([]AIGatewayMessage, 0, len(rows))
	for _, row := range rows {
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

func buildConversationSummary(content string) string {
	return buildConversationTitle(content)
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

	specs := buildAutoActionProposalSpecs(req, message)
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
	spec autoActionProposalSpec,
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
		if replicas, ok := jsonIntValue(aiActionPayload(row.ChangeJSON)["replicas"]); ok && replicas == targetReplicas {
			return true
		}
		if replicas, ok := jsonIntValue(row.ChangeJSON["target_replicas"]); ok && replicas == targetReplicas {
			return true
		}
	}
	return false
}

type autoActionProposalSpec struct {
	ProposalType   string
	TargetResource AIActionTargetResource
	Payload        model.JSONMap
	Reason         string
}

func buildAutoActionProposalSpecs(req AIChatRequest, message string) []autoActionProposalSpec {
	target := normalizeAIActionTarget(AIActionTargetResource{
		Kind:      req.ResourceKind,
		Namespace: req.Namespace,
		Name:      req.ResourceName,
	})
	if target.Kind == "" || target.Namespace == "" || target.Name == "" {
		return nil
	}
	if _, ok := aiActionWorkloadGVR(target.Kind); !ok {
		return nil
	}

	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" || hasTentativeActionIntent(trimmedMessage) {
		return nil
	}

	specs := make([]autoActionProposalSpec, 0, 2)
	if isExplicitRestartIntent(trimmedMessage) {
		specs = append(specs, autoActionProposalSpec{
			ProposalType:   aiActionTypeRestartWorkload,
			TargetResource: target,
			Payload:        model.JSONMap{},
			Reason:         "Auto-generated rollout restart proposal from the latest chat intent. Please verify scope before confirming.",
		})
	}

	if !strings.EqualFold(target.Kind, "DaemonSet") {
		if replicas, ok := detectScaleReplicaTarget(trimmedMessage); ok && isExplicitScaleIntent(trimmedMessage) {
			specs = append(specs, autoActionProposalSpec{
				ProposalType:   aiActionTypeScaleWorkload,
				TargetResource: target,
				Payload: model.JSONMap{
					"replicas": replicas,
				},
				Reason: "Auto-generated scale proposal from the latest chat intent. Please verify target replicas before confirming.",
			})
		}
	}

	return specs
}

func isExplicitRestartIntent(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if strings.Contains(lower, "rollout restart") || strings.Contains(message, "\u6eda\u52a8\u91cd\u542f") {
		return true
	}
	if !strings.Contains(lower, "restart") && !strings.Contains(message, "\u91cd\u542f") {
		return false
	}
	if strings.HasPrefix(lower, "restart ") || strings.HasPrefix(lower, "please restart") {
		return true
	}
	return containsAny(
		message,
		"\u5e2e\u6211\u91cd\u542f",
		"\u8bf7\u91cd\u542f",
		"\u6267\u884c\u91cd\u542f",
		"\u53d1\u8d77\u91cd\u542f",
		"\u5148\u91cd\u542f",
		"\u7acb\u5373\u91cd\u542f",
	)
}

func isExplicitScaleIntent(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if containsAny(lower, "scale", "replica", "replicas") {
		return true
	}
	return containsAny(
		message,
		"\u6269\u5bb9",
		"\u7f29\u5bb9",
		"\u6269\u7f29\u5bb9",
		"\u526f\u672c",
		"\u526f\u672c\u6570",
		"\u8c03\u6574\u526f\u672c",
		"\u8c03\u6574\u5230",
		"\u6539\u6210",
		"\u6539\u4e3a",
	)
}

func detectScaleReplicaTarget(message string) (int, bool) {
	for _, pattern := range scaleReplicaPatterns {
		matches := pattern.FindStringSubmatch(message)
		if len(matches) < 2 {
			continue
		}
		replicas, err := strconv.Atoi(matches[1])
		if err != nil || replicas < 0 {
			continue
		}
		return replicas, true
	}
	return 0, false
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

func containsAny(text string, candidates ...string) bool {
	for _, candidate := range candidates {
		if candidate != "" && strings.Contains(text, candidate) {
			return true
		}
	}
	return false
}

func ptrUint64(v uint64) *uint64 {
	return &v
}

func buildAIAutoDiagnosticsPlan(mode string, req AIChatRequest, message string) aiAutoDiagnosticsPlan {
	normalizedMode := normalizeAIAssistantMode(mode)
	if normalizedMode == "diagnose" {
		return aiAutoDiagnosticsPlan{
			Enabled: true,
			Timeout: 20 * time.Second,
		}
	}

	if shouldRunChatDiagnostics(req, message) {
		return aiAutoDiagnosticsPlan{
			Enabled:  true,
			Optional: true,
			Timeout:  12 * time.Second,
		}
	}

	return aiAutoDiagnosticsPlan{}
}

func shouldRunChatDiagnostics(req AIChatRequest, message string) bool {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return false
	}
	if isGenericKnowledgeQuestion(trimmedMessage) {
		return false
	}

	namespace := strings.TrimSpace(req.Namespace)
	kind := strings.TrimSpace(req.ResourceKind)
	name := strings.TrimSpace(req.ResourceName)
	if kind != "" && name != "" {
		return true
	}
	if namespace != "" && looksLikeScopedClusterQuestion(trimmedMessage) {
		return true
	}
	return false
}

func isGenericKnowledgeQuestion(message string) bool {
	if containsAny(message,
		"哪些方面", "从哪些方面", "一般怎么", "通常怎么", "如何", "怎么做", "是什么", "有哪些", "最佳实践", "注意事项", "巡检思路", "排查思路", "设计方案", "实施步骤",
	) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(message))
	return containsAny(lower,
		"what is", "how to", "best practice", "checklist", "overview", "introduction", "general", "typically", "usually",
	)
}

func looksLikeScopedClusterQuestion(message string) bool {
	if containsAny(message,
		"这个集群", "当前集群", "本集群", "这个命名空间", "当前命名空间", "这个服务", "这个 deployment", "这个 pod", "帮我看", "帮我查", "看看", "查一下", "分析一下", "诊断一下", "排查一下",
	) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(message))
	return containsAny(lower,
		"check", "inspect", "diagnose", "analyze", "look into", "current cluster", "this cluster", "this namespace", "show me",
	)
}
