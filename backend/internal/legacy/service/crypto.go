package service

import (
	"time"

	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	cachetransport "k8s-platform-backend/internal/transport/cache"
)

// PermissionAuditCredentialTTL is retained as a composition-root compatibility
// seam while the encrypted credential store itself is owned by the Kops
// Kubernetes adapter.
func PermissionAuditCredentialTTL() time.Duration {
	return kopsclient.PermissionAuditCredentialTTL()
}

// PermissionAuditCredentialStore is an alias, not a legacy implementation.
type PermissionAuditCredentialStore = kopsclient.PermissionAuditCredentialStore

func NewPermissionAuditCredentialStore(cache cachetransport.CacheStore, secret string) *PermissionAuditCredentialStore {
	return kopsclient.NewPermissionAuditCredentialStore(cache, secret)
}
