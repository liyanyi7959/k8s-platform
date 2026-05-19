package service

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// ---------------------------------------------------------------------------
// Constants & shared types for caching
// ---------------------------------------------------------------------------

const informerSyncTimeout = 12 * time.Second
const informerBackoffBase = 15 * time.Second
const informerBackoffMax = 5 * time.Minute

type cacheBackoffState struct {
	fails int
	next  time.Time
}

func backoffDuration(fails int) time.Duration {
	if fails <= 0 {
		return informerBackoffBase
	}
	d := informerBackoffBase
	for i := 1; i < fails; i++ {
		d *= 2
		if d >= informerBackoffMax {
			return informerBackoffMax
		}
	}
	if d > informerBackoffMax {
		d = informerBackoffMax
	}
	return d
}

func safeClose(ch chan struct{}) {
	defer func() { _ = recover() }()
	close(ch)
}

// ---------------------------------------------------------------------------
// objCacheManager — typed-resource informer cache (Deployments, StatefulSets, …)
// ---------------------------------------------------------------------------

type objCacheManager struct {
	mu       sync.Mutex
	clusters map[uint64]map[string]*objCacheEntry
	backoff  map[uint64]map[string]*cacheBackoffState
}

type objCacheEntry struct {
	stopCh   chan struct{}
	informer cache.SharedIndexInformer
	synced   atomic.Bool
	kind     string
}

func newObjCacheManager() *objCacheManager {
	return &objCacheManager{clusters: map[uint64]map[string]*objCacheEntry{}, backoff: map[uint64]map[string]*cacheBackoffState{}}
}

func (m *objCacheManager) stop(clusterID uint64) {
	m.mu.Lock()
	entries := m.clusters[clusterID]
	if entries != nil {
		delete(m.clusters, clusterID)
	}
	if m.backoff != nil {
		delete(m.backoff, clusterID)
	}
	m.mu.Unlock()
	for _, e := range entries {
		if e != nil && e.stopCh != nil {
			close(e.stopCh)
		}
	}
}

func (m *objCacheManager) isBackoff(clusterID uint64, kind string) bool {
	if m == nil {
		return false
	}
	now := time.Now()
	per := m.backoff[clusterID]
	if per == nil {
		return false
	}
	st := per[kind]
	if st == nil {
		return false
	}
	return now.Before(st.next)
}

func (m *objCacheManager) recordFailure(clusterID uint64, kind string) {
	if m == nil {
		return
	}
	now := time.Now()
	per := m.backoff[clusterID]
	if per == nil {
		per = map[string]*cacheBackoffState{}
		m.backoff[clusterID] = per
	}
	st := per[kind]
	if st == nil {
		st = &cacheBackoffState{}
		per[kind] = st
	}
	st.fails++
	st.next = now.Add(backoffDuration(st.fails))
}

func (m *objCacheManager) resetBackoff(clusterID uint64, kind string) {
	if m == nil {
		return
	}
	per := m.backoff[clusterID]
	if per == nil {
		return
	}
	delete(per, kind)
	if len(per) == 0 {
		delete(m.backoff, clusterID)
	}
}

func (m *objCacheManager) removeEntry(clusterID uint64, kind string, entry *objCacheEntry) {
	if m == nil || entry == nil {
		return
	}
	per := m.clusters[clusterID]
	if per == nil {
		return
	}
	cur := per[kind]
	if cur != entry {
		return
	}
	delete(per, kind)
	if len(per) == 0 {
		delete(m.clusters, clusterID)
	}
}

