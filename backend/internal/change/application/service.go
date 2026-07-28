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
}

type ConfirmResult struct {
	Decision domain.ConfirmationDecision
	Proposal domain.Proposal
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
		case domain.DecisionExecute:
			err = repository.ReserveExecution(ctx, proposal, approval)
		default:
			err = domain.ErrNotConfirmable
		}
		if err != nil {
			return err
		}
		result = ConfirmResult{Decision: decision, Proposal: proposal}
		return nil
	})
	return result, err
}
