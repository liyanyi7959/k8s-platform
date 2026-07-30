package router

import (
	"github.com/gin-gonic/gin"

	legacyprovision "k8s-platform-backend/internal/integration/provisioning"
	"k8s-platform-backend/internal/middleware"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
)

func registerDeployRoutes(authed *gin.RouterGroup, serverAccess *legacyprovision.ServerAccessController, serverCtl *provisionhttp.ServerController, credentialCtl *provisionhttp.CredentialController, planCtl *provisionhttp.DeployPlanController, runtimeCtl *provisionhttp.RuntimeController, taskCtl *provisionhttp.TaskController, configCtl *provisionhttp.DeployConfigController) {
	if serverAccess == nil || serverCtl == nil || credentialCtl == nil || planCtl == nil || runtimeCtl == nil || taskCtl == nil {
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

	deploy.GET("/servers", readServer, serverCtl.List)
	deploy.GET("/servers/summary", readServer, serverCtl.Summary)
	deploy.POST("/servers", writeServer, serverCtl.Create)
	deploy.GET("/servers/:id", readServer, serverCtl.Get)
	deploy.PUT("/servers/:id", writeServer, serverCtl.Update)
	deploy.POST("/servers/:id/test-ssh", readServer, serverAccess.TestSSH)
	deploy.POST("/servers/:id/connection-checks", writeServer, serverAccess.TestSSH)
	deploy.POST("/servers/:id/terminal-session", writeServer, serverAccess.CreateTerminalSession)
	deploy.POST("/servers/:id/terminal-sessions", writeServer, serverAccess.CreateTerminalSession)
	deploy.GET("/servers/terminal/ws", writeServer, serverAccess.TerminalWS)
	deploy.DELETE("/servers/:id", deleteServer, serverCtl.Delete)

	deploy.GET("/credentials", readCredential, credentialCtl.List)
	deploy.POST("/credentials", writeCredential, credentialCtl.Create)
	deploy.GET("/credentials/:id", readCredential, credentialCtl.Get)
	deploy.PUT("/credentials/:id", writeCredential, credentialCtl.Update)
	deploy.DELETE("/credentials/:id", deleteCredential, credentialCtl.Delete)
	deploy.POST("/credentials/batch-delete", deleteCredential, credentialCtl.BatchDelete)
	deploy.POST("/credential-deletion-requests", deleteCredential, credentialCtl.BatchDelete)

	deploy.GET("/plans", readPlan, planCtl.List)
	deploy.POST("/plans", writePlan, planCtl.Create)
	deploy.GET("/plans/:id", readPlan, planCtl.Get)
	deploy.PUT("/plans/:id", writePlan, planCtl.Update)
	deploy.GET("/plans/:id/dry-run", readPlan, runtimeCtl.DryRun)
	deploy.POST("/plans/:id/preflight", execDeploy, runtimeCtl.Preflight)
	deploy.POST("/plans/:id/preflight/ignore", execDeploy, runtimeCtl.SetPreflightIgnore)
	deploy.GET("/plans/:id/simulations", readPlan, runtimeCtl.DryRun)
	deploy.POST("/plans/:id/preflight-checks", execDeploy, runtimeCtl.Preflight)
	deploy.POST("/plans/:id/preflight-checks/overrides", execDeploy, runtimeCtl.SetPreflightIgnore)
	deploy.GET("/plans/:id/ansible-config", readPlan, runtimeCtl.AnsibleConfig)
	deploy.POST("/plans/:id/execute", execDeploy, runtimeCtl.Execute)
	deploy.POST("/plans/:id/cancel", execDeploy, runtimeCtl.Cancel)
	deploy.POST("/plans/:id/retry", execDeploy, runtimeCtl.Retry)
	deploy.POST("/plans/:id/steps/:stepKey/retry", execDeploy, runtimeCtl.RetryStep)
	deploy.POST("/plans/:id/executions", execDeploy, runtimeCtl.Execute)
	deploy.POST("/plans/:id/cancellation-requests", execDeploy, runtimeCtl.Cancel)
	deploy.POST("/plans/:id/retry-attempts", execDeploy, runtimeCtl.Retry)
	deploy.POST("/plans/:id/steps/:stepKey/retry-attempts", execDeploy, runtimeCtl.RetryStep)
	deploy.GET("/plans/:id/addons/task", readPlan, runtimeCtl.LatestAddonTask)
	deploy.POST("/plans/:id/addons/install", execDeploy, runtimeCtl.InstallAddons)
	deploy.POST("/plans/:id/addons/retry", execDeploy, runtimeCtl.RetryAddons)
	deploy.POST("/plans/:id/addon-installations", execDeploy, runtimeCtl.InstallAddons)
	deploy.POST("/plans/:id/addon-retry-attempts", execDeploy, runtimeCtl.RetryAddons)
	deploy.DELETE("/plans/:id", deletePlan, planCtl.Delete)

	deploy.GET("/tasks/:taskId", readPlan, taskCtl.Get)
	deploy.GET("/tasks/:taskId/logs", readPlan, taskCtl.Logs)
	deploy.GET("/tasks/:taskId/logs/sse", readPlan, taskCtl.LogsSSE)

	if configCtl != nil {
		deploy.GET("/configs", readPlan, configCtl.ListConfigs)
		deploy.GET("/configs/os-types", readPlan, configCtl.ListSupportedOSTypes)
		deploy.GET("/configs/:id", readPlan, configCtl.GetConfig)
		deploy.PUT("/configs/:id", writePlan, configCtl.UpdateConfig)
		deploy.GET("/configs/:id/versions", readPlan, configCtl.GetConfigVersions)

		deploy.GET("/repositories", readPlan, configCtl.ListRepositories)
		deploy.POST("/repositories", writePlan, configCtl.CreateRepository)
		deploy.GET("/repositories/:id", readPlan, configCtl.GetRepository)
		deploy.PUT("/repositories/:id", writePlan, configCtl.UpdateRepository)
		deploy.DELETE("/repositories/:id", deletePlan, configCtl.DeleteRepository)

		deploy.GET("/ansible/playbook", readPlan, configCtl.GetAnsiblePlaybook)
		deploy.GET("/ansible/inventory-template", readPlan, configCtl.GetAnsibleInventoryTemplate)
		deploy.GET("/ansible/env-check", readPlan, configCtl.CheckAnsibleEnv)
		deploy.GET("/ansible/tree", readPlan, configCtl.GetAnsibleTree)
	}
}
