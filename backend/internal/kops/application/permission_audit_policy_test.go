package application

import "testing"

func TestPermissionAuditScanTargetsNormalizesAndFallsBack(t *testing.T) {
	full := PermissionAuditScanTargets(nil)
	if len(full) < 20 {
		t.Fatalf("full scan target count = %d, want complete baseline", len(full))
	}
	filtered := PermissionAuditScanTargets([]string{" Deployment ", "PDB", "unknown"})
	if len(filtered) != 2 || filtered[0].Resource != "deployments" || filtered[1].Resource != "poddisruptionbudgets" {
		t.Fatalf("filtered targets = %#v", filtered)
	}
	if got := len(PermissionAuditScanTargets([]string{"unknown"})); got != len(full) {
		t.Fatalf("unknown allowlist target count = %d, want %d", got, len(full))
	}
}

func TestNormalizePermissionAuditAllowlistItem(t *testing.T) {
	for input, want := range map[string]string{
		"deployment": "deployments", "persistent-volume-claim": "persistentvolumeclaims", "CRDs": "customresourcedefinitions", "": "", "unknown": "",
	} {
		if got := NormalizePermissionAuditAllowlistItem(input); got != want {
			t.Fatalf("NormalizePermissionAuditAllowlistItem(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizePermissionAuditSort(t *testing.T) {
	if field, order := NormalizePermissionAuditListSort(" UPDATED_AT ", "ASC"); field != "updated_at" || order != "asc" {
		t.Fatalf("list sort = (%q, %q)", field, order)
	}
	if field, order := NormalizePermissionAuditFindingSort("name; drop table", "ascending"); field != "" || order != "desc" {
		t.Fatalf("finding sort = (%q, %q)", field, order)
	}
}

func TestPermissionAuditOwnershipAndRiskPolicy(t *testing.T) {
	targets := NewPermissionAuditNamespaceSet([]string{"platform"})
	if got := PermissionAuditWorkloadOwnership("platform", targets); got != PermissionAuditOwnershipDirect {
		t.Fatalf("workload ownership = %q", got)
	}
	meta := PermissionAuditResourceMeta{Kind: "StorageClass", Name: "fast", Namespaced: false}
	if got := PermissionAuditResourceOwnership(meta, targets, nil, map[string]bool{"fast": true}, nil); got != PermissionAuditOwnershipShared {
		t.Fatalf("storage class ownership = %q", got)
	}
	blocker := PermissionAuditResourceMeta{Kind: "ClusterRole", Name: "ops", Namespaced: false}
	if got := PermissionAuditResourceRiskLevel(blocker, PermissionAuditOwnershipDirect, true); got != "critical" {
		t.Fatalf("blocker risk = %q", got)
	}
	if got := PermissionAuditResourcePrivilegeClass(blocker, PermissionAuditOwnershipDirect, true); got != PermissionAuditPrivilegeClusterScoped {
		t.Fatalf("blocker privilege = %q", got)
	}
	codes := PermissionAuditResourceReasonCodes(blocker, PermissionAuditOwnershipDirect, true)
	if len(codes) != 3 || codes[0] != "cluster_scoped_resource" || codes[2] != "deployment_blocker" {
		t.Fatalf("reason codes = %#v", codes)
	}
	if got := PermissionAuditWorkloadRiskLevel(true, true, false, PermissionAuditOwnershipDirect); got != "critical" {
		t.Fatalf("workload risk = %q", got)
	}
	if !PermissionAuditIsClusterScopedResource("nodes") || PermissionAuditIsClusterScopedResource("pods") {
		t.Fatal("cluster scoped resource policy mismatch")
	}
	if !PermissionAuditIsHighRiskVerb("delete") || PermissionAuditIsHighRiskVerb("list") {
		t.Fatal("high risk verb policy mismatch")
	}
	if got := PermissionAuditResourceSummary("ClusterRole", PermissionAuditOwnershipDirect, "cluster", true); got == "" {
		t.Fatal("resource summary should not be empty")
	}
	if got := PermissionAuditWorkloadSummary("Deployment", "api", "runtime", PermissionAuditPrivilegeRuntimeHigh); got == "" {
		t.Fatal("workload summary should not be empty")
	}
	if got := PermissionAuditGVRKey("apps", "v1", "deployments"); got != "apps|v1|deployments" {
		t.Fatalf("GVR key = %q", got)
	}
	if got := PermissionAuditRoleKey("Role", "ops", "reader"); got != "role|ops|reader" {
		t.Fatalf("role key = %q", got)
	}
	if got := PermissionAuditStringSlice([]any{"watch", "get", ""}); len(got) != 2 || got[0] != "get" || got[1] != "watch" {
		t.Fatalf("rule fields = %#v", got)
	}
}
