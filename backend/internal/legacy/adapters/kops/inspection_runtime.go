package kops

import (
	"context"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

// InspectionRuntime keeps the existing read-model implementations behind the
// Kops inspection port while legacy controller dependencies are retired.
type InspectionRuntime struct {
	namespace *service.NamespaceDiagnosisService
	resource  *service.ResourceInspectionService
}

func NewInspectionRuntime(namespace *service.NamespaceDiagnosisService, resource *service.ResourceInspectionService) *InspectionRuntime {
	return &InspectionRuntime{namespace: namespace, resource: resource}
}

func (r *InspectionRuntime) Namespace(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if r == nil || r.namespace == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.namespace.GetNamespaceInspection(ctx, clusterID, namespace)
	return value, translateKopsRuntimeError(err)
}

func (r *InspectionRuntime) NamespaceWorkloadInventory(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if r == nil || r.namespace == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.namespace.GetNamespaceWorkloadInventory(ctx, clusterID, namespace)
	return value, translateKopsRuntimeError(err)
}

func (r *InspectionRuntime) Pod(ctx context.Context, clusterID uint64, namespace, name string) (any, error) {
	if r == nil || r.resource == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.resource.InspectPod(ctx, clusterID, namespace, name)
	return value, translateKopsRuntimeError(err)
}
