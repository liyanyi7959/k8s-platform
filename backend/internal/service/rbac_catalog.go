package service

import "strings"

type PermissionCatalogItem struct {
	Code          string
	Description   string
	Category      string
	CategoryLabel string
	Builtin       bool
}

var builtinPermissionCatalog = []PermissionCatalogItem{
	{Code: "cluster:read", Description: "集群查看", Category: "cluster", CategoryLabel: "平台与集群", Builtin: true},
	{Code: "cluster:create", Description: "集群接入与变更", Category: "cluster", CategoryLabel: "平台与集群", Builtin: true},
	{Code: "namespace:read", Description: "命名空间查看", Category: "namespace", CategoryLabel: "命名空间", Builtin: true},
	{Code: "namespace:write", Description: "命名空间管理", Category: "namespace", CategoryLabel: "命名空间", Builtin: true},
	{Code: "k8s:read", Description: "K8s 通用资源查看", Category: "k8s", CategoryLabel: "工作负载与资源", Builtin: true},
	{Code: "k8s:write", Description: "K8s 通用资源管理", Category: "k8s", CategoryLabel: "工作负载与资源", Builtin: true},
	{Code: "k8s:exec", Description: "Pod 终端进入", Category: "security", CategoryLabel: "高风险与敏感操作", Builtin: true},
	{Code: "k8s:secret_reveal", Description: "Secret 明文查看", Category: "security", CategoryLabel: "高风险与敏感操作", Builtin: true},
	{Code: "k8s:rbac_read", Description: "K8s RBAC 查看", Category: "rbac", CategoryLabel: "RBAC 与权限治理", Builtin: true},
	{Code: "k8s:rbac_write", Description: "K8s RBAC 管理", Category: "rbac", CategoryLabel: "RBAC 与权限治理", Builtin: true},
	{Code: "k8s:permission_audit", Description: "K8s 最小权限分析", Category: "rbac", CategoryLabel: "RBAC 与权限治理", Builtin: true},
	{Code: "ai:chat", Description: "AI 助手对话", Category: "ai", CategoryLabel: "AI 助手", Builtin: true},
	{Code: "ai:diagnose", Description: "AI 故障诊断", Category: "ai", CategoryLabel: "AI 助手", Builtin: true},
	{Code: "ai:image", Description: "AI 图片理解", Category: "ai", CategoryLabel: "AI 助手", Builtin: true},
	{Code: "ai:tool_exec", Description: "AI 查询工具执行", Category: "ai", CategoryLabel: "AI 助手", Builtin: true},
	{Code: "ai:change_propose", Description: "AI 变更建议生成", Category: "ai", CategoryLabel: "AI 助手", Builtin: true},
	{Code: "ai:change_confirm", Description: "AI 变更确认审批", Category: "ai", CategoryLabel: "AI 助手", Builtin: true},
	{Code: "ai:model_admin", Description: "AI 模型与提供商配置", Category: "ai", CategoryLabel: "AI 助手", Builtin: true},
	{Code: "ai:audit_read", Description: "AI 审计记录查看", Category: "ai", CategoryLabel: "AI 助手", Builtin: true},
	{Code: "user:read", Description: "用户、角色与审计查看", Category: "system", CategoryLabel: "系统管理", Builtin: true},
	{Code: "user:write", Description: "用户与角色管理", Category: "system", CategoryLabel: "系统管理", Builtin: true},
}

func BuiltinPermissionCatalog() []PermissionCatalogItem {
	out := make([]PermissionCatalogItem, len(builtinPermissionCatalog))
	copy(out, builtinPermissionCatalog)
	return out
}

func BuiltinPermissions() map[string]string {
	out := make(map[string]string, len(builtinPermissionCatalog))
	for _, item := range builtinPermissionCatalog {
		out[item.Code] = item.Description
	}
	return out
}

func DescribePermission(code string, desc *string) PermissionCatalogItem {
	for _, item := range builtinPermissionCatalog {
		if item.Code != code {
			continue
		}
		if item.Description == "" && desc != nil {
			item.Description = strings.TrimSpace(*desc)
		}
		return item
	}

	description := code
	if desc != nil && strings.TrimSpace(*desc) != "" {
		description = strings.TrimSpace(*desc)
	}
	category, categoryLabel := derivePermissionCategory(code)
	return PermissionCatalogItem{
		Code:          code,
		Description:   description,
		Category:      category,
		CategoryLabel: categoryLabel,
		Builtin:       false,
	}
}

func derivePermissionCategory(code string) (string, string) {
	switch {
	case strings.HasPrefix(code, "cluster:"):
		return "cluster", "平台与集群"
	case strings.HasPrefix(code, "namespace:"):
		return "namespace", "命名空间"
	case strings.HasPrefix(code, "ai:"):
		return "ai", "AI 助手"
	case strings.HasPrefix(code, "k8s:rbac_") || code == "k8s:permission_audit":
		return "rbac", "RBAC 与权限治理"
	case code == "k8s:exec" || code == "k8s:secret_reveal":
		return "security", "高风险与敏感操作"
	case strings.HasPrefix(code, "k8s:"):
		return "k8s", "工作负载与资源"
	case strings.HasPrefix(code, "user:"):
		return "system", "系统管理"
	default:
		return "custom", "自定义权限"
	}
}

func permissionCategoryOrder(category string) int {
	switch category {
	case "system":
		return 1
	case "cluster":
		return 2
	case "namespace":
		return 3
	case "k8s":
		return 4
	case "rbac":
		return 5
	case "ai":
		return 6
	case "security":
		return 7
	default:
		return 99
	}
}
