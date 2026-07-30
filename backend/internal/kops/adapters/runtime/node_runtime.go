package runtime

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

var nodeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}

type NodeRuntime struct {
	service    *service.K8sService
	operations *NodeOperations
}

func NewNodeRuntime(service *service.K8sService, operations *NodeOperations) *NodeRuntime {
	return &NodeRuntime{service: service, operations: operations}
}

func (r *NodeRuntime) List(ctx context.Context, query kopsapp.NodeListQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.List(ctx, query.ClusterID, nodeGVR, "", query.SortBy, query.Order, nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}

func (r *NodeRuntime) Detail(ctx context.Context, ref kopsapp.NodeReference) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetObject(ctx, ref.ClusterID, nodeGVR, "", ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"obj": value}, nil
}

func (r *NodeRuntime) YAML(ctx context.Context, ref kopsapp.NodeReference) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, nodeGVR, "", ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}

func (r *NodeRuntime) Pods(ctx context.Context, ref kopsapp.NodeReference, sortBy, order string) (any, error) {
	if r == nil || r.operations == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.operations.ListPods(ctx, ref.ClusterID, ref.Name, sortBy, order)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}

func (r *NodeRuntime) Events(ctx context.Context, ref kopsapp.NodeReference) (any, error) {
	if r == nil || r.operations == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.operations.ListEvents(ctx, ref.ClusterID, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}

func (r *NodeRuntime) SetSchedulable(ctx context.Context, ref kopsapp.NodeReference, unschedulable bool) error {
	if r == nil || r.operations == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.operations.SetSchedulable(ctx, ref.ClusterID, ref.Name, unschedulable))
}

func (r *NodeRuntime) Drain(ctx context.Context, input kopsapp.NodeDrainInput) error {
	if r == nil || r.operations == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.operations.Drain(ctx, input.ClusterID, input.Name, NodeDrainOptions{
		TimeoutSeconds: input.TimeoutSeconds, Force: input.Force, IgnoreDaemonSets: input.IgnoreDaemonSets,
	}))
}

func (r *NodeRuntime) Delete(ctx context.Context, ref kopsapp.NodeReference) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, nodeGVR, "", ref.Name))
}
