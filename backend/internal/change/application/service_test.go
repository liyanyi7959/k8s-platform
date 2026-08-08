package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"k8s-platform-backend/internal/change/domain"
	"k8s-platform-backend/internal/change/ports"
)

func TestConfirmRecordsFirstApprovalForDoubleConfirmation(t *testing.T) {
	repository := &repositoryStub{proposal: domain.Proposal{ID: 42, Status: domain.StatusPendingConfirm, ConfirmLevel: domain.ConfirmDouble}}
	service := NewService(repository)
	service.now = func() time.Time { return time.Unix(100, 0).UTC() }

	result, err := service.Confirm(context.Background(), validCommand(7))
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != domain.DecisionRecordFirstApproval || repository.firstApproval.ActorID != 7 {
		t.Fatalf("result=%+v approval=%+v", result, repository.firstApproval)
	}
	if repository.executionApproval.ActorID != 0 {
		t.Fatal("execution must not be reserved after the first approval")
	}
	if result.Execution != nil {
		t.Fatal("no execution must be registered after the first approval")
	}
	if len(repository.revisions) != 1 || repository.revisions[0].Decision != "first_approved" {
		t.Fatalf("revisions=%+v", repository.revisions)
	}
}

func TestConfirmReservesExecutionForSecondActor(t *testing.T) {
	firstActor := uint64(7)
	repository := &repositoryStub{proposal: domain.Proposal{ID: 42, Status: domain.StatusApproved, ConfirmLevel: domain.ConfirmDouble, ApprovedBy: &firstActor}}
	service := NewService(repository)

	result, err := service.Confirm(context.Background(), validCommand(8))
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != domain.DecisionExecute || repository.executionApproval.ActorID != 8 {
		t.Fatalf("result=%+v approval=%+v", result, repository.executionApproval)
	}
	if result.Execution == nil || result.Execution.OperatorID != 8 || result.Execution.Status != domain.ExecutionRunning {
		t.Fatalf("execution=%+v", result.Execution)
	}
}

func TestConfirmRejectsSameSecondApproverBeforePersistence(t *testing.T) {
	firstActor := uint64(7)
	repository := &repositoryStub{proposal: domain.Proposal{ID: 42, Status: domain.StatusApproved, ConfirmLevel: domain.ConfirmDouble, ApprovedBy: &firstActor}}
	_, err := NewService(repository).Confirm(context.Background(), validCommand(7))
	if !errors.Is(err, domain.ErrSameApprover) {
		t.Fatalf("expected same approver error, got %v", err)
	}
	if repository.executionApproval.ActorID != 0 {
		t.Fatal("rejected approval must not be persisted")
	}
	if len(repository.executions) != 0 {
		t.Fatal("rejected approval must not register an execution")
	}
}

