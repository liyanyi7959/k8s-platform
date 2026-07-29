package service

import (
	"testing"

	model "k8s-platform-backend/internal/ai/domain"
)

func TestBuildAIWorkloadSummaryFromObject(t *testing.T) {
	summary := BuildAIWorkloadSummary("Deployment", map[string]any{
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
