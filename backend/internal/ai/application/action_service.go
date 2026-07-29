package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/ai/domain"
)

// PreparedAction is the domain-neutral output of an action runtime while a
// proposal is being assembled. Kops and other execution mechanisms implement
// this contract outside the AI application layer.
type PreparedAction struct {
	ActionType   string
	Target       ActionTargetResource
	RiskLevel    string
	ConfirmLevel string
	Title        string
	Summary      string
	Change       domain.JSONMap
	Preview      string
	Diff         string
}

type ActionExecutionResult struct {
	Result  domain.JSONMap
	Summary string
}

type ActionExecutorPort interface {
	PrepareAction(context.Context, uint64, CreateActionProposalRequest) (PreparedAction, error)
	ExecuteAction(context.Context, domain.AIActionProposal) (ActionExecutionResult, error)
}

type ActionConfirmationRequest struct {
	ClusterID        uint64
	ProposalID       uint64
	ActorID          uint64
	ActorName        string
	RiskAccepted     bool
	ConfirmationText string
}

type ActionConfirmationDecision string

const (
	ActionFirstApproval ActionConfirmationDecision = "first_approval"
	ActionExecute       ActionConfirmationDecision = "execute"
)

type ActionConfirmationPort interface {
	ConfirmAction(context.Context, ActionConfirmationRequest) (ActionConfirmationDecision, error)
}

type ActionProposalResult struct {
	ProposalID               uint64             `json:"proposal_id"`
	Status                   string             `json:"status"`
	RiskLevel                string             `json:"risk_level"`
	NeedSecondConfirm        bool               `json:"need_second_confirm"`
	RequiredConfirmationText string             `json:"required_confirmation_text"`
	Preview                  string             `json:"preview"`
	Diff                     string             `json:"diff"`
	Proposal                 ActionProposalItem `json:"proposal"`
}

type ActionConfirmationResult struct {
	ProposalID      uint64               `json:"proposal_id"`
	ExecutionStatus string               `json:"execution_status"`
	ResultSummary   string               `json:"result_summary"`
	Proposal        ActionProposalItem   `json:"proposal"`
	Execution       *ActionExecutionItem `json:"execution,omitempty"`
}

// ActionService owns the AI proposal and execution lifecycle. The actual
// cluster work and change confirmation aggregate are injected through ports,
// so this use case remains independent from Kops and Change internals.
type ActionService struct {
	db           *gorm.DB
	executor     ActionExecutorPort
	confirmation ActionConfirmationPort
	projections  *ActionProjectionService
}

func NewActionService(db *gorm.DB, executor ActionExecutorPort, confirmation ActionConfirmationPort) *ActionService {
	return &ActionService{
		db:           db,
		executor:     executor,
		confirmation: confirmation,
		projections:  NewActionProjectionService(db, ActionConfirmationText),
	}
}

func ActionConfirmationText(proposalID uint64) string {
	return fmt.Sprintf("confirm-proposal-%d", proposalID)
}

func (s *ActionService) CreateProposal(ctx context.Context, clusterID, userID uint64, username string, req CreateActionProposalRequest) (ActionProposalResult, error) {
	if s == nil || s.db == nil {
		return ActionProposalResult{}, errors.New("db is required")
	}
	if s.executor == nil {
		return ActionProposalResult{}, errors.New("action executor is required")
	}
	if clusterID == 0 || req.ConversationID == 0 {
		return ActionProposalResult{}, ErrorWithMessage(ErrInvalidParams, "集群或会话参数无效")
	}
	if NormalizeActionType(req.ProposalType) == "" {
		return ActionProposalResult{}, ErrorWithMessage(ErrInvalidParams, "当前动作类型暂不支持")
	}

	var conversation domain.AIConversation
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ? AND cluster_id = ?", req.ConversationID, clusterID).First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ActionProposalResult{}, ErrNotFound
		}
		return ActionProposalResult{}, err
	}

	prepared, err := s.executor.PrepareAction(ctx, clusterID, req)
	if err != nil {
		return ActionProposalResult{}, err
	}
	row := domain.AIActionProposal{
		ConversationID: req.ConversationID, MessageID: req.MessageID, ClusterID: clusterID,
		ActionType: prepared.ActionType, TargetKind: prepared.Target.Kind, TargetNamespace: prepared.Target.Namespace,
		TargetName: prepared.Target.Name, RiskLevel: prepared.RiskLevel, ConfirmLevel: prepared.ConfirmLevel,
		Status: "pending_confirm", Title: prepared.Title, Summary: prepared.Summary, ChangeJSON: prepared.Change,
		CreatedBy: userID, CreatedByName: strings.TrimSpace(username),
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := appendActionProposalMessage(tx, row.ConversationID, userID, row.Title, row.Summary); err != nil {
			return err
		}
		return tx.Model(&domain.AIConversation{}).Where("id = ?", row.ConversationID).Updates(map[string]any{
			"status": "waiting_confirm", "last_message_at": time.Now().UTC(),
		}).Error
	}); err != nil {
		return ActionProposalResult{}, err
	}

	proposal, err := s.GetProposal(ctx, clusterID, row.ID)
	if err != nil {
		return ActionProposalResult{}, err
	}
	return ActionProposalResult{
		ProposalID: row.ID, Status: row.Status, RiskLevel: row.RiskLevel, NeedSecondConfirm: row.ConfirmLevel == "double",
		RequiredConfirmationText: ActionConfirmationText(row.ID), Preview: prepared.Preview, Diff: prepared.Diff, Proposal: proposal,
	}, nil
}

