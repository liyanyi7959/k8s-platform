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
	db     *gorm.DB
	k8sSvc *K8sService
}

func NewAIToolService(db *gorm.DB, k8sSvc *K8sService) *AIToolService {
	return &AIToolService{db: db, k8sSvc: k8sSvc}
}

func (s *AIToolService) RunAutoDiagnostics(ctx context.Context, req AIToolContextRequest) ([]AIToolCallItem, string, error) {
	if s.db == nil {
		return nil, "", errors.New("db is required")
	}
	if s.k8sSvc == nil {
		return nil, "", errors.New("k8s service is required")
	}
	if req.ClusterID == 0 || req.ConversationID == 0 {
		return nil, "", ErrWithMessage(ErrInvalidParams, "AI 诊断上下文无效")
	}

	items := make([]AIToolCallItem, 0, 4)
	contextBlocks := make([]string, 0, 4)

	run := func(
		toolName string,
		params map[string]any,
		action func(context.Context) (string, any, error),
	) {
		item, block := s.executeTool(ctx, req, toolName, params, action)
		if item.ID > 0 {
			items = append(items, item)
		}
		if strings.TrimSpace(block) != "" {
			contextBlocks = append(contextBlocks, block)
		}
	}

	run("cluster.health", map[string]any{
		"cluster_id": req.ClusterID,
	}, func(ctx context.Context) (string, any, error) {
		apiOK, nodeReady, nodeTotal, version, err := s.k8sSvc.CheckHealth(ctx, req.ClusterID)
		if err != nil {
			return "", nil, err
		}
		result := map[string]any{
			"api_ok":      apiOK,
			"node_ready":  nodeReady,
			"node_total":  nodeTotal,
			"k8s_version": version,
		}
		summary := fmt.Sprintf("API %t, Node Ready %d/%d, Version %s", apiOK, nodeReady, nodeTotal, version)
		return summary, result, nil
	})

	run("cluster.inventory", map[string]any{
		"cluster_id": req.ClusterID,
	}, func(ctx context.Context) (string, any, error) {
		namespaces, err := s.k8sSvc.List(ctx, req.ClusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}, "", "metadata.name", "asc", nil)
		if err != nil {
			return "", nil, err
		}
		pods, err := s.k8sSvc.List(ctx, req.ClusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, "", "metadata.namespace", "asc", nil)
		if err != nil {
			return "", nil, err
		}

		podCounts := make(map[string]int, len(namespaces))
		for _, nsObj := range namespaces {
			ns := aiObjectMetaString(nsObj, "name")
			if ns != "" {
				podCounts[ns] = 0
			}
		}
		for _, pod := range pods {
			ns := aiObjectMetaString(pod, "namespace")
			if ns == "" {
				ns = "default"
			}
			podCounts[ns]++
		}

		namespaceItems := make([]map[string]any, 0, len(namespaces))
		for _, nsObj := range namespaces {
			ns := aiObjectMetaString(nsObj, "name")
			if ns == "" {
				continue
			}
			namespaceItems = append(namespaceItems, map[string]any{
				"name":      ns,
				"pod_count": podCounts[ns],
			})
		}

		result := map[string]any{
			"namespace_count": len(namespaceItems),
			"pod_count":       len(pods),
			"namespaces":      namespaceItems,
		}
		summary := fmt.Sprintf("集群共有 %d 个命名空间、%d 个 Pod", len(namespaceItems), len(pods))
		return summary, result, nil
	})

	namespace := strings.TrimSpace(req.Namespace)
	if namespace != "" {
		run("namespace.health", map[string]any{
			"cluster_id": req.ClusterID,
			"namespace":  namespace,
		}, func(ctx context.Context) (string, any, error) {
			result, err := s.k8sSvc.GetNamespaceHealth(ctx, req.ClusterID, namespace)
			if err != nil {
				return "", nil, err
			}
			counts, _ := result["pod_counts"].(map[string]int)
			abnormalCount := counts["abnormal"]
			warningEventCount, _ := result["warning_event_count"].(int)
			summary := fmt.Sprintf("Namespace %s 检测到 %d 个异常 Pod、%d 条 Warning 事件", namespace, abnormalCount, warningEventCount)
			return summary, result, nil
		})

		run("namespace.summary", map[string]any{
			"cluster_id": req.ClusterID,
			"namespace":  namespace,
		}, func(ctx context.Context) (string, any, error) {
			items, total, err := s.k8sSvc.GetNamespaceResourcesSummary(ctx, req.ClusterID, namespace)
			if err != nil {
				return "", nil, err
			}
			result := map[string]any{
				"namespace": namespace,
				"total":     total,
				"items":     items,
			}
			summary := fmt.Sprintf("Namespace %s 共发现 %d 个资源对象", namespace, total)
			return summary, result, nil
		})
	}

	kind := strings.TrimSpace(req.ResourceKind)
	name := strings.TrimSpace(req.ResourceName)
	if kind != "" && name != "" {
		s.runResourceDiagnostics(ctx, req, kind, name, &items, &contextBlocks)
	}

	return items, strings.Join(contextBlocks, "\n\n"), nil
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

