// exec_session.go 管理 Pod Exec 会话的临时存储。
//
// 设计要点：
// - ExecSessionStore 为结构体（而非全局变量），支持依赖注入与单元测试；
// - 会话仅存于内存，带 TTL（默认 10 分钟），防止泄漏；
// - 支持 Get（可重复读取）与 Take（读取并删除）两种语义；
// - 通过互斥锁保证并发安全。
package service

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// ExecSession 描述一次 Pod Exec 的目标信息。
// 在前端发起 WebSocket 连接前，先由 REST 接口创建会话并返回 session_id；
// 建链时携带 session_id，服务端据此找到对应的 exec 目标。
type ExecSession struct {
	ClusterID uint64
	Namespace string
	Pod       string
	Container *string
	Command   []string
	TTY       *bool
	CreatedAt time.Time
}

// execSessionItem 为内部存储单元，附带过期时间。
type execSessionItem struct {
	sess      ExecSession
	expiresAt time.Time
}

// ExecSessionStore 提供 Pod Exec 会话的临时存储。
// 取代原有的包级全局变量，便于测试和生命周期管理。
type ExecSessionStore struct {
	mu         sync.RWMutex
	m          map[string]execSessionItem
	ttl        time.Duration
	gcInterval time.Duration
	stopCh     chan struct{}
	stopOnce   sync.Once
}

// defaultExecSessionTTL 为会话的默认过期时间。
const defaultExecSessionTTL = 10 * time.Minute

// defaultExecSessionGCInterval 为后台清理周期。
const defaultExecSessionGCInterval = time.Minute

// NewExecSessionStore 创建一个 ExecSessionStore。
// ttl 为会话过期时间；若 <= 0 则使用默认值（10 分钟）。
func NewExecSessionStore(ttl time.Duration) *ExecSessionStore {
	if ttl <= 0 {
		ttl = defaultExecSessionTTL
	}
	store := &ExecSessionStore{
		m:          make(map[string]execSessionItem),
		ttl:        ttl,
		gcInterval: defaultExecSessionGCInterval,
		stopCh:     make(chan struct{}),
	}
	go store.gcLoop()
	return store
}

func (s *ExecSessionStore) gcLoop() {
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

func (s *ExecSessionStore) deleteExpired(now time.Time) int {
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

// Close 停止后台清理协程。
// 当前主要用于测试和未来的显式资源释放。
func (s *ExecSessionStore) Close() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

// NewSessionID 生成随机 session id（32 位十六进制字符串）。
func (s *ExecSessionStore) NewSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Put 写入一个 exec 会话。
func (s *ExecSessionStore) Put(id string, sess ExecSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = execSessionItem{
		sess:      sess,
		expiresAt: time.Now().UTC().Add(s.ttl),
	}
}

// Get 读取 exec 会话（不删除）。
// 若 session 不存在或已过期，返回 false；过期项会在读取时被清理。
func (s *ExecSessionStore) Get(id string) (ExecSession, bool) {
	now := time.Now().UTC()
	s.mu.RLock()
	it, ok := s.m[id]
	s.mu.RUnlock()
	if !ok {
		return ExecSession{}, false
	}
	if now.After(it.expiresAt) {
		s.mu.Lock()
		delete(s.m, id)
		s.mu.Unlock()
		return ExecSession{}, false
	}
	return it.sess, true
}

// Take 读取并删除 exec 会话（一次性消费）。
// 用于 WebSocket 建链阶段，避免 session_id 被重复使用。
func (s *ExecSessionStore) Take(id string) (ExecSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.m[id]
	if !ok {
		return ExecSession{}, false
	}
	if time.Now().UTC().After(it.expiresAt) {
		delete(s.m, id)
		return ExecSession{}, false
	}
	delete(s.m, id)
	return it.sess, true
}

// Len 返回当前存储的会话数量（包含已过期但尚未被清理的）。
// 主要用于调试/监控。
func (s *ExecSessionStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}
