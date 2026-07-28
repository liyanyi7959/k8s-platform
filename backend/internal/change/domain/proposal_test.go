package domain

import (
	"errors"
	"testing"
)

func TestProposalConfirmRequiresTwoDifferentActors(t *testing.T) {
	proposal := Proposal{ID: 42, Status: StatusPendingConfirm, ConfirmLevel: ConfirmDouble}
	confirmation := Confirmation{RiskAccepted: true, Text: ConfirmationText(proposal.ID)}

	decision, err := proposal.Confirm(Actor{ID: 7}, confirmation)
	if err != nil || decision != DecisionRecordFirstApproval {
		t.Fatalf("first approval: decision=%q err=%v", decision, err)
	}

	proposal.Status = StatusApproved
	proposal.ApprovedBy = uint64Ptr(7)
	if _, err := proposal.Confirm(Actor{ID: 7}, confirmation); !errors.Is(err, ErrSameApprover) {
		t.Fatalf("same approver must be rejected: %v", err)
	}
	decision, err = proposal.Confirm(Actor{ID: 8}, confirmation)
	if err != nil || decision != DecisionExecute {
		t.Fatalf("second approval: decision=%q err=%v", decision, err)
	}
}

func TestProposalConfirmRejectsTerminalState(t *testing.T) {
	proposal := Proposal{ID: 42, Status: StatusSucceeded, ConfirmLevel: ConfirmSingle}
	_, err := proposal.Confirm(Actor{ID: 7}, Confirmation{RiskAccepted: true, Text: ConfirmationText(proposal.ID)})
	if !errors.Is(err, ErrAlreadyExecuted) {
		t.Fatalf("expected already executed, got %v", err)
	}
}

func TestValidateConfirmationRequiresRiskAndExactPhrase(t *testing.T) {
	if err := ValidateConfirmation(9, Confirmation{Text: ConfirmationText(9)}); !errors.Is(err, ErrRiskNotAccepted) {
		t.Fatalf("expected risk acceptance error, got %v", err)
	}
	if err := ValidateConfirmation(9, Confirmation{RiskAccepted: true, Text: "wrong"}); !errors.Is(err, ErrInvalidConfirmation) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
}

func uint64Ptr(value uint64) *uint64 { return &value }