func (s *AIToolService) ListConversationToolCalls(ctx context.Context, conversationID uint64) ([]AIToolCallItem, error) {
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	if conversationID == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "会话 ID 无效")
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

func (s *AIToolService) runResourceDiagnostics(
	ctx context.Context,
	req AIToolContextRequest,
	kind string,
	name string,
	items *[]AIToolCallItem,
	contextBlocks *[]string,
) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "pod":
		item, block := s.executeTool(ctx, req, "pod.inspect", map[string]any{
			"namespace": req.Namespace,
			"name":      name,
		}, func(ctx context.Context) (string, any, error) {
			obj, err := s.k8sSvc.GetObject(ctx, req.ClusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, req.Namespace, name)
			if err != nil {
				return "", nil, err
			}
			logText, logErr := s.k8sSvc.PodLogs(ctx, req.ClusterID, req.Namespace, name, "", 120, false)
			result := map[string]any{
				"object": obj,
				"logs":   truncateForModel(logText, 6000),
			}
			summary := fmt.Sprintf("已读取 Pod %s/%s 的对象信息和最近日志", req.Namespace, name)
			if logErr != nil {
				result["logs_error"] = logErr.Error()
				summary = fmt.Sprintf("已读取 Pod %s/%s 对象信息，日志读取失败", req.Namespace, name)
			}
			return summary, result, nil
		})
		if item.ID > 0 {
			*items = append(*items, item)
		}
		if strings.TrimSpace(block) != "" {
			*contextBlocks = append(*contextBlocks, block)
		}
	case "node":
		item, block := s.executeTool(ctx, req, "node.inspect", map[string]any{
			"name": name,
		}, func(ctx context.Context) (string, any, error) {
			obj, err := s.k8sSvc.GetObject(ctx, req.ClusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, "", name)
			if err != nil {
				return "", nil, err
			}
			events, evErr := s.k8sSvc.ListNodeEvents(ctx, req.ClusterID, name)
			result := map[string]any{
				"object": obj,
			}
			summary := fmt.Sprintf("已读取 Node %s 的对象信息", name)
			if evErr == nil {
				result["events"] = events
				summary = fmt.Sprintf("已读取 Node %s 的对象信息和关联事件", name)
			} else {
				result["events_error"] = evErr.Error()
			}
			return summary, result, nil
		})
		if item.ID > 0 {
			*items = append(*items, item)
		}
		if strings.TrimSpace(block) != "" {
			*contextBlocks = append(*contextBlocks, block)
		}
	case "deployment":
		item, block := s.executeTool(ctx, req, "deployment.inspect", map[string]any{
			"namespace": req.Namespace,
			"name":      name,
		}, func(ctx context.Context) (string, any, error) {
			obj, err := s.k8sSvc.GetObject(ctx, req.ClusterID, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, req.Namespace, name)
			if err != nil {
				return "", nil, err
			}
			history, historyErr := s.k8sSvc.RolloutHistory(ctx, req.ClusterID, req.Namespace, name, "Deployment")
			result := map[string]any{
				"object": obj,
			}
			summary := fmt.Sprintf("已读取 Deployment %s/%s 的对象信息", req.Namespace, name)
			if historyErr == nil {
				result["rollout_history"] = history
				summary = fmt.Sprintf("已读取 Deployment %s/%s 的对象信息和发布历史", req.Namespace, name)
			} else {
				result["rollout_history_error"] = historyErr.Error()
			}
			return summary, result, nil
		})
		if item.ID > 0 {
			*items = append(*items, item)
		}
		if strings.TrimSpace(block) != "" {
			*contextBlocks = append(*contextBlocks, block)
		}
	default:
		if req.Namespace == "" {
			return
		}
		item, block := s.executeTool(ctx, req, "resource.yaml", map[string]any{
			"kind":      kind,
			"namespace": req.Namespace,
			"name":      name,
		}, func(ctx context.Context) (string, any, error) {
			gvr, ok := aiGenericNamespacedGVR(kind)
			if !ok {
				return "", nil, ErrWithMessage(ErrInvalidParams, "当前资源类型暂不支持自动取证")
			}
			yamlText, err := s.k8sSvc.GetYAML(ctx, req.ClusterID, gvr, req.Namespace, name)
			if err != nil {
				return "", nil, err
			}
			result := map[string]any{
				"yaml": truncateForModel(yamlText, 6000),
			}
			summary := fmt.Sprintf("已读取 %s %s/%s 的 YAML", kind, req.Namespace, name)
			return summary, result, nil
		})
		if item.ID > 0 {
			*items = append(*items, item)
		}
		if strings.TrimSpace(block) != "" {
			*contextBlocks = append(*contextBlocks, block)
		}
	}
}

