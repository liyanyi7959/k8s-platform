package service

import (
	"testing"
	"time"
)

func TestExecSessionStoreDeleteExpired(t *testing.T) {
	store := &ExecSessionStore{
		m: map[string]execSessionItem{
			"expired": {
				sess:      ExecSession{Pod: "expired-pod"},
				expiresAt: time.Now().UTC().Add(-time.Second),
			},
			"active": {
				sess:      ExecSession{Pod: "active-pod"},
				expiresAt: time.Now().UTC().Add(time.Second),
			},
		},
		stopCh: make(chan struct{}),
	}

	removed := store.deleteExpired(time.Now().UTC())
	if removed != 1 {
		t.Fatalf("deleteExpired() removed %d items, want 1", removed)
	}
	if _, ok := store.m["expired"]; ok {
		t.Fatal("expired session still exists after cleanup")
	}
	if _, ok := store.m["active"]; !ok {
		t.Fatal("active session was removed unexpectedly")
	}
}

func TestExecSessionStoreGCLoopCleansExpiredSessions(t *testing.T) {
	store := &ExecSessionStore{
		m:          make(map[string]execSessionItem),
		ttl:        20 * time.Millisecond,
		gcInterval: 10 * time.Millisecond,
		stopCh:     make(chan struct{}),
	}
	go store.gcLoop()
	defer store.Close()

	store.Put("sid", ExecSession{Pod: "demo"})

	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if store.Len() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := store.Len(); got != 0 {
		t.Fatalf("gcLoop() did not clean expired session, len=%d", got)
	}
}
