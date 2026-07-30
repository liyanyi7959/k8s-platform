package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStreamTicketIDUsesPathTicketAfterQueryIsRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/streams/v2/pod-exec/ticket-from-path?ignored=value", nil)
	context.Params = gin.Params{{Key: "ticket_id", Value: "ticket-from-path"}}

	// A query read must never change where the adapter gets a capability ticket.
	if got := context.Query("ignored"); got != "value" {
		t.Fatalf("ignored = %q, want value", got)
	}
	context.Request.URL.RawQuery = "session_id=legacy-ticket"

	if got := streamTicketID(context); got != "ticket-from-path" {
		t.Fatalf("stream ticket = %q, want ticket-from-path", got)
	}
}

func TestStreamTicketIDRejectsQueryTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/ws?session_id=legacy-ticket", nil)

	if got := streamTicketID(context); got != "" {
		t.Fatalf("stream ticket = %q, want empty", got)
	}
}
