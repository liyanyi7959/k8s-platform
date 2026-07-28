package router

import (
	"github.com/gin-gonic/gin"

	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	"k8s-platform-backend/internal/legacy/controller"
	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/internal/middleware"
)

func registerIncidentV2Routes(r *gin.Engine, d Deps, auditSvc *service.AuditService, ctl *incidenthttp.Controller) {
	if ctl == nil {
		return
	}
	v2 := r.Group("/api/v2")
	v2.Use(middleware.AuthRequiredV2(d.JWTMgr, d.RbacSvc))
	if auditSvc != nil {
		v2.Use(middleware.AuditLogger(auditSvc))
	}
	read := middleware.RequirePermV2("monitor:read")
	manage := middleware.RequirePermV2("incident:manage")
	incidents := v2.Group("/incidents")
	incidents.GET("", read, ctl.List)
	incidents.GET("/:id", read, ctl.Get)
	incidents.POST("/:id/acknowledgements", manage, ctl.Acknowledge)
	incidents.POST("/:id/diagnosis-runs", manage, ctl.Diagnose)
	incidents.POST("/:id/approval-requests", manage, ctl.RequestApproval)
	incidents.POST("/:id/execution-starts", manage, ctl.StartExecution)
	incidents.POST("/:id/verification-runs", manage, ctl.StartVerification)
	incidents.POST("/:id/resolution-attempts", manage, ctl.Resolve)
}

func registerAutomationTaskRoutes(authed *gin.RouterGroup, ctl *controller.AutomationTaskController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePerm("automation:read")
	execute := middleware.RequirePerm("automation:execute")
	tasks := authed.Group("/automation/tasks")
	tasks.GET("", read, ctl.List)
	tasks.GET("/:id", read, ctl.Get)
	tasks.GET("/:id/logs", read, ctl.Logs)
	tasks.POST("/:id/cancel", execute, ctl.Cancel)
	tasks.POST("/:id/cancellation-requests", execute, ctl.Cancel)
}

func registerMonitorIncidentRoutes(authed *gin.RouterGroup, ctl *controller.MonitorIncidentController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePerm("monitor:read")
	write := middleware.RequirePerm("monitor:write")
	manage := middleware.RequirePerm("incident:manage")
	monitor := authed.Group("/monitor")
	monitor.GET("/alerts", read, ctl.ListAlertRules)
	monitor.POST("/alerts", write, ctl.CreateAlertRule)
	monitor.PUT("/alerts/:id", write, ctl.UpdateAlertRule)
	monitor.PUT("/alerts/:id/toggle", write, ctl.ToggleAlertRule)
	monitor.PATCH("/alerts/:id", write, ctl.ToggleAlertRule)
	monitor.DELETE("/alerts/:id", write, ctl.DeleteAlertRule)
	monitor.GET("/incidents", read, ctl.ListIncidents)
	monitor.GET("/incidents/:id", read, ctl.GetIncident)
	monitor.POST("/incidents/:id/transition", manage, ctl.TransitionIncident)
	monitor.POST("/incidents/:id/ai-conversation", manage, ctl.LinkAIConversation)
	monitor.POST("/incidents/:id/ai-proposal", manage, ctl.LinkAIProposal)
}

