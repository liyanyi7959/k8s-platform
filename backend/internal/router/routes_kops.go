package router

import (
	"github.com/gin-gonic/gin"

	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/controller"
	"k8s-platform-backend/internal/middleware"
)

type k8sPerms struct {
	read                   gin.HandlerFunc
	write                  gin.HandlerFunc
	namespaceRead          gin.HandlerFunc
	namespaceWrite         gin.HandlerFunc
	secretReveal           gin.HandlerFunc
	rbacRead               gin.HandlerFunc
	rbacWrite              gin.HandlerFunc
	exec                   gin.HandlerFunc
	resourceSupportRead    gin.HandlerFunc
	storageSnapshotSupport gin.HandlerFunc // 与 resourceSupportRead 相同，独立字段便于后续扩展
}

// k8sRouteArgs 封装子路由注册函数所需的全部依赖。
type k8sRouteArgs struct {
	k8s           *gin.RouterGroup
	d             Deps
	ctl           *controller.K8sController
	manifest      *kopshttp.ManifestController
	namespace     *kopshttp.NamespaceController
	metrics       *kopshttp.MetricsController
	connectivity  *kopshttp.ConnectivityController
	nodes         *kopshttp.NodeController
	platform      *kopshttp.PlatformResourceController
	relationships *kopshttp.RelationshipResourceController
	batch         *kopshttp.BatchController
	network       *kopshttp.NetworkController
	configuration *kopshttp.ConfigurationController
	storage       *kopshttp.StorageController
	helm          *kopshttp.HelmController
	workloads     *kopshttp.WorkloadController
	pods          *kopshttp.PodController
	inspection    *kopshttp.InspectionController
	creator       *kopshttp.ResourceCreatorController
	perm          k8sPerms
}

func registerK8sRoutes(authed *gin.RouterGroup, d Deps, ctl *controller.K8sController, manifest *kopshttp.ManifestController, namespace *kopshttp.NamespaceController, metrics *kopshttp.MetricsController, connectivity *kopshttp.ConnectivityController, nodes *kopshttp.NodeController, platform *kopshttp.PlatformResourceController, relationships *kopshttp.RelationshipResourceController, batch *kopshttp.BatchController, network *kopshttp.NetworkController, configuration *kopshttp.ConfigurationController, storage *kopshttp.StorageController, helm *kopshttp.HelmController, workloads *kopshttp.WorkloadController, pods *kopshttp.PodController, inspection *kopshttp.InspectionController, creators ...*kopshttp.ResourceCreatorController) {
	if ctl == nil || manifest == nil || namespace == nil || metrics == nil || connectivity == nil || nodes == nil || platform == nil || relationships == nil || batch == nil || network == nil || configuration == nil || storage == nil || helm == nil || workloads == nil || pods == nil || inspection == nil {
		return
	}
	creator := &kopshttp.ResourceCreatorController{}
	if len(creators) > 0 && creators[0] != nil {
		creator = creators[0]
	}
	k8s := authed.Group("")
	resourceSupportReadPerm := middleware.RequireAnyPerm("k8s:read", "k8s:rbac_read")
	args := k8sRouteArgs{
		k8s: k8s, d: d, ctl: ctl, manifest: manifest, namespace: namespace, metrics: metrics, connectivity: connectivity, nodes: nodes, platform: platform, relationships: relationships, batch: batch, network: network, configuration: configuration, storage: storage, helm: helm, workloads: workloads, pods: pods, inspection: inspection, creator: creator,
		perm: k8sPerms{
			read:                   middleware.RequirePerm("k8s:read"),
			write:                  middleware.RequirePerm("k8s:write"),
			namespaceRead:          middleware.RequireAnyPerm("namespace:read", "k8s:read"),
			namespaceWrite:         middleware.RequireAnyPerm("namespace:write", "k8s:write"),
			secretReveal:           middleware.RequirePerm("k8s:secret_reveal"),
			rbacRead:               middleware.RequirePerm("k8s:rbac_read"),
			rbacWrite:              middleware.RequirePerm("k8s:rbac_write"),
			exec:                   middleware.RequirePerm("k8s:exec"),
			resourceSupportRead:    resourceSupportReadPerm,
			storageSnapshotSupport: resourceSupportReadPerm,
		},
	}

	registerClusterResourceRoutes(args)
	registerWorkloadRoutes(args)
	registerNetworkingRoutes(args)
	registerConfigStorageRoutes(args)
	registerRBACRoutes(args)
	registerBatchRoutes(args)
	registerHelmRoutes(args)
	registerCanonicalResourceUpdateRoutes(args)
}

