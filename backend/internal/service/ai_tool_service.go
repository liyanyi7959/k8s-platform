package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/internal/model"
)

type AIToolContextRequest struct {
	ConversationID uint64
	MessageID      uint64
	ClusterID      uint64
	UserID         uint64
	Username       string
	UserPerms      []string
	Query          string
	Namespace      string
	ResourceKind   string
	ResourceName   string
}

type AIToolCallItem struct {
	ID            uint64        `json:"id"`
	MessageID     *uint64       `json:"message_id,omitempty"`
	ToolName      string        `json:"tool_name"`
	ToolKind      string        `json:"tool_kind"`
	Status        string        `json:"status"`
	RiskLevel     string        `json:"risk_level"`
	ConfirmLevel  string        `json:"confirm_level"`
	ResultSummary string        `json:"result_summary"`
	Result        model.JSONMap `json:"result,omitempty"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	CreatedAt     string        `json:"created_at"`
}

type AIToolService struct {
	db       *gorm.DB
	registry *AIToolRegistry
}

func NewAIToolService(db *gorm.DB, registry *AIToolRegistry) *AIToolService {
	return &AIToolService{db: db, registry: registry}
}

func (s *AIToolService) RunAutoDiagnostics(ctx context.Context, req AIToolContextRequest) ([]AIToolCallItem, string, error) {
	if s.db == nil {
		return nil, "", errors.New("db is required")
	}
	if s.registry == nil {
		return nil, "", errors.New("tool registry is required")
	}
	if req.ClusterID == 0 || req.ConversationID == 0 {
		return nil, "", ErrWithMessage(ErrInvalidParams, "AI diagnostic context is invalid")
	}

	items := make([]AIToolCallItem, 0, 6)
	contextBlocks := make([]string, 0, 6)

	run := func(toolName string, params map[string]any) {
		item, block := s.executeRegisteredTool(ctx, req, toolName, params)
		if item.ID > 0 {
			items = append(items, item)
		}
		if strings.TrimSpace(block) != "" {
			contextBlocks = append(contextBlocks, block)
		}
	}

	query := strings.TrimSpace(req.Query)
	namespace := strings.TrimSpace(req.Namespace)
	kind := strings.TrimSpace(req.ResourceKind)
	name := strings.TrimSpace(req.ResourceName)
	broadInspection := aiNeedsBroadInspection(query)
	clusterInspection := aiNeedsClusterInspection(query)
	controlPlaneInspection := aiNeedsControlPlaneInspection(query)
	yamlIntent := aiNeedsResourceYAML(query)

	run("cluster.health", map[string]any{
		"cluster_id": req.ClusterID,
	})
	if namespace == "" || broadInspection || clusterInspection {
		run("cluster.overview", map[string]any{
			"cluster_id": req.ClusterID,
		})
	}
	if namespace == "" && (controlPlaneInspection || broadInspection) {
		run("cluster.certificate_risks", map[string]any{
			"cluster_id": req.ClusterID,
		})
	}

	if namespace != "" {
		if broadInspection {
			run("namespace.inspect", map[string]any{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
			})
		} else {
			run("namespace.health", map[string]any{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
			})
			run("namespace.summary", map[string]any{
				"cluster_id": req.ClusterID,
				"namespace":  namespace,
			})
		}
	}

	if kind != "" && name != "" {
		switch strings.ToLower(kind) {
		case "pod":
			run("pod.inspect", map[string]any{
				"namespace": namespace,
				"name":      name,
			})
		case "node":
			run("node.inspect", map[string]any{
				"name": name,
			})
		case "deployment":
			run("deployment.inspect", map[string]any{
				"namespace": namespace,
				"name":      name,
			})
		default:
			run("resource.inspect", map[string]any{
				"kind":      kind,
				"namespace": namespace,
				"name":      name,
			})
			if yamlIntent {
				run("resource.yaml", map[string]any{
					"kind":      kind,
					"namespace": namespace,
					"name":      name,
				})
			}
		}
	}

	return items, strings.Join(contextBlocks, "\n\n"), nil
}

func (s *AIToolService) ListConversationToolCalls(ctx context.Context, conversationID uint64) ([]AIToolCallItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	if conversationID == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "conversation ID is invalid")
	}
	var rows []model.AIToolCall
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]AIToolCallItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, buildAIToolCallItem(row))
	}
	return items, nil
}

func (s *AIToolService) executeRegisteredTool(
	ctx context.Context,
	req AIToolContextRequest,
	toolName string,
	params map[string]any,
) (AIToolCallItem, string) {
	def, ok := s.registry.Get(toolName)
	if !ok {
		return AIToolCallItem{}, ""
	}

	row := model.AIToolCall{
		ConversationID: req.ConversationID,
		MessageID:      &req.MessageID,
		ClusterID:      req.ClusterID,
		ToolName:       toolName,
		ToolKind:       def.Category,
		ExecutionMode:  "auto",
		Status:         "executing",
		RiskLevel:      def.RiskLevel,
		ConfirmLevel:   def.ConfirmLevel,
		CreatedBy:      req.UserID,
		CreatedByName:  strings.TrimSpace(req.Username),
	}
	if payload, err := json.Marshal(params); err == nil {
		row.CommandText = string(payload)
		row.ParamsJSON = model.JSONMap(params)
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return AIToolCallItem{}, ""
	}

	if err := toolPermissionErr(def.RequiredPermissions, req.UserPerms, toolName); err != nil {
		row.Status = "failed"
		row.ErrorMessage = err.Error()
		row.ResultSummary = err.Error()
		_ = s.db.WithContext(ctx).Model(&model.AIToolCall{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":         row.Status,
			"error_message":  row.ErrorMessage,
			"result_summary": row.ResultSummary,
		}).Error
		return buildAIToolCallItem(row), ""
	}

	handlerCtx := ctx
	cancel := func() {}
	if def.Timeout > 0 {
		handlerCtx, cancel = context.WithTimeout(ctx, def.Timeout)
	}
	defer cancel()

	result, err := def.Handler(handlerCtx, req, params)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(handlerCtx.Err(), context.DeadlineExceeded) {
			err = ErrWithMessage(ErrK8sTimeout, fmt.Sprintf("tool %s execution timed out", toolName))
		}
		row.Status = "failed"
		row.ErrorMessage = firstUserFacingError(err)
		row.ResultSummary = firstUserFacingError(err)
		_ = s.db.WithContext(ctx).Model(&model.AIToolCall{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":         row.Status,
			"error_message":  row.ErrorMessage,
			"result_summary": row.ResultSummary,
		}).Error
		return buildAIToolCallItem(row), ""
	}

	resultJSON := toolEvidenceMap(result)
	row.Status = "succeeded"
	row.ResultSummary = result.Summary
	row.ResultJSON = resultJSON
	_ = s.db.WithContext(ctx).Model(&model.AIToolCall{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":         row.Status,
		"result_summary": row.ResultSummary,
		"result_json":    row.ResultJSON,
	}).Error

	row.ResultJSON = resultJSON
	contextBlock := "工具 " + toolName + ": " + result.Summary + "\n" + compactToolResult(resultJSON)
	return buildAIToolCallItem(row), contextBlock
}

func buildAIToolCallItem(row model.AIToolCall) AIToolCallItem {
	return AIToolCallItem{
		ID:            row.ID,
		MessageID:     row.MessageID,
		ToolName:      row.ToolName,
		ToolKind:      row.ToolKind,
		Status:        row.Status,
		RiskLevel:     row.RiskLevel,
		ConfirmLevel:  row.ConfirmLevel,
		ResultSummary: row.ResultSummary,
		Result:        row.ResultJSON,
		ErrorMessage:  row.ErrorMessage,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func compactToolResult(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return truncateForModel(string(b), 3000)
}

func truncateForModel(input string, limit int) string {
	raw := strings.TrimSpace(input)
	if limit <= 0 || len([]rune(raw)) <= limit {
		return raw
	}
	return string([]rune(raw)[:limit]) + "..."
}

func aiObjectMetaString(item any, key string) string {
	obj, ok := item.(map[string]any)
	if !ok {
		return ""
	}
	meta, ok := obj["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(meta[key]))
}

func aiSupportedResourceGVR(kind string) (schema.GroupVersionResource, bool, bool) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "namespace":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}, false, true
	case "node":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, false, true
	case "pod":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, true, true
	case "deployment":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, true, true
	case "statefulset":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, true, true
	case "daemonset":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, true, true
	case "replicaset":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}, true, true
	case "service":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}, true, true
	case "ingress":
		return schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}, true, true
	case "ingressclass":
		return schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"}, false, true
	case "networkpolicy":
		return schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}, true, true
	case "configmap":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, true, true
	case "secret":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}, true, true
	case "serviceaccount":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}, true, true
	case "endpoint":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}, true, true
	case "endpointslice":
		return schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}, true, true
	case "lease":
		return schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}, true, true
	case "pdb", "poddisruptionbudget":
		return schema.GroupVersionResource{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}, true, true
	case "role":
		return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"}, true, true
	case "clusterrole":
		return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}, false, true
	case "rolebinding":
		return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}, true, true
	case "clusterrolebinding":
		return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}, false, true
	case "hpa", "horizontalpodautoscaler":
		return schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}, true, true
	case "event":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}, true, true
	case "pvc", "persistentvolumeclaim":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}, true, true
	case "pv", "persistentvolume":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}, false, true
	case "storageclass":
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}, false, true
	case "csidriver":
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csidrivers"}, false, true
	case "csinode":
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csinodes"}, false, true
	case "csistoragecapacity":
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csistoragecapacities"}, true, true
	case "volumeattachment":
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "volumeattachments"}, false, true
	case "volumesnapshot":
		return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots"}, true, true
	case "volumesnapshotclass":
		return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotclasses"}, false, true
	case "volumesnapshotcontent":
		return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotcontents"}, false, true
	case "resourcequota":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"}, true, true
	case "limitrange":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "limitranges"}, true, true
	case "customresourcedefinition", "crd":
		return schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}, false, true
	case "apiservice":
		return schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}, false, true
	case "priorityclass":
		return schema.GroupVersionResource{Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"}, false, true
	case "runtimeclass":
		return schema.GroupVersionResource{Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses"}, false, true
	case "validatingwebhookconfiguration":
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}, false, true
	case "mutatingwebhookconfiguration":
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}, false, true
	case "validatingadmissionpolicy":
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingadmissionpolicies"}, false, true
	case "validatingadmissionpolicybinding":
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingadmissionpolicybindings"}, false, true
	case "job":
		return schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}, true, true
	case "cronjob":
		return schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, true, true
	default:
		return schema.GroupVersionResource{}, false, false
	}
}

func aiGenericNamespacedGVR(kind string) (schema.GroupVersionResource, bool) {
	gvr, namespaced, ok := aiSupportedResourceGVR(kind)
	if !ok || !namespaced {
		return schema.GroupVersionResource{}, false
	}
	return gvr, true
}
