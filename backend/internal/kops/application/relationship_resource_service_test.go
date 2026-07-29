package application

import (
	"context"
	"errors"
	"testing"
)

type relationshipResourceRuntimeSpy struct{ edit RelationshipResourceEdit }

func (spy *relationshipResourceRuntimeSpy) List(context.Context, RelationshipResource, RelationshipResourceListQuery) (any, error) {
	return nil, nil
}
func (spy *relationshipResourceRuntimeSpy) YAML(context.Context, RelationshipResource, RelationshipResourceReference) (any, error) {
	return nil, nil
}
func (spy *relationshipResourceRuntimeSpy) Apply(_ context.Context, _ RelationshipResource, edit RelationshipResourceEdit) error {
	spy.edit = edit
	return nil
}
func (spy *relationshipResourceRuntimeSpy) Delete(context.Context, RelationshipResource, RelationshipResourceReference) error {
	return nil
}

func TestRelationshipResourceServiceEnforcesResourceScope(t *testing.T) {
	spy := &relationshipResourceRuntimeSpy{}
	service := NewRelationshipResourceService(spy)
	if err := service.Apply(context.Background(), RelationshipVolumeAttachment, RelationshipResourceEdit{ClusterID: 2, Namespace: "ignored", YAML: "apiVersion: storage.k8s.io/v1"}); err != nil || spy.edit.Namespace != "" {
		t.Fatalf("cluster-scoped apply=%#v err=%v", spy.edit, err)
	}
	if err := service.Apply(context.Background(), RelationshipReplicaSet, RelationshipResourceEdit{ClusterID: 2, YAML: "apiVersion: apps/v1"}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("missing namespace error=%v", err)
	}
}
