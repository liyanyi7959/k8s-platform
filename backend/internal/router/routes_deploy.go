package router

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
)

func registerDeployRoutes(authed *gin.RouterGroup, serverAccess *provisionhttp.ServerAccessController, serverCtl *provisionhttp.ServerController, credentialCtl *provisionhttp.CredentialController, planCtl *provisionhttp.DeployPlanController, runtimeCtl *provisionhttp.RuntimeController, taskCtl *provisionhttp.TaskController, configCtl *provisionhttp.DeployConfigController) {
	if serverAccess == nil || serverCtl == nil || credentialCtl == nil || planCtl == nil || runtimeCtl == nil || taskCtl == nil {
		return
	}
	deploy := authed.Group("/provisioning")

	readServer := middleware.RequirePermV2("deploy:server_read")
	writeServer := middleware.RequirePermV2("deploy:server_write")
	deleteServer := middleware.RequirePermV2("deploy:server_delete")
	readCredential := middleware.RequirePermV2("credential:read")
	writeCredential := middleware.RequirePermV2("credential:write")
	deleteCredential := middleware.RequirePermV2("credential:delete")
	readPlan := middleware.RequirePermV2("deploy:plan_read")
	writePlan := middleware.RequirePermV2("deploy:plan_write")
	deletePlan := middleware.RequirePermV2("deploy:plan_delete")
	execDeploy := middleware.RequirePermV2("deploy:execute")

	deploy.GET("/servers", readServer, serverCtl.List)
	deploy.GET("/servers/summary", readServer, serverCtl.Summary)
	deploy.POST("/servers", writeServer, serverCtl.Create)
	deploy.GET("/servers/:id", readServer, serverCtl.Get)
	deploy.PATCH("/servers/:id", writeServer, serverCtl.Update)
	deploy.POST("/servers/:id/connection-checks", writeServer, serverAccess.TestSSH)
	deploy.POST("/servers/:id/terminal-tickets", writeServer, serverAccess.CreateTerminalSession)
	deploy.DELETE("/servers/:id", deleteServer, serverCtl.Delete)

	deploy.GET("/credentials", readCredential, credentialCtl.List)
	deploy.POST("/credentials", writeCredential, credentialCtl.Create)
	deploy.GET("/credentials/:id", readCredential, credentialCtl.Get)
	deploy.PATCH("/credentials/:id", writeCredential, credentialCtl.Update)
	deploy.DELETE("/credentials/:id", deleteCredential, credentialCtl.Delete)
	deploy.POST("/credential-deletion-requests", deleteCredential, credentialCtl.BatchDelete)

	deploy.GET("/plans", readPlan, planCtl.List)
	deploy.POST("/plans", writePlan, planCtl.Create)
	deploy.GET("/plans/:id", readPlan, planCtl.Get)
	deploy.PATCH("/plans/:id", writePlan, planCtl.Update)
	deploy.GET("/plans/:id/simulations", readPlan, runtimeCtl.DryRun)
	deploy.POST("/plans/:id/preflight-runs", execDeploy, runtimeCtl.Preflight)
	deploy.POST("/plans/:id/preflight-runs/overrides", execDeploy, runtimeCtl.SetPreflightIgnore)
	deploy.GET("/plans/:id/ansible-config", readPlan, runtimeCtl.AnsibleConfig)
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

	deploy.GET("/executions/:taskId", readPlan, taskCtl.Get)
	deploy.GET("/executions/:taskId/logs", readPlan, taskCtl.Logs)
	deploy.GET("/executions/:taskId/events", readPlan, taskCtl.LogsSSE)

	if configCtl != nil {
		deploy.GET("/configurations", readPlan, configCtl.ListConfigs)
		deploy.GET("/configurations/os-types", readPlan, configCtl.ListSupportedOSTypes)
		deploy.GET("/configurations/:id", readPlan, configCtl.GetConfig)
		deploy.PATCH("/configurations/:id", writePlan, configCtl.UpdateConfig)
		deploy.GET("/configurations/:id/versions", readPlan, configCtl.GetConfigVersions)

		deploy.GET("/repositories", readPlan, configCtl.ListRepositories)
		deploy.POST("/repositories", writePlan, configCtl.CreateRepository)
		deploy.GET("/repositories/:id", readPlan, configCtl.GetRepository)
		deploy.PATCH("/repositories/:id", writePlan, configCtl.UpdateRepository)
		deploy.DELETE("/repositories/:id", deletePlan, configCtl.DeleteRepository)

		deploy.GET("/ansible/playbook", readPlan, configCtl.GetAnsiblePlaybook)
		deploy.GET("/ansible/inventory-template", readPlan, configCtl.GetAnsibleInventoryTemplate)
		deploy.GET("/ansible/env-check", readPlan, configCtl.CheckAnsibleEnv)
		deploy.GET("/ansible/tree", readPlan, configCtl.GetAnsibleTree)
	}
}