func registerCanonicalResourceUpdateRoutes(a k8sRouteArgs) {
	k8s, ctl, connectivity, platform, relationships, batch, network, configuration, storage, p := a.k8s, a.ctl, a.connectivity, a.platform, a.relationships, a.batch, a.network, a.configuration, a.storage, a.perm
	namespaced := []struct {
		resource string
		handler  gin.HandlerFunc
	}{
		{"hpas", platform.Apply(kopsapp.PlatformHorizontalPodAutoscaler)}, {"pdbs", platform.Apply(kopsapp.PlatformPodDisruptionBudget)}, {"leases", connectivity.EditLease},
		{"resourcequotas", platform.Apply(kopsapp.PlatformResourceQuota)}, {"limitranges", platform.Apply(kopsapp.PlatformLimitRange)},
		{"replicasets", relationships.Apply(kopsapp.RelationshipReplicaSet)}, {"services", network.EditService}, {"ingresses", network.EditIngress},
		{"networkpolicies", platform.Apply(kopsapp.PlatformNetworkPolicy)}, {"endpoints", connectivity.EditEndpoints}, {"endpointslices", connectivity.EditEndpointSlice},
		{"configmaps", configuration.EditConfigMap}, {"secrets", configuration.EditSecret}, {"serviceaccounts", platform.Apply(kopsapp.PlatformServiceAccount)},
		{"csistoragecapacities", platform.Apply(kopsapp.PlatformCSIStorageCapacity)}, {"volumesnapshots", storage.Apply(kopsapp.StorageVolumeSnapshot)},
		{"jobs", batch.EditJob}, {"cronjobs", batch.EditCronJob},
	}
	for _, route := range namespaced {
		k8s.PATCH("/clusters/:id/"+route.resource+"/:ns/:name", p.write, route.handler)
	}
	clusterScoped := []struct {
		resource string
		handler  gin.HandlerFunc
	}{
		{"customresourcedefinitions", platform.Apply(kopsapp.PlatformCustomResourceDefinition)}, {"apiservices", platform.Apply(kopsapp.PlatformAPIService)},
		{"priorityclasses", platform.Apply(kopsapp.PlatformPriorityClass)}, {"runtimeclasses", platform.Apply(kopsapp.PlatformRuntimeClass)},
		{"validatingwebhookconfigurations", platform.Apply(kopsapp.PlatformValidatingWebhookConfiguration)},
		{"mutatingwebhookconfigurations", platform.Apply(kopsapp.PlatformMutatingWebhookConfiguration)},
		{"validatingadmissionpolicies", platform.Apply(kopsapp.PlatformValidatingAdmissionPolicy)},
		{"validatingadmissionpolicybindings", platform.Apply(kopsapp.PlatformValidatingAdmissionPolicyBinding)},
		{"ingressclasses", network.EditIngressClass}, {"storageclasses", storage.Apply(kopsapp.StorageClass)},
		{"csidrivers", platform.Apply(kopsapp.PlatformCSIDriver)}, {"csinodes", platform.Apply(kopsapp.PlatformCSINode)},
		{"volumeattachments", relationships.Apply(kopsapp.RelationshipVolumeAttachment)}, {"volumesnapshotclasses", storage.Apply(kopsapp.StorageVolumeSnapshotClass)},
		{"volumesnapshotcontents", storage.Apply(kopsapp.StorageVolumeSnapshotContent)},
	}
	for _, route := range clusterScoped {
		k8s.PATCH("/clusters/:id/"+route.resource+"/:name", p.write, route.handler)
	}
	k8s.PATCH("/clusters/:id/roles/:ns/:name", p.rbacWrite, platform.Apply(kopsapp.PlatformRole))
	k8s.PATCH("/clusters/:id/clusterroles/:name", p.rbacWrite, platform.Apply(kopsapp.PlatformClusterRole))
	k8s.PATCH("/clusters/:id/rolebindings/:ns/:name", p.rbacWrite, platform.Apply(kopsapp.PlatformRoleBinding))
	k8s.PATCH("/clusters/:id/clusterrolebindings/:name", p.rbacWrite, platform.Apply(kopsapp.PlatformClusterRoleBinding))
	k8s.PATCH("/clusters/:id/workloads/deployments/:ns/:name", p.write, ctl.EditDeployment)
	k8s.PATCH("/clusters/:id/workloads/statefulsets/:ns/:name", p.write, ctl.EditStatefulSet)
	k8s.PATCH("/clusters/:id/workloads/daemonsets/:ns/:name", p.write, ctl.EditDaemonSet)
	k8s.PATCH("/clusters/:id/workloads/:kind/:ns/:name/yaml", p.write, ctl.EditWorkloadYAML)
	k8s.POST("/clusters/:id/workloads/:kind/:ns/:name/scale-operations", p.write, ctl.ScaleWorkload)
	k8s.POST("/clusters/:id/workloads/:kind/:ns/:name/restart-operations", p.write, ctl.RestartWorkload)
}

// ── 集群级资源：Namespace / Node / HPA / PDB / Event / CRD / APIService / PriorityClass / RuntimeClass / Webhook / Lease ──