func registerDeployRoutes(authed *gin.RouterGroup, ctl *controller.DeployController, configCtl *controller.DeployConfigController) {
	if ctl == nil {
		return
	}
	deploy := authed.Group("/deploy")

	readServer := middleware.RequirePerm("deploy:server_read")
	writeServer := middleware.RequirePerm("deploy:server_write")
	deleteServer := middleware.RequirePerm("deploy:server_delete")
	readCredential := middleware.RequirePerm("credential:read")
	writeCredential := middleware.RequirePerm("credential:write")
	deleteCredential := middleware.RequirePerm("credential:delete")
	readPlan := middleware.RequirePerm("deploy:plan_read")
	writePlan := middleware.RequirePerm("deploy:plan_write")
	deletePlan := middleware.RequirePerm("deploy:plan_delete")
	execDeploy := middleware.RequirePerm("deploy:execute")

	// 服务器管理
	deploy.GET("/servers", readServer, ctl.ListServers)
	deploy.GET("/servers/summary", readServer, ctl.GetServerSummary)
	deploy.POST("/servers", writeServer, ctl.CreateServer)
	deploy.GET("/servers/:id", readServer, ctl.GetServer)
	deploy.PUT("/servers/:id", writeServer, ctl.UpdateServer)
	deploy.POST("/servers/:id/test-ssh", readServer, ctl.TestSSH)
	deploy.POST("/servers/:id/connection-checks", writeServer, ctl.TestSSH)
	deploy.POST("/servers/:id/terminal-session", writeServer, ctl.CreateServerTerminalSession)
	deploy.POST("/servers/:id/terminal-sessions", writeServer, ctl.CreateServerTerminalSession)
	deploy.GET("/servers/terminal/ws", writeServer, ctl.ServerTerminalWS)
	deploy.DELETE("/servers/:id", deleteServer, ctl.DeleteServer)

	// SSH 凭证
	deploy.GET("/credentials", readCredential, ctl.ListCredentials)
	deploy.POST("/credentials", writeCredential, ctl.CreateCredential)
	deploy.GET("/credentials/:id", readCredential, ctl.GetCredential)
	deploy.PUT("/credentials/:id", writeCredential, ctl.UpdateCredential)
	deploy.DELETE("/credentials/:id", deleteCredential, ctl.DeleteCredential)
	deploy.POST("/credentials/batch-delete", deleteCredential, ctl.BatchDeleteCredentials)
	deploy.POST("/credential-deletion-requests", deleteCredential, ctl.BatchDeleteCredentials)

	// 部署计划
	deploy.GET("/plans", readPlan, ctl.ListPlans)
	deploy.POST("/plans", writePlan, ctl.CreatePlan)
	deploy.GET("/plans/:id", readPlan, ctl.GetPlan)
	deploy.PUT("/plans/:id", writePlan, ctl.UpdatePlan)
	deploy.GET("/plans/:id/dry-run", readPlan, ctl.DryRunPlan)
	deploy.POST("/plans/:id/preflight", execDeploy, ctl.PreflightPlan)
	deploy.POST("/plans/:id/preflight/ignore", execDeploy, ctl.SetPreflightIgnore)
	deploy.GET("/plans/:id/simulations", readPlan, ctl.DryRunPlan)
	deploy.POST("/plans/:id/preflight-checks", execDeploy, ctl.PreflightPlan)
	deploy.POST("/plans/:id/preflight-checks/overrides", execDeploy, ctl.SetPreflightIgnore)
	deploy.GET("/plans/:id/ansible-config", readPlan, ctl.GetPlanAnsibleConfig)
	deploy.POST("/plans/:id/execute", execDeploy, ctl.ExecutePlan)
	deploy.POST("/plans/:id/cancel", execDeploy, ctl.CancelPlan)
	deploy.POST("/plans/:id/retry", execDeploy, ctl.RetryPlan)
	deploy.POST("/plans/:id/steps/:stepKey/retry", execDeploy, ctl.RetryDeployStep)
	deploy.POST("/plans/:id/executions", execDeploy, ctl.ExecutePlan)
	deploy.POST("/plans/:id/cancellation-requests", execDeploy, ctl.CancelPlan)
	deploy.POST("/plans/:id/retry-attempts", execDeploy, ctl.RetryPlan)
	deploy.POST("/plans/:id/steps/:stepKey/retry-attempts", execDeploy, ctl.RetryDeployStep)
	deploy.GET("/plans/:id/addons/task", readPlan, ctl.GetPlanAddonTask)
	deploy.POST("/plans/:id/addons/install", execDeploy, ctl.InstallPlanAddons)
	deploy.POST("/plans/:id/addons/retry", execDeploy, ctl.RetryPlanAddons)
	deploy.POST("/plans/:id/addon-installations", execDeploy, ctl.InstallPlanAddons)
	deploy.POST("/plans/:id/addon-retry-attempts", execDeploy, ctl.RetryPlanAddons)
	deploy.DELETE("/plans/:id", deletePlan, ctl.DeletePlan)

	// 部署任务日志
	deploy.GET("/tasks/:taskId", readPlan, ctl.GetDeployTask)
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

		// Ansible 部署配置展示
		deploy.GET("/ansible/playbook", readPlan, configCtl.GetAnsiblePlaybook)
		deploy.GET("/ansible/inventory-template", readPlan, configCtl.GetAnsibleInventoryTemplate)
		deploy.GET("/ansible/env-check", readPlan, configCtl.CheckAnsibleEnv)
		deploy.GET("/ansible/tree", readPlan, configCtl.GetAnsibleTree)
	}
}

// ── 项目管理 ──

func registerProjectRoutes(authed *gin.RouterGroup, ctl *controller.ProjectController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePerm("project:read")
	write := middleware.RequirePerm("project:write")
	authed.GET("/projects", read, ctl.ListProjects)
	authed.GET("/projects/:id", read, ctl.GetProject)
	authed.POST("/projects", write, ctl.CreateProject)
	authed.PUT("/projects/:id", write, ctl.UpdateProject)
	authed.DELETE("/projects/:id", write, ctl.DeleteProject)
	// 项目下所有命名空间的资源统计
	authed.GET("/projects/:id/resources", read, ctl.GetProjectResources)
	// 分配命名空间到项目
	authed.PUT("/projects/:id/namespaces", write, ctl.AssignNamespaces)
}

// ── 应用商店 ──

func registerAppTemplateRoutes(authed *gin.RouterGroup, ctl *controller.AppTemplateController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePerm("appstore:read")
	write := middleware.RequirePerm("appstore:write")
	authed.GET("/app-templates", read, ctl.ListAppTemplates)
	authed.GET("/app-templates/:id", read, ctl.GetAppTemplate)
	authed.POST("/app-templates", write, ctl.CreateAppTemplate)
	authed.PUT("/app-templates/:id", write, ctl.UpdateAppTemplate)
	authed.DELETE("/app-templates/:id", write, ctl.DeleteAppTemplate)
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
	audits.POST("/:id/cancellation-requests", auditPerm, ctl.Cancel)
	audits.POST("/ad-hoc-audits", auditPerm, ctl.CreateAdhoc)
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
	clusters.POST("/:id/health-checks", ctl.CheckHealth)
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
