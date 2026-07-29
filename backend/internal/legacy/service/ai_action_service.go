package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	aiapp "k8s-platform-backend/internal/ai/application"
	model "k8s-platform-backend/internal/ai/domain"
	changemysql "k8s-platform-backend/internal/change/adapters/mysql"
	changeapp "k8s-platform-backend/internal/change/application"
	changedomain "k8s-platform-backend/internal/change/domain"
	changeports "k8s-platform-backend/internal/change/ports"
	kopsdomain "k8s-platform-backend/internal/kops/domain"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	aiActionTypeRestartWorkload      = aiapp.ActionTypeRestartWorkload
	aiActionTypeScaleWorkload        = aiapp.ActionTypeScaleWorkload
	aiActionTypeUpdateWorkloadImage  = aiapp.ActionTypeUpdateWorkloadImage
	aiActionTypePauseWorkloadRollout = aiapp.ActionTypePauseWorkloadRollout
	aiActionTypeRolloutUndo          = aiapp.ActionTypeRolloutUndo
	aiActionTypeDeleteWorkload       = aiapp.ActionTypeDeleteWorkload
	aiActionTypeDeleteResource       = aiapp.ActionTypeDeleteResource
	aiActionTypeDeletePod            = aiapp.ActionTypeDeletePod
	aiActionTypeCordonNode           = aiapp.ActionTypeCordonNode
	aiActionTypeUncordonNode         = aiapp.ActionTypeUncordonNode
	aiActionTypeDrainNode            = aiapp.ActionTypeDrainNode
	aiActionTypeTriggerCronJob       = aiapp.ActionTypeTriggerCronJob
	aiActionTypeSuspendCronJob       = aiapp.ActionTypeSuspendCronJob
	aiActionTypeDeleteCompletedJobs  = aiapp.ActionTypeDeleteCompletedJobs
	aiActionTypeApplyManifest        = aiapp.ActionTypeApplyManifest
)

type AIActionTargetResource = aiapp.ActionTargetResource

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

type AIActionExecutionItem = aiapp.ActionExecutionItem
type AIActionProposalItem = aiapp.ActionProposalItem
type CreateAIActionProposalResult = aiapp.ActionProposalResult
type ConfirmAIActionProposalResult = aiapp.ActionConfirmationResult

// AIActionService is a compatibility facade for callers that have not moved
// to the AI application package yet. Proposal orchestration belongs to
// aiapp.ActionService; this type only adapts the legacy Kops and Change ports.
type AIActionService struct{ core *aiapp.ActionService }

func NewAIActionService(db *gorm.DB, workloadSvc *WorkloadActionService) *AIActionService {
	return NewAIActionServiceWithChangeService(db, workloadSvc, changeapp.NewService(changemysql.NewRepository(db)))
}

func NewAIActionServiceWithChangeService(db *gorm.DB, workloadSvc *WorkloadActionService, changeService *changeapp.Service) *AIActionService {
	return &AIActionService{core: aiapp.NewActionService(db, workloadActionExecutor{service: workloadSvc}, changeConfirmationAdapter{service: changeService})}
}

func (s *AIActionService) CreateProposal(ctx context.Context, clusterID, userID uint64, username string, req CreateAIActionProposalRequest) (CreateAIActionProposalResult, error) {
	if s == nil || s.core == nil {
		return CreateAIActionProposalResult{}, errors.New("action application service is required")
	}
	result, err := s.core.CreateProposal(ctx, clusterID, userID, username, aiapp.CreateActionProposalRequest{
		ConversationID: req.ConversationID, MessageID: req.MessageID, ProposalType: req.ProposalType,
		TargetResource: req.TargetResource, Payload: map[string]any(req.Payload), Reason: req.Reason,
	})
	return result, mapActionApplicationError(err)
}

func (s *AIActionService) ConfirmProposal(ctx context.Context, clusterID, proposalID, userID uint64, username string, req ConfirmAIActionProposalRequest) (ConfirmAIActionProposalResult, error) {
	if s == nil || s.core == nil {
		return ConfirmAIActionProposalResult{}, errors.New("action application service is required")
	}
	result, err := s.core.ConfirmProposal(ctx, clusterID, proposalID, userID, username, aiapp.ConfirmActionProposalRequest{
		ConfirmationText: req.ConfirmationText, ConfirmRisk: req.ConfirmRisk, OperatorComment: req.OperatorComment,
	})
	return result, mapActionApplicationError(err)
}

