package service

import (
	"context"
	"sort"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type NamespaceResourceSummaryItem struct {
	Key   string
	Label string
	Count int
}

type namespaceSummarySpec struct {
	key   string
	label string
	gvr   schema.GroupVersionResource
}

const namespaceSummaryConcurrency = 8

func buildNamespaceSummarySpecs(resourceLists []*metav1.APIResourceList) []namespaceSummarySpec {
	seen := make(map[string]struct{}, len(resourceLists))
	specs := make([]namespaceSummarySpec, 0, len(resourceLists))
	for _, rl := range resourceLists {
		if rl == nil {
			continue
		}
		gv, err := schema.ParseGroupVersion(strings.TrimSpace(rl.GroupVersion))
		if err != nil {
			continue
		}
		for _, apiRes := range rl.APIResources {
			if !shouldIncludeNamespaceSummaryResource(apiRes) {
				continue
			}
			key := namespaceSummaryDedupKey(apiRes)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			label := strings.TrimSpace(apiRes.Kind)
			if label == "" {
				label = strings.TrimSpace(apiRes.Name)
			}
			specs = append(specs, namespaceSummarySpec{
				key:   key,
				label: label,
				gvr:   schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: apiRes.Name},
			})
		}
	}
	sort.Slice(specs, func(i, j int) bool {
		if specs[i].label == specs[j].label {
			return specs[i].key < specs[j].key
		}
		return specs[i].label < specs[j].label
	})
	return specs
}

func shouldIncludeNamespaceSummaryResource(apiRes metav1.APIResource) bool {
	if !apiRes.Namespaced {
		return false
	}
	name := strings.TrimSpace(apiRes.Name)
	if name == "" || strings.Contains(name, "/") {
		return false
	}
	return namespaceSummaryHasVerb(apiRes, "list")
}

func namespaceSummaryHasVerb(apiRes metav1.APIResource, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	for _, verb := range apiRes.Verbs {
		if strings.EqualFold(strings.TrimSpace(verb), target) {
			return true
		}
	}
	return false
}

func namespaceSummaryDedupKey(apiRes metav1.APIResource) string {
	name := strings.ToLower(strings.TrimSpace(apiRes.Name))
	kind := strings.ToLower(strings.TrimSpace(apiRes.Kind))
	switch {
	case name == "" && kind == "":
		return ""
	case kind == "":
		return name
	case name == "":
		return kind
	default:
		return name + "|" + kind
	}
}

func countNamespaceSummaryObjects(spec namespaceSummarySpec, list []any) int {
	count := 0
	for _, item := range list {
		if shouldIgnoreNamespaceSummaryObject(spec, item) {
			continue
		}
		count++
	}
	return count
}

func shouldIgnoreNamespaceSummaryObject(spec namespaceSummarySpec, item any) bool {
	obj, ok := item.(map[string]any)
	if !ok || obj == nil {
		return false
	}
	name, _, _ := unstructured.NestedString(obj, "metadata", "name")
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	switch {
	case spec.gvr.Group == "" && spec.gvr.Version == "v1" && spec.gvr.Resource == "configmaps":
		return name == "kube-root-ca.crt"
	case spec.gvr.Group == "" && spec.gvr.Version == "v1" && spec.gvr.Resource == "serviceaccounts":
		return name == "default"
	case spec.gvr.Group == "" && spec.gvr.Version == "v1" && spec.gvr.Resource == "secrets":
		typ, _, _ := unstructured.NestedString(obj, "type")
		saName, _, _ := unstructured.NestedString(obj, "metadata", "annotations", "kubernetes.io/service-account.name")
		lowerName := strings.ToLower(name)
		if strings.EqualFold(strings.TrimSpace(typ), "kubernetes.io/service-account-token") && strings.EqualFold(strings.TrimSpace(saName), "default") {
			return true
		}
		return strings.HasPrefix(lowerName, "default-token-") || strings.HasPrefix(lowerName, "default-dockercfg-")
	default:
		return false
	}
}

func (s *K8sService) GetNamespaceResourcesSummary(ctx context.Context, clusterID uint64, namespace string) ([]NamespaceResourceSummaryItem, int, error) {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return nil, 0, ErrInvalidParams
	}
	dc, err := s.discoveryClient(ctx, clusterID)
	if err != nil {
		return nil, 0, err
	}
	resourceLists, err := dc.ServerPreferredResources()
	if err != nil && len(resourceLists) == 0 {
		return nil, 0, err
	}
	summarySpecs := buildNamespaceSummarySpecs(resourceLists)
	if len(summarySpecs) == 0 {
		return []NamespaceResourceSummaryItem{}, 0, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	items := make([]NamespaceResourceSummaryItem, len(summarySpecs))
	var firstErr error
	var errMu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, namespaceSummaryConcurrency)

	for idx, spec := range summarySpecs {
		wg.Add(1)
		go func(index int, spec namespaceSummarySpec) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			list, err := s.List(ctx, clusterID, spec.gvr, ns, "", "", nil)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
					cancel()
				}
				errMu.Unlock()
				return
			}
			items[index] = NamespaceResourceSummaryItem{Key: spec.key, Label: spec.label, Count: countNamespaceSummaryObjects(spec, list)}
		}(idx, spec)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, 0, firstErr
	}

	total := 0
	for _, item := range items {
		total += item.Count
	}
	return items, total, nil
}
