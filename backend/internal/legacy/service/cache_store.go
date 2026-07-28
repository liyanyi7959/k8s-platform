package service

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"k8s-platform-backend/internal/config"
)

type CacheStore interface {
	Enabled() bool
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	Close() error
}

type NoopCacheStore struct{}

func (NoopCacheStore) Enabled() bool { return false }
func (NoopCacheStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return nil, false, nil
}
func (NoopCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (NoopCacheStore) Del(ctx context.Context, key string) error { return nil }
func (NoopCacheStore) Close() error                              { return nil }

type RedisCacheStore struct {
	client *redis.Client
}

func NewRedisCacheStore(cfg config.RedisConfig) (CacheStore, error) {
	if !cfg.Enabled {
		return NoopCacheStore{}, nil
	}
	addr := cfg.Addr
	if addr == "" {
		return nil, errors.New("redis.addr is required")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &RedisCacheStore{client: client}, nil
}

func (s *RedisCacheStore) Enabled() bool {
	return s != nil && s.client != nil
}

func (s *RedisCacheStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if s == nil || s.client == nil {
		return nil, false, nil
	}
	v, err := s.client.Get(ctx, key).Bytes()
	if err == nil {
		return v, true, nil
	}
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	return nil, false, err
}

func (s *RedisCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *RedisCacheStore) Del(ctx context.Context, key string) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Del(ctx, key).Err()
}

func (s *RedisCacheStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}
