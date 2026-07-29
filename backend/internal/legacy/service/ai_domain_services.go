package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	model "k8s-platform-backend/internal/ai/domain"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

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
	k8sSvc  *K8sService
	summary kopsapp.NamespaceResourceSummaryReader
}

func NewNamespaceDiagnosisService(k8sSvc *K8sService, summary ...kopsapp.NamespaceResourceSummaryReader) *NamespaceDiagnosisService {
	service := &NamespaceDiagnosisService{k8sSvc: k8sSvc}
	if len(summary) > 0 {
		service.summary = summary[0]
	}
	return service
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
	if s == nil || s.summary == nil {
		return AIToolResult{}, ErrK8s
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return AIToolResult{}, ErrInvalidParams
	}
	resourceSummary, err := s.summary.Summary(ctx, clusterID, ns)
	if err != nil {
		return AIToolResult{}, err
	}
	summary := fmt.Sprintf("Namespace %s has %d resource kinds", ns, resourceSummary.Total)
	evidenceItems := make([]model.JSONMap, 0, len(resourceSummary.Items))
	for _, item := range resourceSummary.Items {
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
			"total":     resourceSummary.Total,
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
			Name:      AIObjectMetaString(obj, "name"),
			Namespace: AIObjectMetaString(obj, "namespace"),
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

func BuildAIResourceOverview(kind, namespace, name string, obj map[string]any) model.JSONMap {
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
