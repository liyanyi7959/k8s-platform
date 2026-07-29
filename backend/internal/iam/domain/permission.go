package domain

import "strings"

type PermissionMetadata struct {
	Code          string
	Description   string
	Category      string
	CategoryLabel string
	Builtin       bool
}

var builtinPermissions = []PermissionMetadata{
	{Code: "cluster:read", Description: "集群查看", Category: "cluster", CategoryLabel: "平台与集群", Builtin: true},
	{Code: "cluster:create", Description: "集群接入与变更", Category: "cluster", CategoryLabel: "平台与集群", Builtin: true},
	{Code: "project:read", Description: "项目查看", Category: "project", CategoryLabel: "项目管理", Builtin: true},
	{Code: "project:write", Description: "项目管理", Category: "project", CategoryLabel: "项目管理", Builtin: true},
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
	{Code: "monitor:read", Description: "监控规则与事件查看", Category: "monitor", CategoryLabel: "监控与事件", Builtin: true},
	{Code: "monitor:write", Description: "监控规则管理", Category: "monitor", CategoryLabel: "监控与事件", Builtin: true},
	{Code: "incident:manage", Description: "事件认领、处置与验证", Category: "monitor", CategoryLabel: "监控与事件", Builtin: true},
	{Code: "automation:read", Description: "自动化任务查看", Category: "automation", CategoryLabel: "自动化与变更", Builtin: true},
	{Code: "automation:execute", Description: "自动化任务取消与执行", Category: "automation", CategoryLabel: "自动化与变更", Builtin: true},
	{Code: "credential:read", Description: "凭据库查看", Category: "credential", CategoryLabel: "凭据与密钥", Builtin: true},
	{Code: "credential:write", Description: "凭据库管理", Category: "credential", CategoryLabel: "凭据与密钥", Builtin: true},
	{Code: "credential:delete", Description: "凭据库删除", Category: "credential", CategoryLabel: "凭据与密钥", Builtin: true},
	{Code: "deploy:server_read", Description: "部署服务器查看", Category: "deploy", CategoryLabel: "在线部署", Builtin: true},
	{Code: "deploy:server_write", Description: "部署服务器管理", Category: "deploy", CategoryLabel: "在线部署", Builtin: true},
	{Code: "deploy:server_delete", Description: "部署服务器删除", Category: "deploy", CategoryLabel: "在线部署", Builtin: true},
	{Code: "deploy:plan_read", Description: "部署计划查看", Category: "deploy", CategoryLabel: "在线部署", Builtin: true},
	{Code: "deploy:plan_write", Description: "部署计划管理", Category: "deploy", CategoryLabel: "在线部署", Builtin: true},
	{Code: "deploy:plan_delete", Description: "部署计划删除", Category: "deploy", CategoryLabel: "在线部署", Builtin: true},
	{Code: "deploy:execute", Description: "执行在线部署", Category: "deploy", CategoryLabel: "在线部署", Builtin: true},
	{Code: "appstore:read", Description: "应用商店查看", Category: "appstore", CategoryLabel: "应用商店", Builtin: true},
	{Code: "appstore:write", Description: "应用商店模板管理", Category: "appstore", CategoryLabel: "应用商店", Builtin: true},
}

func BuiltinPermissionCatalog() []PermissionMetadata {
	result := make([]PermissionMetadata, len(builtinPermissions))
	copy(result, builtinPermissions)
	return result
}

func DescribePermission(code string, description *string) PermissionMetadata {
	for _, item := range builtinPermissions {
		if item.Code == code {
			return item
		}
	}
	value := code
	if description != nil && strings.TrimSpace(*description) != "" {
		value = strings.TrimSpace(*description)
	}
	category, label := permissionCategory(code)
	return PermissionMetadata{Code: code, Description: value, Category: category, CategoryLabel: label}
}

func PermissionCategoryOrder(category string) int {
	orders := map[string]int{"system": 1, "cluster": 2, "credential": 3, "monitor": 4, "automation": 5, "project": 6, "namespace": 6, "k8s": 7, "rbac": 8, "ai": 9, "deploy": 10, "appstore": 11, "security": 12}
	if order, ok := orders[category]; ok {
		return order
	}
	return 99
}

func HasNamespacePermission(codes []string) bool {
	for _, code := range codes {
		if strings.HasPrefix(strings.TrimSpace(code), "namespace:") {
			return true
		}
	}
	return false
}

func permissionCategory(code string) (string, string) {
	switch {
	case strings.HasPrefix(code, "cluster:"):
		return "cluster", "平台与集群"
	case strings.HasPrefix(code, "project:"):
		return "project", "项目管理"
	case strings.HasPrefix(code, "namespace:"):
		return "namespace", "命名空间"
	case strings.HasPrefix(code, "credential:"):
		return "credential", "凭据与密钥"
	case strings.HasPrefix(code, "monitor:") || strings.HasPrefix(code, "incident:"):
		return "monitor", "监控与事件"
	case strings.HasPrefix(code, "automation:"):
		return "automation", "自动化与变更"
	case strings.HasPrefix(code, "deploy:"):
		return "deploy", "在线部署"
	case strings.HasPrefix(code, "appstore:"):
		return "appstore", "应用商店"
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
