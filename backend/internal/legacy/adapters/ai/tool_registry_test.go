package ai

import (
	"errors"
	"testing"

	aiapp "k8s-platform-backend/internal/ai/application"
	model "k8s-platform-backend/internal/ai/domain"
)

func newTestAIToolRegistry() *AIToolRegistry {
	return &AIToolRegistry{catalog: aiapp.NewToolCatalog()}
}

// ────────── Register / Get ──────────

func TestAIToolRegistry_RegisterAndGet(t *testing.T) {
	r := newTestAIToolRegistry()
	r.register(AIToolDefinition{Name: "test.tool", Category: "query", Description: "test"})

	def, ok := r.Get("test.tool")
	if !ok {
		t.Fatal("expected to find registered tool")
	}
	if def.Category != "query" {
		t.Fatalf("category = %q, want query", def.Category)
	}
}

func TestAIToolRegistry_GetTrimsWhitespace(t *testing.T) {
	r := newTestAIToolRegistry()
	r.register(AIToolDefinition{Name: "my.tool"})

	_, ok := r.Get("  my.tool  ")
	if !ok {
		t.Fatal("Get should trim whitespace")
	}
}

func TestAIToolRegistry_GetMissingReturnsFalse(t *testing.T) {
	r := newTestAIToolRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Fatal("expected false for missing tool")
	}
}

func TestAIToolRegistry_GetNilRegistryReturnsFalse(t *testing.T) {
	var r *AIToolRegistry
	_, ok := r.Get("any")
	if ok {
		t.Fatal("nil registry should return false")
	}
}

func TestAIToolRegistry_ListNilRegistryReturnsNil(t *testing.T) {
	var r *AIToolRegistry
	if r.List() != nil {
		t.Fatal("nil registry List should return nil")
	}
}

func TestAIToolRegistry_RegisterSkipsEmptyName(t *testing.T) {
	r := newTestAIToolRegistry()
	r.register(AIToolDefinition{Name: ""})
	r.register(AIToolDefinition{Name: "   "})
	if len(r.List()) != 0 {
		t.Fatal("empty names should be skipped")
	}
}

func TestAIToolRegistry_RegisterOverwritesDuplicate(t *testing.T) {
	r := newTestAIToolRegistry()
	r.register(AIToolDefinition{Name: "dup", Category: "old"})
	r.register(AIToolDefinition{Name: "dup", Category: "new"})

	def, ok := r.Get("dup")
	if !ok {
		t.Fatal("expected tool to exist")
	}
	if def.Category != "new" {
		t.Fatalf("category = %q, want new (duplicate should overwrite)", def.Category)
	}
	if definitions := r.List(); len(definitions) != 1 {
		t.Fatalf("List() returned %d definitions after duplicate registration, want 1", len(definitions))
	}
}

// ────────── List ordering ──────────

func TestAIToolRegistry_ListSortsByCategoryThenName(t *testing.T) {
	r := newTestAIToolRegistry()
	r.register(AIToolDefinition{Name: "b.tool", Category: "z"})
	r.register(AIToolDefinition{Name: "a.tool", Category: "z"})
	r.register(AIToolDefinition{Name: "c.tool", Category: "a"})

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}
	if list[0].Name != "c.tool" {
		t.Fatalf("first = %q, want c.tool", list[0].Name)
	}
	if list[1].Name != "a.tool" {
		t.Fatalf("second = %q, want a.tool", list[1].Name)
	}
	if list[2].Name != "b.tool" {
		t.Fatalf("third = %q, want b.tool", list[2].Name)
	}
}

// ────────── ListCatalog ──────────

