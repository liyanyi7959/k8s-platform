package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/internal/model"
)

type ClusterReadModelService struct {
	dashboardSvc *DashboardService
}

func NewClusterReadModelService(dashboardSvc *DashboardService) *ClusterReadModelService {
	return &ClusterReadModelService{dashboardSvc: dashboardSvc}
}

func (s *ClusterReadModelService) GetClusterHealth(ctx context.Context, clusterID uint64) (AIToolResult, error) {
	if s == nil || s.dashboardSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	overview, err := s.dashboardSvc.GetClusterOverview(ctx, clusterID)
	if err != nil {
		return AIToolResult{}, err
	}
	return buildAIClusterHealthResult(clusterID, overview), nil
}

func (s *ClusterReadModelService) GetClusterInventory(ctx context.Context, clusterID uint64) (AIToolResult, error) {
	return s.GetClusterOverview(ctx, clusterID)
}

func (s *ClusterReadModelService) GetClusterOverview(ctx context.Context, clusterID uint64) (AIToolResult, error) {
	if s == nil || s.dashboardSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	overview, err := s.dashboardSvc.GetClusterOverview(ctx, clusterID)
	if err != nil {
		return AIToolResult{}, err
	}
	return buildAIClusterOverviewResult(clusterID, overview), nil
}

func (s *ClusterReadModelService) GetClusterCertificateRisks(ctx context.Context, clusterID uint64) (AIToolResult, error) {
	if s == nil || s.dashboardSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	risks, err := s.dashboardSvc.GetClusterCertificateRisks(ctx, clusterID)
	if err != nil {
		return AIToolResult{}, err
	}
	return buildAIClusterCertificateRiskResult(clusterID, risks), nil
}

func buildAIClusterHealthResult(clusterID uint64, overview map[string]any) AIToolResult {
	cluster := aiMapValue(overview, "cluster")
	stats := aiMapValue(overview, "stats")
	nodes := aiMapValue(stats, "nodes")
	pods := aiMapValue(stats, "pods")
	workloads := aiMapValue(stats, "workloads")
	cpu := aiMapValue(stats, "cpu")
	memory := aiMapValue(stats, "memory")

	version := strings.TrimSpace(fmt.Sprint(cluster["k8s_version"]))
	if version == "" {
		version = "unknown"
	}
	summary := fmt.Sprintf(
		"API %t, nodes ready %d/%d, version %s, running pods %d/%d",
		aiBoolValue(cluster["api_ok"]),
		aiIntValue(nodes["ready"]),
		aiIntValue(nodes["total"]),
		version,
		aiIntValue(pods["running"]),
		aiIntValue(pods["total"]),
	)

	return AIToolResult{
		Summary: summary,
		Evidence: model.JSONMap{
			"cluster":   model.JSONMap(cluster),
			"nodes":     model.JSONMap(nodes),
			"pods":      model.JSONMap(pods),
			"workloads": model.JSONMap(workloads),
			"cpu":       model.JSONMap(cpu),
			"memory":    model.JSONMap(memory),
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"source":     "cluster.health",
		},
	}
}

func buildAIClusterOverviewResult(clusterID uint64, overview map[string]any) AIToolResult {
	cluster := aiMapValue(overview, "cluster")
	stats := aiMapValue(overview, "stats")
	charts := aiMapValue(overview, "charts")
	anomalies := aiMapValue(overview, "anomalies")
	pods := aiMapValue(stats, "pods")
	workloads := aiMapValue(stats, "workloads")
	cpu := aiMapValue(stats, "cpu")
	memory := aiMapValue(stats, "memory")

	summary := fmt.Sprintf(
		"Cluster overview: nodes ready %d/%d, pods %d total (%d running, %d pending, %d failed), workloads %d deployments/%d statefulsets/%d daemonsets, CPU %d%%, memory %d%%",
		aiIntValue(aiMapValue(stats, "nodes")["ready"]),
		aiIntValue(aiMapValue(stats, "nodes")["total"]),
		aiIntValue(pods["total"]),
		aiIntValue(pods["running"]),
		aiIntValue(pods["pending"]),
		aiIntValue(pods["failed"]),
		aiIntValue(workloads["deployments"]),
		aiIntValue(workloads["statefulsets"]),
		aiIntValue(workloads["daemonsets"]),
		aiIntValue(cpu["used_percent"]),
		aiIntValue(memory["used_percent"]),
	)

	return AIToolResult{
		Summary: summary,
		Evidence: model.JSONMap{
			"cluster":              model.JSONMap(cluster),
			"stats":                model.JSONMap(stats),
			"pod_phase":            charts["pod_phase"],
			"namespace_pods_top":   charts["namespace_pods_top"],
			"node_ready":           charts["node_ready"],
			"failed_pods":          anomalies["failed_pods"],
			"certificate_endpoint": "cluster.certificate_risks",
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"source":     "cluster.overview",
		},
	}
}

