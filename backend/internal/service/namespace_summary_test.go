package service

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestBuildNamespaceSummarySpecs_FiltersAndDeduplicates(t *testing.T) {
	resourceLists := []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: metav1.Verbs{"get", "list"}},
				{Name: "pods/status", Kind: "Pod", Namespaced: true, Verbs: metav1.Verbs{"get"}},
				{Name: "events", Kind: "Event", Namespaced: true, Verbs: metav1.Verbs{"list"}},
			},
		},
		{
			GroupVersion: "events.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "events", Kind: "Event", Namespaced: true, Verbs: metav1.Verbs{"list"}},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: metav1.Verbs{"list"}},
				{Name: "controllerrevisions", Kind: "ControllerRevision", Namespaced: true, Verbs: metav1.Verbs{"get"}},
			},
		},
		{
			GroupVersion: "rbac.authorization.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "clusterroles", Kind: "ClusterRole", Namespaced: false, Verbs: metav1.Verbs{"list"}},
			},
		},
	}

	specs := buildNamespaceSummarySpecs(resourceLists)
	got := make([]string, 0, len(specs))
	for _, spec := range specs {
		got = append(got, spec.label+":"+spec.gvr.Group+"/"+spec.gvr.Version+"/"+spec.gvr.Resource)
	}
	want := []string{
		"Deployment:apps/v1/deployments",
		"Event:/v1/events",
		"Pod:/v1/pods",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("specs = %v, want %v", got, want)
	}
}

func TestCountNamespaceSummaryObjects_IgnoreNamespaceDefaults(t *testing.T) {
	tests := []struct {
		name string
		spec namespaceSummarySpec
		list []any
		want int
	}{
		{
			name: "configmap ignores kube root ca",
			spec: namespaceSummarySpec{gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}},
			list: []any{
				map[string]any{"metadata": map[string]any{"name": "kube-root-ca.crt"}},
				map[string]any{"metadata": map[string]any{"name": "custom-config"}},
			},
			want: 1,
		},
		{
			name: "serviceaccount ignores default",
			spec: namespaceSummarySpec{gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}},
			list: []any{
				map[string]any{"metadata": map[string]any{"name": "default"}},
				map[string]any{"metadata": map[string]any{"name": "builder"}},
			},
			want: 1,
		},
		{
			name: "secret ignores default token secret",
			spec: namespaceSummarySpec{gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}},
			list: []any{
				map[string]any{
					"type": "kubernetes.io/service-account-token",
					"metadata": map[string]any{
						"name":        "default-token-abcde",
						"annotations": map[string]any{"kubernetes.io/service-account.name": "default"},
					},
				},
				map[string]any{"metadata": map[string]any{"name": "custom-secret"}, "type": "Opaque"},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countNamespaceSummaryObjects(tt.spec, tt.list); got != tt.want {
				t.Fatalf("countNamespaceSummaryObjects() = %d, want %d", got, tt.want)
			}
		})
	}
}
