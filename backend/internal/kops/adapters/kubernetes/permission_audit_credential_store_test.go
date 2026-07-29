package kubernetes

import (
	"context"
	"strings"
	"testing"
	"time"

	cachetransport "k8s-platform-backend/internal/transport/cache"
)

func TestPermissionAuditCredentialStoreRoundTripAndDelete(t *testing.T) {
	store := NewPermissionAuditCredentialStore(cachetransport.NoopCacheStore{}, "test-master-key")
	if err := store.Put(context.Background(), 17, "credential", time.Minute); err != nil {
		t.Fatal(err)
	}
	value, found, err := store.Get(context.Background(), 17)
	if err != nil || !found || value != "credential" {
		t.Fatalf("credential = (%q, %t, %v)", value, found, err)
	}
	store.Delete(context.Background(), 17)
	_, found, err = store.Get(context.Background(), 17)
	if err != nil || found {
		t.Fatalf("credential after delete = (found=%t, err=%v)", found, err)
	}
}

func TestPermissionAuditCredentialStoreUsesEncryptedCachePayload(t *testing.T) {
	cache := &memoryCredentialCache{}
	store := NewPermissionAuditCredentialStore(cache, "test-master-key")
	if err := store.Put(context.Background(), 23, "sensitive-kubeconfig", time.Minute); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cache.values[store.key(23)]), "sensitive-kubeconfig") {
		t.Fatal("cache payload must not contain plaintext credential")
	}
	value, found, err := store.Get(context.Background(), 23)
	if err != nil || !found || value != "sensitive-kubeconfig" {
		t.Fatalf("credential = (%q, %t, %v)", value, found, err)
	}
}

type memoryCredentialCache struct{ values map[string][]byte }

func (m *memoryCredentialCache) Enabled() bool { return true }
func (m *memoryCredentialCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	value, found := m.values[key]
	return value, found, nil
}
func (m *memoryCredentialCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if m.values == nil {
		m.values = map[string][]byte{}
	}
	m.values[key] = append([]byte(nil), value...)
	return nil
}
func (m *memoryCredentialCache) Del(_ context.Context, key string) error {
	delete(m.values, key)
	return nil
}
func (m *memoryCredentialCache) Close() error { return nil }
