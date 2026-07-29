package ai

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	aiapp "k8s-platform-backend/internal/ai/application"
	model "k8s-platform-backend/internal/ai/domain"
	fleetmysql "k8s-platform-backend/internal/fleet/adapters/mysql"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
)

type AIToolContextRequest = aiapp.ToolContextRequest
type AIToolResult = aiapp.ToolResult
type AIToolDefinition = aiapp.ToolDefinition
type AIToolPlanStep = aiapp.ToolPlanStep
type AIToolCatalogItem = aiapp.ToolCatalogItem
type AIToolCallItem = aiapp.ToolCallItem
type AIToolService = aiapp.ToolService

// The registry is an infrastructure composition root. It delegates proposal
// execution to the sibling AI action runtime; tool definitions remain owned
// by aiapp.ToolCatalog.
type CreateAIActionProposalRequest = aiapp.CreateActionProposalRequest
type AIActionTargetResource = aiapp.ActionTargetResource

var (
	ErrNotFound        = service.ErrNotFound
	ErrConflict        = service.ErrConflict
	ErrInvalidParams   = service.ErrInvalidParams
	ErrK8s             = service.ErrK8s
	ErrK8sNetwork      = service.ErrK8sNetwork
	ErrK8sTimeout      = service.ErrK8sTimeout
	ErrK8sUnauthorized = service.ErrK8sUnauthorized
	ErrK8sForbidden    = service.ErrK8sForbidden
	ErrK8sTLS          = service.ErrK8sTLS
)

func ErrWithMessage(kind error, message string) error { return service.ErrWithMessage(kind, message) }

func truncateForModel(input string, limit int) string {
	raw := strings.TrimSpace(input)
	if limit <= 0 || len([]rune(raw)) <= limit {
		return raw
	}
	return string([]rune(raw)[:limit]) + "..."
}

func jsonIntValue(value any) (int, bool) { return aiapp.IntegerInputValue(value) }
func ptrUint64(value uint64) *uint64     { return &value }

func aiBoolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return err == nil && parsed
	default:
		return false
	}
}

