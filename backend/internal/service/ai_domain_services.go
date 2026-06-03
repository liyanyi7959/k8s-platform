package service

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/internal/model"
)

type ClusterReadModelService struct {
	k8sSvc *K8sService
}

func NewClusterReadModelService(k8sSvc *K8sService) *ClusterReadModelService {
	return &ClusterReadModelService{k8sSvc: k8sSvc}
}

func (s *ClusterReadModelService) GetClusterHealth(ctx context.Context, clusterID uint64) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	apiOK, nodeReady, nodeTotal, version, err := s.k8sSvc.CheckHealth(ctx, clusterID)
	if err != nil {
		return AIToolResult{}, err
	}
	evidence := model.JSONMap{
		"api_ok":      apiOK,
		"node_ready":  nodeReady,
		"node_total":  nodeTotal,
		"k8s_version": version,
	}
	rawRef := model.JSONMap{
		"cluster_id": clusterID,
		"source":     "cluster.health",
	}
	summary := fmt.Sprintf("API %t, Node Ready %d/%d, Version %s", apiOK, nodeReady, nodeTotal, version)
	return AIToolResult{Summary: summary, Evidence: evidence, RawRef: rawRef}, nil
}

func (s *ClusterReadModelService) GetClusterInventory(ctx context.Context, clusterID uint64) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	namespaces, err := s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}, "", "metadata.name", "asc", nil)
	if err != nil {
		return AIToolResult{}, err
	}
	pods, err := s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, "", "metadata.namespace", "asc", nil)
	if err != nil {
		return AIToolResult{}, err
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

	namespaceItems := make([]model.JSONMap, 0, len(namespaces))
	for _, nsObj := range namespaces {
		ns := aiObjectMetaString(nsObj, "name")
		if ns == "" {
			continue
		}
		namespaceItems = append(namespaceItems, model.JSONMap{
			"name":      ns,
			"pod_count": podCounts[ns],
		})
	}

	evidence := model.JSONMap{
		"namespace_count": len(namespaceItems),
		"pod_count":       len(pods),
		"namespaces":      namespaceItems,
	}
	rawRef := model.JSONMap{
		"cluster_id": clusterID,
		"source":     "cluster.inventory",
	}
	summary := fmt.Sprintf("Cluster has %d namespaces and %d pods", len(namespaceItems), len(pods))
	return AIToolResult{Summary: summary, Evidence: evidence, RawRef: rawRef}, nil
}

type NamespaceDiagnosisService struct {
	k8sSvc *K8sService
}

func NewNamespaceDiagnosisService(k8sSvc *K8sService) *NamespaceDiagnosisService {
	return &NamespaceDiagnosisService{k8sSvc: k8sSvc}
}

func (s *NamespaceDiagnosisService) GetNamespaceHealth(ctx context.Context, clusterID uint64, namespace string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	result, err := s.k8sSvc.GetNamespaceHealth(ctx, clusterID, ns)
	if err != nil {
		return AIToolResult{}, err
	}
	counts, _ := result["pod_counts"].(map[string]int)
	abnormalCount := counts["abnormal"]
	warningEventCount, _ := result["warning_event_count"].(int)
	summary := fmt.Sprintf("Namespace %s detected %d abnormal pods and %d warning events", ns, abnormalCount, warningEventCount)
	return AIToolResult{
		Summary: summary,
		Evidence: model.JSONMap{
			"namespace":           ns,
			"pod_counts":          counts,
			"total_restarts":      result["total_restarts"],
			"abnormal_pods":       result["abnormal_pods"],
			"warning_event_count": warningEventCount,
			"warning_events":      result["warning_events"],
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"source":     "namespace.health",
		},
	}, nil
}

func (s *NamespaceDiagnosisService) GetNamespaceResourceSummary(ctx context.Context, clusterID uint64, namespace string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	items, total, err := s.k8sSvc.GetNamespaceResourcesSummary(ctx, clusterID, ns)
	if err != nil {
		return AIToolResult{}, err
	}
	summary := fmt.Sprintf("Namespace %s has %d resource kinds", ns, total)
	evidenceItems := make([]model.JSONMap, 0, len(items))
	for _, item := range items {
		evidenceItems = append(evidenceItems, model.JSONMap{
			"key":   item.Key,
			"label": item.Label,
			"count": item.Count,
		})
	}
	return AIToolResult{
		Summary: summary,
		Evidence: model.JSONMap{
			"namespace": ns,
			"total":     total,
			"items":     evidenceItems,
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"source":     "namespace.summary",
		},
	}, nil
}

