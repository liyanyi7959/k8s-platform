package kops

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/kops/adapters/legacycompat"
)

// NamespaceWorkloadRuntime is the retained Kubernetes transport adapter for
// Kops namespace workload projections.
type NamespaceWorkloadRuntime struct{ k8s *service.K8sService }

func NewNamespaceWorkloadRuntime(k8s *service.K8sService) *NamespaceWorkloadRuntime {
	return &NamespaceWorkloadRuntime{k8s: k8s}
}

func (r *NamespaceWorkloadRuntime) ListNamespaceWorkloads(ctx context.Context, clusterID uint64, kind kopsapp.NamespaceWorkloadKind, namespace string) ([]map[string]any, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, err := namespaceWorkloadGVR(kind)
	if err != nil {
		return nil, err
	}
	items, err := r.k8s.List(ctx, clusterID, gvr, namespace, "metadata.name", "asc", nil)
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

func namespaceWorkloadGVR(kind kopsapp.NamespaceWorkloadKind) (schema.GroupVersionResource, error) {
	switch kind {
	case kopsapp.NamespaceWorkloadDeployment:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil
	case kopsapp.NamespaceWorkloadStatefulSet:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, nil
	case kopsapp.NamespaceWorkloadDaemonSet:
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unsupported namespace workload kind %q", kind)
	}
}