func TestAIToolRegistry_ListCatalog(t *testing.T) {
	r := newTestAIToolRegistry()
	r.register(AIToolDefinition{
		Name:                "tool.a",
		Category:            "query",
		Description:         "desc a",
		RequiredPermissions: []string{"perm1", "perm2"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15e9,
	})
	r.register(AIToolDefinition{
		Name:                "tool.b",
		Category:            "query",
		Description:         "desc b",
		RequiredPermissions: []string{"perm1"},
		RiskLevel:           "medium",
		ConfirmLevel:        "double",
		Timeout:             20e9,
	})

	items := r.ListCatalog([]string{"perm1", "perm2"})
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}

	itemA := items[0]
	if itemA.Name != "tool.a" {
		t.Fatalf("first item = %q, want tool.a", itemA.Name)
	}
	if !itemA.Available {
		t.Fatal("tool.a should be available")
	}
	if len(itemA.MissingPermissions) != 0 {
		t.Fatalf("tool.a missing perms = %v, want empty", itemA.MissingPermissions)
	}
	if itemA.TimeoutSeconds != 15 {
		t.Fatalf("tool.a timeout = %d, want 15", itemA.TimeoutSeconds)
	}

	itemB := items[1]
	if !itemB.Available {
		t.Fatal("tool.b should be available")
	}
}

func TestAIToolRegistry_ListCatalogShowsMissingPermissions(t *testing.T) {
	r := newTestAIToolRegistry()
	r.register(AIToolDefinition{
		Name:                "restricted",
		RequiredPermissions: []string{"perm1", "perm2", "perm3"},
		Timeout:             10e9,
	})

	items := r.ListCatalog([]string{"perm1"})
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].Available {
		t.Fatal("should not be available with missing perms")
	}
	if len(items[0].MissingPermissions) != 2 {
		t.Fatalf("missing = %v, want 2 items", items[0].MissingPermissions)
	}
}

// ────────── hasAllPermissions ──────────

