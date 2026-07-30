package router

import (
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
)

func registerClusterResourceRoutes(a k8sRouteArgs) {
	k8s, namespace, metrics, connectivity, nodes, platform, inspection, storage, p := a.k8s, a.namespace, a.metrics, a.connectivity, a.nodes, a.platform, a.inspection, a.storage, a.perm

	k8s.GET("/clusters/:id/namespaces", p.namespaceRead, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), namespace.List)
	k8s.POST("/clusters/:id/namespaces", p.namespaceWrite, namespace.Create)
	k8s.DELETE("/clusters/:id/namespaces/:ns", p.namespaceWrite, namespace.Delete)
	k8s.GET("/clusters/:id/namespaces/:ns/yaml", p.namespaceRead, namespace.YAML)
	k8s.GET("/clusters/:id/namespaces/:ns/resource-summary", p.namespaceRead, namespace.Summary)
	k8s.GET("/clusters/:id/namespaces/:ns/inspection", p.namespaceRead, inspection.Namespace)
	k8s.GET("/clusters/:id/namespaces/:ns/workload-inventory", p.namespaceRead, inspection.NamespaceWorkloadInventory)

	k8s.GET("/clusters/:id/nodes", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), nodes.List)
	k8s.GET("/clusters/:id/nodes/:name", p.read, nodes.Detail)
	k8s.GET("/clusters/:id/nodes/:name/yaml", p.read, nodes.YAML)
	k8s.GET("/clusters/:id/nodes/:name/pods", p.read, nodes.Pods)
	k8s.GET("/clusters/:id/nodes/:name/events", p.read, nodes.Events)
	k8s.POST("/clusters/:id/nodes/:name/cordon-requests", p.write, nodes.Cordon)
	k8s.POST("/clusters/:id/nodes/:name/uncordon-requests", p.write, nodes.Uncordon)
	k8s.POST("/clusters/:id/nodes/:name/drain-requests", p.write, nodes.Drain)
	k8s.DELETE("/clusters/:id/nodes/:name", p.write, nodes.Delete)

	k8s.GET("/clusters/:id/hpas", p.read, platform.List(kopsapp.PlatformHorizontalPodAutoscaler))
	k8s.DELETE("/clusters/:id/hpas/:ns/:name", p.write, platform.Delete(kopsapp.PlatformHorizontalPodAutoscaler))
	k8s.GET("/clusters/:id/hpas/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformHorizontalPodAutoscaler))

	k8s.GET("/clusters/:id/pdbs", p.read, platform.List(kopsapp.PlatformPodDisruptionBudget))
	k8s.DELETE("/clusters/:id/pdbs/:ns/:name", p.write, platform.Delete(kopsapp.PlatformPodDisruptionBudget))
	k8s.GET("/clusters/:id/pdbs/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformPodDisruptionBudget))

	k8s.GET("/clusters/:id/events", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), namespace.Events)

	k8s.GET("/clusters/:id/leases", p.read, connectivity.ListLeases)
	k8s.DELETE("/clusters/:id/leases/:ns/:name", p.write, connectivity.DeleteLease)
	k8s.GET("/clusters/:id/leases/:ns/:name/yaml", p.read, connectivity.LeaseYAML)

	k8s.GET("/clusters/:id/resourcequotas", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformResourceQuota))
	k8s.DELETE("/clusters/:id/resourcequotas/:ns/:name", p.write, platform.Delete(kopsapp.PlatformResourceQuota))
	k8s.GET("/clusters/:id/resourcequotas/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformResourceQuota))

	k8s.GET("/clusters/:id/limitranges", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformLimitRange))
	k8s.DELETE("/clusters/:id/limitranges/:ns/:name", p.write, platform.Delete(kopsapp.PlatformLimitRange))
	k8s.GET("/clusters/:id/limitranges/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformLimitRange))

	k8s.GET("/clusters/:id/customresourcedefinitions", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformCustomResourceDefinition))
	k8s.DELETE("/clusters/:id/customresourcedefinitions/:name", p.write, platform.Delete(kopsapp.PlatformCustomResourceDefinition))
	k8s.GET("/clusters/:id/customresourcedefinitions/:name/yaml", p.read, platform.YAML(kopsapp.PlatformCustomResourceDefinition))

	k8s.GET("/clusters/:id/apiservices", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformAPIService))
	k8s.DELETE("/clusters/:id/apiservices/:name", p.write, platform.Delete(kopsapp.PlatformAPIService))
	k8s.GET("/clusters/:id/apiservices/:name/yaml", p.read, platform.YAML(kopsapp.PlatformAPIService))

	k8s.GET("/clusters/:id/priorityclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformPriorityClass))
	k8s.DELETE("/clusters/:id/priorityclasses/:name", p.write, platform.Delete(kopsapp.PlatformPriorityClass))
	k8s.GET("/clusters/:id/priorityclasses/:name/yaml", p.read, platform.YAML(kopsapp.PlatformPriorityClass))

	k8s.GET("/clusters/:id/runtimeclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformRuntimeClass))
	k8s.DELETE("/clusters/:id/runtimeclasses/:name", p.write, platform.Delete(kopsapp.PlatformRuntimeClass))
	k8s.GET("/clusters/:id/runtimeclasses/:name/yaml", p.read, platform.YAML(kopsapp.PlatformRuntimeClass))

	k8s.GET("/clusters/:id/validatingwebhookconfigurations", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformValidatingWebhookConfiguration))
	k8s.DELETE("/clusters/:id/validatingwebhookconfigurations/:name", p.write, platform.Delete(kopsapp.PlatformValidatingWebhookConfiguration))
	k8s.GET("/clusters/:id/validatingwebhookconfigurations/:name/yaml", p.read, platform.YAML(kopsapp.PlatformValidatingWebhookConfiguration))

	k8s.GET("/clusters/:id/mutatingwebhookconfigurations", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformMutatingWebhookConfiguration))
	k8s.DELETE("/clusters/:id/mutatingwebhookconfigurations/:name", p.write, platform.Delete(kopsapp.PlatformMutatingWebhookConfiguration))
	k8s.GET("/clusters/:id/mutatingwebhookconfigurations/:name/yaml", p.read, platform.YAML(kopsapp.PlatformMutatingWebhookConfiguration))

	k8s.GET("/clusters/:id/validatingadmissionpolicies", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformValidatingAdmissionPolicy))
	k8s.DELETE("/clusters/:id/validatingadmissionpolicies/:name", p.write, platform.Delete(kopsapp.PlatformValidatingAdmissionPolicy))
	k8s.GET("/clusters/:id/validatingadmissionpolicies/:name/yaml", p.read, platform.YAML(kopsapp.PlatformValidatingAdmissionPolicy))

	k8s.GET("/clusters/:id/validatingadmissionpolicybindings", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformValidatingAdmissionPolicyBinding))
	k8s.DELETE("/clusters/:id/validatingadmissionpolicybindings/:name", p.write, platform.Delete(kopsapp.PlatformValidatingAdmissionPolicyBinding))
	k8s.GET("/clusters/:id/validatingadmissionpolicybindings/:name/yaml", p.read, platform.YAML(kopsapp.PlatformValidatingAdmissionPolicyBinding))

	k8s.GET("/clusters/:id/capabilities", p.resourceSupportRead, storage.ResourceSupport)

	k8s.GET("/clusters/:id/nodes/metrics", p.read, metrics.NodeMetrics)
	k8s.GET("/clusters/:id/metrics/source", p.read, metrics.Source)
	k8s.GET("/clusters/:id/metrics/trend", p.read, metrics.Trend)
	k8s.POST("/clusters/:id/metrics/source-detection-runs", p.write, metrics.Detect)
	k8s.POST("/clusters/:id/metrics/source-change-requests", p.write, metrics.Switch)
	k8s.POST("/clusters/:id/metrics/health-checks", p.read, metrics.HealthCheck)
}
