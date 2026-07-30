package memory

import (
	"testing"
	"time"

	"k8s-platform-backend/internal/provisioning/ports"
)

func TestTerminalSessionStoreConsumesTicketOnce(t *testing.T) {
	store := NewTerminalSessionStore(time.Minute)
	t.Cleanup(store.Close)

	store.Put("ticket", ports.TerminalSession{UserID: 7, ServerID: 11})
	if session, ok := store.Get("ticket"); !ok || session.UserID != 7 || session.ServerID != 11 {
		t.Fatalf("Get() = %#v, %v", session, ok)
	}
	if session, ok := store.Take("ticket"); !ok || session.ServerID != 11 {
		t.Fatalf("Take() = %#v, %v", session, ok)
	}
	if _, ok := store.Get("ticket"); ok {
		t.Fatal("ticket remained available after Take")
	}
}

func TestTerminalSessionStoreExpiresTickets(t *testing.T) {
	store := NewTerminalSessionStore(time.Nanosecond)
	t.Cleanup(store.Close)
	store.Put("expired", ports.TerminalSession{})
	time.Sleep(time.Millisecond)
	if _, ok := store.Get("expired"); ok {
		t.Fatal("expired ticket remained available")
	}
}
