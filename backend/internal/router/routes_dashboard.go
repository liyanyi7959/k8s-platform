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
	clusters := authed.Group("/clusters")
	clusters.Use(middleware.RequirePermV2("cluster:read"))
	clusters.GET("/:id/overview", middleware.CacheJSON(d.CacheStore, d.CacheTTL), ctl.GetClusterOverview)
	clusters.GET("/:id/certificate-risks", ctl.GetClusterCertificateRisks)
}
