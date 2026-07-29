package application

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/ai/domain"
)

type ActionExecutionItem struct {
	ID              uint64         `json:"id"`
	ProposalID      uint64         `json:"proposal_id"`
	Status          string         `json:"status"`
	ExecutionNo     int            `json:"execution_no"`
	OperatorID      uint64         `json:"operator_id"`
	OperatorName    string         `json:"operator_name"`
	CommandSnapshot string         `json:"command_snapshot"`
	Result          domain.JSONMap `json:"result,omitempty"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	StartedAt       *string        `json:"started_at,omitempty"`
	FinishedAt      *string        `json:"finished_at,omitempty"`
	CreatedAt       string         `json:"created_at"`
}

type ActionProposalItem struct {
	ID                       uint64                `json:"id"`
	ConversationID           uint64                `json:"conversation_id"`
	MessageID                *uint64               `json:"message_id,omitempty"`
	ToolCallID               *uint64               `json:"tool_call_id,omitempty"`
	ClusterID                uint64                `json:"cluster_id"`
	ActionType               string                `json:"action_type"`
	TargetKind               string                `json:"target_kind"`
	TargetNamespace          string                `json:"target_namespace"`
	TargetName               string                `json:"target_name"`
	RiskLevel                string                `json:"risk_level"`
	ConfirmLevel             string                `json:"confirm_level"`
	Status                   string                `json:"status"`
	Title                    string                `json:"title"`
	Summary                  string                `json:"summary"`
	Change                   domain.JSONMap        `json:"change,omitempty"`
	CreatedBy                uint64                `json:"created_by"`
	CreatedByName            string                `json:"created_by_name"`
	ApprovedBy               *uint64               `json:"approved_by,omitempty"`
	ApprovedByName           string                `json:"approved_by_name"`
	ApprovedAt               *string               `json:"approved_at,omitempty"`
	SecondApprovedBy         *uint64               `json:"second_approved_by,omitempty"`
	SecondApprovedName       string                `json:"second_approved_name"`
	SecondApprovedAt         *string               `json:"second_approved_at,omitempty"`
	RequiredConfirmationText string                `json:"required_confirmation_text"`
	LatestExecution          *ActionExecutionItem  `json:"latest_execution,omitempty"`
	Executions               []ActionExecutionItem `json:"executions,omitempty"`
	CreatedAt                string                `json:"created_at"`
	UpdatedAt                string                `json:"updated_at"`
}

type ActionProjectionService struct {
	db               *gorm.DB
	confirmationText func(uint64) string
}

func NewActionProjectionService(db *gorm.DB, confirmationText func(uint64) string) *ActionProjectionService {
	return &ActionProjectionService{db: db, confirmationText: confirmationText}
}

func (s *ActionProjectionService) ListConversation(ctx context.Context, conversationID uint64) ([]ActionProposalItem, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db is required")
	}
	if conversationID == 0 {
		return nil, ErrorWithMessage(ErrInvalidParams, "会话 ID 无效")
	}
	var proposals []domain.AIActionProposal
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("created_at DESC, id DESC").Find(&proposals).Error; err != nil {
		return nil, err
	}
	executionMap, err := s.listExecutions(ctx, proposals)
	if err != nil {
		return nil, err
	}
	items := make([]ActionProposalItem, 0, len(proposals))
	for _, row := range proposals {
		items = append(items, BuildActionProposalItem(row, executionMap[row.ID], s.confirmationText))
	}
	return items, nil
}

func (s *ActionProjectionService) listExecutions(ctx context.Context, proposals []domain.AIActionProposal) (map[uint64][]domain.AIActionExecution, error) {
	result := make(map[uint64][]domain.AIActionExecution, len(proposals))
	if len(proposals) == 0 {
		return result, nil
	}
	ids := make([]uint64, 0, len(proposals))
	for _, proposal := range proposals {
		ids = append(ids, proposal.ID)
	}
	var rows []domain.AIActionExecution
	if err := s.db.WithContext(ctx).Where("proposal_id IN ?", ids).Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ProposalID] = append(result[row.ProposalID], row)
	}
	return result, nil
}

func BuildActionProposalItem(row domain.AIActionProposal, executions []domain.AIActionExecution, confirmationText func(uint64) string) ActionProposalItem {
	var approvedAt, secondApprovedAt *string
	if row.ApprovedAt != nil {
		value := row.ApprovedAt.UTC().Format(time.RFC3339)
		approvedAt = &value
	}
	if row.SecondApprovedAt != nil {
		value := row.SecondApprovedAt.UTC().Format(time.RFC3339)
		secondApprovedAt = &value
	}
	items := make([]ActionExecutionItem, 0, len(executions))
	for _, execution := range executions {
		items = append(items, BuildActionExecutionItem(execution))
	}
	var latest *ActionExecutionItem
	if len(items) > 0 {
		value := items[0]
		latest = &value
	}
	confirmation := ""
	if confirmationText != nil {
		confirmation = confirmationText(row.ID)
	}
	return ActionProposalItem{
		ID: row.ID, ConversationID: row.ConversationID, MessageID: row.MessageID, ToolCallID: row.ToolCallID, ClusterID: row.ClusterID,
		ActionType: row.ActionType, TargetKind: row.TargetKind, TargetNamespace: row.TargetNamespace, TargetName: row.TargetName,
		RiskLevel: row.RiskLevel, ConfirmLevel: row.ConfirmLevel, Status: row.Status, Title: row.Title, Summary: row.Summary,
		Change: row.ChangeJSON, CreatedBy: row.CreatedBy, CreatedByName: row.CreatedByName, ApprovedBy: row.ApprovedBy,
		ApprovedByName: row.ApprovedByName, ApprovedAt: approvedAt, SecondApprovedBy: row.SecondApprovedBy,
		SecondApprovedName: row.SecondApprovedName, SecondApprovedAt: secondApprovedAt, RequiredConfirmationText: confirmation,
		LatestExecution: latest, Executions: items, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func BuildActionExecutionItem(row domain.AIActionExecution) ActionExecutionItem {
	var startedAt, finishedAt *string
	if row.StartedAt != nil {
		value := row.StartedAt.UTC().Format(time.RFC3339)
		startedAt = &value
	}
	if row.FinishedAt != nil {
		value := row.FinishedAt.UTC().Format(time.RFC3339)
		finishedAt = &value
	}
	return ActionExecutionItem{ID: row.ID, ProposalID: row.ProposalID, Status: row.Status, ExecutionNo: row.ExecutionNo,
		OperatorID: row.OperatorID, OperatorName: row.OperatorName, CommandSnapshot: row.CommandSnapshot, Result: row.ResultJSON,
		ErrorMessage: row.ErrorMessage, StartedAt: startedAt, FinishedAt: finishedAt, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339)}
}
