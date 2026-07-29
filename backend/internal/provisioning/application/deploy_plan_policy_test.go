package application

import (
	"errors"
	"testing"

	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
)

func TestNormalizeDeployPlanAppliesDefaultsAndCanonicalVersion(t *testing.T) {
	plan, nodes, err := NormalizeDeployPlan(DeployPlanInput{Name: "demo", ClusterName: "demo-cluster", K8sVersion: "1.31.0", Nodes: []DeployPlanNodeInput{{ServerID: 1, Role: "master"}}}, 7)
	if err != nil {
		t.Fatal(err)
	}
	if plan.K8sVersion != "v1.31.0" || plan.PodCIDR != "10.244.0.0/16" || len(nodes) != 1 || plan.CreatedBy != 7 {
		t.Fatalf("normalized plan = %#v, nodes = %#v", plan, nodes)
	}
}

func TestNormalizeDeployPlanRejectsUnsafeTopology(t *testing.T) {
	_, _, err := NormalizeDeployPlan(DeployPlanInput{Name: "demo", ClusterName: "demo", K8sVersion: "v1.31.0", PodCIDR: "10.0.0.0/16", SvcCIDR: "10.0.1.0/24", Nodes: []DeployPlanNodeInput{{ServerID: 1, Role: "master"}, {ServerID: 2, Role: "master"}}, StepOverrides: map[string]provisiondomain.DeployPlanStepOverride{}}, 1)
	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
