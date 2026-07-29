package router

import (
	"context"
	"errors"
	"fmt"

	aigateway "k8s-platform-backend/internal/ai/adapters/gateway"
	aihttp "k8s-platform-backend/internal/ai/adapters/http"
	aiapp "k8s-platform-backend/internal/ai/application"
	audithttp "k8s-platform-backend/internal/audit/adapters/http"
	auditmysql "k8s-platform-backend/internal/audit/adapters/mysql"
	auditapp "k8s-platform-backend/internal/audit/application"
	changemysql "k8s-platform-backend/internal/change/adapters/mysql"
	changeapp "k8s-platform-backend/internal/change/application"
	fleethttp "k8s-platform-backend/internal/fleet/adapters/http"
	fleetmysql "k8s-platform-backend/internal/fleet/adapters/mysql"
	fleetapp "k8s-platform-backend/internal/fleet/application"
	"k8s-platform-backend/internal/fleet/domain"
	iamhttp "k8s-platform-backend/internal/iam/adapters/http"
	iammysql "k8s-platform-backend/internal/iam/adapters/mysql"
	iamapp "k8s-platform-backend/internal/iam/application"
	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	incidentmysql "k8s-platform-backend/internal/incident/adapters/mysql"
	incidentapp "k8s-platform-backend/internal/incident/application"
	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
	legacyai "k8s-platform-backend/internal/legacy/adapters/ai"
	legacykops "k8s-platform-backend/internal/legacy/adapters/kops"
	legacyprovision "k8s-platform-backend/internal/legacy/adapters/provisioning"
	"k8s-platform-backend/internal/legacy/service"
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
	actions     *service.AIActionService
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
	taskStore       *platformapp.TaskStore
	clusterRegistry *fleetapp.Registry
	k8s             *service.K8sService
	manifestApply   *legacykops.ManifestRuntime
	execSessions    *kopsapp.ExecSessionStore
	logSessions     *kopsapp.PodLogSessionStore
	dashboard       *service.DashboardService
	deploy          *service.DeployService
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

type namespaceResourceReader struct{ k8s *service.K8sService }

func (reader namespaceResourceReader) Summary(ctx context.Context, clusterID uint64, namespace string) ([]workspaceports.ResourceCount, int, error) {
	items, total, err := reader.k8s.GetNamespaceResourcesSummary(ctx, clusterID, namespace)
	if err != nil {
		return nil, 0, err
	}
	result := make([]workspaceports.ResourceCount, 0, len(items))
	for _, item := range items {
		result = append(result, workspaceports.ResourceCount{Key: item.Key, Count: item.Count})
	}
	return result, total, nil
}

func buildWorkspaceModule(d Deps, runtime moduleRuntime) workspaceModule {
	applicationService := workspaceapp.NewService(workspacemysql.NewRepository(d.DB), namespaceResourceReader{k8s: runtime.k8s})
	return workspaceModule{projects: workspacehttp.NewController(applicationService)}
}

func buildModuleRuntime(d Deps) moduleRuntime {
	taskStore := platformapp.NewTaskStore(d.DB)
	clusterRegistry := fleetapp.NewRegistry(fleetmysql.NewRegistry(d.DB, d.EncryptionKey))
	k8sService := service.NewK8sService(clusterRegistry, d.CacheStore, d.CacheTTL, d.K8sInsecureTLS)
	manifestApply := legacykops.NewManifestRuntime(d.DB, k8sService)
	return moduleRuntime{
		taskStore:       taskStore,
		clusterRegistry: clusterRegistry,
		k8s:             k8sService,
		manifestApply:   manifestApply,
		execSessions:    kopsapp.NewExecSessionStore(0),
		logSessions:     kopsapp.NewPodLogSessionStore(0),
		dashboard:       service.NewDashboardService(d.DB, clusterRegistry, k8sService, d.CacheStore),
		deploy:          service.NewDeployService(d.DB, d.EncryptionKey, taskStore, clusterRegistry),
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

type fleetClusterRuntime struct{ k8s *service.K8sService }

func (runtime fleetClusterRuntime) NormalizeAndValidate(ctx context.Context, value string) (string, error) {
	normalized, err := kopsclient.NormalizeKubeconfigContent(value)
	if err != nil {
		return "", fleetRuntimeError(err)
	}
	if err := runtime.k8s.ValidateKubeconfigFormat(ctx, normalized); err != nil {
		return "", fleetRuntimeError(err)
	}
	return normalized, nil
}
func (runtime fleetClusterRuntime) CheckHealth(ctx context.Context, id uint64) (bool, int, int, string, error) {
	apiOK, ready, total, version, err := runtime.k8s.CheckHealth(ctx, id)
	return apiOK, ready, total, version, fleetRuntimeError(err)
}
func (runtime fleetClusterRuntime) StopCaches(id uint64) { runtime.k8s.StopClusterCaches(id) }

func fleetRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, service.ErrInvalidParams):
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	case errors.Is(err, service.ErrNotFound):
		return fmt.Errorf("%w: %v", domain.ErrNotFound, err)
	case errors.Is(err, service.ErrK8sUnauthorized):
		return domain.ErrRuntimeUnauthorized
	case errors.Is(err, service.ErrK8sForbidden):
		return domain.ErrRuntimeForbidden
	case errors.Is(err, service.ErrK8sNetwork):
		return domain.ErrRuntimeNetwork
	case errors.Is(err, service.ErrK8sTimeout):
		return domain.ErrRuntimeTimeout
	case errors.Is(err, service.ErrK8sTLS):
		return domain.ErrRuntimeTLS
	case errors.Is(err, service.ErrK8s):
		return domain.ErrRuntime
	default:
		return err
	}
}

