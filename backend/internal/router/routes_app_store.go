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
	read := middleware.RequirePermV2("appstore:read")
	write := middleware.RequirePermV2("appstore:write")
	authed.GET("/application-templates", read, ctl.ListAppTemplates)
	authed.GET("/application-templates/:id", read, ctl.GetAppTemplate)
	authed.POST("/application-templates", write, ctl.CreateAppTemplate)
	authed.PATCH("/application-templates/:id", write, ctl.UpdateAppTemplate)
	authed.DELETE("/application-templates/:id", write, ctl.DeleteAppTemplate)
}