func buildAIClusterCertificateRiskResult(clusterID uint64, risks []map[string]any) AIToolResult {
	counts := model.JSONMap{
		"critical": 0,
		"warn":     0,
		"ok":       0,
		"unknown":  0,
	}
	for _, item := range risks {
		status := strings.TrimSpace(fmt.Sprint(item["status"]))
		if _, ok := counts[status]; !ok {
			status = "unknown"
		}
		counts[status] = aiIntValue(counts[status]) + 1
	}

	summary := fmt.Sprintf(
		"Cluster certificate risks: %d critical, %d warning, %d ok, %d unknown",
		aiIntValue(counts["critical"]),
		aiIntValue(counts["warn"]),
		aiIntValue(counts["ok"]),
		aiIntValue(counts["unknown"]),
	)

	return AIToolResult{
		Summary: summary,
		Evidence: model.JSONMap{
			"total":     len(risks),
			"by_status": counts,
			"items":     risks,
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"source":     "cluster.certificate_risks",
		},
	}
}

func aiMapValue(source map[string]any, key string) map[string]any {
	if source == nil {
		return nil
	}
	value, _ := source[key].(map[string]any)
	return value
}

func aiBoolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		return err == nil && parsed
	default:
		return false
	}
}

func aiIntValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return parsed
		}
	}
	return 0
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

