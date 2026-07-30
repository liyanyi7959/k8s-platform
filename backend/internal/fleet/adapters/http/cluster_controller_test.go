package http

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/fleet/domain"
	"k8s-platform-backend/pkg/resp"
)

func multipartImportContext(t *testing.T, filename, content string) *gin.Context {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "demo")
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte(content))
	_ = writer.Close()
	request := httptest.NewRequest("POST", "/clusters/import", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = request
	return ctx
}

func TestBindImportMultipartFile(t *testing.T) {
	ctx := multipartImportContext(t, "config.yaml", "apiVersion: v1")
	var request importRequest
	if err := bindImport(ctx, &request); err != nil {
		t.Fatal(err)
	}
	if request.Name != "demo" || request.Kubeconfig != "apiVersion: v1" {
		t.Fatalf("unexpected request: %#v", request)
	}
}
func TestBindImportRejectsUnsupportedFile(t *testing.T) {
	ctx := multipartImportContext(t, "config.exe", "invalid")
	var request importRequest
	if err := bindImport(ctx, &request); err == nil || !strings.Contains(err.Error(), "文件类型") {
		t.Fatalf("expected file type error, got %v", err)
	}
}
func TestBindImportJSON(t *testing.T) {
	request := httptest.NewRequest("POST", "/clusters/import", strings.NewReader(`{"name":"devops7.2","kubeconfig":"apiVersion: v1","description":"demo"}`))
	request.Header.Set("Content-Type", "application/json")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = request
	var input importRequest
	if err := bindImport(ctx, &input); err != nil {
		t.Fatal(err)
	}
	if input.Name != "devops7.2" || input.Description != "demo" {
		t.Fatalf("unexpected request: %#v", input)
	}
}

func TestWriteClusterErrorKeepsClusterCredentialFailureSeparateFromPlatformSession(t *testing.T) {
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)

	writeClusterError(ctx, domain.ErrRuntimeUnauthorized)

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
