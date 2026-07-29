package application

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"
)

const (
	InformerSyncTimeout  = 12 * time.Second
	informerBackoffBase  = 15 * time.Second
	informerBackoffLimit = 5 * time.Minute
)

// InformerBackoffDelay returns the bounded retry delay after a failed cache
// synchronization. Keeping this schedule in the Kops application layer makes
// every Kubernetes runtime adapter use the same failure policy.
func InformerBackoffDelay(failures int) time.Duration {
	if failures <= 1 {
		return informerBackoffBase
	}
	delay := informerBackoffBase
	for attempt := 1; attempt < failures; attempt++ {
		delay *= 2
		if delay >= informerBackoffLimit {
			return informerBackoffLimit
		}
	}
	return delay
}

// InformerRefreshInterval bounds the Redis snapshot refresh cadence so short
// TTLs do not spin and long TTLs do not leave read snapshots stale.
func InformerRefreshInterval(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		ttl = 20 * time.Second
	}
	interval := ttl / 2
	if interval < 2*time.Second {
		return 2 * time.Second
	}
	if interval > 10*time.Second {
		return 10 * time.Second
	}
	return interval
}

// ObjectListCacheKey produces the stable Redis key for a cluster-wide typed
// resource snapshot. The runtime owns persistence; this package owns the key
// contract shared by all Kops reads.
func ObjectListCacheKey(clusterID uint64, kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "unknown"
	}
	kind = strings.ReplaceAll(kind, ":", "_")
	return fmt.Sprintf("k8s:v1:cluster:%d:%s_all", clusterID, kind)
}

func PodsAllCacheKey(clusterID uint64) string {
	return fmt.Sprintf("k8s:v1:cluster:%d:pods_all", clusterID)
}

// PodsListCacheKey includes the namespace and selector dimensions used by pod
// list responses while preserving the all-pods key for the unconstrained case.
func PodsListCacheKey(clusterID uint64, namespace, labelSelector string) string {
	namespace = strings.TrimSpace(namespace)
	labelSelector = strings.TrimSpace(labelSelector)
	if namespace == "" && labelSelector == "" {
		return PodsAllCacheKey(clusterID)
	}
	if namespace == "" {
		namespace = "_all"
	}
	namespace = strings.ReplaceAll(namespace, ":", "_")
	selectorKey := "none"
	if labelSelector != "" {
		sum := sha1.Sum([]byte(labelSelector))
		selectorKey = fmt.Sprintf("%x", sum[:])
	}
	return fmt.Sprintf("k8s:v1:cluster:%d:pods:ns:%s:ls:%s", clusterID, namespace, selectorKey)
}