func TestHasAllPermissions(t *testing.T) {
	tests := []struct {
		name   string
		user   []string
		req    []string
		expect bool
	}{
		{"empty required", []string{"a"}, nil, true},
		{"empty user", nil, []string{"a"}, false},
		{"all present", []string{"a", "b", "c"}, []string{"a", "c"}, true},
		{"missing one", []string{"a"}, []string{"a", "b"}, false},
		{"extra spaces", []string{" a ", " b "}, []string{"a", "b"}, true},
		{"empty strings ignored", []string{"", "a"}, []string{"a"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aiapp.HasAllToolPermissions(tt.user, tt.req)
			if got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

// ────────── missingPermissions ──────────

func TestMissingPermissions(t *testing.T) {
	tests := []struct {
		name   string
		user   []string
		req    []string
		expect []string
	}{
		{"none missing", []string{"a", "b"}, []string{"a", "b"}, nil},
		{"all missing", nil, []string{"a", "b"}, []string{"a", "b"}},
		{"partial", []string{"a"}, []string{"a", "b", "c"}, []string{"b", "c"}},
		{"empty required", []string{"a"}, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aiapp.MissingToolPermissions(tt.user, tt.req)
			if len(got) != len(tt.expect) {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
			for i, v := range got {
				if v != tt.expect[i] {
					t.Fatalf("got[%d] = %q, want %q", i, v, tt.expect[i])
				}
			}
		})
	}
}

// ────────── toolPermissionErr ──────────

func TestToolPermissionErr_ReturnsNilWhenSatisfied(t *testing.T) {
	err := toolPermissionErr([]string{"a"}, []string{"a", "b"}, "my.tool")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestToolPermissionErr_ReturnsErrorWhenMissing(t *testing.T) {
	err := toolPermissionErr([]string{"a", "x"}, []string{"a"}, "my.tool")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrK8sForbidden) {
		t.Fatalf("expected ErrK8sForbidden, got %v", err)
	}
}

// ────────── toolEvidenceMap ──────────

func TestToolEvidenceMap(t *testing.T) {
	result := AIToolResult{
		Summary:  "all good",
		Evidence: model.JSONMap{"key": "val"},
		RawRef:   model.JSONMap{"source": "test"},
	}
	m := aiapp.ToolEvidenceMap(result)
	if m["summary"] != "all good" {
		t.Fatalf("summary = %v", m["summary"])
	}
	if m["evidence"] == nil {
		t.Fatal("expected evidence")
	}
	if m["raw_ref"] == nil {
		t.Fatal("expected raw_ref")
	}
}

func TestToolEvidenceMap_OmitsEmpty(t *testing.T) {
	result := AIToolResult{Summary: "only summary"}
	m := aiapp.ToolEvidenceMap(result)
	if m["evidence"] != nil {
		t.Fatal("evidence should be omitted when empty")
	}
	if m["raw_ref"] != nil {
		t.Fatal("raw_ref should be omitted when empty")
	}
}

// ────────── cloneJSONMap ──────────

func TestCloneJSONMap_NilReturnsNil(t *testing.T) {
	if aiapp.CloneToolJSONMap(nil) != nil {
		t.Fatal("cloneJSONMap(nil) should return nil")
	}
}

func TestCloneJSONMap_DeepCopy(t *testing.T) {
	original := model.JSONMap{
		"str": "hello",
		"num": 42,
		"nested": model.JSONMap{
			"inner": "value",
		},
		"arr": []any{"a", "b"},
	}
	cloned := aiapp.CloneToolJSONMap(original)

	cloned["str"] = "changed"
	cloned["nested"].(model.JSONMap)["inner"] = "changed"
	cloned["arr"].([]any)[0] = "x"

	if original["str"] != "hello" {
		t.Fatal("original str was mutated")
	}
	if original["nested"].(model.JSONMap)["inner"] != "value" {
		t.Fatal("original nested was mutated")
	}
}

func TestCloneJSONMap_HandlesNestedJSONMap(t *testing.T) {
	original := model.JSONMap{
		"nested": model.JSONMap{"k": "v"},
	}
	cloned := aiapp.CloneToolJSONMap(original)
	inner, ok := cloned["nested"].(model.JSONMap)
	if !ok || inner["k"] != "v" {
		t.Fatal("nested model.JSONMap clone failed")
	}
}

// ────────── aiStringSliceValue ──────────

func TestAIStringSliceValue_StringSlice(t *testing.T) {
	got := aiapp.StringSliceInput([]string{"a", " b ", "", "c"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[1] != "b" {
		t.Fatalf("got[1] = %q, want b", got[1])
	}
}

func TestAIStringSliceValue_AnySlice(t *testing.T) {
	got := aiapp.StringSliceInput([]any{"x", 123, "y"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[1] != "123" {
		t.Fatalf("got[1] = %q, want 123", got[1])
	}
}

func TestAIStringSliceValue_DefaultReturnsNil(t *testing.T) {
	if aiapp.StringSliceInput("not a slice") != nil {
		t.Fatal("non-slice should return nil")
	}
	if aiapp.StringSliceInput(nil) != nil {
		t.Fatal("nil should return nil")
	}
}

// ────────── buildApplyManifestProposalRequest ──────────

func TestBuildApplyManifestProposalRequest_EmptyYAMLError(t *testing.T) {
	_, err := buildApplyManifestProposalRequest(AIToolContextRequest{}, map[string]any{"yaml": ""})
	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("expected ErrInvalidParams, got %v", err)
	}
}

func TestBuildApplyManifestProposalRequest_Success(t *testing.T) {
	req := AIToolContextRequest{ConversationID: 10, MessageID: 5}
	input := map[string]any{
		"yaml":              "apiVersion: v1\nkind: ConfigMap",
		"default_namespace": "test-ns",
		"title":             "My ConfigMap",
		"reason":            "testing",
	}
	result, err := buildApplyManifestProposalRequest(req, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProposalType != aiActionTypeApplyManifest {
		t.Fatalf("proposal type = %q, want %q", result.ProposalType, aiActionTypeApplyManifest)
	}
	if result.ConversationID != 10 {
		t.Fatalf("conversationID = %d, want 10", result.ConversationID)
	}
	if result.TargetResource.Namespace != "test-ns" {
		t.Fatalf("namespace = %q, want test-ns", result.TargetResource.Namespace)
	}
	if result.Payload["title"] != "My ConfigMap" {
		t.Fatalf("title = %v", result.Payload["title"])
	}
}

// ────────── PlanAutoDiagnostics: additional scenarios ──────────

func TestPlanAutoDiagnostics_ReturnsNilForZeroCluster(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{ClusterID: 0})
	if steps != nil {
		t.Fatalf("expected nil for clusterID=0, got %v", steps)
	}
}

func TestPlanAutoDiagnostics_AddsResourceEventsForWarningQuery(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    1,
		Query:        "这个 pod 有什么 warning 事件",
		Namespace:    "default",
		ResourceKind: "Pod",
		ResourceName: "my-pod",
	})
	found := false
	for _, s := range steps {
		if s.ToolName == "resource.events" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected resource.events in plan for warning query, got %v", steps)
	}
}

func TestPlanAutoDiagnostics_AddsResourceLogsForLogQuery(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    1,
		Query:        "show me the crash logs of this pod",
		Namespace:    "default",
		ResourceKind: "Pod",
		ResourceName: "crashing-pod",
	})
	found := false
	for _, s := range steps {
		if s.ToolName == "resource.logs" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected resource.logs for crash query, got %v", steps)
	}
}

func TestPlanAutoDiagnostics_NodeUsesNodeInspect(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    1,
		Query:        "check this node",
		ResourceKind: "Node",
		ResourceName: "node-1",
	})
	found := false
	for _, s := range steps {
		if s.ToolName == "node.inspect" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected node.inspect, got %v", steps)
	}
}

func TestPlanAutoDiagnostics_GenericResourceUsesResourceInspect(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    1,
		Query:        "check this configmap",
		Namespace:    "default",
		ResourceKind: "ConfigMap",
		ResourceName: "my-cm",
	})
	found := false
	for _, s := range steps {
		if s.ToolName == "resource.inspect" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected resource.inspect for ConfigMap, got %v", steps)
	}
}

