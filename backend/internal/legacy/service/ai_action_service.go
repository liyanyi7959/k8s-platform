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

	model "k8s-platform-backend/internal/ai/domain"
	changemysql "k8s-platform-backend/internal/change/adapters/mysql"
	changeapp "k8s-platform-backend/internal/change/application"
	changedomain "k8s-platform-backend/internal/change/domain"
	changeports "k8s-platform-backend/internal/change/ports"
	legacymodel "k8s-platform-backend/internal/legacy/model"
)

const (
	aiActionTypeRestartWorkload      = "restart_workload"
	aiActionTypeScaleWorkload        = "scale_workload"
	aiActionTypeUpdateWorkloadImage  = "update_workload_image"
	aiActionTypePauseWorkloadRollout = "pause_workload_rollout"
	aiActionTypeRolloutUndo          = "rollout_undo"
	aiActionTypeDeleteWorkload       = "delete_workload"
	aiActionTypeDeleteResource       = "delete_resource"
	aiActionTypeDeletePod            = "delete_pod"
	aiActionTypeCordonNode           = "cordon_node"
	aiActionTypeUncordonNode         = "uncordon_node"
	aiActionTypeDrainNode            = "drain_node"
	aiActionTypeTriggerCronJob       = "trigger_cronjob"
	aiActionTypeSuspendCronJob       = "suspend_cronjob"
	aiActionTypeDeleteCompletedJobs  = "delete_completed_jobs"
	aiActionTypeApplyManifest        = "apply_manifest"
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
	Payload        legacymodel.JSONMap    `json:"payload"`
	Reason         string                 `json:"reason"`
}

type ConfirmAIActionProposalRequest struct {
	ConfirmationText string `json:"confirmation_text"`
	ConfirmRisk      bool   `json:"confirm_risk"`
	OperatorComment  string `json:"operator_comment"`
}

type AIActionExecutionItem struct {
	ID              uint64              `json:"id"`
	ProposalID      uint64              `json:"proposal_id"`
	Status          string              `json:"status"`
	ExecutionNo     int                 `json:"execution_no"`
	OperatorID      uint64              `json:"operator_id"`
	OperatorName    string              `json:"operator_name"`
	CommandSnapshot string              `json:"command_snapshot"`
	Result          legacymodel.JSONMap `json:"result,omitempty"`
	ErrorMessage    string              `json:"error_message,omitempty"`
	StartedAt       *string             `json:"started_at,omitempty"`
	FinishedAt      *string             `json:"finished_at,omitempty"`
	CreatedAt       string              `json:"created_at"`
}

type AIActionProposalItem struct {
	ID                       uint64                  `json:"id"`
	ConversationID           uint64                  `json:"conversation_id"`
	MessageID                *uint64                 `json:"message_id,omitempty"`
	ToolCallID               *uint64                 `json:"tool_call_id,omitempty"`
	ClusterID                uint64                  `json:"cluster_id"`
	ActionType               string                  `json:"action_type"`
	TargetKind               string                  `json:"target_kind"`
	TargetNamespace          string                  `json:"target_namespace"`
	TargetName               string                  `json:"target_name"`
	RiskLevel                string                  `json:"risk_level"`
	ConfirmLevel             string                  `json:"confirm_level"`
	Status                   string                  `json:"status"`
	Title                    string                  `json:"title"`
	Summary                  string                  `json:"summary"`
	Change                   legacymodel.JSONMap     `json:"change,omitempty"`
	CreatedBy                uint64                  `json:"created_by"`
	CreatedByName            string                  `json:"created_by_name"`
	ApprovedBy               *uint64                 `json:"approved_by,omitempty"`
	ApprovedByName           string                  `json:"approved_by_name"`
	ApprovedAt               *string                 `json:"approved_at,omitempty"`
	SecondApprovedBy         *uint64                 `json:"second_approved_by,omitempty"`
	SecondApprovedName       string                  `json:"second_approved_name"`
	SecondApprovedAt         *string                 `json:"second_approved_at,omitempty"`
	RequiredConfirmationText string                  `json:"required_confirmation_text"`
	LatestExecution          *AIActionExecutionItem  `json:"latest_execution,omitempty"`
	Executions               []AIActionExecutionItem `json:"executions,omitempty"`
	CreatedAt                string                  `json:"created_at"`
	UpdatedAt                string                  `json:"updated_at"`
}

