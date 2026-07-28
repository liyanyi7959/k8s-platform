package service

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
)

type ManifestApplyOptions struct {
	DefaultNamespace string
	DryRun           bool
	CreateOnly       bool
}

type ManifestApplyResultItem struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
	Operation  string `json:"operation"`
	Resource   string `json:"resource"`
	Scope      string `json:"scope"`
}

func (s *K8sService) ApplyManifestYAML(ctx context.Context, clusterID uint64, yamlContent string, opts ManifestApplyOptions) ([]ManifestApplyResultItem, error) {
	trimmed := strings.TrimSpace(yamlContent)
	if trimmed == "" {
		return nil, ErrInvalidParams
	}

	client, err := s.dynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	mapper, err := s.discoveryRESTMapper(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	decoder := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(trimmed), 4096)
	results := make([]ManifestApplyResultItem, 0, 4)
	docIndex := 0
	for {
		var raw map[string]any
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("invalid yaml document #%d: %w", docIndex+1, err)
		}
		if len(raw) == 0 {
			continue
		}
		docIndex++

		items, err := flattenManifestObjects(&unstructured.Unstructured{Object: raw})
		if err != nil {
			return nil, fmt.Errorf("invalid yaml document #%d: %w", docIndex, err)
		}
		for itemIndex, obj := range items {
			result, err := s.applyManifestObject(ctx, client, mapper, obj, opts)
			if err != nil {
				if len(items) > 1 {
					return nil, fmt.Errorf("document #%d item #%d apply failed: %w", docIndex, itemIndex+1, err)
				}
				return nil, fmt.Errorf("document #%d apply failed: %w", docIndex, err)
			}
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "yaml 中没有可应用的资源")
	}
	return results, nil
}

func (s *K8sService) discoveryRESTMapper(ctx context.Context, clusterID uint64) (meta.RESTMapper, error) {
	dc, err := s.discoveryClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	groupResources, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return nil, normalizeK8sErr(err)
	}
	return restmapper.NewDiscoveryRESTMapper(groupResources), nil
}

func (s *K8sService) applyManifestObject(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured, opts ManifestApplyOptions) (ManifestApplyResultItem, error) {
	if obj == nil || len(obj.Object) == 0 {
		return ManifestApplyResultItem{}, ErrWithMessage(ErrInvalidParams, "empty manifest object")
	}
	gvk := obj.GroupVersionKind()
	if gvk.Empty() {
		return ManifestApplyResultItem{}, ErrWithMessage(ErrInvalidParams, "manifest 缺少 apiVersion 或 kind")
	}
	mapping, err := mapper.RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
	if err != nil {
		return ManifestApplyResultItem{}, normalizeK8sErr(err)
	}

	manifestObj := obj.DeepCopy()
	sanitizeManifestForApply(manifestObj)

	resourceClient := client.Resource(mapping.Resource)
	var resource dynamic.ResourceInterface = resourceClient
	scope := "cluster"
	namespace := strings.TrimSpace(manifestObj.GetNamespace())
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		scope = "namespace"
		if namespace == "" {
			namespace = strings.TrimSpace(opts.DefaultNamespace)
			if namespace == "" {
				return ManifestApplyResultItem{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("%s/%s 缺少 namespace", gvk.GroupVersion().String(), gvk.Kind))
			}
			manifestObj.SetNamespace(namespace)
		}
		resource = resourceClient.Namespace(namespace)
	}

	name := strings.TrimSpace(manifestObj.GetName())
	if name == "" && strings.TrimSpace(manifestObj.GetGenerateName()) == "" {
		return ManifestApplyResultItem{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("%s 缺少 metadata.name 或 metadata.generateName", gvk.Kind))
	}

	createOptions := metav1.CreateOptions{}
	updateOptions := metav1.UpdateOptions{}
	if opts.DryRun {
		createOptions.DryRun = []string{metav1.DryRunAll}
		updateOptions.DryRun = []string{metav1.DryRunAll}
	}

	operation := "create"
	resultObj := manifestObj
	if name != "" {
		existing, err := resource.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ManifestApplyResultItem{}, normalizeK8sErr(err)
			}
			created, createErr := resource.Create(ctx, manifestObj, createOptions)
			if createErr != nil {
				return ManifestApplyResultItem{}, normalizeK8sErr(createErr)
			}
			resultObj = created
		} else {
			if opts.CreateOnly {
				return ManifestApplyResultItem{}, ErrWithMessage(ErrConflict, fmt.Sprintf("%s %s 已存在，无法重复创建", gvk.Kind, name))
			}
			operation = "update"
			manifestObj.SetResourceVersion(existing.GetResourceVersion())
			updated, updateErr := resource.Update(ctx, manifestObj, updateOptions)
			if updateErr != nil {
				return ManifestApplyResultItem{}, normalizeK8sErr(updateErr)
			}
			resultObj = updated
		}
	} else {
		created, createErr := resource.Create(ctx, manifestObj, createOptions)
		if createErr != nil {
			return ManifestApplyResultItem{}, normalizeK8sErr(createErr)
		}
		resultObj = created
	}

	return ManifestApplyResultItem{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Namespace:  resultObj.GetNamespace(),
		Name:       resultObj.GetName(),
		Operation:  operation,
		Resource:   mapping.Resource.Resource,
		Scope:      scope,
	}, nil
}

func sanitizeManifestForApply(obj *unstructured.Unstructured) {
	if obj == nil {
		return
	}
	obj.SetUID("")
	obj.SetResourceVersion("")
	obj.SetGeneration(0)
	obj.SetManagedFields(nil)
	obj.SetCreationTimestamp(metav1.Time{})
	unstructured.RemoveNestedField(obj.Object, "status")
	metaMap, ok := obj.Object["metadata"].(map[string]any)
	if !ok || metaMap == nil {
		return
	}
	delete(metaMap, "uid")
	delete(metaMap, "resourceVersion")
	delete(metaMap, "generation")
	delete(metaMap, "creationTimestamp")
	delete(metaMap, "managedFields")
	delete(metaMap, "selfLink")
	obj.Object["metadata"] = metaMap
}

func flattenManifestObjects(obj *unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	if obj == nil || len(obj.Object) == 0 {
		return nil, nil
	}
	if !strings.EqualFold(strings.TrimSpace(obj.GetKind()), "List") {
		return []*unstructured.Unstructured{obj}, nil
	}
	items, found, err := unstructured.NestedSlice(obj.Object, "items")
	if err != nil {
		return nil, err
	}
	if !found || len(items) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "List 清单中没有 items")
	}
	result := make([]*unstructured.Unstructured, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok || len(entry) == 0 {
			continue
		}
		result = append(result, &unstructured.Unstructured{Object: entry})
	}
	if len(result) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "List 清单中没有有效对象")
	}
	return result, nil
}