func (s *NamespaceDiagnosisService) GetNamespaceInspection(ctx context.Context, clusterID uint64, namespace string) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return AIToolResult{}, ErrInvalidParams
	}

	healthResult, err := s.GetNamespaceHealth(ctx, clusterID, ns)
	if err != nil {
		return AIToolResult{}, err
	}
	summaryResult, err := s.GetNamespaceResourceSummary(ctx, clusterID, ns)
	if err != nil {
		return AIToolResult{}, err
	}
	metricsEvidence, metricsSummary, metricsErr := s.namespacePodMetricsEvidence(ctx, clusterID, ns)
	if metricsErr != nil {
		return AIToolResult{}, metricsErr
	}
	workloadResult, err := s.GetNamespaceWorkloadInventory(ctx, clusterID, ns)
	if err != nil {
		return AIToolResult{}, err
	}

	podCounts, _ := healthResult.Evidence["pod_counts"].(map[string]int)
	abnormalCount := podCounts["abnormal"]
	warningEventCount, _ := healthResult.Evidence["warning_event_count"].(int)
	totalResources, _ := summaryResult.Evidence["total"].(int)

	summaryParts := []string{
		fmt.Sprintf("Namespace %s inspection", ns),
		fmt.Sprintf("%d abnormal pods", abnormalCount),
		fmt.Sprintf("%d warning events", warningEventCount),
		fmt.Sprintf("%d resources", totalResources),
	}
	if strings.TrimSpace(metricsSummary) != "" {
		summaryParts = append(summaryParts, metricsSummary)
	}
	if strings.TrimSpace(workloadResult.Summary) != "" {
		summaryParts = append(summaryParts, workloadResult.Summary)
	}

	return AIToolResult{
		Summary: strings.Join(summaryParts, ", "),
		Evidence: model.JSONMap{
			"namespace":          ns,
			"health":             healthResult.Evidence,
			"resource_summary":   summaryResult.Evidence,
			"pod_metrics":        metricsEvidence,
			"workload_inventory": workloadResult.Evidence,
		},
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"source":     "namespace.inspect",
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

	podEvidence := buildAIPodOverview(obj)
	relationshipEvidence := s.inspectPodRelationships(ctx, clusterID, ns, obj)
	containerEvidence, aggregateResources := buildAIPodContainerResources(obj)
	conditionEvidence := collectAIResourceConditions(obj)
	metricsEvidence, metricsSummary := s.inspectSinglePodMetrics(ctx, clusterID, ns, podName)
	logsEvidence, logsSummary := s.inspectPodLogEvidence(ctx, clusterID, ns, podName)

	summaryParts := []string{
		fmt.Sprintf("Pod %s/%s", ns, podName),
	}
	if phase := strings.TrimSpace(fmt.Sprint(podEvidence["phase"])); phase != "" {
		summaryParts = append(summaryParts, "phase "+phase)
	}
	if controller := aiRelationshipControllerSummary(relationshipEvidence); controller != "" {
		summaryParts = append(summaryParts, "owner "+controller)
	}
	if ready := strings.TrimSpace(fmt.Sprint(podEvidence["ready"])); ready != "" {
		summaryParts = append(summaryParts, "ready "+ready)
	}
	if strings.TrimSpace(metricsSummary) != "" {
		summaryParts = append(summaryParts, metricsSummary)
	}
	if strings.TrimSpace(logsSummary) != "" {
		summaryParts = append(summaryParts, logsSummary)
	}

	evidence := model.JSONMap{
		"pod":           podEvidence,
		"conditions":    conditionEvidence,
		"containers":    containerEvidence,
		"resources":     aggregateResources,
		"relationships": relationshipEvidence,
		"metrics":       metricsEvidence,
	}
	for key, value := range logsEvidence {
		evidence[key] = value
	}

	return AIToolResult{
		Summary:  strings.Join(summaryParts, ", "),
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
	workloadSummary := buildAIWorkloadSummaryFromObject("Deployment", obj)
	conditions := collectAIResourceConditions(obj)
	history, historyErr := s.k8sSvc.RolloutHistory(ctx, clusterID, ns, deploymentName, "Deployment")
	evidence := model.JSONMap{
		"deployment": workloadSummary,
		"conditions": conditions,
		"object":     obj,
	}
	summary := buildAIDeploymentInspectSummary(ns, deploymentName, workloadSummary, conditions)
	if historyErr == nil {
		evidence["rollout_history"] = history
		if len(history) > 0 {
			evidence["rollout_revision_count"] = len(history)
			summary = summary + fmt.Sprintf(", rollout revisions %d", len(history))
		} else {
			summary = summary + ", rollout history included"
		}
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

func buildAIDeploymentInspectSummary(namespace, name string, workload model.JSONMap, conditions []model.JSONMap) string {
	parts := []string{
		fmt.Sprintf("Deployment %s/%s", strings.TrimSpace(namespace), strings.TrimSpace(name)),
	}

	desired := aiIntValue(workload["desired_replicas"])
	ready := aiIntValue(workload["ready_replicas"])
	available := aiIntValue(workload["available_replicas"])
	updated := aiIntValue(workload["updated_replicas"])
	parts = append(parts, fmt.Sprintf("replicas ready %d/%d", ready, desired))
	parts = append(parts, fmt.Sprintf("available %d", available))
	parts = append(parts, fmt.Sprintf("updated %d", updated))

	if containers, ok := workload["containers"].([]model.JSONMap); ok && len(containers) > 0 {
		imageParts := make([]string, 0, len(containers))
		for _, container := range containers {
			containerName := strings.TrimSpace(fmt.Sprint(container["name"]))
			image := strings.TrimSpace(fmt.Sprint(container["image"]))
			if containerName == "" && image == "" {
				continue
			}
			if containerName == "" {
				imageParts = append(imageParts, image)
				continue
			}
			imageParts = append(imageParts, containerName+"="+image)
		}
		if len(imageParts) > 0 {
			parts = append(parts, "images "+strings.Join(imageParts, ", "))
		}
	}

	if len(conditions) > 0 {
		condition := conditions[0]
		condType := strings.TrimSpace(fmt.Sprint(condition["type"]))
		condStatus := strings.TrimSpace(fmt.Sprint(condition["status"]))
		if condType != "" {
			parts = append(parts, fmt.Sprintf("condition %s=%s", condType, condStatus))
		}
	}

	return strings.Join(parts, ", ")
}

func (s *ResourceInspectionService) InspectResource(
	ctx context.Context,
	clusterID uint64,
	kind,
	namespace,
	name string,
	policy *ResourceExportPolicyService,
) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	resKind := strings.TrimSpace(kind)
	resName := strings.TrimSpace(name)
	ns := strings.TrimSpace(namespace)
	if resKind == "" || resName == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	gvr, namespaced, ok := aiSupportedResourceGVR(resKind)
	if !ok {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "unsupported resource kind for AI inspection")
	}
	if namespaced && ns == "" {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "namespace is required for the selected resource kind")
	}
	if !namespaced {
		ns = ""
	}
	if policy == nil {
		policy = NewResourceExportPolicyService()
	}

	obj, err := s.k8sSvc.GetObject(ctx, clusterID, gvr, ns, resName)
	if err != nil {
		return AIToolResult{}, err
	}
	safeObj, maskedObject := policy.SanitizeObject(resKind, obj)

	evidence := model.JSONMap{
		"overview":   buildAIResourceOverview(resKind, ns, resName, safeObj),
		"conditions": collectAIResourceConditions(safeObj),
		"object":     safeObj,
	}
	yamlText, yamlErr := s.k8sSvc.GetYAML(ctx, clusterID, gvr, ns, resName)
	if yamlErr == nil {
		exportedYAML, maskedYAML := policy.MaskYAML(resKind, yamlText)
		evidence["yaml"] = truncateForModel(exportedYAML, 6000)
		if maskedObject || maskedYAML {
			evidence["masked"] = true
		}
	} else {
		evidence["yaml_error"] = firstUserFacingError(yamlErr)
	}

	return AIToolResult{
		Summary:  buildAIResourceInspectSummary(resKind, ns, resName, safeObj, maskedObject),
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"name":       resName,
			"kind":       resKind,
			"source":     "resource.inspect",
		},
	}, nil
}

