package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func APIDeprecation() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/") && isLegacyActionPath(path) {
			c.Header("Deprecation", "true")
		}
		c.Next()
	}
}

func isLegacyActionPath(path string) bool {
	if strings.HasSuffix(path, "/edit") ||
		strings.HasSuffix(path, "/toggle") ||
		strings.HasSuffix(path, "/check-health") ||
		strings.HasSuffix(path, "/test-ssh") ||
		strings.HasSuffix(path, "/detail") ||
		strings.HasSuffix(path, "/transition") ||
		strings.HasSuffix(path, "/terminal-session") ||
		strings.HasSuffix(path, "/dry-run") ||
		strings.HasSuffix(path, "/preflight") ||
		strings.HasSuffix(path, "/preflight/ignore") ||
		strings.HasSuffix(path, "/addons/install") ||
		strings.HasSuffix(path, "/addons/retry") ||
		strings.HasSuffix(path, "/adhoc") ||
		strings.HasSuffix(path, "/cordon") ||
		strings.HasSuffix(path, "/uncordon") ||
		strings.HasSuffix(path, "/drain") ||
		strings.HasSuffix(path, "/metrics/detect") ||
		strings.HasSuffix(path, "/metrics/switch") ||
		strings.HasSuffix(path, "/metrics/health-check") ||
		strings.HasSuffix(path, "/logs/session") ||
		strings.HasSuffix(path, "/exec") ||
		strings.HasSuffix(path, "/manifests/apply") ||
		strings.HasSuffix(path, "/rollout-undo") ||
		strings.HasSuffix(path, "/workloads/image") ||
		strings.HasSuffix(path, "/workloads/rollout-pause") ||
		strings.HasSuffix(path, "/trigger") ||
		strings.HasSuffix(path, "/suspend") ||
		strings.HasSuffix(path, "/helm/install") ||
		strings.HasSuffix(path, "/upgrade") ||
		strings.HasSuffix(path, "/rollback") ||
		strings.HasSuffix(path, "/reset-password") ||
		strings.HasSuffix(path, "/reveal") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/deploy/") {
		return strings.HasSuffix(path, "/execute") ||
			strings.HasSuffix(path, "/cancel") ||
			strings.HasSuffix(path, "/retry") ||
			strings.HasSuffix(path, "/batch-delete")
	}
	return strings.HasSuffix(path, "/cancel") ||
		strings.HasSuffix(path, "/workloads/scale") ||
		strings.HasSuffix(path, "/workloads/restart")
}
