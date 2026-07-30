package router

import (
	"github.com/gin-gonic/gin"

	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	"k8s-platform-backend/internal/middleware"
)

func registerIncidentV2Routes(authed *gin.RouterGroup, ctl *incidenthttp.Controller) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePermV2("monitor:read")
	manage := middleware.RequirePermV2("incident:manage")
	incidents := authed.Group("/incidents")
	incidents.GET("", read, ctl.List)
	incidents.GET("/:id", read, ctl.Get)
	incidents.POST("/:id/acknowledgements", manage, ctl.Acknowledge)
	incidents.POST("/:id/diagnosis-runs", manage, ctl.Diagnose)
	incidents.POST("/:id/approval-requests", manage, ctl.RequestApproval)
	incidents.POST("/:id/execution-starts", manage, ctl.StartExecution)
	incidents.POST("/:id/verification-runs", manage, ctl.StartVerification)
	incidents.POST("/:id/resolution-attempts", manage, ctl.Resolve)
}

func registerMonitorIncidentRoutes(authed *gin.RouterGroup, ctl *incidenthttp.LegacyController) {
	if ctl == nil {
		return
	}
	read := middleware.RequirePermV2("monitor:read")
	write := middleware.RequirePermV2("monitor:write")
	manage := middleware.RequirePermV2("incident:manage")
	monitoring := authed.Group("/monitoring")
	monitoring.GET("/alert-rules", read, ctl.ListAlertRules)
	monitoring.POST("/alert-rules", write, ctl.CreateAlertRule)
	monitoring.PATCH("/alert-rules/:id", write, ctl.UpdateAlertRule)
	monitoring.DELETE("/alert-rules/:id", write, ctl.DeleteAlertRule)
	incidents := authed.Group("/incidents")
	incidents.POST("/:id/ai-conversation-links", manage, ctl.LinkAIConversation)
	incidents.POST("/:id/ai-proposal-links", manage, ctl.LinkAIProposal)
}
