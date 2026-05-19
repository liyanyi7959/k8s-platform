package service

import (
	"context"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type NamespaceResourceSummaryItem struct {
	Key   string
	Label string
	Count int
}

var namespaceSummarySpecs = []struct {
	key   string
	label string
	gvr   schema.GroupVersionResource
}{
	{key: "deployments", label: "Deployment", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}},
	{key: "statefulsets", label: "StatefulSet", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}},
	{key: "daemonsets", label: "DaemonSet", gvr: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}},
	{key: "pods", label: "Pod", gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}},
	{key: "services", label: "Service", gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}},
	{key: "ingresses", label: "Ingress", gvr: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}},
	{key: "configmaps", label: "ConfigMap", gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}},
	{key: "secrets", label: "Secret", gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}},
	{key: "pvcs", label: "PVC", gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}},
	{key: "serviceaccounts", label: "ServiceAccount", gvr: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}},
	{key: "roles", label: "Role", gvr: schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"}},
	{key: "rolebindings", label: "RoleBinding", gvr: schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}},
	{key: "jobs", label: "Job", gvr: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}},
	{key: "cronjobs", label: "CronJob", gvr: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}},
}

func (s *K8sService) GetNamespaceResourcesSummary(ctx context.Context, clusterID uint64, namespace string) ([]NamespaceResourceSummaryItem, int, error) {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return nil, 0, ErrInvalidParams
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	items := make([]NamespaceResourceSummaryItem, len(namespaceSummarySpecs))
	var firstErr error
	var errMu sync.Mutex
	var wg sync.WaitGroup

	for idx, spec := range namespaceSummarySpecs {
		wg.Add(1)
		go func(index int, spec struct {
			key   string
			label string
			gvr   schema.GroupVersionResource
		}) {
			defer wg.Done()
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
			items[index] = NamespaceResourceSummaryItem{Key: spec.key, Label: spec.label, Count: len(list)}
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