func (s *ResourceInspectionService) ExportResourceYAML(ctx context.Context, clusterID uint64, kind, namespace, name string, policy *ResourceExportPolicyService) (AIToolResult, error) {
	return s.exportResourceYAML(ctx, clusterID, kind, namespace, name, policy, false)
}

func (s *ResourceInspectionService) ExportMaskedResourceYAML(ctx context.Context, clusterID uint64, kind, namespace, name string, policy *ResourceExportPolicyService) (AIToolResult, error) {
	return s.exportResourceYAML(ctx, clusterID, kind, namespace, name, policy, true)
}

func (s *ResourceInspectionService) exportResourceYAML(
	ctx context.Context,
	clusterID uint64,
	kind,
	namespace,
	name string,
	policy *ResourceExportPolicyService,
	forceMasked bool,
) (AIToolResult, error) {
	if s == nil || s.k8sSvc == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	resKind := strings.TrimSpace(kind)
	resName := strings.TrimSpace(name)
	if resKind == "" || resName == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	gvr, namespaced, ok := aiSupportedResourceGVR(resKind)
	if !ok {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "unsupported resource kind for AI export")
	}
	if namespaced && ns == "" {
		return AIToolResult{}, ErrWithMessage(ErrInvalidParams, "namespace is required for the selected resource kind")
	}
	if !namespaced {
		ns = ""
	}
	yamlText, err := s.k8sSvc.GetYAML(ctx, clusterID, gvr, ns, resName)
	if err != nil {
		return AIToolResult{}, err
	}
	if policy == nil {
		policy = NewResourceExportPolicyService()
	}
	var (
		exportedYAML string
		masked       bool
	)
	if forceMasked {
		exportedYAML, masked, err = policy.ExportMaskedYAML(resKind, yamlText)
	} else {
		exportedYAML, masked, err = policy.ExportYAML(resKind, yamlText)
	}
	if err != nil {
		return AIToolResult{}, err
	}
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
	if masked || forceMasked {
		summary = fmt.Sprintf("Exported %s %s/%s YAML with masking", resKind, ns, resName)
	}
	source := "resource.yaml"
	if forceMasked {
		source = "resource.masked_yaml"
	}
	return AIToolResult{
		Summary:  summary,
		Evidence: evidence,
		RawRef: model.JSONMap{
			"cluster_id": clusterID,
			"namespace":  ns,
			"name":       resName,
			"kind":       resKind,
			"source":     source,
		},
	}, nil
}

