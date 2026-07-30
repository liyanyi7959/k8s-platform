package runtime

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime/schema"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

type PlatformResourceRuntime struct{ service *service.K8sService }

func NewPlatformResourceRuntime(service *service.K8sService) *PlatformResourceRuntime {
	return &PlatformResourceRuntime{service: service}
}

func (r *PlatformResourceRuntime) List(ctx context.Context, resource kopsapp.PlatformResource, query kopsapp.PlatformResourceListQuery) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, ok := platformResourceGVR(resource)
	if !ok {
		return nil, kopsapp.ErrInvalidParams
	}
	value, err := r.service.List(ctx, query.ClusterID, gvr, query.Namespace, query.SortBy, query.Order, nil)
	if errors.Is(err, service.ErrNotFound) {
		return map[string]any{"list": []any{}}, nil
	}
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}

func (r *PlatformResourceRuntime) YAML(ctx context.Context, resource kopsapp.PlatformResource, ref kopsapp.PlatformResourceReference) (any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	gvr, ok := platformResourceGVR(resource)
	if !ok {
		return nil, kopsapp.ErrInvalidParams
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}

func (r *PlatformResourceRuntime) Apply(ctx context.Context, resource kopsapp.PlatformResource, edit kopsapp.PlatformResourceEdit) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, ok := platformResourceGVR(resource)
	if !ok {
		return kopsapp.ErrInvalidParams
	}
	return translateKopsRuntimeError(r.service.ApplyYAML(ctx, edit.ClusterID, gvr, edit.Namespace, edit.YAML))
}

func (r *PlatformResourceRuntime) Delete(ctx context.Context, resource kopsapp.PlatformResource, ref kopsapp.PlatformResourceReference) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	gvr, ok := platformResourceGVR(resource)
	if !ok {
		return kopsapp.ErrInvalidParams
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}

func platformResourceGVR(resource kopsapp.PlatformResource) (schema.GroupVersionResource, bool) {
	switch resource {
	case kopsapp.PlatformRuntimeClass:
		return schema.GroupVersionResource{Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses"}, true
	case kopsapp.PlatformCSIDriver:
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csidrivers"}, true
	case kopsapp.PlatformCSINode:
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csinodes"}, true
	case kopsapp.PlatformCSIStorageCapacity:
		return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csistoragecapacities"}, true
	case kopsapp.PlatformValidatingAdmissionPolicy:
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingadmissionpolicies"}, true
	case kopsapp.PlatformValidatingAdmissionPolicyBinding:
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingadmissionpolicybindings"}, true
	case kopsapp.PlatformPodDisruptionBudget:
		return schema.GroupVersionResource{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}, true
	case kopsapp.PlatformRole:
		return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"}, true
	case kopsapp.PlatformClusterRole:
		return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}, true
	case kopsapp.PlatformRoleBinding:
		return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}, true
	case kopsapp.PlatformClusterRoleBinding:
		return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}, true
	case kopsapp.PlatformCustomResourceDefinition:
		return schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}, true
	case kopsapp.PlatformAPIService:
		return schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}, true
	case kopsapp.PlatformPriorityClass:
		return schema.GroupVersionResource{Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"}, true
	case kopsapp.PlatformValidatingWebhookConfiguration:
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}, true
	case kopsapp.PlatformMutatingWebhookConfiguration:
		return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}, true
	case kopsapp.PlatformNetworkPolicy:
		return schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}, true
	case kopsapp.PlatformResourceQuota:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"}, true
	case kopsapp.PlatformLimitRange:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "limitranges"}, true
	case kopsapp.PlatformServiceAccount:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}, true
	case kopsapp.PlatformHorizontalPodAutoscaler:
		return schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}, true
	default:
		return schema.GroupVersionResource{}, false
	}
}
