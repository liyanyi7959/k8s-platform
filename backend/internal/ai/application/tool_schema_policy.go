package application

import (
	"sort"
	"strings"

	"k8s-platform-backend/internal/ai/domain"
)

// BuildToolInputJSONSchema converts the concise platform tool schema into the
// JSON Schema exposed to an LLM. Cluster identity remains platform-injected
// and is therefore never exposed as a model-supplied argument.
func BuildToolInputJSONSchema(input domain.JSONMap) domain.JSONMap {
	if len(input) == 0 {
		return domain.JSONMap{"type": "object", "properties": domain.JSONMap{}}
	}

	properties := make(domain.JSONMap, len(input))
	required := make([]string, 0, len(input))
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "cluster_id" {
			continue
		}

		property, optional := toolInputPropertySchema(input[key])
		properties[key] = property
		if !optional {
			required = append(required, key)
		}
	}

	schema := domain.JSONMap{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// ToolEvidenceMap returns the persisted evidence envelope for a completed
// tool invocation without leaking empty sections into the read model.
func ToolEvidenceMap(result ToolResult) domain.JSONMap {
	out := domain.JSONMap{"summary": result.Summary}
	if len(result.Evidence) > 0 {
		out["evidence"] = result.Evidence
	}
	if len(result.RawRef) > 0 {
		out["raw_ref"] = result.RawRef
	}
	return out
}

func ClusterToolInputSchema() domain.JSONMap {
	return domain.JSONMap{"cluster_id": "number"}
}

func NamespaceToolInputSchema() domain.JSONMap {
	return domain.JSONMap{"cluster_id": "number", "namespace": "string"}
}

func ScopedResourceToolInputSchema(kind string) domain.JSONMap {
	return domain.JSONMap{"cluster_id": "number", "resource_kind": kind, "namespace": "string", "name": "string"}
}

func ClusterScopedNameToolInputSchema(kind string) domain.JSONMap {
	return domain.JSONMap{"cluster_id": "number", "resource_kind": kind, "name": "string"}
}

func GenericResourceToolInputSchema() domain.JSONMap {
	return domain.JSONMap{"cluster_id": "number", "kind": "string", "namespace": "string?", "name": "string"}
}

func ToolOutputSchema(source string) domain.JSONMap {
	return domain.JSONMap{"summary": "string", "evidence": "object", "raw_ref": domain.JSONMap{"source": source}}
}

// CloneToolJSONMap copies nested JSON map and list containers so catalog
// responses cannot mutate shared registry definitions.
func CloneToolJSONMap(input domain.JSONMap) domain.JSONMap {
	if input == nil {
		return nil
	}
	cloned := make(domain.JSONMap, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case domain.JSONMap:
			cloned[key] = CloneToolJSONMap(typed)
		case map[string]any:
			cloned[key] = CloneToolJSONMap(domain.JSONMap(typed))
		case []any:
			items := make([]any, len(typed))
			copy(items, typed)
			cloned[key] = items
		default:
			cloned[key] = value
		}
	}
	return cloned
}

func toolInputPropertySchema(value any) (domain.JSONMap, bool) {
	switch typed := value.(type) {
	case string:
		optional := strings.HasSuffix(typed, "?")
		typeName := strings.TrimSuffix(typed, "?")
		if strings.Contains(typeName, "|") {
			return domain.JSONMap{"type": "string", "enum": strings.Split(typeName, "|")}, optional
		}
		return domain.JSONMap{"type": typeName}, optional
	case []string:
		if len(typed) > 0 {
			return domain.JSONMap{"type": "array", "items": domain.JSONMap{"type": typed[0]}}, false
		}
		return domain.JSONMap{"type": "string"}, false
	case []any:
		if len(typed) > 0 {
			if itemType, ok := typed[0].(string); ok {
				return domain.JSONMap{"type": "array", "items": domain.JSONMap{"type": itemType}}, false
			}
		}
		return domain.JSONMap{"type": "array"}, false
	default:
		return domain.JSONMap{"type": "string"}, false
	}
}
