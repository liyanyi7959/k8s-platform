package application

import "testing"

func TestBuildAutoDiagnosticToolPlanUsesOnlyAvailableTools(t *testing.T) {
	steps := BuildAutoDiagnosticToolPlan(ToolContextRequest{ClusterID: 7, Query: "check this pod warning event", Namespace: "ops", ResourceKind: "Pod", ResourceName: "api"}, func(name string) bool {
		return name == "pod.inspect" || name == "resource.events"
	})
	if len(steps) != 2 || steps[0].ToolName != "pod.inspect" || steps[1].ToolName != "resource.events" {
		t.Fatalf("steps = %#v", steps)
	}
}

func TestBuildAutoDiagnosticToolPlanPreservesStrictScopeUnlessBroadened(t *testing.T) {
	available := func(string) bool { return true }
	strict := BuildAutoDiagnosticToolPlan(ToolContextRequest{ClusterID: 7, Query: "current replica count", Namespace: "ops", ResourceKind: "Deployment", ResourceName: "api"}, available)
	for _, step := range strict {
		if step.ToolName == "namespace.inspect" || step.ToolName == "namespace.workloads" {
			t.Fatalf("strict scope unexpectedly added %q: %#v", step.ToolName, strict)
		}
	}
	broadened := BuildAutoDiagnosticToolPlan(ToolContextRequest{ClusterID: 7, Query: "compare with other deployments in the whole namespace", Namespace: "ops", ResourceKind: "Deployment", ResourceName: "api"}, available)
	if !hasPlannedTool(broadened, "namespace.inspect") {
		t.Fatalf("broadened scope plan = %#v", broadened)
	}
}

func TestBuildAutoDiagnosticToolPlanProtectsSecretYAML(t *testing.T) {
	steps := BuildAutoDiagnosticToolPlan(ToolContextRequest{ClusterID: 7, Query: "show yaml", Namespace: "ops", ResourceKind: "Secret", ResourceName: "token"}, nil)
	if !hasPlannedTool(steps, "resource.masked_yaml") || hasPlannedTool(steps, "resource.yaml") {
		t.Fatalf("secret YAML plan = %#v", steps)
	}
}

func hasPlannedTool(steps []ToolPlanStep, name string) bool {
	for _, step := range steps {
		if step.ToolName == name {
			return true
		}
	}
	return false
}
