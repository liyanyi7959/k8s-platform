package application

import (
	"strings"
	"testing"
)

func TestBuildInspectionPodContainerResources(t *testing.T) {
	containers, totals := buildInspectionPodContainerResources(map[string]any{
		"spec": map[string]any{
			"containers": []any{map[string]any{
				"name":  "app",
				"image": "nginx:1.27",
				"resources": map[string]any{
					"requests": map[string]any{"cpu": "100m", "memory": "128Mi"},
					"limits":   map[string]any{"cpu": "500m", "memory": "512Mi"},
				},
			}},
		},
	})

	if len(containers) != 1 {
		t.Fatalf("containers len = %d, want 1", len(containers))
	}
	if got := containers[0]["cpu_limit"]; got != "500m" {
		t.Fatalf("cpu_limit = %v, want 500m", got)
	}
	requests, _ := totals["requests"].(map[string]any)
	limits, _ := totals["limits"].(map[string]any)
	if got := inspectionInt(requests["cpu_millicores"]); got != 100 {
		t.Fatalf("request cpu = %d, want 100", got)
	}
	if got := inspectionInt(limits["cpu_millicores"]); got != 500 {
		t.Fatalf("limit cpu = %d, want 500", got)
	}
}

func TestBuildInspectionDeploymentSummary(t *testing.T) {
	summary := BuildInspectionDeploymentSummary("devops", "bkci-auth", map[string]any{
		"desired_replicas":   3,
		"ready_replicas":     2,
		"available_replicas": 2,
		"updated_replicas":   3,
		"containers":         []map[string]any{{"name": "app", "image": "bkci-auth:v2"}},
	}, []map[string]any{{"type": "Available", "status": "True"}})

	for _, expected := range []string{"ready 2/3", "app=bkci-auth:v2", "Available=True"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("summary %q does not contain %q", summary, expected)
		}
	}
}
