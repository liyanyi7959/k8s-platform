package domain

import "testing"

func TestNewNamespaceScope(t *testing.T) {
	scope, err := NewNamespaceScope(1, []string{" default ", "kube-system", "default"}, []string{"namespace:read"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope == nil {
		t.Fatal("scope should not be nil")
	}
	if scope.ClusterID != 1 {
		t.Fatalf("ClusterID = %d, want 1", scope.ClusterID)
	}
	if len(scope.Namespaces) != 2 || scope.Namespaces[0] != "default" || scope.Namespaces[1] != "kube-system" {
		t.Fatalf("Namespaces = %v, want [default kube-system]", scope.Namespaces)
	}
}

func TestNewNamespaceScopeRejectsMissingCluster(t *testing.T) {
	if _, err := NewNamespaceScope(0, []string{"default"}, []string{"namespace:read"}); err != ErrInvalidParams {
		t.Fatalf("err = %v, want ErrInvalidParams", err)
	}
}

func TestNewNamespaceScopeRejectsWithoutNamespacePermission(t *testing.T) {
	if _, err := NewNamespaceScope(1, []string{"default"}, []string{"k8s:read"}); err != ErrInvalidParams {
		t.Fatalf("err = %v, want ErrInvalidParams", err)
	}
}

func TestNewNamespaceScopeEmptyNamespaces(t *testing.T) {
	scope, err := NewNamespaceScope(1, []string{"", " "}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope != nil {
		t.Fatalf("scope = %+v, want nil", scope)
	}
}
