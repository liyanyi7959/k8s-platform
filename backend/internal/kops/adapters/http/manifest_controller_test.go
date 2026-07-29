package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

type manifestControllerRuntime struct{ input kopsapp.ManifestApplyInput }

func (runtime *manifestControllerRuntime) Execute(_ context.Context, input kopsapp.ManifestApplyInput) (*kopsapp.ManifestApplyResult, error) {
	runtime.input = input
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
