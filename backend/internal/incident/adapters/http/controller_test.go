package http

import (
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/incident/domain"
	"k8s-platform-backend/pkg/problem"
)

func TestWriteErrorUsesStandardProblemCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testCases := []struct {
		name   string
		err    error
		status int
		kind   problem.Kind
	}{
		{name: "not found", err: domain.ErrNotFound, status: stdhttp.StatusNotFound, kind: problem.KindIncidentNotFound},
		{name: "version conflict", err: domain.ErrVersionConflict, status: stdhttp.StatusConflict, kind: problem.KindVersionConflict},
		{name: "invalid transition", err: domain.ErrInvalidTransition, status: stdhttp.StatusConflict, kind: problem.KindInvalidIncidentTransition},
		{name: "validation", err: domain.ErrValidation, status: stdhttp.StatusUnprocessableEntity, kind: problem.KindDomainValidation},
		{name: "internal", err: errors.New("database unavailable"), status: stdhttp.StatusInternalServerError, kind: problem.KindInternal},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(stdhttp.MethodGet, "/api/v2/incidents/42", nil)

			writeError(context, testCase.err)

			if recorder.Code != testCase.status {
				t.Fatalf("status = %d, want %d", recorder.Code, testCase.status)
			}
			var details problem.Details
			if err := json.Unmarshal(recorder.Body.Bytes(), &details); err != nil {
				t.Fatalf("decode problem details: %v", err)
			}
			if details.Type != problem.TypeBaseURI+string(testCase.kind) {
				t.Fatalf("type = %q, want %q", details.Type, problem.TypeBaseURI+string(testCase.kind))
			}
		})
	}
}
