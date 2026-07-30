package router

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/audit/ports"
	incidenthttp "k8s-platform-backend/internal/incident/adapters/http"
	"k8s-platform-backend/internal/middleware"
)

func registerIncidentV2Routes(r *gin.Engine, d Deps, auditRecorder ports.Recorder, ctl *incidenthttp.Controller) {
	if ctl == nil {
		return
	}
	v2 := r.Group("/api/v2")
	v2.Use(middleware.AuthRequiredV2(d.JWTMgr, d.AuthorizationReader))
	if auditRecorder != nil {
		v2.Use(middleware.AuditLogger(auditRecorder))
	}
	read := middleware.RequirePermV2("monitor:read")
	manage := middleware.RequirePermV2("incident:manage")
	incidents := v2.Group("/incidents")
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
	read := middleware.RequirePerm("monitor:read")
	write := middleware.RequirePerm("monitor:write")
	manage := middleware.RequirePerm("incident:manage")
	monitor := authed.Group("/monitor")
	monitor.GET("/alerts", read, ctl.ListAlertRules)
	monitor.POST("/alerts", write, ctl.CreateAlertRule)
	monitor.PUT("/alerts/:id", write, ctl.UpdateAlertRule)
	monitor.PUT("/alerts/:id/toggle", write, ctl.ToggleAlertRule)
	monitor.PATCH("/alerts/:id", write, ctl.ToggleAlertRule)
	monitor.DELETE("/alerts/:id", write, ctl.DeleteAlertRule)
	monitor.GET("/incidents", read, ctl.ListIncidents)
	monitor.GET("/incidents/:id", read, ctl.GetIncident)
	monitor.POST("/incidents/:id/transition", manage, ctl.TransitionIncident)
	monitor.POST("/incidents/:id/ai-conversation", manage, ctl.LinkAIConversation)
	monitor.POST("/incidents/:id/ai-proposal", manage, ctl.LinkAIProposal)
}
