package kops

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

type WorkloadRuntime struct{ service *service.K8sService }

func NewWorkloadRuntime(service *service.K8sService) *WorkloadRuntime {
	return &WorkloadRuntime{service: service}
}

func (r *WorkloadRuntime) List(ctx context.Context, query kopsapp.WorkloadQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	list := func(kind kopsapp.WorkloadKind) ([]any, error) {
		gvr, err := workloadGVR(kind)
		if err != nil {
			return nil, err
		}
		value, err := r.service.List(ctx, query.ClusterID, gvr, query.Namespace, query.SortBy, query.Order, map[string]string{"label_selector": query.LabelSelector})
		return value, translateKopsRuntimeError(err)
	}
	if query.Kind != "" {
		value, err := list(query.Kind)
		if err != nil {
			return nil, err
		}
		return map[string]any{"list": value}, nil
	}
	merged := make([]any, 0, 128)
	for _, kind := range []kopsapp.WorkloadKind{kopsapp.WorkloadDeployment, kopsapp.WorkloadStatefulSet, kopsapp.WorkloadDaemonSet} {
		value, err := list(kind)
		if err != nil {
			return nil, err
		}
		merged = append(merged, value...)
	}
	return map[string]any{"list": merged}, nil
}

func (r *WorkloadRuntime) History(ctx context.Context, ref kopsapp.WorkloadRef) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.RolloutHistory(ctx, ref.ClusterID, ref.Namespace, ref.Name, string(ref.Kind))
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"history": value}, nil
}

func (r *WorkloadRuntime) Undo(ctx context.Context, ref kopsapp.WorkloadRef, revision int) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.RolloutUndo(ctx, ref.ClusterID, ref.Namespace, ref.Name, string(ref.Kind), revision))
}

func (r *WorkloadRuntime) Patch(ctx context.Context, ref kopsapp.WorkloadRef, patch map[string]any) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := workloadGVR(ref.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name, patch))
}

func (r *WorkloadRuntime) Image(ctx context.Context, input kopsapp.WorkloadImage) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.UpdateWorkloadImage(ctx, input.ClusterID, input.Namespace, input.Name, string(input.Kind), input.Container, input.Image))
}

func (r *WorkloadRuntime) Pause(ctx context.Context, ref kopsapp.WorkloadRef, paused bool) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.UpdateWorkloadPaused(ctx, ref.ClusterID, ref.Namespace, ref.Name, string(ref.Kind), paused))
}

func (r *WorkloadRuntime) Object(ctx context.Context, ref kopsapp.WorkloadRef) (map[string]any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, err := workloadGVR(ref.Kind)
	if err != nil {
		return nil, err
	}
	value, err := r.service.GetObject(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	return value, translateKopsRuntimeError(err)
}

func (r *WorkloadRuntime) ApplyYAML(ctx context.Context, input kopsapp.WorkloadYAMLEdit) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := workloadGVR(input.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.service.ApplyYAML(ctx, input.ClusterID, gvr, input.Namespace, input.YAML))
}

func (r *WorkloadRuntime) Delete(ctx context.Context, ref kopsapp.WorkloadRef) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, err := workloadGVR(ref.Kind)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}

func (r *WorkloadRuntime) YAML(ctx context.Context, ref kopsapp.WorkloadRef) (string, error) {
	if r == nil || r.service == nil {
		return "", kopsapp.ErrConflict
	}
	gvr, err := workloadGVR(ref.Kind)
	if err != nil {
		return "", err
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	return value, translateKopsRuntimeError(err)
}

func workloadGVR(kind kopsapp.WorkloadKind) (schema.GroupVersionResource, error) {
	switch kind {
	case kopsapp.WorkloadDeployment:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil
	case kopsapp.WorkloadStatefulSet:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, nil
	case kopsapp.WorkloadDaemonSet:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, nil
	default:
		return schema.GroupVersionResource{}, kopsapp.ErrInvalidParams
	}
}
