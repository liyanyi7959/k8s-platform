// router 负责注册 HTTP 路由，并把 controller 绑定到 gin.Engine。
//
// 路由组织约定：
// - 存量 API 使用 `/api/v1`；逐领域迁移后的 API 使用 `/api/v2`
// - v1 保留历史响应信封，v2 使用 HTTP 语义与 Problem Details
// - 认证/鉴权通过中间件完成：
//   - RequestID：为每个请求注入 request id，便于排障追踪
//   - AuthRequiredWithRBAC：解析 JWT，并加载/校验权限点
//   - RequirePerm：对具体路由进行权限点校验
package router

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

// New 组装路由。接收 Deps 依赖容器，消除长参数列表。
func New(d Deps) (*gin.Engine, error) {
	r := gin.New()

	// ── 全局中间件 ──
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLogger())
	r.Use(middleware.RecoveryWithZap())
	r.Use(middleware.CORS())
	r.Use(middleware.APIDeprecation())

	modules := buildApplicationModules(d)

	// ── 健康检查 ──
	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		if d.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "database is not configured"})
			return
		}
		sqlDB, err := d.DB.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "database is unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// ── 路由注册 ──
	registerRoutes(r, d, modules)

	return r, nil
}

//nolint:funlen // 路由注册表天然是长函数，按业务域分段组织
func registerRoutes(
	r *gin.Engine,
	d Deps,
	modules applicationModules,
) {
	api := r.Group("/api/v1")

	// ── 公开接口（无需认证） ──
	api.POST("/auth/login", d.AuthCtl.Login)
	api.POST("/auth/logout", d.AuthCtl.Logout)
	api.GET("/auth/me", d.AuthCtl.Me)
	api.GET("/auth/captcha", d.AuthCtl.GetCaptcha)
	api.POST("/auth/password-reset/request", d.AuthCtl.RequestPasswordReset)
	api.POST("/auth/password-reset/confirm", d.AuthCtl.ConfirmPasswordReset)
	// Alertmanager 使用单独的共享令牌接入，避免将告警入口暴露为匿名写接口。
	if token := strings.TrimSpace(os.Getenv("AIOPS_ALERTMANAGER_WEBHOOK_TOKEN")); token != "" && modules.incident.legacy != nil {
		api.POST("/monitor/webhooks/alertmanager", func(c *gin.Context) {
			if c.GetHeader("X-AIOPS-Webhook-Token") != token {
				resp.Fail(c, 4010, "Webhook 认证失败")
				c.Abort()
				return
			}
			modules.incident.legacy.IngestAlertmanager(c)
		})
	}

	// ── 需认证接口 ──
	authed := api.Group("")
	authed.Use(middleware.AuthRequiredWithRBAC(d.JWTMgr, d.AuthorizationReader))

	// ── 审计中间件（仅对写操作生效） ──
	if modules.audit.service != nil {
		authed.Use(middleware.AuditLogger(modules.audit.service))
	}

	authed.POST("/auth/change-password", d.AuthCtl.ChangePassword)

	registerClusterRoutes(authed, d, modules.fleet.clusters)
	registerDashboardRoutes(authed, d, modules.fleet.dashboard)
	registerPermissionAuditRoutes(authed, modules.kops.permissionAudit, modules.kops.rbac)
	registerK8sRoutes(authed, d, modules.kops.resources, modules.kops.manifests, modules.kops.namespaces, modules.kops.metrics)
	registerWebSocketRoutes(authed, modules.kops.resources)
	registerAuditRoutes(authed, modules.audit.controller)
	registerUserRoutes(authed, modules.iam.users)
	registerSystemRoutes(authed, modules.platform.settings)
	registerAIRoutes(authed, modules.ai.controller, modules.ai.management)
	registerDeployRoutes(authed, modules.provisioning.deploy, modules.provisioning.servers, modules.provisioning.credentials, modules.provisioning.plans, modules.provisioning.runtime, modules.provisioning.tasks, modules.provisioning.config)
	registerProjectRoutes(authed, modules.workspace.projects)
	registerAppTemplateRoutes(authed, modules.provisioning.appTemplate)
	registerMonitorIncidentRoutes(authed, modules.incident.legacy)
	registerAutomationTaskRoutes(authed, modules.provisioning.automation)
	registerIncidentV2Routes(r, d, modules.audit.service, modules.incident.v2)
}
