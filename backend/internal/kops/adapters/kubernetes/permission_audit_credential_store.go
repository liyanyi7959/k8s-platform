package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	cachetransport "k8s-platform-backend/internal/transport/cache"
	secretcrypto "k8s-platform-backend/internal/transport/secretcrypto"
)

const permissionAuditCredentialTTL = 2 * time.Hour

func PermissionAuditCredentialTTL() time.Duration { return permissionAuditCredentialTTL }

// PermissionAuditCredentialStore keeps ad-hoc kubeconfigs encrypted in the
// configured cache, with a process-local fallback for disabled caches. The
// permission-audit engine only receives its narrow credential lifecycle API.
type PermissionAuditCredentialStore struct {
	cache  cachetransport.CacheStore
	secret string
	mu     sync.RWMutex
	mem    map[uint64]string
}

func NewPermissionAuditCredentialStore(cache cachetransport.CacheStore, secret string) *PermissionAuditCredentialStore {
	return &PermissionAuditCredentialStore{cache: cache, secret: secret, mem: map[uint64]string{}}
}

func (s *PermissionAuditCredentialStore) Put(ctx context.Context, auditID uint64, kubeconfig string, ttl time.Duration) error {
	if auditID == 0 || strings.TrimSpace(kubeconfig) == "" {
		return credentialStoreError("临时凭据不能为空")
	}
	if s.cache != nil && s.cache.Enabled() {
		payload := []byte(kubeconfig)
		if strings.TrimSpace(s.secret) != "" {
			encrypted, err := secretcrypto.Encrypt(s.secret, kubeconfig)
			if err != nil {
				return credentialStoreError("临时凭据加密失败")
			}
			payload = []byte(encrypted)
		}
		if err := s.cache.Set(ctx, s.key(auditID), payload, ttl); err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.mem[auditID] = kubeconfig
	s.mu.Unlock()
	return nil
}

func (s *PermissionAuditCredentialStore) Get(ctx context.Context, auditID uint64) (value string, found bool, err error) {
	if auditID == 0 {
		return "", false, nil
	}
	if s.cache != nil && s.cache.Enabled() {
		payload, cached, cacheErr := s.cache.Get(ctx, s.key(auditID))
		if cacheErr != nil {
			return "", false, cacheErr
		}
		if cached && len(payload) > 0 {
			if strings.TrimSpace(s.secret) != "" {
				value, err = secretcrypto.Decrypt(s.secret, string(payload))
				if err != nil {
					return "", false, credentialStoreError("临时凭据解密失败")
				}
				return value, true, nil
			}
			return string(payload), true, nil
		}
	}
	s.mu.RLock()
	value, found = s.mem[auditID]
	s.mu.RUnlock()
	return value, found && strings.TrimSpace(value) != "", nil
}

func (s *PermissionAuditCredentialStore) Delete(ctx context.Context, auditID uint64) {
	if auditID == 0 {
		return
	}
	if s.cache != nil && s.cache.Enabled() {
		_ = s.cache.Del(ctx, s.key(auditID))
	}
	s.mu.Lock()
	delete(s.mem, auditID)
	s.mu.Unlock()
}

func (s *PermissionAuditCredentialStore) key(auditID uint64) string {
	return fmt.Sprintf("k8s:permission-audit:cred:%d", auditID)
}

type credentialStoreError string

func (e credentialStoreError) Error() string       { return string(e) }
func (e credentialStoreError) UserMessage() string { return string(e) }
