package application

import (
	"strings"
	"testing"

	"k8s-platform-backend/internal/ai/domain"
)

func TestNormalizeModelAnswerStripsInvalidToolCalls(t *testing.T) {
	content, structured := NormalizeModelAnswer("排查结果如下。\n<tool_call>{\"name\":\"cluster.overview\"}</tool_call>\n请继续查看 deployment 事件。")
	if strings.Contains(content, "<tool_call>") {
		t.Fatalf("normalized content still contains tool call block: %q", content)
	}
	guard, ok := structured["response_guard"].(domain.JSONMap)
	if !ok || guard["tool_call_stripped"] != true {
		t.Fatalf("expected tool-call response guard, got %#v", structured)
	}
}

func TestNormalizeModelAnswerUsesFallbackForOnlyToolCalls(t *testing.T) {
	content, structured := NormalizeModelAnswer("<tool_call>{\"name\":\"cluster.overview\"}</tool_call>")
	if !strings.Contains(content, "无效的工具调用片段") {
		t.Fatalf("expected fallback content, got %q", content)
	}
	guard, ok := structured["response_guard"].(domain.JSONMap)
	if !ok || guard["fallback_applied"] != true {
		t.Fatalf("expected fallback response guard, got %#v", structured)
	}
}

func TestNormalizeModelAnswerParsesValidSuggestedActions(t *testing.T) {
	content, structured := NormalizeModelAnswer("建议重启工作负载。\nAI_ACTIONS_JSON:{\"suggested_actions\":[{\"action_type\":\"restart_workload\",\"target_kind\":\"deployment\",\"target_namespace\":\"default\",\"target_name\":\"api\"}]}")
	if content != "建议重启工作负载。" {
		t.Fatalf("content = %q", content)
	}
	actions, ok := structured["suggested_actions"].([]map[string]any)
	if !ok || len(actions) != 1 || actions[0]["action_type"] != ActionTypeRestartWorkload {
		t.Fatalf("suggested actions = %#v", structured)
	}
}
