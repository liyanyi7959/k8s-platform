package application

import (
	"testing"
	"time"
)

func TestInformerCacheTimingPolicy(t *testing.T) {
	for _, test := range []struct {
		failures int
		want     time.Duration
	}{
		{failures: 0, want: 15 * time.Second},
		{failures: 1, want: 15 * time.Second},
		{failures: 2, want: 30 * time.Second},
		{failures: 6, want: 5 * time.Minute},
	} {
		if got := InformerBackoffDelay(test.failures); got != test.want {
			t.Fatalf("InformerBackoffDelay(%d) = %s, want %s", test.failures, got, test.want)
		}
	}
	for _, test := range []struct {
		ttl  time.Duration
		want time.Duration
	}{
		{ttl: 0, want: 10 * time.Second},
		{ttl: time.Second, want: 2 * time.Second},
		{ttl: 8 * time.Second, want: 4 * time.Second},
		{ttl: time.Minute, want: 10 * time.Second},
	} {
		if got := InformerRefreshInterval(test.ttl); got != test.want {
			t.Fatalf("InformerRefreshInterval(%s) = %s, want %s", test.ttl, got, test.want)
		}
	}
}

func TestInformerCacheKeysStayStable(t *testing.T) {
	if got := ObjectListCacheKey(7, "apps:v1"); got != "k8s:v1:cluster:7:apps_v1_all" {
		t.Fatalf("ObjectListCacheKey() = %q", got)
	}
	if got := PodsListCacheKey(7, "", ""); got != PodsAllCacheKey(7) {
		t.Fatalf("PodsListCacheKey() = %q, want all-pods key", got)
	}
	if got := PodsListCacheKey(7, " default ", "app=api"); got != "k8s:v1:cluster:7:pods:ns:default:ls:a02d6a0ccb94ba5d6f3bc955fec9fec4f5fadb6e" {
		t.Fatalf("PodsListCacheKey() = %q", got)
	}
}
