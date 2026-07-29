package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// NamespaceDiagnosisResult is the shared, transport-neutral namespace read
// model used by Kops HTTP inspection and AI tools.
type NamespaceDiagnosisResult struct {
	Summary  string         `json:"summary"`
	Evidence map[string]any `json:"evidence"`
	RawRef   map[string]any `json:"raw_ref"`
}

type NamespaceDiagnosisRuntime interface {
	NamespaceHealth(context.Context, uint64, string) (map[string]any, error)
	SupportsPodMetrics(context.Context, uint64) (bool, error)
	ListPodMetrics(context.Context, uint64, string) ([]map[string]any, error)
}

type NamespaceDiagnosisReader interface {
	NamespaceHealth(context.Context, uint64, string) (NamespaceDiagnosisResult, error)
	NamespaceResourceSummary(context.Context, uint64, string) (NamespaceDiagnosisResult, error)
	NamespaceWorkloadInventory(context.Context, uint64, string) (NamespaceWorkloadResult, error)
	NamespaceInspection(context.Context, uint64, string) (NamespaceDiagnosisResult, error)
}

type NamespaceDiagnosisService struct {
	runtime   NamespaceDiagnosisRuntime
	summary   NamespaceResourceSummaryReader
	workloads NamespaceWorkloadReader
}

func NewNamespaceDiagnosisService(runtime NamespaceDiagnosisRuntime, summary NamespaceResourceSummaryReader, workloads NamespaceWorkloadReader) *NamespaceDiagnosisService {
	return &NamespaceDiagnosisService{runtime: runtime, summary: summary, workloads: workloads}
}

func (s *NamespaceDiagnosisService) NamespaceHealth(ctx context.Context, clusterID uint64, namespace string) (NamespaceDiagnosisResult, error) {
	namespace = strings.TrimSpace(namespace)
	if clusterID == 0 || namespace == "" {
		return NamespaceDiagnosisResult{}, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return NamespaceDiagnosisResult{}, ErrConflict
	}
	value, err := s.runtime.NamespaceHealth(ctx, clusterID, namespace)
	if err != nil {
		return NamespaceDiagnosisResult{}, err
	}
	counts := namespaceDiagnosisIntMap(value["pod_counts"])
	abnormalCount := counts["abnormal"]
	warningEventCount := inspectionInt(value["warning_event_count"])
	return NamespaceDiagnosisResult{
		Summary: fmt.Sprintf("Namespace %s detected %d abnormal pods and %d warning events", namespace, abnormalCount, warningEventCount),
		Evidence: map[string]any{
			"namespace": namespace, "pod_counts": value["pod_counts"], "total_restarts": value["total_restarts"],
			"abnormal_pods": value["abnormal_pods"], "warning_event_count": warningEventCount, "warning_events": value["warning_events"],
		},
		RawRef: map[string]any{"cluster_id": clusterID, "namespace": namespace, "source": "namespace.health"},
	}, nil
}

func (s *NamespaceDiagnosisService) NamespaceResourceSummary(ctx context.Context, clusterID uint64, namespace string) (NamespaceDiagnosisResult, error) {
	namespace = strings.TrimSpace(namespace)
	if clusterID == 0 || namespace == "" {
		return NamespaceDiagnosisResult{}, ErrInvalidParams
	}
	if s == nil || s.summary == nil {
		return NamespaceDiagnosisResult{}, ErrConflict
	}
	resourceSummary, err := s.summary.Summary(ctx, clusterID, namespace)
	if err != nil {
		return NamespaceDiagnosisResult{}, err
	}
	items := make([]map[string]any, 0, len(resourceSummary.Items))
	for _, item := range resourceSummary.Items {
		items = append(items, map[string]any{"key": item.Key, "label": item.Label, "count": item.Count})
	}
	return NamespaceDiagnosisResult{
		Summary:  fmt.Sprintf("Namespace %s has %d resource kinds", namespace, resourceSummary.Total),
		Evidence: map[string]any{"namespace": namespace, "total": resourceSummary.Total, "items": items},
		RawRef:   map[string]any{"cluster_id": clusterID, "namespace": namespace, "source": "namespace.summary"},
	}, nil
}

func (s *NamespaceDiagnosisService) NamespaceWorkloadInventory(ctx context.Context, clusterID uint64, namespace string) (NamespaceWorkloadResult, error) {
	if s == nil || s.workloads == nil {
		return NamespaceWorkloadResult{}, ErrConflict
	}
	return s.workloads.NamespaceWorkloadInventory(ctx, clusterID, namespace)
}

