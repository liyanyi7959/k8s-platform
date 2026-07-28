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
