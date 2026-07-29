package kops

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

// NamespaceDiagnosisRuntime is the Kubernetes transport adapter for the
// Kops-owned namespace diagnostic policy.
type NamespaceDiagnosisRuntime struct {
	k8s        *service.K8sService
	operations *kopsruntime.PodOperations
}

func NewNamespaceDiagnosisRuntime(k8s *service.K8sService, operations *kopsruntime.PodOperations) *NamespaceDiagnosisRuntime {
	return &NamespaceDiagnosisRuntime{k8s: k8s, operations: operations}
}

func (r *NamespaceDiagnosisRuntime) NamespaceHealth(ctx context.Context, clusterID uint64, namespace string) (map[string]any, error) {
	if r == nil || r.operations == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.operations.NamespaceHealth(ctx, clusterID, namespace)
	return value, translateKopsRuntimeError(err)
}

func (r *NamespaceDiagnosisRuntime) SupportsPodMetrics(ctx context.Context, clusterID uint64) (bool, error) {
	if r == nil || r.k8s == nil {
		return false, kopsapp.ErrConflict
	}
	value, err := r.k8s.SupportsCompatibleGVR(ctx, clusterID, schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"})
	return value, translateKopsRuntimeError(err)
}

func (r *NamespaceDiagnosisRuntime) ListPodMetrics(ctx context.Context, clusterID uint64, namespace string) ([]map[string]any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	items, err := r.k8s.List(ctx, clusterID, schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}, namespace, "metadata.name", "asc", nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if object, ok := item.(map[string]any); ok && object != nil {
			result = append(result, object)
		}
	}
	return result, nil
}
