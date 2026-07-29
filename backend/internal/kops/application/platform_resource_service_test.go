package application

import (
	"context"
	"errors"
	"testing"
)

type platformResourceRuntimeSpy struct{ edit PlatformResourceEdit }

func (spy *platformResourceRuntimeSpy) List(context.Context, PlatformResource, PlatformResourceListQuery) (any, error) {
	return nil, nil
}
func (spy *platformResourceRuntimeSpy) YAML(context.Context, PlatformResource, PlatformResourceReference) (any, error) {
	return nil, nil
}
func (spy *platformResourceRuntimeSpy) Apply(_ context.Context, _ PlatformResource, edit PlatformResourceEdit) error {
	spy.edit = edit
	return nil
}
func (spy *platformResourceRuntimeSpy) Delete(context.Context, PlatformResource, PlatformResourceReference) error {
	return nil
}

func TestPlatformResourceServiceEnforcesResourceScope(t *testing.T) {
	spy := &platformResourceRuntimeSpy{}
	service := NewPlatformResourceService(spy)
	if err := service.Apply(context.Background(), PlatformRuntimeClass, PlatformResourceEdit{ClusterID: 2, Namespace: "ignored", YAML: " apiVersion: node.k8s.io/v1 "}); err != nil || spy.edit.Namespace != "" {
		t.Fatalf("cluster-scoped apply=%#v err=%v", spy.edit, err)
	}
	if err := service.Apply(context.Background(), PlatformCSIStorageCapacity, PlatformResourceEdit{ClusterID: 2, YAML: "apiVersion: storage.k8s.io/v1"}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("missing namespace error=%v", err)
	}
	if err := service.Apply(context.Background(), PlatformRoleBinding, PlatformResourceEdit{ClusterID: 2, YAML: "apiVersion: rbac.authorization.k8s.io/v1"}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("missing policy namespace error=%v", err)
	}
}
