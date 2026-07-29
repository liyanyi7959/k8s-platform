package application

import (
	"context"
	"errors"
	"testing"
)

type connectivityRuntimeSpy struct {
	edit     ConnectivityEdit
	resource ConnectivityResource
}

func (spy *connectivityRuntimeSpy) List(context.Context, ConnectivityResource, ConnectivityListQuery) (any, error) {
	return nil, nil
}
func (spy *connectivityRuntimeSpy) YAML(context.Context, ConnectivityResource, ConnectivityReference) (any, error) {
	return nil, nil
}
func (spy *connectivityRuntimeSpy) Apply(_ context.Context, resource ConnectivityResource, edit ConnectivityEdit) error {
	spy.resource, spy.edit = resource, edit
	return nil
}
func (spy *connectivityRuntimeSpy) Delete(context.Context, ConnectivityResource, ConnectivityReference) error {
	return nil
}
func TestConnectivityServiceValidatesAndNormalizesEdits(t *testing.T) {
	spy := &connectivityRuntimeSpy{}
	service := NewConnectivityService(spy)
	if err := service.Apply(context.Background(), ConnectivityLeases, ConnectivityEdit{ClusterID: 2, Namespace: " ops ", YAML: " apiVersion: v1 "}); err != nil {
		t.Fatalf("Apply() error=%v", err)
	}
	if spy.resource != ConnectivityLeases || spy.edit.Namespace != "ops" || spy.edit.YAML != "apiVersion: v1" {
		t.Fatalf("edit=%#v resource=%q", spy.edit, spy.resource)
	}
	if err := service.Delete(context.Background(), ConnectivityEndpoints, ConnectivityReference{ClusterID: 2, Namespace: "", Name: "api"}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid delete error=%v", err)
	}
}
