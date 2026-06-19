// router 负责注册 HTTP 路由，并把 controller 绑定到 gin.Engine。
//
// 路由组织约定：
// - 所有 API 以 `/api/v1` 为前缀
// - 统一响应结构由 pkg/resp 负责封装（无论成功/失败通常都返回 HTTP 200，错误由 code/message 表达）
// - 认证/鉴权通过中间件完成：
//   - RequestID：为每个请求注入 request id，便于排障追踪
//   - AuthRequiredWithRBAC：解析 JWT，并加载/校验权限点
//   - RequirePerm：对具体路由进行权限点校验
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/controller"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
)

// New 组装路由。接收 Deps 依赖容器，消除长参数列表。
func New(d Deps) (*gin.Engine, error) {
	r := gin.New()

	// ── 全局中间件 ──
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLogger())
	r.Use(middleware.RecoveryWithZap())
	r.Use(middleware.CORS())

	// ── 基础依赖（供当前仍在使用的 K8S 能力复用） ──
	taskStore := service.NewTaskStore(d.DB)

	// ── 审计服务 ──
	var auditSvc *service.AuditService
	if d.DB != nil {
		auditSvc = service.NewAuditService(d.DB)
	}

	// ── DB 依赖模块 ──
	var clusterManageCtl *controller.ClusterManageController
	var k8sCtl *controller.K8sController
	var dashboardCtl *controller.DashboardController
	var permissionAuditCtl *controller.K8sPermissionAuditController
	var auditCtl *controller.AuditController
	var userCtl *controller.UserController
	var aiCtl *controller.AIController
	var deployCtl *controller.DeployController
	var deployConfigCtl *controller.DeployConfigController

	if d.DB != nil {
		clusterReg := service.NewClusterRegistryService(d.DB, d.EncryptionKey)
		k8sSvc := service.NewK8sService(clusterReg, d.CacheStore, d.CacheTTL, d.K8sInsecureTLS)
		manifestApplySvc := service.NewManifestApplyRecordService(d.DB, k8sSvc)
		clusterManageCtl = controller.NewClusterManageController(clusterReg, k8sSvc)
		execSessions := service.NewExecSessionStore(0)
		logSessions := service.NewPodLogSessionStore(0)
		dashboardSvc := service.NewDashboardService(d.DB, clusterReg, k8sSvc, d.CacheStore)
		dashboardCtl = controller.NewDashboardController(dashboardSvc)
		permissionAuditSvc := service.NewK8sPermissionAuditService(d.DB, taskStore, clusterReg, k8sSvc, d.CacheStore, d.EncryptionKey)
		permissionAuditCtl = controller.NewK8sPermissionAuditController(permissionAuditSvc)
		auditCtl = controller.NewAuditController(auditSvc)
		userCtl = controller.NewUserController(d.RbacSvc)
		aiProviderSvc := service.NewAIProviderService(d.DB, d.EncryptionKey)
		aiRouteSettingsSvc := service.NewAIRouteSettingsService(d.DB)
		aiGatewaySvc := service.NewAIGatewayService(d.DB, d.EncryptionKey)
		aiConversationSvc := service.NewAIConversationService(d.DB)
		clusterReadModelSvc := service.NewClusterReadModelService(dashboardSvc)
		namespaceDiagnosisSvc := service.NewNamespaceDiagnosisService(k8sSvc)
		resourceInspectionSvc := service.NewResourceInspectionService(k8sSvc)
		resourceQuerySvc := service.NewResourceQueryService(k8sSvc)
		resourceExportPolicySvc := service.NewResourceExportPolicyService()
		workloadActionSvc := service.NewWorkloadActionService(k8sSvc, manifestApplySvc)
		aiActionSvc := service.NewAIActionService(d.DB, workloadActionSvc)
		k8sCtl = controller.NewK8sController(
			k8sSvc,
			manifestApplySvc,
			execSessions,
			logSessions,
			namespaceDiagnosisSvc,
			resourceInspectionSvc,
		)
		aiToolRegistry := service.NewAIToolRegistry(clusterReadModelSvc, namespaceDiagnosisSvc, resourceInspectionSvc, resourceQuerySvc, aiActionSvc, resourceExportPolicySvc)
		aiToolSvc := service.NewAIToolService(d.DB, aiToolRegistry)
		aiChatSvc := service.NewAIChatService(d.DB, aiGatewaySvc, aiToolSvc, aiActionSvc)
		aiCtl = controller.NewAIController(aiProviderSvc, aiRouteSettingsSvc, aiConversationSvc, aiChatSvc, aiToolSvc, aiActionSvc)
		deploySvc := service.NewDeployService(d.DB, d.EncryptionKey, taskStore, clusterReg)
		deployCtl = controller.NewDeployController(deploySvc)
		deployConfigSvc := service.NewDeployConfigService(d.DB)
		deployConfigCtl = controller.NewDeployConfigController(deployConfigSvc)
	}

	// ── 健康检查 ──
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ── 路由注册 ──
	registerRoutes(r, d, auditSvc, clusterManageCtl, k8sCtl, dashboardCtl, permissionAuditCtl, auditCtl, userCtl, aiCtl, deployCtl, deployConfigCtl)

	return r, nil
}