func registerClusterResourceRoutes(a k8sRouteArgs) {
	k8s, namespace, metrics, connectivity, nodes, platform, inspection, storage, p := a.k8s, a.namespace, a.metrics, a.connectivity, a.nodes, a.platform, a.inspection, a.storage, a.perm

	// Namespace
	k8s.GET("/clusters/:id/namespaces", p.namespaceRead, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), namespace.List)
	k8s.POST("/clusters/:id/namespaces", p.namespaceWrite, namespace.Create)
	k8s.DELETE("/clusters/:id/namespaces/:ns", p.namespaceWrite, namespace.Delete)
	k8s.GET("/clusters/:id/namespaces/:ns/yaml", p.namespaceRead, namespace.YAML)
	k8s.GET("/clusters/:id/namespaces/:ns/resources-summary", p.namespaceRead, namespace.Summary)
	k8s.GET("/clusters/:id/namespaces/:ns/inspection", p.namespaceRead, inspection.Namespace)
	k8s.GET("/clusters/:id/namespaces/:ns/workload-inventory", p.namespaceRead, inspection.NamespaceWorkloadInventory)

	// Node
	k8s.GET("/clusters/:id/nodes", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), nodes.List)
	k8s.GET("/clusters/:id/nodes/:name/detail", p.read, nodes.Detail)
	k8s.GET("/clusters/:id/nodes/:name/yaml", p.read, nodes.YAML)
	k8s.GET("/clusters/:id/nodes/:name/pods", p.read, nodes.Pods)
	k8s.GET("/clusters/:id/nodes/:name/events", p.read, nodes.Events)
	k8s.POST("/clusters/:id/nodes/:name/cordon", p.write, nodes.Cordon)
	k8s.POST("/clusters/:id/nodes/:name/uncordon", p.write, nodes.Uncordon)
	k8s.POST("/clusters/:id/nodes/:name/drain", p.write, nodes.Drain)
	k8s.POST("/clusters/:id/nodes/:name/cordon-requests", p.write, nodes.Cordon)
	k8s.POST("/clusters/:id/nodes/:name/uncordon-requests", p.write, nodes.Uncordon)
	k8s.POST("/clusters/:id/nodes/:name/drain-requests", p.write, nodes.Drain)
	k8s.DELETE("/clusters/:id/nodes/:name", p.write, nodes.Delete)

	// HPA
	k8s.GET("/clusters/:id/hpas", p.read, platform.List(kopsapp.PlatformHorizontalPodAutoscaler))
	k8s.PATCH("/clusters/:id/hpas/edit", p.write, platform.Apply(kopsapp.PlatformHorizontalPodAutoscaler))
	k8s.DELETE("/clusters/:id/hpas/:ns/:name", p.write, platform.Delete(kopsapp.PlatformHorizontalPodAutoscaler))
	k8s.GET("/clusters/:id/hpas/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformHorizontalPodAutoscaler))

	// PDB
	k8s.GET("/clusters/:id/pdbs", p.read, platform.List(kopsapp.PlatformPodDisruptionBudget))
	k8s.PATCH("/clusters/:id/pdbs/edit", p.write, platform.Apply(kopsapp.PlatformPodDisruptionBudget))
	k8s.DELETE("/clusters/:id/pdbs/:ns/:name", p.write, platform.Delete(kopsapp.PlatformPodDisruptionBudget))
	k8s.GET("/clusters/:id/pdbs/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformPodDisruptionBudget))

	// Event
	k8s.GET("/clusters/:id/events", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), namespace.Events)

	// Lease
	k8s.GET("/clusters/:id/leases", p.read, connectivity.ListLeases)
	k8s.PATCH("/clusters/:id/leases/edit", p.write, connectivity.EditLease)
	k8s.DELETE("/clusters/:id/leases/:ns/:name", p.write, connectivity.DeleteLease)
	k8s.GET("/clusters/:id/leases/:ns/:name/yaml", p.read, connectivity.LeaseYAML)

	// ResourceQuota
	k8s.GET("/clusters/:id/resourcequotas", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformResourceQuota))
	k8s.PATCH("/clusters/:id/resourcequotas/edit", p.write, platform.Apply(kopsapp.PlatformResourceQuota))
	k8s.DELETE("/clusters/:id/resourcequotas/:ns/:name", p.write, platform.Delete(kopsapp.PlatformResourceQuota))
	k8s.GET("/clusters/:id/resourcequotas/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformResourceQuota))

	// LimitRange
	k8s.GET("/clusters/:id/limitranges", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformLimitRange))
	k8s.PATCH("/clusters/:id/limitranges/edit", p.write, platform.Apply(kopsapp.PlatformLimitRange))
	k8s.DELETE("/clusters/:id/limitranges/:ns/:name", p.write, platform.Delete(kopsapp.PlatformLimitRange))
	k8s.GET("/clusters/:id/limitranges/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformLimitRange))

	// CustomResourceDefinition
	k8s.GET("/clusters/:id/customresourcedefinitions", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformCustomResourceDefinition))
	k8s.PATCH("/clusters/:id/customresourcedefinitions/edit", p.write, platform.Apply(kopsapp.PlatformCustomResourceDefinition))
	k8s.DELETE("/clusters/:id/customresourcedefinitions/:name", p.write, platform.Delete(kopsapp.PlatformCustomResourceDefinition))
	k8s.GET("/clusters/:id/customresourcedefinitions/:name/yaml", p.read, platform.YAML(kopsapp.PlatformCustomResourceDefinition))

	// APIService
	k8s.GET("/clusters/:id/apiservices", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformAPIService))
	k8s.PATCH("/clusters/:id/apiservices/edit", p.write, platform.Apply(kopsapp.PlatformAPIService))
	k8s.DELETE("/clusters/:id/apiservices/:name", p.write, platform.Delete(kopsapp.PlatformAPIService))
	k8s.GET("/clusters/:id/apiservices/:name/yaml", p.read, platform.YAML(kopsapp.PlatformAPIService))

	// PriorityClass
	k8s.GET("/clusters/:id/priorityclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformPriorityClass))
	k8s.PATCH("/clusters/:id/priorityclasses/edit", p.write, platform.Apply(kopsapp.PlatformPriorityClass))
	k8s.DELETE("/clusters/:id/priorityclasses/:name", p.write, platform.Delete(kopsapp.PlatformPriorityClass))
	k8s.GET("/clusters/:id/priorityclasses/:name/yaml", p.read, platform.YAML(kopsapp.PlatformPriorityClass))

	// RuntimeClass
	k8s.GET("/clusters/:id/runtimeclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformRuntimeClass))
	k8s.PATCH("/clusters/:id/runtimeclasses/edit", p.write, platform.Apply(kopsapp.PlatformRuntimeClass))
	k8s.DELETE("/clusters/:id/runtimeclasses/:name", p.write, platform.Delete(kopsapp.PlatformRuntimeClass))
	k8s.GET("/clusters/:id/runtimeclasses/:name/yaml", p.read, platform.YAML(kopsapp.PlatformRuntimeClass))

	// ValidatingWebhookConfiguration
	k8s.GET("/clusters/:id/validatingwebhookconfigurations", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformValidatingWebhookConfiguration))
	k8s.PATCH("/clusters/:id/validatingwebhookconfigurations/edit", p.write, platform.Apply(kopsapp.PlatformValidatingWebhookConfiguration))
	k8s.DELETE("/clusters/:id/validatingwebhookconfigurations/:name", p.write, platform.Delete(kopsapp.PlatformValidatingWebhookConfiguration))
	k8s.GET("/clusters/:id/validatingwebhookconfigurations/:name/yaml", p.read, platform.YAML(kopsapp.PlatformValidatingWebhookConfiguration))

	// MutatingWebhookConfiguration
	k8s.GET("/clusters/:id/mutatingwebhookconfigurations", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformMutatingWebhookConfiguration))
	k8s.PATCH("/clusters/:id/mutatingwebhookconfigurations/edit", p.write, platform.Apply(kopsapp.PlatformMutatingWebhookConfiguration))
	k8s.DELETE("/clusters/:id/mutatingwebhookconfigurations/:name", p.write, platform.Delete(kopsapp.PlatformMutatingWebhookConfiguration))
	k8s.GET("/clusters/:id/mutatingwebhookconfigurations/:name/yaml", p.read, platform.YAML(kopsapp.PlatformMutatingWebhookConfiguration))

	// ValidatingAdmissionPolicy
	k8s.GET("/clusters/:id/validatingadmissionpolicies", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformValidatingAdmissionPolicy))
	k8s.PATCH("/clusters/:id/validatingadmissionpolicies/edit", p.write, platform.Apply(kopsapp.PlatformValidatingAdmissionPolicy))
	k8s.DELETE("/clusters/:id/validatingadmissionpolicies/:name", p.write, platform.Delete(kopsapp.PlatformValidatingAdmissionPolicy))
	k8s.GET("/clusters/:id/validatingadmissionpolicies/:name/yaml", p.read, platform.YAML(kopsapp.PlatformValidatingAdmissionPolicy))

	// ValidatingAdmissionPolicyBinding
	k8s.GET("/clusters/:id/validatingadmissionpolicybindings", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformValidatingAdmissionPolicyBinding))
	k8s.PATCH("/clusters/:id/validatingadmissionpolicybindings/edit", p.write, platform.Apply(kopsapp.PlatformValidatingAdmissionPolicyBinding))
	k8s.DELETE("/clusters/:id/validatingadmissionpolicybindings/:name", p.write, platform.Delete(kopsapp.PlatformValidatingAdmissionPolicyBinding))
	k8s.GET("/clusters/:id/validatingadmissionpolicybindings/:name/yaml", p.read, platform.YAML(kopsapp.PlatformValidatingAdmissionPolicyBinding))

	// Resource support
	k8s.GET("/clusters/:id/resource-support", p.resourceSupportRead, storage.ResourceSupport)
	k8s.GET("/clusters/:id/storage-snapshot-support", p.storageSnapshotSupport, storage.ResourceSupport)

	// 资源使用率监控
	k8s.GET("/clusters/:id/nodes/metrics", p.read, metrics.NodeMetrics)
	k8s.GET("/clusters/:id/metrics/source", p.read, metrics.Source)
	k8s.POST("/clusters/:id/metrics/detect", p.write, metrics.Detect)
	k8s.POST("/clusters/:id/metrics/switch", p.write, metrics.Switch)
	k8s.GET("/clusters/:id/metrics/trend", p.read, metrics.Trend)
	k8s.POST("/clusters/:id/metrics/health-check", p.read, metrics.HealthCheck)
	k8s.POST("/clusters/:id/metrics/source-detection-runs", p.write, metrics.Detect)
	k8s.POST("/clusters/:id/metrics/source-change-requests", p.write, metrics.Switch)
	k8s.POST("/clusters/:id/metrics/health-checks", p.read, metrics.HealthCheck)
}