func (s *NamespaceDiagnosisService) namespacePodMetricsEvidence(ctx context.Context, clusterID uint64, namespace string) (model.JSONMap, string, error) {
	metricsGVR := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
	supported, err := s.k8sSvc.SupportsCompatibleGVR(ctx, clusterID, metricsGVR)
	if err != nil {
		return nil, "", err
	}
	if !supported {
		return model.JSONMap{
			"supported": false,
			"message":   "metrics API is not available in the current cluster",
		}, "metrics unavailable", nil
	}

	items, err := s.k8sSvc.List(ctx, clusterID, metricsGVR, namespace, "metadata.name", "asc", nil)
	if err != nil {
		return nil, "", err
	}

	type podMetricSummary struct {
		Name      string
		Namespace string
		CPUText   string
		MemText   string
		CPUMilli  int64
		MemBytes  int64
	}

	summaries := make([]podMetricSummary, 0, len(items))
	var totalCPU int64
	var totalMemory int64
	var latestTimestamp string
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok || obj == nil {
			continue
		}
		cpuMilli, memBytes := aiMetricUsageTotals(obj)
		summaries = append(summaries, podMetricSummary{
			Name:      aiObjectMetaString(obj, "name"),
			Namespace: aiObjectMetaString(obj, "namespace"),
			CPUText:   formatAIMillicores(cpuMilli),
			MemText:   formatAIMemoryBytes(memBytes),
			CPUMilli:  cpuMilli,
			MemBytes:  memBytes,
		})
		totalCPU += cpuMilli
		totalMemory += memBytes
		if ts := strings.TrimSpace(fmt.Sprint(obj["timestamp"])); ts != "" && ts > latestTimestamp {
			latestTimestamp = ts
		}
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].CPUMilli == summaries[j].CPUMilli {
			if summaries[i].MemBytes == summaries[j].MemBytes {
				return summaries[i].Name < summaries[j].Name
			}
			return summaries[i].MemBytes > summaries[j].MemBytes
		}
		return summaries[i].CPUMilli > summaries[j].CPUMilli
	})

	topPods := make([]model.JSONMap, 0, minInt(len(summaries), 10))
	for i := 0; i < len(summaries) && i < 10; i++ {
		item := summaries[i]
		topPods = append(topPods, model.JSONMap{
			"name":           item.Name,
			"namespace":      item.Namespace,
			"cpu":            item.CPUText,
			"memory":         item.MemText,
			"cpu_millicores": item.CPUMilli,
			"memory_bytes":   item.MemBytes,
		})
	}

	summary := fmt.Sprintf(
		"metrics cover %d pods, CPU %s, memory %s",
		len(summaries),
		formatAIMillicores(totalCPU),
		formatAIMemoryBytes(totalMemory),
	)
	return model.JSONMap{
		"supported":            true,
		"pod_count":            len(summaries),
		"total_cpu_millicores": totalCPU,
		"total_cpu":            formatAIMillicores(totalCPU),
		"total_memory_bytes":   totalMemory,
		"total_memory":         formatAIMemoryBytes(totalMemory),
		"collected_at":         latestTimestamp,
		"top_pods":             topPods,
	}, summary, nil
}

func aiMetricUsageTotals(obj map[string]any) (int64, int64) {
	containers, _ := obj["containers"].([]any)
	var totalCPU int64
	var totalMemory int64
	for _, item := range containers {
		container, _ := item.(map[string]any)
		if container == nil {
			continue
		}
		usage, _ := container["usage"].(map[string]any)
		if usage == nil {
			continue
		}
		totalCPU += parseAIMetricQuantity(usage["cpu"], true)
		totalMemory += parseAIMetricQuantity(usage["memory"], false)
	}
	return totalCPU, totalMemory
}

func parseAIMetricQuantity(raw any, cpu bool) int64 {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" {
		return 0
	}
	q, err := resource.ParseQuantity(text)
	if err != nil {
		return 0
	}
	if cpu {
		return q.MilliValue()
	}
	return q.Value()
}

func formatAIMillicores(value int64) string {
	if value >= 1000 || value <= -1000 {
		return fmt.Sprintf("%.2f cores", float64(value)/1000)
	}
	return fmt.Sprintf("%dm", value)
}

func formatAIMemoryBytes(value int64) string {
	const (
		ki = 1024
		mi = 1024 * ki
		gi = 1024 * mi
		ti = 1024 * gi
	)
	switch {
	case value >= ti || value <= -ti:
		return fmt.Sprintf("%.2f Ti", float64(value)/float64(ti))
	case value >= gi || value <= -gi:
		return fmt.Sprintf("%.2f Gi", float64(value)/float64(gi))
	case value >= mi || value <= -mi:
		return fmt.Sprintf("%.1f Mi", float64(value)/float64(mi))
	case value >= ki || value <= -ki:
		return fmt.Sprintf("%.1f Ki", float64(value)/float64(ki))
	default:
		return fmt.Sprintf("%d B", value)
	}
}

