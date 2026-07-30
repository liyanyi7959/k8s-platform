package router

import (
	"context"
	"errors"

	fleetapp "k8s-platform-backend/internal/fleet/application"
	fleetdomain "k8s-platform-backend/internal/fleet/domain"
	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
	kopsports "k8s-platform-backend/internal/kops/ports"
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
		metrics:         kopshttp.NewMetricsController(kopsapp.NewMetricsService(orchestrationkops.NewMetricsRuntime(runtime.k8s, fleetMetricsStore{registry: runtime.clusterRegistry}))),
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

// fleetMetricsStore is the composition-root adapter from the Fleet aggregate
// to the narrow Kops metrics port. The workflow sees only monitoring fields,
// not Fleet's registry implementation or domain aggregate.
type fleetMetricsStore struct{ registry *fleetapp.Registry }

func (store fleetMetricsStore) MonitorSource(ctx context.Context, clusterID uint64) (kopsports.MetricsCluster, error) {
	if store.registry == nil {
		return kopsports.MetricsCluster{}, kopsapp.ErrConflict
	}
	cluster, err := store.registry.MonitorSource(ctx, clusterID)
	if err != nil {
		return kopsports.MetricsCluster{}, fleetMetricsStoreError(err)
	}
	return kopsports.MetricsCluster{
		MonitorSource: cluster.MonitorSource, PrometheusURL: cluster.PrometheusURL,
		PrometheusStatus: cluster.PrometheusStatus, PrometheusDetectedAt: cluster.PrometheusDetectedAt,
	}, nil
}

func (store fleetMetricsStore) UpdateMonitorSource(ctx context.Context, clusterID uint64, source, url, status string) error {
	if store.registry == nil {
		return kopsapp.ErrConflict
	}
	return fleetMetricsStoreError(store.registry.UpdateMonitorSource(ctx, clusterID, source, url, status))
}

func fleetMetricsStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fleetdomain.ErrValidation):
		return kopsapp.ErrInvalidParams
	case errors.Is(err, fleetdomain.ErrNotFound):
		return kopsapp.ErrNotFound
	case errors.Is(err, fleetdomain.ErrConflict):
		return kopsapp.ErrConflict
	default:
		return err
	}
}
