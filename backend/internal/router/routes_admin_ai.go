package router

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/legacy/controller"
	"k8s-platform-backend/internal/middleware"
)

func registerWebSocketRoutes(authed *gin.RouterGroup, k8sCtl *controller.K8sController) {
	ws := authed.Group("/ws")
	if k8sCtl != nil {
		ws.GET("/pod-log", middleware.RequirePerm("k8s:read"), k8sCtl.PodLogWS)
		ws.GET("/pod-exec", middleware.RequirePerm("k8s:exec"), k8sCtl.PodExecWS)
	}
}

func registerAuditRoutes(authed *gin.RouterGroup, ctl *controller.AuditController) {
	if ctl == nil {
		return
	}
	authed.GET("/audit-logs", middleware.RequirePerm("user:read"), ctl.List)
}

func registerUserRoutes(authed *gin.RouterGroup, ctl *controller.UserController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePerm("user:read")
	write := middleware.RequirePerm("user:write")

	// 用户管理
	authed.GET("/users", read, ctl.ListUsers)
	authed.POST("/users", write, ctl.CreateUser)
	authed.PUT("/users/:id", write, ctl.UpdateUser)
	authed.DELETE("/users/:id", write, ctl.DeleteUser)
	authed.POST("/users/:id/reset-password", write, ctl.ResetPassword)
	authed.POST("/users/:id/password-reset-requests", write, ctl.ResetPassword)

	// 角色管理
	authed.GET("/roles", read, ctl.ListRoles)
	authed.POST("/roles", write, ctl.CreateRole)
	authed.PUT("/roles/:id", write, ctl.UpdateRole)
	authed.DELETE("/roles/:id", write, ctl.DeleteRole)

	// 权限点列表
	authed.GET("/permissions", read, ctl.ListPermissions)
}

func registerSystemRoutes(authed *gin.RouterGroup, ctl *controller.SystemSettingController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePerm("user:read")
	write := middleware.RequirePerm("user:write")

	authed.GET("/system/settings", read, ctl.Get)
	authed.PUT("/system/settings", write, ctl.Update)
}

func registerAIRoutes(authed *gin.RouterGroup, ctl *controller.AIController) {
	if ctl == nil {
		return
	}

	aiReadPerm := middleware.RequireAnyPerm("ai:chat", "ai:diagnose", "ai:image", "ai:model_admin")
	aiWritePerm := middleware.RequirePerm("ai:model_admin")
	clusterReadPerm := middleware.RequirePerm("cluster:read")

	ai := authed.Group("/ai")
	ai.GET("/providers", aiWritePerm, ctl.ListProviders)
	ai.POST("/providers", aiWritePerm, ctl.CreateProvider)
	ai.PATCH("/providers/:id", aiWritePerm, ctl.PatchProvider)
	ai.DELETE("/providers/:id", aiWritePerm, ctl.DeleteProvider)
	ai.GET("/models", aiReadPerm, ctl.ListModels)
	ai.GET("/tools", aiReadPerm, ctl.ListTools)
	ai.POST("/models", aiWritePerm, ctl.CreateModel)
	ai.PATCH("/models/:id", aiWritePerm, ctl.PatchModel)
	ai.DELETE("/models/:id", aiWritePerm, ctl.DeleteModel)
	ai.GET("/route-settings", aiWritePerm, ctl.GetRouteSettings)
	ai.PUT("/route-settings", aiWritePerm, ctl.UpdateRouteSettings)
	ai.GET("/conversations", aiReadPerm, ctl.ListConversations)
	ai.GET("/conversations/:id", aiReadPerm, ctl.GetConversation)
	ai.DELETE("/conversations/:id", aiReadPerm, ctl.DeleteConversation)
	ai.GET("/files/:id/content", aiReadPerm, ctl.DownloadAttachmentContent)

	clusters := authed.Group("/clusters")
	clusters.POST("/:id/ai/conversations", clusterReadPerm, middleware.RequireAnyPerm("ai:chat", "ai:diagnose"), ctl.CreateConversation)
	clusters.POST("/:id/ai/chat", clusterReadPerm, middleware.RequireAnyPerm("ai:chat", "ai:diagnose"), ctl.SendChat)
	clusters.POST("/:id/ai/chat/stream", clusterReadPerm, middleware.RequireAnyPerm("ai:chat", "ai:diagnose"), ctl.SendChatStream)
	clusters.POST("/:id/ai/actions/propose", clusterReadPerm, middleware.RequirePerm("ai:change_propose"), ctl.CreateActionProposal)
	clusters.POST("/:id/ai/actions/:actionId/confirm", clusterReadPerm, middleware.RequirePerm("ai:change_confirm"), middleware.RequirePerm("k8s:write"), ctl.ConfirmActionProposal)
}
