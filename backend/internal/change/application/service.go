package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"k8s-platform-backend/internal/change/domain"
	"k8s-platform-backend/internal/change/ports"
)

var ErrInvalidCommand = errors.New("invalid change confirmation command")

type ConfirmCommand struct {
	ClusterID        uint64
	ProposalID       uint64
	ActorID          uint64
	ActorName        string
	RiskAccepted     bool
	ConfirmationText string
	// IdempotencyKey 可选。为空时跳过幂等检查（兼容现有调用方）；非空时
	// 重复提交同一键直接返回已处理结果，避免重复登记执行。
	IdempotencyKey string
}

type ConfirmResult struct {
	Decision domain.ConfirmationDecision
	Proposal domain.Proposal
	// Execution 在决策为 execute 且已登记执行时非空。
	Execution *domain.Execution
	// AlreadyProcessed 表示幂等键命中，本次确认此前已登记，无需再次执行。
	AlreadyProcessed bool
}

// CompleteCommand 回填已确认执行的成功结果。
type CompleteCommand struct {
	ProposalID uint64
	ResultJSON string
	Summary    string
}

// FailCommand 回填已确认执行的失败结果。
type FailCommand struct {
	ProposalID uint64
	Message    string
}

type Service struct {
	repository ports.ProposalRepository
	now        func() time.Time
}

func NewService(repository ports.ProposalRepository) *Service {
	return &Service{repository: repository, now: func() time.Time { return time.Now().UTC() }}
}

// Confirm serializes approval decisions per proposal. Reserving execution in the
// same transaction prevents two concurrent confirmations from executing twice.
func (service *Service) Confirm(ctx context.Context, command ConfirmCommand) (ConfirmResult, error) {
	if service == nil || service.repository == nil || command.ClusterID == 0 || command.ProposalID == 0 || command.ActorID == 0 {
		return ConfirmResult{}, ErrInvalidCommand
	}

	var result ConfirmResult
	err := service.repository.Transaction(ctx, func(repository ports.ProposalRepository) error {
		// 幂等键命中：本次确认此前已登记，直接返回已处理结果。
		if key := strings.TrimSpace(command.IdempotencyKey); key != "" {
			execution, err := repository.FindExecutionByKey(ctx, command.ProposalID, key)
			if err == nil {
				result = ConfirmResult{Decision: decisionForStatus(execution.Status), Proposal: proposalSnapshot(ctx, repository, command.ClusterID, command.ProposalID), Execution: &execution, AlreadyProcessed: true}
				return nil
			}
			if !errors.Is(err, ports.ErrExecutionNotFound) {
				return err
			}
		}

		proposal, err := repository.FindForUpdate(ctx, command.ClusterID, command.ProposalID)
		if err != nil {
			return err
		}
		decision, err := proposal.Confirm(
			domain.Actor{ID: command.ActorID},
			domain.Confirmation{RiskAccepted: command.RiskAccepted, Text: command.ConfirmationText},
		)
		if err != nil {
			return err
		}

		approval := ports.Approval{ActorID: command.ActorID, ActorName: strings.TrimSpace(command.ActorName), At: service.now()}
		switch decision {
		case domain.DecisionRecordFirstApproval:
			err = repository.RecordFirstApproval(ctx, proposal, approval)
			if err == nil {
				err = repository.AppendRevision(ctx, domain.Revision{ProposalID: command.ProposalID, FromStatus: proposal.Status, ToStatus: domain.StatusApproved, ActorID: approval.ActorID, ActorName: approval.ActorName, Decision: "first_approved", CreatedAt: approval.At})
			}
		case domain.DecisionExecute:
			err = repository.ReserveExecution(ctx, proposal, approval)
			if err == nil {
				err = repository.AppendRevision(ctx, domain.Revision{ProposalID: command.ProposalID, FromStatus: proposal.Status, ToStatus: domain.StatusExecuting, ActorID: approval.ActorID, ActorName: approval.ActorName, Decision: "execution_reserved", CreatedAt: approval.At})
			}
			if err == nil {
				execution, createErr := repository.CreateExecution(ctx, newExecution(command, proposal, approval, service.now()))
				if errors.Is(createErr, ports.ErrIdempotencyConflict) {
					execution, createErr = repository.FindExecutionByKey(ctx, command.ProposalID, strings.TrimSpace(command.IdempotencyKey))
				}
				if createErr != nil {
					err = createErr
				} else {
					result.Execution = &execution
				}
			}
		default:
			err = domain.ErrNotConfirmable
		}
		if err != nil {
			return err
		}
		result = ConfirmResult{Decision: decision, Proposal: proposal, Execution: result.Execution}
		return nil
	})
	return result, err
}

