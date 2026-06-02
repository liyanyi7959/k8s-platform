package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/internal/model"
)

const (
	aiActionTypeRestartWorkload = "restart_workload"
	aiActionTypeScaleWorkload   = "scale_workload"
	aiActionTypeApplyManifest   = "apply_manifest"
)

type AIActionTargetResource struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type CreateAIActionProposalRequest struct {
	ConversationID uint64                 `json:"conversation_id"`
	MessageID      *uint64                `json:"message_id"`
	ProposalType   string                 `json:"proposal_type"`
	TargetResource AIActionTargetResource `json:"target_resource"`
	Payload        model.JSONMap          `json:"payload"`
	Reason         string                 `json:"reason"`
}

type ConfirmAIActionProposalRequest struct {
	ConfirmationText string `json:"confirmation_text"`
	ConfirmRisk      bool   `json:"confirm_risk"`
	OperatorComment  string `json:"operator_comment"`
}

type AIActionExecutionItem struct {
	ID              uint64        `json:"id"`
	ProposalID      uint64        `json:"proposal_id"`
	Status          string        `json:"status"`
	ExecutionNo     int           `json:"execution_no"`
	OperatorID      uint64        `json:"operator_id"`
	OperatorName    string        `json:"operator_name"`
	CommandSnapshot string        `json:"command_snapshot"`
	Result          model.JSONMap `json:"result,omitempty"`
	ErrorMessage    string        `json:"error_message,omitempty"`
	StartedAt       *string       `json:"started_at,omitempty"`
	FinishedAt      *string       `json:"finished_at,omitempty"`
	CreatedAt       string        `json:"created_at"`
}

type AIActionProposalItem struct {
	ID                 uint64                  `json:"id"`
	ConversationID     uint64                  `json:"conversation_id"`
	MessageID          *uint64                 `json:"message_id,omitempty"`
	ToolCallID         *uint64                 `json:"tool_call_id,omitempty"`
	ClusterID          uint64                  `json:"cluster_id"`
	ActionType         string                  `json:"action_type"`
	TargetKind         string                  `json:"target_kind"`
	TargetNamespace    string                  `json:"target_namespace"`
	TargetName         string                  `json:"target_name"`
	RiskLevel          string                  `json:"risk_level"`
	ConfirmLevel       string                  `json:"confirm_level"`
	Status             string                  `json:"status"`
	Title              string                  `json:"title"`
	Summary            string                  `json:"summary"`
	Change             model.JSONMap           `json:"change,omitempty"`
	CreatedBy          uint64                  `json:"created_by"`
	CreatedByName      string                  `json:"created_by_name"`
	ApprovedBy         *uint64                 `json:"approved_by,omitempty"`
	ApprovedByName     string                  `json:"approved_by_name"`
	ApprovedAt         *string                 `json:"approved_at,omitempty"`
	SecondApprovedBy   *uint64                 `json:"second_approved_by,omitempty"`
	SecondApprovedName string                  `json:"second_approved_name"`
	SecondApprovedAt   *string                 `json:"second_approved_at,omitempty"`
	LatestExecution    *AIActionExecutionItem  `json:"latest_execution,omitempty"`
	Executions         []AIActionExecutionItem `json:"executions,omitempty"`
	CreatedAt          string                  `json:"created_at"`
	UpdatedAt          string                  `json:"updated_at"`
}

type CreateAIActionProposalResult struct {
	ProposalID        uint64               `json:"proposal_id"`
	Status            string               `json:"status"`
	RiskLevel         string               `json:"risk_level"`
	NeedSecondConfirm bool                 `json:"need_second_confirm"`
	Preview           string               `json:"preview"`
	Diff              string               `json:"diff"`
	Proposal          AIActionProposalItem `json:"proposal"`
}

type ConfirmAIActionProposalResult struct {
	ProposalID      uint64                 `json:"proposal_id"`
	ExecutionStatus string                 `json:"execution_status"`
	ResultSummary   string                 `json:"result_summary"`
	Proposal        AIActionProposalItem   `json:"proposal"`
	Execution       *AIActionExecutionItem `json:"execution,omitempty"`
}

