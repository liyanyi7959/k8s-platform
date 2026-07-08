package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

type memoryCacheStore struct {
	mu    sync.Mutex
	items map[string][]byte
}

func newMemoryCacheStore() *memoryCacheStore {
	return &memoryCacheStore{items: map[string][]byte{}}
}

func (m *memoryCacheStore) Enabled() bool {
	return true
}

func (m *memoryCacheStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.items[key]
	if !ok {
		return nil, false, nil
	}
	copied := append([]byte(nil), value...)
	return copied, true, nil
}

func (m *memoryCacheStore) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = append([]byte(nil), value...)
	return nil
}

func (m *memoryCacheStore) Del(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

func (m *memoryCacheStore) Close() error {
	return nil
}

func TestLoginAttemptService_RecordFailureDetail(t *testing.T) {
	store := newMemoryCacheStore()
	svc := NewLoginAttemptService(store)
	ctx := context.Background()

	initial := svc.GetStatus(ctx, "admin")
	if initial.MaxAttempts != MaxLoginFailures || initial.RemainingAttempts != MaxLoginFailures {
		t.Fatalf("unexpected initial status: %+v", initial)
	}

	for attempt := 1; attempt < MaxLoginFailures; attempt++ {
		status := svc.RecordFailureDetail(ctx, "admin")
		if status.Locked {
			t.Fatalf("attempt %d should not be locked yet: %+v", attempt, status)
		}
		if status.FailedAttempts != attempt {
			t.Fatalf("attempt %d failed_attempts = %d", attempt, status.FailedAttempts)
		}
		wantRemaining := MaxLoginFailures - attempt
		if status.RemainingAttempts != wantRemaining {
			t.Fatalf("attempt %d remaining_attempts = %d, want %d", attempt, status.RemainingAttempts, wantRemaining)
		}
	}

	locked := svc.RecordFailureDetail(ctx, "admin")
	if !locked.Locked {
		t.Fatalf("expected account to be locked on final attempt: %+v", locked)
	}
	if locked.RemainingAttempts != 0 {
		t.Fatalf("locked remaining_attempts = %d, want 0", locked.RemainingAttempts)
	}
	if locked.LockRemainingSeconds <= 0 {
		t.Fatalf("expected positive lock remaining seconds: %+v", locked)
	}

	svc.ResetFailures(ctx, "admin")
	reset := svc.GetStatus(ctx, "admin")
	if reset.Locked || reset.FailedAttempts != 0 || reset.RemainingAttempts != MaxLoginFailures {
		t.Fatalf("unexpected status after reset: %+v", reset)
	}
}

func TestLoginAttemptService_DisabledStoreReturnsEmptyStatus(t *testing.T) {
	svc := NewLoginAttemptService(NoopCacheStore{})
	status := svc.RecordFailureDetail(context.Background(), "admin")
	if status.Locked || status.MaxAttempts != 0 || status.RemainingAttempts != 0 {
		t.Fatalf("expected empty status when cache is disabled: %+v", status)
	}
}

func TestFormatLockMessage(t *testing.T) {
	if got := FormatLockMessage(125); got != "账号已被锁定，请 2 分 05 秒后重试" {
		t.Fatalf("FormatLockMessage(125) = %q", got)
	}
	if got := FormatLockMessage(12); got != "账号已被锁定，请 12 秒后重试" {
		t.Fatalf("FormatLockMessage(12) = %q", got)
	}
}
