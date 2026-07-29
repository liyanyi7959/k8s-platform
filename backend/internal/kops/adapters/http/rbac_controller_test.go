package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRBACControllerBuild(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller := NewRBACController()
	router.POST("/rbac", controller.Build)
	request := httptest.NewRequest(http.MethodPost, "/rbac", strings.NewReader(`{"service_account":"agent","sa_namespace":"ops","target_namespaces":["dev"],"namespace_rows":[{"resources":["pods"],"verbs":["get"]}]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "agent-ns-ops") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
