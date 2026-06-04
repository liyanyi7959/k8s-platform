package service

import (
	"strings"
	"testing"

	"k8s-platform-backend/internal/model"
)

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
			message: "做一个 k8s 集群巡检会从哪些方面进行？",
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
		{
			name: "chat mode collects diagnostics for namespace inspection request",
			mode: "chat",
			req: AIChatRequest{
				Namespace: "blueking",
			},
			message: "针对当前 blueking 命名空间做一次完整巡检，包含资源个数和 pod 资源使用情况",
			enabled: true,
		},
		{
			name:    "chat mode collects diagnostics for cluster overview request",
			mode:    "chat",
			req:     AIChatRequest{},
			message: "帮我看下当前集群的整体状态和资源使用情况",
			enabled: true,
		},
		{
			name:    "chat mode collects diagnostics for control plane request",
			mode:    "chat",
			req:     AIChatRequest{},
			message: "检查一下当前集群控制面和证书风险",
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

func TestBuildAIMessageScopeSnapshot(t *testing.T) {
	providerID := uint64(7)
	modelID := uint64(11)
	snapshot := buildAIMessageScopeSnapshot(
		model.AIConversation{
			ID:            23,
			AssistantMode: "chat",
		},
		AIChatRequest{
			ClusterID:    5,
			Namespace:    "payments",
			ResourceKind: "Deployment",
			ResourceName: "api",
			PreferModel:  "gpt-4.1",
		},
		&providerID,
		&modelID,
	)

	requestScope, ok := snapshot["request_scope"].(model.JSONMap)
	if !ok {
		t.Fatalf("expected request_scope JSON map, got %#v", snapshot["request_scope"])
	}
	if got := requestScope["cluster_id"]; got != uint64(5) {
		t.Fatalf("request_scope.cluster_id = %#v, want %d", got, 5)
	}
	if got := requestScope["conversation_id"]; got != uint64(23) {
		t.Fatalf("request_scope.conversation_id = %#v, want %d", got, 23)
	}
	if got := requestScope["assistant_mode"]; got != "chat" {
		t.Fatalf("request_scope.assistant_mode = %#v, want %q", got, "chat")
	}
	if got := requestScope["namespace"]; got != "payments" {
		t.Fatalf("request_scope.namespace = %#v, want %q", got, "payments")
	}
	if got := requestScope["resource_kind"]; got != "Deployment" {
		t.Fatalf("request_scope.resource_kind = %#v, want %q", got, "Deployment")
	}
	if got := requestScope["resource_name"]; got != "api" {
		t.Fatalf("request_scope.resource_name = %#v, want %q", got, "api")
	}
	if got := requestScope["prefer_model"]; got != "gpt-4.1" {
		t.Fatalf("request_scope.prefer_model = %#v, want %q", got, "gpt-4.1")
	}
	if got := requestScope["provider_id"]; got != providerID {
		t.Fatalf("request_scope.provider_id = %#v, want %d", got, providerID)
	}
	if got := requestScope["model_id"]; got != modelID {
		t.Fatalf("request_scope.model_id = %#v, want %d", got, modelID)
	}
}

func TestBuildAIGatewayScopeNote(t *testing.T) {
	note := buildAIGatewayScopeNote("devops", "Deployment", "bkci-auth")
	if !strings.Contains(note, "namespace=devops") {
		t.Fatalf("expected namespace scope note, got %q", note)
	}
	if !strings.Contains(note, "kind=Deployment") || !strings.Contains(note, "name=bkci-auth") {
		t.Fatalf("expected resource scope note, got %q", note)
	}
	if !strings.Contains(note, "Do not list sibling resources") {
		t.Fatalf("expected strict scope guard, got %q", note)
	}
}
