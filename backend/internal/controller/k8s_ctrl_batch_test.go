package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTriggerCronJob_InvalidClusterID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/clusters/invalid/cronjobs/default/demo/trigger", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}, {Key: "ns", Value: "default"}, {Key: "name", Value: "demo"}}

	ctl := &K8sController{}
	ctl.TriggerCronJob(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}

func TestSuspendCronJob_InvalidClusterID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/clusters/invalid/cronjobs/default/demo/suspend", strings.NewReader(`{"suspend":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "invalid"}, {Key: "ns", Value: "default"}, {Key: "name", Value: "demo"}}

	ctl := &K8sController{}
	ctl.SuspendCronJob(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}

func TestDeleteCompletedJobs_InvalidClusterID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/clusters/invalid/jobs/completed?older_than_hours=24", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	ctl := &K8sController{}
	ctl.DeleteCompletedJobs(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}