type aiProjectRow struct {
	ID          uint64 `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	Description string `gorm:"column:description"`
	ClusterID   uint64 `gorm:"column:cluster_id"`
	Namespaces  string `gorm:"column:namespaces"`
	QuotaCPU    string `gorm:"column:quota_cpu"`
	QuotaMemory string `gorm:"column:quota_memory"`
	QuotaPods   string `gorm:"column:quota_pods"`
}

func (aiProjectRow) TableName() string { return "projects" }

type clusterReadModel interface {
	GetClusterHealth(context.Context, uint64) (AIToolResult, error)
	GetClusterInventory(context.Context, uint64) (AIToolResult, error)
	GetClusterOverview(context.Context, uint64) (AIToolResult, error)
	GetClusterCertificateRisks(context.Context, uint64) (AIToolResult, error)
}

type AIToolRegistry struct {
	db      *gorm.DB
	catalog *aiapp.ToolCatalog
}

func aiInspectionToolResult(value *kopsapp.InspectionResult) AIToolResult {
	if value == nil {
		return AIToolResult{}
	}
	return AIToolResult{
		Summary:  value.Summary,
		Evidence: model.JSONMap(value.Evidence),
		RawRef:   model.JSONMap(value.RawRef),
	}
}

func aiNamespaceDiagnosisToolResult(value kopsapp.NamespaceDiagnosisResult) AIToolResult {
	return AIToolResult{Summary: value.Summary, Evidence: model.JSONMap(value.Evidence), RawRef: model.JSONMap(value.RawRef)}
}

// aiResourceInspectionToolResult is a composition-only bridge: Kops owns the
// Kubernetes read and presentation model, while the AI policy owns the sole
// decision to redact sensitive data before it reaches a model.
func aiResourceInspectionToolResult(value *kopsapp.InspectionResourceRead, policy *aiapp.ResourceExportPolicyService) AIToolResult {
	if value == nil {
		return AIToolResult{}
	}
	if policy == nil {
		policy = aiapp.NewResourceExportPolicyService()
	}
	ref := value.Reference
	safeObject, maskedObject := policy.SanitizeObject(ref.Kind, value.Object)
	evidence := model.JSONMap{
		"overview":   kopsapp.BuildInspectionResourceOverview(ref.Kind, ref.Namespace, ref.Name, safeObject),
		"conditions": kopsapp.InspectionResourceConditions(safeObject),
		"object":     safeObject,
	}
	if value.YAMLError != "" {
		evidence["yaml_error"] = value.YAMLError
	} else {
		exportedYAML, maskedYAML := policy.MaskYAML(ref.Kind, value.YAML)
		evidence["yaml"] = truncateForModel(exportedYAML, 6000)
		if maskedObject || maskedYAML {
			evidence["masked"] = true
		}
	}
	return AIToolResult{
		Summary:  kopsapp.BuildInspectionResourceSummary(ref.Kind, ref.Namespace, ref.Name, safeObject, maskedObject),
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": ref.ClusterID,
			"namespace":  ref.Namespace,
			"name":       ref.Name,
			"kind":       ref.Kind,
			"source":     "resource.inspect",
		},
	}
}

func aiResourceYAMLToolResult(value *kopsapp.InspectionResourceRead, policy *aiapp.ResourceExportPolicyService, forceMasked bool) (AIToolResult, error) {
	if value == nil {
		return AIToolResult{}, ErrConflict
	}
	if policy == nil {
		policy = aiapp.NewResourceExportPolicyService()
	}
	ref := value.Reference
	var (
		exportedYAML string
		masked       bool
		err          error
	)
	if forceMasked {
		exportedYAML, masked, err = policy.ExportMaskedYAML(ref.Kind, value.YAML)
	} else {
		exportedYAML, masked, err = policy.ExportYAML(ref.Kind, value.YAML)
	}
	if errors.Is(err, model.ErrSensitiveResourceExport) {
		return AIToolResult{}, ErrWithMessage(ErrK8sForbidden, err.Error())
	}
	if err != nil {
		return AIToolResult{}, err
	}
	evidence := model.JSONMap{"kind": ref.Kind, "namespace": ref.Namespace, "name": ref.Name, "yaml": truncateForModel(exportedYAML, 6000), "masked": masked}
	if masked {
		evidence["redaction_policy"] = "sensitive_fields_masked"
	}
	summary := fmt.Sprintf("Exported %s %s/%s YAML", ref.Kind, ref.Namespace, ref.Name)
	if masked || forceMasked {
		summary = fmt.Sprintf("Exported %s %s/%s YAML with masking", ref.Kind, ref.Namespace, ref.Name)
	}
	source := "resource.yaml"
	if forceMasked {
		source = "resource.masked_yaml"
	}
	return AIToolResult{Summary: summary, Evidence: evidence, RawRef: model.JSONMap{"cluster_id": ref.ClusterID, "namespace": ref.Namespace, "name": ref.Name, "kind": ref.Kind, "source": source}}, nil
}

func mapKopsInspectionError(err error) error {
	if err == nil {
		return nil
	}
	for _, candidate := range []struct{ application, legacy error }{
		{kopsapp.ErrInvalidParams, ErrInvalidParams}, {kopsapp.ErrNotFound, ErrNotFound}, {kopsapp.ErrConflict, ErrConflict},
		{kopsapp.ErrRuntimeNetwork, ErrK8sNetwork}, {kopsapp.ErrRuntimeTimeout, ErrK8sTimeout}, {kopsapp.ErrRuntimeUnauthorized, ErrK8sUnauthorized},
		{kopsapp.ErrRuntimeForbidden, ErrK8sForbidden}, {kopsapp.ErrRuntimeTLS, ErrK8sTLS}, {kopsapp.ErrRuntime, ErrK8s},
	} {
		if errors.Is(err, candidate.application) {
			var withMessage interface{ UserMessage() string }
			if errors.As(err, &withMessage) && withMessage != nil && strings.TrimSpace(withMessage.UserMessage()) != "" {
				return ErrWithMessage(candidate.legacy, withMessage.UserMessage())
			}
			return candidate.legacy
		}
	}
	return err
}

// NewToolRegistry composes the infrastructure-backed AI tool handlers.
// aiapp.ToolCatalog retains definition ownership and exposes the registry via
// aiapp.ToolRegistryPort without importing any runtime adapters.
func NewToolRegistry(
	db *gorm.DB,
	clusterSvc clusterReadModel,
	namespaceSvc kopsapp.NamespaceDiagnosisReader,
	inspectionSvc *kopsapp.InspectionService,
	resourceQuerySvc aiapp.ResourceQueryPort,
	actionSvc *ActionRuntime,
	exportPolicySvc *aiapp.ResourceExportPolicyService,
) *AIToolRegistry {
	r := &AIToolRegistry{
		db:      db,
		catalog: aiapp.NewToolCatalog(),
	}
	createProposalHandler := func(builder func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error)) func(context.Context, AIToolContextRequest, map[string]any) (AIToolResult, error) {
		return func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			if actionSvc == nil {
				return AIToolResult{}, ErrWithMessage(ErrConflict, "AI action proposal service is not initialized")
			}
			proposalReq, err := builder(req, input)
			if err != nil {
				return AIToolResult{}, err
			}
			result, err := actionSvc.CreateProposal(ctx, req.ClusterID, req.UserID, req.Username, proposalReq)
			if err != nil {
				return AIToolResult{}, err
			}
			evidence := model.JSONMap{
				"proposal_id":                result.ProposalID,
				"status":                     result.Status,
				"risk_level":                 result.RiskLevel,
				"need_second_confirm":        result.NeedSecondConfirm,
				"required_confirmation_text": result.RequiredConfirmationText,
				"preview":                    result.Preview,
				"diff":                       result.Diff,
				"proposal":                   model.JSONMap{"id": result.Proposal.ID, "action_type": result.Proposal.ActionType, "title": result.Proposal.Title, "summary": result.Proposal.Summary},
			}
			return AIToolResult{
				Summary:  fmt.Sprintf("Created proposal %d: %s", result.ProposalID, strings.TrimSpace(result.Proposal.Title)),
				Evidence: evidence,
				RawRef: model.JSONMap{
					"cluster_id":   req.ClusterID,
					"conversation": req.ConversationID,
					"proposal_id":  result.ProposalID,
					"source":       "proposal",
				},
			}, nil
		}
	}

	r.register(AIToolDefinition{
		Name:                "cluster.overview",
		Category:            "query",
		Description:         "Platform cluster overview reused from dashboard capability",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15 * time.Second,
		InputSchema:         aiClusterInputSchema(),
		OutputSchema:        aiToolOutputSchema("cluster.overview"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return clusterSvc.GetClusterOverview(ctx, req.ClusterID)
		},
	})

	r.register(AIToolDefinition{
		Name:                "cluster.health",
		Category:            "query",
		Description:         "Cluster API and workload health from platform overview",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             10 * time.Second,
		InputSchema:         aiClusterInputSchema(),
		OutputSchema:        aiToolOutputSchema("cluster.health"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return clusterSvc.GetClusterHealth(ctx, req.ClusterID)
		},
	})

	r.register(AIToolDefinition{
		Name:                "cluster.inventory",
		Category:            "query",
		Description:         "Cluster overview alias for backward compatibility",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15 * time.Second,
		InputSchema:         aiClusterInputSchema(),
		OutputSchema:        aiToolOutputSchema("cluster.inventory"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return clusterSvc.GetClusterInventory(ctx, req.ClusterID)
		},
	})

	r.register(AIToolDefinition{
		Name:                "cluster.certificate_risks",
		Category:            "query",
		Description:         "Control plane and cluster certificate risk overview from dashboard capability",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15 * time.Second,
		InputSchema:         aiClusterInputSchema(),
		OutputSchema:        aiToolOutputSchema("cluster.certificate_risks"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return clusterSvc.GetClusterCertificateRisks(ctx, req.ClusterID)
		},
	})

	r.register(AIToolDefinition{
		Name:                "namespace.workloads",
		Category:            "query",
		Description:         "Namespace workload inventory with names and replica status reused from platform workload views",
		RequiredPermissions: []string{"ai:tool_exec", "namespace:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15 * time.Second,
		InputSchema:         aiNamespaceInputSchema(),
		OutputSchema:        aiToolOutputSchema("namespace.workloads"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			value, err := namespaceSvc.NamespaceWorkloadInventory(ctx, req.ClusterID, req.Namespace)
			return aiNamespaceDiagnosisToolResult(kopsapp.NamespaceDiagnosisResult{Summary: value.Summary, Evidence: value.Evidence, RawRef: value.RawRef}), mapKopsInspectionError(err)
		},
	})

	r.register(AIToolDefinition{
		Name:                "namespace.inspect",
		Category:            "query",
		Description:         "Full namespace inspection with health, counts, pod metrics and workload inventory",
		RequiredPermissions: []string{"ai:tool_exec", "namespace:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema:         aiNamespaceInputSchema(),
		OutputSchema:        aiToolOutputSchema("namespace.inspect"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			value, err := namespaceSvc.NamespaceInspection(ctx, req.ClusterID, req.Namespace)
			return aiNamespaceDiagnosisToolResult(value), mapKopsInspectionError(err)
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
		InputSchema:         aiNamespaceInputSchema(),
		OutputSchema:        aiToolOutputSchema("namespace.health"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			value, err := namespaceSvc.NamespaceHealth(ctx, req.ClusterID, req.Namespace)
			return aiNamespaceDiagnosisToolResult(value), mapKopsInspectionError(err)
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
		InputSchema:         aiNamespaceInputSchema(),
		OutputSchema:        aiToolOutputSchema("namespace.summary"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			value, err := namespaceSvc.NamespaceResourceSummary(ctx, req.ClusterID, req.Namespace)
			return aiNamespaceDiagnosisToolResult(value), mapKopsInspectionError(err)
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
		InputSchema:         aiScopedResourceInputSchema("Pod"),
		OutputSchema:        aiToolOutputSchema("pod.inspect"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			value, err := inspectionSvc.Pod(ctx, req.ClusterID, req.Namespace, req.ResourceName)
			return aiInspectionToolResult(value), mapKopsInspectionError(err)
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
		InputSchema:         aiClusterScopedNameInputSchema("Node"),
		OutputSchema:        aiToolOutputSchema("node.inspect"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			value, err := inspectionSvc.Node(ctx, req.ClusterID, req.ResourceName)
			return aiInspectionToolResult(value), mapKopsInspectionError(err)
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
		InputSchema:         aiScopedResourceInputSchema("Deployment"),
		OutputSchema:        aiToolOutputSchema("deployment.inspect"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			value, err := inspectionSvc.Deployment(ctx, req.ClusterID, req.Namespace, req.ResourceName)
			return aiInspectionToolResult(value), mapKopsInspectionError(err)
		},
	})

	r.register(AIToolDefinition{
		Name:                "resource.list",
		Category:            "query",
		Description:         "List supported Kubernetes resources through the shared platform query service",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":     "number",
			"kind":           "string",
			"namespace":      "string?",
			"label_selector": "string?",
			"sort_by":        "string?",
			"order":          "string?",
			"limit":          "number?",
		},
		OutputSchema: aiToolOutputSchema("resource.list"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			limit, _ := jsonIntValue(input["limit"])
			return resourceQuerySvc.ListResources(
				ctx,
				req.ClusterID,
				strings.TrimSpace(fmt.Sprint(input["kind"])),
				strings.TrimSpace(fmt.Sprint(input["namespace"])),
				strings.TrimSpace(fmt.Sprint(input["label_selector"])),
				strings.TrimSpace(fmt.Sprint(input["sort_by"])),
				strings.TrimSpace(fmt.Sprint(input["order"])),
				limit,
			)
		},
	})

	r.register(AIToolDefinition{
		Name:                "resource.search",
		Category:            "query",
		Description:         "Search resources by name keyword through the shared platform query service",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id": "number",
			"keyword":    "string",
			"namespace":  "string?",
			"kinds":      []string{"string"},
			"limit":      "number?",
		},
		OutputSchema: aiToolOutputSchema("resource.search"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			limit, _ := jsonIntValue(input["limit"])
			return resourceQuerySvc.SearchResources(
				ctx,
				req.ClusterID,
				strings.TrimSpace(fmt.Sprint(input["keyword"])),
				strings.TrimSpace(fmt.Sprint(input["namespace"])),
				aiapp.StringSliceInput(input["kinds"]),
				limit,
			)
		},
	})

	r.register(AIToolDefinition{
		Name:                "resource.inspect",
		Category:            "inspect",
		Description:         "Inspect a supported Kubernetes resource with masking policy",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		RedactionPolicy:     "mask_sensitive",
		InputSchema:         aiGenericResourceInputSchema(),
		OutputSchema:        aiToolOutputSchema("resource.inspect"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			kind := strings.TrimSpace(fmt.Sprint(input["kind"]))
			namespace := strings.TrimSpace(fmt.Sprint(input["namespace"]))
			name := strings.TrimSpace(fmt.Sprint(input["name"]))
			value, err := inspectionSvc.Resource(ctx, req.ClusterID, kind, namespace, name)
			if err != nil {
				return AIToolResult{}, mapKopsInspectionError(err)
			}
			return aiResourceInspectionToolResult(value, exportPolicySvc), nil
		},
	})

	r.register(AIToolDefinition{
		Name:                "resource.events",
		Category:            "inspect",
		Description:         "Collect recent events for a specific Kubernetes resource",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema:         aiGenericResourceInputSchema(),
		OutputSchema:        aiToolOutputSchema("resource.events"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			return resourceQuerySvc.GetResourceEvents(
				ctx,
				req.ClusterID,
				strings.TrimSpace(fmt.Sprint(input["kind"])),
				strings.TrimSpace(fmt.Sprint(input["namespace"])),
				strings.TrimSpace(fmt.Sprint(input["name"])),
			)
		},
	})

	r.register(AIToolDefinition{
		Name:                "resource.logs",
		Category:            "inspect",
		Description:         "Collect recent logs for a Pod or workload-related Pod",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             25 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id": "number",
			"kind":       "string",
			"namespace":  "string?",
			"name":       "string",
			"container":  "string?",
			"tail_lines": "number?",
			"previous":   "boolean?",
		},
		OutputSchema: aiToolOutputSchema("resource.logs"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			tailLines, _ := jsonIntValue(input["tail_lines"])
			return resourceQuerySvc.GetResourceLogs(
				ctx,
				req.ClusterID,
				strings.TrimSpace(fmt.Sprint(input["kind"])),
				strings.TrimSpace(fmt.Sprint(input["namespace"])),
				strings.TrimSpace(fmt.Sprint(input["name"])),
				strings.TrimSpace(fmt.Sprint(input["container"])),
				int64(tailLines),
				aiBoolValue(input["previous"]),
			)
		},
	})

	r.register(AIToolDefinition{
		Name:                "resource.related",
		Category:            "inspect",
		Description:         "Find related pods and controllers for a specific resource",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema:         aiGenericResourceInputSchema(),
		OutputSchema:        aiToolOutputSchema("resource.related"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			return resourceQuerySvc.GetRelatedResources(
				ctx,
				req.ClusterID,
				strings.TrimSpace(fmt.Sprint(input["kind"])),
				strings.TrimSpace(fmt.Sprint(input["namespace"])),
				strings.TrimSpace(fmt.Sprint(input["name"])),
			)
		},
	})

	r.register(AIToolDefinition{
		Name:                "proposal.node.cordon",
		Category:            "proposal",
		Description:         "Create a node cordon proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema:         aiClusterScopedNameInputSchema("Node"),
		OutputSchema:        aiToolOutputSchema("proposal.node.cordon"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeCordonNode,
				TargetResource: AIActionTargetResource{Kind: "Node", Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload:        model.JSONMap{},
				Reason:         strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.node.uncordon",
		Category:            "proposal",
		Description:         "Create a node uncordon proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema:         aiClusterScopedNameInputSchema("Node"),
		OutputSchema:        aiToolOutputSchema("proposal.node.uncordon"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeUncordonNode,
				TargetResource: AIActionTargetResource{Kind: "Node", Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload:        model.JSONMap{},
				Reason:         strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.node.drain",
		Category:            "proposal",
		Description:         "Create a node drain proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "high",
		ConfirmLevel:        "double",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":      "number",
			"resource_kind":   "Node",
			"name":            "string",
			"timeout_seconds": "number?",
			"force":           "boolean?",
			"reason":          "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.node.drain"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeDrainNode,
				TargetResource: AIActionTargetResource{Kind: "Node", Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload: model.JSONMap{
					"timeout_seconds": input["timeout_seconds"],
					"force":           input["force"],
				},
				Reason: strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.workload.image",
		Category:            "proposal",
		Description:         "Create a workload image update proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":     "number",
			"resource_kind":  "Deployment|StatefulSet|DaemonSet",
			"namespace":      "string",
			"name":           "string",
			"container_name": "string",
			"image":          "string",
			"reason":         "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.workload.image"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeUpdateWorkloadImage,
				TargetResource: AIActionTargetResource{Kind: strings.TrimSpace(fmt.Sprint(input["resource_kind"])), Namespace: strings.TrimSpace(fmt.Sprint(input["namespace"])), Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload: model.JSONMap{
					"container_name": strings.TrimSpace(fmt.Sprint(input["container_name"])),
					"image":          strings.TrimSpace(fmt.Sprint(input["image"])),
				},
				Reason: strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.workload.pause",
		Category:            "proposal",
		Description:         "Create a workload rollout pause or resume proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":    "number",
			"resource_kind": "Deployment",
			"namespace":     "string",
			"name":          "string",
			"paused":        "boolean?",
			"reason":        "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.workload.pause"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypePauseWorkloadRollout,
				TargetResource: AIActionTargetResource{Kind: "Deployment", Namespace: strings.TrimSpace(fmt.Sprint(input["namespace"])), Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload:        model.JSONMap{"paused": input["paused"]},
				Reason:         strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.workload.undo",
		Category:            "proposal",
		Description:         "Create a workload rollout undo proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":    "number",
			"resource_kind": "Deployment",
			"namespace":     "string",
			"name":          "string",
			"revision":      "number?",
			"reason":        "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.workload.undo"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeRolloutUndo,
				TargetResource: AIActionTargetResource{Kind: "Deployment", Namespace: strings.TrimSpace(fmt.Sprint(input["namespace"])), Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload:        model.JSONMap{"revision": input["revision"]},
				Reason:         strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.workload.delete",
		Category:            "proposal",
		Description:         "Create a workload delete proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "high",
		ConfirmLevel:        "double",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":    "number",
			"resource_kind": "Deployment|StatefulSet|DaemonSet",
			"namespace":     "string",
			"name":          "string",
			"reason":        "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.workload.delete"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeDeleteWorkload,
				TargetResource: AIActionTargetResource{Kind: strings.TrimSpace(fmt.Sprint(input["resource_kind"])), Namespace: strings.TrimSpace(fmt.Sprint(input["namespace"])), Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload:        model.JSONMap{},
				Reason:         strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.resource.delete",
		Category:            "proposal",
		Description:         "Create a generic resource delete proposal for network, config, or storage resources",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "high",
		ConfirmLevel:        "double",
		Timeout:             20 * time.Second,
		InputSchema:         aiGenericResourceInputSchema(),
		OutputSchema:        aiToolOutputSchema("proposal.resource.delete"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeDeleteResource,
				TargetResource: AIActionTargetResource{Kind: strings.TrimSpace(fmt.Sprint(input["kind"])), Namespace: strings.TrimSpace(fmt.Sprint(input["namespace"])), Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload:        model.JSONMap{},
				Reason:         strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.network.apply",
		Category:            "proposal",
		Description:         "Create a network domain manifest proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":        "number",
			"default_namespace": "string?",
			"yaml":              "string",
			"title":             "string?",
			"reason":            "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.network.apply"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return buildApplyManifestProposalRequest(req, input)
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.config.apply",
		Category:            "proposal",
		Description:         "Create a config domain manifest proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":        "number",
			"default_namespace": "string?",
			"yaml":              "string",
			"title":             "string?",
			"reason":            "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.config.apply"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return buildApplyManifestProposalRequest(req, input)
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.storage.apply",
		Category:            "proposal",
		Description:         "Create a storage domain manifest proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":        "number",
			"default_namespace": "string?",
			"yaml":              "string",
			"title":             "string?",
			"reason":            "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.storage.apply"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return buildApplyManifestProposalRequest(req, input)
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.batch.trigger_cronjob",
		Category:            "proposal",
		Description:         "Create a CronJob trigger proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":    "number",
			"resource_kind": "CronJob",
			"namespace":     "string",
			"name":          "string",
			"reason":        "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.batch.trigger_cronjob"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeTriggerCronJob,
				TargetResource: AIActionTargetResource{Kind: "CronJob", Namespace: strings.TrimSpace(fmt.Sprint(input["namespace"])), Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload:        model.JSONMap{},
				Reason:         strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.batch.suspend_cronjob",
		Category:            "proposal",
		Description:         "Create a CronJob suspend or resume proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":    "number",
			"resource_kind": "CronJob",
			"namespace":     "string",
			"name":          "string",
			"suspend":       "boolean?",
			"reason":        "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.batch.suspend_cronjob"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeSuspendCronJob,
				TargetResource: AIActionTargetResource{Kind: "CronJob", Namespace: strings.TrimSpace(fmt.Sprint(input["namespace"])), Name: strings.TrimSpace(fmt.Sprint(input["name"]))},
				Payload:        model.JSONMap{"suspend": input["suspend"]},
				Reason:         strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
	})

	r.register(AIToolDefinition{
		Name:                "proposal.batch.delete_completed_jobs",
		Category:            "proposal",
		Description:         "Create a completed Jobs cleanup proposal using the shared platform action service",
		RequiredPermissions: []string{"ai:tool_exec", "ai:change_propose", "k8s:write"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id":       "number",
			"namespace":        "string",
			"older_than_hours": "number?",
			"reason":           "string?",
		},
		OutputSchema: aiToolOutputSchema("proposal.batch.delete_completed_jobs"),
		Handler: createProposalHandler(func(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
			return CreateAIActionProposalRequest{
				ConversationID: req.ConversationID,
				MessageID:      ptrUint64(req.MessageID),
				ProposalType:   aiActionTypeDeleteCompletedJobs,
				TargetResource: AIActionTargetResource{Kind: "Job", Namespace: strings.TrimSpace(fmt.Sprint(input["namespace"]))},
				Payload: model.JSONMap{
					"namespace":        strings.TrimSpace(fmt.Sprint(input["namespace"])),
					"older_than_hours": input["older_than_hours"],
				},
				Reason: strings.TrimSpace(fmt.Sprint(input["reason"])),
			}, nil
		}),
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
		InputSchema:         aiGenericResourceInputSchema(),
		OutputSchema:        aiToolOutputSchema("resource.yaml"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			kind := strings.TrimSpace(fmt.Sprint(input["kind"]))
			namespace := strings.TrimSpace(fmt.Sprint(input["namespace"]))
			name := strings.TrimSpace(fmt.Sprint(input["name"]))
			value, err := inspectionSvc.ResourceYAML(ctx, req.ClusterID, kind, namespace, name)
			if err != nil {
				return AIToolResult{}, mapKopsInspectionError(err)
			}
			return aiResourceYAMLToolResult(value, exportPolicySvc, false)
		},
	})

	r.register(AIToolDefinition{
		Name:                "resource.masked_yaml",
		Category:            "export",
		Description:         "Export resource YAML with forced masking for sensitive resources",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "medium",
		ConfirmLevel:        "single",
		Timeout:             20 * time.Second,
		RedactionPolicy:     "force_mask_sensitive",
		InputSchema:         aiGenericResourceInputSchema(),
		OutputSchema:        aiToolOutputSchema("resource.masked_yaml"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			kind := strings.TrimSpace(fmt.Sprint(input["kind"]))
			namespace := strings.TrimSpace(fmt.Sprint(input["namespace"]))
			name := strings.TrimSpace(fmt.Sprint(input["name"]))
			value, err := inspectionSvc.ResourceYAML(ctx, req.ClusterID, kind, namespace, name)
			if err != nil {
				return AIToolResult{}, mapKopsInspectionError(err)
			}
			return aiResourceYAMLToolResult(value, exportPolicySvc, true)
		},
	})

	// ─── 平台级只读查询工具 ───────────────────────────────────

	// platform.helm.releases — 列出指定集群和命名空间下的 Helm releases
	r.register(AIToolDefinition{
		Name:                "platform.helm.releases",
		Category:            "query",
		Description:         "列出指定集群和命名空间下的 Helm releases，包含 release 名称、chart、版本、状态等信息",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             15 * time.Second,
		InputSchema: model.JSONMap{
			"cluster_id": "uint64",
			"namespace":  "string?",
		},
		OutputSchema: aiToolOutputSchema("platform.helm.releases"),
		Handler: func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error) {
			ns := strings.TrimSpace(fmt.Sprint(input["namespace"]))
			// Helm v3 将 release 存储在 Secret 中，通过 label owner=helm 标识
			return resourceQuerySvc.ListResources(ctx, req.ClusterID, "Secret", ns, "owner=helm", "", "", 0)
		},
	})

	// platform.projects — 列出平台项目列表
	r.register(AIToolDefinition{
		Name:                "platform.projects",
		Category:            "query",
		Description:         "列出平台所有项目，包含项目名称、描述、关联集群、命名空间及配额信息",
		RequiredPermissions: []string{"ai:tool_exec", "namespace:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             10 * time.Second,
		InputSchema:         model.JSONMap{},
		OutputSchema:        aiToolOutputSchema("platform.projects"),
		Handler: func(ctx context.Context, _ AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			if r.db == nil {
				return AIToolResult{}, ErrWithMessage(ErrNotFound, "database not available")
			}
			var projects []aiProjectRow
			if err := r.db.WithContext(ctx).Find(&projects).Error; err != nil {
				return AIToolResult{}, ErrWithMessage(ErrNotFound, "查询项目列表失败: "+err.Error())
			}
			items := make([]map[string]any, 0, len(projects))
			for _, p := range projects {
				items = append(items, map[string]any{
					"id":           p.ID,
					"name":         p.Name,
					"description":  p.Description,
					"cluster_id":   p.ClusterID,
					"namespaces":   p.Namespaces,
					"quota_cpu":    p.QuotaCPU,
					"quota_memory": p.QuotaMemory,
					"quota_pods":   p.QuotaPods,
				})
			}
			summary := fmt.Sprintf("共 %d 个项目", len(projects))
			return AIToolResult{
				Summary:  summary,
				Evidence: model.JSONMap{"items": items, "total": len(projects)},
			}, nil
		},
	})

	// platform.app_templates — 列出应用商店模板
	r.register(AIToolDefinition{
		Name:                "platform.app_templates",
		Category:            "query",
		Description:         "列出应用商店中所有可用模板，包含模板名称、分类、部署类型等信息",
		RequiredPermissions: []string{"ai:tool_exec", "namespace:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             10 * time.Second,
		InputSchema:         model.JSONMap{},
		OutputSchema:        aiToolOutputSchema("platform.app_templates"),
		Handler: func(ctx context.Context, _ AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			if r.db == nil {
				return AIToolResult{}, ErrWithMessage(ErrNotFound, "database not available")
			}
			var templates []provisiondomain.AppTemplate
			if err := r.db.WithContext(ctx).Find(&templates).Error; err != nil {
				return AIToolResult{}, ErrWithMessage(ErrNotFound, "查询应用模板列表失败: "+err.Error())
			}
			items := make([]map[string]any, 0, len(templates))
			for _, t := range templates {
				items = append(items, map[string]any{
					"id":           t.ID,
					"name":         t.Name,
					"display_name": t.DisplayName,
					"description":  t.Description,
					"category":     t.Category,
					"icon":         t.Icon,
					"deploy_type":  t.DeployType,
					"is_builtin":   t.IsBuiltin,
				})
			}
			summary := fmt.Sprintf("共 %d 个应用模板", len(templates))
			return AIToolResult{
				Summary:  summary,
				Evidence: model.JSONMap{"items": items, "total": len(templates)},
			}, nil
		},
	})

	// platform.clusters — 列出所有集群
	r.register(AIToolDefinition{
		Name:                "platform.clusters",
		Category:            "query",
		Description:         "列出平台所有集群，包含集群名称、状态、K8s 版本和节点数等信息",
		RequiredPermissions: []string{"ai:tool_exec", "k8s:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             10 * time.Second,
		InputSchema:         model.JSONMap{},
		OutputSchema:        aiToolOutputSchema("platform.clusters"),
		Handler: func(ctx context.Context, _ AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			if r.db == nil {
				return AIToolResult{}, ErrWithMessage(ErrNotFound, "database not available")
			}
			var clusters []fleetmysql.ClusterRow
			if err := r.db.WithContext(ctx).Find(&clusters).Error; err != nil {
				return AIToolResult{}, ErrWithMessage(ErrNotFound, "查询集群列表失败: "+err.Error())
			}
			items := make([]map[string]any, 0, len(clusters))
			for _, c := range clusters {
				items = append(items, map[string]any{
					"id":          c.ID,
					"name":        c.Name,
					"type":        c.Type,
					"status":      c.Status,
					"k8s_version": c.K8sVersion,
					"node_count":  c.NodeCount,
				})
			}
			summary := fmt.Sprintf("共 %d 个集群", len(clusters))
			return AIToolResult{
				Summary:  summary,
				Evidence: model.JSONMap{"items": items, "total": len(clusters)},
			}, nil
		},
	})

	// platform.users — 列出平台用户
	r.register(AIToolDefinition{
		Name:                "platform.users",
		Category:            "query",
		Description:         "列出平台所有用户，包含用户名、邮箱和状态信息（不包含密码等敏感字段）",
		RequiredPermissions: []string{"ai:tool_exec", "namespace:read"},
		RiskLevel:           "low",
		ConfirmLevel:        "single",
		Timeout:             10 * time.Second,
		InputSchema:         model.JSONMap{},
		OutputSchema:        aiToolOutputSchema("platform.users"),
		Handler: func(ctx context.Context, _ AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			if r.db == nil {
				return AIToolResult{}, ErrWithMessage(ErrNotFound, "database not available")
			}
			var users []struct {
				ID       uint64 `gorm:"column:id"`
				Username string `gorm:"column:username"`
				Email    string `gorm:"column:email"`
				Status   string `gorm:"column:status"`
			}
			if err := r.db.WithContext(ctx).Table("users").Where("deleted_at IS NULL").Find(&users).Error; err != nil {
				return AIToolResult{}, ErrWithMessage(ErrNotFound, "查询用户列表失败: "+err.Error())
			}
			items := make([]map[string]any, 0, len(users))
			for _, u := range users {
				items = append(items, map[string]any{
					"id":       u.ID,
					"username": u.Username,
					"email":    u.Email,
					"status":   u.Status,
				})
			}
			summary := fmt.Sprintf("共 %d 个用户", len(users))
			return AIToolResult{
				Summary:  summary,
				Evidence: model.JSONMap{"items": items, "total": len(users)},
			}, nil
		},
	})

	return r
}

// NewAIToolRegistry remains an adapter-level compatibility constructor for
// callers that still use the former registry name.
func NewAIToolRegistry(
	db *gorm.DB,
	clusterSvc clusterReadModel,
	namespaceSvc kopsapp.NamespaceDiagnosisReader,
	inspectionSvc *kopsapp.InspectionService,
	resourceQuerySvc aiapp.ResourceQueryPort,
	actionSvc *ActionRuntime,
	exportPolicySvc *aiapp.ResourceExportPolicyService,
) *AIToolRegistry {
	return NewToolRegistry(db, clusterSvc, namespaceSvc, inspectionSvc, resourceQuerySvc, actionSvc, exportPolicySvc)
}

func (r *AIToolRegistry) register(def AIToolDefinition) {
	if r == nil {
		return
	}
	if r.catalog == nil {
		r.catalog = aiapp.NewToolCatalog()
	}
	r.catalog.Register(def)
}

func (r *AIToolRegistry) Get(name string) (AIToolDefinition, bool) {
	if r == nil || r.catalog == nil {
		return AIToolDefinition{}, false
	}
	return r.catalog.Get(name)
}

func (r *AIToolRegistry) ValidateToolPermissions(required, userPerms []string, toolName string) error {
	return toolPermissionErr(required, userPerms, toolName)
}

func (r *AIToolRegistry) List() []AIToolDefinition {
	if r == nil {
		return nil
	}
	if r.catalog == nil {
		return []AIToolDefinition{}
	}
	return r.catalog.List()
}

func (r *AIToolRegistry) ListCatalog(userPerms []string) []AIToolCatalogItem {
	if r == nil || r.catalog == nil {
		return []AIToolCatalogItem{}
	}
	return r.catalog.ListCatalog(userPerms)
}

func (r *AIToolRegistry) PlanAutoDiagnostics(req AIToolContextRequest) []AIToolPlanStep {
	if r == nil || r.catalog == nil {
		return nil
	}
	return r.catalog.PlanAutoDiagnostics(req)
}

func toolPermissionErr(required, userPerms []string, toolName string) error {
	missing := aiapp.MissingToolPermissions(userPerms, required)
	if len(missing) == 0 {
		return nil
	}
	return ErrWithMessage(ErrK8sForbidden, fmt.Sprintf("tool %s requires permissions: %s", toolName, strings.Join(missing, ", ")))
}

func aiClusterInputSchema() model.JSONMap {
	return aiapp.ClusterToolInputSchema()
}

func aiNamespaceInputSchema() model.JSONMap {
	return aiapp.NamespaceToolInputSchema()
}

func aiScopedResourceInputSchema(kind string) model.JSONMap {
	return aiapp.ScopedResourceToolInputSchema(kind)
}

func aiClusterScopedNameInputSchema(kind string) model.JSONMap {
	return aiapp.ClusterScopedNameToolInputSchema(kind)
}

func aiGenericResourceInputSchema() model.JSONMap {
	return aiapp.GenericResourceToolInputSchema()
}

func aiToolOutputSchema(source string) model.JSONMap {
	return aiapp.ToolOutputSchema(source)
}

func buildApplyManifestProposalRequest(req AIToolContextRequest, input map[string]any) (CreateAIActionProposalRequest, error) {
	yamlText := strings.TrimSpace(fmt.Sprint(input["yaml"]))
	if yamlText == "" {
		return CreateAIActionProposalRequest{}, ErrWithMessage(ErrInvalidParams, "yaml is required for manifest proposal tools")
	}
	return CreateAIActionProposalRequest{
		ConversationID: req.ConversationID,
		MessageID:      ptrUint64(req.MessageID),
		ProposalType:   aiActionTypeApplyManifest,
		TargetResource: AIActionTargetResource{
			Namespace: strings.TrimSpace(fmt.Sprint(input["default_namespace"])),
		},
		Payload: model.JSONMap{
			"yaml":              yamlText,
			"default_namespace": strings.TrimSpace(fmt.Sprint(input["default_namespace"])),
			"title":             strings.TrimSpace(fmt.Sprint(input["title"])),
		},
		Reason: strings.TrimSpace(fmt.Sprint(input["reason"])),
	}, nil
}