type AIActionService struct {
	db          *gorm.DB
	k8sSvc      *K8sService
	manifestSvc *ManifestApplyRecordService
}

func NewAIActionService(db *gorm.DB, k8sSvc *K8sService, manifestSvc *ManifestApplyRecordService) *AIActionService {
	return &AIActionService{db: db, k8sSvc: k8sSvc, manifestSvc: manifestSvc}
}

func (s *AIActionService) CreateProposal(
	ctx context.Context,
	clusterID uint64,
	userID uint64,
	username string,
	req CreateAIActionProposalRequest,
) (CreateAIActionProposalResult, error) {
	if s.db == nil {
		return CreateAIActionProposalResult{}, errors.New("db is required")
	}
	if s.k8sSvc == nil {
		return CreateAIActionProposalResult{}, errors.New("k8s service is required")
	}
	if clusterID == 0 || req.ConversationID == 0 {
		return CreateAIActionProposalResult{}, ErrWithMessage(ErrInvalidParams, "集群或会话参数无效")
	}

	actionType := normalizeAIActionType(req.ProposalType)
	if actionType == "" {
		return CreateAIActionProposalResult{}, ErrWithMessage(ErrInvalidParams, "当前动作类型暂不支持")
	}

	var conversation model.AIConversation
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND id = ? AND cluster_id = ?", req.ConversationID, clusterID).
		First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreateAIActionProposalResult{}, ErrNotFound
		}
		return CreateAIActionProposalResult{}, err
	}

	proposalRow, preview, diffText, err := s.buildProposalRow(ctx, clusterID, userID, username, req)
	if err != nil {
		return CreateAIActionProposalResult{}, err
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&proposalRow).Error; err != nil {
			return err
		}
		if err := s.appendProposalMessageTx(tx, proposalRow.ConversationID, userID, proposalRow.Title, proposalRow.Summary); err != nil {
			return err
		}
		return tx.Model(&model.AIConversation{}).
			Where("id = ?", proposalRow.ConversationID).
			Updates(map[string]any{
				"status":          "waiting_confirm",
				"last_message_at": time.Now().UTC(),
			}).Error
	}); err != nil {
		return CreateAIActionProposalResult{}, err
	}

	item, err := s.GetProposal(ctx, clusterID, proposalRow.ID)
	if err != nil {
		return CreateAIActionProposalResult{}, err
	}

	return CreateAIActionProposalResult{
		ProposalID:        proposalRow.ID,
		Status:            proposalRow.Status,
		RiskLevel:         proposalRow.RiskLevel,
		NeedSecondConfirm: proposalRow.ConfirmLevel == "double",
		Preview:           preview,
		Diff:              diffText,
		Proposal:          item,
	}, nil
}

