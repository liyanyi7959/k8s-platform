package http

import (
	"net/http/httptest"
	"testing"
)

func TestPodStreamSameOriginRequiresMatchingOrigin(t *testing.T) {
	request := httptest.NewRequest("GET", "http://console.example.test/streams/v2/pod-exec/ticket", nil)
	if podStreamSameOrigin(request) {
		t.Fatal("stream upgrade without Origin must be rejected")
	}
	request.Header.Set("Origin", "https://console.example.test")
	if !podStreamSameOrigin(request) {
		t.Fatal("same-host Origin must be accepted")
	}
	request.Header.Set("Origin", "https://attacker.example.test")
	if podStreamSameOrigin(request) {
		t.Fatal("cross-origin stream upgrade must be rejected")
	}
}
