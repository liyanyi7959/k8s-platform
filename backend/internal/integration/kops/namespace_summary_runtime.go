package kops

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/kops/adapters/legacycompat"
)

// NamespaceSummaryRuntime is the Kubernetes-facing side of the namespace
// inventory. It deliberately contains only discovery and list transport;
// selection, filtering and aggregation live in Kops application.
type NamespaceSummaryRuntime struct{ service *service.K8sService }

func NewNamespaceSummaryRuntime(service *service.K8sService) *NamespaceSummaryRuntime {
	return &NamespaceSummaryRuntime{service: service}
}

func (r *NamespaceSummaryRuntime) PreferredResources(ctx context.Context, clusterID uint64) ([]kopsapp.NamespaceAPIResourceList, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	client, err := r.service.DiscoveryClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	resourceLists, err := client.ServerPreferredResources()
	result := make([]kopsapp.NamespaceAPIResourceList, 0, len(resourceLists))
	for _, resourceList := range resourceLists {
		if resourceList == nil {
			continue
		}
		resources := make([]kopsapp.NamespaceAPIResource, 0, len(resourceList.APIResources))
		for _, resource := range resourceList.APIResources {
			resources = append(resources, kopsapp.NamespaceAPIResource{
				Name:       resource.Name,
				Kind:       resource.Kind,
				Namespaced: resource.Namespaced,
				Verbs:      append([]string(nil), resource.Verbs...),
			})
		}
		result = append(result, kopsapp.NamespaceAPIResourceList{
			GroupVersion: resourceList.GroupVersion,
			Resources:    resources,
		})
	}
	return result, err
}

func (r *NamespaceSummaryRuntime) ListNamespaceResources(ctx context.Context, clusterID uint64, resource kopsapp.NamespaceResourceDescriptor, namespace string) ([]any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	return r.service.List(ctx, clusterID, resourceGroupVersionResource(resource), namespace, "", "", nil)
}

func resourceGroupVersionResource(resource kopsapp.NamespaceResourceDescriptor) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: resource.Group, Version: resource.Version, Resource: resource.Resource}
}