func (s *AIActionService) GetProposal(ctx context.Context, clusterID, proposalID uint64) (AIActionProposalItem, error) {
	if s == nil || s.core == nil {
		return AIActionProposalItem{}, errors.New("action application service is required")
	}
	result, err := s.core.GetProposal(ctx, clusterID, proposalID)
	return result, mapActionApplicationError(err)
}

func (s *AIActionService) ListConversationProposals(ctx context.Context, conversationID uint64) ([]AIActionProposalItem, error) {
	if s == nil || s.core == nil {
		return nil, errors.New("action application service is required")
	}
	result, err := s.core.ListConversationProposals(ctx, conversationID)
	return result, mapActionApplicationError(err)
}

func listConversationActionProposals(ctx context.Context, db *gorm.DB, conversationID uint64) ([]AIActionProposalItem, error) {
	return aiapp.NewActionProjectionService(db, aiapp.ActionConfirmationText).ListConversation(ctx, conversationID)
}

type workloadActionExecutor struct{ service *WorkloadActionService }

func (adapter workloadActionExecutor) PrepareAction(ctx context.Context, clusterID uint64, req aiapp.CreateActionProposalRequest) (aiapp.PreparedAction, error) {
	if adapter.service == nil {
		return aiapp.PreparedAction{}, errors.New("workload action service is required")
	}
	target := normalizeAIActionTarget(req.TargetResource)
	prepared, err := adapter.service.PrepareProposal(ctx, PrepareWorkloadActionRequest{
		ClusterID: clusterID, ActionType: req.ProposalType,
		Target:  WorkloadActionTarget{Kind: target.Kind, Namespace: target.Namespace, Name: target.Name},
		Payload: kopsdomain.JSONMap(req.Payload), Reason: strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return aiapp.PreparedAction{}, err
	}
	return aiapp.PreparedAction{
		ActionType: prepared.ActionType, Target: aiapp.ActionTargetResource{Kind: prepared.Target.Kind, Namespace: prepared.Target.Namespace, Name: prepared.Target.Name},
		RiskLevel: prepared.RiskLevel, ConfirmLevel: prepared.ConfirmLevel, Title: prepared.Title, Summary: prepared.Summary,
		Change: model.JSONMap(prepared.Change), Preview: prepared.Preview, Diff: prepared.Diff,
	}, nil
}

func (adapter workloadActionExecutor) ExecuteAction(ctx context.Context, proposal model.AIActionProposal) (aiapp.ActionExecutionResult, error) {
	if adapter.service == nil {
		return aiapp.ActionExecutionResult{}, errors.New("workload action service is required")
	}
	result, err := adapter.service.ExecuteProposalAction(ctx, ExecuteWorkloadActionRequest{
		ClusterID: proposal.ClusterID, ActionType: proposal.ActionType,
		Target: WorkloadActionTarget{Kind: proposal.TargetKind, Namespace: proposal.TargetNamespace, Name: proposal.TargetName},
		Change: kopsdomain.JSONMap(proposal.ChangeJSON), ProposalTitle: proposal.Title, CreatedBy: proposal.CreatedBy, CreatedByName: proposal.CreatedByName,
	})
	if err != nil {
		return aiapp.ActionExecutionResult{}, err
	}
	return aiapp.ActionExecutionResult{Result: model.JSONMap(result.Result), Summary: result.Summary}, nil
}

type changeConfirmationAdapter struct{ service *changeapp.Service }

func (adapter changeConfirmationAdapter) ConfirmAction(ctx context.Context, req aiapp.ActionConfirmationRequest) (aiapp.ActionConfirmationDecision, error) {
	if adapter.service == nil {
		return "", errors.New("change application service is required")
	}
	result, err := adapter.service.Confirm(ctx, changeapp.ConfirmCommand{
		ClusterID: req.ClusterID, ProposalID: req.ProposalID, ActorID: req.ActorID, ActorName: req.ActorName,
		RiskAccepted: req.RiskAccepted, ConfirmationText: req.ConfirmationText,
	})
	if err != nil {
		return "", mapChangeConfirmationApplicationError(err)
	}
	switch result.Decision {
	case changedomain.DecisionRecordFirstApproval:
		return aiapp.ActionFirstApproval, nil
	case changedomain.DecisionExecute:
		return aiapp.ActionExecute, nil
	default:
		return "", aiapp.ErrorWithMessage(aiapp.ErrConflict, "该提案当前状态不允许确认")
	}
}

func mapActionApplicationError(err error) error {
	if err == nil {
		return nil
	}
	for _, candidate := range []struct{ application, legacy error }{
		{aiapp.ErrInvalidParams, ErrInvalidParams}, {aiapp.ErrNotFound, ErrNotFound}, {aiapp.ErrConflict, ErrConflict}, {aiapp.ErrCrypto, ErrCrypto},
	} {
		if errors.Is(err, candidate.application) {
			if message, ok := UserMessage(err); ok {
				return ErrWithMessage(candidate.legacy, message)
			}
			return candidate.legacy
		}
	}
	return err
}

func mapChangeConfirmationApplicationError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, changedomain.ErrAlreadyExecuted):
		return aiapp.ErrorWithMessage(aiapp.ErrConflict, "该提案已执行，无需重复确认")
	case errors.Is(err, changedomain.ErrNotConfirmable):
		return aiapp.ErrorWithMessage(aiapp.ErrConflict, "该提案当前状态不允许确认")
	case errors.Is(err, changedomain.ErrSameApprover):
		return aiapp.ErrorWithMessage(aiapp.ErrConflict, "双人确认提案需要由其他成员完成第二次确认")
	case errors.Is(err, changedomain.ErrRiskNotAccepted):
		return aiapp.ErrorWithMessage(aiapp.ErrInvalidParams, "请先确认已知晓本次变更风险")
	case errors.Is(err, changedomain.ErrInvalidConfirmation):
		return aiapp.ErrorWithMessage(aiapp.ErrInvalidParams, "确认短语不正确")
	case errors.Is(err, changeports.ErrProposalNotFound):
		return aiapp.ErrNotFound
	case errors.Is(err, changeports.ErrConcurrentChange):
		return aiapp.ErrorWithMessage(aiapp.ErrConflict, "提案已被其他请求更新，请刷新后重试")
	case errors.Is(err, changeapp.ErrInvalidCommand):
		return aiapp.ErrorWithMessage(aiapp.ErrInvalidParams, "提案确认参数无效")
	default:
		return err
	}
}

