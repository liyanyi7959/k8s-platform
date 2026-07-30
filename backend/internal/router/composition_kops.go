package router

import (
	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
	orchestrationkops "k8s-platform-backend/internal/orchestration/kops"
)

func buildKopsModule(d Deps, runtime moduleRuntime) kopsModule {
	inspection := kopsapp.NewInspectionService(kopsruntime.NewInspectionRuntime(runtime.k8s, runtime.nodeOperations, runtime.workloadOperations, runtime.podStreams, runtime.namespaceDiagnosis, runtime.namespaceWorkloads))
	permissionAuditTransport := kopsclient.NewPermissionAuditTransport(runtime.clusterRegistry, d.K8sInsecureTLS)
	permissionAuditCredentials := kopsclient.NewPermissionAuditCredentialStore(d.CacheStore, d.EncryptionKey)
	permissionAuditEngine := orchestrationkops.NewPermissionAuditEngine(
		d.DB,
		runtime.taskStore,
		runtime.clusterRegistry,
		permissionAuditTransport,
		permissionAuditCredentials,
		kopsclient.PermissionAuditCredentialTTL(),
	)
	return kopsModule{
		manifests:       kopshttp.NewManifestController(kopsapp.NewManifestService(runtime.manifestApply)),
		namespaces:      kopshttp.NewNamespaceController(kopsapp.NewNamespaceService(kopsruntime.NewNamespaceRuntime(runtime.k8s, runtime.nodeOperations, runtime.namespaceSummary))),
		metrics:         kopshttp.NewMetricsController(kopsapp.NewMetricsService(orchestrationkops.NewMetricsRuntime(runtime.k8s, runtime.clusterRegistry))),
		connectivity:    kopshttp.NewConnectivityController(kopsapp.NewConnectivityService(kopsruntime.NewConnectivityRuntime(runtime.k8s))),
		nodes:           kopshttp.NewNodeController(kopsapp.NewNodeService(kopsruntime.NewNodeRuntime(runtime.k8s, runtime.nodeOperations))),
		platform:        kopshttp.NewPlatformResourceController(kopsapp.NewPlatformResourceService(kopsruntime.NewPlatformResourceRuntime(runtime.k8s))),
		relationships:   kopshttp.NewRelationshipResourceController(kopsapp.NewRelationshipResourceService(kopsruntime.NewRelationshipResourceRuntime(runtime.k8s))),
		batch:           kopshttp.NewBatchController(kopsapp.NewBatchService(kopsruntime.NewBatchRuntime(runtime.k8s))),
		network:         kopshttp.NewNetworkController(kopsapp.NewNetworkService(kopsruntime.NewNetworkRuntime(runtime.k8s))),
		configuration:   kopshttp.NewConfigurationController(kopsapp.NewConfigurationService(kopsruntime.NewConfigurationRuntime(runtime.k8s))),
		storage:         kopshttp.NewStorageController(kopsapp.NewStorageService(kopsruntime.NewStorageRuntime(runtime.k8s))),
		workloads:       kopshttp.NewWorkloadController(kopsapp.NewWorkloadService(kopsruntime.NewWorkloadRuntime(runtime.k8s, runtime.workloadOperations))),
		podLogStream:    kopshttp.NewPodLogStreamController(kopsruntime.NewPodLogStreamRuntime(runtime.podStreams), runtime.logSessions),
		podExecStream:   kopshttp.NewPodExecStreamController(kopsruntime.NewPodExecStreamRuntime(runtime.podStreams), runtime.execSessions),
		helm:            kopshttp.NewHelmController(kopsapp.NewHelmService(orchestrationkops.NewHelmRuntime(runtime.k8s, runtime.nodeOperations, orchestrationkops.NewMasterHelmRuntime(d.DB, d.EncryptionKey)))),
		pods:            kopshttp.NewPodController(kopsapp.NewPodService(kopsruntime.NewPodRuntime(runtime.k8s, runtime.podOperations, runtime.podStreams), runtime.logSessions, runtime.execSessions)),
		inspection:      kopshttp.NewInspectionController(inspection),
		creator:         kopshttp.NewResourceCreatorController(kopsapp.NewResourceCreatorService(kopsruntime.NewResourceCreatorRuntime(runtime.k8s))),
		permissionAudit: kopshttp.NewPermissionAuditController(kopsapp.NewPermissionAuditService(orchestrationkops.NewPermissionAuditRuntimeWithStore(permissionAuditEngine, d.DB))),
		rbac:            kopshttp.NewRBACController(),
	}
}
