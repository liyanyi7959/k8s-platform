package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s-platform-backend/internal/model"
)

type AIToolResult struct {
	Summary  string        `json:"summary"`
	Evidence model.JSONMap `json:"evidence,omitempty"`
	RawRef   model.JSONMap `json:"raw_ref,omitempty"`
}

type AIToolDefinition struct {
	Name                string
	Category            string
	Description         string
	RequiredPermissions []string
	RiskLevel           string
	ConfirmLevel        string
	Timeout             time.Duration
	RedactionPolicy     string
	Handler             func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error)
}

type AIToolRegistry struct {
	defs  map[string]AIToolDefinition
	order []string
}

func NewAIToolRegistry(
	clusterSvc *ClusterReadModelService,
	namespaceSvc *NamespaceDiagnosisService,
	inspectionSvc *ResourceInspectionService,
	exportPolicySvc *ResourceExportPolicyService,
) *AIToolRegistry {
	r := &AIToolRegistry{
		defs:  map[string]AIToolDefinition{},
		order: make([]string, 0, 8),
	}

	r.register(AIToolDefinition{
		Name:                "cluster.health",
		Category:            "query",
		Description:         "Cluster health overview",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             10 * time.Second,
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return clusterSvc.GetClusterHealth(ctx, req.ClusterID)
		},
	})

	r.register(AIToolDefinition{
		Name:                "cluster.inventory",
		Category:            "query",
		Description:         "Cluster inventory snapshot",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15 * time.Second,
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return clusterSvc.GetClusterInventory(ctx, req.ClusterID)
		},
	})

	r.register(AIToolDefinition{
		Name:                "namespace.health",
		Category:            "query",
		Description:         "Namespace health diagnostics",
		RequiredPermissions: []string{"ai:tool_exec", "namespace:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15 * time.Second,
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return namespaceSvc.GetNamespaceHealth(ctx, req.ClusterID, req.Namespace)
		},
	})

	r.register(AIToolDefinition{
		Name:                "namespace.summary",
		Category:            "query",
		Description:         "Namespace resource summary",
		RequiredPermissions: []string{"ai:tool_exec", "namespace:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15 * time.Second,
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return namespaceSvc.GetNamespaceResourceSummary(ctx, req.ClusterID, req.Namespace)
		},
	})

	r.register(AIToolDefinition{
		Name:                "pod.inspect",
		Category:            "inspect",
		Description:         "Inspect pod details and recent logs",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return inspectionSvc.InspectPod(ctx, req.ClusterID, req.Namespace, req.ResourceName)
		},
	})

	r.register(AIToolDefinition{
		Name:                "node.inspect",
		Category:            "inspect",
		Description:         "Inspect node details and related events",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return inspectionSvc.InspectNode(ctx, req.ClusterID, req.ResourceName)
		},
	})

	r.register(AIToolDefinition{
		Name:                "deployment.inspect",
		Category:            "inspect",
		Description:         "Inspect deployment details and rollout history",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return inspectionSvc.InspectDeployment(ctx, req.ClusterID, req.Namespace, req.ResourceName)
		},
	})

	r.register(AIToolDefinition{
		Name:                "resource.yaml",
		Category:            "export",
		Description:         "Export resource YAML with redaction policy",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		RedactionPolicy:     "mask_sensitive",
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			kind := strings.TrimSpace(fmt.Sprint(input["kind"]))
			namespace := strings.TrimSpace(fmt.Sprint(input["namespace"]))
			name := strings.TrimSpace(fmt.Sprint(input["name"]))
			return inspectionSvc.ExportResourceYAML(ctx, req.ClusterID, kind, namespace, name, exportPolicySvc)
		},
	})

	return r
}

func (r *AIToolRegistry) register(def AIToolDefinition) {
	name := strings.TrimSpace(def.Name)
	if name == "" {
		return
	}
	if r.defs == nil {
		r.defs = map[string]AIToolDefinition{}
	}
	if _, exists := r.defs[name]; !exists {
		r.order = append(r.order, name)
	}
	r.defs[name] = def
}

func (r *AIToolRegistry) Get(name string) (AIToolDefinition, bool) {
	if r == nil {
		return AIToolDefinition{}, false
	}
	def, ok := r.defs[strings.TrimSpace(name)]
	return def, ok
}

func (r *AIToolRegistry) List() []AIToolDefinition {
	if r == nil {
		return nil
	}
	out := make([]AIToolDefinition, 0, len(r.defs))
	for _, name := range r.order {
		if def, ok := r.defs[name]; ok {
			out = append(out, def)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category == out[j].Category {
			return out[i].Name < out[j].Name
		}
		return out[i].Category < out[j].Category
	})
	return out
}

func hasAllPermissions(userPerms []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(userPerms) == 0 {
		return false
	}
	permSet := make(map[string]struct{}, len(userPerms))
	for _, perm := range userPerms {
		perm = strings.TrimSpace(perm)
		if perm == "" {
			continue
		}
		permSet[perm] = struct{}{}
	}
	for _, requiredPerm := range required {
		requiredPerm = strings.TrimSpace(requiredPerm)
		if requiredPerm == "" {
			continue
		}
		if _, ok := permSet[requiredPerm]; !ok {
			return false
		}
	}
	return true
}

func missingPermissions(userPerms []string, required []string) []string {
	if len(required) == 0 {
		return nil
	}
	permSet := make(map[string]struct{}, len(userPerms))
	for _, perm := range userPerms {
		perm = strings.TrimSpace(perm)
		if perm != "" {
			permSet[perm] = struct{}{}
		}
	}
	missing := make([]string, 0, len(required))
	for _, requiredPerm := range required {
		requiredPerm = strings.TrimSpace(requiredPerm)
		if requiredPerm == "" {
			continue
		}
		if _, ok := permSet[requiredPerm]; !ok {
			missing = append(missing, requiredPerm)
		}
	}
	return missing
}

func toolPermissionErr(required, userPerms []string, toolName string) error {
	missing := missingPermissions(userPerms, required)
	if len(missing) == 0 {
		return nil
	}
	return ErrWithMessage(ErrK8sForbidden, fmt.Sprintf("tool %s requires permissions: %s", toolName, strings.Join(missing, ", ")))
}

func toolEvidenceMap(result AIToolResult) model.JSONMap {
	out := model.JSONMap{
		"summary": result.Summary,
	}
	if len(result.Evidence) > 0 {
		out["evidence"] = result.Evidence
	}
	if len(result.RawRef) > 0 {
		out["raw_ref"] = result.RawRef
	}
	return out
}
