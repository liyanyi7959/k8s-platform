package application

import (
	"context"
	"errors"
	"testing"
)

type storageRuntimeSpy struct{ pvc CreatePVCInput }

func (*storageRuntimeSpy) List(context.Context, StorageResource, StorageListQuery) (any, error) {
	return nil, nil
}
func (*storageRuntimeSpy) YAML(context.Context, StorageResource, StorageReference) (any, error) {
	return nil, nil
}
func (*storageRuntimeSpy) Delete(context.Context, StorageResource, StorageReference) error {
	return nil
}
func (*storageRuntimeSpy) Apply(context.Context, StorageResource, StorageEdit) error { return nil }
func (spy *storageRuntimeSpy) CreatePVC(_ context.Context, value CreatePVCInput) error {
	spy.pvc = value
	return nil
}
func (*storageRuntimeSpy) Supports(context.Context, uint64, StorageCapability) (bool, error) {
	return true, nil
}
func TestStorageServiceNormalizesPVCAndChecksScope(t *testing.T) {
	spy := &storageRuntimeSpy{}
	service := NewStorageService(spy)
	if err := service.CreatePVC(context.Background(), CreatePVCInput{ClusterID: 1, Namespace: " ops ", Name: " data ", Capacity: " 1Gi ", AccessModes: []string{"ReadWriteOnce", "ReadWriteOnce", "ReadOnlyMany"}}); err != nil {
		t.Fatalf("CreatePVC() error=%v", err)
	}
	if spy.pvc.Namespace != "ops" || spy.pvc.Name != "data" || spy.pvc.Capacity != "1Gi" || len(spy.pvc.AccessModes) != 2 {
		t.Fatalf("pvc=%#v", spy.pvc)
	}
	if err := service.CreatePVC(context.Background(), CreatePVCInput{ClusterID: 1, Namespace: "ops", Name: "data", Capacity: "not-a-quantity"}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid capacity error=%v", err)
	}
	if err := service.CreatePVC(context.Background(), CreatePVCInput{ClusterID: 1, Namespace: "ops", Name: "data", Capacity: "1Gi", AccessModes: []string{"invalid"}}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid mode error=%v", err)
	}
	if err := service.Apply(context.Background(), StorageVolumeSnapshot, StorageEdit{ClusterID: 1, YAML: "apiVersion: v1"}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("Apply invalid scope error=%v", err)
	}
}
