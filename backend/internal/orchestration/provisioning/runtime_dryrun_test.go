package provisioning

import (
	"context"
	"errors"
	"testing"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
)

func TestDryRunPlanInputAssociatesServersWithPlanNodes(t *testing.T) {
	input := dryRunPlanInput(
		model.DeployPlan{ID: 17, Name: "production", ClusterName: "prod-k8s", K8sVersion: "v1.30.2", CNIType: "calico"},
		[]model.DeployPlanNode{{ServerID: 2, Role: "worker", SortOrder: 20}, {ServerID: 1, Role: "master", SortOrder: 10}, {ServerID: 3, Role: "worker", SortOrder: 30}},
		[]model.DeployServer{{ID: 1, Name: "control-1", IP: "10.0.0.10"}, {ID: 2, Name: "worker-1", IP: "10.0.0.20"}},
	)
	if input.PlanID != 17 || input.PlanName != "production" || input.ClusterName != "prod-k8s" || input.K8sVersion != "v1.30.2" || input.CNIType != "calico" {
		t.Fatalf("unexpected plan input: %#v", input)
	}
	if len(input.Nodes) != 3 {
		t.Fatalf("node count = %d, want 3", len(input.Nodes))
	}
	if input.Nodes[0].ServerName != "worker-1" || input.Nodes[1].ServerName != "control-1" {
		t.Fatalf("resolved server names = %#v", input.Nodes)
	}
	if input.Nodes[2].ServerName != "" || input.Nodes[2].IP != "" {
		t.Fatalf("missing server must preserve empty identity: %#v", input.Nodes[2])
	}

	result, err := provisionapp.BuildDryRun(input)
	if err != nil {
		t.Fatalf("BuildDryRun() error = %v", err)
	}
	if len(result.Nodes) != 3 || result.Nodes[0].ServerID != 1 || result.Nodes[0].ServerName != "control-1" {
		t.Fatalf("result nodes = %#v", result.Nodes)
	}
}

func TestRuntimeDryRunPreservesValidationAndAvailabilityErrors(t *testing.T) {
	runtime := NewRuntime(nil, nil, nil)
	if _, err := runtime.DryRun(context.Background(), 0); !errors.Is(err, provisionapp.ErrInvalidParams) {
		t.Fatalf("DryRun(0) error = %v, want invalid params", err)
	}
	if _, err := runtime.DryRun(context.Background(), 1); !errors.Is(err, provisionapp.ErrConflict) {
		t.Fatalf("DryRun without database error = %v, want conflict", err)
	}
}