// ── 工作负载：Pod / Deployment / StatefulSet / DaemonSet / ReplicaSet / Manifest ──

func registerWorkloadRoutes(a k8sRouteArgs) {
	k8s, manifest, metrics, relationships, workloads, pods, inspection, creator, p := a.k8s, a.manifest, a.metrics, a.relationships, a.workloads, a.pods, a.inspection, a.creator, a.perm

	// Pod
	k8s.GET("/clusters/:id/pods", p.read, pods.List)
	k8s.GET("/clusters/:id/podmetrics", p.read, pods.Metrics)
	// 资源使用率监控：Pod 维度 CPU/内存使用量（与上方 /podmetrics 区分，后者返回原始 PodMetrics 资源）
	k8s.GET("/clusters/:id/pods/metrics", p.read, metrics.PodMetrics)
	k8s.GET("/clusters/:id/pods/:ns/:pod/inspection", p.read, inspection.Pod)
	k8s.GET("/clusters/:id/pods/:ns/:pod/yaml", p.read, pods.YAML)
	k8s.GET("/clusters/:id/pods/:ns/:pod/logs", p.read, pods.Logs)
	k8s.POST("/clusters/:id/pods/:ns/:pod/logs/session", p.read, pods.CreateLogSession)
	k8s.POST("/clusters/:id/pods/:ns/:pod/log-sessions", p.read, pods.CreateLogSession)
	k8s.DELETE("/clusters/:id/pods/:ns/:pod", p.write, pods.Delete)
	k8s.POST("/clusters/:id/pods/:ns/:pod/exec", p.exec, pods.CreateExecSession)
	k8s.POST("/clusters/:id/pods/:ns/:pod/exec-sessions", p.exec, pods.CreateExecSession)

	// Manifest
	k8s.GET("/clusters/:id/manifests/records", p.write, manifest.List)
	k8s.GET("/clusters/:id/manifests/records/:recordId", p.write, manifest.Get)
	k8s.POST("/clusters/:id/manifests/apply", p.write, manifest.Apply)
	k8s.POST("/clusters/:id/manifest-applications", p.write, manifest.Apply)

	// Workload (Deployment/StatefulSet/DaemonSet)
	k8s.GET("/clusters/:id/workloads", p.read, workloads.List)
	k8s.GET("/clusters/:id/workloads/deployments/:ns/:name/rollout-history", p.read, workloads.History)
	k8s.POST("/clusters/:id/workloads/deployments/:ns/:name/rollout-undo", p.write, workloads.Undo)
	k8s.POST("/clusters/:id/workloads/deployments/:ns/:name/rollback-attempts", p.write, workloads.Undo)
	k8s.PATCH("/clusters/:id/workloads/scale", p.write, workloads.Scale)
	k8s.PATCH("/clusters/:id/workloads/restart", p.write, workloads.Restart)
	k8s.PATCH("/clusters/:id/workloads/image", p.write, workloads.Image)
	k8s.PATCH("/clusters/:id/workloads/rollout-pause", p.write, workloads.Pause)
	k8s.POST("/clusters/:id/workloads/:kind/:ns/:name/image-updates", p.write, workloads.Image)
	k8s.PATCH("/clusters/:id/workloads/:kind/:ns/:name/pause-state", p.write, workloads.Pause)
	k8s.POST("/clusters/:id/workloads/deployments", p.write, creator.CreateDeployment)
	k8s.POST("/clusters/:id/workloads/statefulsets", p.write, creator.CreateStatefulSet)
	k8s.POST("/clusters/:id/workloads/daemonsets", p.write, creator.CreateDaemonSet)
	k8s.PATCH("/clusters/:id/workloads/deployments/edit", p.write, workloads.Edit(kopsapp.WorkloadDeployment))
	k8s.PATCH("/clusters/:id/workloads/statefulsets/edit", p.write, workloads.Edit(kopsapp.WorkloadStatefulSet))
	k8s.PATCH("/clusters/:id/workloads/daemonsets/edit", p.write, workloads.Edit(kopsapp.WorkloadDaemonSet))
	k8s.PATCH("/clusters/:id/workloads/yaml/edit", p.write, workloads.ApplyYAML)
	k8s.DELETE("/clusters/:id/workloads/:kind/:ns/:name", p.write, workloads.Delete)
	k8s.GET("/clusters/:id/workloads/:kind/:ns/:name/yaml", p.read, workloads.YAML)

	// ReplicaSet
	k8s.GET("/clusters/:id/replicasets", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), relationships.List(kopsapp.RelationshipReplicaSet))
	k8s.PATCH("/clusters/:id/replicasets/edit", p.write, relationships.Apply(kopsapp.RelationshipReplicaSet))
	k8s.DELETE("/clusters/:id/replicasets/:ns/:name", p.write, relationships.Delete(kopsapp.RelationshipReplicaSet))
	k8s.GET("/clusters/:id/replicasets/:ns/:name/yaml", p.read, relationships.YAML(kopsapp.RelationshipReplicaSet))
}

