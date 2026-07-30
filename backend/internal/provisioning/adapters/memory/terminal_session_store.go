package memory

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"k8s-platform-backend/internal/provisioning/ports"
)

const (
	defaultTerminalSessionTTL        = time.Minute
	defaultTerminalSessionGCInterval = time.Minute
)

type terminalSessionItem struct {
	session   ports.TerminalSession
	expiresAt time.Time
}

// TerminalSessionStore is the in-memory adapter for one-time provisioning
// terminal tickets. It is deliberately independent from Kops' Pod Exec store.
type TerminalSessionStore struct {
	mu         sync.RWMutex
	sessions   map[string]terminalSessionItem
	ttl        time.Duration
	gcInterval time.Duration
	stopCh     chan struct{}
	stopOnce   sync.Once
}

func NewTerminalSessionStore(ttl time.Duration) *TerminalSessionStore {
	if ttl <= 0 {
		ttl = defaultTerminalSessionTTL
	}
	store := &TerminalSessionStore{
		sessions:   make(map[string]terminalSessionItem),
		ttl:        ttl,
		gcInterval: defaultTerminalSessionGCInterval,
		stopCh:     make(chan struct{}),
	}
	go store.gcLoop()
	return store
}

func (s *TerminalSessionStore) Close() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stopCh) })
}

func (s *TerminalSessionStore) NewSessionID() string {
	value := make([]byte, 16)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

func (s *TerminalSessionStore) Put(id string, session ports.TerminalSession) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = terminalSessionItem{session: session, expiresAt: time.Now().UTC().Add(s.ttl)}
}

func (s *TerminalSessionStore) Get(id string) (ports.TerminalSession, bool) {
	if s == nil || id == "" {
		return ports.TerminalSession{}, false
	}
	now := time.Now().UTC()
	s.mu.RLock()
	item, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return ports.TerminalSession{}, false
	}
	if now.After(item.expiresAt) {
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		return ports.TerminalSession{}, false
	}
	return item.session, true
}

func (s *TerminalSessionStore) Take(id string) (ports.TerminalSession, bool) {
	if s == nil || id == "" {
		return ports.TerminalSession{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessions[id]
	if !ok {
		return ports.TerminalSession{}, false
	}
	delete(s.sessions, id)
	if time.Now().UTC().After(item.expiresAt) {
		return ports.TerminalSession{}, false
	}
	return item.session, true
}

func (s *TerminalSessionStore) gcLoop() {
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

func (s *TerminalSessionStore) deleteExpired(now time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, item := range s.sessions {
		if now.After(item.expiresAt) {
			delete(s.sessions, id)
		}
	}
}

var _ ports.TerminalSessionStore = (*TerminalSessionStore)(nil)
