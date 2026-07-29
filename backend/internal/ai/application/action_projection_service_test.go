package application

import (
	"testing"
	"time"

	"k8s-platform-backend/internal/ai/domain"
)

func TestBuildActionProposalItem(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	item := BuildActionProposalItem(domain.AIActionProposal{
		ID: 10, ConversationID: 100, ClusterID: 1, ActionType: "scale_workload", TargetKind: "Deployment",
		TargetNamespace: "default", TargetName: "nginx", RiskLevel: "medium", ConfirmLevel: "single",
		Status: "pending", Title: "Scale to 3", Summary: "scaling up", ChangeJSON: domain.JSONMap{"replicas": 3},
		CreatedBy: 1, CreatedByName: "admin", CreatedAt: now, UpdatedAt: now,
	}, nil, func(id uint64) string { return "confirm-proposal-" + "10" })
	if item.ID != 10 || item.Change["replicas"] != 3 {
		t.Fatalf("proposal item = %#v", item)
	}
	if item.RequiredConfirmationText != "confirm-proposal-10" {
		t.Fatalf("confirmation text = %q", item.RequiredConfirmationText)
	}
}

func TestBuildActionProposalItemUsesLatestExecution(t *testing.T) {
	now := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	item := BuildActionProposalItem(domain.AIActionProposal{ID: 20, CreatedAt: now, UpdatedAt: now}, []domain.AIActionExecution{{
		ID: 1, ProposalID: 20, Status: "succeeded", ExecutionNo: 1, ResultJSON: domain.JSONMap{"pods_restarted": 3}, CreatedAt: now,
	}}, nil)
	if item.LatestExecution == nil || item.LatestExecution.Status != "succeeded" || len(item.Executions) != 1 {
		t.Fatalf("execution projection = %#v", item)
	}
}