func (s *ActionService) ConfirmProposal(ctx context.Context, clusterID, proposalID, userID uint64, username string, req ConfirmActionProposalRequest) (ActionConfirmationResult, error) {
	if s == nil || s.db == nil {
		return ActionConfirmationResult{}, errors.New("db is required")
	}
	if s.executor == nil {
		return ActionConfirmationResult{}, errors.New("action executor is required")
	}
	if s.confirmation == nil {
		return ActionConfirmationResult{}, errors.New("action confirmation service is required")
	}
	if clusterID == 0 || proposalID == 0 {
		return ActionConfirmationResult{}, ErrorWithMessage(ErrInvalidParams, "提案参数无效")
	}

	decision, err := s.confirmation.ConfirmAction(ctx, ActionConfirmationRequest{
		ClusterID: clusterID, ProposalID: proposalID, ActorID: userID, ActorName: username,
		RiskAccepted: req.ConfirmRisk, ConfirmationText: req.ConfirmationText,
	})
	if err != nil {
		return ActionConfirmationResult{}, err
	}
	if decision == ActionFirstApproval {
		proposal, err := s.GetProposal(ctx, clusterID, proposalID)
		if err != nil {
			return ActionConfirmationResult{}, err
		}
		return ActionConfirmationResult{ProposalID: proposalID, ExecutionStatus: "waiting_second_confirm", ResultSummary: "已完成首次确认，等待第二位审批人确认后执行", Proposal: proposal}, nil
	}
	if decision != ActionExecute {
		return ActionConfirmationResult{}, ErrorWithMessage(ErrConflict, "该提案当前状态不允许确认")
	}

	var proposalRow domain.AIActionProposal
	if err := s.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", proposalID, clusterID).First(&proposalRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ActionConfirmationResult{}, ErrNotFound
		}
		return ActionConfirmationResult{}, err
	}
	execution, summary, err := s.executeProposal(ctx, proposalRow, userID, username, strings.TrimSpace(req.OperatorComment))
	if err != nil {
		return ActionConfirmationResult{}, err
	}
	proposal, err := s.GetProposal(ctx, clusterID, proposalID)
	if err != nil {
		return ActionConfirmationResult{}, err
	}
	item := BuildActionExecutionItem(execution)
	return ActionConfirmationResult{ProposalID: proposalID, ExecutionStatus: execution.Status, ResultSummary: summary, Proposal: proposal, Execution: &item}, nil
}

func (s *ActionService) GetProposal(ctx context.Context, clusterID, proposalID uint64) (ActionProposalItem, error) {
	if s == nil || s.db == nil {
		return ActionProposalItem{}, errors.New("db is required")
	}
	if clusterID == 0 || proposalID == 0 {
		return ActionProposalItem{}, ErrorWithMessage(ErrInvalidParams, "提案参数无效")
	}
	var proposal domain.AIActionProposal
	if err := s.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", proposalID, clusterID).First(&proposal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ActionProposalItem{}, ErrNotFound
		}
		return ActionProposalItem{}, err
	}
	items, err := s.ListConversationProposals(ctx, proposal.ConversationID)
	if err != nil {
		return ActionProposalItem{}, err
	}
	for _, item := range items {
		if item.ID == proposalID {
			return item, nil
		}
	}
	return ActionProposalItem{}, ErrNotFound
}

func (s *ActionService) ListConversationProposals(ctx context.Context, conversationID uint64) ([]ActionProposalItem, error) {
	if s == nil || s.projections == nil {
		return nil, errors.New("action projection service is required")
	}
	return s.projections.ListConversation(ctx, conversationID)
}