//nolint:funlen // 路由注册表天然是长函数，按业务域分段组织
func registerRoutes(
	r *gin.Engine, d Deps,
	auditSvc *service.AuditService,
	clusterManageCtl *controller.ClusterManageController,
	k8sCtl *controller.K8sController,
	dashboardCtl *controller.DashboardController,
	permissionAuditCtl *controller.K8sPermissionAuditController,
	auditCtl *controller.AuditController,
	userCtl *controller.UserController,
	aiCtl *controller.AIController,
	deployCtl *controller.DeployController,
	deployConfigCtl *controller.DeployConfigController,
) {
	api := r.Group("/api/v1")

	// ── 公开接口（无需认证） ──
	api.POST("/auth/login", d.AuthCtl.Login)
	api.POST("/auth/logout", d.AuthCtl.Logout)
	api.GET("/auth/me", d.AuthCtl.Me)

	// ── 需认证接口 ──
	authed := api.Group("")
	authed.Use(middleware.AuthRequiredWithRBAC(d.JWTMgr, d.RbacSvc))

	// ── 审计中间件（仅对写操作生效） ──
	if auditSvc != nil {
		authed.Use(middleware.AuditLogger(auditSvc))
	}

	authed.POST("/auth/change-password", d.AuthCtl.ChangePassword)

	registerClusterRoutes(authed, d, clusterManageCtl)
	registerDashboardRoutes(authed, d, dashboardCtl)
	registerPermissionAuditRoutes(authed, permissionAuditCtl)
	registerK8sRoutes(authed, d, k8sCtl)
	registerWebSocketRoutes(authed, k8sCtl)
	registerAuditRoutes(authed, auditCtl)
	registerUserRoutes(authed, userCtl)
	registerAIRoutes(authed, aiCtl)
	registerDeployRoutes(authed, deployCtl, deployConfigCtl)
}

func registerDeployRoutes(authed *gin.RouterGroup, ctl *controller.DeployController, configCtl *controller.DeployConfigController) {
	if ctl == nil {
		return
	}
	deploy := authed.Group("/deploy")

	readServer := middleware.RequirePerm("deploy:server_read")
	writeServer := middleware.RequirePerm("deploy:server_write")
	deleteServer := middleware.RequirePerm("deploy:server_delete")
	readPlan := middleware.RequirePerm("deploy:plan_read")
	writePlan := middleware.RequirePerm("deploy:plan_write")
	deletePlan := middleware.RequirePerm("deploy:plan_delete")
	execDeploy := middleware.RequirePerm("deploy:execute")

	// 服务器管理
	deploy.GET("/servers", readServer, ctl.ListServers)
	deploy.POST("/servers", writeServer, ctl.CreateServer)
	deploy.GET("/servers/:id", readServer, ctl.GetServer)
	deploy.PUT("/servers/:id", writeServer, ctl.UpdateServer)
	deploy.POST("/servers/:id/test-ssh", readServer, ctl.TestSSH)
	deploy.DELETE("/servers/:id", deleteServer, ctl.DeleteServer)

	// SSH 凭证
	deploy.GET("/credentials", readServer, ctl.ListCredentials)
	deploy.POST("/credentials", writeServer, ctl.CreateCredential)
	deploy.GET("/credentials/:id", readServer, ctl.GetCredential)
	deploy.PUT("/credentials/:id", writeServer, ctl.UpdateCredential)
	deploy.DELETE("/credentials/:id", deleteServer, ctl.DeleteCredential)
	deploy.POST("/credentials/batch-delete", deleteServer, ctl.BatchDeleteCredentials)

	// 部署计划
	deploy.GET("/plans", readPlan, ctl.ListPlans)
	deploy.POST("/plans", writePlan, ctl.CreatePlan)
	deploy.GET("/plans/:id/dry-run", readPlan, ctl.DryRunPlan)
	deploy.POST("/plans/:id/execute", execDeploy, ctl.ExecutePlan)
	deploy.POST("/plans/:id/cancel", execDeploy, ctl.CancelPlan)
	deploy.POST("/plans/:id/retry", execDeploy, ctl.RetryPlan)
	deploy.DELETE("/plans/:id", deletePlan, ctl.DeletePlan)

	// 部署任务日志
	deploy.GET("/tasks/:taskId/logs", readPlan, ctl.GetDeployTaskLogs)
	deploy.GET("/tasks/:taskId/logs/sse", readPlan, ctl.GetDeployTaskLogsSSE)

	// 部署配置管理
	if configCtl != nil {
		deploy.GET("/configs", readPlan, configCtl.ListConfigs)
		deploy.GET("/configs/os-types", readPlan, configCtl.ListSupportedOSTypes)
		deploy.GET("/configs/:id", readPlan, configCtl.GetConfig)
		deploy.PUT("/configs/:id", writePlan, configCtl.UpdateConfig)
		deploy.GET("/configs/:id/versions", readPlan, configCtl.GetConfigVersions)

		// 仓库配置管理
		deploy.GET("/repositories", readPlan, configCtl.ListRepositories)
		deploy.POST("/repositories", writePlan, configCtl.CreateRepository)
		deploy.GET("/repositories/:id", readPlan, configCtl.GetRepository)
		deploy.PUT("/repositories/:id", writePlan, configCtl.UpdateRepository)
		deploy.DELETE("/repositories/:id", deletePlan, configCtl.DeleteRepository)
	}
}

