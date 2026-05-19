package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
)

func TestCreatePodLogSession_InvalidTailLines(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/clusters/1/pods/default/demo/logs/session", strings.NewReader(`{"tail_lines":-1}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}, {Key: "ns", Value: "default"}, {Key: "pod", Value: "demo"}}

	ctl := &K8sController{logSessions: service.NewPodLogSessionStore(0)}
	ctl.CreatePodLogSession(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}

func TestPodLogWS_MissingSessionID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/ws/pod-log", nil)

	ctl := &K8sController{logSessions: service.NewPodLogSessionStore(0)}
	ctl.PodLogWS(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}
