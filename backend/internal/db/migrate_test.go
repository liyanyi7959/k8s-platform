package db

import "testing"

func TestSplitSQLStatementsKeepsSemicolonsInsideStrings(t *testing.T) {
	sql := []byte(`
-- comment line should be ignored
UPDATE deploy_configs
SET command_template = 'echo "before"; echo "after"'
WHERE step_key = 'pre_check';

INSERT INTO deploy_configs(step_key, os_type, step_name, step_order, command_template, description, enabled, timeout_seconds, retry_count, created_by)
VALUES ('join_workers', 'centos', 'Worker 节点加入', 4, 'sudo kubeadm join --token abc; echo done', 'desc', 1, 300, 0, 0);
`)

	parts := splitSQLStatements(sql)
	if len(parts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %#v", len(parts), parts)
	}

	if want := `SET command_template = 'echo "before"; echo "after"'`; !contains(parts[0], want) {
		t.Fatalf("first statement lost embedded semicolon: %s", parts[0])
	}
	if want := `'sudo kubeadm join --token abc; echo done'`; !contains(parts[1], want) {
		t.Fatalf("second statement lost embedded semicolon: %s", parts[1])
	}
}

func contains(text, sub string) bool {
	return len(sub) == 0 || (len(text) >= len(sub) && stringIndex(text, sub) >= 0)
}

func stringIndex(text, sub string) int {
	for i := 0; i+len(sub) <= len(text); i++ {
		if text[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
