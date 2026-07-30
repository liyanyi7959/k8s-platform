package router

import (
	"github.com/gin-gonic/gin"

	fleethttp "k8s-platform-backend/internal/fleet/adapters/http"
	"k8s-platform-backend/internal/middleware"
)

func registerDashboardRoutes(authed *gin.RouterGroup, d Deps, ctl *fleethttp.DashboardController) {
	if ctl == nil {
		return
	}
	dash := authed.Group("/dashboard")
	dash.Use(middleware.RequirePerm("cluster:read"))
	dash.GET("/clusters/:id/overview", middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.GetClusterOverview)
	dash.GET("/clusters/:id/certificate-risks", ctl.GetClusterCertificateRisks)
}
