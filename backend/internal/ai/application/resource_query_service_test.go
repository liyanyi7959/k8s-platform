package application

import (
	"context"
	"testing"

	"k8s-platform-backend/internal/ai/domain"
)

type resourceQueryRuntimeStub struct {
	listCalls []resourceQueryListCall
	lists     map[string][]map[string]any
	objects   map[string]map[string]any
	events    []map[string]any
	logs      string
}

type resourceQueryListCall struct {
	resource  ResourceReference
	namespace string
	options   map[string]string
}

func (s *resourceQueryRuntimeStub) List(_ context.Context, _ uint64, resource ResourceReference, namespace, _, _ string, options map[string]string) ([]map[string]any, error) {
	s.listCalls = append(s.listCalls, resourceQueryListCall{resource: resource, namespace: namespace, options: options})
	return s.lists[resource.Resource], nil
}

func (s *resourceQueryRuntimeStub) Get(_ context.Context, _ uint64, resource ResourceReference, _, name string) (map[string]any, error) {
	return s.objects[resource.Resource+"/"+name], nil
}

func (s *resourceQueryRuntimeStub) ListNodeEvents(context.Context, uint64, string) ([]map[string]any, error) {
	return s.events, nil
}

func (s *resourceQueryRuntimeStub) PodLogs(context.Context, uint64, string, string, string, int64, bool) (string, error) {
	return s.logs, nil
}

type resourceQueryPresenterStub struct{}

func (resourceQueryPresenterStub) ListItemSummary(kind string, object map[string]any) domain.JSONMap {
	return domain.JSONMap{"kind": kind, "namespace": objectMetaString(object, "namespace"), "name": objectMetaString(object, "name")}
}

func TestResourceQueryListNormalizesClusterScopeAndLimit(t *testing.T) {
	runtime := &resourceQueryRuntimeStub{lists: map[string][]map[string]any{"nodes": {
		resourceQueryObject("", "node-b"), resourceQueryObject("", "node-a"),
	}}}
	service := NewResourceQueryService(runtime, resourceQueryPresenterStub{})
	result, err := service.ListResources(context.Background(), 7, "Node", "ignored", "team=ops", "", "", 1)
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	if got := runtime.listCalls[0].namespace; got != "" {
		t.Fatalf("cluster-scoped list namespace = %q, want empty", got)
	}
	if got := result.Evidence["returned"]; got != 1 {
		t.Fatalf("returned = %v, want 1", got)
	}
	if got := result.Evidence["label_selector"]; got != "team=ops" {
		t.Fatalf("label selector = %v", got)
	}
}

func TestResourceQuerySearchSortsByKindNamespaceAndName(t *testing.T) {
	runtime := &resourceQueryRuntimeStub{lists: map[string][]map[string]any{
		"pods":        {resourceQueryObject("team-b", "api"), resourceQueryObject("team-a", "worker")},
		"deployments": {resourceQueryObject("team-a", "api")},
	}}
	service := NewResourceQueryService(runtime, resourceQueryPresenterStub{})
	result, err := service.SearchResources(context.Background(), 7, "a", "", []string{"Pod", "Deployment"}, 10)
	if err != nil {
		t.Fatalf("SearchResources() error = %v", err)
	}
	items := result.Evidence["items"].([]domain.JSONMap)
	if len(items) != 2 || items[0]["kind"] != "Deployment" || items[1]["kind"] != "Pod" {
		t.Fatalf("items = %#v, want sorted deployment then pod", items)
	}
}

func TestResourceQueryEventsUseObjectUIDSelector(t *testing.T) {
	runtime := &resourceQueryRuntimeStub{
		lists:   map[string][]map[string]any{"events": {{"metadata": map[string]any{"name": "event-1"}}}, "pods": nil},
		objects: map[string]map[string]any{"pods/api": {"metadata": map[string]any{"uid": "pod-uid"}}},
	}
	service := NewResourceQueryService(runtime, resourceQueryPresenterStub{})
	result, err := service.GetResourceEvents(context.Background(), 7, "Pod", "team-a", "api")
	if err != nil {
		t.Fatalf("GetResourceEvents() error = %v", err)
	}
	if got := runtime.listCalls[0].options["field_selector"]; got != "involvedObject.kind=Pod,involvedObject.name=api,involvedObject.namespace=team-a,involvedObject.uid=pod-uid" {
		t.Fatalf("field selector = %q", got)
	}
	if got := result.RawRef["source"]; got != "resource.events" {
		t.Fatalf("source = %v", got)
	}
}

func TestResourceQueryFindsSecretConsumers(t *testing.T) {
	runtime := &resourceQueryRuntimeStub{lists: map[string][]map[string]any{"pods": {
		{"metadata": map[string]any{"name": "api", "namespace": "team-a", "ownerReferences": []any{map[string]any{"kind": "Deployment", "name": "api"}}}, "spec": map[string]any{"containers": []any{map[string]any{"envFrom": []any{map[string]any{"secretRef": map[string]any{"name": "db"}}}}}}},
	}}}
	service := NewResourceQueryService(runtime, resourceQueryPresenterStub{})
	result, err := service.GetRelatedResources(context.Background(), 7, "Secret", "team-a", "db")
	if err != nil {
		t.Fatalf("GetRelatedResources() error = %v", err)
	}
	if got := result.Evidence["pod_count"]; got != 1 {
		t.Fatalf("pod_count = %v", got)
	}
	controllers := result.Evidence["controllers"].([]domain.JSONMap)
	if len(controllers) != 1 || controllers[0]["name"] != "api" {
		t.Fatalf("controllers = %#v", controllers)
	}
}

func resourceQueryObject(namespace, name string) map[string]any {
	return map[string]any{"metadata": map[string]any{"namespace": namespace, "name": name}}
}
