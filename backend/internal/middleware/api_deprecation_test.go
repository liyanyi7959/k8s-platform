package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIDeprecationMarksLegacyActionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paths := []string{
		"/api/v1/clusters/1/configmaps/edit",
		"/api/v1/clusters/1/nodes/node-a/drain",
		"/api/v1/clusters/1/manifests/apply",
		"/api/v1/clusters/1/helm/releases/default/demo/upgrade",
		"/api/v1/deploy/plans/1/preflight/ignore",
		"/api/v1/automation/tasks/1/cancel",
	}
	for _, path := range paths {
		if !isLegacyActionPath(path) {
			t.Errorf("legacy route was not recognized: %s", path)
		}
	}
}

func TestAPIDeprecationLeavesCanonicalRoutesUnmarked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paths := []string{
		"/api/v1/clusters/1/configmaps/default/demo",
		"/api/v1/clusters/1/nodes/node-a/drain-requests",
		"/api/v1/clusters/1/manifest-applications",
		"/api/v1/clusters/1/helm/releases/default/demo/upgrade-attempts",
		"/api/v1/deploy/plans/1/preflight-checks/overrides",
		"/api/v1/automation/tasks/1/cancellation-requests",
	}
	for _, path := range paths {
		if isLegacyActionPath(path) {
			t.Errorf("canonical route was marked deprecated: %s", path)
		}
	}
}
