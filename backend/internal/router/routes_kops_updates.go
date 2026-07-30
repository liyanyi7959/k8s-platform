package router

import (
	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

// registerCanonicalResourceUpdateRoutes preserves the newer REST-shaped update
// aliases alongside the original /edit endpoints owned by each resource domain.
func registerCanonicalResourceUpdateRoutes(a k8sRouteArgs) {
	k8s, connectivity, platform, relationships, batch, network, configuration, storage, workloads, p := a.k8s, a.connectivity, a.platform, a.relationships, a.batch, a.network, a.configuration, a.storage, a.workloads, a.perm
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
	k8s.PATCH("/clusters/:id/workloads/deployments/:ns/:name", p.write, workloads.Edit(kopsapp.WorkloadDeployment))
	k8s.PATCH("/clusters/:id/workloads/statefulsets/:ns/:name", p.write, workloads.Edit(kopsapp.WorkloadStatefulSet))
	k8s.PATCH("/clusters/:id/workloads/daemonsets/:ns/:name", p.write, workloads.Edit(kopsapp.WorkloadDaemonSet))
	k8s.PATCH("/clusters/:id/workloads/:kind/:ns/:name/yaml", p.write, workloads.ApplyYAML)
	k8s.POST("/clusters/:id/workloads/:kind/:ns/:name/scale-operations", p.write, workloads.Scale)
	k8s.POST("/clusters/:id/workloads/:kind/:ns/:name/restart-operations", p.write, workloads.Restart)
}
