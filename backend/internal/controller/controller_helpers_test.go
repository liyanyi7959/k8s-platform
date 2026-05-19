package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// jsonBody 从 recorder 中解析 JSON 响应
func jsonBody(w *httptest.ResponseRecorder) resp.ApiResponse[any] {
	var r resp.ApiResponse[any]
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	return r
}

// ──────────────────────────────────────────────────────────
//  WriteServiceErr — 通用映射
// ──────────────────────────────────────────────────────────

func TestWriteServiceErr_InvalidParams(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, service.ErrInvalidParams)

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200", w.Code)
	}
	body := jsonBody(w)
	if body.Code != 4000 {
		t.Errorf("code = %d, want 4000", body.Code)
	}
	if body.Message != "参数错误" {
		t.Errorf("message = %q, want %q", body.Message, "参数错误")
	}
}

func TestWriteServiceErr_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, service.ErrNotFound)

	body := jsonBody(w)
	if body.Code != 4040 {
		t.Errorf("code = %d, want 4040", body.Code)
	}
}

func TestWriteServiceErr_Conflict(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, service.ErrConflict)

	body := jsonBody(w)
	if body.Code != 4090 {
		t.Errorf("code = %d, want 4090", body.Code)
	}
}

func TestWriteServiceErr_UnknownError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, errors.New("something unexpected"))

	body := jsonBody(w)
	if body.Code != 5000 {
		t.Errorf("code = %d, want 5000", body.Code)
	}
	if body.Message != "内部错误" {
		t.Errorf("message = %q, want %q", body.Message, "内部错误")
	}
}

// ──────────────────────────────────────────────────────────
//  WriteServiceErr — 带 UserMessage 的 ServiceError
// ──────────────────────────────────────────────────────────

func TestWriteServiceErr_WithUserMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := service.ErrWithMessage(service.ErrNotFound, "集群不存在")
	WriteServiceErr(c, err)

	body := jsonBody(w)
	if body.Code != 4040 {
		t.Errorf("code = %d, want 4040", body.Code)
	}
	if body.Message != "集群不存在" {
		t.Errorf("message = %q, want %q", body.Message, "集群不存在")
	}
}

// ──────────────────────────────────────────────────────────
//  WriteServiceErr — K8s 额外映射
// ──────────────────────────────────────────────────────────

func TestWriteServiceErr_K8sNetwork(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, service.ErrK8sNetwork, K8sErrMappings...)

	body := jsonBody(w)
	if body.Code != 5000 {
		t.Errorf("code = %d, want 5000", body.Code)
	}
	if body.Message != "网络连接失败" {
		t.Errorf("message = %q, want %q", body.Message, "网络连接失败")
	}
}

func TestWriteServiceErr_K8sUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, service.ErrK8sUnauthorized, K8sErrMappings...)

	body := jsonBody(w)
	if body.Code != 1002 {
		t.Errorf("code = %d, want 1002", body.Code)
	}
}

func TestWriteServiceErr_K8sForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, service.ErrK8sForbidden, K8sErrMappings...)

	body := jsonBody(w)
	if body.Code != 1003 {
		t.Errorf("code = %d, want 1003", body.Code)
	}
}

func TestWriteServiceErr_K8sWithUserMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := service.ErrWithMessage(service.ErrK8sTimeout, "连接 10.0.0.1:6443 超时")
	WriteServiceErr(c, err, K8sErrMappings...)

	body := jsonBody(w)
	if body.Code != 5000 {
		t.Errorf("code = %d, want 5000", body.Code)
	}
	if body.Message != "连接 10.0.0.1:6443 超时" {
		t.Errorf("message = %q, want %q", body.Message, "连接 10.0.0.1:6443 超时")
	}
}

// ──────────────────────────────────────────────────────────
//  WriteServiceErr — SSH 额外映射
// ──────────────────────────────────────────────────────────

func TestWriteServiceErr_SSHAuth(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, service.ErrSSHAuth, SSHErrMappings...)

	body := jsonBody(w)
	if body.Code != 5000 {
		t.Errorf("code = %d, want 5000", body.Code)
	}
	if body.Message != "SSH 凭据不正确" {
		t.Errorf("message = %q, want %q", body.Message, "SSH 凭据不正确")
	}
}

func TestWriteServiceErr_SSHTimeout(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	WriteServiceErr(c, service.ErrSSHTimeout, SSHErrMappings...)

	body := jsonBody(w)
	if body.Code != 5000 {
		t.Errorf("code = %d, want 5000", body.Code)
	}
}

// ──────────────────────────────────────────────────────────
//  parseInt / parseInt64 / parseInt64Ptr
// ──────────────────────────────────────────────────────────

func TestParseInt_Valid(t *testing.T) {
	if got := parseInt("42", 0); got != 42 {
		t.Errorf("parseInt(\"42\", 0) = %d, want 42", got)
	}
}

func TestParseInt_Empty(t *testing.T) {
	if got := parseInt("", 10); got != 10 {
		t.Errorf("parseInt(\"\", 10) = %d, want 10", got)
	}
}

func TestParseInt_Invalid(t *testing.T) {
	if got := parseInt("abc", 5); got != 5 {
		t.Errorf("parseInt(\"abc\", 5) = %d, want 5", got)
	}
}

func TestParseInt_Negative(t *testing.T) {
	if got := parseInt("-3", 0); got != -3 {
		t.Errorf("parseInt(\"-3\", 0) = %d, want -3", got)
	}
}

func TestParseInt64_Valid(t *testing.T) {
	if got := parseInt64("9999999999", 0); got != 9999999999 {
		t.Errorf("parseInt64 = %d, want 9999999999", got)
	}
}

func TestParseInt64_Empty(t *testing.T) {
	if got := parseInt64("", 100); got != 100 {
		t.Errorf("parseInt64(\"\", 100) = %d, want 100", got)
	}
}

func TestParseInt64_Invalid(t *testing.T) {
	if got := parseInt64("xyz", 7); got != 7 {
		t.Errorf("parseInt64(\"xyz\", 7) = %d, want 7", got)
	}
}

func TestParseInt64Ptr_Valid(t *testing.T) {
	ptr := parseInt64Ptr("123")
	if ptr == nil {
		t.Fatal("expected non-nil")
	}
	if *ptr != 123 {
		t.Errorf("*ptr = %d, want 123", *ptr)
	}
}

func TestParseInt64Ptr_Empty(t *testing.T) {
	ptr := parseInt64Ptr("")
	if ptr != nil {
		t.Errorf("expected nil for empty string, got %v", *ptr)
	}
}

func TestParseInt64Ptr_Invalid(t *testing.T) {
	ptr := parseInt64Ptr("not-a-number")
	if ptr != nil {
		t.Errorf("expected nil for invalid input, got %v", *ptr)
	}
}
