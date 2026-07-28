package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestDashboardCertificateRisksPersistAcrossServiceInstances(t *testing.T) {
	store := newMemoryCacheStore()
	ctx := context.Background()
	written := []map[string]any{{
		"key":       "api-server",
		"name":      "API server certificate",
		"status":    "warn",
		"days_left": 12,
	}}

	first := NewDashboardService(nil, nil, nil, store)
	first.setCachedClusterCertRisks(ctx, 16, written, 10*time.Minute)

	second := NewDashboardService(nil, nil, nil, store)
	loaded, ok := second.getCachedClusterCertRisks(ctx, 16)
	if !ok || len(loaded) != 1 {
		t.Fatalf("expected persisted certificate risks, got ok=%v data=%v", ok, loaded)
	}
	if loaded[0]["key"] != "api-server" || loaded[0]["status"] != "warn" {
		t.Fatalf("unexpected persisted certificate risk: %#v", loaded[0])
	}

	loaded[0]["status"] = "mutated"
	again, ok := second.getCachedClusterCertRisks(ctx, 16)
	if !ok || again[0]["status"] != "warn" {
		t.Fatalf("cache result must be cloned, got %#v", again)
	}
}

func TestDashboardCertificateRisksIgnoreExpiredPersistentEntry(t *testing.T) {
	store := newMemoryCacheStore()
	expired := cachedClusterCertRisks{
		Data:      []map[string]any{{"key": "expired"}},
		ExpiresAt: time.Now().Add(-time.Minute),
	}
	payload, err := json.Marshal(expired)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set(context.Background(), clusterCertRiskCacheKey(16), payload, time.Minute); err != nil {
		t.Fatal(err)
	}

	svc := NewDashboardService(nil, nil, nil, store)
	if loaded, ok := svc.getCachedClusterCertRisks(context.Background(), 16); ok || len(loaded) > 0 {
		t.Fatalf("expired certificate risks must not be served: %#v", loaded)
	}
}
