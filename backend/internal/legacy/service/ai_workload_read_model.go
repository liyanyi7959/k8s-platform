package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	model "k8s-platform-backend/internal/ai/domain"
)

func (s *NamespaceDiagnosisService) GetNamespaceWorkloadInventory(ctx context.Context, clusterID uint64, namespace string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	evidence, summary, err := s.namespaceWorkloadInventoryEvidence(ctx, clusterID, ns)
	if err != nil {
		return AIToolResult{}, err
	}
	return AIToolResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"source":     "namespace.workloads",
		},
	}, nil
}

func (s *NamespaceDiagnosisService) namespaceWorkloadInventoryEvidence(ctx context.Context, clusterID uint64, namespace string) (model.JSONMap, string, error) {
	workloadSpecs := []struct {
		kind string
		gvr  schema.GroupVersionResource
		key  string
	}{
		{kind: "Deployment", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, key: "deployments"},
		{kind: "StatefulSet", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, key: "statefulsets"},
		{kind: "DaemonSet", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, key: "daemonsets"},
	}

	itemsByKind := model.JSONMap{}
	flattened := make([]model.JSONMap, 0, 32)
	counts := model.JSONMap{}
	totalDesired := 0
	totalReady := 0

	for _, spec := range workloadSpecs {
		list, err := s.k8sSvc.List(ctx, clusterID, spec.gvr, namespace, "metadata.name", "asc", nil)
		if err != nil {
			return nil, "", err
		}

		summaries := make([]model.JSONMap, 0, len(list))
		for _, item := range list {
			obj, _ := item.(map[string]any)
			if obj == nil {
				continue
			}
			summary := BuildAIWorkloadSummary(spec.kind, obj)
			summaries = append(summaries, summary)
			flattened = append(flattened, summary)
			totalDesired += aiIntValue(summary["desired_replicas"])
			totalReady += aiIntValue(summary["ready_replicas"])
		}

		itemsByKind[spec.key] = summaries
		counts[spec.key] = len(summaries)
	}

	sort.SliceStable(flattened, func(i, j int) bool {
		leftKind := strings.TrimSpace(fmt.Sprint(flattened[i]["kind"]))
		rightKind := strings.TrimSpace(fmt.Sprint(flattened[j]["kind"]))
		if leftKind == rightKind {
			return strings.TrimSpace(fmt.Sprint(flattened[i]["name"])) < strings.TrimSpace(fmt.Sprint(flattened[j]["name"]))
		}
		return leftKind < rightKind
	})

	summary := fmt.Sprintf(
		"Namespace %s workload inventory: %d workloads (%d Deployments, %d StatefulSets, %d DaemonSets), ready replicas %d/%d",
		namespace,
		len(flattened),
		aiIntValue(counts["deployments"]),
		aiIntValue(counts["statefulsets"]),
		aiIntValue(counts["daemonsets"]),
		totalReady,
		totalDesired,
	)

	return model.JSONMap{
		"namespace": namespace,
		"counts":    counts,
		"items":     flattened,
		"by_kind":   itemsByKind,
		"replicas": model.JSONMap{
			"desired": totalDesired,
			"ready":   totalReady,
		},
	}, summary, nil
}

func BuildAIWorkloadSummary(kind string, obj map[string]any) model.JSONMap {
	summary := model.JSONMap{
		"kind":      strings.TrimSpace(kind),
		"name":      AIObjectMetaString(obj, "name"),
		"namespace": AIObjectMetaString(obj, "namespace"),
	}
	if obj == nil {
		return summary
	}

	status, _ := obj["status"].(map[string]any)
	spec, _ := obj["spec"].(map[string]any)
	summary["desired_replicas"] = aiWorkloadDesiredReplicas(kind, spec, status)
	summary["ready_replicas"] = aiWorkloadReadyReplicas(kind, status)

	if value := aiIntValue(status["availableReplicas"]); value > 0 || strings.EqualFold(kind, "Deployment") {
		summary["available_replicas"] = value
	}
	if value := aiIntValue(status["updatedReplicas"]); value > 0 || strings.EqualFold(kind, "Deployment") || strings.EqualFold(kind, "StatefulSet") {
		summary["updated_replicas"] = value
	}
	if value := aiIntValue(status["currentReplicas"]); value > 0 || strings.EqualFold(kind, "StatefulSet") {
		summary["current_replicas"] = value
	}
	if value := aiIntValue(status["currentNumberScheduled"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["current_replicas"] = value
	}
	if value := aiIntValue(status["numberAvailable"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["available_replicas"] = value
	}
	if value := aiIntValue(status["updatedNumberScheduled"]); value > 0 || strings.EqualFold(kind, "DaemonSet") {
		summary["updated_replicas"] = value
	}

	if selector, ok := spec["selector"].(map[string]any); ok && selector != nil {
		if matchLabels, ok := selector["matchLabels"].(map[string]any); ok && len(matchLabels) > 0 {
			summary["selector"] = matchLabels
		}
	}

	containers := aiTemplateContainersFromObject(obj)
	if len(containers) > 0 {
		summary["containers"] = containers
	}

	return summary
}

func aiWorkloadDesiredReplicas(kind string, spec, status map[string]any) int {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "daemonset":
		if status != nil {
			if value := aiIntValue(status["desiredNumberScheduled"]); value > 0 {
				return value
			}
			return aiIntValue(status["currentNumberScheduled"])
		}
		return 0
	default:
		if spec != nil {
			if value := aiIntValue(spec["replicas"]); value > 0 {
				return value
			}
		}
		if status != nil {
			return aiIntValue(status["replicas"])
		}
		return 0
	}
}

func aiWorkloadReadyReplicas(kind string, status map[string]any) int {
	if status == nil {
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "daemonset":
		return aiIntValue(status["numberReady"])
	default:
		return aiIntValue(status["readyReplicas"])
	}
}

func aiTemplateContainersFromObject(obj map[string]any) []model.JSONMap {
	spec, _ := obj["spec"].(map[string]any)
	if spec == nil {
		return nil
	}
	if template, ok := spec["template"].(map[string]any); ok && template != nil {
		if templateSpec, ok := template["spec"].(map[string]any); ok && templateSpec != nil {
			return aiContainerNameImageList(templateSpec)
		}
	}
	return aiContainerNameImageList(spec)
}

func aiContainerNameImageList(spec map[string]any) []model.JSONMap {
	if spec == nil {
		return nil
	}
	containers := make([]model.JSONMap, 0, 8)
	appendList := func(items []any, containerType string) {
		for _, item := range items {
			container, _ := item.(map[string]any)
			if container == nil {
				continue
			}
			containers = append(containers, model.JSONMap{
				"name":  strings.TrimSpace(fmt.Sprint(container["name"])),
				"image": strings.TrimSpace(fmt.Sprint(container["image"])),
				"type":  containerType,
			})
		}
	}
	if list, _ := spec["containers"].([]any); len(list) > 0 {
		appendList(list, "container")
	}
	if list, _ := spec["initContainers"].([]any); len(list) > 0 {
		appendList(list, "initContainer")
	}
	return containers
}
