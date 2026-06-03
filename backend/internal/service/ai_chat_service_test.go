package service

import "testing"

func TestBuildAIAutoDiagnosticsPlan(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		req     AIChatRequest
		message string
		enabled bool
	}{
		{
			name:    "diagnose mode always collects diagnostics",
			mode:    "diagnose",
			req:     AIChatRequest{Namespace: "blueking"},
			message: "帮我看看这个命名空间是否异常",
			enabled: true,
		},
		{
			name: "chat mode skips generic knowledge question",
			mode: "chat",
			req: AIChatRequest{
				Namespace: "blueking",
			},
			message: "做一个 k8s 集群的巡检会从哪些方面进行呢？",
			enabled: false,
		},
		{
			name: "chat mode collects diagnostics for scoped resource question",
			mode: "chat",
			req: AIChatRequest{
				Namespace:    "blueking",
				ResourceKind: "Deployment",
				ResourceName: "api-server",
			},
			message: "帮我看一下这个 deployment 现在是否健康",
			enabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildAIAutoDiagnosticsPlan(tt.mode, tt.req, tt.message)
			if plan.Enabled != tt.enabled {
				t.Fatalf("buildAIAutoDiagnosticsPlan().Enabled = %v, want %v", plan.Enabled, tt.enabled)
			}
		})
	}
}
