package runtime

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime/schema"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type PodRuntime struct {
	service    *service.K8sService
	operations *PodOperations
	streams    *PodStreamOperations
}

func NewPodRuntime(service *service.K8sService, operations *PodOperations, streams *PodStreamOperations) *PodRuntime {
	return &PodRuntime{service: service, operations: operations, streams: streams}
}
func (r *PodRuntime) List(ctx context.Context, query kopsapp.PodListQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	var extra map[string]string
	if query.LabelSelector != "" {
		extra = map[string]string{"label_selector": query.LabelSelector}
	}
	value, err := r.service.List(ctx, query.ClusterID, podGVR, query.Namespace, query.SortBy, query.Order, extra)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}
func (r *PodRuntime) Metrics(ctx context.Context, query kopsapp.PodListQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.List(ctx, query.ClusterID, podMetricsGVR, query.Namespace, query.SortBy, query.Order, nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}
func (r *PodRuntime) YAML(ctx context.Context, ref kopsapp.PodReference) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, podGVR, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}
func (r *PodRuntime) Logs(ctx context.Context, input kopsapp.PodLogsInput) (any, error) {
	if r == nil || r.streams == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.streams.PodLogs(ctx, input.ClusterID, input.Namespace, input.Name, input.Container, input.TailLines, input.Previous)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}
func (r *PodRuntime) Delete(ctx context.Context, ref kopsapp.PodReference, force bool) error {
	if r == nil || r.operations == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.operations.Delete(ctx, ref.ClusterID, ref.Namespace, ref.Name, force))
}

var podGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
var podMetricsGVR = schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
