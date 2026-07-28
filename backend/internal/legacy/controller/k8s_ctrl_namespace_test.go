package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetNamespaceResourcesSummary_InvalidClusterID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/clusters/invalid/namespaces/default/resources-summary", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}, {Key: "ns", Value: "default"}}

	ctl := &K8sController{}
	ctl.GetNamespaceResourcesSummary(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}

func TestListEvents_InvalidClusterID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/clusters/invalid/events?involved_object_kind=Pod", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	ctl := &K8sController{}
	ctl.ListEvents(c)

	body := jsonBody(w)
	if body.Code != 4000 {
		t.Fatalf("code = %d, want 4000", body.Code)
	}
}
