package router

import (
	"context"
	"errors"

	aigateway "k8s-platform-backend/internal/ai/adapters/gateway"
	aihttp "k8s-platform-backend/internal/ai/adapters/http"
	aiapp "k8s-platform-backend/internal/ai/application"
	audithttp "k8s-platform-backend/internal/audit/adapters/http"
	auditmysql "k8s-platform-backend/internal/audit/adapters/mysql"
	auditapp "k8s-platform-backend/internal/audit/application"
	changemysql "k8s-platform-backend/internal/change/adapters/mysql"
	changeapp "k8s-platform-backend/internal/change/application"
	fleethttp "k8s-platform-backend/internal/fleet/adapters/http"
	fleetkubernetes "k8s-platform-backend/internal/fleet/adapters/kubernetes"
	fleetmysql "k8s-platform-backend/internal/fleet/adapters/mysql"
	fleetapp "k8s-platform-backend/internal/fleet/application"
	iamhttp "k8s-platform-backend/internal/iam/adapters/http"
	iammysql "k8s-platform-backend/internal/iam/adapters/mysql"
	iamapp "k8s-platform-backend/internal/iam/application"
	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	incidentmysql "k8s-platform-backend/internal/incident/adapters/mysql"
	incidentapp "k8s-platform-backend/internal/incident/application"
	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
	legacyai "k8s-platform-backend/internal/legacy/adapters/ai"
	legacyfleet "k8s-platform-backend/internal/legacy/adapters/fleet"
	legacykops "k8s-platform-backend/internal/legacy/adapters/kops"
	legacyprovision "k8s-platform-backend/internal/legacy/adapters/provisioning"
	platformhttp "k8s-platform-backend/internal/platform/adapters/http"
	platformmysql "k8s-platform-backend/internal/platform/adapters/mysql"
	platformapp "k8s-platform-backend/internal/platform/application"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	workspacehttp "k8s-platform-backend/internal/workspace/adapters/http"
	workspacemysql "k8s-platform-backend/internal/workspace/adapters/mysql"
	workspaceapp "k8s-platform-backend/internal/workspace/application"
	workspaceports "k8s-platform-backend/internal/workspace/ports"
)

type applicationModules struct {
	audit        auditModule
	iam          iamModule
	platform     platformModule
	workspace    workspaceModule
	fleet        fleetModule
	kops         kopsModule
	ai           aiModule
	change       changeModule
	provisioning provisioningModule
	incident     incidentModule
}

type auditModule struct {
	service    *auditapp.Service
	controller *audithttp.Controller
}

type iamModule struct {
	users *iamhttp.Controller
}

type platformModule struct {
	settings *platformhttp.SettingsController
}

type workspaceModule struct {
	projects *workspacehttp.Controller
}

type fleetModule struct {
	clusters  *fleethttp.ClusterController
	dashboard *fleethttp.DashboardController
}

type kopsModule struct {
	permissionAudit *kopshttp.PermissionAuditController
	rbac            *kopshttp.RBACController
	manifests       *kopshttp.ManifestController
	namespaces      *kopshttp.NamespaceController
	metrics         *kopshttp.MetricsController
	connectivity    *kopshttp.ConnectivityController
	nodes           *kopshttp.NodeController
	platform        *kopshttp.PlatformResourceController
	relationships   *kopshttp.RelationshipResourceController
	batch           *kopshttp.BatchController
	network         *kopshttp.NetworkController
	configuration   *kopshttp.ConfigurationController
	storage         *kopshttp.StorageController
	workloads       *kopshttp.WorkloadController
	podLogStream    *kopshttp.PodLogStreamController
	podExecStream   *kopshttp.PodExecStreamController
	helm            *kopshttp.HelmController
	pods            *kopshttp.PodController
	inspection      *kopshttp.InspectionController
	creator         *kopshttp.ResourceCreatorController
}

type aiModule struct {
	runtime    *aihttp.RuntimeController
	management *aihttp.ManagementController
}

type changeModule struct {
	application *changeapp.Service
}

type provisioningModule struct {
	serverAccess *legacyprovision.ServerAccessController
	servers      *provisionhttp.ServerController
	credentials  *provisionhttp.CredentialController
	plans        *provisionhttp.DeployPlanController
	runtime      *provisionhttp.RuntimeController
	tasks        *provisionhttp.TaskController
	config       *provisionhttp.DeployConfigController
	automation   *provisionhttp.AutomationTaskController
	appTemplate  *provisionhttp.AppTemplateController
}

type incidentModule struct {
	legacy *incidenthttp.LegacyController
	v2     *incidenthttp.Controller
}

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

// automationTaskRuntime bridges the platform-wide task centre into the
// Provisioning automation use case at the composition root.
type automationTaskRuntime struct{ tasks *platformapp.TaskService }

