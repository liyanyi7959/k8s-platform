package service

import "testing"

func TestPodLogSessionStorePutTake(t *testing.T) {
	store := NewPodLogSessionStore(0)
	defer store.Close()
	store.Put("sid", PodLogSession{Namespace: "default", Pod: "demo", Follow: true, TailLines: 200})
	if store.Len() != 1 {
		t.Fatalf("len = %d, want 1", store.Len())
	}
	sess, ok := store.Take("sid")
	if !ok {
		t.Fatal("expected session to exist")
	}
	if sess.Namespace != "default" || sess.Pod != "demo" || !sess.Follow || sess.TailLines != 200 {
		t.Fatalf("unexpected session: %+v", sess)
	}
	if store.Len() != 0 {
		t.Fatalf("len = %d, want 0", store.Len())
	}
}
