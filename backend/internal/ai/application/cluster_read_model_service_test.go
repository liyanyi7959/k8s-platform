package application

import (
	"context"
	"testing"

	"k8s-platform-backend/internal/ai/domain"
)

type clusterReadPortSpy struct {
	overview map[string]any
	risks    []map[string]any
}

func (s clusterReadPortSpy) ClusterOverview(context.Context, uint64) (map[string]any, error) {
	return s.overview, nil
}
func (s clusterReadPortSpy) ClusterCertificateRisks(context.Context, uint64) ([]map[string]any, error) {
	return s.risks, nil
}

func TestClusterReadModelServiceBuildsOverview(t *testing.T) {
	service := NewClusterReadModelService(clusterReadPortSpy{overview: map[string]any{
		"cluster": map[string]any{"api_ok": true, "k8s_version": "v1.30"},
		"stats":   map[string]any{"nodes": map[string]any{"ready": 2, "total": 3}, "pods": map[string]any{"total": 8, "running": 6, "pending": 1, "failed": 1}, "workloads": map[string]any{"deployments": 2, "statefulsets": 1, "daemonsets": 1}, "cpu": map[string]any{"used_percent": 44}, "memory": map[string]any{"used_percent": 55}},
		"charts":  map[string]any{}, "anomalies": map[string]any{},
	}})
	result, err := service.GetClusterOverview(context.Background(), 7)
	if err != nil || result.RawRef["cluster_id"] != uint64(7) {
		t.Fatalf("overview result = %#v, err = %v", result, err)
	}
	if result.Evidence["certificate_endpoint"] != "cluster.certificate_risks" {
		t.Fatalf("evidence = %#v", result.Evidence)
	}
}

func TestClusterReadModelServiceSummarizesCertificateRisks(t *testing.T) {
	service := NewClusterReadModelService(clusterReadPortSpy{risks: []map[string]any{{"status": "critical"}, {"status": "warn"}, {"status": "other"}}})
	result, err := service.GetClusterCertificateRisks(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	counts := result.Evidence["by_status"].(domain.JSONMap)
	if counts["critical"] != 1 || counts["warn"] != 1 || counts["unknown"] != 1 {
		t.Fatalf("counts = %#v", counts)
	}
}