func (s *AIActionService) ConfirmProposal(
	ctx context.Context,
	clusterID uint64,
	proposalID uint64,
	userID uint64,
	username string,
	req ConfirmAIActionProposalRequest,
) (ConfirmAIActionProposalResult, error) {
	if s.db == nil {
		return ConfirmAIActionProposalResult{}, errors.New("db is required")
	}
	if s.k8sSvc == nil {
		return ConfirmAIActionProposalResult{}, errors.New("k8s service is required")
	}
	if clusterID == 0 || proposalID == 0 {
		return ConfirmAIActionProposalResult{}, ErrWithMessage(ErrInvalidParams, "提案参数无效")
	}

	var proposal model.AIActionProposal
	if err := s.db.WithContext(ctx).
		Where("id = ? AND cluster_id = ?", proposalID, clusterID).
		First(&proposal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ConfirmAIActionProposalResult{}, ErrNotFound
		}
		return ConfirmAIActionProposalResult{}, err
	}

	switch proposal.Status {
	case "executing", "succeeded":
		return ConfirmAIActionProposalResult{}, ErrWithMessage(ErrConflict, "该提案已执行，无需重复确认")
	case "cancelled", "failed":
		return ConfirmAIActionProposalResult{}, ErrWithMessage(ErrConflict, "该提案当前状态不允许确认")
	}

	now := time.Now().UTC()
	if proposal.ConfirmLevel == "double" && proposal.ApprovedBy == nil {
		if err := s.db.WithContext(ctx).Model(&model.AIActionProposal{}).
			Where("id = ?", proposal.ID).
			Updates(map[string]any{
				"status":           "approved",
				"approved_by":      userID,
				"approved_by_name": strings.TrimSpace(username),
				"approved_at":      &now,
			}).Error; err != nil {
			return ConfirmAIActionProposalResult{}, err
		}
		item, err := s.GetProposal(ctx, clusterID, proposal.ID)
		if err != nil {
			return ConfirmAIActionProposalResult{}, err
		}
		return ConfirmAIActionProposalResult{
			ProposalID:      proposal.ID,
			ExecutionStatus: "waiting_second_confirm",
			ResultSummary:   "已完成首次确认，等待第二次确认后执行",
			Proposal:        item,
		}, nil
	}

	executionRow, summary, err := s.executeProposal(ctx, proposal, userID, username, strings.TrimSpace(req.OperatorComment))
	if err != nil {
		return ConfirmAIActionProposalResult{}, err
	}
	item, err := s.GetProposal(ctx, clusterID, proposal.ID)
	if err != nil {
		return ConfirmAIActionProposalResult{}, err
	}

	return ConfirmAIActionProposalResult{
		ProposalID:      proposal.ID,
		ExecutionStatus: executionRow.Status,
		ResultSummary:   summary,
		Proposal:        item,
		Execution:       ptrAIActionExecutionItem(buildAIActionExecutionItem(executionRow)),
	}, nil
}

func (s *AIActionService) GetProposal(ctx context.Context, clusterID uint64, proposalID uint64) (AIActionProposalItem, error) {
	if s.db == nil {
		return AIActionProposalItem{}, errors.New("db is required")
	}
	if clusterID == 0 || proposalID == 0 {
		return AIActionProposalItem{}, ErrWithMessage(ErrInvalidParams, "提案参数无效")
	}
	var row model.AIActionProposal
	if err := s.db.WithContext(ctx).
		Where("id = ? AND cluster_id = ?", proposalID, clusterID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AIActionProposalItem{}, ErrNotFound
		}
		return AIActionProposalItem{}, err
	}
	items, err := listConversationActionProposals(ctx, s.db, row.ConversationID)
	if err != nil {
		return AIActionProposalItem{}, err
	}
	for _, item := range items {
		if item.ID == proposalID {
			return item, nil
		}
	}
	return AIActionProposalItem{}, ErrNotFound
}

func (s *AIActionService) ListConversationProposals(ctx context.Context, conversationID uint64) ([]AIActionProposalItem, error) {
	return listConversationActionProposals(ctx, s.db, conversationID)
}

func listConversationActionProposals(ctx context.Context, db *gorm.DB, conversationID uint64) ([]AIActionProposalItem, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}
	if conversationID == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "会话 ID 无效")
	}

	var proposals []model.AIActionProposal
	if err := db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC, id DESC").
		Find(&proposals).Error; err != nil {
		return nil, err
	}

	executionMap, err := listActionExecutionsByProposal(ctx, db, proposals)
	if err != nil {
		return nil, err
	}

	items := make([]AIActionProposalItem, 0, len(proposals))
	for _, row := range proposals {
		items = append(items, buildAIActionProposalItem(row, executionMap[row.ID]))
	}
	return items, nil
}

