package domain

import (
	"errors"
	"testing"
	"time"
)

func TestIncidentExecuteEnforcesLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	incident := Incident{ID: 7, Status: StatusOpen, Version: 1}

	entry, err := incident.Execute(CommandAcknowledge, 1, "", Actor{ID: 9, Name: "operator"}, now)
	if err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if incident.Status != StatusAcknowledged || incident.Version != 2 {
		t.Fatalf("unexpected aggregate: %+v", incident)
	}
	if incident.AssigneeID == nil || *incident.AssigneeID != 9 || entry.Type != "acknowledge" {
		t.Fatalf("operator was not recorded: %+v %+v", incident, entry)
	}

	if _, err := incident.Execute(CommandStartExecution, 2, "", Actor{}, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestIncidentExecuteRejectsStaleVersion(t *testing.T) {
	incident := Incident{Status: StatusOpen, Version: 3}
	if _, err := incident.Execute(CommandAcknowledge, 2, "", Actor{}, time.Now()); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestIncidentResolveRequiresVerificationOrReason(t *testing.T) {
	incident := Incident{Status: StatusVerifying, Version: 1}
	if _, err := incident.Execute(CommandResolve, 1, "", Actor{}, time.Now()); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if _, err := incident.Execute(CommandResolve, 1, "manual verification passed", Actor{}, time.Now()); err != nil {
		t.Fatalf("manual resolution: %v", err)
	}
	if incident.VerifiedAt == nil || incident.ResolvedAt == nil {
		t.Fatalf("manual verification evidence was not recorded: %+v", incident)
	}
}
