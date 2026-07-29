package application

import (
	"context"
	"errors"
	"testing"
)

type inspectionRuntimeSpy struct {
	namespace string
	podName   string
}

func (spy *inspectionRuntimeSpy) Namespace(_ context.Context, _ uint64, namespace string) (any, error) {
	spy.namespace = namespace
	return map[string]string{"namespace": namespace}, nil
}

func (spy *inspectionRuntimeSpy) NamespaceWorkloadInventory(_ context.Context, _ uint64, namespace string) (any, error) {
	spy.namespace = namespace
	return nil, nil
}

func (spy *inspectionRuntimeSpy) Pod(_ context.Context, _ uint64, _ string, name string) (any, error) {
	spy.podName = name
	return nil, nil
}

func TestInspectionServiceValidatesAndNormalizesReferences(t *testing.T) {
	spy := &inspectionRuntimeSpy{}
	service := NewInspectionService(spy)
	if _, err := service.Namespace(context.Background(), 2, " ops "); err != nil || spy.namespace != "ops" {
		t.Fatalf("Namespace() namespace=%q err=%v", spy.namespace, err)
	}
	if _, err := service.Pod(context.Background(), 2, "ops", " api "); err != nil || spy.podName != "api" {
		t.Fatalf("Pod() name=%q err=%v", spy.podName, err)
	}
	if _, err := service.Pod(context.Background(), 0, "ops", "api"); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid pod error=%v", err)
	}
}