func listActionExecutionsByProposal(
	ctx context.Context,
	db *gorm.DB,
	proposals []model.AIActionProposal,
) (map[uint64][]model.AIActionExecution, error) {
	result := make(map[uint64][]model.AIActionExecution, len(proposals))
	if len(proposals) == 0 {
		return result, nil
	}
	ids := make([]uint64, 0, len(proposals))
	for _, row := range proposals {
		ids = append(ids, row.ID)
	}

	var rows []model.AIActionExecution
	if err := db.WithContext(ctx).
		Where("proposal_id IN ?", ids).
		Order("created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ProposalID] = append(result[row.ProposalID], row)
	}
	return result, nil
}

func (s *AIActionService) buildProposalRow(
	ctx context.Context,
	clusterID uint64,
	userID uint64,
	username string,
	req CreateAIActionProposalRequest,
) (model.AIActionProposal, string, string, error) {
	target := normalizeAIActionTarget(req.TargetResource)
	payload := req.Payload
	if payload == nil {
		payload = model.JSONMap{}
	}
	reason := strings.TrimSpace(req.Reason)

	row := model.AIActionProposal{
		ConversationID:  req.ConversationID,
		MessageID:       req.MessageID,
		ClusterID:       clusterID,
		ActionType:      normalizeAIActionType(req.ProposalType),
		TargetKind:      target.Kind,
		TargetNamespace: target.Namespace,
		TargetName:      target.Name,
		Status:          "pending_confirm",
		CreatedBy:       userID,
		CreatedByName:   strings.TrimSpace(username),
	}

	switch row.ActionType {
	case aiActionTypeRestartWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok || target.Namespace == "" || target.Name == "" {
			return model.AIActionProposal{}, "", "", ErrWithMessage(ErrInvalidParams, "重启提案的目标资源参数无效")
		}
		if _, err := s.k8sSvc.GetObject(ctx, clusterID, gvr, target.Namespace, target.Name); err != nil {
			return model.AIActionProposal{}, "", "", err
		}
		row.RiskLevel = "low"
		row.ConfirmLevel = "single"
		row.Title = fmt.Sprintf("重启 %s %s/%s", target.Kind, target.Namespace, target.Name)
		row.Summary = firstNonEmpty(reason, fmt.Sprintf("建议对 %s %s/%s 发起滚动重启，以验证问题是否已恢复。", target.Kind, target.Namespace, target.Name))
		row.ChangeJSON = model.JSONMap{
			"payload": payload,
			"preview": fmt.Sprintf("将为 %s %s/%s 写入 rollout restart 注解。", target.Kind, target.Namespace, target.Name),
		}
		return row, aiActionString(row.ChangeJSON["preview"]), "", nil
	case aiActionTypeScaleWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok || target.Namespace == "" || target.Name == "" || strings.EqualFold(target.Kind, "DaemonSet") {
			return model.AIActionProposal{}, "", "", ErrWithMessage(ErrInvalidParams, "扩缩容提案的目标资源参数无效")
		}
		replicas, ok := jsonIntValue(payload["replicas"])
		if !ok || replicas < 0 {
			return model.AIActionProposal{}, "", "", ErrWithMessage(ErrInvalidParams, "扩缩容提案缺少有效的 replicas 参数")
		}
		obj, err := s.k8sSvc.GetObject(ctx, clusterID, gvr, target.Namespace, target.Name)
		if err != nil {
			return model.AIActionProposal{}, "", "", err
		}
		currentReplicas := jsonNestedInt(obj, "spec", "replicas")
		row.RiskLevel = "low"
		row.ConfirmLevel = "single"
		row.Title = fmt.Sprintf("调整 %s %s/%s 副本数", target.Kind, target.Namespace, target.Name)
		row.Summary = firstNonEmpty(reason, fmt.Sprintf("建议将 %s %s/%s 的副本数从 %d 调整为 %d。", target.Kind, target.Namespace, target.Name, currentReplicas, replicas))
		row.ChangeJSON = model.JSONMap{
			"payload":          payload,
			"current_replicas": currentReplicas,
			"target_replicas":  replicas,
			"preview":          fmt.Sprintf("副本数变更预览: %d -> %d", currentReplicas, replicas),
		}
		return row, aiActionString(row.ChangeJSON["preview"]), fmt.Sprintf("replicas: %d -> %d", currentReplicas, replicas), nil
	case aiActionTypeApplyManifest:
		yamlText := strings.TrimSpace(aiActionString(payload["yaml"]))
		if yamlText == "" {
			return model.AIActionProposal{}, "", "", ErrWithMessage(ErrInvalidParams, "Manifest 提案缺少 YAML 内容")
		}
		defaultNamespace := strings.TrimSpace(aiActionString(payload["default_namespace"]))
		row.RiskLevel = "medium"
		row.ConfirmLevel = "single"
		row.Title = firstNonEmpty(strings.TrimSpace(aiActionString(payload["title"])), "应用 AI 生成的 Manifest")
		row.Summary = firstNonEmpty(reason, "建议执行一份已预览的 Manifest 变更，请在确认前仔细检查 YAML 内容。")
		row.ChangeJSON = model.JSONMap{
			"payload":           payload,
			"default_namespace": defaultNamespace,
			"preview":           truncateForModel(yamlText, 1000),
		}
		return row, truncateForModel(yamlText, 240), truncateForModel(yamlText, 1200), nil
	default:
		return model.AIActionProposal{}, "", "", ErrWithMessage(ErrInvalidParams, "当前动作类型暂不支持")
	}
}