func (runtime automationTaskRuntime) ListAutomationTasks(request provisionapp.AutomationTaskListRequest) (any, error) {
	if runtime.tasks == nil {
		return nil, provisionapp.ErrConflict
	}
	return runtime.tasks.List(platformapp.ListTasksRequest{
		Page: request.Page, PageSize: request.PageSize, Type: request.Type, Status: request.Status,
		Keyword: request.Keyword, SortBy: request.SortBy, Order: request.Order,
	}), nil
}

func (runtime automationTaskRuntime) GetAutomationTask(taskID int64) (any, error) {
	if runtime.tasks == nil {
		return nil, provisionapp.ErrConflict
	}
	task, found := runtime.tasks.Get(taskID)
	if !found {
		return nil, provisionapp.ErrNotFound
	}
	return task, nil
}

func (runtime automationTaskRuntime) AutomationTaskLogs(taskID int64, offset, limit int) ([]string, error) {
	if runtime.tasks == nil {
		return nil, provisionapp.ErrConflict
	}
	logs, found := runtime.tasks.Logs(taskID, offset, limit)
	if !found {
		return nil, provisionapp.ErrNotFound
	}
	return logs, nil
}

func (runtime automationTaskRuntime) CancelAutomationTask(taskID int64) error {
	if runtime.tasks == nil {
		return provisionapp.ErrConflict
	}
	err := runtime.tasks.Cancel(taskID)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, platformapp.ErrTaskNotFound):
		return provisionapp.ErrNotFound
	case errors.Is(err, platformapp.ErrTaskCannotCancel):
		return provisionapp.ErrConflict
	default:
		return err
	}
}

func buildApplicationModules(d Deps) applicationModules {
	if d.DB == nil {
		return applicationModules{}
	}

	runtime := buildModuleRuntime(d)
	modules := applicationModules{}
	modules.audit = buildAuditModule(d)
	modules.iam = buildIAMModule(d)
	modules.platform = buildPlatformModule(d)
	modules.workspace = buildWorkspaceModule(d, runtime)
	modules.fleet = buildFleetModule(d, runtime)
	modules.kops = buildKopsModule(d, runtime)
	modules.change = buildChangeModule(d, runtime)
	modules.ai = buildAIModule(d, runtime, modules.change)
	modules.provisioning = buildProvisioningModule(d, runtime)
	modules.incident = buildIncidentModule(d)
	return modules
}

func buildIAMModule(d Deps) iamModule {
	repository := iammysql.NewAuthRepository(d.DB)
	authService := d.IAMAuthService
	if authService == nil {
		authService = iamapp.NewAuthService(repository, iammysql.BcryptHasher{}, d.CacheStore, d.CacheTTL)
	}
	users := iamapp.NewUserManagement(repository, iammysql.BcryptHasher{}, authService)
	roles := iamapp.NewRoleManagement(repository, authService)
	return iamModule{users: iamhttp.NewController(users, roles)}
}

type namespaceResourceReader struct {
	summary kopsapp.NamespaceResourceSummaryReader
}

func (reader namespaceResourceReader) Summary(ctx context.Context, clusterID uint64, namespace string) ([]workspaceports.ResourceCount, int, error) {
	summary, err := reader.summary.Summary(ctx, clusterID, namespace)
	if err != nil {
		return nil, 0, err
	}
	result := make([]workspaceports.ResourceCount, 0, len(summary.Items))
	for _, item := range summary.Items {
		result = append(result, workspaceports.ResourceCount{Key: item.Key, Count: item.Count})
	}
	return result, summary.Total, nil
}

func buildWorkspaceModule(d Deps, runtime moduleRuntime) workspaceModule {
	applicationService := workspaceapp.NewService(workspacemysql.NewRepository(d.DB), namespaceResourceReader{summary: runtime.namespaceSummary})
	return workspaceModule{projects: workspacehttp.NewController(applicationService)}
}

