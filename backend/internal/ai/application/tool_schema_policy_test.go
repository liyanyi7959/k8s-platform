package application

import (
	"reflect"
	"testing"

	"k8s-platform-backend/internal/ai/domain"
)

func TestBuildToolInputJSONSchema(t *testing.T) {
	schema := BuildToolInputJSONSchema(domain.JSONMap{
		"cluster_id": "integer",
		"kind":       "Deployment|StatefulSet",
		"namespace":  "string?",
		"names":      []string{"string"},
	})
	properties := schema["properties"].(domain.JSONMap)
	if _, exposed := properties["cluster_id"]; exposed {
		t.Fatal("cluster_id must remain platform-injected")
	}
	if got := properties["kind"].(domain.JSONMap)["enum"]; !reflect.DeepEqual(got, []string{"Deployment", "StatefulSet"}) {
		t.Fatalf("kind enum = %#v", got)
	}
	if got := properties["names"].(domain.JSONMap)["items"].(domain.JSONMap)["type"]; got != "string" {
		t.Fatalf("array item type = %#v", got)
	}
	if got := schema["required"]; !reflect.DeepEqual(got, []string{"kind", "names"}) {
		t.Fatalf("required = %#v", got)
	}
}

func TestBuildToolInputJSONSchemaEmptyInput(t *testing.T) {
	schema := BuildToolInputJSONSchema(nil)
	if schema["type"] != "object" || len(schema["properties"].(domain.JSONMap)) != 0 {
		t.Fatalf("empty schema = %#v", schema)
	}
}

func TestToolSchemaHelpersCopyAndEnvelope(t *testing.T) {
	input := domain.JSONMap{"nested": domain.JSONMap{"value": "original"}, "items": []any{"a"}}
	cloned := CloneToolJSONMap(input)
	cloned["nested"].(domain.JSONMap)["value"] = "changed"
	cloned["items"].([]any)[0] = "changed"
	if input["nested"].(domain.JSONMap)["value"] != "original" || input["items"].([]any)[0] != "a" {
		t.Fatalf("source schema mutated: %#v", input)
	}
	evidence := ToolEvidenceMap(ToolResult{Summary: "ok", Evidence: domain.JSONMap{"resource": "pod"}})
	if evidence["summary"] != "ok" || evidence["evidence"] == nil || evidence["raw_ref"] != nil {
		t.Fatalf("evidence envelope = %#v", evidence)
	}
	if ScopedResourceToolInputSchema("Pod")["resource_kind"] != "Pod" || ToolOutputSchema("pod.inspect")["raw_ref"] == nil {
		t.Fatal("tool schema helpers lost scope metadata")
	}
}
