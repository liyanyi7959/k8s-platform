package service

import (
	"testing"

	"k8s-platform-backend/internal/model"
)

func TestBuildAIClusterOverviewResult(t *testing.T) {
	result := buildAIClusterOverviewResult(7, map[string]any{
		"cluster": map[string]any{
			"id":          7,
			"name":        "demo",
			"status":      "ready",
			"api_ok":      true,
			"k8s_version": "v1.30.1",
		},
		"stats": map[string]any{
			"nodes": map[string]any{"total": 3, "ready": 2},
			"pods": map[string]any{
				"total":     12,
				"running":   9,
				"pending":   2,
				"failed":    1,
				"succeeded": 0,
			},
			"workloads": map[string]any{
				"deployments":  4,
				"statefulsets": 1,
				"daemonsets":   2,
			},
			"cpu":    map[string]any{"used_percent": 36},
			"memory": map[string]any{"used_percent": 58},
		},
		"charts": map[string]any{
			"pod_phase": map[string]any{
				"running": 9,
				"pending": 2,
				"failed":  1,
			},
			"namespace_pods_top": []map[string]any{
				{"namespace": "prod", "pods": 8},
			},
			"node_ready": map[string]any{"ready": 2, "total": 3},
		},
		"anomalies": map[string]any{
			"failed_pods": []map[string]any{
				{"namespace": "prod", "name": "api-0", "reason": "CrashLoopBackOff"},
			},
		},
	})

	if result.RawRef["source"] != "cluster.overview" {
		t.Fatalf("unexpected source: %#v", result.RawRef["source"])
	}
	if result.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if result.Evidence["stats"] == nil {
		t.Fatal("expected stats in evidence")
	}
	if result.Evidence["failed_pods"] == nil {
		t.Fatal("expected failed pod evidence")
	}
}

func TestBuildAIClusterCertificateRiskResult(t *testing.T) {
	result := buildAIClusterCertificateRiskResult(8, []map[string]any{
		{"component": "API Server", "status": "critical"},
		{"component": "etcd", "status": "warn"},
		{"component": "scheduler", "status": "ok"},
		{"component": "controller-manager", "status": "mystery"},
	})

	byStatus, _ := result.Evidence["by_status"].(model.JSONMap)
	if byStatus == nil {
		t.Fatal("expected by_status counts")
	}
	if got := aiIntValue(byStatus["critical"]); got != 1 {
		t.Fatalf("critical count = %d, want 1", got)
	}
	if got := aiIntValue(byStatus["warn"]); got != 1 {
		t.Fatalf("warn count = %d, want 1", got)
	}
	if got := aiIntValue(byStatus["ok"]); got != 1 {
		t.Fatalf("ok count = %d, want 1", got)
	}
	if got := aiIntValue(byStatus["unknown"]); got != 1 {
		t.Fatalf("unknown count = %d, want 1", got)
	}
}
