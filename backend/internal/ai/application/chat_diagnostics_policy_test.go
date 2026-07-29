package application

import "testing"

func TestBuildChatDiagnosticsPlan(t *testing.T) {
	if plan := BuildChatDiagnosticsPlan(ChatDiagnosticsRequest{AssistantMode: "diagnose"}); !plan.Enabled || plan.Optional {
		t.Fatalf("diagnose plan = %#v", plan)
	}
	if plan := BuildChatDiagnosticsPlan(ChatDiagnosticsRequest{AssistantMode: "chat", Namespace: "ops", Message: "how to perform a cluster health check"}); plan.Enabled {
		t.Fatalf("generic knowledge plan = %#v", plan)
	}
	if plan := BuildChatDiagnosticsPlan(ChatDiagnosticsRequest{AssistantMode: "chat", Namespace: "ops", ResourceKind: "Deployment", ResourceName: "api", Message: "is it healthy?"}); !plan.Enabled || !plan.Optional {
		t.Fatalf("scoped resource plan = %#v", plan)
	}
	if plan := BuildChatDiagnosticsPlan(ChatDiagnosticsRequest{AssistantMode: "chat", Message: "check current cluster certificate expiry"}); !plan.Enabled {
		t.Fatalf("control-plane plan = %#v", plan)
	}
}

func TestNeedsConfigSearch(t *testing.T) {
	if !NeedsConfigSearch("检查数据库连接和密码配置") {
		t.Fatal("expected configuration search for database intent")
	}
	if NeedsConfigSearch("what is a deployment") {
		t.Fatal("unexpected configuration search for generic question")
	}
}