func (s *ActionService) executeProposal(ctx context.Context, proposal domain.AIActionProposal, userID uint64, username, operatorComment string) (domain.AIActionExecution, string, error) {
	executionNo, err := s.nextExecutionNo(ctx, proposal.ID)
	if err != nil {
		return domain.AIActionExecution{}, "", err
	}
	startedAt := time.Now().UTC()
	execution := domain.AIActionExecution{
		ProposalID: proposal.ID, ConversationID: proposal.ConversationID, ClusterID: proposal.ClusterID,
		ExecutionNo: executionNo, Status: "running", StartedAt: &startedAt, OperatorID: userID,
		OperatorName: strings.TrimSpace(username), CommandSnapshot: actionExecutionSnapshot(proposal, operatorComment),
	}
	if err := s.db.WithContext(ctx).Create(&execution).Error; err != nil {
		return domain.AIActionExecution{}, "", err
	}

	proposalUpdates := map[string]any{"status": "executing"}
	if proposal.ConfirmLevel == "double" {
		proposalUpdates["second_approved_by"] = userID
		proposalUpdates["second_approved_name"] = strings.TrimSpace(username)
		proposalUpdates["second_approved_at"] = &startedAt
	} else if proposal.ApprovedBy == nil {
		proposalUpdates["approved_by"] = userID
		proposalUpdates["approved_by_name"] = strings.TrimSpace(username)
		proposalUpdates["approved_at"] = &startedAt
	}
	if err := s.db.WithContext(ctx).Model(&domain.AIActionProposal{}).Where("id = ?", proposal.ID).Updates(proposalUpdates).Error; err != nil {
		return domain.AIActionExecution{}, "", err
	}

	result, executeErr := s.executor.ExecuteAction(ctx, proposal)
	finishedAt := time.Now().UTC()
	if executeErr != nil {
		s.finishFailedExecution(ctx, execution.ID, proposal, userID, finishedAt, executeErr)
		return domain.AIActionExecution{}, "", executeErr
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.AIActionExecution{}).Where("id = ?", execution.ID).Updates(map[string]any{
			"status": "succeeded", "finished_at": &finishedAt, "result_json": result.Result, "error_message": "",
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.AIActionProposal{}).Where("id = ?", proposal.ID).Update("status", "succeeded").Error; err != nil {
			return err
		}
		if err := appendActionExecutionMessage(tx, proposal.ConversationID, userID, proposal.Title, "succeeded", result.Summary); err != nil {
			return err
		}
		return tx.Model(&domain.AIConversation{}).Where("id = ?", proposal.ConversationID).Updates(map[string]any{
			"status": "open", "last_message_at": &finishedAt,
		}).Error
	}); err != nil {
		return domain.AIActionExecution{}, "", err
	}
	execution.Status = "succeeded"
	execution.ResultJSON = result.Result
	execution.FinishedAt = &finishedAt
	return execution, result.Summary, nil
}

// finishFailedExecution intentionally preserves the original action failure:
// errors while recording the failure should not obscure an executor error.
func (s *ActionService) finishFailedExecution(ctx context.Context, executionID uint64, proposal domain.AIActionProposal, userID uint64, finishedAt time.Time, executionErr error) {
	message := actionErrorMessage(executionErr)
	_ = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_ = tx.Model(&domain.AIActionExecution{}).Where("id = ?", executionID).Updates(map[string]any{
			"status": "failed", "finished_at": &finishedAt, "error_message": message,
		}).Error
		_ = tx.Model(&domain.AIActionProposal{}).Where("id = ?", proposal.ID).Update("status", "failed").Error
		_ = appendActionExecutionMessage(tx, proposal.ConversationID, userID, proposal.Title, "failed", message)
		_ = tx.Model(&domain.AIConversation{}).Where("id = ?", proposal.ConversationID).Updates(map[string]any{
			"status": "open", "last_message_at": &finishedAt,
		}).Error
		return nil
	})
}

func (s *ActionService) nextExecutionNo(ctx context.Context, proposalID uint64) (int, error) {
	var maxNo int
	if err := s.db.WithContext(ctx).Model(&domain.AIActionExecution{}).Select("COALESCE(MAX(execution_no), 0)").Where("proposal_id = ?", proposalID).Scan(&maxNo).Error; err != nil {
		return 0, err
	}
	return maxNo + 1, nil
}

func actionExecutionSnapshot(proposal domain.AIActionProposal, operatorComment string) string {
	raw, err := json.Marshal(map[string]any{
		"action_type": proposal.ActionType, "target_kind": proposal.TargetKind, "target_namespace": proposal.TargetNamespace,
		"target_name": proposal.TargetName, "change": proposal.ChangeJSON, "operator_comment": strings.TrimSpace(operatorComment),
	})
	if err != nil {
		return ""
	}
	return string(raw)
}

func appendActionProposalMessage(tx *gorm.DB, conversationID, userID uint64, title, summary string) error {
	return tx.Create(&domain.AIMessage{
		ConversationID: conversationID, Role: "system", MessageType: "proposal",
		Content: fmt.Sprintf("已生成变更提案：%s\n%s", strings.TrimSpace(title), strings.TrimSpace(summary)),
		Status:  "created", CreatedBy: userID,
	}).Error
}

func appendActionExecutionMessage(tx *gorm.DB, conversationID, userID uint64, title, status, summary string) error {
	return tx.Create(&domain.AIMessage{
		ConversationID: conversationID, Role: "system", MessageType: "action_execution",
		Content: fmt.Sprintf("变更提案执行%s：%s\n%s", ActionExecutionStatusLabel(status), strings.TrimSpace(title), strings.TrimSpace(summary)),
		Status:  "created", CreatedBy: userID,
	}).Error
}

func actionErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var userMessage interface{ UserMessage() string }
	if errors.As(err, &userMessage) {
		if message := strings.TrimSpace(userMessage.UserMessage()); message != "" {
			return message
		}
	}
	return strings.TrimSpace(err.Error())
}
