package router

import (
	"github.com/gin-gonic/gin"
	"k8s-platform-backend/internal/cicd/adapters/http"
	"k8s-platform-backend/internal/middleware"
)

func registerCICDRoutes(authed *gin.RouterGroup, ctl *http.Controller) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePermV2("cicd:read")
	write := middleware.RequirePermV2("cicd:write")
	execute := middleware.RequirePermV2("cicd:execute")
	g := authed.Group("/cicd")
	g.GET("/summary", read, ctl.Summary)
	g.GET("/pipelines", read, ctl.ListPipelines)
	g.POST("/pipelines", write, ctl.CreatePipeline)
	g.GET("/pipelines/:id", read, ctl.GetPipeline)
	g.PATCH("/pipelines/:id", write, ctl.UpdatePipeline)
	g.DELETE("/pipelines/:id", write, ctl.DeletePipeline)
	g.POST("/pipelines/:id/runs", execute, ctl.CreateRun)
	g.GET("/runs", read, ctl.ListRuns)
	g.GET("/runs/:id", read, ctl.GetRun)
	g.POST("/runs/:id/cancellation-requests", execute, ctl.CancelRun)
	g.GET("/artifacts", read, ctl.ListArtifacts)
	g.GET("/artifacts/:id", read, ctl.GetArtifact)
	g.GET("/environments", read, ctl.ListEnvironments)
	g.POST("/environments", write, ctl.CreateEnvironment)
	g.GET("/environments/:id", read, ctl.GetEnvironment)
	g.PATCH("/environments/:id", write, ctl.UpdateEnvironment)
	g.DELETE("/environments/:id", write, ctl.DeleteEnvironment)
}
