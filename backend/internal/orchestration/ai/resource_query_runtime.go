package ai

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	aiapp "k8s-platform-backend/internal/ai/application"
	aidomain "k8s-platform-backend/internal/ai/domain"
	kopsapp "k8s-platform-backend/internal/kops/application"
	kopsports "k8s-platform-backend/internal/kops/ports"
)

// NodeEventReader keeps node-event selection in the Kops infrastructure
// adapter while letting AI retain its own resource-query port.
type NodeEventReader interface {
	ListEvents(context.Context, uint64, string) ([]any, error)
}

// PodLogReader keeps log streaming transport in the Kops infrastructure
// adapter while AI only depends on its resource-query port.
type PodLogReader interface {
	PodLogs(context.Context, uint64, string, string, string, int64, bool) (string, error)
}

// ResourceQueryRuntime adapts the retained Kubernetes transport to AI's
// resource-query port through the controlled generic API boundary. Query
// orchestration stays in ai/application.
type ResourceQueryRuntime struct {
	resources  kopsports.GenericResourceQuery
	nodeEvents NodeEventReader
	podLogs    PodLogReader
}

func NewResourceQueryRuntime(resources kopsports.GenericResourceQuery, nodeEvents NodeEventReader, podLogs PodLogReader) *ResourceQueryRuntime {
	return &ResourceQueryRuntime{resources: resources, nodeEvents: nodeEvents, podLogs: podLogs}
}

func (r *ResourceQueryRuntime) List(ctx context.Context, clusterID uint64, resource aiapp.ResourceReference, namespace, sortBy, order string, options map[string]string) ([]map[string]any, error) {
	if r == nil || r.resources == nil {
		return nil, aiapp.ErrConflict
	}
	items, err := r.resources.List(ctx, clusterID, resourceGVR(resource), namespace, sortBy, order, options)
	if err != nil {
		return nil, err
	}
	return mapResourceItems(items), nil
}

func (r *ResourceQueryRuntime) Get(ctx context.Context, clusterID uint64, resource aiapp.ResourceReference, namespace, name string) (map[string]any, error) {
	if r == nil || r.resources == nil {
		return nil, aiapp.ErrConflict
	}
	return r.resources.GetObject(ctx, clusterID, resourceGVR(resource), namespace, name)
}

func (r *ResourceQueryRuntime) ListNodeEvents(ctx context.Context, clusterID uint64, name string) ([]map[string]any, error) {
	if r == nil || r.nodeEvents == nil {
		return nil, aiapp.ErrConflict
	}
	items, err := r.nodeEvents.ListEvents(ctx, clusterID, name)
	if err != nil {
		return nil, err
	}
	return mapResourceItems(items), nil
}

func (r *ResourceQueryRuntime) PodLogs(ctx context.Context, clusterID uint64, namespace, pod, container string, tailLines int64, previous bool) (string, error) {
	if r == nil || r.podLogs == nil {
		return "", aiapp.ErrConflict
	}
	return r.podLogs.PodLogs(ctx, clusterID, namespace, pod, container, tailLines, previous)
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
		return aidomain.JSONMap(kopsapp.BuildInspectionResourceOverview(resourceKind, objectMetaString(object, "namespace"), objectMetaString(object, "name"), object))
	}
}

func objectMetaString(item any, key string) string {
	object, ok := item.(map[string]any)
	if !ok {
		return ""
	}
	metadata, ok := object["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(metadata[key]))
}
