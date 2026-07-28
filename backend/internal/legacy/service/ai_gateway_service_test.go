package service

import (
	"context"
	"strings"
	"testing"
)

func TestResolveInvocationTarget(t *testing.T) {
	t.Skip("跳过需要数据库的测试")
	tests := []struct {
		name    string
		req     AIGatewayRequest
		wantErr bool
	}{
		{
			name: "valid request with provider and model",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(1),
				ModelID:    uint64Ptr(1),
			},
			wantErr: false,
		},
		{
			name: "request with prefer model code",
			req: AIGatewayRequest{
				PreferModelCode: "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "request with invalid provider",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(999),
			},
			wantErr: true,
		},
		{
			name: "request with invalid model",
			req: AIGatewayRequest{
				ModelID: uint64Ptr(999),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &AIGatewayService{}
			_, _, _, err := service.resolveInvocationTarget(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveInvocationTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestBuildAISystemPromptV2(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{
			name: "diagnose mode",
			mode: "diagnose",
			want: "诊断模式提示词",
		},
		{
			name: "chat mode",
			mode: "chat",
			want: "聊天模式提示词",
		},
		{
			name: "default mode",
			mode: "",
			want: "默认提示词",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAISystemPromptV2(tt.mode)
			if len(got) == 0 {
				t.Errorf("buildAISystemPromptV2() returned empty string")
			}
			if !strings.Contains(got, "Format the final answer in Markdown") {
				t.Errorf("buildAISystemPromptV2() should require Markdown formatting")
			}
		})
	}
}

func TestInvoke(t *testing.T) {
	t.Skip("跳过需要数据库的测试")
	tests := []struct {
		name    string
		req     AIGatewayRequest
		wantErr bool
	}{
		{
			name: "mock provider request",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(1),
				ModelID:    uint64Ptr(1),
			},
			wantErr: false,
		},
		{
			name: "openai compatible provider request",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(2),
				ModelID:    uint64Ptr(2),
			},
			wantErr: false,
		},
		{
			name: "unsupported provider type",
			req: AIGatewayRequest{
				ProviderID: uint64Ptr(3),
				ModelID:    uint64Ptr(3),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &AIGatewayService{}
			_, err := service.Invoke(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

func TestBuildOpenAICompatibleMessagesPlacesFreshEvidenceNearCurrentUser(t *testing.T) {
	req := AIGatewayRequest{
		AssistantMode: "diagnose",
		Messages: []AIGatewayMessage{
			{Role: "user", Content: "old question"},
			{Role: "assistant", Content: "old answer without evidence"},
			{Role: "user", Content: "inspect current cluster"},
		},
		DiagnosticNotes: "Platform diagnostic tool: cluster.health\nSummary: API true, nodes ready 8/8",
	}

	messages := buildOpenAICompatibleMessages(req)
	if len(messages) < 4 {
		t.Fatalf("expected enough messages, got %d", len(messages))
	}

	last := messages[len(messages)-1]
	if role := strings.TrimSpace(last["role"].(string)); role != "user" {
		t.Fatalf("expected last message to be current user, got %q", role)
	}
	if content := strings.TrimSpace(last["content"].(string)); !strings.Contains(content, "inspect current cluster") {
		t.Fatalf("expected last user message content to be preserved, got %q", content)
	}

	evidenceIndex := -1
	oldAssistantIndex := -1
	for index, message := range messages {
		content := strings.TrimSpace(message["content"].(string))
		if strings.Contains(content, "Fresh platform diagnostic evidence exists for this round") {
			evidenceIndex = index
		}
		if strings.Contains(content, "old answer without evidence") {
			oldAssistantIndex = index
		}
		if strings.Contains(content, "No platform diagnostic evidence was collected for this round") {
			t.Fatal("did not expect no-evidence warning when diagnostic notes exist")
		}
	}

	if evidenceIndex < 0 {
		t.Fatal("expected fresh evidence system message to be present")
	}
	if oldAssistantIndex < 0 {
		t.Fatal("expected prior assistant history to be present")
	}
	if evidenceIndex <= oldAssistantIndex {
		t.Fatalf("expected fresh evidence message after prior history, got evidence=%d history=%d", evidenceIndex, oldAssistantIndex)
	}
}

func TestBuildOpenAICompatibleMessagesAddsNoEvidenceWarningOnlyWhenNotesMissing(t *testing.T) {
	req := AIGatewayRequest{
		AssistantMode: "diagnose",
		Messages: []AIGatewayMessage{
			{Role: "user", Content: "inspect current cluster"},
		},
	}

	messages := buildOpenAICompatibleMessages(req)
	found := false
	for _, message := range messages {
		content, _ := message["content"].(string)
		if strings.Contains(content, "No platform diagnostic evidence was collected for this round") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected no-evidence guidance when diagnostic notes are missing")
	}
}

func TestBuildOpenAICurrentTurnContentIncludesDiagnosticSummary(t *testing.T) {
	content := buildOpenAICurrentTurnContent(
		"inspect current cluster",
		"Confirmed platform evidence for this round:\n- cluster.health: API true, nodes ready 8/8",
		"",
	)

	if !strings.Contains(content, "Confirmed platform evidence for this round") {
		t.Fatalf("expected diagnostic summary in current turn content, got %q", content)
	}
	if !strings.Contains(content, "Current user request:\ninspect current cluster") {
		t.Fatalf("expected original user request to remain in current turn content, got %q", content)
	}
}
