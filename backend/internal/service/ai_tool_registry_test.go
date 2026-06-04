package service

import "testing"

func TestPlanAutoDiagnosticsAddsNamespaceWorkloadsForScopedPod(t *testing.T) {
	registry := NewAIToolRegistry(nil, nil, nil, nil, nil, nil)
	steps := registry.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    7,
		Query:        "check this pod and explain which deployment owns it",
		Namespace:    "devops",
		ResourceKind: "Pod",
		ResourceName: "demo-pod",
	})

	found := false
	for _, step := range steps {
		if step.ToolName == "namespace.workloads" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected namespace.workloads in plan, got %#v", steps)
	}
}

func TestPlanAutoDiagnosticsKeepsScopedDeploymentStrict(t *testing.T) {
	registry := NewAIToolRegistry(nil, nil, nil, nil, nil, nil)
	steps := registry.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    7,
		Query:        "当前的副本数是多少？如果小于 2 就扩到 2 个副本",
		Namespace:    "devops",
		ResourceKind: "Deployment",
		ResourceName: "bkci-auth",
	})

	hasDeploymentInspect := false
	for _, step := range steps {
		switch step.ToolName {
		case "deployment.inspect":
			hasDeploymentInspect = true
		case "namespace.workloads", "namespace.health", "namespace.summary", "namespace.inspect", "cluster.overview":
			t.Fatalf("unexpected broad-scope tool %q in strict scoped plan: %#v", step.ToolName, steps)
		}
	}
	if !hasDeploymentInspect {
		t.Fatalf("expected deployment.inspect in plan, got %#v", steps)
	}
}

func TestPlanAutoDiagnosticsAllowsExplicitScopeBroadening(t *testing.T) {
	registry := NewAIToolRegistry(nil, nil, nil, nil, nil, nil)
	steps := registry.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    7,
		Query:        "先看这个 deployment，再对比整个命名空间里的其他 deployment",
		Namespace:    "devops",
		ResourceKind: "Deployment",
		ResourceName: "bkci-auth",
	})

	foundNamespaceInspect := false
	for _, step := range steps {
		if step.ToolName == "namespace.inspect" {
			foundNamespaceInspect = true
			break
		}
	}
	if !foundNamespaceInspect {
		t.Fatalf("expected namespace.inspect when user explicitly broadens scope, got %#v", steps)
	}
}

func TestAIToolRegistryIncludesSharedResourceQueryTools(t *testing.T) {
	registry := NewAIToolRegistry(nil, nil, nil, nil, nil, nil)
	expected := map[string]bool{
		"resource.list":                         false,
		"resource.search":                       false,
		"resource.events":                       false,
		"resource.logs":                         false,
		"resource.related":                      false,
		"proposal.node.cordon":                  false,
		"proposal.workload.image":               false,
		"proposal.resource.delete":              false,
		"proposal.batch.trigger_cronjob":        false,
		"proposal.batch.delete_completed_jobs":  false,
	}
	for _, def := range registry.List() {
		if _, ok := expected[def.Name]; ok {
			expected[def.Name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("expected tool %q to be registered", name)
		}
	}
}