func (s *AIToolService) executeTool(
	ctx context.Context,
	req AIToolContextRequest,
	toolName string,
	params map[string]any,
	action func(context.Context) (string, any, error),
) (AIToolCallItem, string) {
	row := model.AIToolCall{
		ConversationID: req.ConversationID,
		MessageID:      &req.MessageID,
		ClusterID:      req.ClusterID,
		ToolName:       toolName,
		ToolKind:       "query",
		ExecutionMode:  "auto",
		Status:         "executing",
		RiskLevel:      "low",
		ConfirmLevel:   "single",
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

	summary, result, err := action(ctx)
	update := map[string]any{}
	contextBlock := ""
	if err != nil {
		update["status"] = "failed"
		update["error_message"] = err.Error()
		update["result_summary"] = err.Error()
		_ = s.db.WithContext(ctx).Model(&model.AIToolCall{}).Where("id = ?", row.ID).Updates(update).Error
		row.Status = "failed"
		row.ErrorMessage = err.Error()
		row.ResultSummary = err.Error()
		return buildAIToolCallItem(row), ""
	}

	resultMap := model.JSONMap{
		"output": result,
	}
	update["status"] = "succeeded"
	update["result_summary"] = summary
	update["result_json"] = resultMap
	_ = s.db.WithContext(ctx).Model(&model.AIToolCall{}).Where("id = ?", row.ID).Updates(update).Error

	row.Status = "succeeded"
	row.ResultSummary = summary
	row.ResultJSON = resultMap
	contextBlock = "工具 " + toolName + ": " + summary + "\n" + compactToolResult(result)
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

func aiGenericNamespacedGVR(kind string) (schema.GroupVersionResource, bool) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "service":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}, true
	case "ingress":
		return schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}, true
	case "configmap":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, true
	case "secret":
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}, true
	case "statefulset":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, true
	case "daemonset":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, true
	default:
		return schema.GroupVersionResource{}, false
	}
}
