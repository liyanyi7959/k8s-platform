package domain

import "testing"

func TestPermissionAuditPersistenceValueObjects(t *testing.T) {
	value, err := JSONMap{"mode": "full"}.Value()
	if err != nil || string(value.([]byte)) != `{"mode":"full"}` {
		t.Fatalf("JSONMap value = %v, %v", value, err)
	}
	var reasons JSONStringSlice
	if err := reasons.Scan([]byte(`["rbac_write"]`)); err != nil || len(reasons) != 1 || reasons[0] != "rbac_write" {
		t.Fatalf("JSONStringSlice scan = %#v, %v", reasons, err)
	}
	if (K8sPermissionAudit{}).TableName() != "k8s_permission_audits" || (K8sPermissionAuditFinding{}).TableName() != "k8s_permission_audit_findings" {
		t.Fatal("permission audit table mapping changed")
	}
}
