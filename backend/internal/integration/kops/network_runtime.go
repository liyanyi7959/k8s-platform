package kops

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/kops/adapters/legacycompat"
)

type NetworkRuntime struct{ service *service.K8sService }

func NewNetworkRuntime(service *service.K8sService) *NetworkRuntime {
	return &NetworkRuntime{service: service}
}

func (r *NetworkRuntime) List(ctx context.Context, resource kopsapp.NetworkResource, query kopsapp.NetworkListQuery) (any, error) {
	gvr, err := networkGVR(resource)
	if err != nil {
		return nil, err
	}
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.List(ctx, query.ClusterID, gvr, query.Namespace, query.SortBy, query.Order, nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}

func (r *NetworkRuntime) YAML(ctx context.Context, resource kopsapp.NetworkResource, ref kopsapp.NetworkReference) (any, error) {
	gvr, err := networkGVR(resource)
	if err != nil {
		return nil, err
	}
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}

func (r *NetworkRuntime) Delete(ctx context.Context, resource kopsapp.NetworkResource, ref kopsapp.NetworkReference) error {
	gvr, err := networkGVR(resource)
	if err != nil {
		return err
	}
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}

func (r *NetworkRuntime) EditService(ctx context.Context, input kopsapp.ServiceEditInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	patch := map[string]any{}
	if input.Labels != nil || input.Annotations != nil {
		metadata := map[string]any{}
		if input.Labels != nil {
			metadata["labels"] = input.Labels
		}
		if input.Annotations != nil {
			metadata["annotations"] = input.Annotations
		}
		patch["metadata"] = metadata
	}
	if input.Type != nil || input.Selector != nil {
		spec := map[string]any{}
		if input.Type != nil {
			if *input.Type == "" {
				spec["type"] = nil
			} else {
				spec["type"] = *input.Type
			}
		}
		if input.Selector != nil {
			spec["selector"] = input.Selector
		}
		patch["spec"] = spec
	}
	if len(patch) == 0 {
		return nil
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, input.ClusterID, networkServiceGVR, input.Namespace, input.Name, patch))
}

func (r *NetworkRuntime) EditIngress(ctx context.Context, input kopsapp.IngressEditInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	patch := map[string]any{}
	if input.Labels != nil || input.Annotations != nil {
		metadata := map[string]any{}
		if input.Labels != nil {
			metadata["labels"] = input.Labels
		}
		if input.Annotations != nil {
			metadata["annotations"] = input.Annotations
		}
		patch["metadata"] = metadata
	}
	if input.IngressClassName != nil {
		if *input.IngressClassName == "" {
			patch["spec"] = map[string]any{"ingressClassName": nil}
		} else {
			patch["spec"] = map[string]any{"ingressClassName": *input.IngressClassName}
		}
	}
	if len(patch) == 0 {
		return nil
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, input.ClusterID, networkIngressGVR, input.Namespace, input.Name, patch))
}

func (r *NetworkRuntime) EditIngressClass(ctx context.Context, input kopsapp.IngressClassEditInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	patch := map[string]any{}
	if input.Labels != nil || input.Annotations != nil || input.IsDefault != nil {
		metadata := map[string]any{}
		if input.Labels != nil {
			metadata["labels"] = input.Labels
		}
		if input.Annotations != nil || input.IsDefault != nil {
			annotations := map[string]*string{}
			for key, value := range input.Annotations {
				annotations[key] = value
			}
			if input.IsDefault != nil {
				const defaultClassAnnotation = "ingressclass.kubernetes.io/is-default-class"
				if *input.IsDefault {
					value := "true"
					annotations[defaultClassAnnotation] = &value
				} else {
					annotations[defaultClassAnnotation] = nil
				}
			}
			metadata["annotations"] = annotations
		}
		patch["metadata"] = metadata
	}
	if input.Controller != nil {
		patch["spec"] = map[string]any{"controller": *input.Controller}
	}
	if len(patch) == 0 {
		return nil
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, input.ClusterID, networkIngressClassGVR, "", input.Name, patch))
}

func networkGVR(resource kopsapp.NetworkResource) (schema.GroupVersionResource, error) {
	switch resource {
	case kopsapp.NetworkKubernetesService:
		return networkServiceGVR, nil
	case kopsapp.NetworkIngress:
		return networkIngressGVR, nil
	case kopsapp.NetworkIngressClass:
		return networkIngressClassGVR, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown network resource")
	}
}

var networkServiceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
var networkIngressGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
var networkIngressClassGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"}
