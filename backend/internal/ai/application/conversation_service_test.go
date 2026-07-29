package application

import (
	"testing"
	"time"

	"k8s-platform-backend/internal/ai/domain"
)

func TestNormalizeAssistantMode(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "", want: "diagnose"},
		{input: "DIAGNOSE", want: "diagnose"},
		{input: " chat ", want: "chat"},
		{input: "invalid", want: ""},
	} {
		if got := NormalizeAssistantMode(test.input); got != test.want {
			t.Fatalf("NormalizeAssistantMode(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestBuildConversationItem(t *testing.T) {
	now := time.Date(2026, 7, 29, 9, 0, 0, 0, time.UTC)
	item := BuildConversationItem(domain.AIConversation{
		ID: 11, ClusterID: 5, Title: "诊断", Status: "open", AssistantMode: "diagnose",
		CreatedAt: now, UpdatedAt: now, LastMessageAt: &now,
	}, 3)
	if item.ID != 11 || item.MessageCount != 3 || item.LastMessageAt == nil {
		t.Fatalf("unexpected conversation item: %+v", item)
	}
}
