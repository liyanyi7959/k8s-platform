package application

import (
	"strings"
	"testing"
)

func TestBuildChatDiagnosticEvidenceDigest(t *testing.T) {
	digest := BuildChatDiagnosticEvidenceDigest([]ToolCallItem{
		{ToolName: "cluster.health", Status: "succeeded", ResultSummary: "API true, nodes ready 8/8"},
		{ToolName: "cluster.overview", Status: "succeeded", ResultSummary: "pods 386 total, CPU 5%, memory 65%"},
		{ToolName: "resource.logs", Status: "failed", ResultSummary: "should not be included"},
	}, "Tool: cluster.health\nEvidence JSON:\n{}")

	for _, expected := range []string{
		"Confirmed platform evidence for this round",
		"cluster.health: API true, nodes ready 8/8",
		"Detailed platform evidence",
	} {
		if !strings.Contains(digest, expected) {
			t.Fatalf("digest does not contain %q: %q", expected, digest)
		}
	}
	if strings.Contains(digest, "should not be included") {
		t.Fatalf("failed tool summary should not be included in digest: %q", digest)
	}
}

func TestEffectiveChatGatewayMode(t *testing.T) {
	if got := EffectiveChatGatewayMode("chat", []ToolCallItem{{ToolName: "cluster.health", Status: "succeeded", ResultSummary: "ok"}}, "evidence"); got != "diagnose" {
		t.Fatalf("EffectiveChatGatewayMode(chat with evidence) = %q, want diagnose", got)
	}
	if got := EffectiveChatGatewayMode("chat", nil, ""); got != "chat" {
		t.Fatalf("EffectiveChatGatewayMode(chat without evidence) = %q, want chat", got)
	}
	if got := EffectiveChatGatewayMode("diagnose", nil, ""); got != "diagnose" {
		t.Fatalf("EffectiveChatGatewayMode(diagnose) = %q, want diagnose", got)
	}
}