func (s *AIActionService) executeProposal(
	ctx context.Context,
	proposal model.AIActionProposal,
	userID uint64,
	username string,
	operatorComment string,
) (model.AIActionExecution, string, error) {
	executionNo, err := s.nextExecutionNo(ctx, proposal.ID)
	if err != nil {
		return model.AIActionExecution{}, "", err
	}

	startedAt := time.Now().UTC()
	commandSnapshot := s.buildExecutionSnapshot(proposal, operatorComment)
	executionRow := model.AIActionExecution{
		ProposalID:      proposal.ID,
		ConversationID:  proposal.ConversationID,
		ClusterID:       proposal.ClusterID,
		ExecutionNo:     executionNo,
		Status:          "running",
		StartedAt:       &startedAt,
		OperatorID:      userID,
		OperatorName:    strings.TrimSpace(username),
		CommandSnapshot: commandSnapshot,
	}
	if err := s.db.WithContext(ctx).Create(&executionRow).Error; err != nil {
		return model.AIActionExecution{}, "", err
	}

	approveUpdates := map[string]any{
		"status":           "executing",
		"approved_by_name": strings.TrimSpace(username),
	}
	if proposal.ApprovedBy == nil {
		approveUpdates["approved_by"] = userID
		approveUpdates["approved_at"] = &startedAt
	}
	if err := s.db.WithContext(ctx).Model(&model.AIActionProposal{}).
		Where("id = ?", proposal.ID).
		Updates(approveUpdates).Error; err != nil {
		return model.AIActionExecution{}, "", err
	}

	resultJSON, summary, execErr := s.runProposalAction(ctx, proposal)
	finishedAt := time.Now().UTC()
	if execErr != nil {
		_ = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			_ = tx.Model(&model.AIActionExecution{}).
				Where("id = ?", executionRow.ID).
				Updates(map[string]any{
					"status":        "failed",
					"finished_at":   &finishedAt,
					"error_message": firstUserFacingError(execErr),
				}).Error
			_ = tx.Model(&model.AIActionProposal{}).
				Where("id = ?", proposal.ID).
				Updates(map[string]any{
					"status": "failed",
				}).Error
			_ = s.appendExecutionMessageTx(tx, proposal.ConversationID, userID, proposal.Title, "failed", firstUserFacingError(execErr))
			_ = tx.Model(&model.AIConversation{}).
				Where("id = ?", proposal.ConversationID).
				Updates(map[string]any{
					"status":          "open",
					"last_message_at": &finishedAt,
				}).Error
			return nil
		})
		return model.AIActionExecution{}, "", execErr
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AIActionExecution{}).
			Where("id = ?", executionRow.ID).
			Updates(map[string]any{
				"status":        "succeeded",
				"finished_at":   &finishedAt,
				"result_json":   resultJSON,
				"error_message": "",
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AIActionProposal{}).
			Where("id = ?", proposal.ID).
			Updates(map[string]any{
				"status": "succeeded",
			}).Error; err != nil {
			return err
		}
		if err := s.appendExecutionMessageTx(tx, proposal.ConversationID, userID, proposal.Title, "succeeded", summary); err != nil {
			return err
		}
		return tx.Model(&model.AIConversation{}).
			Where("id = ?", proposal.ConversationID).
			Updates(map[string]any{
				"status":          "open",
				"last_message_at": &finishedAt,
			}).Error
	}); err != nil {
		return model.AIActionExecution{}, "", err
	}

	executionRow.Status = "succeeded"
	executionRow.ResultJSON = resultJSON
	executionRow.FinishedAt = &finishedAt
	return executionRow, summary, nil
}

