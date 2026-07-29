package kops

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type RelationshipResourceRuntime struct{ service *service.K8sService }

func NewRelationshipResourceRuntime(service *service.K8sService) *RelationshipResourceRuntime {
	return &RelationshipResourceRuntime{service: service}
}

func (r *RelationshipResourceRuntime) List(ctx context.Context, resource kopsapp.RelationshipResource, query kopsapp.RelationshipResourceListQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, ok := relationshipResourceGVR(resource)
	if !ok {
		return nil, kopsapp.ErrInvalidParams
	}
	value, err := r.service.List(ctx, query.ClusterID, gvr, query.Namespace, query.SortBy, query.Order, nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}

func (r *RelationshipResourceRuntime) YAML(ctx context.Context, resource kopsapp.RelationshipResource, ref kopsapp.RelationshipResourceReference) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, ok := relationshipResourceGVR(resource)
	if !ok {
		return nil, kopsapp.ErrInvalidParams
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}

func (r *RelationshipResourceRuntime) Apply(ctx context.Context, resource kopsapp.RelationshipResource, edit kopsapp.RelationshipResourceEdit) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, ok := relationshipResourceGVR(resource)
	if !ok {
		return kopsapp.ErrInvalidParams
	}
	return translateKopsRuntimeError(r.service.ApplyYAML(ctx, edit.ClusterID, gvr, edit.Namespace, edit.YAML))
}

func (r *RelationshipResourceRuntime) Delete(ctx context.Context, resource kopsapp.RelationshipResource, ref kopsapp.RelationshipResourceReference) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, ok := relationshipResourceGVR(resource)
	if !ok {
		return kopsapp.ErrInvalidParams
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}

func relationshipResourceGVR(resource kopsapp.RelationshipResource) (schema.GroupVersionResource, bool) {
	switch resource {
	case kopsapp.RelationshipReplicaSet:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}, true
	case kopsapp.RelationshipVolumeAttachment:
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "volumeattachments"}, true
	default:
		return schema.GroupVersionResource{}, false
	}
}
