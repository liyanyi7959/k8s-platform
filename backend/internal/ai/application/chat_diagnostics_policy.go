package application

import (
	"strings"
	"time"
)

// ChatDiagnosticsRequest contains only the scope required to decide whether a
// chat turn should collect live Kubernetes evidence.
type ChatDiagnosticsRequest struct {
	AssistantMode string
	Namespace     string
	ResourceKind  string
	ResourceName  string
	Message       string
}

type ChatDiagnosticsPlan struct {
	Enabled  bool
	Optional bool
	Timeout  time.Duration
}

func BuildChatDiagnosticsPlan(request ChatDiagnosticsRequest) ChatDiagnosticsPlan {
	if NormalizeAssistantMode(request.AssistantMode) == "diagnose" {
		return ChatDiagnosticsPlan{Enabled: true, Timeout: 20 * time.Second}
	}
	if ShouldRunChatDiagnostics(request) {
		return ChatDiagnosticsPlan{Enabled: true, Optional: true, Timeout: 12 * time.Second}
	}
	return ChatDiagnosticsPlan{}
}

func ShouldRunChatDiagnostics(request ChatDiagnosticsRequest) bool {
	message := strings.TrimSpace(request.Message)
	if message == "" || IsGenericKnowledgeQuestion(message) {
		return false
	}
	namespace := strings.TrimSpace(request.Namespace)
	if strings.TrimSpace(request.ResourceKind) != "" && strings.TrimSpace(request.ResourceName) != "" {
		return true
	}
	if namespace != "" {
		return LooksLikeScopedClusterQuestion(message) || NeedsBroadInspection(message) || NeedsConfigSearch(message) || NeedsResourceYAML(message)
	}
	return NeedsClusterInspection(message) || NeedsBroadInspection(message) || NeedsControlPlaneInspection(message)
}

func IsGenericKnowledgeQuestion(message string) bool {
	if containsChatPolicyAny(message, "哪些方面", "从哪些方面", "一般怎么", "通常怎么", "如何", "怎么做", "是什么", "有哪些", "最佳实践", "注意事项", "巡检思路", "排查思路", "设计方案", "实施步骤") {
		return true
	}
	return containsChatPolicyAny(strings.ToLower(strings.TrimSpace(message)), "what is", "how to", "best practice", "checklist", "overview", "introduction", "general", "typically", "usually")
}

func LooksLikeScopedClusterQuestion(message string) bool {
	if containsChatPolicyAny(message, "这个集群", "当前集群", "本集群", "这个命名空间", "当前命名空间", "这个服务", "这个 deployment", "这个 pod", "帮我看", "帮我查", "看看", "查一下", "分析一下", "诊断一下", "排查一下") {
		return true
	}
	return containsChatPolicyAny(strings.ToLower(strings.TrimSpace(message)), "check", "inspect", "diagnose", "analyze", "look into", "current cluster", "this cluster", "this namespace", "show me", "health", "usage", "metrics")
}

func NeedsClusterInspection(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	return text != "" && (containsChatPolicyAny(text, "current cluster", "this cluster", "cluster status", "cluster overview", "cluster health") || containsChatPolicyAny(message, "当前集群", "这个集群", "本集群", "集群状态", "集群概览", "集群健康"))
}

func NeedsControlPlaneInspection(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	return text != "" && (containsChatPolicyAny(text, "control plane", "apiserver", "api server", "controller manager", "scheduler", "etcd", "certificate", "cert expiry") || containsChatPolicyAny(message, "控制面", "api server", "apiserver", "调度器", "controller-manager", "etcd", "证书", "证书风险", "证书过期"))
}

func NeedsBroadInspection(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	return text != "" && (containsChatPolicyAny(text, "inspection", "full inspection", "full check", "health check", "resource usage", "metrics", "usage", "inventory") || containsChatPolicyAny(message, "巡检", "巡查", "完整检查", "完整巡检", "资源使用", "使用情况", "资源个数", "资源数量", "指标", "健康检查", "排查"))
}

func NeedsResourceYAML(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	return text != "" && (containsChatPolicyAny(text, "yaml", "manifest", "spec", "mysql", "database", "connection", "password", "config", "secret", "environment", "env", "datasource") || containsChatPolicyAny(message, "配置", "清单", "导出", "数据库", "连接", "密码", "密钥", "环境变量", "数据源"))
}

func NeedsConfigSearch(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	return text != "" && (containsChatPolicyAny(text, "mysql", "database", "redis", "mongodb", "postgres", "config") || containsChatPolicyAny(message, "数据库", "配置", "连接", "密码", "密钥", "数据源", "环境变量"))
}

func containsChatPolicyAny(text string, candidates ...string) bool {
	for _, candidate := range candidates {
		if candidate != "" && strings.Contains(text, candidate) {
			return true
		}
	}
	return false
}