func (s *AIActionService) runProposalAction(
	ctx context.Context,
	proposal model.AIActionProposal,
) (model.JSONMap, string, error) {
	changePayload := aiActionPayload(proposal.ChangeJSON)
	switch normalizeAIActionType(proposal.ActionType) {
	case aiActionTypeRestartWorkload:
		gvr, ok := aiActionWorkloadGVR(proposal.TargetKind)
		if !ok {
			return nil, "", ErrWithMessage(ErrInvalidParams, "重启提案资源类型不支持")
		}
		patch := map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"metadata": map[string]any{
						"annotations": map[string]any{
							"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339),
						},
					},
				},
			},
		}
		if err := s.k8sSvc.PatchJSON(ctx, proposal.ClusterID, gvr, proposal.TargetNamespace, proposal.TargetName, patch); err != nil {
			return nil, "", err
		}
		summary := fmt.Sprintf("已触发 %s %s/%s 的滚动重启。", proposal.TargetKind, proposal.TargetNamespace, proposal.TargetName)
		return model.JSONMap{
			"action_type": proposal.ActionType,
			"target":      fmt.Sprintf("%s/%s/%s", proposal.TargetKind, proposal.TargetNamespace, proposal.TargetName),
			"summary":     summary,
		}, summary, nil
	case aiActionTypeScaleWorkload:
		gvr, ok := aiActionWorkloadGVR(proposal.TargetKind)
		if !ok {
			return nil, "", ErrWithMessage(ErrInvalidParams, "扩缩容提案资源类型不支持")
		}
		targetReplicas, ok := jsonIntValue(changePayload["replicas"])
		if !ok || targetReplicas < 0 {
			targetReplicas, ok = jsonIntValue(proposal.ChangeJSON["target_replicas"])
		}
		if !ok || targetReplicas < 0 {
			return nil, "", ErrWithMessage(ErrInvalidParams, "扩缩容提案缺少有效副本数")
		}
		if err := s.k8sSvc.PatchJSON(ctx, proposal.ClusterID, gvr, proposal.TargetNamespace, proposal.TargetName, map[string]any{
			"spec": map[string]any{
				"replicas": targetReplicas,
			},
		}); err != nil {
			return nil, "", err
		}
		summary := fmt.Sprintf("已将 %s %s/%s 的副本数调整为 %d。", proposal.TargetKind, proposal.TargetNamespace, proposal.TargetName, targetReplicas)
		return model.JSONMap{
			"action_type":     proposal.ActionType,
			"target_replicas": targetReplicas,
			"summary":         summary,
		}, summary, nil
	case aiActionTypeApplyManifest:
		if s.manifestSvc == nil {
			return nil, "", ErrWithMessage(ErrConflict, "Manifest 执行服务未初始化")
		}
		yamlText := strings.TrimSpace(aiActionString(changePayload["yaml"]))
		if yamlText == "" {
			return nil, "", ErrWithMessage(ErrInvalidParams, "Manifest 提案缺少 YAML 内容")
		}
		defaultNamespace := strings.TrimSpace(aiActionString(changePayload["default_namespace"]))
		result, err := s.manifestSvc.Execute(ctx, ManifestApplyExecuteRequest{
			ClusterID:        proposal.ClusterID,
			YAML:             yamlText,
			DefaultNamespace: defaultNamespace,
			DryRun:           false,
			SourceLabel:      "AI 动作提案",
			SourceResource:   proposal.Title,
			CreatedBy:        proposal.CreatedBy,
			CreatedByName:    proposal.CreatedByName,
		})
		if err != nil {
			return nil, "", err
		}
		summary := firstNonEmpty(strings.TrimSpace(result.Summary), "Manifest 已执行")
		return model.JSONMap{
			"action_type": proposal.ActionType,
			"record_id":   result.RecordID,
			"status":      result.Status,
			"summary":     summary,
		}, summary, nil
	default:
		return nil, "", ErrWithMessage(ErrInvalidParams, "当前动作类型暂不支持执行")
	}
}

