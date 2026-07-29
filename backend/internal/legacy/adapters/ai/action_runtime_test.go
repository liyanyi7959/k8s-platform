package ai

import (
	"context"
	"errors"
	"testing"

	aiapp "k8s-platform-backend/internal/ai/application"
	changeapp "k8s-platform-backend/internal/change/application"
	changedomain "k8s-platform-backend/internal/change/domain"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

func TestMapActionRuntimeErrorPreservesAIApplicationError(t *testing.T) {
	source := aiapp.ErrorWithMessage(aiapp.ErrInvalidParams, "proposal scope is invalid")
	err := mapActionRuntimeError(source)
	if !errors.Is(err, aiapp.ErrInvalidParams) {
		t.Fatalf("mapped error = %v, want AI invalid params", err)
	}
	if message, ok := actionRuntimeUserMessage(err); !ok || message != "proposal scope is invalid" {
		t.Fatalf("mapped message = %q, %v", message, ok)
	}
}

func TestMapActionRuntimeErrorTranslatesKopsErrorToAIBoundary(t *testing.T) {
	err := mapActionRuntimeError(kopsapp.ErrWithMessage(kopsapp.ErrConflict, "workload changed"))
	if !errors.Is(err, aiapp.ErrConflict) {
		t.Fatalf("mapped error = %v, want AI conflict", err)
	}
	if message, ok := actionRuntimeUserMessage(err); !ok || message != "workload changed" {
		t.Fatalf("mapped message = %q, %v", message, ok)
	}
}

func TestMapChangeConfirmationApplicationErrorUsesAIErrorVocabulary(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "already executed", err: changedomain.ErrAlreadyExecuted, want: aiapp.ErrConflict},
		{name: "risk not accepted", err: changedomain.ErrRiskNotAccepted, want: aiapp.ErrInvalidParams},
		{name: "invalid command", err: changeapp.ErrInvalidCommand, want: aiapp.ErrInvalidParams},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapChangeConfirmationApplicationError(tt.err); !errors.Is(got, tt.want) {
				t.Fatalf("mapped error = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionRuntimeRequiresCoreService(t *testing.T) {
	var runtime *ActionRuntime
	if _, err := runtime.CreateProposal(context.Background(), 1, 2, "ops", aiapp.CreateActionProposalRequest{}); err == nil {
		t.Fatal("nil runtime should reject proposal creation")
	}
}
