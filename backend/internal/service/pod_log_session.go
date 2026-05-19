package service

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// PodLogSession 描述一次 Pod 日志 WebSocket 会话的目标信息。
type PodLogSession struct {
	ClusterID uint64
	Namespace string
	Pod       string
	Container *string
	Follow    bool
	TailLines int64
	CreatedAt time.Time
}

type podLogSessionItem struct {
	sess      PodLogSession
	expiresAt time.Time
}

// PodLogSessionStore 提供 Pod 日志会话的临时存储。
type PodLogSessionStore struct {
	mu         sync.RWMutex
	m          map[string]podLogSessionItem
	ttl        time.Duration
	gcInterval time.Duration
	stopCh     chan struct{}
	stopOnce   sync.Once
}

func NewPodLogSessionStore(ttl time.Duration) *PodLogSessionStore {
	if ttl <= 0 {
		ttl = defaultExecSessionTTL
	}
	store := &PodLogSessionStore{
		m:          make(map[string]podLogSessionItem),
		ttl:        ttl,
		gcInterval: defaultExecSessionGCInterval,
		stopCh:     make(chan struct{}),
	}
	go store.gcLoop()
	return store
}

func (s *PodLogSessionStore) gcLoop() {
	if s == nil || s.gcInterval <= 0 {
		return
	}
	ticker := time.NewTicker(s.gcInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.deleteExpired(time.Now().UTC())
		case <-s.stopCh:
			return
		}
	}
}

func (s *PodLogSessionStore) deleteExpired(now time.Time) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for id, item := range s.m {
		if now.After(item.expiresAt) {
			delete(s.m, id)
			removed++
		}
	}
	return removed
}

func (s *PodLogSessionStore) Close() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *PodLogSessionStore) NewSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *PodLogSessionStore) Put(id string, sess PodLogSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = podLogSessionItem{sess: sess, expiresAt: time.Now().UTC().Add(s.ttl)}
}

func (s *PodLogSessionStore) Take(id string) (PodLogSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.m[id]
	if !ok {
		return PodLogSession{}, false
	}
	if time.Now().UTC().After(it.expiresAt) {
		delete(s.m, id)
		return PodLogSession{}, false
	}
	delete(s.m, id)
	return it.sess, true
}

func (s *PodLogSessionStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}
