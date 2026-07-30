package router

import (
	"github.com/gin-gonic/gin"

	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	"k8s-platform-backend/internal/middleware"
)

func registerPermissionAuditRoutes(authed *gin.RouterGroup, auditCtl *kopshttp.PermissionAuditController, rbac *kopshttp.RBACController) {
	if auditCtl == nil {
		return
	}
	auditPerm := middleware.RequirePerm("k8s:permission_audit")
	clusters := authed.Group("/clusters")
	clusters.POST("/:id/permission-audits", auditPerm, auditCtl.CreateManaged)
	clusters.GET("/:id/permission-audits/latest", auditPerm, auditCtl.LatestForCluster)
	clusters.GET("/:id/permission-audits/recommend-rbac", auditPerm, auditCtl.RecommendRBAC)
	if rbac != nil {
		clusters.GET("/:id/permission-audits/rbac-matrix/default", auditPerm, rbac.Default)
		clusters.POST("/:id/permission-audits/rbac-matrix/yaml", auditPerm, rbac.Build)
	}

	audits := authed.Group("/permission-audits")
	audits.GET("", auditPerm, auditCtl.List)
	audits.GET("/:id", auditPerm, auditCtl.Get)
	audits.GET("/:id/logs", auditPerm, auditCtl.Logs)
	audits.GET("/:id/compare", auditPerm, auditCtl.Compare)
	audits.GET("/:id/findings", auditPerm, auditCtl.ListFindings)
	audits.POST("/:id/cancel", auditPerm, auditCtl.Cancel)
	audits.POST("/adhoc", auditPerm, auditCtl.CreateAdhoc)
	audits.POST("/:id/cancellation-requests", auditPerm, auditCtl.Cancel)
	audits.POST("/ad-hoc-audits", auditPerm, auditCtl.CreateAdhoc)
}
