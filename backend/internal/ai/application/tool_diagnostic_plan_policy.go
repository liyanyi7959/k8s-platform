package application

import (
	"strings"

	"k8s-platform-backend/internal/ai/domain"
)

// BuildAutoDiagnosticToolPlan selects read-only tools for an AI diagnostic
// turn. Availability is supplied by the runtime registry so this policy has
// no dependency on tool handlers or Kubernetes clients.
func BuildAutoDiagnosticToolPlan(request ToolContextRequest, available func(string) bool) []ToolPlanStep {
	if request.ClusterID == 0 {
		return nil
	}
	if available == nil {
		available = func(string) bool { return true }
	}

	steps := make([]ToolPlanStep, 0, 6)
	add := func(toolName, reason string, params domain.JSONMap) {
		if !available(toolName) {
			return
		}
		steps = append(steps, ToolPlanStep{ToolName: toolName, Params: cloneDiagnosticPlanJSON(params), Reason: strings.TrimSpace(reason)})
	}

	query := strings.TrimSpace(request.Query)
	namespace := strings.TrimSpace(request.Namespace)
	kind := strings.TrimSpace(request.ResourceKind)
	name := strings.TrimSpace(request.ResourceName)
	strictResourceScope := kind != "" && name != ""
	broadInspection := NeedsBroadInspection(query)
	clusterInspection := NeedsClusterInspection(query)
	controlPlaneInspection := NeedsControlPlaneInspection(query)
	broadensScope := ExplicitlyBroadensResourceScope(query)
	yamlIntent := NeedsResourceYAML(query)

	if !strictResourceScope || clusterInspection || controlPlaneInspection || broadInspection || broadensScope {
		add("cluster.health", "Baseline cluster health reused from platform read model", domain.JSONMap{"cluster_id": request.ClusterID})
	}
	if !strictResourceScope && (namespace == "" || broadInspection || clusterInspection) {
		add("cluster.overview", "Cluster-wide overview requested by the current question scope", domain.JSONMap{"cluster_id": request.ClusterID})
	}
	if !strictResourceScope && namespace == "" && (controlPlaneInspection || broadInspection) {
		add("cluster.certificate_risks", "Control plane or certificate risk inspection requested", domain.JSONMap{"cluster_id": request.ClusterID})
	}

	if namespace != "" {
		if strictResourceScope {
			if broadInspection || broadensScope {
				add("namespace.inspect", "The current question explicitly broadens from the target resource to the namespace scope", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace})
			} else if NeedsPodOwnershipContext(kind, query) {
				add("namespace.workloads", "Scoped pod inspection needs workload inventory to explain ownership", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace})
			}
		} else if broadInspection {
			add("namespace.inspect", "Full namespace inspection is needed for the current scoped question", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace})
		} else {
			add("namespace.health", "Scoped namespace health evidence is needed", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace})
			add("namespace.summary", "Scoped namespace inventory summary is needed", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace})
			if strings.EqualFold(kind, "pod") || containsAutoDiagnosticPlanAny(strings.ToLower(query), "deployment", "statefulset", "daemonset", "replicaset", "workload", "replica") {
				add("namespace.workloads", "Scoped workload inventory is needed to explain pod ownership and replica context", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace})
			}
		}
	}

	if kind != "" && name == "" && namespace != "" {
		add("resource.list", "用户指定了资源类型但未指定名称，先列出该类型的资源", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace, "kind": kind})
		return steps
	}
	if kind == "" || name == "" {
		if namespace != "" && (yamlIntent || NeedsConfigSearch(query)) {
			add("namespace.workloads", "列出工作负载，帮助发现 Redis 相关应用及其环境变量配置", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace})
			add("resource.list", "用户可能需要查看配置资源，先列出命名空间下的 ConfigMap", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace, "kind": "ConfigMap"})
			add("resource.list", "用户可能需要查看密钥资源，先列出命名空间下的 Secret", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace, "kind": "Secret"})
			add("resource.list", "用户可能需要查看 Redis 服务，先列出 Service", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace, "kind": "Service"})
		}
		return steps
	}

	switch strings.ToLower(kind) {
	case "pod":
		add("pod.inspect", "The question targets a specific pod", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace, "name": name})
	case "node":
		add("node.inspect", "The question targets a specific node", domain.JSONMap{"cluster_id": request.ClusterID, "name": name})
	case "deployment":
		add("deployment.inspect", "The question targets a specific deployment", domain.JSONMap{"cluster_id": request.ClusterID, "namespace": namespace, "name": name})
	default:
		add("resource.inspect", "The question targets a specific Kubernetes resource", domain.JSONMap{"cluster_id": request.ClusterID, "kind": kind, "namespace": namespace, "name": name})
	}
	if NeedsResourceEvents(query) {
		add("resource.events", "The user asked for warning reasons or event evidence on the current target resource", domain.JSONMap{"cluster_id": request.ClusterID, "kind": kind, "namespace": namespace, "name": name})
	}
	if NeedsResourceLogs(kind, query) {
		add("resource.logs", "The user asked for logs or crash evidence on the current target scope", domain.JSONMap{"cluster_id": request.ClusterID, "kind": kind, "namespace": namespace, "name": name, "tail_lines": 80})
	}
	if NeedsRelatedResources(query) {
		add("resource.related", "The user asked for ownership, dependency, or related resource context", domain.JSONMap{"cluster_id": request.ClusterID, "kind": kind, "namespace": namespace, "name": name})
	}
	if yamlIntent {
		yamlToolName := "resource.yaml"
		if strings.EqualFold(kind, "secret") {
			yamlToolName = "resource.masked_yaml"
		}
		add(yamlToolName, "The user explicitly asked for YAML or manifest details", domain.JSONMap{"cluster_id": request.ClusterID, "kind": kind, "namespace": namespace, "name": name})
	}
	return steps
}

