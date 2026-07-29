package kops

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

// InspectionRuntime is the infrastructure adapter for Kops inspection reads.
// Namespace diagnosis remains a separate transitional service; all object,
// event, metric and log access is delegated directly to K8sService here.
type InspectionRuntime struct {
	k8s       *service.K8sService
	namespace *service.NamespaceDiagnosisService
}

func NewInspectionRuntime(k8s *service.K8sService, namespace *service.NamespaceDiagnosisService) *InspectionRuntime {
	return &InspectionRuntime{k8s: k8s, namespace: namespace}
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

func (r *InspectionRuntime) Object(ctx context.Context, ref kopsapp.InspectionResourceReference) (map[string]any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, err := inspectionGVR(ref.Kind)
	if err != nil {
		return nil, err
	}
	value, err := r.k8s.GetObject(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	return value, translateKopsRuntimeError(err)
}

func (r *InspectionRuntime) YAML(ctx context.Context, ref kopsapp.InspectionResourceReference) (string, error) {
	if r == nil || r.k8s == nil {
		return "", kopsapp.ErrConflict
	}
	gvr, err := inspectionGVR(ref.Kind)
	if err != nil {
		return "", err
	}
	value, err := r.k8s.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	return value, translateKopsRuntimeError(err)
}

func (r *InspectionRuntime) NodeEvents(ctx context.Context, clusterID uint64, name string) ([]any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.k8s.ListNodeEvents(ctx, clusterID, name)
	return value, translateKopsRuntimeError(err)
}

func (r *InspectionRuntime) RolloutHistory(ctx context.Context, ref kopsapp.InspectionResourceReference) ([]any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.k8s.RolloutHistory(ctx, ref.ClusterID, ref.Namespace, ref.Name, ref.Kind)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	items := make([]any, 0, len(value))
	for _, item := range value {
		items = append(items, item)
	}
	return items, nil
}

func (r *InspectionRuntime) Supports(ctx context.Context, clusterID uint64, capability string) (bool, error) {
	if r == nil || r.k8s == nil {
		return false, kopsapp.ErrConflict
	}
	if !strings.EqualFold(strings.TrimSpace(capability), "pod_metrics") {
		return false, kopsapp.ErrInvalidParams
	}
	value, err := r.k8s.SupportsCompatibleGVR(ctx, clusterID, schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"})
	return value, translateKopsRuntimeError(err)
}

func (r *InspectionRuntime) List(ctx context.Context, clusterID uint64, kind, namespace string) ([]any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	if !strings.EqualFold(strings.TrimSpace(kind), "pod_metrics") {
		return nil, kopsapp.ErrInvalidParams
	}
	value, err := r.k8s.List(ctx, clusterID, schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}, namespace, "metadata.name", "asc", nil)
	return value, translateKopsRuntimeError(err)
}

func (r *InspectionRuntime) PodLogs(ctx context.Context, clusterID uint64, namespace, name string, tailLines int64) (string, error) {
	if r == nil || r.k8s == nil {
		return "", kopsapp.ErrConflict
	}
	value, err := r.k8s.PodLogs(ctx, clusterID, namespace, name, "", tailLines, false)
	return value, translateKopsRuntimeError(err)
}

func inspectionGVR(kind string) (schema.GroupVersionResource, error) {
	descriptor, ok := kopsapp.InspectionResourceDescriptorFor(kind)
	if !ok {
		return schema.GroupVersionResource{}, kopsapp.ErrInvalidParams
	}
	return schema.GroupVersionResource{Group: descriptor.Group, Version: descriptor.Version, Resource: descriptor.Resource}, nil
}