func (s *AIActionService) nextExecutionNo(ctx context.Context, proposalID uint64) (int, error) {
	var maxNo int
	if err := s.db.WithContext(ctx).
		Model(&model.AIActionExecution{}).
		Select("COALESCE(MAX(execution_no), 0)").
		Where("proposal_id = ?", proposalID).
		Scan(&maxNo).Error; err != nil {
		return 0, err
	}
	return maxNo + 1, nil
}

func (s *AIActionService) buildExecutionSnapshot(proposal model.AIActionProposal, operatorComment string) string {
	payload := map[string]any{
		"action_type":      proposal.ActionType,
		"target_kind":      proposal.TargetKind,
		"target_namespace": proposal.TargetNamespace,
		"target_name":      proposal.TargetName,
		"change":           proposal.ChangeJSON,
		"operator_comment": strings.TrimSpace(operatorComment),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (s *AIActionService) appendProposalMessageTx(
	tx *gorm.DB,
	conversationID uint64,
	userID uint64,
	title string,
	summary string,
) error {
	message := model.AIMessage{
		ConversationID: conversationID,
		Role:           "system",
		MessageType:    "proposal",
		Content:        fmt.Sprintf("已生成变更提案：%s\n%s", strings.TrimSpace(title), strings.TrimSpace(summary)),
		Status:         "created",
		CreatedBy:      userID,
	}
	return tx.Create(&message).Error
}

func (s *AIActionService) appendExecutionMessageTx(
	tx *gorm.DB,
	conversationID uint64,
	userID uint64,
	title string,
	status string,
	summary string,
) error {
	content := fmt.Sprintf("变更提案执行%s：%s\n%s", aiActionExecutionStatusLabel(status), strings.TrimSpace(title), strings.TrimSpace(summary))
	message := model.AIMessage{
		ConversationID: conversationID,
		Role:           "system",
		MessageType:    "action_execution",
		Content:        content,
		Status:         "created",
		CreatedBy:      userID,
	}
	return tx.Create(&message).Error
}

func buildAIActionProposalItem(
	row model.AIActionProposal,
	executions []model.AIActionExecution,
) AIActionProposalItem {
	var approvedAt *string
	if row.ApprovedAt != nil {
		v := row.ApprovedAt.UTC().Format(time.RFC3339)
		approvedAt = &v
	}
	var secondApprovedAt *string
	if row.SecondApprovedAt != nil {
		v := row.SecondApprovedAt.UTC().Format(time.RFC3339)
		secondApprovedAt = &v
	}

	executionItems := make([]AIActionExecutionItem, 0, len(executions))
	for _, row := range executions {
		executionItems = append(executionItems, buildAIActionExecutionItem(row))
	}

	var latestExecution *AIActionExecutionItem
	if len(executionItems) > 0 {
		latestExecution = ptrAIActionExecutionItem(executionItems[0])
	}

	return AIActionProposalItem{
		ID:                 row.ID,
		ConversationID:     row.ConversationID,
		MessageID:          row.MessageID,
		ToolCallID:         row.ToolCallID,
		ClusterID:          row.ClusterID,
		ActionType:         row.ActionType,
		TargetKind:         row.TargetKind,
		TargetNamespace:    row.TargetNamespace,
		TargetName:         row.TargetName,
		RiskLevel:          row.RiskLevel,
		ConfirmLevel:       row.ConfirmLevel,
		Status:             row.Status,
		Title:              row.Title,
		Summary:            row.Summary,
		Change:             row.ChangeJSON,
		CreatedBy:          row.CreatedBy,
		CreatedByName:      row.CreatedByName,
		ApprovedBy:         row.ApprovedBy,
		ApprovedByName:     row.ApprovedByName,
		ApprovedAt:         approvedAt,
		SecondApprovedBy:   row.SecondApprovedBy,
		SecondApprovedName: row.SecondApprovedName,
		SecondApprovedAt:   secondApprovedAt,
		LatestExecution:    latestExecution,
		Executions:         executionItems,
		CreatedAt:          row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func buildAIActionExecutionItem(row model.AIActionExecution) AIActionExecutionItem {
	var startedAt *string
	if row.StartedAt != nil {
		v := row.StartedAt.UTC().Format(time.RFC3339)
		startedAt = &v
	}
	var finishedAt *string
	if row.FinishedAt != nil {
		v := row.FinishedAt.UTC().Format(time.RFC3339)
		finishedAt = &v
	}
	return AIActionExecutionItem{
		ID:              row.ID,
		ProposalID:      row.ProposalID,
		Status:          row.Status,
		ExecutionNo:     row.ExecutionNo,
		OperatorID:      row.OperatorID,
		OperatorName:    row.OperatorName,
		CommandSnapshot: row.CommandSnapshot,
		Result:          row.ResultJSON,
		ErrorMessage:    row.ErrorMessage,
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		CreatedAt:       row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func normalizeAIActionType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case aiActionTypeRestartWorkload:
		return aiActionTypeRestartWorkload
	case aiActionTypeScaleWorkload:
		return aiActionTypeScaleWorkload
	case aiActionTypeApplyManifest:
		return aiActionTypeApplyManifest
	default:
		return ""
	}
}

func normalizeAIActionTarget(target AIActionTargetResource) AIActionTargetResource {
	kind := strings.TrimSpace(target.Kind)
	switch strings.ToLower(kind) {
	case "deployment":
		kind = "Deployment"
	case "statefulset":
		kind = "StatefulSet"
	case "daemonset":
		kind = "DaemonSet"
	}
	return AIActionTargetResource{
		Kind:      kind,
		Namespace: strings.TrimSpace(target.Namespace),
		Name:      strings.TrimSpace(target.Name),
	}
}

func aiActionWorkloadGVR(kind string) (schema.GroupVersionResource, bool) {
	switch strings.TrimSpace(kind) {
	case "Deployment":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, true
	case "StatefulSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, true
	case "DaemonSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, true
	default:
		return schema.GroupVersionResource{}, false
	}
}

func aiActionPayload(change model.JSONMap) model.JSONMap {
	if change == nil {
		return model.JSONMap{}
	}
	if payload, ok := change["payload"].(map[string]any); ok && payload != nil {
		return model.JSONMap(payload)
	}
	return model.JSONMap{}
}

func aiActionExecutionStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded":
		return "成功"
	case "failed":
		return "失败"
	default:
		return "完成"
	}
}

func jsonIntValue(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func jsonNestedInt(obj map[string]any, keys ...string) int {
	current := obj
	for i := 0; i < len(keys)-1; i++ {
		next, _ := current[keys[i]].(map[string]any)
		if next == nil {
			return 0
		}
		current = next
	}
	value, ok := jsonIntValue(current[keys[len(keys)-1]])
	if !ok {
		return 0
	}
	return value
}

func ptrAIActionExecutionItem(item AIActionExecutionItem) *AIActionExecutionItem {
	return &item
}

func aiActionString(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	case []byte:
		return strings.TrimSpace(string(value))
	default:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
