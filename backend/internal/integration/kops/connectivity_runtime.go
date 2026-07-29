package kops

import (
	"context"
	"fmt"
	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ConnectivityRuntime struct{ service *service.K8sService }

func NewConnectivityRuntime(service *service.K8sService) *ConnectivityRuntime {
	return &ConnectivityRuntime{service: service}
}
func (r *ConnectivityRuntime) List(ctx context.Context, resource kopsapp.ConnectivityResource, query kopsapp.ConnectivityListQuery) (any, error) {
	gvr, err := connectivityGVR(resource)
	if err != nil {
		return nil, err
	}
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.List(ctx, query.ClusterID, gvr, query.Namespace, query.SortBy, query.Order, nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}
func (r *ConnectivityRuntime) YAML(ctx context.Context, resource kopsapp.ConnectivityResource, ref kopsapp.ConnectivityReference) (any, error) {
	gvr, err := connectivityGVR(resource)
	if err != nil {
		return nil, err
	}
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}
func (r *ConnectivityRuntime) Apply(ctx context.Context, resource kopsapp.ConnectivityResource, edit kopsapp.ConnectivityEdit) error {
	gvr, err := connectivityGVR(resource)
	if err != nil {
		return err
	}
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.ApplyYAML(ctx, edit.ClusterID, gvr, edit.Namespace, edit.YAML))
}
func (r *ConnectivityRuntime) Delete(ctx context.Context, resource kopsapp.ConnectivityResource, ref kopsapp.ConnectivityReference) error {
	gvr, err := connectivityGVR(resource)
	if err != nil {
		return err
	}
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}
func connectivityGVR(resource kopsapp.ConnectivityResource) (schema.GroupVersionResource, error) {
	switch resource {
	case kopsapp.ConnectivityEndpoints:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}, nil
	case kopsapp.ConnectivityEndpointSlices:
		return schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}, nil
	case kopsapp.ConnectivityLeases:
		return schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown connectivity resource")
	}
}