// CompleteExecution 将提案的活跃执行回填为成功终态，并追加审计修订。
// 无活跃执行时静默忽略（幂等），便于编排层统一回填。
func (service *Service) CompleteExecution(ctx context.Context, command CompleteCommand) error {
	if service == nil || service.repository == nil || command.ProposalID == 0 {
		return ErrInvalidCommand
	}
	finishedAt := service.now()
	return service.repository.Transaction(ctx, func(repository ports.ProposalRepository) error {
		execution, err := repository.FindActiveExecution(ctx, command.ProposalID)
		if err != nil {
			if errors.Is(err, ports.ErrExecutionNotFound) {
				return nil
			}
			return err
		}
		updated, err := execution.Complete(finishedAt)
		if err != nil {
			return err
		}
		if err := repository.CompleteExecution(ctx, updated, command.ResultJSON); err != nil {
			return err
		}
		return repository.AppendRevision(ctx, domain.Revision{ProposalID: command.ProposalID, FromStatus: domain.StatusExecuting, ToStatus: domain.StatusSucceeded, ActorID: updated.OperatorID, ActorName: updated.OperatorName, Decision: "execution_completed", CreatedAt: finishedAt})
	})
}

// FailExecution 将提案的活跃执行回填为失败终态，并追加审计修订。
func (service *Service) FailExecution(ctx context.Context, command FailCommand) error {
	if service == nil || service.repository == nil || command.ProposalID == 0 {
		return ErrInvalidCommand
	}
	finishedAt := service.now()
	return service.repository.Transaction(ctx, func(repository ports.ProposalRepository) error {
		execution, err := repository.FindActiveExecution(ctx, command.ProposalID)
		if err != nil {
			if errors.Is(err, ports.ErrExecutionNotFound) {
				return nil
			}
			return err
		}
		updated, err := execution.Fail(finishedAt)
		if err != nil {
			return err
		}
		if err := repository.FailExecution(ctx, updated, command.Message); err != nil {
			return err
		}
		return repository.AppendRevision(ctx, domain.Revision{ProposalID: command.ProposalID, FromStatus: domain.StatusExecuting, ToStatus: domain.StatusFailed, ActorID: updated.OperatorID, ActorName: updated.OperatorName, Decision: "execution_failed", CreatedAt: finishedAt})
	})
}

// RecoverTimedOutExecutions 将超过 timeout 仍未完成的执行标记为失败，作为
// 进程内 worker 的兜底回收，避免僵尸执行永久占用状态。只做状态与审计，
// 不执行任何实际操作。
func (service *Service) RecoverTimedOutExecutions(ctx context.Context, timeout time.Duration) error {
	if service == nil || service.repository == nil {
		return ErrInvalidCommand
	}
	executions, err := service.repository.ListTimedOutExecutions(ctx, service.now().Add(-timeout), 50)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		if err := service.FailExecution(ctx, FailCommand{ProposalID: execution.ProposalID, Message: "execution timed out"}); err != nil {
			return err
		}
	}
	return nil
}

// newExecution 从确认命令组装执行登记。当前确认流程同一提案仅登记一次
// 执行（状态机保证 executing 只进入一次），执行号固定从 1 开始。
func newExecution(command ConfirmCommand, proposal domain.Proposal, approval ports.Approval, startedAt time.Time) domain.Execution {
	return domain.Execution{
		ProposalID:     command.ProposalID,
		ClusterID:      command.ClusterID,
		ExecutionNo:    1,
		Status:         domain.ExecutionRunning,
		IdempotencyKey: strings.TrimSpace(command.IdempotencyKey),
		OperatorID:     approval.ActorID,
		OperatorName:   approval.ActorName,
		StartedAt:      &startedAt,
	}
}

func decisionForStatus(status domain.ExecutionStatus) domain.ConfirmationDecision {
	switch status {
	case domain.ExecutionSucceeded, domain.ExecutionFailed, domain.ExecutionRunning:
		return domain.DecisionExecute
	default:
		return domain.DecisionRecordFirstApproval
	}
}

// proposalSnapshot 在幂等命中路径下读取提案快照，避免状态机重复推进。
func proposalSnapshot(ctx context.Context, repository ports.ProposalRepository, clusterID, proposalID uint64) domain.Proposal {
	proposal, err := repository.FindForUpdate(ctx, clusterID, proposalID)
	if err != nil {
		return domain.Proposal{ID: proposalID}
	}
	return proposal
}