func (s *K8sService) getOrStartObjCache(ctx context.Context, clusterID uint64, kind string) (*objCacheEntry, error) {
	if s == nil || s.objCache == nil {
		return nil, errors.New("dependency missing")
	}
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	k := strings.TrimSpace(kind)
	if k == "" {
		return nil, ErrInvalidParams
	}

	s.objCache.mu.Lock()
	per := s.objCache.clusters[clusterID]
	existing := (*objCacheEntry)(nil)
	if per != nil {
		existing = per[k]
	}
	backoff := existing == nil && s.objCache.isBackoff(clusterID, k)
	s.objCache.mu.Unlock()
	if existing != nil {
		return existing, nil
	}
	if backoff {
		return nil, ErrK8sNetwork
	}

	cs, err := s.typedClientForInformer(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	stopCh := make(chan struct{})
	factory := informers.NewSharedInformerFactory(cs, 0)

	var informer cache.SharedIndexInformer
	switch k {
	case "deployments":
		informer = factory.Apps().V1().Deployments().Informer()
	case "statefulsets":
		informer = factory.Apps().V1().StatefulSets().Informer()
	case "configmaps":
		informer = factory.Core().V1().ConfigMaps().Informer()
	case "secrets":
		informer = factory.Core().V1().Secrets().Informer()
	case "serviceaccounts":
		informer = factory.Core().V1().ServiceAccounts().Informer()
	case "hpas":
		informer = factory.Autoscaling().V2().HorizontalPodAutoscalers().Informer()
	default:
		close(stopCh)
		return nil, ErrInvalidParams
	}

	entry := &objCacheEntry{stopCh: stopCh, informer: informer, kind: k}

	s.objCache.mu.Lock()
	cur := s.objCache.clusters[clusterID]
	if cur == nil {
		cur = map[string]*objCacheEntry{}
		s.objCache.clusters[clusterID] = cur
	}
	if existed := cur[k]; existed != nil {
		s.objCache.mu.Unlock()
		close(stopCh)
		return existed, nil
	}
	cur[k] = entry
	s.objCache.mu.Unlock()

	go func() {
		syncedCh := make(chan bool, 1)
		go func() {
			factory.Start(stopCh)
			syncedCh <- cache.WaitForCacheSync(stopCh, informer.HasSynced)
		}()

		select {
		case ok := <-syncedCh:
			if ok {
				entry.synced.Store(true)
				s.objCache.mu.Lock()
				s.objCache.resetBackoff(clusterID, k)
				s.objCache.mu.Unlock()
				s.startObjAllRedisRefresher(clusterID, entry, stopCh)
			}
		case <-time.After(informerSyncTimeout):
			s.objCache.mu.Lock()
			s.objCache.recordFailure(clusterID, k)
			s.objCache.removeEntry(clusterID, k, entry)
			s.objCache.mu.Unlock()
			safeClose(stopCh)
		}
	}()

	return entry, nil
}

func (s *K8sService) startObjAllRedisRefresher(clusterID uint64, entry *objCacheEntry, stopCh <-chan struct{}) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || clusterID == 0 || entry == nil || entry.informer == nil {
		return
	}
	ttl := s.podTTL
	if ttl <= 0 {
		ttl = 20 * time.Second
	}
	interval := ttl / 2
	if interval < 2*time.Second {
		interval = 2 * time.Second
	}
	if interval > 10*time.Second {
		interval = 10 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				if !entry.informer.HasSynced() {
					continue
				}
				switch entry.kind {
				case "deployments":
					items := deploymentsFromInformerStore(entry.informer)
					if len(items) == 0 {
						continue
					}
					arr := make([]appsv1.Deployment, 0, len(items))
					for _, it := range items {
						if it != nil {
							arr = append(arr, *it)
						}
					}
					b, err := json.Marshal(arr)
					if err != nil || len(b) == 0 {
						continue
					}
					wctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, entry.kind), b, ttl)
					cancel()
				case "statefulsets":
					items := statefulSetsFromInformerStore(entry.informer)
					if len(items) == 0 {
						continue
					}
					arr := make([]appsv1.StatefulSet, 0, len(items))
					for _, it := range items {
						if it != nil {
							arr = append(arr, *it)
						}
					}
					b, err := json.Marshal(arr)
					if err != nil || len(b) == 0 {
						continue
					}
					wctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, entry.kind), b, ttl)
					cancel()
				case "configmaps":
					items := configMapsFromInformerStore(entry.informer)
					if len(items) == 0 {
						continue
					}
					arr := make([]corev1.ConfigMap, 0, len(items))
					for _, it := range items {
						if it != nil {
							arr = append(arr, *it)
						}
					}
					b, err := json.Marshal(arr)
					if err != nil || len(b) == 0 {
						continue
					}
					wctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, entry.kind), b, ttl)
					cancel()
				case "secrets":
					items := secretsFromInformerStore(entry.informer)
					if len(items) == 0 {
						continue
					}
					arr := make([]corev1.Secret, 0, len(items))
					for _, it := range items {
						if it != nil {
							arr = append(arr, *it)
						}
					}
					b, err := json.Marshal(arr)
					if err != nil || len(b) == 0 {
						continue
					}
					wctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, entry.kind), b, ttl)
					cancel()
				}
			}
		}
	}()
}

// ---------------------------------------------------------------------------
// podCacheManager — Pod informer cache
// ---------------------------------------------------------------------------

type podCacheManager struct {
	mu       sync.Mutex
	clusters map[uint64]*podCacheEntry
	backoff  map[uint64]*cacheBackoffState
}

type podCacheEntry struct {
	stopCh   chan struct{}
	informer cache.SharedIndexInformer
	synced   atomic.Bool
}

func newPodCacheManager() *podCacheManager {
	return &podCacheManager{clusters: map[uint64]*podCacheEntry{}, backoff: map[uint64]*cacheBackoffState{}}
}

func (m *podCacheManager) stop(clusterID uint64) {
	m.mu.Lock()
	e := m.clusters[clusterID]
	if e != nil {
		delete(m.clusters, clusterID)
	}
	if m.backoff != nil {
		delete(m.backoff, clusterID)
	}
	m.mu.Unlock()
	if e != nil {
		close(e.stopCh)
	}
}

func (m *podCacheManager) isBackoff(clusterID uint64) bool {
	if m == nil {
		return false
	}
	st := m.backoff[clusterID]
	if st == nil {
		return false
	}
	return time.Now().Before(st.next)
}

