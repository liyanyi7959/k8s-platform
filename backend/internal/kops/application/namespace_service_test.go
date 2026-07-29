package application

import (
	"context"
	"errors"
	"testing"
)

type namespaceRuntimeSpy struct {
	created   NamespaceCreateInput
	clusterID uint64
}

func (spy *namespaceRuntimeSpy) List(context.Context, uint64, string, string) (any, error) {
	return nil, nil
}
func (spy *namespaceRuntimeSpy) Create(_ context.Context, clusterID uint64, input NamespaceCreateInput) error {
	spy.clusterID, spy.created = clusterID, input
	return nil
}
func (spy *namespaceRuntimeSpy) Delete(context.Context, uint64, string) error      { return nil }
func (spy *namespaceRuntimeSpy) YAML(context.Context, uint64, string) (any, error) { return nil, nil }
func (spy *namespaceRuntimeSpy) Summary(context.Context, uint64, string) (any, error) {
	return nil, nil
}
func (spy *namespaceRuntimeSpy) Events(context.Context, EventListQuery) (any, error) { return nil, nil }
func TestNamespaceServiceOwnsNamespaceValidation(t *testing.T) {
	spy := &namespaceRuntimeSpy{}
	service := NewNamespaceService(spy)
	if err := service.Create(context.Background(), 3, NamespaceCreateInput{Name: " dev ", Labels: map[string]string{" team ": " platform ", "": "ignored"}}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if spy.clusterID != 3 || spy.created.Name != "dev" || len(spy.created.Labels) != 1 || spy.created.Labels["team"] != "platform" {
		t.Fatalf("created input = %#v cluster=%d", spy.created, spy.clusterID)
	}
	if err := service.Delete(context.Background(), 0, "dev"); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("Delete invalid error = %v", err)
	}
	if _, err := service.YAML(context.Background(), 3, " "); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("YAML invalid error = %v", err)
	}
}
