package controller

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
	req := httptest.NewRequest("POST", "/clusters/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	return ctx
}

func TestBindImportClusterReqMultipartFile(t *testing.T) {
	ctx := multipartImportContext(t, "config.yaml", "apiVersion: v1")
	var req importClusterReq
	if err := bindImportClusterReq(ctx, &req); err != nil {
		t.Fatal(err)
	}
	if req.Name != "demo" || req.Kubeconfig != "apiVersion: v1" {
		t.Fatalf("unexpected request: %#v", req)
	}
}

func TestBindImportClusterReqRejectsUnsupportedFile(t *testing.T) {
	ctx := multipartImportContext(t, "config.exe", "invalid")
	var req importClusterReq
	if err := bindImportClusterReq(ctx, &req); err == nil || !strings.Contains(err.Error(), "文件类型") {
		t.Fatalf("expected file type error, got %v", err)
	}
}
