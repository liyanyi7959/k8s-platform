package router

import (
	"strings"

	"github.com/gin-gonic/gin"

	aihttp "k8s-platform-backend/internal/ai/adapters/http"
	audithttp "k8s-platform-backend/internal/audit/adapters/http"
	iamhttp "k8s-platform-backend/internal/iam/adapters/http"
	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	"k8s-platform-backend/internal/middleware"
	platformhttp "k8s-platform-backend/internal/platform/adapters/http"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
)

func registerWebSocketRoutes(r *gin.Engine, d Deps, podLog *kopshttp.PodLogStreamController, podExec *kopshttp.PodExecStreamController, terminal *provisionhttp.ServerAccessController) {
	if podLog == nil && podExec == nil && terminal == nil {
		return
	}
	streams := r.Group("/streams/v2")
	streams.Use(middleware.V2Contract(), middleware.AuthRequiredV2(d.JWTMgr, d.AuthorizationReader))
	streams.GET("/:ticket_id", func(c *gin.Context) {
		ticketID := strings.TrimSpace(c.Param("ticket_id"))
		kind := strings.TrimSpace(c.Query("kind"))
		if ticketID == "" {
			c.Status(400)
			return
		}
		switch kind {
		case "pod-log":
			if podLog != nil {
				middleware.RequirePermV2("k8s:read")(c)
				if !c.IsAborted() {
					podLog.Stream(c)
				}
				return
			}
		case "pod-exec":
			if podExec != nil {
				middleware.RequirePermV2("k8s:exec")(c)
				if !c.IsAborted() {
					podExec.Stream(c)
				}
				return
			}
		case "server-terminal":
			if terminal != nil {
				middleware.RequirePermV2("deploy:server_write")(c)
				if !c.IsAborted() {
					terminal.TerminalWS(c)
				}
				return
			}
		}
		c.Status(404)
	})
}

func registerAuditRoutes(authed *gin.RouterGroup, ctl *audithttp.Controller) {
	if ctl == nil {
		return
	}
	authed.GET("/audit-logs", middleware.RequirePermV2("user:read"), ctl.List)
}

func registerUserRoutes(authed *gin.RouterGroup, ctl *iamhttp.Controller) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePermV2("user:read")
	write := middleware.RequirePermV2("user:write")

	// 用户管理
	authed.GET("/users", read, ctl.ListUsers)
	authed.POST("/users", write, ctl.CreateUser)
	authed.PATCH("/users/:id", write, ctl.UpdateUser)
	authed.DELETE("/users/:id", write, ctl.DeleteUser)
	authed.POST("/users/:id/password-reset-requests", write, ctl.ResetPassword)

	// 角色管理
	authed.GET("/roles", read, ctl.ListRoles)
	authed.POST("/roles", write, ctl.CreateRole)
	authed.PATCH("/roles/:id", write, ctl.UpdateRole)
	authed.DELETE("/roles/:id", write, ctl.DeleteRole)

	// 权限点列表
	authed.GET("/permissions", read, ctl.ListPermissions)
}

func registerSystemRoutes(authed *gin.RouterGroup, ctl *platformhttp.SettingsController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePermV2("user:read")
	write := middleware.RequirePermV2("user:write")

	authed.GET("/platform/settings", read, ctl.Get)
	authed.PATCH("/platform/settings", write, ctl.Update)
}

func registerAIRoutes(authed *gin.RouterGroup, runtime *aihttp.RuntimeController, management *aihttp.ManagementController) {
	if runtime == nil || management == nil {
		return
	}

	aiReadPerm := middleware.RequireAnyPermV2("ai:chat", "ai:diagnose", "ai:image", "ai:model_admin")
	aiWritePerm := middleware.RequirePermV2("ai:model_admin")
	clusterReadPerm := middleware.RequirePermV2("cluster:read")

	ai := authed.Group("/ai")
	ai.GET("/providers", aiWritePerm, management.ListProviders)
	ai.POST("/providers", aiWritePerm, management.CreateProvider)
	ai.PATCH("/providers/:id", aiWritePerm, management.PatchProvider)
	ai.DELETE("/providers/:id", aiWritePerm, management.DeleteProvider)
	ai.GET("/models", aiReadPerm, management.ListModels)
	ai.GET("/tools", aiReadPerm, runtime.ListTools)
	ai.POST("/models", aiWritePerm, management.CreateModel)
	ai.PATCH("/models/:id", aiWritePerm, management.PatchModel)
	ai.DELETE("/models/:id", aiWritePerm, management.DeleteModel)
	ai.GET("/route-settings", aiWritePerm, management.GetRouteSettings)
	ai.PATCH("/route-settings", aiWritePerm, management.UpdateRouteSettings)
	ai.GET("/conversations", aiReadPerm, management.ListConversations)
	ai.GET("/conversations/:id", aiReadPerm, runtime.GetConversation)
	ai.DELETE("/conversations/:id", aiReadPerm, management.DeleteConversation)
	ai.GET("/files/:id/content", aiReadPerm, runtime.DownloadAttachmentContent)

	ai.POST("/conversations", clusterReadPerm, middleware.RequireAnyPermV2("ai:chat", "ai:diagnose"), management.CreateConversation)
	ai.POST("/conversations/:id/messages", clusterReadPerm, middleware.RequireAnyPermV2("ai:chat", "ai:diagnose"), runtime.SendChatOrStream)
	ai.GET("/conversations/:id/events", clusterReadPerm, middleware.RequireAnyPermV2("ai:chat", "ai:diagnose"), runtime.SendChatStream)
	authed.POST("/change-proposals", clusterReadPerm, middleware.RequirePermV2("ai:change_propose"), runtime.CreateActionProposal)
	authed.POST("/change-proposals/:id/approvals", clusterReadPerm, middleware.RequirePermV2("ai:change_confirm"), middleware.RequirePermV2("k8s:write"), runtime.ConfirmActionProposal)
}
