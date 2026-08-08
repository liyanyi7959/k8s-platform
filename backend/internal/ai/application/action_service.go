package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s-platform-backend/internal/ai/domain"
	"k8s-platform-backend/internal/ai/ports"
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
	ClusterID, ProposalID, ActorID uint64
	ActorName                      string
	RiskAccepted                   bool
	ConfirmationText               string
	// IdempotencyKey 可选。非空时 Change 域据此识别重复确认，重放返回快照而不重复执行。
	IdempotencyKey string
}
type ActionConfirmationDecision string

const (
	ActionFirstApproval ActionConfirmationDecision = "first_approval"
	ActionExecute       ActionConfirmationDecision = "execute"
	// ActionReplayed 表示幂等键命中的重复确认，此前已登记，不应再次执行。
	ActionReplayed ActionConfirmationDecision = "replayed"
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

// ActionService owns the AI proposal and execution lifecycle. Persistence is
// supplied by a port so transaction and ORM details stay in the adapter.
type ActionService struct {
	repository   ports.ActionRepository
	executor     ActionExecutorPort
	confirmation ActionConfirmationPort
	projections  *ActionProjectionService
}

func NewActionService(repository ports.ActionRepository, executor ActionExecutorPort, confirmation ActionConfirmationPort) *ActionService {
	return &ActionService{repository: repository, executor: executor, confirmation: confirmation, projections: NewActionProjectionService(repository, ActionConfirmationText)}
}
func ActionConfirmationText(proposalID uint64) string {
	return fmt.Sprintf("confirm-proposal-%d", proposalID)
}

func (s *ActionService) CreateProposal(ctx context.Context, clusterID, userID uint64, username string, req CreateActionProposalRequest) (ActionProposalResult, error) {
	if s == nil || s.repository == nil {
		return ActionProposalResult{}, errors.New("action repository is required")
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
	if _, err := s.repository.FindConversationInCluster(ctx, req.ConversationID, clusterID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ActionProposalResult{}, ErrNotFound
		}
		return ActionProposalResult{}, err
	}
	prepared, err := s.executor.PrepareAction(ctx, clusterID, req)
	if err != nil {
		return ActionProposalResult{}, err
	}
	row := domain.AIActionProposal{ConversationID: req.ConversationID, MessageID: req.MessageID, ClusterID: clusterID, ActionType: prepared.ActionType, TargetKind: prepared.Target.Kind, TargetNamespace: prepared.Target.Namespace, TargetName: prepared.Target.Name, RiskLevel: prepared.RiskLevel, ConfirmLevel: prepared.ConfirmLevel, Status: "pending_confirm", Title: prepared.Title, Summary: prepared.Summary, ChangeJSON: prepared.Change, CreatedBy: userID, CreatedByName: strings.TrimSpace(username)}
	if err := s.repository.CreateActionProposal(ctx, &row, actionProposalMessage(row.ConversationID, userID, row.Title, row.Summary), time.Now().UTC()); err != nil {
		return ActionProposalResult{}, err
	}
	proposal, err := s.GetProposal(ctx, clusterID, row.ID)
	if err != nil {
		return ActionProposalResult{}, err
	}
	return ActionProposalResult{ProposalID: row.ID, Status: row.Status, RiskLevel: row.RiskLevel, NeedSecondConfirm: row.ConfirmLevel == "double", RequiredConfirmationText: ActionConfirmationText(row.ID), Preview: prepared.Preview, Diff: prepared.Diff, Proposal: proposal}, nil
}

func (s *ActionService) ConfirmProposal(ctx context.Context, clusterID, proposalID, userID uint64, username string, req ConfirmActionProposalRequest) (ActionConfirmationResult, error) {
	if s == nil || s.repository == nil {
		return ActionConfirmationResult{}, errors.New("action repository is required")
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
	decision, err := s.confirmation.ConfirmAction(ctx, ActionConfirmationRequest{ClusterID: clusterID, ProposalID: proposalID, ActorID: userID, ActorName: username, RiskAccepted: req.ConfirmRisk, ConfirmationText: req.ConfirmationText, IdempotencyKey: strings.TrimSpace(req.IdempotencyKey)})
	if err != nil {
		return ActionConfirmationResult{}, err
	}
	if decision == ActionReplayed {
		// 幂等键命中的重复确认：返回已有提案与执行快照，不再登记或执行。
		proposal, err := s.GetProposal(ctx, clusterID, proposalID)
		if err != nil {
			return ActionConfirmationResult{}, err
		}
		executions, err := s.repository.ListActionExecutions(ctx, []uint64{proposalID})
		if err != nil {
			return ActionConfirmationResult{}, err
		}
		if len(executions) == 0 {
			return ActionConfirmationResult{ProposalID: proposalID, ExecutionStatus: "already_processed", ResultSummary: "本次确认已处理，请刷新查看最新状态", Proposal: proposal}, nil
		}
		item := BuildActionExecutionItem(executions[0])
		return ActionConfirmationResult{ProposalID: proposalID, ExecutionStatus: executions[0].Status, ResultSummary: "本次确认已处理（幂等命中），请刷新查看最新状态", Proposal: proposal, Execution: &item}, nil
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
	proposalRow, err := s.repository.FindActionProposalInCluster(ctx, proposalID, clusterID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
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
	if s == nil || s.repository == nil {
		return ActionProposalItem{}, errors.New("action repository is required")
	}
	if clusterID == 0 || proposalID == 0 {
		return ActionProposalItem{}, ErrorWithMessage(ErrInvalidParams, "提案参数无效")
	}
	proposal, err := s.repository.FindActionProposalInCluster(ctx, proposalID, clusterID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
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
	execution := domain.AIActionExecution{ProposalID: proposal.ID, ConversationID: proposal.ConversationID, ClusterID: proposal.ClusterID, ExecutionNo: executionNo, Status: "running", StartedAt: &startedAt, OperatorID: userID, OperatorName: strings.TrimSpace(username), CommandSnapshot: actionExecutionSnapshot(proposal, operatorComment)}
	if err := s.repository.CreateActionExecution(ctx, &execution); err != nil {
		return domain.AIActionExecution{}, "", err
	}
	approval := ports.ActionApproval{ProposalID: proposal.ID}
	if proposal.ConfirmLevel == "double" {
		approval.SecondApprovedBy, approval.SecondApprovedName, approval.SecondApprovedAt = &userID, strings.TrimSpace(username), &startedAt
	} else if proposal.ApprovedBy == nil {
		approval.ApprovedBy, approval.ApprovedByName, approval.ApprovedAt = &userID, strings.TrimSpace(username), &startedAt
	}
	if err := s.repository.MarkActionExecuting(ctx, proposal.ID, approval); err != nil {
		return domain.AIActionExecution{}, "", err
	}
	result, executeErr := s.executor.ExecuteAction(ctx, proposal)
	finishedAt := time.Now().UTC()
	if executeErr != nil {
		s.finishFailedExecution(ctx, execution.ID, proposal, userID, finishedAt, executeErr)
		return domain.AIActionExecution{}, "", executeErr
	}
	if err := s.repository.CompleteActionExecution(ctx, ports.ActionCompletion{ExecutionID: execution.ID, ProposalID: proposal.ID, ConversationID: proposal.ConversationID, UserID: userID, Title: proposal.Title, Summary: result.Summary, FinishedAt: finishedAt, Result: result.Result}); err != nil {
		return domain.AIActionExecution{}, "", err
	}
	execution.Status = "succeeded"
	execution.ResultJSON = result.Result
	execution.FinishedAt = &finishedAt
	return execution, result.Summary, nil
}
func (s *ActionService) finishFailedExecution(ctx context.Context, executionID uint64, proposal domain.AIActionProposal, userID uint64, finishedAt time.Time, executionErr error) {
	_ = s.repository.FailActionExecution(ctx, ports.ActionFailure{ExecutionID: executionID, ProposalID: proposal.ID, ConversationID: proposal.ConversationID, UserID: userID, Title: proposal.Title, Message: actionErrorMessage(executionErr), FinishedAt: finishedAt})
}
func (s *ActionService) nextExecutionNo(ctx context.Context, proposalID uint64) (int, error) {
	return s.repository.NextActionExecutionNumber(ctx, proposalID)
}
func actionExecutionSnapshot(proposal domain.AIActionProposal, operatorComment string) string {
	raw, err := json.Marshal(map[string]any{"action_type": proposal.ActionType, "target_kind": proposal.TargetKind, "target_namespace": proposal.TargetNamespace, "target_name": proposal.TargetName, "change": proposal.ChangeJSON, "operator_comment": strings.TrimSpace(operatorComment)})
	if err != nil {
		return ""
	}
	return string(raw)
}
func actionProposalMessage(conversationID, userID uint64, title, summary string) domain.AIMessage {
	return domain.AIMessage{ConversationID: conversationID, Role: "system", MessageType: "proposal", Content: fmt.Sprintf("已生成变更提案：%s\n%s", strings.TrimSpace(title), strings.TrimSpace(summary)), Status: "created", CreatedBy: userID}
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
