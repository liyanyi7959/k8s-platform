package application

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

func TestBuildNamespaceSummarySpecsFiltersAndDeduplicates(t *testing.T) {
	resourceLists := []NamespaceAPIResourceList{
		{GroupVersion: "v1", Resources: []NamespaceAPIResource{
			{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: []string{"get", "list"}},
			{Name: "pods/status", Kind: "Pod", Namespaced: true, Verbs: []string{"get"}},
			{Name: "events", Kind: "Event", Namespaced: true, Verbs: []string{"list"}},
		}},
		{GroupVersion: "events.k8s.io/v1", Resources: []NamespaceAPIResource{{Name: "events", Kind: "Event", Namespaced: true, Verbs: []string{"list"}}}},
		{GroupVersion: "apps/v1", Resources: []NamespaceAPIResource{
			{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: []string{"list"}},
			{Name: "controllerrevisions", Kind: "ControllerRevision", Namespaced: true, Verbs: []string{"get"}},
		}},
		{GroupVersion: "rbac.authorization.k8s.io/v1", Resources: []NamespaceAPIResource{{Name: "clusterroles", Kind: "ClusterRole", Namespaced: false, Verbs: []string{"list"}}}},
	}

	specs := buildNamespaceSummarySpecs(resourceLists)
	got := make([]string, 0, len(specs))
	for _, spec := range specs {
		got = append(got, spec.label+":"+spec.resource.Group+"/"+spec.resource.Version+"/"+spec.resource.Resource)
	}
	want := []string{"Deployment:apps/v1/deployments", "Event:/v1/events", "Pod:/v1/pods"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("specs = %v, want %v", got, want)
	}
}

