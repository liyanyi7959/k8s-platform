package router

import (
	"context"

	changemysql "k8s-platform-backend/internal/change/adapters/mysql"
	changeapp "k8s-platform-backend/internal/change/application"
	"k8s-platform-backend/internal/legacy/controller"
	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	incidentmysql "k8s-platform-backend/internal/incident/adapters/mysql"
	incidentapp "k8s-platform-backend/internal/incident/application"
	"k8s-platform-backend/internal/legacy/service"
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
	service    *service.AuditService
	controller *controller.AuditController
}

type iamModule struct {
	users *controller.UserController
}

type platformModule struct {
	settings *controller.SystemSettingController
}

type workspaceModule struct {
	projects *controller.ProjectController
}

type fleetModule struct {
	clusters  *controller.ClusterManageController
	dashboard *controller.DashboardController
}

type kopsModule struct {
	resources       *controller.K8sController
	permissionAudit *controller.K8sPermissionAuditController
}

type aiModule struct {
	controller *controller.AIController
}

type changeModule struct {
	application *changeapp.Service
	actions     *service.AIActionService
}

type provisioningModule struct {
	deploy      *controller.DeployController
	config      *controller.DeployConfigController
	automation  *controller.AutomationTaskController
	appTemplate *controller.AppTemplateController
}

type incidentModule struct {
	legacy *controller.MonitorIncidentController
	v2     *incidenthttp.Controller
}

type moduleRuntime struct {
	taskStore       *service.TaskStore
	clusterRegistry *service.ClusterRegistryService
	k8s             *service.K8sService
	manifestApply   *service.ManifestApplyRecordService
	execSessions    *service.ExecSessionStore
	logSessions     *service.PodLogSessionStore
	dashboard       *service.DashboardService
	deploy          *service.DeployService
}

func buildApplicationModules(d Deps) applicationModules {
	if d.DB == nil {
		return applicationModules{}
	}

	runtime := buildModuleRuntime(d)
	modules := applicationModules{}
	modules.audit = buildAuditModule(d)
	modules.iam = iamModule{users: controller.NewUserController(d.RbacSvc)}
	modules.platform = buildPlatformModule(d)
	modules.workspace = workspaceModule{projects: controller.NewProjectController(service.NewProjectService(d.DB), runtime.k8s)}
	modules.fleet = buildFleetModule(runtime)
	modules.kops = buildKopsModule(d, runtime)
	modules.change = buildChangeModule(d, runtime)
	modules.ai = buildAIModule(d, runtime, modules.change)
	modules.provisioning = buildProvisioningModule(d, runtime)
	modules.incident = buildIncidentModule(d)
	return modules
}

func buildModuleRuntime(d Deps) moduleRuntime {
	taskStore := service.NewTaskStore(d.DB)
	clusterRegistry := service.NewClusterRegistryService(d.DB, d.EncryptionKey)
	k8sService := service.NewK8sService(clusterRegistry, d.CacheStore, d.CacheTTL, d.K8sInsecureTLS)
	manifestApply := service.NewManifestApplyRecordService(d.DB, k8sService)
	return moduleRuntime{
		taskStore:       taskStore,
		clusterRegistry: clusterRegistry,
		k8s:             k8sService,
		manifestApply:   manifestApply,
		execSessions:    service.NewExecSessionStore(0),
		logSessions:     service.NewPodLogSessionStore(0),
		dashboard:       service.NewDashboardService(d.DB, clusterRegistry, k8sService, d.CacheStore),
		deploy:          service.NewDeployService(d.DB, d.EncryptionKey, taskStore, clusterRegistry),
	}
}

func buildAuditModule(d Deps) auditModule {
	auditService := service.NewAuditService(d.DB)
	return auditModule{service: auditService, controller: controller.NewAuditController(auditService)}
}

func buildPlatformModule(d Deps) platformModule {
	return platformModule{
		settings: controller.NewSystemSettingController(service.NewSystemSettingsService(d.DB)),
	}
}

func buildFleetModule(runtime moduleRuntime) fleetModule {
	return fleetModule{
		clusters:  controller.NewClusterManageController(runtime.clusterRegistry, runtime.k8s),
		dashboard: controller.NewDashboardController(runtime.dashboard),
	}
}

func buildKopsModule(d Deps, runtime moduleRuntime) kopsModule {
	namespaceDiagnosis := service.NewNamespaceDiagnosisService(runtime.k8s)
	resourceInspection := service.NewResourceInspectionService(runtime.k8s)
	return kopsModule{
		resources: controller.NewK8sController(
			runtime.k8s,
			runtime.manifestApply,
			runtime.execSessions,
			runtime.logSessions,
			namespaceDiagnosis,
			resourceInspection,
			runtime.deploy,
		),
		permissionAudit: controller.NewK8sPermissionAuditController(service.NewK8sPermissionAuditService(
			d.DB,
			runtime.taskStore,
			runtime.clusterRegistry,
			runtime.k8s,
			d.CacheStore,
			d.EncryptionKey,
		)),
	}
}

func buildChangeModule(d Deps, runtime moduleRuntime) changeModule {
	workloadAction := service.NewWorkloadActionService(runtime.k8s, runtime.manifestApply)
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
	fileService := service.NewAIFileService(d.DB, d.AIUploadDir)
	toolRegistry := service.NewAIToolRegistry(
		d.DB,
		service.NewClusterReadModelService(runtime.dashboard),
		namespaceDiagnosis,
		resourceInspection,
		resourceQuery,
		change.actions,
		service.NewResourceExportPolicyService(),
	)
	toolService := service.NewAIToolService(d.DB, toolRegistry)
	chatService := service.NewAIChatService(
		d.DB,
		service.NewAIGatewayService(d.DB, d.EncryptionKey),
		toolService,
		change.actions,
		fileService,
	)
	return aiModule{controller: controller.NewAIController(
		service.NewAIProviderService(d.DB, d.EncryptionKey),
		service.NewAIRouteSettingsService(d.DB),
		service.NewAIConversationService(d.DB),
		chatService,
		fileService,
		toolService,
		change.actions,
	)}
}

func buildProvisioningModule(d Deps, runtime moduleRuntime) provisioningModule {
	appTemplateService := service.NewAppTemplateService(d.DB)
	_ = appTemplateService.SeedBuiltinAppTemplates(context.Background())
	return provisioningModule{
		deploy:      controller.NewDeployController(runtime.deploy, runtime.execSessions),
		config:      controller.NewDeployConfigController(service.NewDeployConfigService(d.DB)),
		automation:  controller.NewAutomationTaskController(service.NewTaskService(runtime.taskStore)),
		appTemplate: controller.NewAppTemplateController(appTemplateService),
	}
}

func buildIncidentModule(d Deps) incidentModule {
	repository := incidentmysql.NewRepository(d.DB)
	applicationService := incidentapp.NewService(repository, repository)
	return incidentModule{
		legacy: controller.NewMonitorIncidentController(service.NewMonitorIncidentService(d.DB), applicationService),
		v2:     incidenthttp.NewController(applicationService),
	}
}