func registerPermissionAuditRoutes(authed *gin.RouterGroup, ctl *controller.K8sPermissionAuditController) {
	if ctl == nil {
		return
	}
	auditPerm := middleware.RequirePerm("k8s:permission_audit")
	clusters := authed.Group("/clusters")
	clusters.POST("/:id/permission-audits", auditPerm, ctl.CreateManaged)
	clusters.GET("/:id/permission-audits/latest", auditPerm, ctl.LatestForCluster)
	clusters.GET("/:id/permission-audits/recommend-rbac", auditPerm, ctl.RecommendRBAC)
	clusters.GET("/:id/permission-audits/rbac-matrix/default", auditPerm, ctl.DefaultRBACMatrix)
	clusters.POST("/:id/permission-audits/rbac-matrix/yaml", auditPerm, ctl.RBACFromMatrix)

	audits := authed.Group("/permission-audits")
	audits.GET("", auditPerm, ctl.List)
	audits.GET("/:id", auditPerm, ctl.Get)
	audits.GET("/:id/logs", auditPerm, ctl.Logs)
	audits.GET("/:id/compare", auditPerm, ctl.Compare)
	audits.GET("/:id/findings", auditPerm, ctl.ListFindings)
	audits.POST("/:id/cancel", auditPerm, ctl.Cancel)
	audits.POST("/adhoc", auditPerm, ctl.CreateAdhoc)
}

// ── 集群管理 ──

func registerClusterRoutes(authed *gin.RouterGroup, d Deps, ctl *controller.ClusterManageController) {
	if ctl == nil {
		return
	}
	clusters := authed.Group("/clusters")
	clusters.Use(middleware.RequirePerm("cluster:read"))
	clusters.GET("", ctl.List)
	clusters.GET("/:id", ctl.Get)
	clusters.POST("/:id/check-health", ctl.CheckHealth)
	clusters.PATCH("/:id", middleware.RequirePerm("cluster:create"), ctl.Patch)
	clusters.DELETE("/:id", middleware.RequirePerm("cluster:create"), ctl.Delete)

	authed.POST("/clusters/import", middleware.RequirePerm("cluster:create"), ctl.Import)
}

// ── 仪表盘 ──

func registerDashboardRoutes(authed *gin.RouterGroup, d Deps, ctl *controller.DashboardController) {
	if ctl == nil {
		return
	}
	dash := authed.Group("/dashboard")
	dash.Use(middleware.RequirePerm("cluster:read"))
	dash.GET("/clusters/:id/overview", middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.GetClusterOverview)
	dash.GET("/clusters/:id/certificate-risks", ctl.GetClusterCertificateRisks)
}

// ── K8s 资源 ──

// k8sPerms 封装 K8s 路由注册所需的所有权限中间件，避免在子函数中重复声明。
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
	k8s  *gin.RouterGroup
	d    Deps
	ctl  *controller.K8sController
	perm k8sPerms
}