func TestConfirmReplaysIdempotentConfirmation(t *testing.T) {
	repository := &repositoryStub{
		proposal: domain.Proposal{ID: 42, Status: domain.StatusExecuting, ConfirmLevel: domain.ConfirmSingle},
		executionByKey: map[string]domain.Execution{
			"confirm-42": {ID: 9, ProposalID: 42, Status: domain.ExecutionSucceeded},
		},
	}
	service := NewService(repository)
	service.now = func() time.Time { return time.Unix(100, 0).UTC() }

	command := validCommand(7)
	command.IdempotencyKey = "confirm-42"
	result, err := service.Confirm(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyProcessed || result.Decision != domain.DecisionExecute {
		t.Fatalf("result=%+v", result)
	}
	if result.Execution == nil || result.Execution.Status != domain.ExecutionSucceeded {
		t.Fatalf("execution=%+v", result.Execution)
	}
	if len(repository.executions) != 0 {
		t.Fatal("idempotent replay must not register a new execution")
	}
	if repository.firstApproval.ActorID != 0 || repository.executionApproval.ActorID != 0 {
		t.Fatal("idempotent replay must not mutate approval state")
	}
}

func TestConfirmRegistersExecutionWithIdempotencyKey(t *testing.T) {
	repository := &repositoryStub{proposal: domain.Proposal{ID: 42, Status: domain.StatusPendingConfirm, ConfirmLevel: domain.ConfirmSingle}}
	service := NewService(repository)
	service.now = func() time.Time { return time.Unix(100, 0).UTC() }

	command := validCommand(7)
	command.IdempotencyKey = "confirm-42"
	result, err := service.Confirm(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != domain.DecisionExecute || result.Execution == nil || result.Execution.IdempotencyKey != "confirm-42" {
		t.Fatalf("result=%+v execution=%+v", result, result.Execution)
	}
}

func TestCompleteExecutionMarksSucceeded(t *testing.T) {
	startedAt := time.Unix(100, 0).UTC()
	repository := &repositoryStub{activeExecution: &domain.Execution{ID: 9, ProposalID: 42, Status: domain.ExecutionRunning, OperatorID: 7, OperatorName: "operator", StartedAt: &startedAt}}
	service := NewService(repository)
	service.now = func() time.Time { return time.Unix(200, 0).UTC() }

	err := service.CompleteExecution(context.Background(), CompleteCommand{ProposalID: 42, ResultJSON: `{"ok":true}`})
	if err != nil {
		t.Fatal(err)
	}
	execution := *repository.activeExecution
	if execution.Status != domain.ExecutionSucceeded || execution.FinishedAt == nil || execution.FinishedAt.Unix() != 200 {
		t.Fatalf("execution=%+v", execution)
	}
	if len(repository.revisions) != 1 || repository.revisions[0].ToStatus != domain.StatusSucceeded || repository.revisions[0].Decision != "execution_completed" {
		t.Fatalf("revisions=%+v", repository.revisions)
	}
}

func TestFailExecutionMarksFailed(t *testing.T) {
	startedAt := time.Unix(100, 0).UTC()
	repository := &repositoryStub{activeExecution: &domain.Execution{ID: 9, ProposalID: 42, Status: domain.ExecutionRunning, OperatorID: 7, StartedAt: &startedAt}}
	service := NewService(repository)
	service.now = func() time.Time { return time.Unix(200, 0).UTC() }

	err := service.FailExecution(context.Background(), FailCommand{ProposalID: 42, Message: "boom"})
	if err != nil {
		t.Fatal(err)
	}
	execution := *repository.activeExecution
	if execution.Status != domain.ExecutionFailed || execution.FinishedAt == nil || execution.FinishedAt.Unix() != 200 {
		t.Fatalf("execution=%+v", execution)
	}
	if len(repository.revisions) != 1 || repository.revisions[0].ToStatus != domain.StatusFailed || repository.revisions[0].Decision != "execution_failed" {
		t.Fatalf("revisions=%+v", repository.revisions)
	}
}

func TestCompleteExecutionIgnoresMissingActiveExecution(t *testing.T) {
	repository := &repositoryStub{}
	err := NewService(repository).CompleteExecution(context.Background(), CompleteCommand{ProposalID: 42})
	if err != nil {
		t.Fatalf("missing active execution must be ignored, got %v", err)
	}
}

func TestRecoverTimedOutExecutions(t *testing.T) {
	startedAt := time.Unix(100, 0).UTC()
	repository := &repositoryStub{
		timedOut:        []domain.Execution{{ID: 9, ProposalID: 42, Status: domain.ExecutionRunning, StartedAt: &startedAt}},
		activeExecution: &domain.Execution{ID: 9, ProposalID: 42, Status: domain.ExecutionRunning, OperatorID: 7, StartedAt: &startedAt},
	}
	service := NewService(repository)
	service.now = func() time.Time { return time.Unix(200, 0).UTC() }

	err := service.RecoverTimedOutExecutions(context.Background(), 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if repository.activeExecution.Status != domain.ExecutionFailed {
		t.Fatalf("execution=%+v", repository.activeExecution)
	}
}

func validCommand(actorID uint64) ConfirmCommand {
	return ConfirmCommand{
		ClusterID: 1, ProposalID: 42, ActorID: actorID, ActorName: "operator",
		RiskAccepted: true, ConfirmationText: domain.ConfirmationText(42),
	}
}

type repositoryStub struct {
	proposal          domain.Proposal
	firstApproval     ports.Approval
	executionApproval ports.Approval

	executions      []domain.Execution
	revisions       []domain.Revision
	activeExecution *domain.Execution
	executionByKey  map[string]domain.Execution
	timedOut        []domain.Execution
}

func (repository *repositoryStub) Transaction(_ context.Context, fn func(ports.ProposalRepository) error) error {
	return fn(repository)
}

func (repository *repositoryStub) FindForUpdate(_ context.Context, _, _ uint64) (domain.Proposal, error) {
	return repository.proposal, nil
}

func (repository *repositoryStub) RecordFirstApproval(_ context.Context, _ domain.Proposal, approval ports.Approval) error {
	repository.firstApproval = approval
	return nil
}

func (repository *repositoryStub) ReserveExecution(_ context.Context, _ domain.Proposal, approval ports.Approval) error {
	repository.executionApproval = approval
	return nil
}

func (repository *repositoryStub) AppendRevision(_ context.Context, revision domain.Revision) error {
	repository.revisions = append(repository.revisions, revision)
	return nil
}

func (repository *repositoryStub) CreateExecution(_ context.Context, execution domain.Execution) (domain.Execution, error) {
	if execution.IdempotencyKey != "" {
		if repository.executionByKey == nil {
			repository.executionByKey = make(map[string]domain.Execution)
		}
		if _, exists := repository.executionByKey[execution.IdempotencyKey]; exists {
			return domain.Execution{}, ports.ErrIdempotencyConflict
		}
		repository.executionByKey[execution.IdempotencyKey] = execution
	}
	repository.executions = append(repository.executions, execution)
	return execution, nil
}

func (repository *repositoryStub) FindExecutionByKey(_ context.Context, _ uint64, key string) (domain.Execution, error) {
	if execution, ok := repository.executionByKey[key]; ok {
		return execution, nil
	}
	return domain.Execution{}, ports.ErrExecutionNotFound
}

func (repository *repositoryStub) FindActiveExecution(_ context.Context, _ uint64) (domain.Execution, error) {
	if repository.activeExecution != nil {
		return *repository.activeExecution, nil
	}
	return domain.Execution{}, ports.ErrExecutionNotFound
}

func (repository *repositoryStub) CompleteExecution(_ context.Context, execution domain.Execution, _ string) error {
	repository.activeExecution = &execution
	return nil
}

func (repository *repositoryStub) FailExecution(_ context.Context, execution domain.Execution, _ string) error {
	repository.activeExecution = &execution
	return nil
}

func (repository *repositoryStub) ListTimedOutExecutions(_ context.Context, _ time.Time, _ int) ([]domain.Execution, error) {
	return repository.timedOut, nil
}
