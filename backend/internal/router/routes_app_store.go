package router

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
)

func registerAppTemplateRoutes(authed *gin.RouterGroup, ctl *provisionhttp.AppTemplateController) {
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