func buildFleetModule(d Deps, runtime moduleRuntime) fleetModule {
	return fleetModule{
		clusters:  fleethttp.NewClusterController(runtime.clusterRegistry, fleetClusterRuntime{k8s: runtime.k8s}),
		dashboard: fleethttp.NewDashboardController(runtime.dashboard),
	}
}

func buildKopsModule(d Deps, runtime moduleRuntime) kopsModule {
	namespaceDiagnosis := service.NewNamespaceDiagnosisService(runtime.k8s)
	resourceInspection := service.NewResourceInspectionService(runtime.k8s)
	permissionAuditService := service.NewK8sPermissionAuditService(
		d.DB,
		runtime.taskStore,
		runtime.clusterRegistry,
		runtime.k8s,
		d.CacheStore,
		d.EncryptionKey,
	)
	return kopsModule{
		manifests:       kopshttp.NewManifestController(kopsapp.NewManifestService(runtime.manifestApply)),
		namespaces:      kopshttp.NewNamespaceController(kopsapp.NewNamespaceService(legacykops.NewNamespaceRuntime(runtime.k8s))),
		metrics:         kopshttp.NewMetricsController(kopsapp.NewMetricsService(legacykops.NewMetricsRuntime(runtime.k8s))),
		connectivity:    kopshttp.NewConnectivityController(kopsapp.NewConnectivityService(legacykops.NewConnectivityRuntime(runtime.k8s))),
		nodes:           kopshttp.NewNodeController(kopsapp.NewNodeService(legacykops.NewNodeRuntime(runtime.k8s))),
		platform:        kopshttp.NewPlatformResourceController(kopsapp.NewPlatformResourceService(legacykops.NewPlatformResourceRuntime(runtime.k8s))),
		relationships:   kopshttp.NewRelationshipResourceController(kopsapp.NewRelationshipResourceService(legacykops.NewRelationshipResourceRuntime(runtime.k8s))),
		batch:           kopshttp.NewBatchController(kopsapp.NewBatchService(legacykops.NewBatchRuntime(runtime.k8s))),
		network:         kopshttp.NewNetworkController(kopsapp.NewNetworkService(legacykops.NewNetworkRuntime(runtime.k8s))),
		configuration:   kopshttp.NewConfigurationController(kopsapp.NewConfigurationService(legacykops.NewConfigurationRuntime(runtime.k8s))),
		storage:         kopshttp.NewStorageController(kopsapp.NewStorageService(legacykops.NewStorageRuntime(runtime.k8s))),
		workloads:       kopshttp.NewWorkloadController(kopsapp.NewWorkloadService(legacykops.NewWorkloadRuntime(runtime.k8s))),
		podLogStream:    kopshttp.NewPodLogStreamController(legacykops.NewPodLogStreamRuntime(runtime.k8s), runtime.logSessions),
		podExecStream:   kopshttp.NewPodExecStreamController(legacykops.NewPodExecStreamRuntime(runtime.k8s), runtime.execSessions),
		helm:            kopshttp.NewHelmController(kopsapp.NewHelmService(legacykops.NewHelmRuntime(runtime.k8s, runtime.deploy))),
		pods:            kopshttp.NewPodController(kopsapp.NewPodService(legacykops.NewPodRuntime(runtime.k8s), runtime.logSessions, runtime.execSessions)),
		inspection:      kopshttp.NewInspectionController(kopsapp.NewInspectionService(legacykops.NewInspectionRuntime(namespaceDiagnosis, resourceInspection))),
		creator:         kopshttp.NewResourceCreatorController(kopsapp.NewResourceCreatorService(legacykops.NewResourceCreatorRuntime(runtime.k8s))),
		permissionAudit: kopshttp.NewPermissionAuditController(kopsapp.NewPermissionAuditService(legacykops.NewPermissionAuditRuntime(permissionAuditService))),
		rbac:            kopshttp.NewRBACController(),
	}
}