// The helpers below keep legacy Kops callers source-compatible while their
// runtime implementation is migrated. The policy itself resides in aiapp.
func normalizeAIActionType(v string) string { return aiapp.NormalizeActionType(v) }
func normalizeAIActionTarget(target AIActionTargetResource) AIActionTargetResource {
	return aiapp.NormalizeActionTarget(target)
}
func aiActionWorkloadGVR(kind string) (schema.GroupVersionResource, bool) {
	return aiapp.ActionWorkloadGVR(kind)
}

func aiActionPayload(change model.JSONMap) model.JSONMap {
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

func jsonIntValue(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case int8:
		return int(number), true
	case int16:
		return int(number), true
	case int32:
		return int(number), true
	case int64:
		return int(number), true
	case uint:
		return int(number), true
	case uint8:
		return int(number), true
	case uint16:
		return int(number), true
	case uint32:
		return int(number), true
	case uint64:
		return int(number), true
	case float32:
		return int(number), true
	case float64:
		return int(number), true
	default:
		return 0, false
	}
}

func jsonNestedInt(obj map[string]any, keys ...string) int {
	if len(keys) == 0 {
		return 0
	}
	current := obj
	for _, key := range keys[:len(keys)-1] {
		next, _ := current[key].(map[string]any)
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

func ptrAIActionExecutionItem(item AIActionExecutionItem) *AIActionExecutionItem { return &item }

func validateAIActionConfirmation(proposal model.AIActionProposal, req ConfirmAIActionProposalRequest) error {
	err := changedomain.ValidateConfirmation(proposal.ID, changedomain.Confirmation{RiskAccepted: req.ConfirmRisk, Text: req.ConfirmationText})
	return mapChangeConfirmationError(err)
}

func buildAIActionConfirmationText(proposalID uint64) string {
	return aiapp.ActionConfirmationText(proposalID)
}

func mapChangeConfirmationError(err error) error {
	return mapActionApplicationError(mapChangeConfirmationApplicationError(err))
}

func aiActionString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
