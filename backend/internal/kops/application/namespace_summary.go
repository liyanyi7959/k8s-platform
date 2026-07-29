package application

import (
	"context"
	"sort"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NamespaceResourceSummaryItem is the transport-neutral count of one
// namespaced Kubernetes resource kind.
type NamespaceResourceSummaryItem struct {
	Key   string
	Label string
	Count int
}

// NamespaceResourceSummary is the aggregate returned to callers that need a
// compact namespace inventory.
type NamespaceResourceSummary struct {
	Items []NamespaceResourceSummaryItem
	Total int
}

// NamespaceResourceDescriptor identifies a Kubernetes resource without
// leaking a client implementation into the application boundary.
type NamespaceResourceDescriptor struct {
	Group    string
	Version  string
	Resource string
}

// NamespaceAPIResource is the discovery information needed to decide whether
// a resource can participate in a namespace summary.
type NamespaceAPIResource struct {
	Name       string
	Kind       string
	Namespaced bool
	Verbs      []string
}

// NamespaceAPIResourceList is the application representation of a discovery
// APIResourceList. Kubernetes discovery remains an adapter responsibility.
type NamespaceAPIResourceList struct {
	GroupVersion string
	Resources    []NamespaceAPIResource
}

// NamespaceSummaryRuntime performs Kubernetes discovery and resource list
// calls. The application service owns selection, filtering, concurrency and
// aggregate calculation.
type NamespaceSummaryRuntime interface {
	PreferredResources(context.Context, uint64) ([]NamespaceAPIResourceList, error)
	ListNamespaceResources(context.Context, uint64, NamespaceResourceDescriptor, string) ([]any, error)
}

// NamespaceResourceSummaryReader lets other bounded contexts consume the
// stable summary result without depending on a Kubernetes adapter.
type NamespaceResourceSummaryReader interface {
	Summary(context.Context, uint64, string) (NamespaceResourceSummary, error)
}

type NamespaceSummaryService struct{ runtime NamespaceSummaryRuntime }

const namespaceSummaryConcurrency = 8

func NewNamespaceSummaryService(runtime NamespaceSummaryRuntime) *NamespaceSummaryService {
	return &NamespaceSummaryService{runtime: runtime}
}

func (s *NamespaceSummaryService) Summary(ctx context.Context, clusterID uint64, namespace string) (NamespaceResourceSummary, error) {
	namespace = strings.TrimSpace(namespace)
	if clusterID == 0 || namespace == "" {
		return NamespaceResourceSummary{}, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return NamespaceResourceSummary{}, ErrConflict
	}

	resourceLists, discoveryErr := s.runtime.PreferredResources(ctx, clusterID)
	if discoveryErr != nil && len(resourceLists) == 0 {
		return NamespaceResourceSummary{}, discoveryErr
	}
	specs := buildNamespaceSummarySpecs(resourceLists)
	if len(specs) == 0 {
		return NamespaceResourceSummary{Items: []NamespaceResourceSummaryItem{}}, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	items := make([]NamespaceResourceSummaryItem, len(specs))
	sem := make(chan struct{}, namespaceSummaryConcurrency)
	var firstErr error
	var failOnce sync.Once
	var wg sync.WaitGroup
	for index, spec := range specs {
		wg.Add(1)
		go func(index int, spec namespaceSummarySpec) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			objects, err := s.runtime.ListNamespaceResources(ctx, clusterID, spec.resource, namespace)
			if err != nil {
				failOnce.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}
			items[index] = NamespaceResourceSummaryItem{
				Key:   spec.key,
				Label: spec.label,
				Count: countNamespaceSummaryObjects(spec, objects),
			}
		}(index, spec)
	}
	wg.Wait()
	if firstErr != nil {
		return NamespaceResourceSummary{}, firstErr
	}

	total := 0
	for _, item := range items {
		total += item.Count
	}
	return NamespaceResourceSummary{Items: items, Total: total}, nil
}

type namespaceSummarySpec struct {
	key      string
	label    string
	resource NamespaceResourceDescriptor
}

func buildNamespaceSummarySpecs(resourceLists []NamespaceAPIResourceList) []namespaceSummarySpec {
	seen := make(map[string]struct{}, len(resourceLists))
	specs := make([]namespaceSummarySpec, 0, len(resourceLists))
	for _, resourceList := range resourceLists {
		groupVersion, err := schema.ParseGroupVersion(strings.TrimSpace(resourceList.GroupVersion))
		if err != nil {
			continue
		}
		for _, apiResource := range resourceList.Resources {
			if !shouldIncludeNamespaceSummaryResource(apiResource) {
				continue
			}
			key := namespaceSummaryDedupKey(apiResource)
			if key == "" {
				continue
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			label := strings.TrimSpace(apiResource.Kind)
			if label == "" {
				label = strings.TrimSpace(apiResource.Name)
			}
			specs = append(specs, namespaceSummarySpec{
				key:   key,
				label: label,
				resource: NamespaceResourceDescriptor{
					Group:    groupVersion.Group,
					Version:  groupVersion.Version,
					Resource: apiResource.Name,
				},
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

func shouldIncludeNamespaceSummaryResource(apiResource NamespaceAPIResource) bool {
	if !apiResource.Namespaced {
		return false
	}
	name := strings.TrimSpace(apiResource.Name)
	if name == "" || strings.Contains(name, "/") {
		return false
	}
	return namespaceSummaryHasVerb(apiResource, "list")
}

func namespaceSummaryHasVerb(apiResource NamespaceAPIResource, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	for _, verb := range apiResource.Verbs {
		if strings.EqualFold(strings.TrimSpace(verb), target) {
			return true
		}
	}
	return false
}

func namespaceSummaryDedupKey(apiResource NamespaceAPIResource) string {
	name := strings.ToLower(strings.TrimSpace(apiResource.Name))
	kind := strings.ToLower(strings.TrimSpace(apiResource.Kind))
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

func countNamespaceSummaryObjects(spec namespaceSummarySpec, objects []any) int {
	count := 0
	for _, object := range objects {
		if shouldIgnoreNamespaceSummaryObject(spec, object) {
			continue
		}
		count++
	}
	return count
}

func shouldIgnoreNamespaceSummaryObject(spec namespaceSummarySpec, object any) bool {
	item, ok := object.(map[string]any)
	if !ok || item == nil {
		return false
	}
	name := strings.TrimSpace(nestedNamespaceSummaryString(item, "metadata", "name"))
	if name == "" {
		return false
	}
	switch {
	case spec.resource.Group == "" && spec.resource.Version == "v1" && spec.resource.Resource == "configmaps":
		return name == "kube-root-ca.crt"
	case spec.resource.Group == "" && spec.resource.Version == "v1" && spec.resource.Resource == "serviceaccounts":
		return name == "default"
	case spec.resource.Group == "" && spec.resource.Version == "v1" && spec.resource.Resource == "secrets":
		typ := nestedNamespaceSummaryString(item, "type")
		serviceAccount := nestedNamespaceSummaryString(item, "metadata", "annotations", "kubernetes.io/service-account.name")
		lowerName := strings.ToLower(name)
		if strings.EqualFold(strings.TrimSpace(typ), "kubernetes.io/service-account-token") && strings.EqualFold(strings.TrimSpace(serviceAccount), "default") {
			return true
		}
		return strings.HasPrefix(lowerName, "default-token-") || strings.HasPrefix(lowerName, "default-dockercfg-")
	default:
		return false
	}
}

func nestedNamespaceSummaryString(object map[string]any, fields ...string) string {
	var value any = object
	for _, field := range fields {
		current, ok := value.(map[string]any)
		if !ok {
			return ""
		}
		value = current[field]
	}
	text, _ := value.(string)
	return text
}