// ── 网络：Service / Ingress / IngressClass / NetworkPolicy / Endpoints / EndpointSlice ──

func registerNetworkingRoutes(a k8sRouteArgs) {
	k8s, network, connectivity, platform, creator, p := a.k8s, a.network, a.connectivity, a.platform, a.creator, a.perm

	// Service
	k8s.GET("/clusters/:id/services", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), network.List(kopsapp.NetworkKubernetesService))
	k8s.POST("/clusters/:id/services", p.write, creator.CreateService)
	k8s.PATCH("/clusters/:id/services/edit", p.write, network.EditService)
	k8s.DELETE("/clusters/:id/services/:ns/:name", p.write, network.Delete(kopsapp.NetworkKubernetesService))
	k8s.GET("/clusters/:id/services/:ns/:name/yaml", p.read, network.YAML(kopsapp.NetworkKubernetesService))

	// Ingress
	k8s.GET("/clusters/:id/ingresses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), network.List(kopsapp.NetworkIngress))
	k8s.POST("/clusters/:id/ingresses", p.write, creator.CreateIngress)
	k8s.PATCH("/clusters/:id/ingresses/edit", p.write, network.EditIngress)
	k8s.DELETE("/clusters/:id/ingresses/:ns/:name", p.write, network.Delete(kopsapp.NetworkIngress))
	k8s.GET("/clusters/:id/ingresses/:ns/:name/yaml", p.read, network.YAML(kopsapp.NetworkIngress))

	// NetworkPolicy
	k8s.GET("/clusters/:id/networkpolicies", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformNetworkPolicy))
	k8s.PATCH("/clusters/:id/networkpolicies/edit", p.write, platform.Apply(kopsapp.PlatformNetworkPolicy))
	k8s.DELETE("/clusters/:id/networkpolicies/:ns/:name", p.write, platform.Delete(kopsapp.PlatformNetworkPolicy))
	k8s.GET("/clusters/:id/networkpolicies/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformNetworkPolicy))

	// IngressClass
	k8s.GET("/clusters/:id/ingressclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), network.List(kopsapp.NetworkIngressClass))
	k8s.PATCH("/clusters/:id/ingressclasses/edit", p.write, network.EditIngressClass)
	k8s.DELETE("/clusters/:id/ingressclasses/:name", p.write, network.Delete(kopsapp.NetworkIngressClass))
	k8s.GET("/clusters/:id/ingressclasses/:name/yaml", p.read, network.YAML(kopsapp.NetworkIngressClass))

	// Endpoints
	k8s.GET("/clusters/:id/endpoints", p.read, connectivity.ListEndpoints)
	k8s.PATCH("/clusters/:id/endpoints/edit", p.write, connectivity.EditEndpoints)
	k8s.DELETE("/clusters/:id/endpoints/:ns/:name", p.write, connectivity.DeleteEndpoints)
	k8s.GET("/clusters/:id/endpoints/:ns/:name/yaml", p.read, connectivity.EndpointsYAML)

	// EndpointSlice
	k8s.GET("/clusters/:id/endpointslices", p.read, connectivity.ListEndpointSlices)
	k8s.PATCH("/clusters/:id/endpointslices/edit", p.write, connectivity.EditEndpointSlice)
	k8s.DELETE("/clusters/:id/endpointslices/:ns/:name", p.write, connectivity.DeleteEndpointSlice)
	k8s.GET("/clusters/:id/endpointslices/:ns/:name/yaml", p.read, connectivity.EndpointSliceYAML)
}

// ── 配置与存储：ConfigMap / Secret / ServiceAccount / PVC / PV / StorageClass / CSI / VolumeSnapshot / VolumeAttachment ──

func registerConfigStorageRoutes(a k8sRouteArgs) {
	k8s, configuration, storage, platform, relationships, p := a.k8s, a.configuration, a.storage, a.platform, a.relationships, a.perm

	// ConfigMap
	k8s.GET("/clusters/:id/configmaps", p.read, configuration.List(kopsapp.ConfigurationConfigMap))
	k8s.PATCH("/clusters/:id/configmaps/edit", p.write, configuration.EditConfigMap)
	k8s.DELETE("/clusters/:id/configmaps/:ns/:name", p.write, configuration.Delete(kopsapp.ConfigurationConfigMap))
	k8s.GET("/clusters/:id/configmaps/:ns/:name/yaml", p.read, configuration.YAML(kopsapp.ConfigurationConfigMap))
	k8s.GET("/clusters/:id/configmaps/:ns/:name/related", p.read, configuration.Related(kopsapp.ConfigurationConfigMap))

	// Secret
	k8s.GET("/clusters/:id/secrets", p.read, configuration.List(kopsapp.ConfigurationSecret))
	k8s.PATCH("/clusters/:id/secrets/edit", p.write, configuration.EditSecret)
	k8s.DELETE("/clusters/:id/secrets/:ns/:name", p.write, configuration.Delete(kopsapp.ConfigurationSecret))
	k8s.GET("/clusters/:id/secrets/:ns/:name/reveal", p.secretReveal, configuration.RevealSecret)
	k8s.GET("/clusters/:id/secrets/:ns/:name/decoded-data", p.secretReveal, configuration.RevealSecret)
	k8s.GET("/clusters/:id/secrets/:ns/:name/yaml", p.read, configuration.YAML(kopsapp.ConfigurationSecret))
	k8s.GET("/clusters/:id/secrets/:ns/:name/related", p.read, configuration.Related(kopsapp.ConfigurationSecret))

	// ServiceAccount
	k8s.GET("/clusters/:id/serviceaccounts", p.read, platform.List(kopsapp.PlatformServiceAccount))
	k8s.PATCH("/clusters/:id/serviceaccounts/edit", p.write, platform.Apply(kopsapp.PlatformServiceAccount))
	k8s.DELETE("/clusters/:id/serviceaccounts/:ns/:name", p.write, platform.Delete(kopsapp.PlatformServiceAccount))
	k8s.GET("/clusters/:id/serviceaccounts/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformServiceAccount))

	// PVC
	k8s.GET("/clusters/:id/pvcs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StoragePersistentVolumeClaim))
	k8s.POST("/clusters/:id/pvcs", p.write, storage.CreatePVC)
	k8s.DELETE("/clusters/:id/pvcs/:ns/:name", p.write, storage.Delete(kopsapp.StoragePersistentVolumeClaim))
	k8s.GET("/clusters/:id/pvcs/:ns/:name/yaml", p.read, storage.YAML(kopsapp.StoragePersistentVolumeClaim))

	// PV
	k8s.GET("/clusters/:id/pvs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StoragePersistentVolume))
	k8s.DELETE("/clusters/:id/pvs/:name", p.write, storage.Delete(kopsapp.StoragePersistentVolume))
	k8s.GET("/clusters/:id/pvs/:name/yaml", p.read, storage.YAML(kopsapp.StoragePersistentVolume))

	// StorageClass
	k8s.GET("/clusters/:id/storageclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StorageClass))
	k8s.PATCH("/clusters/:id/storageclasses/edit", p.write, storage.Apply(kopsapp.StorageClass))
	k8s.DELETE("/clusters/:id/storageclasses/:name", p.write, storage.Delete(kopsapp.StorageClass))
	k8s.GET("/clusters/:id/storageclasses/:name/yaml", p.read, storage.YAML(kopsapp.StorageClass))

	// CSIDriver
	k8s.GET("/clusters/:id/csidrivers", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformCSIDriver))
	k8s.PATCH("/clusters/:id/csidrivers/edit", p.write, platform.Apply(kopsapp.PlatformCSIDriver))
	k8s.DELETE("/clusters/:id/csidrivers/:name", p.write, platform.Delete(kopsapp.PlatformCSIDriver))
	k8s.GET("/clusters/:id/csidrivers/:name/yaml", p.read, platform.YAML(kopsapp.PlatformCSIDriver))

	// CSINode
	k8s.GET("/clusters/:id/csinodes", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformCSINode))
	k8s.PATCH("/clusters/:id/csinodes/edit", p.write, platform.Apply(kopsapp.PlatformCSINode))
	k8s.DELETE("/clusters/:id/csinodes/:name", p.write, platform.Delete(kopsapp.PlatformCSINode))
	k8s.GET("/clusters/:id/csinodes/:name/yaml", p.read, platform.YAML(kopsapp.PlatformCSINode))

	// CSIStorageCapacity
	k8s.GET("/clusters/:id/csistoragecapacities", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), platform.List(kopsapp.PlatformCSIStorageCapacity))
	k8s.PATCH("/clusters/:id/csistoragecapacities/edit", p.write, platform.Apply(kopsapp.PlatformCSIStorageCapacity))
	k8s.DELETE("/clusters/:id/csistoragecapacities/:ns/:name", p.write, platform.Delete(kopsapp.PlatformCSIStorageCapacity))
	k8s.GET("/clusters/:id/csistoragecapacities/:ns/:name/yaml", p.read, platform.YAML(kopsapp.PlatformCSIStorageCapacity))

	// VolumeAttachment
	k8s.GET("/clusters/:id/volumeattachments", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), relationships.List(kopsapp.RelationshipVolumeAttachment))
	k8s.PATCH("/clusters/:id/volumeattachments/edit", p.write, relationships.Apply(kopsapp.RelationshipVolumeAttachment))
	k8s.DELETE("/clusters/:id/volumeattachments/:name", p.write, relationships.Delete(kopsapp.RelationshipVolumeAttachment))
	k8s.GET("/clusters/:id/volumeattachments/:name/yaml", p.read, relationships.YAML(kopsapp.RelationshipVolumeAttachment))

	// VolumeSnapshot
	k8s.GET("/clusters/:id/volumesnapshots", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StorageVolumeSnapshot))
	k8s.PATCH("/clusters/:id/volumesnapshots/edit", p.write, storage.Apply(kopsapp.StorageVolumeSnapshot))
	k8s.DELETE("/clusters/:id/volumesnapshots/:ns/:name", p.write, storage.Delete(kopsapp.StorageVolumeSnapshot))
	k8s.GET("/clusters/:id/volumesnapshots/:ns/:name/yaml", p.read, storage.YAML(kopsapp.StorageVolumeSnapshot))

	// VolumeSnapshotClass
	k8s.GET("/clusters/:id/volumesnapshotclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StorageVolumeSnapshotClass))
	k8s.PATCH("/clusters/:id/volumesnapshotclasses/edit", p.write, storage.Apply(kopsapp.StorageVolumeSnapshotClass))
	k8s.DELETE("/clusters/:id/volumesnapshotclasses/:name", p.write, storage.Delete(kopsapp.StorageVolumeSnapshotClass))
	k8s.GET("/clusters/:id/volumesnapshotclasses/:name/yaml", p.read, storage.YAML(kopsapp.StorageVolumeSnapshotClass))

	// VolumeSnapshotContent
	k8s.GET("/clusters/:id/volumesnapshotcontents", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), storage.List(kopsapp.StorageVolumeSnapshotContent))
	k8s.PATCH("/clusters/:id/volumesnapshotcontents/edit", p.write, storage.Apply(kopsapp.StorageVolumeSnapshotContent))
	k8s.DELETE("/clusters/:id/volumesnapshotcontents/:name", p.write, storage.Delete(kopsapp.StorageVolumeSnapshotContent))
	k8s.GET("/clusters/:id/volumesnapshotcontents/:name/yaml", p.read, storage.YAML(kopsapp.StorageVolumeSnapshotContent))
}

