package kubernetes

import (
	"context"
	"time"

	cachetransport "k8s-platform-backend/internal/transport/cache"
)

// DashboardCache adapts the retained cache transport to the Fleet cache port.
// Cache availability is intentionally best-effort and never part of the
// dashboard application's business contract.
type DashboardCache struct{ store cachetransport.CacheStore }

func NewDashboardCache(store cachetransport.CacheStore) *DashboardCache {
	return &DashboardCache{store: store}
}

func (c *DashboardCache) Enabled() bool {
	return c != nil && c.store != nil && c.store.Enabled()
}

func (c *DashboardCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if c == nil || c.store == nil {
		return nil, false, nil
	}
	return c.store.Get(ctx, key)
}

func (c *DashboardCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.store == nil {
		return nil
	}
	return c.store.Set(ctx, key, value, ttl)
}
