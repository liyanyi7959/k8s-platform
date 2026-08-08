package domain

import "testing"

func TestNewNamespaceAssignmentNormalizesAndDeduplicates(t *testing.T) {
	assignment := NewNamespaceAssignment([]string{"  default ", "kube-system", "", "default", "monitoring "})
	got := assignment.Values()
	want := []string{"default", "kube-system", "monitoring"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if assignment.String() != "default,kube-system,monitoring" {
		t.Fatalf("String() = %q, want %q", assignment.String(), "default,kube-system,monitoring")
	}
}

func TestNamespaceAssignmentContains(t *testing.T) {
	assignment := NewNamespaceAssignment([]string{"default", "kube-system"})
	if !assignment.Contains("default") || assignment.Contains("kube-public") {
		t.Fatal("Contains mismatch")
	}
}

func TestProjectAssignNamespacesUsesAggregate(t *testing.T) {
	project := &Project{Name: "demo"}
	project.AssignNamespaces([]string{"ns-a", "ns-a", " ns-b "})
	if project.Namespaces != "ns-a,ns-b" {
		t.Fatalf("Namespaces = %q, want %q", project.Namespaces, "ns-a,ns-b")
	}
}
