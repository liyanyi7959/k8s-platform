package problem

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteUsesProblemDetailsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/api/v2/incidents/1", nil)
	context.Set("request_id", "request-1")

	Write(context, 409, "https://aiops.local/problems/version-conflict", "conflict", "refresh")

	if recorder.Code != 409 {
		t.Fatalf("status = %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Fatalf("content type = %q", contentType)
	}
	var details Details
	if err := json.Unmarshal(recorder.Body.Bytes(), &details); err != nil {
		t.Fatal(err)
	}
	if details.RequestID != "request-1" || details.Instance != "/api/v2/incidents/1" {
		t.Fatalf("unexpected details: %+v", details)
	}
}
