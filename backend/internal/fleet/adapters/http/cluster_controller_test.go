package http

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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
