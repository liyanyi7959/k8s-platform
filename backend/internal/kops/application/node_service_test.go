package application

import (
	"context"
	"errors"
	"testing"
)

type nodeRuntimeSpy struct {
	ref   NodeReference
	drain NodeDrainInput
}

func (spy *nodeRuntimeSpy) List(context.Context, NodeListQuery) (any, error) { return nil, nil }
func (spy *nodeRuntimeSpy) Detail(_ context.Context, ref NodeReference) (any, error) {
	spy.ref = ref
	return nil, nil
}
func (spy *nodeRuntimeSpy) YAML(context.Context, NodeReference) (any, error) { return nil, nil }
func (spy *nodeRuntimeSpy) Pods(context.Context, NodeReference, string, string) (any, error) {
	return nil, nil
}
func (spy *nodeRuntimeSpy) Events(context.Context, NodeReference) (any, error)        { return nil, nil }
func (spy *nodeRuntimeSpy) SetSchedulable(context.Context, NodeReference, bool) error { return nil }
func (spy *nodeRuntimeSpy) Drain(_ context.Context, input NodeDrainInput) error {
	spy.drain = input
	return nil
}
func (spy *nodeRuntimeSpy) Delete(context.Context, NodeReference) error { return nil }

func TestNodeServiceValidatesAndNormalizesCommands(t *testing.T) {
	spy := &nodeRuntimeSpy{}
	service := NewNodeService(spy)
	if _, err := service.Detail(context.Background(), NodeReference{ClusterID: 2, Name: " worker-a "}); err != nil || spy.ref.Name != "worker-a" {
		t.Fatalf("Detail() ref=%#v err=%v", spy.ref, err)
	}
	if err := service.Drain(context.Background(), NodeDrainInput{NodeReference: NodeReference{ClusterID: 2, Name: " worker-a "}, TimeoutSeconds: 30}); err != nil || spy.drain.Name != "worker-a" {
		t.Fatalf("Drain() input=%#v err=%v", spy.drain, err)
	}
	if err := service.Drain(context.Background(), NodeDrainInput{NodeReference: NodeReference{ClusterID: 2, Name: "worker-a"}, TimeoutSeconds: -1}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid drain error=%v", err)
	}
}
