package domain

import "testing"

func TestManifestApplyRecordTableName(t *testing.T) {
	if got := (ManifestApplyRecord{}).TableName(); got != "manifest_apply_records" {
		t.Fatalf("table name = %q", got)
	}
}
