package application

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildDryRunSortsNodesFiltersRoleStepsAndSummarizesPhases(t *testing.T) {
	result, err := BuildDryRun(DryRunPlanInput{
		PlanID: 7, PlanName: "demo", ClusterName: "demo-cluster", K8sVersion: "v1.31.0", CNIType: "calico",
		Nodes: []DryRunNodeInput{
			{ServerID: 2, ServerName: "worker-a", IP: "192.0.2.12", Role: "worker", SortOrder: 0},
			{ServerID: 1, ServerName: "master-a", IP: "192.0.2.11", Role: "master", SortOrder: 5},
		},
	})
	if err != nil {
		t.Fatalf("BuildDryRun() error = %v", err)
	}
	if result.PlanID != 7 || result.ClusterName != "demo-cluster" || result.CNIType != "calico" {
		t.Fatalf("result metadata = %#v", result)
	}
	if len(result.Nodes) != 2 || result.Nodes[0].ServerID != 1 || result.Nodes[1].ServerID != 2 {
		t.Fatalf("nodes must put master first: %#v", result.Nodes)
	}
	if got, want := dryRunStepKeys(result.Nodes[0].Steps), []string{"pre_check", "bootstrap", "container_runtime", "kubeadm_init", "install_cni", "install_helm", "install_addons", "register"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("master steps = %#v, want %#v", got, want)
	}
	if got, want := dryRunStepKeys(result.Nodes[1].Steps), []string{"pre_check", "bootstrap", "container_runtime", "join_workers"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("worker steps = %#v, want %#v", got, want)
	}
	if got, want := result.Summary, map[string]int{"preflight": 2, "install": 4, "init": 1, "addon": 3, "finalize": 1, "join": 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("summary = %#v, want %#v", got, want)
	}
}

func TestBuildDryRunCopiesStepMetadataAndDefaultsServerName(t *testing.T) {
	input := DryRunPlanInput{PlanID: 1, Nodes: []DryRunNodeInput{{ServerID: 42, Role: "master"}}}
	first, err := BuildDryRun(input)
	if err != nil {
		t.Fatalf("first BuildDryRun() error = %v", err)
	}
	if first.Nodes[0].ServerName != "server-42" {
		t.Fatalf("fallback server name = %q", first.Nodes[0].ServerName)
	}
	first.Nodes[0].Steps[0].Tasks[0] = "mutated"
	first.Nodes[0].Steps[3].DependsOn[0] = "mutated"

	second, err := BuildDryRun(input)
	if err != nil {
		t.Fatalf("second BuildDryRun() error = %v", err)
	}
	if second.Nodes[0].Steps[0].Tasks[0] == "mutated" || second.Nodes[0].Steps[3].DependsOn[0] == "mutated" {
		t.Fatalf("step metadata leaked across calls: %#v", second.Nodes[0].Steps)
	}
}

func TestBuildDryRunValidatesPlanAndNodes(t *testing.T) {
	if _, err := BuildDryRun(DryRunPlanInput{}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("zero input error = %v, want invalid params", err)
	}
	if _, err := BuildDryRun(DryRunPlanInput{PlanID: 1}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("empty nodes error = %v, want invalid params", err)
	}
}

func dryRunStepKeys(steps []DryRunStep) []string {
	keys := make([]string, 0, len(steps))
	for _, step := range steps {
		keys = append(keys, step.Key)
	}
	return keys
}
