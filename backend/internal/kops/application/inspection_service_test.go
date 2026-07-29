package application

import (
	"context"
	"errors"
	"testing"
)

type inspectionRuntimeSpy struct {
	namespace string
	resource  InspectionResourceReference
}

func (spy *inspectionRuntimeSpy) Namespace(_ context.Context, _ uint64, namespace string) (any, error) {
	spy.namespace = namespace
	return map[string]string{"namespace": namespace}, nil
}

func (spy *inspectionRuntimeSpy) NamespaceWorkloadInventory(_ context.Context, _ uint64, namespace string) (any, error) {
	spy.namespace = namespace
	return nil, nil
}

func (spy *inspectionRuntimeSpy) Object(_ context.Context, reference InspectionResourceReference) (map[string]any, error) {
	spy.resource = reference
	return map[string]any{"metadata": map[string]any{"name": reference.Name}}, nil
}

func (*inspectionRuntimeSpy) YAML(context.Context, InspectionResourceReference) (string, error) {
	return "apiVersion: v1\nkind: Pod\n", nil
}

func (*inspectionRuntimeSpy) NodeEvents(context.Context, uint64, string) ([]any, error) {
	return nil, nil
}

func (*inspectionRuntimeSpy) RolloutHistory(context.Context, InspectionResourceReference) ([]any, error) {
	return nil, nil
}

func (*inspectionRuntimeSpy) Supports(context.Context, uint64, string) (bool, error) {
	return false, nil
}

func (*inspectionRuntimeSpy) List(context.Context, uint64, string, string) ([]any, error) {
	return nil, nil
}

func (*inspectionRuntimeSpy) PodLogs(context.Context, uint64, string, string, int64) (string, error) {
	return "", nil
}

func TestInspectionServiceValidatesAndNormalizesReferences(t *testing.T) {
	spy := &inspectionRuntimeSpy{}
	service := NewInspectionService(spy)
	if _, err := service.Namespace(context.Background(), 2, " ops "); err != nil || spy.namespace != "ops" {
		t.Fatalf("Namespace() namespace=%q err=%v", spy.namespace, err)
	}
	if _, err := service.Resource(context.Background(), 2, "Pod", "ops", " api "); err != nil || spy.resource.Name != "api" {
		t.Fatalf("Resource() name=%q err=%v", spy.resource.Name, err)
	}
	if _, err := service.Resource(context.Background(), 0, "Pod", "ops", "api"); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid resource error=%v", err)
	}
}