func ExplicitlyBroadensResourceScope(query string) bool {
	text := strings.ToLower(strings.TrimSpace(query))
	return text != "" && (containsAutoDiagnosticPlanAny(text, "all workloads", "all deployments", "namespace-wide", "entire namespace", "whole namespace", "other deployments", "compare with", "compare to", "sibling resources") || containsAutoDiagnosticPlanAny(query, "全部工作负载", "所有工作负载", "全部 deployment", "所有 deployment", "整个命名空间", "命名空间内全部", "其他 deployment", "其他资源", "对比", "比较"))
}

func NeedsPodOwnershipContext(kind, query string) bool {
	if !strings.EqualFold(strings.TrimSpace(kind), "pod") {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(query))
	return containsAutoDiagnosticPlanAny(text, "owner", "controller", "workload", "deployment", "statefulset", "daemonset", "replicaset", "managed by", "belongs to") || containsAutoDiagnosticPlanAny(query, "归属", "属于", "谁创建", "谁管理", "控制器", "工作负载", "deployment", "statefulset", "daemonset", "replicaset")
}

func NeedsResourceEvents(query string) bool {
	return containsAutoDiagnosticPlanAny(strings.ToLower(strings.TrimSpace(query)), "event", "events", "warning", "warnings", "reason") || containsAutoDiagnosticPlanAny(query, "事件", "告警", "警告", "原因")
}

func NeedsResourceLogs(kind, query string) bool {
	text := strings.ToLower(strings.TrimSpace(query))
	if !containsAutoDiagnosticPlanAny(text, "log", "logs", "crash", "error", "exception") && !containsAutoDiagnosticPlanAny(query, "日志", "报错", "错误", "异常", "崩溃") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "pod", "deployment", "statefulset", "daemonset", "replicaset", "job", "cronjob":
		return true
	default:
		return false
	}
}

func NeedsRelatedResources(query string) bool {
	text := strings.ToLower(strings.TrimSpace(query))
	return containsAutoDiagnosticPlanAny(text, "related", "dependency", "dependencies", "owner", "controller", "managed by", "references") || containsAutoDiagnosticPlanAny(query, "关联", "依赖", "归属", "属于", "控制器", "引用")
}

func containsAutoDiagnosticPlanAny(text string, candidates ...string) bool {
	for _, candidate := range candidates {
		if candidate != "" && strings.Contains(text, candidate) {
			return true
		}
	}
	return false
}

func cloneDiagnosticPlanJSON(value domain.JSONMap) domain.JSONMap {
	if len(value) == 0 {
		return domain.JSONMap{}
	}
	cloned := make(domain.JSONMap, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}
