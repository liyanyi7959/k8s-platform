package problem

import (
	"encoding/json"
	"net/http"
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
	if requestID := recorder.Header().Get(RequestIDHeader); requestID != "request-1" {
		t.Fatalf("response request id = %q", requestID)
	}
	var details Details
	if err := json.Unmarshal(recorder.Body.Bytes(), &details); err != nil {
		t.Fatal(err)
	}
	if details.RequestID != "request-1" || details.Instance != "/api/v2/incidents/1" {
		t.Fatalf("unexpected details: %+v", details)
	}
}

func TestWriteKindUsesCatalogDefinition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPatch, "/api/v2/incidents/1/resolve", nil)
	context.Request.Header.Set(RequestIDHeader, "upstream-request-2")

	WriteKind(context, KindVersionConflict, "请刷新事件详情后重试")

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d", recorder.Code)
	}
	var details Details
	if err := json.Unmarshal(recorder.Body.Bytes(), &details); err != nil {
		t.Fatal(err)
	}
	if details.Type != TypeBaseURI+string(KindVersionConflict) {
		t.Fatalf("type = %q", details.Type)
	}
	if details.Title != "事件已被其他用户更新" {
		t.Fatalf("title = %q", details.Title)
	}
	if details.RequestID != "upstream-request-2" {
		t.Fatalf("request id = %q", details.RequestID)
	}
}

func TestDefinitionForCataloguesV2ProblemTypes(t *testing.T) {
	testCases := []struct {
		kind   Kind
		status int
		title  string
	}{
		{KindInvalidRequest, http.StatusBadRequest, "请求参数错误"},
		{KindUnauthorized, http.StatusUnauthorized, "认证失败"},
		{KindForbidden, http.StatusForbidden, "权限不足"},
		{KindNotFound, http.StatusNotFound, "资源不存在"},
		{KindConflict, http.StatusConflict, "资源状态冲突"},
		{KindValidation, http.StatusUnprocessableEntity, "请求校验失败"},
		{KindClusterCredentialInvalid, http.StatusFailedDependency, "集群凭据无效或已过期"},
		{KindIncidentNotFound, http.StatusNotFound, "事件不存在"},
		{KindVersionConflict, http.StatusConflict, "事件已被其他用户更新"},
		{KindInvalidIncidentTransition, http.StatusConflict, "当前状态不允许此操作"},
		{KindDomainValidation, http.StatusUnprocessableEntity, "事件处置校验失败"},
		{KindInternal, http.StatusInternalServerError, "内部错误"},
	}

	for _, testCase := range testCases {
		t.Run(string(testCase.kind), func(t *testing.T) {
			definition, ok := DefinitionFor(testCase.kind)
			if !ok {
				t.Fatalf("definition missing for %q", testCase.kind)
			}
			if definition.Status != testCase.status || definition.Title != testCase.title {
				t.Fatalf("definition = %+v", definition)
			}
			if definition.TypeURI() != TypeBaseURI+string(testCase.kind) {
				t.Fatalf("type URI = %q", definition.TypeURI())
			}
		})
	}

	if _, ok := DefinitionFor(Kind("not-in-catalog")); ok {
		t.Fatal("unknown kind must not be catalogued")
	}
}

func TestWriteKindFallsBackToInternalProblem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v2/unknown", nil)

	WriteKind(context, Kind("not-in-catalog"), "unexpected")

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
	var details Details
	if err := json.Unmarshal(recorder.Body.Bytes(), &details); err != nil {
		t.Fatal(err)
	}
	if details.Type != TypeBaseURI+string(KindInternal) || details.Title != "内部错误" {
		t.Fatalf("details = %+v", details)
	}
}