func TestPlanAutoDiagnostics_SecretUsesMaskedYAML(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    1,
		Query:        "show the yaml of this secret",
		Namespace:    "default",
		ResourceKind: "Secret",
		ResourceName: "my-secret",
	})
	found := false
	for _, s := range steps {
		if s.ToolName == "resource.masked_yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected resource.masked_yaml for Secret yaml query, got %v", steps)
	}
}

func TestPlanAutoDiagnostics_AddsClusterHealthForBroadQuery(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID: 1,
		Query:     "集群整体运行情况如何",
	})
	foundHealth := false
	foundOverview := false
	for _, s := range steps {
		switch s.ToolName {
		case "cluster.health":
			foundHealth = true
		case "cluster.overview":
			foundOverview = true
		}
	}
	if !foundHealth {
		t.Fatal("expected cluster.health for broad query")
	}
	if !foundOverview {
		t.Fatal("expected cluster.overview for broad query")
	}
}

func TestPlanAutoDiagnostics_NonPodKindDoesNotAddPodInspect(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    1,
		Namespace:    "default",
		ResourceKind: "Service",
		ResourceName: "my-svc",
		Query:        "check this service",
	})
	for _, s := range steps {
		if s.ToolName == "pod.inspect" {
			t.Fatal("should not add pod.inspect for non-pod kind")
		}
	}
}

func TestPlanAutoDiagnostics_NonWorkloadKindSkipsLogs(t *testing.T) {
	r := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	steps := r.PlanAutoDiagnostics(AIToolContextRequest{
		ClusterID:    1,
		Query:        "show me the logs",
		Namespace:    "default",
		ResourceKind: "Service",
		ResourceName: "my-svc",
	})
	for _, s := range steps {
		if s.ToolName == "resource.logs" {
			t.Fatal("should not add resource.logs for Service kind")
		}
	}
}

// ────────── PlanAutoDiagnostics: original integration tests ──────────

func TestPlanAutoDiagnosticsAddsNamespaceWorkloadsForScopedPod(t *testing.T) {
	registry := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
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
	registry := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
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
	registry := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
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
	registry := NewAIToolRegistry(nil, nil, nil, nil, nil, nil, nil)
	expected := map[string]bool{
		"resource.list":                        false,
		"resource.search":                      false,
		"resource.events":                      false,
		"resource.logs":                        false,
		"resource.related":                     false,
		"proposal.node.cordon":                 false,
		"proposal.workload.image":              false,
		"proposal.resource.delete":             false,
		"proposal.batch.trigger_cronjob":       false,
		"proposal.batch.delete_completed_jobs": false,
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
