package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	aiapp "k8s-platform-backend/internal/ai/application"
	"k8s-platform-backend/internal/legacy/model"
	provisiondomain "k8s-platform-backend/internal/provisioning/domain"
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
	InputSchema         model.JSONMap
	OutputSchema        model.JSONMap
	Handler             func(ctx context.Context, req AIToolContextRequest, input map[string]any) (AIToolResult, error)
}

type AIToolPlanStep struct {
	ToolName string        `json:"tool_name"`
	Params   model.JSONMap `json:"params,omitempty"`
	Reason   string        `json:"reason,omitempty"`
}

type AIToolCatalogItem struct {
	Name                string        `json:"name"`
	Category            string        `json:"category"`
	Description         string        `json:"description"`
	RequiredPermissions []string      `json:"required_permissions"`
	MissingPermissions  []string      `json:"missing_permissions,omitempty"`
	RiskLevel           string        `json:"risk_level"`
	ConfirmLevel        string        `json:"confirm_level"`
	TimeoutSeconds      int64         `json:"timeout_seconds"`
	RedactionPolicy     string        `json:"redaction_policy,omitempty"`
	InputSchema         model.JSONMap `json:"input_schema,omitempty"`
	OutputSchema        model.JSONMap `json:"output_schema,omitempty"`
	Available           bool          `json:"available"`
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

type AIToolRegistry struct {
	db    *gorm.DB
	defs  map[string]AIToolDefinition
	order []string
}

func NewAIToolRegistry(
	db *gorm.DB,
	clusterSvc *ClusterReadModelService,
	namespaceSvc *NamespaceDiagnosisService,
	inspectionSvc *ResourceInspectionService,
	resourceQuerySvc *ResourceQueryService,
	actionSvc *AIActionService,
	exportPolicySvc *aiapp.ResourceExportPolicyService,
) *AIToolRegistry {
	r := &AIToolRegistry{
		db:    db,
		defs:  map[string]AIToolDefinition{},
		order: make([]string, 0, 12),
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
			return namespaceSvc.GetNamespaceWorkloadInventory(ctx, req.ClusterID, req.Namespace)
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
			return namespaceSvc.GetNamespaceInspection(ctx, req.ClusterID, req.Namespace)
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
		InputSchema:         aiNamespaceInputSchema(),
		OutputSchema:        aiToolOutputSchema("namespace.summary"),
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
		InputSchema:         aiScopedResourceInputSchema("Pod"),
		OutputSchema:        aiToolOutputSchema("pod.inspect"),
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
		InputSchema:         aiClusterScopedNameInputSchema("Node"),
		OutputSchema:        aiToolOutputSchema("node.inspect"),
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
		InputSchema:         aiScopedResourceInputSchema("Deployment"),
		OutputSchema:        aiToolOutputSchema("deployment.inspect"),
		Handler: func(ctx context.Context, req AIToolContextRequest, _ map[string]any) (AIToolResult, error) {
			return inspectionSvc.InspectDeployment(ctx, req.ClusterID, req.Namespace, req.ResourceName)
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
				aiStringSliceValue(input["kinds"]),
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
			return inspectionSvc.InspectResource(ctx, req.ClusterID, kind, namespace, name, exportPolicySvc)
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
			return inspectionSvc.ExportResourceYAML(ctx, req.ClusterID, kind, namespace, name, exportPolicySvc)
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
			return inspectionSvc.ExportMaskedResourceYAML(ctx, req.ClusterID, kind, namespace, name, exportPolicySvc)
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
			var clusters []model.Cluster
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

func (r *AIToolRegistry) ListCatalog(userPerms []string) []AIToolCatalogItem {
	defs := r.List()
	items := make([]AIToolCatalogItem, 0, len(defs))
	for _, def := range defs {
		missing := missingPermissions(userPerms, def.RequiredPermissions)
		items = append(items, AIToolCatalogItem{
			Name:                def.Name,
			Category:            def.Category,
			Description:         def.Description,
			RequiredPermissions: append([]string(nil), def.RequiredPermissions...),
			MissingPermissions:  missing,
			RiskLevel:           def.RiskLevel,
			ConfirmLevel:        def.ConfirmLevel,
			TimeoutSeconds:      int64(def.Timeout / time.Second),
			RedactionPolicy:     def.RedactionPolicy,
			InputSchema:         cloneJSONMap(def.InputSchema),
			OutputSchema:        cloneJSONMap(def.OutputSchema),
			Available:           len(missing) == 0,
		})
	}
	return items
}

func (r *AIToolRegistry) PlanAutoDiagnostics(req AIToolContextRequest) []AIToolPlanStep {
	if r == nil || req.ClusterID == 0 {
		return nil
	}

	steps := make([]AIToolPlanStep, 0, 6)
	add := func(toolName string, reason string, params model.JSONMap) {
		if _, ok := r.Get(toolName); !ok {
			return
		}
		steps = append(steps, AIToolPlanStep{
			ToolName: toolName,
			Params:   cloneJSONMap(params),
			Reason:   strings.TrimSpace(reason),
		})
	}

	query := strings.TrimSpace(req.Query)
	namespace := strings.TrimSpace(req.Namespace)
	kind := strings.TrimSpace(req.ResourceKind)
	name := strings.TrimSpace(req.ResourceName)
	strictResourceScope := kind != "" && name != ""
	broadInspection := aiNeedsBroadInspection(query)
	clusterInspection := aiNeedsClusterInspection(query)
	controlPlaneInspection := aiNeedsControlPlaneInspection(query)
	broadensScope := aiExplicitlyBroadensResourceScope(query)
	yamlIntent := aiNeedsResourceYAML(query)

	if !strictResourceScope || clusterInspection || controlPlaneInspection || broadInspection || broadensScope {
		add("cluster.health", "Baseline cluster health reused from platform read model", model.JSONMap{
			"cluster_id": req.ClusterID,
		})
	}
	if !strictResourceScope && (namespace == "" || broadInspection || clusterInspection) {
		add("cluster.overview", "Cluster-wide overview requested by the current question scope", model.JSONMap{
			"cluster_id": req.ClusterID,
		})
	}
	if !strictResourceScope && namespace == "" && (controlPlaneInspection || broadInspection) {
		add("cluster.certificate_risks", "Control plane or certificate risk inspection requested", model.JSONMap{
			"cluster_id": req.ClusterID,
		})
	}

	if namespace != "" {
		if strictResourceScope {
			if broadInspection || broadensScope {
				add("namespace.inspect", "The current question explicitly broadens from the target resource to the namespace scope", model.JSONMap{
					"cluster_id": req.ClusterID,
					"namespace":  namespace,
				})
			} else if aiNeedsPodOwnershipContext(kind, query) {
				add("namespace.workloads", "Scoped pod inspection needs workload inventory to explain ownership", model.JSONMap{
					"cluster_id": req.ClusterID,
					"namespace":  namespace,
				})
			}
		} else if broadInspection {
			add("namespace.inspect", "Full namespace inspection is needed for the current scoped question", model.JSONMap{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
			})
		} else {
			add("namespace.health", "Scoped namespace health evidence is needed", model.JSONMap{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
			})
			add("namespace.summary", "Scoped namespace inventory summary is needed", model.JSONMap{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
			})
			if strings.EqualFold(kind, "pod") || containsAny(strings.ToLower(query), "deployment", "statefulset", "daemonset", "replicaset", "workload", "replica") {
				add("namespace.workloads", "Scoped workload inventory is needed to explain pod ownership and replica context", model.JSONMap{
					"cluster_id": req.ClusterID,
					"namespace":  namespace,
				})
			}
		}
	}

	// 用户指定了资源类型但未指定名称时，先列出该类型的资源
	if kind != "" && name == "" && namespace != "" {
		add("resource.list", "用户指定了资源类型但未指定名称，先列出该类型的资源", model.JSONMap{
			"cluster_id": req.ClusterID,
			"namespace":  namespace,
			"kind":       kind,
		})
		return steps
	}

	if kind == "" || name == "" {
		// 有命名空间且检测到配置/数据库相关意图时，发现命名空间下的配置资源和工作负载
		if namespace != "" && (yamlIntent || aiNeedsConfigSearch(query)) {
			// 列出工作负载，LLM 可从中发现 Redis 相关的 Deployment/StatefulSet
			add("namespace.workloads", "列出工作负载，帮助发现 Redis 相关应用及其环境变量配置", model.JSONMap{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
			})
			// 列出 ConfigMap 和 Secret 名称
			add("resource.list", "用户可能需要查看配置资源，先列出命名空间下的 ConfigMap", model.JSONMap{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
				"kind":       "ConfigMap",
			})
			add("resource.list", "用户可能需要查看密钥资源，先列出命名空间下的 Secret", model.JSONMap{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
				"kind":       "Secret",
			})
			// 列出 Service，帮助发现 Redis Service
			add("resource.list", "列出 Service，帮助发现 Redis 服务", model.JSONMap{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
				"kind":       "Service",
			})
		}
		return steps
	}

	switch strings.ToLower(kind) {
	case "pod":
		add("pod.inspect", "The question targets a specific pod", model.JSONMap{
			"cluster_id": req.ClusterID,
			"namespace":  namespace,
			"name":       name,
		})
	case "node":
		add("node.inspect", "The question targets a specific node", model.JSONMap{
			"cluster_id": req.ClusterID,
			"name":       name,
		})
	case "deployment":
		add("deployment.inspect", "The question targets a specific deployment", model.JSONMap{
			"cluster_id": req.ClusterID,
			"namespace":  namespace,
			"name":       name,
		})
	default:
		add("resource.inspect", "The question targets a specific Kubernetes resource", model.JSONMap{
			"cluster_id": req.ClusterID,
			"kind":       kind,
			"namespace":  namespace,
			"name":       name,
		})
	}
	if aiNeedsResourceEvents(query) {
		add("resource.events", "The user asked for warning reasons or event evidence on the current target resource", model.JSONMap{
			"cluster_id": req.ClusterID,
			"kind":       kind,
			"namespace":  namespace,
			"name":       name,
		})
	}
	if aiNeedsResourceLogs(kind, query) {
		add("resource.logs", "The user asked for logs or crash evidence on the current target scope", model.JSONMap{
			"cluster_id": req.ClusterID,
			"kind":       kind,
			"namespace":  namespace,
			"name":       name,
			"tail_lines": 80,
		})
	}
	if aiNeedsRelatedResources(query) {
		add("resource.related", "The user asked for ownership, dependency, or related resource context", model.JSONMap{
			"cluster_id": req.ClusterID,
			"kind":       kind,
			"namespace":  namespace,
			"name":       name,
		})
	}
	if yamlIntent {
		yamlToolName := "resource.yaml"
		if strings.EqualFold(kind, "secret") {
			yamlToolName = "resource.masked_yaml"
		}
		add(yamlToolName, "The user explicitly asked for YAML or manifest details", model.JSONMap{
			"cluster_id": req.ClusterID,
			"kind":       kind,
			"namespace":  namespace,
			"name":       name,
		})
	}

	return steps
}

func aiExplicitlyBroadensResourceScope(query string) bool {
	text := strings.ToLower(strings.TrimSpace(query))
	if text == "" {
		return false
	}
	return containsAny(text,
		"all workloads",
		"all deployments",
		"namespace-wide",
		"entire namespace",
		"whole namespace",
		"other deployments",
		"compare with",
		"compare to",
		"sibling resources",
	) || containsAny(
		query,
		"全部工作负载",
		"所有工作负载",
		"全部 deployment",
		"所有 deployment",
		"整个命名空间",
		"命名空间内全部",
		"其他 deployment",
		"其他资源",
		"对比",
		"比较",
	)
}

func aiNeedsPodOwnershipContext(kind, query string) bool {
	if !strings.EqualFold(strings.TrimSpace(kind), "pod") {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(query))
	return containsAny(text,
		"owner",
		"controller",
		"workload",
		"deployment",
		"statefulset",
		"daemonset",
		"replicaset",
		"managed by",
		"belongs to",
	) || containsAny(
		query,
		"归属",
		"属于",
		"谁创建",
		"谁管理",
		"控制器",
		"工作负载",
		"deployment",
		"statefulset",
		"daemonset",
		"replicaset",
	)
}

func aiNeedsResourceEvents(query string) bool {
	text := strings.ToLower(strings.TrimSpace(query))
	return containsAny(text,
		"event",
		"events",
		"warning",
		"warnings",
		"reason",
	) || containsAny(
		query,
		"事件",
		"告警",
		"警告",
		"原因",
	)
}

func aiNeedsResourceLogs(kind, query string) bool {
	text := strings.ToLower(strings.TrimSpace(query))
	if !containsAny(text, "log", "logs", "crash", "error", "exception") &&
		!containsAny(query, "日志", "报错", "错误", "异常", "崩溃") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "pod", "deployment", "statefulset", "daemonset", "replicaset", "job", "cronjob":
		return true
	default:
		return false
	}
}

func aiNeedsRelatedResources(query string) bool {
	text := strings.ToLower(strings.TrimSpace(query))
	return containsAny(text,
		"related",
		"dependency",
		"dependencies",
		"owner",
		"controller",
		"managed by",
		"references",
	) || containsAny(
		query,
		"关联",
		"依赖",
		"归属",
		"属于",
		"控制器",
		"引用",
	)
}

// aiNeedsConfigSearch 判断用户消息是否涉及配置、数据库或密钥等需要发现 ConfigMap/Secret 的意图
func aiNeedsConfigSearch(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		return false
	}
	return containsAny(text, "mysql", "database", "redis", "mongodb", "postgres", "config") ||
		containsAny(message, "数据库", "配置", "连接", "密码", "密钥", "数据源", "环境变量")
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

func aiClusterInputSchema() model.JSONMap {
	return model.JSONMap{
		"cluster_id": "number",
	}
}

func aiNamespaceInputSchema() model.JSONMap {
	return model.JSONMap{
		"cluster_id": "number",
		"namespace":  "string",
	}
}

func aiScopedResourceInputSchema(kind string) model.JSONMap {
	return model.JSONMap{
		"cluster_id":    "number",
		"resource_kind": kind,
		"namespace":     "string",
		"name":          "string",
	}
}

func aiClusterScopedNameInputSchema(kind string) model.JSONMap {
	return model.JSONMap{
		"cluster_id":    "number",
		"resource_kind": kind,
		"name":          "string",
	}
}

func aiGenericResourceInputSchema() model.JSONMap {
	return model.JSONMap{
		"cluster_id": "number",
		"kind":       "string",
		"namespace":  "string?",
		"name":       "string",
	}
}

func aiToolOutputSchema(source string) model.JSONMap {
	return model.JSONMap{
		"summary":  "string",
		"evidence": "object",
		"raw_ref": model.JSONMap{
			"source": source,
		},
	}
}

func cloneJSONMap(input model.JSONMap) model.JSONMap {
	if input == nil {
		return nil
	}
	out := make(model.JSONMap, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case model.JSONMap:
			out[key] = cloneJSONMap(typed)
		case map[string]any:
			out[key] = cloneJSONMap(model.JSONMap(typed))
		case []any:
			cloned := make([]any, 0, len(typed))
			for _, item := range typed {
				cloned = append(cloned, item)
			}
			out[key] = cloned
		default:
			out[key] = value
		}
	}
	return out
}

func aiStringSliceValue(raw any) []string {
	switch typed := raw.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
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
