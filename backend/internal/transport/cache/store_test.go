package cache

import (
	"context"
	"sync"
	"time"
)

type memoryCacheStore struct {
	mutex sync.Mutex
	items map[string][]byte
}

func newMemoryCacheStore() *memoryCacheStore  { return &memoryCacheStore{items: map[string][]byte{}} }
func (cache *memoryCacheStore) Enabled() bool { return true }
func (cache *memoryCacheStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	value, ok := cache.items[key]
	return append([]byte(nil), value...), ok, nil
}
func (cache *memoryCacheStore) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.items[key] = append([]byte(nil), value...)
	return nil
}
func (cache *memoryCacheStore) Del(_ context.Context, key string) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	delete(cache.items, key)
	return nil
}
func (cache *memoryCacheStore) Close() error { return nil }