func TestCountNamespaceSummaryObjectsIgnoresNamespaceDefaults(t *testing.T) {
	tests := []struct {
		name     string
		resource NamespaceResourceDescriptor
		objects  []any
		want     int
	}{
		{
			name:     "configmap ignores kube root ca",
			resource: NamespaceResourceDescriptor{Version: "v1", Resource: "configmaps"},
			objects: []any{
				map[string]any{"metadata": map[string]any{"name": "kube-root-ca.crt"}},
				map[string]any{"metadata": map[string]any{"name": "custom-config"}},
			},
			want: 1,
		},
		{
			name:     "serviceaccount ignores default",
			resource: NamespaceResourceDescriptor{Version: "v1", Resource: "serviceaccounts"},
			objects: []any{
				map[string]any{"metadata": map[string]any{"name": "default"}},
				map[string]any{"metadata": map[string]any{"name": "builder"}},
			},
			want: 1,
		},
		{
			name:     "secret ignores default token secret",
			resource: NamespaceResourceDescriptor{Version: "v1", Resource: "secrets"},
			objects: []any{
				map[string]any{"type": "kubernetes.io/service-account-token", "metadata": map[string]any{"name": "default-token-abcde", "annotations": map[string]any{"kubernetes.io/service-account.name": "default"}}},
				map[string]any{"metadata": map[string]any{"name": "custom-secret"}, "type": "Opaque"},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := namespaceSummarySpec{resource: tt.resource}
			if got := countNamespaceSummaryObjects(spec, tt.objects); got != tt.want {
				t.Fatalf("countNamespaceSummaryObjects() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNamespaceSummaryServiceAggregatesAndLimitsConcurrentLists(t *testing.T) {
	runtime := &namespaceSummaryRuntimeSpy{
		resources: []NamespaceAPIResourceList{{GroupVersion: "v1", Resources: []NamespaceAPIResource{
			{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: []string{"list"}},
			{Name: "services", Kind: "Service", Namespaced: true, Verbs: []string{"list"}},
			{Name: "secrets", Kind: "Secret", Namespaced: true, Verbs: []string{"list"}},
		}}},
		objects: map[string][]any{
			"pods":     {map[string]any{"metadata": map[string]any{"name": "api"}}},
			"services": {map[string]any{"metadata": map[string]any{"name": "api"}}, map[string]any{"metadata": map[string]any{"name": "web"}}},
			"secrets":  {map[string]any{"metadata": map[string]any{"name": "default-token-abc"}}, map[string]any{"metadata": map[string]any{"name": "app-token"}}},
		},
	}
	result, err := NewNamespaceSummaryService(runtime).Summary(context.Background(), 7, " dev ")
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if result.Total != 4 || runtime.namespace != "dev" {
		t.Fatalf("summary = %#v namespace=%q", result, runtime.namespace)
	}
	if got, want := result.Items, []NamespaceResourceSummaryItem{{Key: "pods|pod", Label: "Pod", Count: 1}, {Key: "secrets|secret", Label: "Secret", Count: 1}, {Key: "services|service", Label: "Service", Count: 2}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("items = %#v, want %#v", got, want)
	}
	if runtime.maxActive > namespaceSummaryConcurrency {
		t.Fatalf("max concurrent lists = %d, limit = %d", runtime.maxActive, namespaceSummaryConcurrency)
	}
}

func TestNamespaceSummaryServiceReturnsListError(t *testing.T) {
	runtime := &namespaceSummaryRuntimeSpy{
		resources: []NamespaceAPIResourceList{{GroupVersion: "v1", Resources: []NamespaceAPIResource{{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: []string{"list"}}}}},
		listErr:   errors.New("list failed"),
	}
	_, err := NewNamespaceSummaryService(runtime).Summary(context.Background(), 7, "dev")
	if !errors.Is(err, runtime.listErr) {
		t.Fatalf("Summary() error = %v, want %v", err, runtime.listErr)
	}
}

func TestNamespaceSummaryServiceUsesPartialDiscoveryResults(t *testing.T) {
	discoveryErr := errors.New("one API group is unavailable")
	runtime := &namespaceSummaryRuntimeSpy{
		resources:    []NamespaceAPIResourceList{{GroupVersion: "v1", Resources: []NamespaceAPIResource{{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: []string{"list"}}}}},
		objects:      map[string][]any{"pods": {map[string]any{"metadata": map[string]any{"name": "api"}}}},
		discoveryErr: discoveryErr,
	}
	result, err := NewNamespaceSummaryService(runtime).Summary(context.Background(), 7, "dev")
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("summary = %#v, want one discovered resource", result)
	}
}

func TestNamespaceSummaryServiceBoundsConcurrentLists(t *testing.T) {
	resources := make([]NamespaceAPIResource, 0, namespaceSummaryConcurrency+1)
	objects := make(map[string][]any, namespaceSummaryConcurrency+1)
	for index := 0; index <= namespaceSummaryConcurrency; index++ {
		name := "resource" + string(rune('a'+index))
		resources = append(resources, NamespaceAPIResource{Name: name, Kind: "Kind" + string(rune('A'+index)), Namespaced: true, Verbs: []string{"list"}})
		objects[name] = []any{}
	}
	started := make(chan struct{}, namespaceSummaryConcurrency+1)
	release := make(chan struct{})
	runtime := &namespaceSummaryRuntimeSpy{
		resources: []NamespaceAPIResourceList{{GroupVersion: "v1", Resources: resources}},
		objects:   objects,
		started:   started,
		release:   release,
	}
	done := make(chan error, 1)
	go func() {
		_, err := NewNamespaceSummaryService(runtime).Summary(context.Background(), 7, "dev")
		done <- err
	}()
	for index := 0; index < namespaceSummaryConcurrency; index++ {
		<-started
	}
	select {
	case <-started:
		t.Fatal("started more list calls than the configured concurrency limit")
	default:
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if runtime.maxActive != namespaceSummaryConcurrency {
		t.Fatalf("max concurrent lists = %d, want %d", runtime.maxActive, namespaceSummaryConcurrency)
	}
}

type namespaceSummaryRuntimeSpy struct {
	resources    []NamespaceAPIResourceList
	objects      map[string][]any
	discoveryErr error
	listErr      error
	started      chan<- struct{}
	release      <-chan struct{}

	mu        sync.Mutex
	namespace string
	active    int
	maxActive int
}

func (spy *namespaceSummaryRuntimeSpy) PreferredResources(context.Context, uint64) ([]NamespaceAPIResourceList, error) {
	return spy.resources, spy.discoveryErr
}

func (spy *namespaceSummaryRuntimeSpy) ListNamespaceResources(_ context.Context, _ uint64, resource NamespaceResourceDescriptor, namespace string) ([]any, error) {
	spy.mu.Lock()
	spy.namespace = namespace
	spy.active++
	if spy.active > spy.maxActive {
		spy.maxActive = spy.active
	}
	spy.mu.Unlock()
	if spy.started != nil {
		spy.started <- struct{}{}
		<-spy.release
	}
	spy.mu.Lock()
	spy.active--
	spy.mu.Unlock()
	if spy.listErr != nil {
		return nil, spy.listErr
	}
	return spy.objects[resource.Resource], nil
}
