package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/auth"
	"k8s-platform-backend/internal/controller"
)

type permissionResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestRegisterK8sRoutes_RejectsInsufficientPerms(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		method   string
		path     string
		perms    []string
		wantCode int
	}{
		{
			name:     "secret reveal does not reuse read perm",
			method:   http.MethodGet,
			path:     "/api/v1/clusters/1/secrets/default/demo/reveal",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		{
			name:     "pod exec session does not reuse write perm",
			method:   http.MethodPost,
			path:     "/api/v1/clusters/1/pods/default/demo/exec",
			perms:    []string{"k8s:write"},
			wantCode: 1003,
		},
		{
			name:     "node delete still requires write perm",
			method:   http.MethodDelete,
			path:     "/api/v1/clusters/1/nodes/node-a",
			perms:    []string{"k8s:read"},
			wantCode: 1003,
		},
		{
			name:     "rbac delete does not accept generic k8s write",
			method:   http.MethodDelete,
			path:     "/api/v1/clusters/1/roles/default/demo",
			perms:    []string{"k8s:write"},
			wantCode: 1003,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performPermissionRequest(t, tt.method, tt.path, tt.perms, func(group *gin.RouterGroup) {
				registerK8sRoutes(group, Deps{}, &controller.K8sController{})
			})
			assertPermissionCode(t, resp, tt.wantCode)
		})
	}
}

func TestRegisterWebSocketRoutes_PodExecRequiresExecPerm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resp := performPermissionRequest(t, http.MethodGet, "/api/v1/ws/pod-exec?session_id=sid", []string{"k8s:read"}, func(group *gin.RouterGroup) {
		registerWebSocketRoutes(group, &controller.K8sController{})
	})

	assertPermissionCode(t, resp, 1003)
}

func performPermissionRequest(t *testing.T, method string, path string, perms []string, register func(group *gin.RouterGroup)) *httptest.ResponseRecorder {
	t.Helper()

	r := gin.New()
	authed := r.Group("/api/v1")
	authed.Use(func(c *gin.Context) {
		c.Set("auth_claims", &auth.Claims{UserID: 1, Username: "tester", Perms: perms})
		c.Next()
	})
	register(authed)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func assertPermissionCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode int) {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected http status: got %d want %d", recorder.Code, http.StatusOK)
	}

	var body permissionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != wantCode {
		t.Fatalf("unexpected business code: got %d want %d, body=%s", body.Code, wantCode, recorder.Body.String())
	}
}