// ── RBAC：Role / ClusterRole / RoleBinding / ClusterRoleBinding ──

func registerRBACRoutes(a k8sRouteArgs) {
	k8s, platform, p := a.k8s, a.platform, a.perm

	// Role
	k8s.GET("/clusters/:id/roles", p.rbacRead, platform.List(kopsapp.PlatformRole))
	k8s.PATCH("/clusters/:id/roles/edit", p.rbacWrite, platform.Apply(kopsapp.PlatformRole))
	k8s.DELETE("/clusters/:id/roles/:ns/:name", p.rbacWrite, platform.Delete(kopsapp.PlatformRole))
	k8s.GET("/clusters/:id/roles/:ns/:name/yaml", p.rbacRead, platform.YAML(kopsapp.PlatformRole))

	// ClusterRole
	k8s.GET("/clusters/:id/clusterroles", p.rbacRead, platform.List(kopsapp.PlatformClusterRole))
	k8s.PATCH("/clusters/:id/clusterroles/edit", p.rbacWrite, platform.Apply(kopsapp.PlatformClusterRole))
	k8s.DELETE("/clusters/:id/clusterroles/:name", p.rbacWrite, platform.Delete(kopsapp.PlatformClusterRole))
	k8s.GET("/clusters/:id/clusterroles/:name/yaml", p.rbacRead, platform.YAML(kopsapp.PlatformClusterRole))

	// RoleBinding
	k8s.GET("/clusters/:id/rolebindings", p.rbacRead, platform.List(kopsapp.PlatformRoleBinding))
	k8s.PATCH("/clusters/:id/rolebindings/edit", p.rbacWrite, platform.Apply(kopsapp.PlatformRoleBinding))
	k8s.DELETE("/clusters/:id/rolebindings/:ns/:name", p.rbacWrite, platform.Delete(kopsapp.PlatformRoleBinding))
	k8s.GET("/clusters/:id/rolebindings/:ns/:name/yaml", p.rbacRead, platform.YAML(kopsapp.PlatformRoleBinding))

	// ClusterRoleBinding
	k8s.GET("/clusters/:id/clusterrolebindings", p.rbacRead, platform.List(kopsapp.PlatformClusterRoleBinding))
	k8s.PATCH("/clusters/:id/clusterrolebindings/edit", p.rbacWrite, platform.Apply(kopsapp.PlatformClusterRoleBinding))
	k8s.DELETE("/clusters/:id/clusterrolebindings/:name", p.rbacWrite, platform.Delete(kopsapp.PlatformClusterRoleBinding))
	k8s.GET("/clusters/:id/clusterrolebindings/:name/yaml", p.rbacRead, platform.YAML(kopsapp.PlatformClusterRoleBinding))
}

