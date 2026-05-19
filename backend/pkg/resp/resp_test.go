package resp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ────────── OK ──────────

func TestOK_StringData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK(c, "hello")

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200", w.Code)
	}

	var body ApiResponse[string]
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if body.Code != 0 {
		t.Errorf("code = %d, want 0", body.Code)
	}
	if body.Message != "ok" {
		t.Errorf("message = %q, want %q", body.Message, "ok")
	}
	if body.Data != "hello" {
		t.Errorf("data = %q, want %q", body.Data, "hello")
	}
}

func TestOK_StructData(t *testing.T) {
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK(c, Item{ID: 1, Name: "test"})

	var body ApiResponse[Item]
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if body.Data.ID != 1 || body.Data.Name != "test" {
		t.Errorf("data = %+v, want {ID:1 Name:test}", body.Data)
	}
}

func TestOK_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK[any](c, nil)

	var body ApiResponse[any]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != 0 {
		t.Errorf("code = %d, want 0", body.Code)
	}
	if body.Data != nil {
		t.Errorf("data should be nil, got %v", body.Data)
	}
}

func TestOK_SetsRespCode(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK(c, "x")

	v, exists := c.Get("resp_code")
	if !exists {
		t.Fatal("resp_code should be set in context")
	}
	if v.(int) != 0 {
		t.Errorf("resp_code = %v, want 0", v)
	}
}

// ────────── Fail ──────────

func TestFail_BasicError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, 5000, "服务器内部错误")

	if w.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200", w.Code)
	}

	var body ApiResponse[any]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != 5000 {
		t.Errorf("code = %d, want 5000", body.Code)
	}
	if body.Message != "服务器内部错误" {
		t.Errorf("message = %q, want %q", body.Message, "服务器内部错误")
	}
	if body.Data != nil {
		t.Errorf("data should be nil, got %v", body.Data)
	}
}

func TestFail_TranslatesInvalidParams(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, 4000, "invalid params")

	var body ApiResponse[any]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Message != "参数错误" {
		t.Errorf("message = %q, want %q", body.Message, "参数错误")
	}
}

func TestFail_TranslatesInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, 5000, "internal error")

	var body ApiResponse[any]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Message != "内部错误" {
		t.Errorf("message = %q, want %q", body.Message, "内部错误")
	}
}

func TestFail_TranslatesCaseInsensitive(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, 4000, "  Invalid Params  ")

	var body ApiResponse[any]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Message != "参数错误" {
		t.Errorf("message = %q, want %q (case-insensitive + trim)", body.Message, "参数错误")
	}
}

func TestFail_NoTranslation(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, 4090, "资源冲突")

	var body ApiResponse[any]
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Message != "资源冲突" {
		t.Errorf("message = %q, want %q", body.Message, "资源冲突")
	}
}

func TestFail_SetsRespCode(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, 4040, "not found")

	v, exists := c.Get("resp_code")
	if !exists {
		t.Fatal("resp_code should be set in context")
	}
	if v.(int) != 4040 {
		t.Errorf("resp_code = %v, want 4040", v)
	}
}
