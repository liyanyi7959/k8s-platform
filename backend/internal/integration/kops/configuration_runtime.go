package kops

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/kops/adapters/legacycompat"
)

type ConfigurationRuntime struct{ service *service.K8sService }

func NewConfigurationRuntime(service *service.K8sService) *ConfigurationRuntime {
	return &ConfigurationRuntime{service: service}
}

func (r *ConfigurationRuntime) List(ctx context.Context, resource kopsapp.ConfigurationResource, query kopsapp.ConfigurationListQuery) (any, error) {
	gvr, err := configurationGVR(resource)
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

func (r *ConfigurationRuntime) YAML(ctx context.Context, resource kopsapp.ConfigurationResource, ref kopsapp.ConfigurationReference) (any, error) {
	gvr, err := configurationGVR(resource)
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

func (r *ConfigurationRuntime) Delete(ctx context.Context, resource kopsapp.ConfigurationResource, ref kopsapp.ConfigurationReference) error {
	gvr, err := configurationGVR(resource)
	if err != nil {
		return err
	}
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}

func (r *ConfigurationRuntime) EditConfigMap(ctx context.Context, input kopsapp.ConfigMapEditInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	patch := map[string]any{}
	if input.Labels != nil {
		patch["metadata"] = map[string]any{"labels": input.Labels}
	}
	if input.Data != nil {
		patch["data"] = input.Data
	}
	if len(patch) == 0 {
		return nil
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, input.ClusterID, configurationConfigMapGVR, input.Namespace, input.Name, patch))
}

func (r *ConfigurationRuntime) EditSecret(ctx context.Context, input kopsapp.SecretEditInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	patch := map[string]any{}
	if input.Type != nil && strings.TrimSpace(*input.Type) != "" {
		patch["type"] = strings.TrimSpace(*input.Type)
	}
	if input.Labels != nil {
		patch["metadata"] = map[string]any{"labels": input.Labels}
	}
	if input.Data != nil {
		patch["data"] = input.Data
	}
	if len(patch) == 0 {
		return nil
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, input.ClusterID, configurationSecretGVR, input.Namespace, input.Name, patch))
}

func (r *ConfigurationRuntime) SecretObject(ctx context.Context, ref kopsapp.ConfigurationReference) (map[string]any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.GetObject(ctx, ref.ClusterID, configurationSecretGVR, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return value, nil
}

func (r *ConfigurationRuntime) Pods(ctx context.Context, clusterID uint64, namespace string) ([]any, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.List(ctx, clusterID, configurationPodGVR, namespace, "", "", nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return value, nil
}

func configurationGVR(resource kopsapp.ConfigurationResource) (schema.GroupVersionResource, error) {
	switch resource {
	case kopsapp.ConfigurationConfigMap:
		return configurationConfigMapGVR, nil
	case kopsapp.ConfigurationSecret:
		return configurationSecretGVR, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown configuration resource")
	}
}

var configurationConfigMapGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
var configurationSecretGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
var configurationPodGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