// ── 批处理：Job / CronJob ──

func registerBatchRoutes(a k8sRouteArgs) {
	k8s, batch, p := a.k8s, a.batch, a.perm

	// Job
	k8s.GET("/clusters/:id/jobs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), batch.List(kopsapp.BatchJob))
	k8s.PATCH("/clusters/:id/jobs/edit", p.write, batch.EditJob)
	k8s.DELETE("/clusters/:id/jobs/completed", p.write, batch.DeleteCompletedJobs)
	k8s.DELETE("/clusters/:id/jobs/:ns/:name", p.write, batch.Delete(kopsapp.BatchJob))
	k8s.GET("/clusters/:id/jobs/:ns/:name/yaml", p.read, batch.YAML(kopsapp.BatchJob))

	// CronJob
	k8s.GET("/clusters/:id/cronjobs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), batch.List(kopsapp.BatchCronJob))
	k8s.PATCH("/clusters/:id/cronjobs/edit", p.write, batch.EditCronJob)
	k8s.POST("/clusters/:id/cronjobs/:ns/:name/trigger", p.write, batch.TriggerCronJob)
	k8s.PATCH("/clusters/:id/cronjobs/:ns/:name/suspend", p.write, batch.SuspendCronJob)
	k8s.POST("/clusters/:id/cronjobs/:ns/:name/execution-requests", p.write, batch.TriggerCronJob)
	k8s.PATCH("/clusters/:id/cronjobs/:ns/:name/suspension-state", p.write, batch.SuspendCronJob)
	k8s.DELETE("/clusters/:id/cronjobs/:ns/:name", p.write, batch.Delete(kopsapp.BatchCronJob))
	k8s.GET("/clusters/:id/cronjobs/:ns/:name/yaml", p.read, batch.YAML(kopsapp.BatchCronJob))
}

// ── Helm 管理 ──

func registerHelmRoutes(a k8sRouteArgs) {
	k8s, helm, p := a.k8s, a.helm, a.perm
	k8s.GET("/clusters/:id/helm/releases", p.read, helm.Releases)
	k8s.GET("/clusters/:id/helm/releases/detail", p.read, helm.ReleaseDetail)
	k8s.GET("/clusters/:id/helm/releases/:ns/:name", p.read, helm.ReleaseDetail)
	k8s.POST("/clusters/:id/helm/preflight", p.write, helm.Preflight)
	k8s.POST("/clusters/:id/helm/install", p.write, helm.Install)
	k8s.POST("/clusters/:id/helm/releases/:ns/:name/upgrade", p.write, helm.Upgrade)
	k8s.POST("/clusters/:id/helm/releases/:ns/:name/rollback", p.write, helm.Rollback)
	k8s.POST("/clusters/:id/helm/preflight-checks", p.write, helm.Preflight)
	k8s.POST("/clusters/:id/helm/releases", p.write, helm.Install)
	k8s.POST("/clusters/:id/helm/releases/:ns/:name/upgrade-attempts", p.write, helm.Upgrade)
	k8s.POST("/clusters/:id/helm/releases/:ns/:name/rollback-attempts", p.write, helm.Rollback)
	k8s.DELETE("/clusters/:id/helm/releases/:ns/:name", p.write, helm.Uninstall)
	k8s.GET("/clusters/:id/helm/repos", p.read, helm.Repositories)
	k8s.POST("/clusters/:id/helm/repos", p.write, helm.AddRepository)
	k8s.DELETE("/clusters/:id/helm/repos/:name", p.write, helm.DeleteRepository)
	k8s.GET("/clusters/:id/helm/search", p.read, helm.Search)
}

// ── WebSocket ──
