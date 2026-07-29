package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

// ---------------------------------------------------------------------------
// Constants & shared types for caching
// ---------------------------------------------------------------------------

type cacheBackoffState struct {
	fails int
	next  time.Time
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

// cachedIDs 返回当前缓存中所有集群 ID。
func (m *objCacheManager) cachedIDs() []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]uint64, 0, len(m.clusters))
	for id := range m.clusters {
		ids = append(ids, id)
	}
	return ids
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
	st.next = now.Add(kopsapp.InformerBackoffDelay(st.fails))
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
		case <-time.After(kopsapp.InformerSyncTimeout):
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
	interval := kopsapp.InformerRefreshInterval(ttl)

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
				case "deployments", "statefulsets", "configmaps", "secrets":
					s.refreshTypedObjectSnapshot(clusterID, entry.kind, entry.informer, ttl)
				}
			}
		}
	}()
}

// refreshTypedObjectSnapshot keeps the retained cache transport responsible
// only for informer snapshots. Read selection, filtering and Secret masking
// are owned by the Kops cached-configuration runtime.
func (s *K8sService) refreshTypedObjectSnapshot(clusterID uint64, kind string, informer cache.SharedIndexInformer, ttl time.Duration) {
	if s == nil || s.cache == nil || informer == nil || clusterID == 0 {
		return
	}
	objects := informer.GetStore().List()
	if len(objects) == 0 {
		return
	}
	var (
		payload []byte
		err     error
	)
	switch kind {
	case "configmaps":
		items := make([]corev1.ConfigMap, 0, len(objects))
		for _, object := range objects {
			if item, ok := object.(*corev1.ConfigMap); ok && item != nil {
				items = append(items, *item.DeepCopy())
			}
		}
		payload, err = json.Marshal(items)
	case "secrets":
		items := make([]corev1.Secret, 0, len(objects))
		for _, object := range objects {
			if item, ok := object.(*corev1.Secret); ok && item != nil {
				items = append(items, *item.DeepCopy())
			}
		}
		payload, err = json.Marshal(items)
	case "deployments":
		items := make([]appsv1.Deployment, 0, len(objects))
		for _, object := range objects {
			if item, ok := object.(*appsv1.Deployment); ok && item != nil {
				items = append(items, *item.DeepCopy())
			}
		}
		payload, err = json.Marshal(items)
	case "statefulsets":
		items := make([]appsv1.StatefulSet, 0, len(objects))
		for _, object := range objects {
			if item, ok := object.(*appsv1.StatefulSet); ok && item != nil {
				items = append(items, *item.DeepCopy())
			}
		}
		payload, err = json.Marshal(items)
	case "pods":
		items := make([]corev1.Pod, 0, len(objects))
		for _, object := range objects {
			if item, ok := object.(*corev1.Pod); ok && item != nil {
				items = append(items, *item.DeepCopy())
			}
		}
		payload, err = json.Marshal(items)
	default:
		return
	}
	if err != nil || len(payload) == 0 {
		return
	}
	writeContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_ = s.cache.Set(writeContext, kopsapp.ObjectListCacheKey(clusterID, kind), payload, ttl)
	cancel()
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

// cachedIDs 返回当前缓存中所有集群 ID。
func (m *podCacheManager) cachedIDs() []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]uint64, 0, len(m.clusters))
	for id := range m.clusters {
		ids = append(ids, id)
	}
	return ids
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
	st.next = time.Now().Add(kopsapp.InformerBackoffDelay(st.fails))
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
		case <-time.After(kopsapp.InformerSyncTimeout):
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
	interval := kopsapp.InformerRefreshInterval(ttl)

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
				s.refreshTypedObjectSnapshot(clusterID, "pods", entry.informer, ttl)
			}
		}
	}()
}
