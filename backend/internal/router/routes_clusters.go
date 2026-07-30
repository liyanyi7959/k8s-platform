package router

import (
	"github.com/gin-gonic/gin"

	fleethttp "k8s-platform-backend/internal/fleet/adapters/http"
	"k8s-platform-backend/internal/middleware"
)

func registerClusterRoutes(authed *gin.RouterGroup, d Deps, ctl *fleethttp.ClusterController) {
	if ctl == nil {
		return
	}
	clusters := authed.Group("/clusters")
	clusters.Use(middleware.RequirePermV2("cluster:read"))
	clusters.GET("", ctl.List)
	clusters.POST("", middleware.RequirePermV2("cluster:create"), ctl.Import)
	clusters.GET("/:id", ctl.Get)
	clusters.POST("/:id/health-checks", ctl.CheckHealth)
	clusters.PATCH("/:id", middleware.RequirePermV2("cluster:create"), ctl.Patch)
	clusters.DELETE("/:id", middleware.RequirePermV2("cluster:create"), ctl.Delete)

	authed.POST("/cluster-imports", middleware.RequirePermV2("cluster:create"), ctl.Import)
}