type CreateAIActionProposalResult struct {
	ProposalID               uint64               `json:"proposal_id"`
	Status                   string               `json:"status"`
	RiskLevel                string               `json:"risk_level"`
	NeedSecondConfirm        bool                 `json:"need_second_confirm"`
	RequiredConfirmationText string               `json:"required_confirmation_text"`
	Preview                  string               `json:"preview"`
	Diff                     string               `json:"diff"`
	Proposal                 AIActionProposalItem `json:"proposal"`
}

type ConfirmAIActionProposalResult struct {
	ProposalID      uint64                 `json:"proposal_id"`
	ExecutionStatus string                 `json:"execution_status"`
	ResultSummary   string                 `json:"result_summary"`
	Proposal        AIActionProposalItem   `json:"proposal"`
	Execution       *AIActionExecutionItem `json:"execution,omitempty"`
}

type AIActionService struct {
	db            *gorm.DB
	workloadSvc   *WorkloadActionService
	changeService *changeapp.Service
}

func NewAIActionService(db *gorm.DB, workloadSvc *WorkloadActionService) *AIActionService {
	return NewAIActionServiceWithChangeService(db, workloadSvc, changeapp.NewService(changemysql.NewRepository(db)))
}

func NewAIActionServiceWithChangeService(db *gorm.DB, workloadSvc *WorkloadActionService, changeService *changeapp.Service) *AIActionService {
	return &AIActionService{db: db, workloadSvc: workloadSvc, changeService: changeService}
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
	if s.workloadSvc == nil {
		return CreateAIActionProposalResult{}, errors.New("workload action service is required")
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
		ProposalID:               proposalRow.ID,
		Status:                   proposalRow.Status,
		RiskLevel:                proposalRow.RiskLevel,
		NeedSecondConfirm:        proposalRow.ConfirmLevel == "double",
		RequiredConfirmationText: buildAIActionConfirmationText(proposalRow.ID),
		Preview:                  preview,
		Diff:                     diffText,
		Proposal:                 item,
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
	if s.workloadSvc == nil {
		return ConfirmAIActionProposalResult{}, errors.New("workload action service is required")
	}
	if clusterID == 0 || proposalID == 0 {
		return ConfirmAIActionProposalResult{}, ErrWithMessage(ErrInvalidParams, "提案参数无效")
	}
	if s.changeService == nil {
		return ConfirmAIActionProposalResult{}, errors.New("change application service is required")
	}

	confirmation, err := s.changeService.Confirm(ctx, changeapp.ConfirmCommand{
		ClusterID: clusterID, ProposalID: proposalID, ActorID: userID, ActorName: username,
		RiskAccepted: req.ConfirmRisk, ConfirmationText: req.ConfirmationText,
	})
	if err != nil {
		return ConfirmAIActionProposalResult{}, mapChangeConfirmationError(err)
	}

	if confirmation.Decision == changedomain.DecisionRecordFirstApproval {
		item, err := s.GetProposal(ctx, clusterID, proposalID)
		if err != nil {
			return ConfirmAIActionProposalResult{}, err
		}
		return ConfirmAIActionProposalResult{
			ProposalID:      proposalID,
			ExecutionStatus: "waiting_second_confirm",
			ResultSummary:   "已完成首次确认，等待第二位审批人确认后执行",
			Proposal:        item,
		}, nil
	}

	if confirmation.Decision != changedomain.DecisionExecute {
		return ConfirmAIActionProposalResult{}, ErrWithMessage(ErrConflict, "该提案当前状态不允许确认")
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
		payload = legacymodel.JSONMap{}
	}
	prepared, err := s.workloadSvc.PrepareProposal(ctx, PrepareWorkloadActionRequest{
		ClusterID:  clusterID,
		ActionType: req.ProposalType,
		Target: WorkloadActionTarget{
			Kind:      target.Kind,
			Namespace: target.Namespace,
			Name:      target.Name,
		},
		Payload: payload,
		Reason:  strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return model.AIActionProposal{}, "", "", err
	}

	row := model.AIActionProposal{
		ConversationID:  req.ConversationID,
		MessageID:       req.MessageID,
		ClusterID:       clusterID,
		ActionType:      prepared.ActionType,
		TargetKind:      prepared.Target.Kind,
		TargetNamespace: prepared.Target.Namespace,
		TargetName:      prepared.Target.Name,
		RiskLevel:       prepared.RiskLevel,
		ConfirmLevel:    prepared.ConfirmLevel,
		Status:          "pending_confirm",
		Title:           prepared.Title,
		Summary:         prepared.Summary,
		ChangeJSON:      model.JSONMap(prepared.Change),
		CreatedBy:       userID,
		CreatedByName:   strings.TrimSpace(username),
	}
	return row, prepared.Preview, prepared.Diff, nil
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
		"status": "executing",
	}
	if proposal.ConfirmLevel == "double" {
		approveUpdates["second_approved_by"] = userID
		approveUpdates["second_approved_name"] = strings.TrimSpace(username)
		approveUpdates["second_approved_at"] = &startedAt
	} else if proposal.ApprovedBy == nil {
		approveUpdates["approved_by"] = userID
		approveUpdates["approved_by_name"] = strings.TrimSpace(username)
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
	executionRow.ResultJSON = model.JSONMap(resultJSON)
	executionRow.FinishedAt = &finishedAt
	return executionRow, summary, nil
}

func (s *AIActionService) runProposalAction(
	ctx context.Context,
	proposal model.AIActionProposal,
) (legacymodel.JSONMap, string, error) {
	result, err := s.workloadSvc.ExecuteProposalAction(ctx, ExecuteWorkloadActionRequest{
		ClusterID:     proposal.ClusterID,
		ActionType:    proposal.ActionType,
		Target:        WorkloadActionTarget{Kind: proposal.TargetKind, Namespace: proposal.TargetNamespace, Name: proposal.TargetName},
		Change:        legacymodel.JSONMap(proposal.ChangeJSON),
		ProposalTitle: proposal.Title,
		CreatedBy:     proposal.CreatedBy,
		CreatedByName: proposal.CreatedByName,
	})
	if err != nil {
		return nil, "", err
	}
	return result.Result, result.Summary, nil
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
		ID:                       row.ID,
		ConversationID:           row.ConversationID,
		MessageID:                row.MessageID,
		ToolCallID:               row.ToolCallID,
		ClusterID:                row.ClusterID,
		ActionType:               row.ActionType,
		TargetKind:               row.TargetKind,
		TargetNamespace:          row.TargetNamespace,
		TargetName:               row.TargetName,
		RiskLevel:                row.RiskLevel,
		ConfirmLevel:             row.ConfirmLevel,
		Status:                   row.Status,
		Title:                    row.Title,
		Summary:                  row.Summary,
		Change:                   legacymodel.JSONMap(row.ChangeJSON),
		CreatedBy:                row.CreatedBy,
		CreatedByName:            row.CreatedByName,
		ApprovedBy:               row.ApprovedBy,
		ApprovedByName:           row.ApprovedByName,
		ApprovedAt:               approvedAt,
		SecondApprovedBy:         row.SecondApprovedBy,
		SecondApprovedName:       row.SecondApprovedName,
		SecondApprovedAt:         secondApprovedAt,
		RequiredConfirmationText: buildAIActionConfirmationText(row.ID),
		LatestExecution:          latestExecution,
		Executions:               executionItems,
		CreatedAt:                row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                row.UpdatedAt.UTC().Format(time.RFC3339),
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
		Result:          legacymodel.JSONMap(row.ResultJSON),
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
	case aiActionTypeUpdateWorkloadImage:
		return aiActionTypeUpdateWorkloadImage
	case aiActionTypePauseWorkloadRollout:
		return aiActionTypePauseWorkloadRollout
	case aiActionTypeRolloutUndo:
		return aiActionTypeRolloutUndo
	case aiActionTypeDeleteWorkload:
		return aiActionTypeDeleteWorkload
	case aiActionTypeDeleteResource:
		return aiActionTypeDeleteResource
	case aiActionTypeDeletePod:
		return aiActionTypeDeletePod
	case aiActionTypeCordonNode:
		return aiActionTypeCordonNode
	case aiActionTypeUncordonNode:
		return aiActionTypeUncordonNode
	case aiActionTypeDrainNode:
		return aiActionTypeDrainNode
	case aiActionTypeTriggerCronJob:
		return aiActionTypeTriggerCronJob
	case aiActionTypeSuspendCronJob:
		return aiActionTypeSuspendCronJob
	case aiActionTypeDeleteCompletedJobs:
		return aiActionTypeDeleteCompletedJobs
	case aiActionTypeApplyManifest:
		return aiActionTypeApplyManifest
	default:
		return ""
	}
}

func normalizeAIActionTarget(target AIActionTargetResource) AIActionTargetResource {
	kind := strings.TrimSpace(target.Kind)
	switch strings.ToLower(kind) {
	case "namespace":
		kind = "Namespace"
	case "node":
		kind = "Node"
	case "pod":
		kind = "Pod"
	case "deployment":
		kind = "Deployment"
	case "statefulset":
		kind = "StatefulSet"
	case "daemonset":
		kind = "DaemonSet"
	case "replicaset":
		kind = "ReplicaSet"
	case "service":
		kind = "Service"
	case "ingress":
		kind = "Ingress"
	case "configmap":
		kind = "ConfigMap"
	case "secret":
		kind = "Secret"
	case "persistentvolumeclaim", "pvc":
		kind = "PersistentVolumeClaim"
	case "persistentvolume", "pv":
		kind = "PersistentVolume"
	case "storageclass":
		kind = "StorageClass"
	case "job":
		kind = "Job"
	case "cronjob":
		kind = "CronJob"
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

func aiActionPayload(change legacymodel.JSONMap) legacymodel.JSONMap {
	if change == nil {
		return legacymodel.JSONMap{}
	}
	if payload, ok := change["payload"].(map[string]any); ok && payload != nil {
		return legacymodel.JSONMap(payload)
	}
	return legacymodel.JSONMap{}
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

func validateAIActionConfirmation(proposal model.AIActionProposal, req ConfirmAIActionProposalRequest) error {
	err := changedomain.ValidateConfirmation(proposal.ID, changedomain.Confirmation{RiskAccepted: req.ConfirmRisk, Text: req.ConfirmationText})
	return mapChangeConfirmationError(err)
}

func buildAIActionConfirmationText(proposalID uint64) string {
	return changedomain.ConfirmationText(proposalID)
}

func mapChangeConfirmationError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, changedomain.ErrAlreadyExecuted):
		return ErrWithMessage(ErrConflict, "该提案已执行，无需重复确认")
	case errors.Is(err, changedomain.ErrNotConfirmable):
		return ErrWithMessage(ErrConflict, "该提案当前状态不允许确认")
	case errors.Is(err, changedomain.ErrSameApprover):
		return ErrWithMessage(ErrConflict, "双人确认提案需要由其他成员完成第二次确认")
	case errors.Is(err, changedomain.ErrRiskNotAccepted):
		return ErrWithMessage(ErrInvalidParams, "请先确认已知晓本次变更风险")
	case errors.Is(err, changedomain.ErrInvalidConfirmation):
		return ErrWithMessage(ErrInvalidParams, "确认短语不正确")
	case errors.Is(err, changeports.ErrProposalNotFound):
		return ErrNotFound
	case errors.Is(err, changeports.ErrConcurrentChange):
		return ErrWithMessage(ErrConflict, "提案已被其他请求更新，请刷新后重试")
	case errors.Is(err, changeapp.ErrInvalidCommand):
		return ErrWithMessage(ErrInvalidParams, "提案确认参数无效")
	default:
		return err
	}
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
