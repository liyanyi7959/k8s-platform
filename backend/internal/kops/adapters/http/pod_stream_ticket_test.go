package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStreamTicketIDUsesPathTicketAfterQueryIsRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/streams/v2/ticket-from-path?kind=pod-exec", nil)
	context.Params = gin.Params{{Key: "ticket_id", Value: "ticket-from-path"}}

	// The router reads kind before delegating. Gin caches that query map, so a
	// later RawQuery rewrite cannot be used to pass a session id downstream.
	if got := context.Query("kind"); got != "pod-exec" {
		t.Fatalf("kind = %q, want pod-exec", got)
	}
	context.Request.URL.RawQuery = "session_id=legacy-ticket"

	if got := streamTicketID(context); got != "ticket-from-path" {
		t.Fatalf("stream ticket = %q, want ticket-from-path", got)
	}
}

func TestStreamTicketIDAcceptsLegacyQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/ws?session_id=legacy-ticket", nil)

	if got := streamTicketID(context); got != "legacy-ticket" {
		t.Fatalf("stream ticket = %q, want legacy-ticket", got)
	}
}