func (s *NamespaceDiagnosisService) NamespaceInspection(ctx context.Context, clusterID uint64, namespace string) (NamespaceDiagnosisResult, error) {
	namespace = strings.TrimSpace(namespace)
	if clusterID == 0 || namespace == "" {
		return NamespaceDiagnosisResult{}, ErrInvalidParams
	}
	if s == nil || s.runtime == nil || s.summary == nil || s.workloads == nil {
		return NamespaceDiagnosisResult{}, ErrConflict
	}
	health, err := s.NamespaceHealth(ctx, clusterID, namespace)
	if err != nil {
		return NamespaceDiagnosisResult{}, err
	}
	resourceSummary, err := s.NamespaceResourceSummary(ctx, clusterID, namespace)
	if err != nil {
		return NamespaceDiagnosisResult{}, err
	}
	metrics, metricsSummary, err := s.namespacePodMetricsEvidence(ctx, clusterID, namespace)
	if err != nil {
		return NamespaceDiagnosisResult{}, err
	}
	workloads, err := s.NamespaceWorkloadInventory(ctx, clusterID, namespace)
	if err != nil {
		return NamespaceDiagnosisResult{}, err
	}

	podCounts := namespaceDiagnosisIntMap(health.Evidence["pod_counts"])
	parts := []string{
		fmt.Sprintf("Namespace %s inspection", namespace),
		fmt.Sprintf("%d abnormal pods", podCounts["abnormal"]),
		fmt.Sprintf("%d warning events", inspectionInt(health.Evidence["warning_event_count"])),
		fmt.Sprintf("%d resources", inspectionInt(resourceSummary.Evidence["total"])),
	}
	if strings.TrimSpace(metricsSummary) != "" {
		parts = append(parts, metricsSummary)
	}
	if strings.TrimSpace(workloads.Summary) != "" {
		parts = append(parts, workloads.Summary)
	}
	return NamespaceDiagnosisResult{
		Summary: strings.Join(parts, ", "),
		Evidence: map[string]any{
			"namespace": namespace, "health": health.Evidence, "resource_summary": resourceSummary.Evidence,
			"pod_metrics": metrics, "workload_inventory": workloads.Evidence,
		},
		RawRef: map[string]any{"cluster_id": clusterID, "namespace": namespace, "source": "namespace.inspect"},
	}, nil
}

func (s *NamespaceDiagnosisService) namespacePodMetricsEvidence(ctx context.Context, clusterID uint64, namespace string) (map[string]any, string, error) {
	supported, err := s.runtime.SupportsPodMetrics(ctx, clusterID)
	if err != nil {
		return nil, "", err
	}
	if !supported {
		return map[string]any{"supported": false, "message": "metrics API is not available in the current cluster"}, "metrics unavailable", nil
	}
	items, err := s.runtime.ListPodMetrics(ctx, clusterID, namespace)
	if err != nil {
		return nil, "", err
	}
	type metricSummary struct {
		name, namespace, cpu, memory string
		cpuMilli, memoryBytes        int64
	}
	summaries := make([]metricSummary, 0, len(items))
	var totalCPU, totalMemory int64
	latestTimestamp := ""
	for _, item := range items {
		cpuMilli, memoryBytes := inspectionMetricTotals(item)
		summaries = append(summaries, metricSummary{
			name: inspectionObjectMetaString(item, "name"), namespace: inspectionObjectMetaString(item, "namespace"),
			cpu: formatInspectionMillicores(cpuMilli), memory: formatInspectionMemory(memoryBytes), cpuMilli: cpuMilli, memoryBytes: memoryBytes,
		})
		totalCPU += cpuMilli
		totalMemory += memoryBytes
		if timestamp := strings.TrimSpace(fmt.Sprint(item["timestamp"])); timestamp != "" && timestamp > latestTimestamp {
			latestTimestamp = timestamp
		}
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].cpuMilli == summaries[j].cpuMilli {
			if summaries[i].memoryBytes == summaries[j].memoryBytes {
				return summaries[i].name < summaries[j].name
			}
			return summaries[i].memoryBytes > summaries[j].memoryBytes
		}
		return summaries[i].cpuMilli > summaries[j].cpuMilli
	})
	topPods := make([]map[string]any, 0, minInspectionInt(len(summaries), 10))
	for index := 0; index < len(summaries) && index < 10; index++ {
		item := summaries[index]
		topPods = append(topPods, map[string]any{"name": item.name, "namespace": item.namespace, "cpu": item.cpu, "memory": item.memory, "cpu_millicores": item.cpuMilli, "memory_bytes": item.memoryBytes})
	}
	return map[string]any{
		"supported": true, "pod_count": len(summaries), "total_cpu_millicores": totalCPU, "total_cpu": formatInspectionMillicores(totalCPU),
		"total_memory_bytes": totalMemory, "total_memory": formatInspectionMemory(totalMemory), "collected_at": latestTimestamp, "top_pods": topPods,
	}, fmt.Sprintf("metrics cover %d pods, CPU %s, memory %s", len(summaries), formatInspectionMillicores(totalCPU), formatInspectionMemory(totalMemory)), nil
}

func namespaceDiagnosisIntMap(value any) map[string]int {
	result := map[string]int{}
	switch values := value.(type) {
	case map[string]int:
		for key, item := range values {
			result[key] = item
		}
	case map[string]any:
		for key, item := range values {
			result[key] = inspectionInt(item)
		}
	}
	return result
}
