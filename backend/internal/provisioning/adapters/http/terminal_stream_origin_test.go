package http

import (
	"net/http/httptest"
	"testing"
)

func TestServerTerminalSameOriginRequiresMatchingOrigin(t *testing.T) {
	request := httptest.NewRequest("GET", "http://console.example.test/streams/v2/server-terminal/ticket", nil)
	if sameOriginServerTerminalRequest(request) {
		t.Fatal("terminal upgrade without Origin must be rejected")
	}
	request.Header.Set("Origin", "https://console.example.test")
	if !sameOriginServerTerminalRequest(request) {
		t.Fatal("same-host Origin must be accepted")
	}
	request.Header.Set("Origin", "https://attacker.example.test")
	if sameOriginServerTerminalRequest(request) {
		t.Fatal("cross-origin terminal upgrade must be rejected")
	}
}
