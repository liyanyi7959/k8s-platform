package application

import (
	"context"
	"sync"
	"testing"
	"time"
)

type memoryAttemptCache struct {
	mutex   sync.Mutex
	items   map[string][]byte
	enabled bool
}

func newMemoryAttemptCache() *memoryAttemptCache {
	return &memoryAttemptCache{items: map[string][]byte{}, enabled: true}
}
func (cache *memoryAttemptCache) Enabled() bool { return cache.enabled }
func (cache *memoryAttemptCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	value, ok := cache.items[key]
	return append([]byte(nil), value...), ok, nil
}
func (cache *memoryAttemptCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.items[key] = append([]byte(nil), value...)
	return nil
}
func (cache *memoryAttemptCache) Del(_ context.Context, key string) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	delete(cache.items, key)
	return nil
}

func TestLoginAttemptServiceLocksAndResets(t *testing.T) {
	service := NewLoginAttemptService(newMemoryAttemptCache())
	ctx := context.Background()
	for attempt := 1; attempt < MaxLoginFailures; attempt++ {
		status := service.RecordFailureDetail(ctx, "admin")
		if status.Locked || status.FailedAttempts != attempt {
			t.Fatalf("unexpected attempt %d status: %+v", attempt, status)
		}
	}
	locked := service.RecordFailureDetail(ctx, "admin")
	if !locked.Locked || locked.RemainingAttempts != 0 {
		t.Fatalf("expected locked status: %+v", locked)
	}
	service.ResetFailures(ctx, "admin")
	reset := service.GetStatus(ctx, "admin")
	if reset.Locked || reset.RemainingAttempts != MaxLoginFailures {
		t.Fatalf("unexpected reset status: %+v", reset)
	}
}
func TestLoginAttemptServiceDisabledCache(t *testing.T) {
	service := NewLoginAttemptService(&memoryAttemptCache{})
	status := service.RecordFailureDetail(context.Background(), "admin")
	if status.MaxAttempts != 0 || status.Locked {
		t.Fatalf("expected empty status: %+v", status)
	}
}
func TestFormatLockMessage(t *testing.T) {
	if got := FormatLockMessage(125); got != "账号已被锁定，请 2 分 05 秒后重试" {
		t.Fatalf("unexpected lock message: %q", got)
	}
}
