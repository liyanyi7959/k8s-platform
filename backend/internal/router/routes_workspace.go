package router

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	workspacehttp "k8s-platform-backend/internal/workspace/adapters/http"
)

func registerProjectRoutes(authed *gin.RouterGroup, ctl *workspacehttp.Controller) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePerm("project:read")
	write := middleware.RequirePerm("project:write")
	authed.GET("/projects", read, ctl.ListProjects)
	authed.GET("/projects/:id", read, ctl.GetProject)
	authed.POST("/projects", write, ctl.CreateProject)
	authed.PUT("/projects/:id", write, ctl.UpdateProject)
	authed.DELETE("/projects/:id", write, ctl.DeleteProject)
	authed.GET("/projects/:id/resources", read, ctl.GetProjectResources)
	authed.PUT("/projects/:id/namespaces", write, ctl.AssignNamespaces)
}