func buildAIResourceOverview(kind, namespace, name string, obj map[string]any) model.JSONMap {
	overview := model.JSONMap{
		"kind":      strings.TrimSpace(kind),
		"namespace": strings.TrimSpace(namespace),
		"name":      strings.TrimSpace(name),
	}
	if obj == nil {
		return overview
	}
	overview["api_version"] = strings.TrimSpace(fmt.Sprint(obj["apiVersion"]))
	if metadata, ok := obj["metadata"].(map[string]any); ok && metadata != nil {
		overview["generation"] = metadata["generation"]
		overview["creation_timestamp"] = metadata["creationTimestamp"]
		if labels, ok := metadata["labels"].(map[string]any); ok {
			overview["label_count"] = len(labels)
		}
		if annotations, ok := metadata["annotations"].(map[string]any); ok {
			overview["annotation_count"] = len(annotations)
		}
	}
	if status, ok := obj["status"].(map[string]any); ok && status != nil {
		if phase := strings.TrimSpace(fmt.Sprint(status["phase"])); phase != "" {
			overview["phase"] = phase
		}
		if replicas := strings.TrimSpace(fmt.Sprint(status["replicas"])); replicas != "" && replicas != "<nil>" {
			overview["replicas"] = replicas
		}
		if readyReplicas := strings.TrimSpace(fmt.Sprint(status["readyReplicas"])); readyReplicas != "" && readyReplicas != "<nil>" {
			overview["ready_replicas"] = readyReplicas
		}
	}
	return overview
}

func collectAIResourceConditions(obj map[string]any) []model.JSONMap {
	if obj == nil {
		return nil
	}
	status, _ := obj["status"].(map[string]any)
	if status == nil {
		return nil
	}
	rawConditions, _ := status["conditions"].([]any)
	if len(rawConditions) == 0 {
		return nil
	}
	conditions := make([]model.JSONMap, 0, minInt(len(rawConditions), 10))
	for _, item := range rawConditions {
		condition, _ := item.(map[string]any)
		if condition == nil {
			continue
		}
		conditions = append(conditions, model.JSONMap{
			"type":               strings.TrimSpace(fmt.Sprint(condition["type"])),
			"status":             strings.TrimSpace(fmt.Sprint(condition["status"])),
			"reason":             strings.TrimSpace(fmt.Sprint(condition["reason"])),
			"message":            strings.TrimSpace(fmt.Sprint(condition["message"])),
			"last_transition_at": strings.TrimSpace(fmt.Sprint(condition["lastTransitionTime"])),
		})
		if len(conditions) >= 10 {
			break
		}
	}
	return conditions
}

func buildAIResourceInspectSummary(kind, namespace, name string, obj map[string]any, masked bool) string {
	scope := strings.TrimSpace(name)
	if ns := strings.TrimSpace(namespace); ns != "" {
		scope = ns + "/" + scope
	}
	parts := []string{fmt.Sprintf("Read %s %s", strings.TrimSpace(kind), scope)}
	status, _ := obj["status"].(map[string]any)
	if phase := strings.TrimSpace(fmt.Sprint(status["phase"])); phase != "" {
		parts = append(parts, "phase "+phase)
	}
	if readyReplicas := strings.TrimSpace(fmt.Sprint(status["readyReplicas"])); readyReplicas != "" && readyReplicas != "<nil>" {
		replicas := strings.TrimSpace(fmt.Sprint(status["replicas"]))
		if replicas == "" || replicas == "<nil>" {
			parts = append(parts, "ready replicas "+readyReplicas)
		} else {
			parts = append(parts, fmt.Sprintf("ready replicas %s/%s", readyReplicas, replicas))
		}
	}
	conditions := collectAIResourceConditions(obj)
	if len(conditions) > 0 {
		first := conditions[0]
		if condType := strings.TrimSpace(fmt.Sprint(first["type"])); condType != "" {
			parts = append(parts, fmt.Sprintf("condition %s=%s", condType, strings.TrimSpace(fmt.Sprint(first["status"]))))
		}
	}
	if masked {
		parts = append(parts, "sensitive fields masked")
	}
	return strings.Join(parts, ", ")
}
