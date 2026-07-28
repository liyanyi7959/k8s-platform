package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetRolloutHistory_InvalidClusterID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/clusters/invalid/workloads/deployments/default/demo/rollout-history", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "invalid"},
		{Key: "ns", Value: "default"},
		{Key: "name", Value: "demo"},
	}

	ctl := &K8sController{}
	ctl.GetRolloutHistory(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}

func TestRolloutUndo_InvalidRevision(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/clusters/1/workloads/deployments/default/demo/rollout-undo", strings.NewReader(`{"kind":"Deployment","revision":-1}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "ns", Value: "default"},
		{Key: "name", Value: "demo"},
	}

	ctl := &K8sController{}
	ctl.RolloutUndo(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}

func TestUpdateImage_InvalidKind(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/clusters/1/workloads/image", strings.NewReader(`{"kind":"Job","namespace":"default","name":"demo","container":"app","image":"nginx:1.27.0"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	ctl := &K8sController{}
	ctl.UpdateImage(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}

func TestUpdateImage_MissingContainer(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/clusters/1/workloads/image", strings.NewReader(`{"kind":"Deployment","namespace":"default","name":"demo","container":"","image":"nginx:1.27.0"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	ctl := &K8sController{}
	ctl.UpdateImage(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}

func TestUpdateWorkloadPaused_InvalidKind(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/clusters/1/workloads/rollout-pause", strings.NewReader(`{"kind":"StatefulSet","namespace":"default","name":"demo","paused":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	ctl := &K8sController{}
	ctl.UpdateWorkloadPaused(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}
