package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type manifestControllerRuntime struct {
	input kopsapp.ManifestApplyInput
	err   error
}

func (runtime *manifestControllerRuntime) Execute(_ context.Context, input kopsapp.ManifestApplyInput) (*kopsapp.ManifestApplyResult, error) {
	runtime.input = input
	if runtime.err != nil {
		return nil, runtime.err
	}
	return &kopsapp.ManifestApplyResult{RecordID: 5, Status: "success", DryRun: input.DryRun}, nil
}

func (runtime *manifestControllerRuntime) List(context.Context, kopsapp.ManifestRecordQuery) (*kopsapp.ManifestRecordPage, error) {
	return &kopsapp.ManifestRecordPage{}, nil
}

func (runtime *manifestControllerRuntime) Get(context.Context, uint64, uint64) (*kopsapp.ManifestRecordDetail, error) {
	return &kopsapp.ManifestRecordDetail{}, nil
}

func TestManifestControllerApplyUsesKopsService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime := &manifestControllerRuntime{}
	router := gin.New()
	router.POST("/clusters/:id/manifests/apply", NewManifestController(kopsapp.NewManifestService(runtime)).Apply)
	request := httptest.NewRequest(http.MethodPost, "/clusters/4/manifests/apply", strings.NewReader(`{"yaml":"  apiVersion: v1  ","dry_run":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"record_id":5`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if runtime.input.ClusterID != 4 || runtime.input.YAML != "apiVersion: v1" || !runtime.input.DryRun {
		t.Fatalf("runtime input = %#v", runtime.input)
	}
}

func TestManifestControllerRejectsInvalidPathAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/manifests/apply", NewManifestController(kopsapp.NewManifestService(&manifestControllerRuntime{})).Apply)
	for _, path := range []string{"/clusters/invalid/manifests/apply", "/clusters/4/manifests/apply"} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"code":4000`) {
			t.Fatalf("%s response = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestManifestControllerKeepsClusterCredentialFailureSeparateFromPlatformSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime := &manifestControllerRuntime{err: kopsapp.ErrRuntimeUnauthorized}
	router := gin.New()
	router.POST("/clusters/:id/manifests/apply", NewManifestController(kopsapp.NewManifestService(runtime)).Apply)
	request := httptest.NewRequest(http.MethodPost, "/clusters/4/manifests/apply", strings.NewReader(`{"yaml":"apiVersion: v1"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	var body resp.ApiResponse[any]
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != resp.CodeClusterCredentialInvalid {
		t.Fatalf("code = %d, want cluster credential error %d", body.Code, resp.CodeClusterCredentialInvalid)
	}
	if body.Code == resp.CodePlatformSessionExpired {
		t.Fatal("cluster credential failure must not use the platform session-expired code")
	}
}
