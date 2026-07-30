package ai

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	aimysql "k8s-platform-backend/internal/ai/adapters/mysql"
	aiapp "k8s-platform-backend/internal/ai/application"
	model "k8s-platform-backend/internal/ai/domain"
	changemysql "k8s-platform-backend/internal/change/adapters/mysql"
	changeapp "k8s-platform-backend/internal/change/application"
	changedomain "k8s-platform-backend/internal/change/domain"
	changeports "k8s-platform-backend/internal/change/ports"
	kopsapp "k8s-platform-backend/internal/kops/application"
	kopsdomain "k8s-platform-backend/internal/kops/domain"
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

// ActionRuntime adapts Kops execution and Change confirmation ports to the
// AI action use case. Proposal orchestration stays in aiapp.ActionService;
// this adapter is deliberately the only place that knows both runtime ports.
type ActionRuntime struct{ core *aiapp.ActionService }

func NewActionRuntime(db *gorm.DB, workloadActions *kopsapp.ActionProposalService) *ActionRuntime {
	return NewActionRuntimeWithChangeService(db, workloadActions, changeapp.NewService(changemysql.NewRepository(db)))
}

func NewActionRuntimeWithChangeService(db *gorm.DB, workloadActions *kopsapp.ActionProposalService, changeService *changeapp.Service) *ActionRuntime {
	return &ActionRuntime{core: aiapp.NewActionService(aimysql.NewRepository(db), workloadActionExecutor{service: workloadActions}, changeConfirmationAdapter{service: changeService})}
}

func (r *ActionRuntime) CreateProposal(ctx context.Context, clusterID, userID uint64, username string, req aiapp.CreateActionProposalRequest) (aiapp.ActionProposalResult, error) {
	if r == nil || r.core == nil {
		return aiapp.ActionProposalResult{}, errors.New("action application service is required")
	}
	result, err := r.core.CreateProposal(ctx, clusterID, userID, username, req)
	return result, mapActionRuntimeError(err)
}

func (r *ActionRuntime) ConfirmProposal(ctx context.Context, clusterID, proposalID, userID uint64, username string, req aiapp.ConfirmActionProposalRequest) (aiapp.ActionConfirmationResult, error) {
	if r == nil || r.core == nil {
		return aiapp.ActionConfirmationResult{}, errors.New("action application service is required")
	}
	result, err := r.core.ConfirmProposal(ctx, clusterID, proposalID, userID, username, req)
	return result, mapActionRuntimeError(err)
}

func (r *ActionRuntime) GetProposal(ctx context.Context, clusterID, proposalID uint64) (aiapp.ActionProposalItem, error) {
	if r == nil || r.core == nil {
		return aiapp.ActionProposalItem{}, errors.New("action application service is required")
	}
	result, err := r.core.GetProposal(ctx, clusterID, proposalID)
	return result, mapActionRuntimeError(err)
}

func (r *ActionRuntime) ListConversationProposals(ctx context.Context, conversationID uint64) ([]aiapp.ActionProposalItem, error) {
	if r == nil || r.core == nil {
		return nil, errors.New("action application service is required")
	}
	result, err := r.core.ListConversationProposals(ctx, conversationID)
	return result, mapActionRuntimeError(err)
}

type workloadActionExecutor struct {
	service *kopsapp.ActionProposalService
}

func (adapter workloadActionExecutor) PrepareAction(ctx context.Context, clusterID uint64, req aiapp.CreateActionProposalRequest) (aiapp.PreparedAction, error) {
	if adapter.service == nil {
		return aiapp.PreparedAction{}, errors.New("workload action service is required")
	}
	target := aiapp.NormalizeActionTarget(req.TargetResource)
	prepared, err := adapter.service.Prepare(ctx, kopsapp.ActionProposalPrepareRequest{
		ClusterID:  clusterID,
		ActionType: req.ProposalType,
		Target: kopsapp.ActionProposalTarget{
			Kind: target.Kind, Namespace: target.Namespace, Name: target.Name,
		},
		Payload: req.Payload,
		Reason:  strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return aiapp.PreparedAction{}, err
	}
	return aiapp.PreparedAction{
		ActionType: prepared.ActionType,
		Target: aiapp.ActionTargetResource{
			Kind: prepared.Target.Kind, Namespace: prepared.Target.Namespace, Name: prepared.Target.Name,
		},
		RiskLevel: prepared.RiskLevel, ConfirmLevel: prepared.ConfirmLevel,
		Title: prepared.Title, Summary: prepared.Summary, Change: model.JSONMap(prepared.Change),
		Preview: prepared.Preview, Diff: prepared.Diff,
	}, nil
}

func (adapter workloadActionExecutor) ExecuteAction(ctx context.Context, proposal model.AIActionProposal) (aiapp.ActionExecutionResult, error) {
	if adapter.service == nil {
		return aiapp.ActionExecutionResult{}, errors.New("workload action service is required")
	}
	result, err := adapter.service.Execute(ctx, kopsapp.ActionProposalExecutionRequest{
		ClusterID:  proposal.ClusterID,
		ActionType: proposal.ActionType,
		Target: kopsapp.ActionProposalTarget{
			Kind: proposal.TargetKind, Namespace: proposal.TargetNamespace, Name: proposal.TargetName,
		},
		Change:        kopsdomain.JSONMap(proposal.ChangeJSON),
		ProposalTitle: proposal.Title,
		CreatedBy:     proposal.CreatedBy,
		CreatedByName: proposal.CreatedByName,
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

func mapActionRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	for _, candidate := range []struct{ source, target error }{
		{aiapp.ErrInvalidParams, aiapp.ErrInvalidParams},
		{aiapp.ErrNotFound, aiapp.ErrNotFound},
		{aiapp.ErrConflict, aiapp.ErrConflict},
		{aiapp.ErrCrypto, aiapp.ErrCrypto},
		{kopsapp.ErrInvalidParams, aiapp.ErrInvalidParams},
		{kopsapp.ErrNotFound, aiapp.ErrNotFound},
		{kopsapp.ErrConflict, aiapp.ErrConflict},
	} {
		if errors.Is(err, candidate.source) {
			if candidate.source == candidate.target {
				return err
			}
			if message, ok := actionRuntimeUserMessage(err); ok {
				return aiapp.ErrorWithMessage(candidate.target, message)
			}
			return candidate.target
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

func actionRuntimeUserMessage(err error) (string, bool) {
	var carrier interface{ UserMessage() string }
	if errors.As(err, &carrier) && carrier != nil {
		if message := strings.TrimSpace(carrier.UserMessage()); message != "" {
			return message, true
		}
	}
	return "", false
}
