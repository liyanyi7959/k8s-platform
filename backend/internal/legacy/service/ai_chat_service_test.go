package service

import (
	"strings"
	"testing"

	aigateway "k8s-platform-backend/internal/ai/adapters/gateway"
	aiapp "k8s-platform-backend/internal/ai/application"
	model "k8s-platform-backend/internal/ai/domain"
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
			plan := aiapp.BuildChatDiagnosticsPlan(aiapp.ChatDiagnosticsRequest{
				AssistantMode: tt.mode,
				Namespace:     tt.req.Namespace,
				ResourceKind:  tt.req.ResourceKind,
				ResourceName:  tt.req.ResourceName,
				Message:       tt.message,
			})
			if plan.Enabled != tt.enabled {
				t.Fatalf("BuildChatDiagnosticsPlan().Enabled = %v, want %v", plan.Enabled, tt.enabled)
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
		nil,
		nil,
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

func TestBuildAIMessageScopeSnapshotIncludesImages(t *testing.T) {
	snapshot := buildAIMessageScopeSnapshot(
		model.AIConversation{
			ID:            7,
			AssistantMode: "diagnose",
		},
		AIChatRequest{
			ClusterID:    3,
			Namespace:    "devops",
			ResourceKind: "Pod",
			ResourceName: "demo",
		},
		nil,
		nil,
		[]model.JSONMap{
			{
				"name":         "image.png",
				"content_type": "image/png",
				"data_url":     "data:image/png;base64,abc",
				"size":         int64(123),
			},
		},
		nil,
	)

	requestScope, ok := snapshot["request_scope"].(model.JSONMap)
	if !ok {
		t.Fatalf("expected request_scope JSON map, got %#v", snapshot["request_scope"])
	}
	if got := requestScope["image_count"]; got != 1 {
		t.Fatalf("request_scope.image_count = %#v, want 1", got)
	}
	images, ok := snapshot["request_images"].([]model.JSONMap)
	if !ok || len(images) != 1 {
		t.Fatalf("request_images = %#v, want one image", snapshot["request_images"])
	}
	if got := images[0]["name"]; got != "image.png" {
		t.Fatalf("request_images[0].name = %#v, want %q", got, "image.png")
	}
}

func TestBuildAIMessageScopeSnapshotIncludesAttachments(t *testing.T) {
	snapshot := buildAIMessageScopeSnapshot(
		model.AIConversation{
			ID:            9,
			AssistantMode: "chat",
		},
		AIChatRequest{
			ClusterID: 4,
			Namespace: "ops",
		},
		nil,
		nil,
		nil,
		[]AIMessageAttachmentItem{
			{
				ID:           12,
				OriginalName: "events.log",
				ContentType:  "text/plain",
				FileSize:     256,
				FileKind:     "text",
				DownloadURL:  "/api/v1/ai/files/12/content",
			},
		},
	)

	requestScope, ok := snapshot["request_scope"].(model.JSONMap)
	if !ok {
		t.Fatalf("expected request_scope JSON map, got %#v", snapshot["request_scope"])
	}
	if got := requestScope["attachment_count"]; got != 1 {
		t.Fatalf("request_scope.attachment_count = %#v, want 1", got)
	}
	attachments, ok := snapshot["request_attachments"].([]model.JSONMap)
	if !ok || len(attachments) != 1 {
		t.Fatalf("request_attachments = %#v, want one attachment", snapshot["request_attachments"])
	}
	if got := attachments[0]["original_name"]; got != "events.log" {
		t.Fatalf("request_attachments[0].original_name = %#v, want %q", got, "events.log")
	}
}

func TestBuildAIGatewayScopeNote(t *testing.T) {
	note := aigateway.BuildScopeNote("devops", "Deployment", "bkci-auth")
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

func TestBuildAIDiagnosticEvidenceDigest(t *testing.T) {
	digest := buildAIDiagnosticEvidenceDigest([]AIToolCallItem{
		{
			ToolName:      "cluster.health",
			Status:        "succeeded",
			ResultSummary: "API true, nodes ready 8/8",
		},
		{
			ToolName:      "cluster.overview",
			Status:        "succeeded",
			ResultSummary: "pods 386 total, CPU 5%, memory 65%",
		},
		{
			ToolName:      "resource.logs",
			Status:        "failed",
			ResultSummary: "should not be included",
		},
	}, "Tool: cluster.health\nEvidence JSON:\n{}")

	if !strings.Contains(digest, "Confirmed platform evidence for this round") {
		t.Fatalf("expected digest header, got %q", digest)
	}
	if !strings.Contains(digest, "cluster.health: API true, nodes ready 8/8") {
		t.Fatalf("expected concise cluster.health summary, got %q", digest)
	}
	if !strings.Contains(digest, "Detailed platform evidence") {
		t.Fatalf("expected detailed evidence section, got %q", digest)
	}
	if strings.Contains(digest, "should not be included") {
		t.Fatalf("failed tool summary should not be included in digest: %q", digest)
	}
}

func TestEffectiveAIGatewayMode(t *testing.T) {
	if got := effectiveAIGatewayMode("chat", []AIToolCallItem{
		{ToolName: "cluster.health", Status: "succeeded", ResultSummary: "ok"},
	}, "evidence"); got != "diagnose" {
		t.Fatalf("effectiveAIGatewayMode(chat with evidence) = %q, want diagnose", got)
	}

	if got := effectiveAIGatewayMode("chat", nil, ""); got != "chat" {
		t.Fatalf("effectiveAIGatewayMode(chat without evidence) = %q, want chat", got)
	}

	if got := effectiveAIGatewayMode("diagnose", nil, ""); got != "diagnose" {
		t.Fatalf("effectiveAIGatewayMode(diagnose) = %q, want diagnose", got)
	}
}

func TestBuildScopedGatewayHistoryFiltersPreviousScope(t *testing.T) {
	rows := []model.AIMessage{
		{
			ID:      1,
			Role:    "user",
			Content: "show deployment a",
			StructuredJSON: model.JSONMap{
				"request_scope": model.JSONMap{
					"assistant_mode": "diagnose",
					"namespace":      "payments",
					"resource_kind":  "Deployment",
					"resource_name":  "api-a",
				},
			},
		},
		{
			ID:      2,
			Role:    "assistant",
			Content: "deployment a reply",
			StructuredJSON: model.JSONMap{
				"request_scope": model.JSONMap{
					"assistant_mode": "diagnose",
					"namespace":      "payments",
					"resource_kind":  "Deployment",
					"resource_name":  "api-a",
				},
			},
		},
		{
			ID:      3,
			Role:    "user",
			Content: "show deployment b",
			StructuredJSON: model.JSONMap{
				"request_scope": model.JSONMap{
					"assistant_mode": "diagnose",
					"namespace":      "payments",
					"resource_kind":  "Deployment",
					"resource_name":  "api-b",
				},
			},
		},
	}

	got := buildScopedGatewayHistory(rows, aiMessageRequestScope{
		AssistantMode: "diagnose",
		Namespace:     "payments",
		ResourceKind:  "Deployment",
		ResourceName:  "api-b",
	})

	if len(got) != 1 {
		t.Fatalf("buildScopedGatewayHistory() len = %d, want 1; rows=%#v", len(got), got)
	}
	if got[0].ID != 3 {
		t.Fatalf("buildScopedGatewayHistory()[0].ID = %d, want 3", got[0].ID)
	}
}

func TestBuildScopedGatewayHistoryKeepsCurrentScopeChain(t *testing.T) {
	rows := []model.AIMessage{
		{
			ID:      10,
			Role:    "user",
			Content: "first question",
			StructuredJSON: model.JSONMap{
				"request_scope": model.JSONMap{
					"assistant_mode": "chat",
					"namespace":      "devops",
					"resource_kind":  "Deployment",
					"resource_name":  "bkci-auth",
				},
			},
		},
		{
			ID:      11,
			Role:    "assistant",
			Content: "first reply",
			StructuredJSON: model.JSONMap{
				"request_scope": model.JSONMap{
					"assistant_mode": "chat",
					"namespace":      "devops",
					"resource_kind":  "Deployment",
					"resource_name":  "bkci-auth",
				},
			},
		},
		{
			ID:      12,
			Role:    "user",
			Content: "follow up",
			StructuredJSON: model.JSONMap{
				"request_scope": model.JSONMap{
					"assistant_mode": "chat",
					"namespace":      "devops",
					"resource_kind":  "Deployment",
					"resource_name":  "bkci-auth",
				},
			},
		},
	}

	got := buildScopedGatewayHistory(rows, aiMessageRequestScope{
		AssistantMode: "chat",
		Namespace:     "devops",
		ResourceKind:  "Deployment",
		ResourceName:  "bkci-auth",
	})

	if len(got) != 3 {
		t.Fatalf("buildScopedGatewayHistory() len = %d, want 3; rows=%#v", len(got), got)
	}
	for i, wantID := range []uint64{10, 11, 12} {
		if got[i].ID != wantID {
			t.Fatalf("buildScopedGatewayHistory()[%d].ID = %d, want %d", i, got[i].ID, wantID)
		}
	}
}
