package service

import (
	"strings"
	"testing"

	"k8s-platform-backend/internal/model"
)

func TestNormalizeAIModelAnswer_StripsInvalidToolCalls(t *testing.T) {
	content, structured := normalizeAIModelAnswer("排查结果如下。\n<tool_call>{\"name\":\"cluster.overview\"}</tool_call>\n请继续查看 deployment 事件。")
	if strings.Contains(content, "<tool_call>") {
		t.Fatalf("normalized content still contains tool call block: %q", content)
	}
	if !strings.Contains(content, "排查结果如下。") || !strings.Contains(content, "请继续查看 deployment 事件。") {
		t.Fatalf("normalized content lost expected user-facing text: %q", content)
	}

	guard, ok := structured["response_guard"].(model.JSONMap)
	if !ok {
		t.Fatalf("expected response_guard JSON map, got %#v", structured["response_guard"])
	}
	if got := guard["tool_call_stripped"]; got != true {
		t.Fatalf("response_guard.tool_call_stripped = %#v, want true", got)
	}
}

func TestNormalizeAIModelAnswer_UsesFallbackWhenOnlyInvalidToolCallsRemain(t *testing.T) {
	content, structured := normalizeAIModelAnswer("<tool_call>{\"name\":\"cluster.overview\"}</tool_call>")
	if !strings.Contains(content, "无效的工具调用片段") {
		t.Fatalf("expected fallback content, got %q", content)
	}

	guard, ok := structured["response_guard"].(model.JSONMap)
	if !ok {
		t.Fatalf("expected response_guard JSON map, got %#v", structured["response_guard"])
	}
	if got := guard["tool_call_stripped"]; got != true {
		t.Fatalf("response_guard.tool_call_stripped = %#v, want true", got)
	}
	if got := guard["fallback_applied"]; got != true {
		t.Fatalf("response_guard.fallback_applied = %#v, want true", got)
	}
}
