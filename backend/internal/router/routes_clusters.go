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
	clusters.Use(middleware.RequirePerm("cluster:read"))
	clusters.GET("", ctl.List)
	clusters.GET("/:id", ctl.Get)
	clusters.POST("/:id/check-health", ctl.CheckHealth)
	clusters.POST("/:id/health-checks", ctl.CheckHealth)
	clusters.PATCH("/:id", middleware.RequirePerm("cluster:create"), ctl.Patch)
	clusters.DELETE("/:id", middleware.RequirePerm("cluster:create"), ctl.Delete)

	authed.POST("/clusters/import", middleware.RequirePerm("cluster:create"), ctl.Import)
}
