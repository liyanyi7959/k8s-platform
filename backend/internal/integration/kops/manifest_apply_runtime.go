package kops

import (
	"context"
	"fmt"
	"io"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

// manifestApplyOptions and manifestApplyResultItem are deliberately local to
// the Kops runtime: they describe Kubernetes apply transport, not an API DTO.
type manifestApplyOptions struct {
	DefaultNamespace string
	DryRun           bool
	CreateOnly       bool
}

type manifestApplyResultItem struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
	Operation  string `json:"operation"`
	Resource   string `json:"resource"`
	Scope      string `json:"scope"`
}

// applyManifestYAML owns document decoding, REST mapping and typed dynamic
// apply behavior. K8sService only provides the authenticated clients.
func (r *ManifestRuntime) applyManifestYAML(ctx context.Context, clusterID uint64, yamlContent string, opts manifestApplyOptions) ([]manifestApplyResultItem, error) {
	if r == nil || r.k8s == nil {
		return nil, kopsapp.ErrConflict
	}
	trimmed := strings.TrimSpace(yamlContent)
	if trimmed == "" {
		return nil, kopsapp.ErrInvalidParams
	}
	client, err := r.k8s.DynamicClient(ctx, clusterID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	mapper, err := r.manifestRESTMapper(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	decoder := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(trimmed), 4096)
	results := make([]manifestApplyResultItem, 0, 4)
	document := 0
	for {
		var raw map[string]any
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, fmt.Sprintf("invalid yaml document #%d: %v", document+1, err))
		}
		if len(raw) == 0 {
			continue
		}
		document++
		objects, err := flattenManifestObjects(&unstructured.Unstructured{Object: raw})
		if err != nil {
			return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, fmt.Sprintf("invalid yaml document #%d: %v", document, err))
		}
		for index, object := range objects {
			item, err := applyManifestObject(ctx, client, mapper, object, opts)
			if err != nil {
				prefix := fmt.Sprintf("document #%d", document)
				if len(objects) > 1 {
					prefix += fmt.Sprintf(" item #%d", index+1)
				}
				return nil, fmt.Errorf("%s apply failed: %w", prefix, err)
			}
			results = append(results, item)
		}
	}
	if len(results) == 0 {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "yaml 中没有可应用的资源")
	}
	return results, nil
}

func (r *ManifestRuntime) manifestRESTMapper(ctx context.Context, clusterID uint64) (meta.RESTMapper, error) {
	discoveryClient, err := r.k8s.DiscoveryClient(ctx, clusterID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	resources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, manifestKubernetesError(err)
	}
	return restmapper.NewDiscoveryRESTMapper(resources), nil
}

func applyManifestObject(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, object *unstructured.Unstructured, opts manifestApplyOptions) (manifestApplyResultItem, error) {
	if object == nil || len(object.Object) == 0 {
		return manifestApplyResultItem{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "empty manifest object")
	}
	gvk := object.GroupVersionKind()
	if gvk.Empty() {
		return manifestApplyResultItem{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "manifest 缺少 apiVersion 或 kind")
	}
	mapping, err := mapper.RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
	if err != nil {
		return manifestApplyResultItem{}, manifestKubernetesError(err)
	}
	value := object.DeepCopy()
	sanitizeManifestForApply(value)
	resourceClient := client.Resource(mapping.Resource)
	var resource dynamic.ResourceInterface = resourceClient
	scope := "cluster"
	namespace := strings.TrimSpace(value.GetNamespace())
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		scope = "namespace"
		if namespace == "" {
			namespace = strings.TrimSpace(opts.DefaultNamespace)
			if namespace == "" {
				return manifestApplyResultItem{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, fmt.Sprintf("%s/%s 缺少 namespace", gvk.GroupVersion().String(), gvk.Kind))
			}
			value.SetNamespace(namespace)
		}
		resource = resourceClient.Namespace(namespace)
	}
	name := strings.TrimSpace(value.GetName())
	if name == "" && strings.TrimSpace(value.GetGenerateName()) == "" {
		return manifestApplyResultItem{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, fmt.Sprintf("%s 缺少 metadata.name 或 metadata.generateName", gvk.Kind))
	}
	createOptions, updateOptions := metav1.CreateOptions{}, metav1.UpdateOptions{}
	if opts.DryRun {
		createOptions.DryRun, updateOptions.DryRun = []string{metav1.DryRunAll}, []string{metav1.DryRunAll}
	}
	operation, result := "create", value
	if name != "" {
		existing, err := resource.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return manifestApplyResultItem{}, manifestKubernetesError(err)
			}
			created, createErr := resource.Create(ctx, value, createOptions)
			if createErr != nil {
				return manifestApplyResultItem{}, manifestKubernetesError(createErr)
			}
			result = created
		} else {
			if opts.CreateOnly {
				return manifestApplyResultItem{}, kopsapp.ErrWithMessage(kopsapp.ErrConflict, fmt.Sprintf("%s %s 已存在，无法重复创建", gvk.Kind, name))
			}
			operation = "update"
			value.SetResourceVersion(existing.GetResourceVersion())
			updated, updateErr := resource.Update(ctx, value, updateOptions)
			if updateErr != nil {
				return manifestApplyResultItem{}, manifestKubernetesError(updateErr)
			}
			result = updated
		}
	} else {
		created, err := resource.Create(ctx, value, createOptions)
		if err != nil {
			return manifestApplyResultItem{}, manifestKubernetesError(err)
		}
		result = created
	}
	return manifestApplyResultItem{APIVersion: gvk.GroupVersion().String(), Kind: gvk.Kind, Namespace: result.GetNamespace(), Name: result.GetName(), Operation: operation, Resource: mapping.Resource.Resource, Scope: scope}, nil
}

func manifestKubernetesError(err error) error {
	return translateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func sanitizeManifestForApply(object *unstructured.Unstructured) {
	if object == nil {
		return
	}
	object.SetUID("")
	object.SetResourceVersion("")
	object.SetGeneration(0)
	object.SetManagedFields(nil)
	object.SetCreationTimestamp(metav1.Time{})
	unstructured.RemoveNestedField(object.Object, "status")
	metadata, ok := object.Object["metadata"].(map[string]any)
	if !ok || metadata == nil {
		return
	}
	for _, key := range []string{"uid", "resourceVersion", "generation", "creationTimestamp", "managedFields", "selfLink"} {
		delete(metadata, key)
	}
	object.Object["metadata"] = metadata
}

func flattenManifestObjects(object *unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	if object == nil || len(object.Object) == 0 {
		return nil, nil
	}
	if !strings.EqualFold(strings.TrimSpace(object.GetKind()), "List") {
		return []*unstructured.Unstructured{object}, nil
	}
	items, found, err := unstructured.NestedSlice(object.Object, "items")
	if err != nil {
		return nil, err
	}
	if !found || len(items) == 0 {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "List 清单中没有 items")
	}
	result := make([]*unstructured.Unstructured, 0, len(items))
	for _, item := range items {
		value, ok := item.(map[string]any)
		if ok && len(value) > 0 {
			result = append(result, &unstructured.Unstructured{Object: value})
		}
	}
	if len(result) == 0 {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "List 清单中没有有效对象")
	}
	return result, nil
}
