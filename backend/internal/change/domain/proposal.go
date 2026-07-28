package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrAlreadyExecuted     = errors.New("change proposal already executed")
	ErrNotConfirmable      = errors.New("change proposal is not confirmable")
	ErrRiskNotAccepted     = errors.New("change risk was not accepted")
	ErrInvalidConfirmation = errors.New("invalid change confirmation")
	ErrSameApprover        = errors.New("second approval requires another actor")
)

type Status string

const (
	StatusPendingConfirm Status = "pending_confirm"
	StatusApproved       Status = "approved"
	StatusExecuting      Status = "executing"
	StatusSucceeded      Status = "succeeded"
	StatusFailed         Status = "failed"
	StatusCancelled      Status = "cancelled"
)

type ConfirmLevel string

const (
	ConfirmSingle ConfirmLevel = "single"
	ConfirmDouble ConfirmLevel = "double"
)

type ConfirmationDecision string

const (
	DecisionRecordFirstApproval ConfirmationDecision = "record_first_approval"
	DecisionExecute             ConfirmationDecision = "execute"
)

type Actor struct {
	ID uint64
}

type Confirmation struct {
	RiskAccepted bool
	Text         string
}

type Proposal struct {
	ID           uint64
	Status       Status
	ConfirmLevel ConfirmLevel
	ApprovedBy   *uint64
}

func (proposal Proposal) Confirm(actor Actor, confirmation Confirmation) (ConfirmationDecision, error) {
	switch proposal.Status {
	case StatusExecuting, StatusSucceeded:
		return "", ErrAlreadyExecuted
	case StatusFailed, StatusCancelled:
		return "", ErrNotConfirmable
	case StatusPendingConfirm, StatusApproved:
	default:
		return "", ErrNotConfirmable
	}
	if err := ValidateConfirmation(proposal.ID, confirmation); err != nil {
		return "", err
	}
	if proposal.ConfirmLevel != ConfirmDouble {
		return DecisionExecute, nil
	}
	if proposal.ApprovedBy == nil {
		return DecisionRecordFirstApproval, nil
	}
	if *proposal.ApprovedBy == actor.ID {
		return "", ErrSameApprover
	}
	return DecisionExecute, nil
}

func ValidateConfirmation(proposalID uint64, confirmation Confirmation) error {
	if !confirmation.RiskAccepted {
		return ErrRiskNotAccepted
	}
	if strings.TrimSpace(confirmation.Text) != ConfirmationText(proposalID) {
		return ErrInvalidConfirmation
	}
	return nil
}

func ConfirmationText(proposalID uint64) string {
	return fmt.Sprintf("confirm-proposal-%d", proposalID)
}
