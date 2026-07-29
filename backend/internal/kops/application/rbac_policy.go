package application

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RBACMatrixRow describes one resource-group permission row supplied to the
// RBAC manifest generator.
type RBACMatrixRow struct {
	APIGroup  string   `json:"api_group"`
	Resources []string `json:"resources"`
	Verbs     []string `json:"verbs"`
	Scope     string   `json:"scope"`
	Label     string   `json:"label"`
}

// RBACMatrixRequest is the policy-level input for a platform access account.
type RBACMatrixRequest struct {
	ServiceAccount   string          `json:"service_account"`
	SANamespace      string          `json:"sa_namespace"`
	TargetNamespaces []string        `json:"target_namespaces"`
	ClusterRows      []RBACMatrixRow `json:"cluster_rows"`
	NamespaceRows    []RBACMatrixRow `json:"namespace_rows"`
}

// DefaultRBACMatrix returns the least-privilege baseline required by the
// platform's Kops resource operations. It contains no persistence or HTTP
// dependencies and can therefore be safely reused by all entry points.
func DefaultRBACMatrix(namespaces []string) RBACMatrixRequest {
	return RBACMatrixRequest{
		ServiceAccount:   "xingku-platform",
		SANamespace:      "kube-system",
		TargetNamespaces: normalizeNamespaces(namespaces),
		ClusterRows: []RBACMatrixRow{
			{Resources: []string{"nodes"}, Verbs: []string{"get", "list", "watch", "update", "patch", "delete"}, Scope: "cluster", Label: "nodes"},
			{Resources: []string{"nodes/status"}, Verbs: []string{"get", "list", "watch"}, Scope: "cluster", Label: "node status"},
			{APIGroup: "metrics.k8s.io", Resources: []string{"nodes", "pods"}, Verbs: []string{"get", "list"}, Scope: "cluster", Label: "metrics"},
			{Resources: []string{"namespaces"}, Verbs: []string{"get", "list", "watch", "create", "delete"}, Scope: "cluster", Label: "namespaces"},
			{Resources: []string{"persistentvolumes"}, Verbs: []string{"get", "list", "watch", "delete"}, Scope: "cluster", Label: "persistent volumes"},
			{APIGroup: "storage.k8s.io", Resources: []string{"storageclasses"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "cluster", Label: "storage classes"},
			{APIGroup: "networking.k8s.io", Resources: []string{"ingressclasses"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "cluster", Label: "ingress classes"},
			{APIGroup: "rbac.authorization.k8s.io", Resources: []string{"clusterroles"}, Verbs: []string{"get", "list", "watch"}, Scope: "cluster", Label: "cluster roles"},
			{APIGroup: "rbac.authorization.k8s.io", Resources: []string{"clusterrolebindings"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "cluster", Label: "cluster role bindings"},
			{Resources: []string{"events"}, Verbs: []string{"get", "list", "watch"}, Scope: "cluster", Label: "events"},
		},
		NamespaceRows: []RBACMatrixRow{
			{APIGroup: "apps", Resources: []string{"deployments", "statefulsets", "daemonsets", "replicasets"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "workloads"},
			{APIGroup: "apps", Resources: []string{"deployments/scale", "statefulsets/scale"}, Verbs: []string{"get", "update", "patch"}, Scope: "namespace", Label: "workload scale"},
			{APIGroup: "batch", Resources: []string{"jobs", "cronjobs"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "batch workloads"},
			{APIGroup: "autoscaling", Resources: []string{"horizontalpodautoscalers"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "HPA"},
			{Resources: []string{"pods"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "pods"},
			{Resources: []string{"pods/log"}, Verbs: []string{"get"}, Scope: "namespace", Label: "pod logs"},
			{Resources: []string{"pods/exec", "pods/eviction"}, Verbs: []string{"create"}, Scope: "namespace", Label: "pod runtime operations"},
			{Resources: []string{"services", "endpoints"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "services"},
			{APIGroup: "networking.k8s.io", Resources: []string{"ingresses"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "ingresses"},
			{Resources: []string{"configmaps", "secrets", "persistentvolumeclaims"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "configuration and storage"},
			{Resources: []string{"serviceaccounts"}, Verbs: []string{"get", "list", "watch"}, Scope: "namespace", Label: "service accounts"},
			{APIGroup: "rbac.authorization.k8s.io", Resources: []string{"roles", "rolebindings"}, Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}, Scope: "namespace", Label: "namespace RBAC"},
			{Resources: []string{"events"}, Verbs: []string{"get", "list", "watch"}, Scope: "namespace", Label: "events"},
		},
	}
}

// FilterRBACMatrixByResources narrows a baseline according to resources that
// were actually included in an audit. An empty or unknown allowlist leaves the
// baseline intact, preserving a usable recommendation for incomplete audits.
func FilterRBACMatrixByResources(matrix RBACMatrixRequest, allowlist []string) RBACMatrixRequest {
	allowed := make(map[string]bool, len(allowlist))
	for _, value := range allowlist {
		if normalized := normalizeResource(value); normalized != "" {
			allowed[normalized] = true
		}
	}
	if len(allowed) == 0 {
		return matrix
	}
	matrix.ClusterRows = filterRBACRows(matrix.ClusterRows, allowed)
	matrix.NamespaceRows = filterRBACRows(matrix.NamespaceRows, allowed)
	return matrix
}

// BuildRBACFromMatrix renders a complete ServiceAccount, roles/bindings and
// token-secret manifest. JSON string quoting gives valid YAML scalars for all
// API groups, resources and verbs supplied by the caller.
func BuildRBACFromMatrix(request RBACMatrixRequest) string {
	serviceAccount := firstNonEmpty(strings.TrimSpace(request.ServiceAccount), "xingku-platform")
	serviceAccountNamespace := firstNonEmpty(strings.TrimSpace(request.SANamespace), "kube-system")
	namespaces := normalizeNamespaces(request.TargetNamespaces)
	clusterRows := validRBACRows(request.ClusterRows)
	namespaceRows := validRBACRows(request.NamespaceRows)

	var builder strings.Builder
	fmt.Fprintf(&builder, "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: %s\n  namespace: %s\n  labels:\n    app.kubernetes.io/name: %s\n    app.kubernetes.io/component: rbac\n", yamlScalar(serviceAccount), yamlScalar(serviceAccountNamespace), yamlScalar(serviceAccount))
	if len(clusterRows) > 0 {
		fmt.Fprintf(&builder, "\n---\napiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  name: %s\nrules:\n", yamlScalar(serviceAccount+"-cluster-ops"))
		writeRBACRules(&builder, clusterRows)
		builder.WriteString("  - nonResourceURLs: [\"/api\", \"/api/*\", \"/apis\", \"/apis/*\", \"/version\", \"/healthz\"]\n    verbs: [\"get\"]\n")
		fmt.Fprintf(&builder, "\n---\napiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRoleBinding\nmetadata:\n  name: %s\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: ClusterRole\n  name: %s\nsubjects:\n  - kind: ServiceAccount\n    name: %s\n    namespace: %s\n", yamlScalar(serviceAccount+"-cluster-ops"), yamlScalar(serviceAccount+"-cluster-ops"), yamlScalar(serviceAccount), yamlScalar(serviceAccountNamespace))
	}
	if len(namespaceRows) > 0 {
		fmt.Fprintf(&builder, "\n---\napiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  name: %s\nrules:\n", yamlScalar(serviceAccount+"-ns-ops"))
		writeRBACRules(&builder, namespaceRows)
		for _, namespace := range namespaces {
			fmt.Fprintf(&builder, "\n---\napiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: %s\n  namespace: %s\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: ClusterRole\n  name: %s\nsubjects:\n  - kind: ServiceAccount\n    name: %s\n    namespace: %s\n", yamlScalar(serviceAccount+"-ns-ops"), yamlScalar(namespace), yamlScalar(serviceAccount+"-ns-ops"), yamlScalar(serviceAccount), yamlScalar(serviceAccountNamespace))
		}
	}
	fmt.Fprintf(&builder, "\n---\napiVersion: v1\nkind: Secret\nmetadata:\n  name: %s\n  namespace: %s\n  annotations:\n    kubernetes.io/service-account.name: %s\ntype: kubernetes.io/service-account-token\n", yamlScalar(serviceAccount+"-token"), yamlScalar(serviceAccountNamespace), yamlScalar(serviceAccount))
	return builder.String()
}

func writeRBACRules(builder *strings.Builder, rows []RBACMatrixRow) {
	for _, row := range rows {
		fmt.Fprintf(builder, "  - apiGroups: [%s]\n    resources: %s\n    verbs: %s\n", yamlScalar(strings.TrimSpace(row.APIGroup)), yamlStringList(row.Resources), yamlStringList(row.Verbs))
	}
}

func validRBACRows(rows []RBACMatrixRow) []RBACMatrixRow {
	valid := make([]RBACMatrixRow, 0, len(rows))
	for _, row := range rows {
		resources, verbs := compactStrings(row.Resources), compactStrings(row.Verbs)
		if len(resources) == 0 || len(verbs) == 0 {
			continue
		}
		row.APIGroup, row.Resources, row.Verbs = strings.TrimSpace(row.APIGroup), resources, verbs
		valid = append(valid, row)
	}
	return valid
}

func filterRBACRows(rows []RBACMatrixRow, allowed map[string]bool) []RBACMatrixRow {
	filtered := make([]RBACMatrixRow, 0, len(rows))
	for _, row := range rows {
		for _, resource := range row.Resources {
			if allowed[normalizeResource(strings.SplitN(resource, "/", 2)[0])] {
				filtered = append(filtered, row)
				break
			}
		}
	}
	return filtered
}

func normalizeNamespaces(values []string) []string {
	values = compactStrings(values)
	if len(values) == 0 {
		return []string{"default"}
	}
	return values
}

func compactStrings(values []string) []string {
	result, seen := make([]string, 0, len(values)), map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func yamlScalar(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func yamlStringList(values []string) string {
	encoded, _ := json.Marshal(values)
	return string(encoded)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeResource(value string) string {
	value = strings.NewReplacer("_", "", "-", "", ".", "", "/", "").Replace(strings.ToLower(strings.TrimSpace(value)))
	singular := map[string]string{
		"deployment": "deployments", "statefulset": "statefulsets", "daemonset": "daemonsets", "pod": "pods", "service": "services", "ingress": "ingresses", "ingressclass": "ingressclasses", "configmap": "configmaps", "secret": "secrets", "serviceaccount": "serviceaccounts", "role": "roles", "rolebinding": "rolebindings", "clusterrole": "clusterroles", "clusterrolebinding": "clusterrolebindings", "persistentvolume": "persistentvolumes", "pv": "persistentvolumes", "persistentvolumeclaim": "persistentvolumeclaims", "pvc": "persistentvolumeclaims", "storageclass": "storageclasses", "job": "jobs", "cronjob": "cronjobs", "poddisruptionbudget": "poddisruptionbudgets", "pdb": "poddisruptionbudgets", "horizontalpodautoscaler": "horizontalpodautoscalers", "hpa": "horizontalpodautoscalers", "namespace": "namespaces", "node": "nodes", "customresourcedefinition": "customresourcedefinitions", "crd": "customresourcedefinitions", "validatingwebhookconfiguration": "validatingwebhookconfigurations", "mutatingwebhookconfiguration": "mutatingwebhookconfigurations", "apiservice": "apiservices", "priorityclass": "priorityclasses",
	}
	if normalized, ok := singular[value]; ok {
		return normalized
	}
	return value
}
