package application

import (
	"context"
	"errors"
	"testing"
)

type namespaceWorkloadRuntimeSpy struct {
	values map[NamespaceWorkloadKind][]map[string]any
}

func (spy *namespaceWorkloadRuntimeSpy) ListNamespaceWorkloads(_ context.Context, _ uint64, kind NamespaceWorkloadKind, _ string) ([]map[string]any, error) {
	return spy.values[kind], nil
}

func TestNamespaceWorkloadInventoryBuildsSortedReplicaSummary(t *testing.T) {
	service := NewNamespaceWorkloadService(&namespaceWorkloadRuntimeSpy{values: map[NamespaceWorkloadKind][]map[string]any{
		NamespaceWorkloadDeployment:  {{"metadata": map[string]any{"name": "api", "namespace": "ops"}, "spec": map[string]any{"replicas": 3}, "status": map[string]any{"readyReplicas": 2}}},
		NamespaceWorkloadStatefulSet: {{"metadata": map[string]any{"name": "db", "namespace": "ops"}, "spec": map[string]any{"replicas": 2}, "status": map[string]any{"readyReplicas": 2}}},
		NamespaceWorkloadDaemonSet:   {{"metadata": map[string]any{"name": "agent", "namespace": "ops"}, "status": map[string]any{"desiredNumberScheduled": 4, "numberReady": 3}}},
	}})

	result, err := service.NamespaceWorkloadInventory(context.Background(), 7, " ops ")
	if err != nil {
		t.Fatalf("NamespaceWorkloadInventory() error = %v", err)
	}
	if result.RawRef["namespace"] != "ops" || result.Evidence["namespace"] != "ops" {
		t.Fatalf("namespace was not normalized: %#v", result)
	}
	counts := result.Evidence["counts"].(map[string]any)
	if counts["deployments"] != 1 || counts["statefulsets"] != 1 || counts["daemonsets"] != 1 {
		t.Fatalf("counts = %#v", counts)
	}
	replicas := result.Evidence["replicas"].(map[string]any)
	if replicas["desired"] != 9 || replicas["ready"] != 7 {
		t.Fatalf("replicas = %#v", replicas)
	}
	items := result.Evidence["items"].([]map[string]any)
	if len(items) != 3 || items[0]["kind"] != "DaemonSet" || items[2]["kind"] != "StatefulSet" {
		t.Fatalf("items not sorted by kind: %#v", items)
	}
}

func TestNamespaceWorkloadInventoryValidatesScope(t *testing.T) {
	service := NewNamespaceWorkloadService(&namespaceWorkloadRuntimeSpy{})
	if _, err := service.NamespaceWorkloadInventory(context.Background(), 0, "ops"); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid cluster error = %v", err)
	}
	if _, err := service.NamespaceWorkloadInventory(context.Background(), 1, " "); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid namespace error = %v", err)
	}
}