func (m *podCacheManager) recordFailure(clusterID uint64) {
	if m == nil {
		return
	}
	st := m.backoff[clusterID]
	if st == nil {
		st = &cacheBackoffState{}
		m.backoff[clusterID] = st
	}
	st.fails++
	st.next = time.Now().Add(backoffDuration(st.fails))
}

func (m *podCacheManager) resetBackoff(clusterID uint64) {
	if m == nil {
		return
	}
	delete(m.backoff, clusterID)
}

func (s *K8sService) getOrStartPodCache(ctx context.Context, clusterID uint64) (*podCacheEntry, error) {
	if s == nil || s.podCache == nil {
		return nil, errors.New("dependency missing")
	}
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}

	s.podCache.mu.Lock()
	existing := s.podCache.clusters[clusterID]
	backoff := existing == nil && s.podCache.isBackoff(clusterID)
	s.podCache.mu.Unlock()
	if existing != nil {
		return existing, nil
	}
	if backoff {
		return nil, ErrK8sNetwork
	}

	cs, err := s.typedClientForInformer(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	stopCh := make(chan struct{})
	factory := informers.NewSharedInformerFactory(cs, 0)
	informer := factory.Core().V1().Pods().Informer()

	entry := &podCacheEntry{
		stopCh:   stopCh,
		informer: informer,
	}

	s.podCache.mu.Lock()
	if cur := s.podCache.clusters[clusterID]; cur != nil {
		s.podCache.mu.Unlock()
		close(stopCh)
		return cur, nil
	}
	s.podCache.clusters[clusterID] = entry
	s.podCache.mu.Unlock()

	go func() {
		syncedCh := make(chan bool, 1)
		go func() {
			factory.Start(stopCh)
			syncedCh <- cache.WaitForCacheSync(stopCh, informer.HasSynced)
		}()

		select {
		case ok := <-syncedCh:
			if ok {
				entry.synced.Store(true)
				s.podCache.mu.Lock()
				s.podCache.resetBackoff(clusterID)
				s.podCache.mu.Unlock()
				s.startPodsAllRedisRefresher(clusterID, entry, stopCh)
			}
		case <-time.After(informerSyncTimeout):
			s.podCache.mu.Lock()
			s.podCache.recordFailure(clusterID)
			if cur := s.podCache.clusters[clusterID]; cur == entry {
				delete(s.podCache.clusters, clusterID)
			}
			s.podCache.mu.Unlock()
			safeClose(stopCh)
		}
	}()

	return entry, nil
}

func (s *K8sService) startPodsAllRedisRefresher(clusterID uint64, entry *podCacheEntry, stopCh <-chan struct{}) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || clusterID == 0 || entry == nil || entry.informer == nil {
		return
	}
	ttl := s.podTTL
	if ttl <= 0 {
		ttl = 20 * time.Second
	}
	interval := ttl / 2
	if interval < 2*time.Second {
		interval = 2 * time.Second
	}
	if interval > 10*time.Second {
		interval = 10 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				if !entry.informer.HasSynced() {
					continue
				}
				pods := podsFromInformerStore(entry.informer)
				if len(pods) == 0 {
					continue
				}
				arr := make([]corev1.Pod, 0, len(pods))
				for _, p := range pods {
					if p != nil {
						arr = append(arr, *p)
					}
				}
				b, err := json.Marshal(arr)
				if err != nil || len(b) == 0 {
					continue
				}
				wctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = s.cache.Set(wctx, s.podsAllCacheKey(clusterID), b, ttl)
				cancel()
			}
		}
	}()
}

// ---------------------------------------------------------------------------
// Cache key helpers
// ---------------------------------------------------------------------------

func (s *K8sService) objAllCacheKey(clusterID uint64, kind string) string {
	k := strings.TrimSpace(kind)
	if k == "" {
		k = "unknown"
	}
	k = strings.ReplaceAll(k, ":", "_")
	return fmt.Sprintf("k8s:v1:cluster:%d:%s_all", clusterID, k)
}

func (s *K8sService) podsAllCacheKey(clusterID uint64) string {
	return fmt.Sprintf("k8s:v1:cluster:%d:pods_all", clusterID)
}

func (s *K8sService) podsListCacheKey(clusterID uint64, namespace string, labelSelector string) string {
	ns := strings.TrimSpace(namespace)
	ls := strings.TrimSpace(labelSelector)
	if ns == "" && ls == "" {
		return s.podsAllCacheKey(clusterID)
	}
	if ns == "" {
		ns = "_all"
	}
	ns = strings.ReplaceAll(ns, ":", "_")
	lsKey := "none"
	if ls != "" {
		sum := sha1.Sum([]byte(ls))
		lsKey = fmt.Sprintf("%x", sum[:])
	}
	return fmt.Sprintf("k8s:v1:cluster:%d:pods:ns:%s:ls:%s", clusterID, ns, lsKey)
}
