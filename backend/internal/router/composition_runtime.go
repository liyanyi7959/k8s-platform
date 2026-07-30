package router

import (
	fleetmysql "k8s-platform-backend/internal/fleet/adapters/mysql"
	fleetapp "k8s-platform-backend/internal/fleet/application"
	legacyfleet "k8s-platform-backend/internal/integration/fleet"
	legacykops "k8s-platform-backend/internal/integration/kops"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
	platformmysql "k8s-platform-backend/internal/platform/adapters/mysql"
	platformapp "k8s-platform-backend/internal/platform/application"
)

// moduleRuntime contains cross-context runtime collaborators constructed once
// at the composition root. It is intentionally not an application dependency.
type moduleRuntime struct {
	taskStore          *platformapp.TaskStore
	clusterRegistry    *fleetapp.Registry
	k8s                *kopsclient.K8sService
	nodeOperations     *kopsruntime.NodeOperations
	podOperations      *kopsruntime.PodOperations
	podStreams         *kopsruntime.PodStreamOperations
	workloadOperations *kopsruntime.WorkloadOperations
	manifestApply      *legacykops.ManifestRuntime
	namespaceSummary   *kopsapp.NamespaceSummaryService
	namespaceWorkloads *kopsapp.NamespaceWorkloadService
	namespaceDiagnosis *kopsapp.NamespaceDiagnosisService
	execSessions       *kopsapp.ExecSessionStore
	logSessions        *kopsapp.PodLogSessionStore
	dashboard          *fleetapp.DashboardService
}

func buildModuleRuntime(d Deps) moduleRuntime {
	taskStore := platformapp.NewTaskStore(platformmysql.NewTaskRepository(d.DB), platformmysql.NewTaskLogRepository(d.DB))
	clusterRegistry := fleetapp.NewRegistry(fleetmysql.NewRegistry(d.DB, d.EncryptionKey))
	k8sService := kopsclient.NewK8sService(clusterRegistry, d.CacheStore, d.CacheTTL, normalizeKubeconfigRegistryError, d.K8sInsecureTLS)
	kubernetesTransport := kopsKubernetesTransport{k8s: k8sService}
	nodeOperations := kopsruntime.NewNodeOperations(kubernetesTransport)
	podOperations := kopsruntime.NewPodOperations(kubernetesTransport)
	podStreams := kopsruntime.NewPodStreamOperations(kubernetesTransport)
	workloadOperations := kopsruntime.NewWorkloadOperations(kubernetesTransport)
	manifestApply := legacykops.NewManifestRuntime(d.DB, k8sService)
	namespaceSummary := kopsapp.NewNamespaceSummaryService(legacykops.NewNamespaceSummaryRuntime(k8sService))
	namespaceWorkloads := kopsapp.NewNamespaceWorkloadService(legacykops.NewNamespaceWorkloadRuntime(k8sService))
	namespaceDiagnosis := kopsapp.NewNamespaceDiagnosisService(legacykops.NewNamespaceDiagnosisRuntime(k8sService, podOperations), namespaceSummary, namespaceWorkloads)
	return moduleRuntime{
		taskStore:          taskStore,
		clusterRegistry:    clusterRegistry,
		k8s:                k8sService,
		nodeOperations:     nodeOperations,
		podOperations:      podOperations,
		podStreams:         podStreams,
		workloadOperations: workloadOperations,
		manifestApply:      manifestApply,
		namespaceSummary:   namespaceSummary,
		namespaceWorkloads: namespaceWorkloads,
		namespaceDiagnosis: namespaceDiagnosis,
		execSessions:       kopsapp.NewExecSessionStore(0),
		logSessions:        kopsapp.NewPodLogSessionStore(0),
		dashboard:          fleetapp.NewDashboardService(clusterRegistry, legacyfleet.NewDashboardRuntime(k8sService), legacyfleet.NewDashboardCache(d.CacheStore)),
	}
}
