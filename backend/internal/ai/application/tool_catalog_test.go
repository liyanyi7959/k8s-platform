package application

import (
	"testing"
	"time"

	"k8s-platform-backend/internal/ai/domain"
)

func TestToolCatalogRegistersGetsAndListsDefinitions(t *testing.T) {
	catalog := NewToolCatalog()
	catalog.Register(ToolDefinition{Name: "b.tool", Category: "query"})
	catalog.Register(ToolDefinition{Name: "a.tool", Category: "query"})
	catalog.Register(ToolDefinition{Name: "c.tool", Category: "action"})
	catalog.Register(ToolDefinition{Name: " a.tool ", Category: "replacement"})
	catalog.Register(ToolDefinition{Name: "  "})

	definition, ok := catalog.Get(" a.tool ")
	if !ok || definition.Category != "replacement" {
		t.Fatalf("Get() = %#v, %v", definition, ok)
	}
	definitions := catalog.List()
	if len(definitions) != 3 {
		t.Fatalf("len(List()) = %d, want 3", len(definitions))
	}
	if definitions[0].Name != "c.tool" || definitions[1].Name != "b.tool" || definitions[2].Name != " a.tool " {
		t.Fatalf("List() = %#v", definitions)
	}
}

func TestToolCatalogBuildsIsolatedAvailabilityCatalog(t *testing.T) {
	catalog := NewToolCatalog()
	catalog.Register(ToolDefinition{
		Name:                "resource.inspect",
		Category:            "query",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		Timeout:             15 * time.Second,
		InputSchema:         domain.JSONMap{"kind": "string", "nested": domain.JSONMap{"value": "string"}},
	})

	items := catalog.ListCatalog([]string{"ai:tool_exec"})
	if len(items) != 1 || items[0].Available || len(items[0].MissingPermissions) != 1 || items[0].MissingPermissions[0] != "k8s:read" {
		t.Fatalf("ListCatalog() = %#v", items)
	}
	if items[0].TimeoutSeconds != 15 {
		t.Fatalf("TimeoutSeconds = %d, want 15", items[0].TimeoutSeconds)
	}
	items[0].RequiredPermissions[0] = "changed"
	items[0].InputSchema["kind"] = "changed"
	items[0].InputSchema["nested"].(domain.JSONMap)["value"] = "changed"

	definition, _ := catalog.Get("resource.inspect")
	if definition.RequiredPermissions[0] != "ai:tool_exec" || definition.InputSchema["kind"] != "string" || definition.InputSchema["nested"].(domain.JSONMap)["value"] != "string" {
		t.Fatalf("catalog definition was mutated: %#v", definition)
	}
}

func TestToolCatalogPlansOnlyRegisteredDiagnosticTools(t *testing.T) {
	catalog := NewToolCatalog()
	catalog.Register(ToolDefinition{Name: "pod.inspect"})
	catalog.Register(ToolDefinition{Name: "resource.events"})

	steps := catalog.PlanAutoDiagnostics(ToolContextRequest{
		ClusterID:    7,
		Query:        "check this pod warning event",
		Namespace:    "ops",
		ResourceKind: "Pod",
		ResourceName: "api",
	})
	if len(steps) != 2 || steps[0].ToolName != "pod.inspect" || steps[1].ToolName != "resource.events" {
		t.Fatalf("PlanAutoDiagnostics() = %#v", steps)
	}
}
