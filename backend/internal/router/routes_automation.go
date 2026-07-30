package router

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
)

func registerAutomationTaskRoutes(authed *gin.RouterGroup, ctl *provisionhttp.AutomationTaskController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePermV2("automation:read")
	execute := middleware.RequirePermV2("automation:execute")
	tasks := authed.Group("/automation/tasks")
	tasks.GET("", read, ctl.List)
	tasks.GET("/:id", read, ctl.Get)
	tasks.GET("/:id/logs", read, ctl.Logs)
	tasks.POST("/:id/cancellation-requests", execute, ctl.Cancel)
}
