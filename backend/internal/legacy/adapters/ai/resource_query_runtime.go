package ai

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	aiapp "k8s-platform-backend/internal/ai/application"
	aidomain "k8s-platform-backend/internal/ai/domain"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

// ResourceQueryRuntime adapts the retained Kubernetes transport to AI's
// resource-query port. Query orchestration stays in ai/application.
type ResourceQueryRuntime struct{ k8s *service.K8sService }

func NewResourceQueryRuntime(k8s *service.K8sService) *ResourceQueryRuntime {
	return &ResourceQueryRuntime{k8s: k8s}
}

func (r *ResourceQueryRuntime) List(ctx context.Context, clusterID uint64, resource aiapp.ResourceReference, namespace, sortBy, order string, options map[string]string) ([]map[string]any, error) {
	if r == nil || r.k8s == nil {
		return nil, aiapp.ErrConflict
	}
	items, err := r.k8s.List(ctx, clusterID, resourceGVR(resource), namespace, sortBy, order, options)
	if err != nil {
		return nil, err
	}
	return mapResourceItems(items), nil
}

func (r *ResourceQueryRuntime) Get(ctx context.Context, clusterID uint64, resource aiapp.ResourceReference, namespace, name string) (map[string]any, error) {
	if r == nil || r.k8s == nil {
		return nil, aiapp.ErrConflict
	}
	return r.k8s.GetObject(ctx, clusterID, resourceGVR(resource), namespace, name)
}

func (r *ResourceQueryRuntime) ListNodeEvents(ctx context.Context, clusterID uint64, name string) ([]map[string]any, error) {
	if r == nil || r.k8s == nil {
		return nil, aiapp.ErrConflict
	}
	items, err := r.k8s.ListNodeEvents(ctx, clusterID, name)
	if err != nil {
		return nil, err
	}
	return mapResourceItems(items), nil
}

func (r *ResourceQueryRuntime) PodLogs(ctx context.Context, clusterID uint64, namespace, pod, container string, tailLines int64, previous bool) (string, error) {
	if r == nil || r.k8s == nil {
		return "", aiapp.ErrConflict
	}
	return r.k8s.PodLogs(ctx, clusterID, namespace, pod, container, tailLines, previous)
}

func resourceGVR(resource aiapp.ResourceReference) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: resource.Group, Version: resource.Version, Resource: resource.Resource}
}

func mapResourceItems(items []any) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		object, _ := item.(map[string]any)
		if object != nil {
			result = append(result, object)
		}
	}
	return result
}

// ResourceQueryPresenter preserves the existing evidence shape while the
// AI application service owns query and relationship policies.
type ResourceQueryPresenter struct{}

func NewResourceQueryPresenter() ResourceQueryPresenter { return ResourceQueryPresenter{} }

func (ResourceQueryPresenter) ListItemSummary(kind string, object map[string]any) aidomain.JSONMap {
	resourceKind := strings.TrimSpace(kind)
	switch strings.ToLower(resourceKind) {
	case "deployment", "statefulset", "daemonset":
		return aidomain.JSONMap(kopsapp.BuildInspectionWorkloadSummary(resourceKind, object))
	case "pod":
		return aidomain.JSONMap(kopsapp.BuildInspectionPodOverview(object))
	default:
		return aidomain.JSONMap(kopsapp.BuildInspectionResourceOverview(resourceKind, service.AIObjectMetaString(object, "namespace"), service.AIObjectMetaString(object, "name"), object))
	}
}
