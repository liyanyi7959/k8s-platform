package service

import (
	"strings"
	"testing"

	model "k8s-platform-backend/internal/ai/domain"
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

func TestBuildAIWorkloadSummaryFromObject(t *testing.T) {
	summary := buildAIWorkloadSummaryFromObject("Deployment", map[string]any{
		"metadata": map[string]any{
			"name":      "web",
			"namespace": "devops",
		},
		"spec": map[string]any{
			"replicas": 3,
			"selector": map[string]any{
				"matchLabels": map[string]any{"app": "web"},
			},
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{"name": "app", "image": "nginx:1.27"},
					},
				},
			},
		},
		"status": map[string]any{
			"readyReplicas":     2,
			"availableReplicas": 2,
			"updatedReplicas":   3,
		},
	})

	if got := summary["name"]; got != "web" {
		t.Fatalf("name = %v, want web", got)
	}
	if got := aiIntValue(summary["desired_replicas"]); got != 3 {
		t.Fatalf("desired_replicas = %d, want 3", got)
	}
	if got := aiIntValue(summary["ready_replicas"]); got != 2 {
		t.Fatalf("ready_replicas = %d, want 2", got)
	}
	containers, _ := summary["containers"].([]model.JSONMap)
	if len(containers) != 1 {
		t.Fatalf("containers len = %d, want 1", len(containers))
	}
}

func TestBuildAIPodContainerResources(t *testing.T) {
	containers, totals := buildAIPodContainerResources(map[string]any{
		"spec": map[string]any{
			"containers": []any{
				map[string]any{
					"name":  "app",
					"image": "nginx:1.27",
					"resources": map[string]any{
						"requests": map[string]any{"cpu": "100m", "memory": "128Mi"},
						"limits":   map[string]any{"cpu": "500m", "memory": "512Mi"},
					},
				},
			},
		},
	})

	if len(containers) != 1 {
		t.Fatalf("containers len = %d, want 1", len(containers))
	}
	if got := containers[0]["cpu_limit"]; got != "500m" {
		t.Fatalf("cpu_limit = %v, want 500m", got)
	}
	requests, _ := totals["requests"].(model.JSONMap)
	limits, _ := totals["limits"].(model.JSONMap)
	if got := aiIntValue(requests["cpu_millicores"]); got != 100 {
		t.Fatalf("request cpu = %d, want 100", got)
	}
	if got := aiIntValue(limits["cpu_millicores"]); got != 500 {
		t.Fatalf("limit cpu = %d, want 500", got)
	}
}

func TestBuildAIDeploymentInspectSummary(t *testing.T) {
	summary := buildAIDeploymentInspectSummary("devops", "bkci-auth", model.JSONMap{
		"desired_replicas":   3,
		"ready_replicas":     2,
		"available_replicas": 2,
		"updated_replicas":   3,
		"containers": []model.JSONMap{
			{
				"name":  "app",
				"image": "bkci-auth:v2",
			},
		},
	}, []model.JSONMap{
		{
			"type":   "Available",
			"status": "True",
		},
	})

	if summary == "" {
		t.Fatal("expected non-empty deployment summary")
	}
	if got := summary; !strings.Contains(got, "ready 2/3") {
		t.Fatalf("expected replica summary, got %q", got)
	}
	if got := summary; !strings.Contains(got, "app=bkci-auth:v2") {
		t.Fatalf("expected image summary, got %q", got)
	}
	if got := summary; !strings.Contains(got, "Available=True") {
		t.Fatalf("expected condition summary, got %q", got)
	}
}
