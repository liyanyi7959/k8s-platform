package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/auth"
	iamapp "k8s-platform-backend/internal/iam/application"
	"k8s-platform-backend/pkg/resp"
)

type authControllerMemoryCache struct {
	items map[string][]byte
}

func authJSONBody(w *httptest.ResponseRecorder) resp.ApiResponse[any] {
	var body resp.ApiResponse[any]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body
}

func newAuthControllerMemoryCache() *authControllerMemoryCache {
	return &authControllerMemoryCache{items: map[string][]byte{}}
}

func (m *authControllerMemoryCache) Enabled() bool {
	return true
}

func (m *authControllerMemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	value, ok := m.items[key]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), value...), true, nil
}

func (m *authControllerMemoryCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.items[key] = append([]byte(nil), value...)
	return nil
}

func (m *authControllerMemoryCache) Del(_ context.Context, key string) error {
	delete(m.items, key)
	return nil
}

func (m *authControllerMemoryCache) Close() error {
	return nil
}

func TestAuthControllerLogin_RequiresCaptchaWhenEnabled(t *testing.T) {
	controller := NewAuthController(
		auth.NewManager("test-secret"),
		iamapp.NewAuthService(nil, nil, nil, time.Hour),
		nil,
		iamapp.NewCaptchaService(newAuthControllerMemoryCache()),
		nil,
		nil,
		time.Hour,
	)

	body, err := json.Marshal(map[string]any{
		"username": "admin",
		"password": "admin@123",
	})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	controller.Login(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200", recorder.Code)
	}

	response := authJSONBody(recorder)
	if response.Code != loginCodeCaptchaInvalid {
		t.Fatalf("code = %d, want %d", response.Code, loginCodeCaptchaInvalid)
	}

	data, ok := response.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map response data, got %T", response.Data)
	}
	if data["reason"] != "captcha_invalid" {
		t.Fatalf("reason = %v, want captcha_invalid", data["reason"])
	}
}