type ResourceInspectionService struct {
	k8sSvc *K8sService
}

func NewResourceInspectionService(k8sSvc *K8sService) *ResourceInspectionService {
	return &ResourceInspectionService{k8sSvc: k8sSvc}
}

func (s *ResourceInspectionService) InspectPod(ctx context.Context, clusterID uint64, namespace, name string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	podName := strings.TrimSpace(name)
	if ns == "" || podName == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	obj, err := s.k8sSvc.GetObject(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, ns, podName)
	if err != nil {
		return AIToolResult{}, err
	}
	logText, logErr := s.k8sSvc.PodLogs(ctx, clusterID, ns, podName, "", 120, false)
	evidence := model.JSONMap{
		"object": obj,
	}
	summary := fmt.Sprintf("Read pod %s/%s object information", ns, podName)
	if logErr != nil {
		evidence["logs_error"] = logErr.Error()
		summary = fmt.Sprintf("Read pod %s/%s object information, logs unavailable", ns, podName)
	} else {
		evidence["logs"] = truncateForModel(logText, 6000)
		summary = fmt.Sprintf("Read pod %s/%s object information and recent logs", ns, podName)
	}
	return AIToolResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"name":       podName,
			"source":     "pod.inspect",
		},
	}, nil
}

func (s *ResourceInspectionService) InspectNode(ctx context.Context, clusterID uint64, name string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	nodeName := strings.TrimSpace(name)
	if nodeName == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	obj, err := s.k8sSvc.GetObject(ctx, clusterID, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, "", nodeName)
	if err != nil {
		return AIToolResult{}, err
	}
	events, evErr := s.k8sSvc.ListNodeEvents(ctx, clusterID, nodeName)
	evidence := model.JSONMap{
		"object": obj,
	}
	summary := fmt.Sprintf("Read node %s object information", nodeName)
	if evErr == nil {
		evidence["events"] = events
		summary = fmt.Sprintf("Read node %s object information and related events", nodeName)
	} else {
		evidence["events_error"] = evErr.Error()
	}
	return AIToolResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"name":       nodeName,
			"source":     "node.inspect",
		},
	}, nil
}

func (s *ResourceInspectionService) InspectDeployment(ctx context.Context, clusterID uint64, namespace, name string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	deploymentName := strings.TrimSpace(name)
	if ns == "" || deploymentName == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	obj, err := s.k8sSvc.GetObject(ctx, clusterID, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, ns, deploymentName)
	if err != nil {
		return AIToolResult{}, err
	}
	history, historyErr := s.k8sSvc.RolloutHistory(ctx, clusterID, ns, deploymentName, "Deployment")
	evidence := model.JSONMap{
		"object": obj,
	}
	summary := fmt.Sprintf("Read deployment %s/%s object information", ns, deploymentName)
	if historyErr == nil {
		evidence["rollout_history"] = history
		summary = fmt.Sprintf("Read deployment %s/%s object information and rollout history", ns, deploymentName)
	} else {
		evidence["rollout_history_error"] = historyErr.Error()
	}
	return AIToolResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"name":       deploymentName,
			"source":     "deployment.inspect",
		},
	}, nil
}

func (s *ResourceInspectionService) ExportResourceYAML(ctx context.Context, clusterID uint64, kind, namespace, name string, policy *ResourceExportPolicyService) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	resKind := strings.TrimSpace(kind)
	resName := strings.TrimSpace(name)
	if resKind == "" || resName == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	gvr, ok := aiGenericNamespacedGVR(resKind)
	if !ok {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "unsupported resource kind for AI export")
	}
	yamlText, err := s.k8sSvc.GetYAML(ctx, clusterID, gvr, ns, resName)
	if err != nil {
		return AIToolResult{}, err
	}
	if policy == nil {
		policy = NewResourceExportPolicyService()
	}
	exportedYAML, masked := policy.MaskYAML(resKind, yamlText)
	evidence := model.JSONMap{
		"kind":      resKind,
		"namespace": ns,
		"name":      resName,
		"yaml":      truncateForModel(exportedYAML, 6000),
		"masked":    masked,
	}
	if masked {
		evidence["redaction_policy"] = "sensitive_fields_masked"
	}
	summary := fmt.Sprintf("Exported %s %s/%s YAML", resKind, ns, resName)
	if masked {
		summary = fmt.Sprintf("Exported %s %s/%s YAML with masking", resKind, ns, resName)
	}
	return AIToolResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"name":       resName,
			"kind":       resKind,
			"source":     "resource.yaml",
		},
	}, nil
}
