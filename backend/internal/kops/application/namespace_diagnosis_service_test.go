package application

import (
	"context"
	"errors"
	"testing"
)

type namespaceDiagnosisRuntimeSpy struct {
	supported bool
	metrics   []map[string]any
}

func (*namespaceDiagnosisRuntimeSpy) NamespaceHealth(context.Context, uint64, string) (map[string]any, error) {
	return map[string]any{
		"pod_counts":          map[string]int{"running": 3, "abnormal": 1},
		"total_restarts":      4,
		"abnormal_pods":       []any{map[string]any{"name": "api"}},
		"warning_event_count": 2,
		"warning_events":      []any{map[string]any{"reason": "BackOff"}},
	}, nil
}

func (spy *namespaceDiagnosisRuntimeSpy) SupportsPodMetrics(context.Context, uint64) (bool, error) {
	return spy.supported, nil
}

func (spy *namespaceDiagnosisRuntimeSpy) ListPodMetrics(context.Context, uint64, string) ([]map[string]any, error) {
	return spy.metrics, nil
}

type namespaceSummaryReaderSpy struct{}

func (namespaceSummaryReaderSpy) Summary(context.Context, uint64, string) (NamespaceResourceSummary, error) {
	return NamespaceResourceSummary{Items: []NamespaceResourceSummaryItem{{Key: "pods", Label: "Pod", Count: 3}}, Total: 3}, nil
}

type namespaceWorkloadReaderSpy struct{}

func (namespaceWorkloadReaderSpy) NamespaceWorkloadInventory(_ context.Context, clusterID uint64, namespace string) (NamespaceWorkloadResult, error) {
	return NamespaceWorkloadResult{Summary: "Namespace " + namespace + " workload inventory", Evidence: map[string]any{"namespace": namespace}, RawRef: map[string]any{"cluster_id": clusterID}}, nil
}

func TestNamespaceDiagnosisInspectionCombinesKopsReadModels(t *testing.T) {
	service := NewNamespaceDiagnosisService(&namespaceDiagnosisRuntimeSpy{supported: true, metrics: []map[string]any{{
		"metadata":   map[string]any{"name": "api", "namespace": "ops"},
		"timestamp":  "2026-07-29T12:00:00Z",
		"containers": []any{map[string]any{"usage": map[string]any{"cpu": "125m", "memory": "64Mi"}}},
	}}}, namespaceSummaryReaderSpy{}, namespaceWorkloadReaderSpy{})

	result, err := service.NamespaceInspection(context.Background(), 7, " ops ")
	if err != nil {
		t.Fatalf("NamespaceInspection() error = %v", err)
	}
	if result.RawRef["namespace"] != "ops" || result.Evidence["namespace"] != "ops" {
		t.Fatalf("namespace normalization failed: %#v", result)
	}
	metrics := result.Evidence["pod_metrics"].(map[string]any)
	if metrics["pod_count"] != 1 || metrics["total_cpu_millicores"] != int64(125) {
		t.Fatalf("metrics = %#v", metrics)
	}
	health := result.Evidence["health"].(map[string]any)
	if health["warning_event_count"] != 2 {
		t.Fatalf("health = %#v", health)
	}
	if result.Evidence["workload_inventory"].(map[string]any)["namespace"] != "ops" {
		t.Fatalf("workload evidence = %#v", result.Evidence["workload_inventory"])
	}
}

func TestNamespaceDiagnosisValidatesScope(t *testing.T) {
	service := NewNamespaceDiagnosisService(&namespaceDiagnosisRuntimeSpy{}, namespaceSummaryReaderSpy{}, namespaceWorkloadReaderSpy{})
	if _, err := service.NamespaceHealth(context.Background(), 0, "ops"); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid cluster error = %v", err)
	}
	if _, err := service.NamespaceInspection(context.Background(), 1, " "); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid namespace error = %v", err)
	}
}
