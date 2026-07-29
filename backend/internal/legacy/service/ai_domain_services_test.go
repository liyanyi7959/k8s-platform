package service

import (
	"strings"
	"testing"

	model "k8s-platform-backend/internal/ai/domain"
)

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