func buildChangeModule(d Deps, runtime moduleRuntime) changeModule {
	workloadAction := kopsapp.NewActionProposalService(legacykops.NewActionProposalRuntime(runtime.k8s, runtime.manifestApply))
	applicationService := changeapp.NewService(changemysql.NewRepository(d.DB))
	return changeModule{
		application: applicationService,
		actions:     service.NewAIActionServiceWithChangeService(d.DB, workloadAction, applicationService),
	}
}

func buildAIModule(d Deps, runtime moduleRuntime, change changeModule) aiModule {
	namespaceDiagnosis := service.NewNamespaceDiagnosisService(runtime.k8s)
	resourceInspection := service.NewResourceInspectionService(runtime.k8s)
	resourceQuery := service.NewResourceQueryService(runtime.k8s)
	fileService := aiapp.NewAIFileService(d.DB, d.AIUploadDir)
	toolRegistry := service.NewAIToolRegistry(
		d.DB,
		aiapp.NewClusterReadModelService(legacyai.NewClusterReadPort(runtime.dashboard)),
		namespaceDiagnosis,
		resourceInspection,
		resourceQuery,
		change.actions,
		aiapp.NewResourceExportPolicyService(),
	)
	toolService := aiapp.NewToolService(d.DB, toolRegistry)
	chatService := service.NewAIChatService(
		d.DB,
		aigateway.NewAIGatewayService(d.DB, d.EncryptionKey),
		toolService,
		change.actions,
		fileService,
	)
	providerService := aiapp.NewAIProviderService(d.DB, d.EncryptionKey)
	routeSettingsService := aiapp.NewAIRouteSettingsService(d.DB)
	conversationService := aiapp.NewConversationService(d.DB)
	conversationDetailService := aiapp.NewConversationDetailService(d.DB, legacyai.NewConversationProjection(toolService, change.actions))
	runtimeAdapter := legacyai.NewRuntime(
		conversationDetailService,
		chatService,
		toolService,
		change.actions,
	)
	return aiModule{runtime: aihttp.NewRuntimeController(runtimeAdapter, fileService), management: aihttp.NewManagementController(providerService, routeSettingsService, conversationService)}
}

func buildProvisioningModule(d Deps, runtime moduleRuntime) provisioningModule {
	appTemplateService := provisionapp.NewAppTemplateService(d.DB)
	serverService := provisionapp.NewServerService(d.DB, d.EncryptionKey)
	credentialService := provisionapp.NewCredentialService(d.DB, d.EncryptionKey)
	_ = appTemplateService.SeedBuiltinAppTemplates(context.Background())
	return provisioningModule{
		serverAccess: legacyprovision.NewServerAccessController(runtime.deploy, runtime.execSessions),
		servers:      provisionhttp.NewServerController(serverService),
		credentials:  provisionhttp.NewCredentialController(credentialService),
		plans:        provisionhttp.NewDeployPlanController(provisionapp.NewDeployPlanService(d.DB)),
		runtime:      provisionhttp.NewRuntimeController(provisionapp.NewRuntimeService(legacyprovision.NewRuntime(runtime.deploy))),
		tasks:        provisionhttp.NewTaskController(provisionapp.NewTaskService(legacyprovision.NewRuntime(runtime.deploy))),
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
