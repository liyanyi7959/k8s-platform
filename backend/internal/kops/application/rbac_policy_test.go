package application

import (
	"strings"
	"testing"
)

func TestDefaultRBACMatrixNormalizesNamespaces(t *testing.T) {
	matrix := DefaultRBACMatrix([]string{" dev ", "dev", "", "ops"})
	if got := strings.Join(matrix.TargetNamespaces, ","); got != "dev,ops" {
		t.Fatalf("namespaces = %q", got)
	}
	if len(matrix.ClusterRows) == 0 || len(matrix.NamespaceRows) == 0 {
		t.Fatal("default matrix must contain both scopes")
	}
}

func TestRBACMatrixFilterAndRender(t *testing.T) {
	matrix := FilterRBACMatrixByResources(DefaultRBACMatrix([]string{"dev"}), []string{"pod", "configmaps"})
	manifest := BuildRBACFromMatrix(matrix)
	if !strings.Contains(manifest, "pods/log") || !strings.Contains(manifest, "configmaps") {
		t.Fatalf("rendered manifest misses allowed resource: %s", manifest)
	}
	if strings.Contains(manifest, "clusterrolebindings") || strings.Contains(manifest, "storageclasses") {
		t.Fatalf("rendered manifest retained filtered resource: %s", manifest)
	}
}
