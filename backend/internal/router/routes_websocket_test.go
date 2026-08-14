package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	cicdhttp "k8s-platform-backend/internal/cicd/adapters/http"
	kopshttp "k8s-platform-backend/internal/kops/adapters/http"
	provisionhttp "k8s-platform-backend/internal/provisioning/adapters/http"
)

func TestRegisterWebSocketRoutes_UsesTypedCapabilityTicketPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerWebSocketRoutes(router, Deps{}, &kopshttp.PodLogStreamController{}, &kopshttp.PodExecStreamController{}, &provisionhttp.ServerAccessController{}, &cicdhttp.CICDLogStreamController{})

	paths := make(map[string]bool)
	for _, route := range router.Routes() {
		paths[route.Path] = true
	}
	for _, want := range []string{
		"/streams/v2/pod-logs/:ticket_id",
		"/streams/v2/pod-exec/:ticket_id",
		"/streams/v2/server-terminal/:ticket_id",
		"/streams/v2/cicd-run-logs/:run_id",
	} {
		if !paths[want] {
			t.Fatalf("WebSocket route %q is not registered", want)
		}
	}
	if paths["/streams/v2/:ticket_id"] {
		t.Fatal("generic stream route must not let a client select the stream kind")
	}
}
