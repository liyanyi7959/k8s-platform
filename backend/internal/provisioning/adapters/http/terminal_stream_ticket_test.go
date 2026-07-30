package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTerminalStreamTicketIDUsesPathTicketAfterQueryIsRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/streams/v2/server-terminal/ticket-from-path?ignored=value", nil)
	context.Params = gin.Params{{Key: "ticket_id", Value: "ticket-from-path"}}

	if got := context.Query("ignored"); got != "value" {
		t.Fatalf("ignored = %q, want value", got)
	}
	context.Request.URL.RawQuery = "session_id=legacy-ticket"

	if got := terminalStreamTicketID(context); got != "ticket-from-path" {
		t.Fatalf("stream ticket = %q, want ticket-from-path", got)
	}
}

func TestTerminalStreamTicketIDRejectsQueryTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/ws?session_id=legacy-ticket", nil)

	if got := terminalStreamTicketID(context); got != "" {
		t.Fatalf("stream ticket = %q, want empty", got)
	}
}