func buildModuleRuntime(d Deps) moduleRuntime {
	taskStore := platformapp.NewTaskStore(d.DB)
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

func buildAuditModule(d Deps) auditModule {
	auditService := auditapp.NewService(auditmysql.NewRepository(d.DB))
	return auditModule{service: auditService, controller: audithttp.NewController(auditService)}
}

func buildPlatformModule(d Deps) platformModule {
	settingsService := platformapp.NewService(platformmysql.NewSettingsRepository(d.DB))
	return platformModule{
		settings: platformhttp.NewSettingsController(settingsService),
	}
}

func buildFleetModule(d Deps, runtime moduleRuntime) fleetModule {
	return fleetModule{
		clusters:  fleethttp.NewClusterController(runtime.clusterRegistry, fleetkubernetes.NewClusterRuntime(fleetKubernetesTransport{k8s: runtime.k8s})),
		dashboard: fleethttp.NewDashboardController(runtime.dashboard),
	}
}

func buildKopsModule(d Deps, runtime moduleRuntime) kopsModule {
	inspection := kopsapp.NewInspectionService(legacykops.NewInspectionRuntime(runtime.k8s, runtime.nodeOperations, runtime.workloadOperations, runtime.podStreams, runtime.namespaceDiagnosis, runtime.namespaceWorkloads))
	permissionAuditTransport := kopsclient.NewPermissionAuditTransport(runtime.clusterRegistry, d.K8sInsecureTLS)
	permissionAuditCredentials := kopsclient.NewPermissionAuditCredentialStore(d.CacheStore, d.EncryptionKey)
	permissionAuditEngine := legacykops.NewPermissionAuditEngine(
		d.DB,
		runtime.taskStore,
		runtime.clusterRegistry,
		permissionAuditTransport,
		permissionAuditCredentials,
		kopsclient.PermissionAuditCredentialTTL(),
	)
	return kopsModule{
		manifests:       kopshttp.NewManifestController(kopsapp.NewManifestService(runtime.manifestApply)),
		namespaces:      kopshttp.NewNamespaceController(kopsapp.NewNamespaceService(legacykops.NewNamespaceRuntime(runtime.k8s, runtime.nodeOperations, runtime.namespaceSummary))),
		metrics:         kopshttp.NewMetricsController(kopsapp.NewMetricsService(legacykops.NewMetricsRuntime(runtime.k8s, runtime.clusterRegistry))),
		connectivity:    kopshttp.NewConnectivityController(kopsapp.NewConnectivityService(legacykops.NewConnectivityRuntime(runtime.k8s))),
		nodes:           kopshttp.NewNodeController(kopsapp.NewNodeService(legacykops.NewNodeRuntime(runtime.k8s, runtime.nodeOperations))),
		platform:        kopshttp.NewPlatformResourceController(kopsapp.NewPlatformResourceService(legacykops.NewPlatformResourceRuntime(runtime.k8s))),
		relationships:   kopshttp.NewRelationshipResourceController(kopsapp.NewRelationshipResourceService(legacykops.NewRelationshipResourceRuntime(runtime.k8s))),
		batch:           kopshttp.NewBatchController(kopsapp.NewBatchService(legacykops.NewBatchRuntime(runtime.k8s))),
		network:         kopshttp.NewNetworkController(kopsapp.NewNetworkService(legacykops.NewNetworkRuntime(runtime.k8s))),
		configuration:   kopshttp.NewConfigurationController(kopsapp.NewConfigurationService(legacykops.NewConfigurationRuntime(runtime.k8s))),
		storage:         kopshttp.NewStorageController(kopsapp.NewStorageService(legacykops.NewStorageRuntime(runtime.k8s))),
		workloads:       kopshttp.NewWorkloadController(kopsapp.NewWorkloadService(legacykops.NewWorkloadRuntime(runtime.k8s, runtime.workloadOperations))),
		podLogStream:    kopshttp.NewPodLogStreamController(legacykops.NewPodLogStreamRuntime(runtime.podStreams), runtime.logSessions),
		podExecStream:   kopshttp.NewPodExecStreamController(legacykops.NewPodExecStreamRuntime(runtime.podStreams), runtime.execSessions),
		helm:            kopshttp.NewHelmController(kopsapp.NewHelmService(legacykops.NewHelmRuntime(runtime.k8s, runtime.nodeOperations, legacykops.NewMasterHelmRuntime(d.DB, d.EncryptionKey)))),
		pods:            kopshttp.NewPodController(kopsapp.NewPodService(legacykops.NewPodRuntime(runtime.k8s, runtime.podOperations, runtime.podStreams), runtime.logSessions, runtime.execSessions)),
		inspection:      kopshttp.NewInspectionController(inspection),
		creator:         kopshttp.NewResourceCreatorController(kopsapp.NewResourceCreatorService(legacykops.NewResourceCreatorRuntime(runtime.k8s))),
		permissionAudit: kopshttp.NewPermissionAuditController(kopsapp.NewPermissionAuditService(legacykops.NewPermissionAuditRuntimeWithStore(permissionAuditEngine, d.DB))),
		rbac:            kopshttp.NewRBACController(),
	}
}

func buildChangeModule(d Deps, runtime moduleRuntime) changeModule {
	applicationService := changeapp.NewService(changemysql.NewRepository(d.DB))
	return changeModule{
		application: applicationService,
	}
}

func buildAIModule(d Deps, runtime moduleRuntime, change changeModule) aiModule {
	workloadAction := kopsapp.NewActionProposalService(legacykops.NewActionProposalRuntime(runtime.k8s, runtime.nodeOperations, runtime.podOperations, runtime.workloadOperations, runtime.manifestApply))
	actions := legacyai.NewActionRuntimeWithChangeService(d.DB, workloadAction, change.application)
	resourceInspection := kopsapp.NewInspectionService(legacykops.NewInspectionRuntime(runtime.k8s, runtime.nodeOperations, runtime.workloadOperations, runtime.podStreams, runtime.namespaceDiagnosis, runtime.namespaceWorkloads))
	resourceQuery := aiapp.NewResourceQueryService(
		legacyai.NewResourceQueryRuntime(runtime.k8s, runtime.nodeOperations, runtime.podStreams),
		legacyai.NewResourceQueryPresenter(),
	)
	fileService := aiapp.NewAIFileService(d.DB, d.AIUploadDir)
	toolRegistry := legacyai.NewToolRegistry(
		d.DB,
		aiapp.NewClusterReadModelService(legacyai.NewClusterReadPort(runtime.dashboard)),
		runtime.namespaceDiagnosis,
		resourceInspection,
		resourceQuery,
		actions,
		aiapp.NewResourceExportPolicyService(),
	)
	toolService := aiapp.NewToolService(d.DB, toolRegistry)
	chatRuntime := legacyai.NewChatRuntime(
		d.DB,
		aigateway.NewAIGatewayService(d.DB, d.EncryptionKey),
		toolService,
		actions,
		fileService,
	)
	providerService := aiapp.NewAIProviderService(d.DB, d.EncryptionKey)
	routeSettingsService := aiapp.NewAIRouteSettingsService(d.DB)
	conversationService := aiapp.NewConversationService(d.DB)
	conversationDetailService := aiapp.NewConversationDetailService(d.DB, legacyai.NewConversationProjection(toolService, actions))
	runtimeAdapter := legacyai.NewRuntime(
		conversationDetailService,
		chatRuntime,
		toolService,
		actions,
	)
	return aiModule{runtime: aihttp.NewRuntimeController(runtimeAdapter, fileService), management: aihttp.NewManagementController(providerService, routeSettingsService, conversationService)}
}

func buildProvisioningModule(d Deps, runtime moduleRuntime) provisioningModule {
	appTemplateService := provisionapp.NewAppTemplateService(d.DB)
	serverService := provisionapp.NewServerService(d.DB, d.EncryptionKey)
	credentialService := provisionapp.NewCredentialService(d.DB, d.EncryptionKey)
	sshRuntime := legacyprovision.NewSSHRuntime(d.DB, d.EncryptionKey)
	preflightRuntime := legacyprovision.NewPreflightRuntime(d.DB, d.EncryptionKey)
	ansibleRunner := legacyprovision.NewAnsibleRunner(d.DB, d.EncryptionKey, runtime.taskStore)
	deploymentExecutor := legacyprovision.NewDeploymentExecutor(d.DB, runtime.taskStore, runtime.clusterRegistry, preflightRuntime, ansibleRunner)
	deploymentRuntime := legacyprovision.NewRuntime(d.DB, deploymentExecutor, preflightRuntime)
	_ = appTemplateService.SeedBuiltinAppTemplates(context.Background())
	return provisioningModule{
		serverAccess: legacyprovision.NewServerAccessController(sshRuntime, runtime.execSessions, serverService),
		servers:      provisionhttp.NewServerController(serverService),
		credentials:  provisionhttp.NewCredentialController(credentialService),
		plans:        provisionhttp.NewDeployPlanController(provisionapp.NewDeployPlanService(d.DB)),
		runtime:      provisionhttp.NewRuntimeController(provisionapp.NewRuntimeService(deploymentRuntime)),
		tasks:        provisionhttp.NewTaskController(provisionapp.NewTaskService(deploymentRuntime)),
		config:       provisionhttp.NewDeployConfigController(provisionapp.NewDeployConfigService(d.DB)),
		automation:   provisionhttp.NewAutomationTaskController(provisionapp.NewAutomationTaskService(automationTaskRuntime{tasks: platformapp.NewTaskService(runtime.taskStore)})),
		appTemplate:  provisionhttp.NewAppTemplateController(appTemplateService),
	}
}

func buildIncidentModule(d Deps) incidentModule {
	repository := incidentmysql.NewRepository(d.DB)
	applicationService := incidentapp.NewService(repository, repository)
	monitoringService := incidentapp.NewMonitoringService(repository)
	return incidentModule{
		legacy: incidenthttp.NewLegacyController(monitoringService, applicationService),
		v2:     incidenthttp.NewController(applicationService),
	}
}
