package router

import (
	"github.com/gin-gonic/gin"

	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
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
	storageSnapshotSupport gin.HandlerFunc
}

// k8sRouteArgs collects the dependencies shared by the resource-domain route
// files. Route declarations themselves live next to their resource domain.
type k8sRouteArgs struct {
	k8s           *gin.RouterGroup
	d             Deps
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

func registerK8sRoutes(authed *gin.RouterGroup, d Deps, manifest *kopshttp.ManifestController, namespace *kopshttp.NamespaceController, metrics *kopshttp.MetricsController, connectivity *kopshttp.ConnectivityController, nodes *kopshttp.NodeController, platform *kopshttp.PlatformResourceController, relationships *kopshttp.RelationshipResourceController, batch *kopshttp.BatchController, network *kopshttp.NetworkController, configuration *kopshttp.ConfigurationController, storage *kopshttp.StorageController, helm *kopshttp.HelmController, workloads *kopshttp.WorkloadController, pods *kopshttp.PodController, inspection *kopshttp.InspectionController, creators ...*kopshttp.ResourceCreatorController) {
	if manifest == nil || namespace == nil || metrics == nil || connectivity == nil || nodes == nil || platform == nil || relationships == nil || batch == nil || network == nil || configuration == nil || storage == nil || helm == nil || workloads == nil || pods == nil || inspection == nil {
		return
	}
	creator := &kopshttp.ResourceCreatorController{}
	if len(creators) > 0 && creators[0] != nil {
		creator = creators[0]
	}
	resourceSupportReadPerm := middleware.RequireAnyPermV2("k8s:read", "k8s:rbac_read")
	k8s := authed.Group("")
	args := k8sRouteArgs{
		k8s: k8s, d: d, manifest: manifest, namespace: namespace, metrics: metrics, connectivity: connectivity, nodes: nodes, platform: platform, relationships: relationships, batch: batch, network: network, configuration: configuration, storage: storage, helm: helm, workloads: workloads, pods: pods, inspection: inspection, creator: creator,
		perm: k8sPerms{
			read:                   middleware.RequirePermV2("k8s:read"),
			write:                  middleware.RequirePermV2("k8s:write"),
			namespaceRead:          middleware.RequireAnyPermV2("namespace:read", "k8s:read"),
			namespaceWrite:         middleware.RequireAnyPermV2("namespace:write", "k8s:write"),
			secretReveal:           middleware.RequirePermV2("k8s:secret_reveal"),
			rbacRead:               middleware.RequirePermV2("k8s:rbac_read"),
			rbacWrite:              middleware.RequirePermV2("k8s:rbac_write"),
			exec:                   middleware.RequirePermV2("k8s:exec"),
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
