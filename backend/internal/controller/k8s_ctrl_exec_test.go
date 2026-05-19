package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
)

func TestPodExecWS_UpgradeFailureKeepsSession(t *testing.T) {
	store := service.NewExecSessionStore(0)
	defer store.Close()
	store.Put("sid", service.ExecSession{ClusterID: 1, Namespace: "default", Pod: "demo"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/ws/pod-exec?session_id=sid", nil)

	ctl := &K8sController{execSessions: store}
	ctl.PodExecWS(c)

	if _, ok := store.Get("sid"); !ok {
		t.Fatal("session was consumed after websocket upgrade failure")
	}
}