func registerK8sRoutes(authed *gin.RouterGroup, d Deps, ctl *controller.K8sController) {
	if ctl == nil {
		return
	}
	k8s := authed.Group("")
	resourceSupportReadPerm := middleware.RequireAnyPerm("k8s:read", "k8s:rbac_read")
	args := k8sRouteArgs{
		k8s: k8s, d: d, ctl: ctl,
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
}

// ── 集群级资源：Namespace / Node / HPA / PDB / Event / CRD / APIService / PriorityClass / RuntimeClass / Webhook / Lease ──

func registerClusterResourceRoutes(a k8sRouteArgs) {
	k8s, ctl, p := a.k8s, a.ctl, a.perm

	// Namespace
	k8s.GET("/clusters/:id/namespaces", p.namespaceRead, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListNamespaces)
	k8s.POST("/clusters/:id/namespaces", p.namespaceWrite, ctl.CreateNamespace)
	k8s.DELETE("/clusters/:id/namespaces/:ns", p.namespaceWrite, ctl.DeleteNamespace)
	k8s.GET("/clusters/:id/namespaces/:ns/yaml", p.namespaceRead, ctl.GetNamespaceYAML)
	k8s.GET("/clusters/:id/namespaces/:ns/resources-summary", p.namespaceRead, ctl.GetNamespaceResourcesSummary)
	k8s.GET("/clusters/:id/namespaces/:ns/inspection", p.namespaceRead, ctl.GetNamespaceInspection)
	k8s.GET("/clusters/:id/namespaces/:ns/workload-inventory", p.namespaceRead, ctl.GetNamespaceWorkloadInventory)

	// Node
	k8s.GET("/clusters/:id/nodes", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListNodes)
	k8s.GET("/clusters/:id/nodes/:name/detail", p.read, ctl.GetNodeDetail)
	k8s.GET("/clusters/:id/nodes/:name/yaml", p.read, ctl.GetNodeYAML)
	k8s.GET("/clusters/:id/nodes/:name/pods", p.read, ctl.ListNodePods)
	k8s.GET("/clusters/:id/nodes/:name/events", p.read, ctl.ListNodeEvents)
	k8s.POST("/clusters/:id/nodes/:name/cordon", p.write, ctl.CordonNode)
	k8s.POST("/clusters/:id/nodes/:name/uncordon", p.write, ctl.UncordonNode)
	k8s.POST("/clusters/:id/nodes/:name/drain", p.write, ctl.DrainNode)
	k8s.DELETE("/clusters/:id/nodes/:name", p.write, ctl.DeleteNode)

	// HPA
	k8s.GET("/clusters/:id/hpas", p.read, ctl.ListHPAs)
	k8s.PATCH("/clusters/:id/hpas/edit", p.write, ctl.EditHPA)
	k8s.DELETE("/clusters/:id/hpas/:ns/:name", p.write, ctl.DeleteHPA)
	k8s.GET("/clusters/:id/hpas/:ns/:name/yaml", p.read, ctl.GetHPAYAML)

	// PDB
	k8s.GET("/clusters/:id/pdbs", p.read, ctl.ListPDBs)
	k8s.PATCH("/clusters/:id/pdbs/edit", p.write, ctl.EditPDB)
	k8s.DELETE("/clusters/:id/pdbs/:ns/:name", p.write, ctl.DeletePDB)
	k8s.GET("/clusters/:id/pdbs/:ns/:name/yaml", p.read, ctl.GetPDBYAML)

	// Event
	k8s.GET("/clusters/:id/events", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListEvents)

	// Lease
	k8s.GET("/clusters/:id/leases", p.read, ctl.ListLeases)
	k8s.PATCH("/clusters/:id/leases/edit", p.write, ctl.EditLease)
	k8s.DELETE("/clusters/:id/leases/:ns/:name", p.write, ctl.DeleteLease)
	k8s.GET("/clusters/:id/leases/:ns/:name/yaml", p.read, ctl.GetLeaseYAML)

	// ResourceQuota
	k8s.GET("/clusters/:id/resourcequotas", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListResourceQuotas)
	k8s.PATCH("/clusters/:id/resourcequotas/edit", p.write, ctl.EditResourceQuota)
	k8s.DELETE("/clusters/:id/resourcequotas/:ns/:name", p.write, ctl.DeleteResourceQuota)
	k8s.GET("/clusters/:id/resourcequotas/:ns/:name/yaml", p.read, ctl.GetResourceQuotaYAML)

	// LimitRange
	k8s.GET("/clusters/:id/limitranges", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListLimitRanges)
	k8s.PATCH("/clusters/:id/limitranges/edit", p.write, ctl.EditLimitRange)
	k8s.DELETE("/clusters/:id/limitranges/:ns/:name", p.write, ctl.DeleteLimitRange)
	k8s.GET("/clusters/:id/limitranges/:ns/:name/yaml", p.read, ctl.GetLimitRangeYAML)

	// CustomResourceDefinition
	k8s.GET("/clusters/:id/customresourcedefinitions", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListCustomResourceDefinitions)
	k8s.PATCH("/clusters/:id/customresourcedefinitions/edit", p.write, ctl.EditCustomResourceDefinition)
	k8s.DELETE("/clusters/:id/customresourcedefinitions/:name", p.write, ctl.DeleteCustomResourceDefinition)
	k8s.GET("/clusters/:id/customresourcedefinitions/:name/yaml", p.read, ctl.GetCustomResourceDefinitionYAML)

	// APIService
	k8s.GET("/clusters/:id/apiservices", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListAPIServices)
	k8s.PATCH("/clusters/:id/apiservices/edit", p.write, ctl.EditAPIService)
	k8s.DELETE("/clusters/:id/apiservices/:name", p.write, ctl.DeleteAPIService)
	k8s.GET("/clusters/:id/apiservices/:name/yaml", p.read, ctl.GetAPIServiceYAML)

	// PriorityClass
	k8s.GET("/clusters/:id/priorityclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListPriorityClasses)
	k8s.PATCH("/clusters/:id/priorityclasses/edit", p.write, ctl.EditPriorityClass)
	k8s.DELETE("/clusters/:id/priorityclasses/:name", p.write, ctl.DeletePriorityClass)
	k8s.GET("/clusters/:id/priorityclasses/:name/yaml", p.read, ctl.GetPriorityClassYAML)

	// RuntimeClass
	k8s.GET("/clusters/:id/runtimeclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListRuntimeClasses)
	k8s.PATCH("/clusters/:id/runtimeclasses/edit", p.write, ctl.EditRuntimeClass)
	k8s.DELETE("/clusters/:id/runtimeclasses/:name", p.write, ctl.DeleteRuntimeClass)
	k8s.GET("/clusters/:id/runtimeclasses/:name/yaml", p.read, ctl.GetRuntimeClassYAML)

	// ValidatingWebhookConfiguration
	k8s.GET("/clusters/:id/validatingwebhookconfigurations", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListValidatingWebhookConfigurations)
	k8s.PATCH("/clusters/:id/validatingwebhookconfigurations/edit", p.write, ctl.EditValidatingWebhookConfiguration)
	k8s.DELETE("/clusters/:id/validatingwebhookconfigurations/:name", p.write, ctl.DeleteValidatingWebhookConfiguration)
	k8s.GET("/clusters/:id/validatingwebhookconfigurations/:name/yaml", p.read, ctl.GetValidatingWebhookConfigurationYAML)

	// MutatingWebhookConfiguration
	k8s.GET("/clusters/:id/mutatingwebhookconfigurations", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListMutatingWebhookConfigurations)
	k8s.PATCH("/clusters/:id/mutatingwebhookconfigurations/edit", p.write, ctl.EditMutatingWebhookConfiguration)
	k8s.DELETE("/clusters/:id/mutatingwebhookconfigurations/:name", p.write, ctl.DeleteMutatingWebhookConfiguration)
	k8s.GET("/clusters/:id/mutatingwebhookconfigurations/:name/yaml", p.read, ctl.GetMutatingWebhookConfigurationYAML)

	// ValidatingAdmissionPolicy
	k8s.GET("/clusters/:id/validatingadmissionpolicies", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListValidatingAdmissionPolicies)
	k8s.PATCH("/clusters/:id/validatingadmissionpolicies/edit", p.write, ctl.EditValidatingAdmissionPolicy)
	k8s.DELETE("/clusters/:id/validatingadmissionpolicies/:name", p.write, ctl.DeleteValidatingAdmissionPolicy)
	k8s.GET("/clusters/:id/validatingadmissionpolicies/:name/yaml", p.read, ctl.GetValidatingAdmissionPolicyYAML)

	// ValidatingAdmissionPolicyBinding
	k8s.GET("/clusters/:id/validatingadmissionpolicybindings", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListValidatingAdmissionPolicyBindings)
	k8s.PATCH("/clusters/:id/validatingadmissionpolicybindings/edit", p.write, ctl.EditValidatingAdmissionPolicyBinding)
	k8s.DELETE("/clusters/:id/validatingadmissionpolicybindings/:name", p.write, ctl.DeleteValidatingAdmissionPolicyBinding)
	k8s.GET("/clusters/:id/validatingadmissionpolicybindings/:name/yaml", p.read, ctl.GetValidatingAdmissionPolicyBindingYAML)

	// Resource support
	k8s.GET("/clusters/:id/resource-support", p.resourceSupportRead, ctl.GetResourceSupport)
	k8s.GET("/clusters/:id/storage-snapshot-support", p.storageSnapshotSupport, ctl.GetStorageSnapshotSupport)
}

// ── 工作负载：Pod / Deployment / StatefulSet / DaemonSet / ReplicaSet / Manifest ──

func registerWorkloadRoutes(a k8sRouteArgs) {
	k8s, ctl, p := a.k8s, a.ctl, a.perm

	// Pod
	k8s.GET("/clusters/:id/pods", p.read, ctl.ListPods)
	k8s.GET("/clusters/:id/podmetrics", p.read, ctl.ListPodMetrics)
	k8s.GET("/clusters/:id/pods/:ns/:pod/inspection", p.read, ctl.GetPodInspection)
	k8s.GET("/clusters/:id/pods/:ns/:pod/yaml", p.read, ctl.GetPodYAML)
	k8s.GET("/clusters/:id/pods/:ns/:pod/logs", p.read, ctl.GetPodLogs)
	k8s.POST("/clusters/:id/pods/:ns/:pod/logs/session", p.read, ctl.CreatePodLogSession)
	k8s.DELETE("/clusters/:id/pods/:ns/:pod", p.write, ctl.DeletePod)
	k8s.POST("/clusters/:id/pods/:ns/:pod/exec", p.exec, ctl.CreatePodExecSession)

	// Manifest
	k8s.GET("/clusters/:id/manifests/records", p.write, ctl.ListManifestRecords)
	k8s.GET("/clusters/:id/manifests/records/:recordId", p.write, ctl.GetManifestRecord)
	k8s.POST("/clusters/:id/manifests/apply", p.write, ctl.ApplyManifest)

	// Workload (Deployment/StatefulSet/DaemonSet)
	k8s.GET("/clusters/:id/workloads", p.read, ctl.ListWorkloads)
	k8s.GET("/clusters/:id/workloads/deployments/:ns/:name/rollout-history", p.read, ctl.GetRolloutHistory)
	k8s.POST("/clusters/:id/workloads/deployments/:ns/:name/rollout-undo", p.write, ctl.RolloutUndo)
	k8s.PATCH("/clusters/:id/workloads/scale", p.write, ctl.ScaleWorkload)
	k8s.PATCH("/clusters/:id/workloads/restart", p.write, ctl.RestartWorkload)
	k8s.PATCH("/clusters/:id/workloads/image", p.write, ctl.UpdateImage)
	k8s.PATCH("/clusters/:id/workloads/rollout-pause", p.write, ctl.UpdateWorkloadPaused)
	k8s.POST("/clusters/:id/workloads/deployments", p.write, ctl.CreateDeployment)
	k8s.POST("/clusters/:id/workloads/statefulsets", p.write, ctl.CreateStatefulSet)
	k8s.POST("/clusters/:id/workloads/daemonsets", p.write, ctl.CreateDaemonSet)
	k8s.PATCH("/clusters/:id/workloads/deployments/edit", p.write, ctl.EditDeployment)
	k8s.PATCH("/clusters/:id/workloads/statefulsets/edit", p.write, ctl.EditStatefulSet)
	k8s.PATCH("/clusters/:id/workloads/daemonsets/edit", p.write, ctl.EditDaemonSet)
	k8s.PATCH("/clusters/:id/workloads/yaml/edit", p.write, ctl.EditWorkloadYAML)
	k8s.DELETE("/clusters/:id/workloads/:kind/:ns/:name", p.write, ctl.DeleteWorkload)
	k8s.GET("/clusters/:id/workloads/:kind/:ns/:name/yaml", p.read, ctl.GetWorkloadYAML)

	// ReplicaSet
	k8s.GET("/clusters/:id/replicasets", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListReplicaSets)
	k8s.PATCH("/clusters/:id/replicasets/edit", p.write, ctl.EditReplicaSet)
	k8s.DELETE("/clusters/:id/replicasets/:ns/:name", p.write, ctl.DeleteReplicaSet)
	k8s.GET("/clusters/:id/replicasets/:ns/:name/yaml", p.read, ctl.GetReplicaSetYAML)
}

// ── 网络：Service / Ingress / IngressClass / NetworkPolicy / Endpoints / EndpointSlice ──

func registerNetworkingRoutes(a k8sRouteArgs) {
	k8s, ctl, p := a.k8s, a.ctl, a.perm

	// Service
	k8s.GET("/clusters/:id/services", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListServices)
	k8s.POST("/clusters/:id/services", p.write, ctl.CreateService)
	k8s.PATCH("/clusters/:id/services/edit", p.write, ctl.EditService)
	k8s.DELETE("/clusters/:id/services/:ns/:name", p.write, ctl.DeleteService)
	k8s.GET("/clusters/:id/services/:ns/:name/yaml", p.read, ctl.GetServiceYAML)

	// Ingress
	k8s.GET("/clusters/:id/ingresses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListIngresses)
	k8s.POST("/clusters/:id/ingresses", p.write, ctl.CreateIngress)
	k8s.PATCH("/clusters/:id/ingresses/edit", p.write, ctl.EditIngress)
	k8s.DELETE("/clusters/:id/ingresses/:ns/:name", p.write, ctl.DeleteIngress)
	k8s.GET("/clusters/:id/ingresses/:ns/:name/yaml", p.read, ctl.GetIngressYAML)

	// NetworkPolicy
	k8s.GET("/clusters/:id/networkpolicies", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListNetworkPolicies)
	k8s.PATCH("/clusters/:id/networkpolicies/edit", p.write, ctl.EditNetworkPolicy)
	k8s.DELETE("/clusters/:id/networkpolicies/:ns/:name", p.write, ctl.DeleteNetworkPolicy)
	k8s.GET("/clusters/:id/networkpolicies/:ns/:name/yaml", p.read, ctl.GetNetworkPolicyYAML)

	// IngressClass
	k8s.GET("/clusters/:id/ingressclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListIngressClasses)
	k8s.PATCH("/clusters/:id/ingressclasses/edit", p.write, ctl.EditIngressClass)
	k8s.DELETE("/clusters/:id/ingressclasses/:name", p.write, ctl.DeleteIngressClass)
	k8s.GET("/clusters/:id/ingressclasses/:name/yaml", p.read, ctl.GetIngressClassYAML)

	// Endpoints
	k8s.GET("/clusters/:id/endpoints", p.read, ctl.ListEndpoints)
	k8s.PATCH("/clusters/:id/endpoints/edit", p.write, ctl.EditEndpoints)
	k8s.DELETE("/clusters/:id/endpoints/:ns/:name", p.write, ctl.DeleteEndpoints)
	k8s.GET("/clusters/:id/endpoints/:ns/:name/yaml", p.read, ctl.GetEndpointsYAML)

	// EndpointSlice
	k8s.GET("/clusters/:id/endpointslices", p.read, ctl.ListEndpointSlices)
	k8s.PATCH("/clusters/:id/endpointslices/edit", p.write, ctl.EditEndpointSlice)
	k8s.DELETE("/clusters/:id/endpointslices/:ns/:name", p.write, ctl.DeleteEndpointSlice)
	k8s.GET("/clusters/:id/endpointslices/:ns/:name/yaml", p.read, ctl.GetEndpointSliceYAML)
}

// ── 配置与存储：ConfigMap / Secret / ServiceAccount / PVC / PV / StorageClass / CSI / VolumeSnapshot / VolumeAttachment ──

func registerConfigStorageRoutes(a k8sRouteArgs) {
	k8s, ctl, p := a.k8s, a.ctl, a.perm

	// ConfigMap
	k8s.GET("/clusters/:id/configmaps", p.read, ctl.ListConfigMaps)
	k8s.PATCH("/clusters/:id/configmaps/edit", p.write, ctl.EditConfigMap)
	k8s.DELETE("/clusters/:id/configmaps/:ns/:name", p.write, ctl.DeleteConfigMap)
	k8s.GET("/clusters/:id/configmaps/:ns/:name/yaml", p.read, ctl.GetConfigMapYAML)
	k8s.GET("/clusters/:id/configmaps/:ns/:name/related", p.read, ctl.GetConfigMapRelated)

	// Secret
	k8s.GET("/clusters/:id/secrets", p.read, ctl.ListSecrets)
	k8s.PATCH("/clusters/:id/secrets/edit", p.write, ctl.EditSecret)
	k8s.DELETE("/clusters/:id/secrets/:ns/:name", p.write, ctl.DeleteSecret)
	k8s.GET("/clusters/:id/secrets/:ns/:name/reveal", p.secretReveal, ctl.GetSecretReveal)
	k8s.GET("/clusters/:id/secrets/:ns/:name/yaml", p.read, ctl.GetSecretYAML)
	k8s.GET("/clusters/:id/secrets/:ns/:name/related", p.read, ctl.GetSecretRelated)

	// ServiceAccount
	k8s.GET("/clusters/:id/serviceaccounts", p.read, ctl.ListServiceAccounts)
	k8s.PATCH("/clusters/:id/serviceaccounts/edit", p.write, ctl.EditServiceAccount)
	k8s.DELETE("/clusters/:id/serviceaccounts/:ns/:name", p.write, ctl.DeleteServiceAccount)
	k8s.GET("/clusters/:id/serviceaccounts/:ns/:name/yaml", p.read, ctl.GetServiceAccountYAML)

	// PVC
	k8s.GET("/clusters/:id/pvcs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListPVCs)
	k8s.POST("/clusters/:id/pvcs", p.write, ctl.CreatePVC)
	k8s.DELETE("/clusters/:id/pvcs/:ns/:name", p.write, ctl.DeletePVC)
	k8s.GET("/clusters/:id/pvcs/:ns/:name/yaml", p.read, ctl.GetPVCYAML)

	// PV
	k8s.GET("/clusters/:id/pvs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListPVs)
	k8s.DELETE("/clusters/:id/pvs/:name", p.write, ctl.DeletePV)
	k8s.GET("/clusters/:id/pvs/:name/yaml", p.read, ctl.GetPVYAML)

	// StorageClass
	k8s.GET("/clusters/:id/storageclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListStorageClasses)
	k8s.PATCH("/clusters/:id/storageclasses/edit", p.write, ctl.EditStorageClass)
	k8s.DELETE("/clusters/:id/storageclasses/:name", p.write, ctl.DeleteStorageClass)
	k8s.GET("/clusters/:id/storageclasses/:name/yaml", p.read, ctl.GetStorageClassYAML)

	// CSIDriver
	k8s.GET("/clusters/:id/csidrivers", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListCSIDrivers)
	k8s.PATCH("/clusters/:id/csidrivers/edit", p.write, ctl.EditCSIDriver)
	k8s.DELETE("/clusters/:id/csidrivers/:name", p.write, ctl.DeleteCSIDriver)
	k8s.GET("/clusters/:id/csidrivers/:name/yaml", p.read, ctl.GetCSIDriverYAML)

	// CSINode
	k8s.GET("/clusters/:id/csinodes", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListCSINodes)
	k8s.PATCH("/clusters/:id/csinodes/edit", p.write, ctl.EditCSINode)
	k8s.DELETE("/clusters/:id/csinodes/:name", p.write, ctl.DeleteCSINode)
	k8s.GET("/clusters/:id/csinodes/:name/yaml", p.read, ctl.GetCSINodeYAML)

	// CSIStorageCapacity
	k8s.GET("/clusters/:id/csistoragecapacities", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListCSIStorageCapacities)
	k8s.PATCH("/clusters/:id/csistoragecapacities/edit", p.write, ctl.EditCSIStorageCapacity)
	k8s.DELETE("/clusters/:id/csistoragecapacities/:ns/:name", p.write, ctl.DeleteCSIStorageCapacity)
	k8s.GET("/clusters/:id/csistoragecapacities/:ns/:name/yaml", p.read, ctl.GetCSIStorageCapacityYAML)

	// VolumeAttachment
	k8s.GET("/clusters/:id/volumeattachments", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListVolumeAttachments)
	k8s.PATCH("/clusters/:id/volumeattachments/edit", p.write, ctl.EditVolumeAttachment)
	k8s.DELETE("/clusters/:id/volumeattachments/:name", p.write, ctl.DeleteVolumeAttachment)
	k8s.GET("/clusters/:id/volumeattachments/:name/yaml", p.read, ctl.GetVolumeAttachmentYAML)

	// VolumeSnapshot
	k8s.GET("/clusters/:id/volumesnapshots", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListVolumeSnapshots)
	k8s.PATCH("/clusters/:id/volumesnapshots/edit", p.write, ctl.EditVolumeSnapshot)
	k8s.DELETE("/clusters/:id/volumesnapshots/:ns/:name", p.write, ctl.DeleteVolumeSnapshot)
	k8s.GET("/clusters/:id/volumesnapshots/:ns/:name/yaml", p.read, ctl.GetVolumeSnapshotYAML)

	// VolumeSnapshotClass
	k8s.GET("/clusters/:id/volumesnapshotclasses", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListVolumeSnapshotClasses)
	k8s.PATCH("/clusters/:id/volumesnapshotclasses/edit", p.write, ctl.EditVolumeSnapshotClass)
	k8s.DELETE("/clusters/:id/volumesnapshotclasses/:name", p.write, ctl.DeleteVolumeSnapshotClass)
	k8s.GET("/clusters/:id/volumesnapshotclasses/:name/yaml", p.read, ctl.GetVolumeSnapshotClassYAML)

	// VolumeSnapshotContent
	k8s.GET("/clusters/:id/volumesnapshotcontents", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListVolumeSnapshotContents)
	k8s.PATCH("/clusters/:id/volumesnapshotcontents/edit", p.write, ctl.EditVolumeSnapshotContent)
	k8s.DELETE("/clusters/:id/volumesnapshotcontents/:name", p.write, ctl.DeleteVolumeSnapshotContent)
	k8s.GET("/clusters/:id/volumesnapshotcontents/:name/yaml", p.read, ctl.GetVolumeSnapshotContentYAML)
}

// ── RBAC：Role / ClusterRole / RoleBinding / ClusterRoleBinding ──

func registerRBACRoutes(a k8sRouteArgs) {
	k8s, ctl, p := a.k8s, a.ctl, a.perm

	// Role
	k8s.GET("/clusters/:id/roles", p.rbacRead, ctl.ListRoles)
	k8s.PATCH("/clusters/:id/roles/edit", p.rbacWrite, ctl.EditRole)
	k8s.DELETE("/clusters/:id/roles/:ns/:name", p.rbacWrite, ctl.DeleteRole)
	k8s.GET("/clusters/:id/roles/:ns/:name/yaml", p.rbacRead, ctl.GetRoleYAML)

	// ClusterRole
	k8s.GET("/clusters/:id/clusterroles", p.rbacRead, ctl.ListClusterRoles)
	k8s.PATCH("/clusters/:id/clusterroles/edit", p.rbacWrite, ctl.EditClusterRole)
	k8s.DELETE("/clusters/:id/clusterroles/:name", p.rbacWrite, ctl.DeleteClusterRole)
	k8s.GET("/clusters/:id/clusterroles/:name/yaml", p.rbacRead, ctl.GetClusterRoleYAML)

	// RoleBinding
	k8s.GET("/clusters/:id/rolebindings", p.rbacRead, ctl.ListRoleBindings)
	k8s.PATCH("/clusters/:id/rolebindings/edit", p.rbacWrite, ctl.EditRoleBinding)
	k8s.DELETE("/clusters/:id/rolebindings/:ns/:name", p.rbacWrite, ctl.DeleteRoleBinding)
	k8s.GET("/clusters/:id/rolebindings/:ns/:name/yaml", p.rbacRead, ctl.GetRoleBindingYAML)

	// ClusterRoleBinding
	k8s.GET("/clusters/:id/clusterrolebindings", p.rbacRead, ctl.ListClusterRoleBindings)
	k8s.PATCH("/clusters/:id/clusterrolebindings/edit", p.rbacWrite, ctl.EditClusterRoleBinding)
	k8s.DELETE("/clusters/:id/clusterrolebindings/:name", p.rbacWrite, ctl.DeleteClusterRoleBinding)
	k8s.GET("/clusters/:id/clusterrolebindings/:name/yaml", p.rbacRead, ctl.GetClusterRoleBindingYAML)
}

// ── 批处理：Job / CronJob ──

func registerBatchRoutes(a k8sRouteArgs) {
	k8s, ctl, p := a.k8s, a.ctl, a.perm

	// Job
	k8s.GET("/clusters/:id/jobs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListJobs)
	k8s.PATCH("/clusters/:id/jobs/edit", p.write, ctl.EditJob)
	k8s.DELETE("/clusters/:id/jobs/completed", p.write, ctl.DeleteCompletedJobs)
	k8s.DELETE("/clusters/:id/jobs/:ns/:name", p.write, ctl.DeleteJob)
	k8s.GET("/clusters/:id/jobs/:ns/:name/yaml", p.read, ctl.GetJobYAML)

	// CronJob
	k8s.GET("/clusters/:id/cronjobs", p.read, middleware.CacheJSON(a.d.CacheStore, a.d.CacheTTL), ctl.ListCronJobs)
	k8s.PATCH("/clusters/:id/cronjobs/edit", p.write, ctl.EditCronJob)
	k8s.POST("/clusters/:id/cronjobs/:ns/:name/trigger", p.write, ctl.TriggerCronJob)
	k8s.PATCH("/clusters/:id/cronjobs/:ns/:name/suspend", p.write, ctl.SuspendCronJob)
	k8s.DELETE("/clusters/:id/cronjobs/:ns/:name", p.write, ctl.DeleteCronJob)
	k8s.GET("/clusters/:id/cronjobs/:ns/:name/yaml", p.read, ctl.GetCronJobYAML)
}

// ── WebSocket ──

func registerWebSocketRoutes(authed *gin.RouterGroup, k8sCtl *controller.K8sController) {
	ws := authed.Group("/ws")
	if k8sCtl != nil {
		ws.GET("/pod-log", middleware.RequirePerm("k8s:read"), k8sCtl.PodLogWS)
		ws.GET("/pod-exec", middleware.RequirePerm("k8s:exec"), k8sCtl.PodExecWS)
	}
}

func registerAuditRoutes(authed *gin.RouterGroup, ctl *controller.AuditController) {
	if ctl == nil {
		return
	}
	authed.GET("/audit-logs", middleware.RequirePerm("user:read"), ctl.List)
}

func registerUserRoutes(authed *gin.RouterGroup, ctl *controller.UserController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePerm("user:read")
	write := middleware.RequirePerm("user:write")

	// 用户管理
	authed.GET("/users", read, ctl.ListUsers)
	authed.POST("/users", write, ctl.CreateUser)
	authed.PUT("/users/:id", write, ctl.UpdateUser)
	authed.DELETE("/users/:id", write, ctl.DeleteUser)
	authed.POST("/users/:id/reset-password", write, ctl.ResetPassword)

	// 角色管理
	authed.GET("/roles", read, ctl.ListRoles)
	authed.POST("/roles", write, ctl.CreateRole)
	authed.PUT("/roles/:id", write, ctl.UpdateRole)
	authed.DELETE("/roles/:id", write, ctl.DeleteRole)

	// 权限点列表
	authed.GET("/permissions", read, ctl.ListPermissions)
}

func registerAIRoutes(authed *gin.RouterGroup, ctl *controller.AIController) {
	if ctl == nil {
		return
	}

	aiReadPerm := middleware.RequireAnyPerm("ai:chat", "ai:diagnose", "ai:image", "ai:model_admin")
	aiWritePerm := middleware.RequirePerm("ai:model_admin")
	clusterReadPerm := middleware.RequirePerm("cluster:read")

	ai := authed.Group("/ai")
	ai.GET("/providers", aiWritePerm, ctl.ListProviders)
	ai.POST("/providers", aiWritePerm, ctl.CreateProvider)
	ai.PATCH("/providers/:id", aiWritePerm, ctl.PatchProvider)
	ai.GET("/models", aiReadPerm, ctl.ListModels)
	ai.GET("/tools", aiReadPerm, ctl.ListTools)
	ai.POST("/models", aiWritePerm, ctl.CreateModel)
	ai.PATCH("/models/:id", aiWritePerm, ctl.PatchModel)
	ai.GET("/route-settings", aiWritePerm, ctl.GetRouteSettings)
	ai.PUT("/route-settings", aiWritePerm, ctl.UpdateRouteSettings)
	ai.GET("/conversations", aiReadPerm, ctl.ListConversations)
	ai.GET("/conversations/:id", aiReadPerm, ctl.GetConversation)
	ai.DELETE("/conversations/:id", aiReadPerm, ctl.DeleteConversation)

	clusters := authed.Group("/clusters")
	clusters.POST("/:id/ai/conversations", clusterReadPerm, middleware.RequireAnyPerm("ai:chat", "ai:diagnose"), ctl.CreateConversation)
	clusters.POST("/:id/ai/chat", clusterReadPerm, middleware.RequireAnyPerm("ai:chat", "ai:diagnose"), ctl.SendChat)
	clusters.POST("/:id/ai/chat/stream", clusterReadPerm, middleware.RequireAnyPerm("ai:chat", "ai:diagnose"), ctl.SendChatStream)
	clusters.POST("/:id/ai/actions/propose", clusterReadPerm, middleware.RequirePerm("ai:change_propose"), ctl.CreateActionProposal)
	clusters.POST("/:id/ai/actions/:actionId/confirm", clusterReadPerm, middleware.RequirePerm("ai:change_confirm"), middleware.RequirePerm("k8s:write"), ctl.ConfirmActionProposal)
}
