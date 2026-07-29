package application

import (
	"fmt"
	"sort"
	"strings"
)

// PermissionAuditScanTarget is a Kubernetes-independent scan descriptor.
// The runtime converts it to a client-go GroupVersionResource at its boundary.
type PermissionAuditScanTarget struct {
	Group    string
	Version  string
	Resource string
}

var defaultPermissionAuditScanTargets = []PermissionAuditScanTarget{
	{Group: "apps", Version: "v1", Resource: "deployments"},
	{Group: "apps", Version: "v1", Resource: "statefulsets"},
	{Group: "apps", Version: "v1", Resource: "daemonsets"},
	{Version: "v1", Resource: "pods"},
	{Version: "v1", Resource: "services"},
	{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"},
	{Version: "v1", Resource: "configmaps"},
	{Version: "v1", Resource: "secrets"},
	{Version: "v1", Resource: "serviceaccounts"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
	{Version: "v1", Resource: "persistentvolumes"},
	{Version: "v1", Resource: "persistentvolumeclaims"},
	{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
	{Group: "batch", Version: "v1", Resource: "jobs"},
	{Group: "batch", Version: "v1", Resource: "cronjobs"},
	{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"},
	{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
	{Version: "v1", Resource: "namespaces"},
	{Version: "v1", Resource: "nodes"},
	{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},
	{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"},
	{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"},
	{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"},
	{Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"},
}

// PermissionAuditScanTargets filters the stable full scan list. An empty or
// unrecognised allowlist intentionally falls back to the complete scan.
func PermissionAuditScanTargets(allowlist []string) []PermissionAuditScanTarget {
	allowed := make(map[string]bool, len(allowlist))
	for _, item := range allowlist {
		if normalized := NormalizePermissionAuditAllowlistItem(item); normalized != "" {
			allowed[normalized] = true
		}
	}
	if len(allowed) == 0 {
		return append([]PermissionAuditScanTarget(nil), defaultPermissionAuditScanTargets...)
	}

	filtered := make([]PermissionAuditScanTarget, 0, len(defaultPermissionAuditScanTargets))
	for _, target := range defaultPermissionAuditScanTargets {
		if allowed[target.Resource] {
			filtered = append(filtered, target)
		}
	}
	if len(filtered) == 0 {
		return append([]PermissionAuditScanTarget(nil), defaultPermissionAuditScanTargets...)
	}
	return filtered
}

// NormalizePermissionAuditAllowlistItem resolves common singular, plural and
// abbreviated resource names to the canonical Kubernetes resource name.
func NormalizePermissionAuditAllowlistItem(value string) string {
	value = strings.NewReplacer("_", "", "-", "", ".", "", "/", "").Replace(strings.ToLower(strings.TrimSpace(value)))
	switch value {
	case "deployment", "deployments":
		return "deployments"
	case "statefulset", "statefulsets":
		return "statefulsets"
	case "daemonset", "daemonsets":
		return "daemonsets"
	case "pod", "pods":
		return "pods"
	case "service", "services":
		return "services"
	case "ingress", "ingresses":
		return "ingresses"
	case "ingressclass", "ingressclasses":
		return "ingressclasses"
	case "configmap", "configmaps":
		return "configmaps"
	case "secret", "secrets":
		return "secrets"
	case "serviceaccount", "serviceaccounts":
		return "serviceaccounts"
	case "role", "roles":
		return "roles"
	case "rolebinding", "rolebindings":
		return "rolebindings"
	case "clusterrole", "clusterroles":
		return "clusterroles"
	case "clusterrolebinding", "clusterrolebindings":
		return "clusterrolebindings"
	case "persistentvolume", "persistentvolumes", "pv", "pvs":
		return "persistentvolumes"
	case "persistentvolumeclaim", "persistentvolumeclaims", "pvc", "pvcs":
		return "persistentvolumeclaims"
	case "storageclass", "storageclasses":
		return "storageclasses"
	case "job", "jobs":
		return "jobs"
	case "cronjob", "cronjobs":
		return "cronjobs"
	case "poddisruptionbudget", "poddisruptionbudgets", "pdb", "pdbs":
		return "poddisruptionbudgets"
	case "horizontalpodautoscaler", "horizontalpodautoscalers", "hpa", "hpas":
		return "horizontalpodautoscalers"
	case "namespace", "namespaces":
		return "namespaces"
	case "node", "nodes":
		return "nodes"
	case "customresourcedefinition", "customresourcedefinitions", "crd", "crds":
		return "customresourcedefinitions"
	case "validatingwebhookconfiguration", "validatingwebhookconfigurations":
		return "validatingwebhookconfigurations"
	case "mutatingwebhookconfiguration", "mutatingwebhookconfigurations":
		return "mutatingwebhookconfigurations"
	case "apiservice", "apiservices":
		return "apiservices"
	case "priorityclass", "priorityclasses":
		return "priorityclasses"
	default:
		return ""
	}
}

const (
	PermissionAuditOwnershipDirect    = "direct"
	PermissionAuditOwnershipShared    = "shared"
	PermissionAuditOwnershipUnrelated = "unrelated"

	PermissionAuditPrivilegeClusterScoped = "cluster_scoped"
	PermissionAuditPrivilegeRuntimeHigh   = "runtime_high"
	PermissionAuditPrivilegeNamespaceOnly = "namespace_only_candidate"
	PermissionAuditPrivilegeSharedCluster = "shared_cluster_dependency"
)

// PermissionAuditResourceMeta carries only the object facts needed for audit
// policy decisions; Kubernetes unstructured objects remain in the runtime.
type PermissionAuditResourceMeta struct {
	Kind       string
	Name       string
	Namespace  string
	Namespaced bool
}

// NewPermissionAuditNamespaceSet normalizes the explicitly scoped namespace
// set. An empty set means all namespaced resources are in scope.
func NewPermissionAuditNamespaceSet(namespaces []string) map[string]bool {
	result := make(map[string]bool, len(namespaces))
	for _, namespace := range namespaces {
		if namespace = strings.TrimSpace(namespace); namespace != "" {
			result[namespace] = true
		}
	}
	return result
}

func PermissionAuditNamespaceInScope(namespace string, targetNamespaces map[string]bool) bool {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return false
	}
	if len(targetNamespaces) == 0 {
		return true
	}
	return targetNamespaces[namespace]
}

func PermissionAuditWorkloadOwnership(namespace string, targetNamespaces map[string]bool) string {
	if PermissionAuditNamespaceInScope(namespace, targetNamespaces) {
		return PermissionAuditOwnershipDirect
	}
	return PermissionAuditOwnershipUnrelated
}

// PermissionAuditResourceOwnership determines ownership from scan scope and
// resource references collected by the runtime.
func PermissionAuditResourceOwnership(meta PermissionAuditResourceMeta, targetNamespaces, serviceAccounts, storageClasses, ingressClasses map[string]bool) string {
	meta.Kind = strings.TrimSpace(meta.Kind)
	meta.Name = strings.TrimSpace(meta.Name)
	meta.Namespace = strings.TrimSpace(meta.Namespace)
	if meta.Namespaced && PermissionAuditNamespaceInScope(meta.Namespace, targetNamespaces) {
		return PermissionAuditOwnershipDirect
	}
	switch meta.Kind {
	case "Namespace":
		if len(targetNamespaces) > 0 && targetNamespaces[meta.Name] {
			return PermissionAuditOwnershipDirect
		}
	case "StorageClass":
		if storageClasses[meta.Name] {
			return PermissionAuditOwnershipShared
		}
	case "IngressClass":
		if ingressClasses[meta.Name] {
			return PermissionAuditOwnershipShared
		}
	}
	if serviceAccounts[PermissionAuditSubjectKey(meta.Namespace, meta.Name)] {
		return PermissionAuditOwnershipShared
	}
	return PermissionAuditOwnershipUnrelated
}

func PermissionAuditResourceRiskLevel(meta PermissionAuditResourceMeta, ownership string, deploymentBlocker bool) string {
	if deploymentBlocker && isPermissionAuditDeploymentBlockerKind(meta.Kind) {
		return "critical"
	}
	if !meta.Namespaced && ownership != PermissionAuditOwnershipUnrelated {
		return "high"
	}
	if meta.Kind == "Secret" && ownership != PermissionAuditOwnershipUnrelated {
		return "medium"
	}
	return "low"
}

func PermissionAuditWorkloadRiskLevel(runtimeHigh, hasRBACWrite, hasSecretWrite bool, ownership string) string {
	if runtimeHigh && hasRBACWrite {
		return "critical"
	}
	if runtimeHigh {
		return "high"
	}
	if hasSecretWrite || ownership == PermissionAuditOwnershipShared {
		return "medium"
	}
	return "low"
}

func PermissionAuditResourcePrivilegeClass(meta PermissionAuditResourceMeta, ownership string, deploymentBlocker bool) string {
	if deploymentBlocker {
		return PermissionAuditPrivilegeClusterScoped
	}
	if ownership == PermissionAuditOwnershipShared && !meta.Namespaced {
		return PermissionAuditPrivilegeSharedCluster
	}
	if meta.Namespaced {
		return PermissionAuditPrivilegeNamespaceOnly
	}
	return PermissionAuditPrivilegeClusterScoped
}

func PermissionAuditResourceReasonCodes(meta PermissionAuditResourceMeta, ownership string, deploymentBlocker bool) []string {
	codes := make([]string, 0, 4)
	if !meta.Namespaced {
		codes = append(codes, "cluster_scoped_resource")
	}
	if ownership == PermissionAuditOwnershipDirect {
		codes = append(codes, "audit_scope_direct")
	}
	if ownership == PermissionAuditOwnershipShared {
		codes = append(codes, "shared_cluster_capability")
	}
	if deploymentBlocker {
		codes = append(codes, "deployment_blocker")
	}
	if len(codes) == 0 {
		return []string{"resource_observed"}
	}
	return codes
}

func PermissionAuditRiskRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

// PermissionAuditResourceSummary preserves the user-facing ownership wording
// alongside the policy which determines those ownership values.
func PermissionAuditResourceSummary(kind, ownership, scope string, deploymentBlocker bool) string {
	if deploymentBlocker {
		return fmt.Sprintf("%s 为平台直接使用的 %s 级资源，当前属于部署阻塞项", kind, scope)
	}
	if ownership == PermissionAuditOwnershipShared {
		return fmt.Sprintf("%s 为平台依赖的共享集群能力", kind)
	}
	if ownership == PermissionAuditOwnershipDirect {
		return fmt.Sprintf("%s 已归属到平台资源清单", kind)
	}
	return fmt.Sprintf("%s 已扫描，但未确认归属到平台", kind)
}

func PermissionAuditWorkloadSummary(kind, name, serviceAccountName, privilegeClass string) string {
	switch privilegeClass {
	case PermissionAuditPrivilegeRuntimeHigh:
		return fmt.Sprintf("%s %s 绑定 ServiceAccount %s，存在运行期高权限依赖", kind, name, serviceAccountName)
	case PermissionAuditPrivilegeSharedCluster:
		return fmt.Sprintf("%s %s 依赖共享集群能力或共享 ServiceAccount %s", kind, name, serviceAccountName)
	default:
		return fmt.Sprintf("%s %s 当前可作为命名空间权限部署候选，ServiceAccount=%s", kind, name, serviceAccountName)
	}
}

func PermissionAuditIsClusterScopedResource(resource string) bool {
	return permissionAuditClusterScopedResources[strings.ToLower(strings.TrimSpace(resource))]
}

func PermissionAuditIsHighRiskVerb(verb string) bool {
	return permissionAuditHighRiskVerbs[strings.ToLower(strings.TrimSpace(verb))]
}

func PermissionAuditSubjectKey(namespace, name string) string {
	return strings.ToLower(strings.TrimSpace(namespace) + "|" + strings.TrimSpace(name))
}

func PermissionAuditGVRKey(group, version, resource string) string {
	return strings.ToLower(strings.Join([]string{strings.TrimSpace(group), strings.TrimSpace(version), strings.TrimSpace(resource)}, "|"))
}

func PermissionAuditRoleKey(kind, namespace, name string) string {
	return strings.ToLower(strings.Join([]string{strings.TrimSpace(kind), strings.TrimSpace(namespace), strings.TrimSpace(name)}, "|"))
}

// PermissionAuditStringSlice normalizes unstructured RBAC rule fields. It
// preserves already-typed string slices for compatibility and sorts values
// read from generic JSON arrays to make scan output deterministic.
func PermissionAuditStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]string); ok {
			return typed
		}
		return []string{}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(fmt.Sprintf("%v", item)); text != "" {
			result = append(result, text)
		}
	}
	sort.Strings(result)
	return result
}

func isPermissionAuditDeploymentBlockerKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "ClusterRole", "ClusterRoleBinding", "CustomResourceDefinition", "ValidatingWebhookConfiguration", "MutatingWebhookConfiguration":
		return true
	default:
		return false
	}
}

var permissionAuditClusterScopedResources = map[string]bool{
	"nodes": true, "namespaces": true, "persistentvolumes": true, "storageclasses": true,
	"clusterroles": true, "clusterrolebindings": true, "customresourcedefinitions": true,
	"validatingwebhookconfigurations": true, "mutatingwebhookconfigurations": true,
	"apiservices": true, "priorityclasses": true, "ingressclasses": true,
}

var permissionAuditHighRiskVerbs = map[string]bool{
	"create": true, "update": true, "patch": true, "delete": true, "deletecollection": true,
	"bind": true, "escalate": true, "impersonate": true,
}

// NormalizePermissionAuditListSort removes fields which are not safe audit
// list columns and canonicalizes the sort direction.
func NormalizePermissionAuditListSort(sortBy, order string) (string, string) {
	return normalizePermissionAuditSort(sortBy, order, map[string]bool{
		"id": true, "display_name": true, "status": true, "created_at": true, "updated_at": true,
	})
}

// NormalizePermissionAuditFindingSort removes fields which are not safe
// finding list columns and canonicalizes the sort direction.
func NormalizePermissionAuditFindingSort(sortBy, order string) (string, string) {
	return normalizePermissionAuditSort(sortBy, order, map[string]bool{
		"id": true, "risk_level": true, "kind": true, "namespace": true, "name": true, "created_at": true,
	})
}

func normalizePermissionAuditSort(sortBy, order string, allowed map[string]bool) (string, string) {
	sortBy = strings.ToLower(strings.TrimSpace(sortBy))
	if !allowed[sortBy] {
		sortBy = ""
	}
	if strings.EqualFold(strings.TrimSpace(order), "asc") {
		return sortBy, "asc"
	}
	return sortBy, "desc"
}
