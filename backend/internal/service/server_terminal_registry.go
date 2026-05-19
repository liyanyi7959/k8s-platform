package service

import (
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultTerminalIdleTimeout     = 15 * time.Minute
	defaultTerminalAbsoluteTimeout = 4 * time.Hour
	defaultTerminalWarnBefore      = 2 * time.Minute
	defaultTerminalMaxSessions     = 8
)

type TerminalSessionEvent struct {
	Type       string `json:"type"`
	Message    string `json:"message,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	DeadlineAt string `json:"deadline_at,omitempty"`
	Command    string `json:"command,omitempty"`
	RiskLevel  string `json:"risk_level,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type TerminalSessionSnapshot struct {
	SessionID          string `json:"session_id"`
	ServerID           uint64 `json:"server_id"`
	ServerName         string `json:"server_name"`
	ServerIP           string `json:"server_ip"`
	ConnectedAt        string `json:"connected_at"`
	LastActiveAt       string `json:"last_active_at"`
	IdleTimeoutSec     int    `json:"idle_timeout_sec"`
	AbsoluteTimeoutSec int    `json:"absolute_timeout_sec"`
	Status             string `json:"status"`
	LastCommand        string `json:"last_command,omitempty"`
	RiskCount          int    `json:"risk_count"`
	RiskLevel          string `json:"risk_level,omitempty"`
}

type TerminalLimits struct {
	IdleTimeoutSec     int `json:"idle_timeout_sec"`
	AbsoluteTimeoutSec int `json:"absolute_timeout_sec"`
	MaxSessionsPerUser int `json:"max_sessions_per_user"`
}

type terminalManagedSession struct {
	mu              sync.Mutex
	session         *ServerTerminalSession
	userID          int64
	serverID        uint64
	serverName      string
	serverIP        string
	createdAt       time.Time
	lastActiveAt    time.Time
	idleTimeout     time.Duration
	absoluteTimeout time.Duration
	status          string
	closeReason     string
	warned          bool
	riskCount       int
	riskLevel       string
	lastCommand     string
	auditID         uint64
	commandBuf      string
	subscribers     map[chan TerminalSessionEvent]struct{}
}

func (s *terminalManagedSession) snapshot() TerminalSessionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return TerminalSessionSnapshot{
		SessionID:          s.session.SessionID,
		ServerID:           s.serverID,
		ServerName:         s.serverName,
		ServerIP:           s.serverIP,
		ConnectedAt:        s.createdAt.UTC().Format(time.RFC3339),
		LastActiveAt:       s.lastActiveAt.UTC().Format(time.RFC3339),
		IdleTimeoutSec:     int(s.idleTimeout / time.Second),
		AbsoluteTimeoutSec: int(s.absoluteTimeout / time.Second),
		Status:             s.status,
		LastCommand:        s.lastCommand,
		RiskCount:          s.riskCount,
		RiskLevel:          s.riskLevel,
	}
}

type terminalRegistry struct {
	mu                 sync.RWMutex
	sessions           map[string]*terminalManagedSession
	idleTimeout        time.Duration
	absoluteTimeout    time.Duration
	maxSessionsPerUser int
	warnBefore         time.Duration
	onClose            func(*terminalManagedSession)
	onRisk             func(*terminalManagedSession, string, string)
	stopCh             chan struct{}
	stopped            chan struct{}
}

func newTerminalRegistry(idleTimeout, absoluteTimeout time.Duration, maxSessionsPerUser int, onClose func(*terminalManagedSession), onRisk func(*terminalManagedSession, string, string)) *terminalRegistry {
	if idleTimeout <= 0 {
		idleTimeout = defaultTerminalIdleTimeout
	}
	if absoluteTimeout <= 0 {
		absoluteTimeout = defaultTerminalAbsoluteTimeout
	}
	if maxSessionsPerUser <= 0 {
		maxSessionsPerUser = defaultTerminalMaxSessions
	}
	r := &terminalRegistry{
		sessions:           make(map[string]*terminalManagedSession),
		idleTimeout:        idleTimeout,
		absoluteTimeout:    absoluteTimeout,
		maxSessionsPerUser: maxSessionsPerUser,
		warnBefore:         defaultTerminalWarnBefore,
		onClose:            onClose,
		onRisk:             onRisk,
		stopCh:             make(chan struct{}),
		stopped:            make(chan struct{}),
	}
	go r.sweep()
	return r
}

func (r *terminalRegistry) Close() {
	close(r.stopCh)
	<-r.stopped
	var ids []string
	r.mu.RLock()
	for id := range r.sessions {
		ids = append(ids, id)
	}
	r.mu.RUnlock()
	for _, id := range ids {
		r.closeSession(id, "服务关闭")
	}
}

func (r *terminalRegistry) Limits() TerminalLimits {
	return TerminalLimits{
		IdleTimeoutSec:     int(r.idleTimeout / time.Second),
		AbsoluteTimeoutSec: int(r.absoluteTimeout / time.Second),
		MaxSessionsPerUser: r.maxSessionsPerUser,
	}
}

func (r *terminalRegistry) CountByUser(userID int64) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, sess := range r.sessions {
		if sess.userID == userID {
			count++
		}
	}
	return count
}

func (r *terminalRegistry) Register(session *ServerTerminalSession, userID int64, serverID uint64, serverName, serverIP string, auditID uint64) *terminalManagedSession {
	now := time.Now().UTC()
	managed := &terminalManagedSession{
		session:         session,
		userID:          userID,
		serverID:        serverID,
		serverName:      serverName,
		serverIP:        serverIP,
		createdAt:       now,
		lastActiveAt:    now,
		idleTimeout:     r.idleTimeout,
		absoluteTimeout: r.absoluteTimeout,
		status:          "connected",
		riskLevel:       "none",
		auditID:         auditID,
		subscribers:     make(map[chan TerminalSessionEvent]struct{}),
	}
	r.mu.Lock()
	r.sessions[session.SessionID] = managed
	r.mu.Unlock()
	return managed
}

func (r *terminalRegistry) Subscribe(sessionID string) (<-chan TerminalSessionEvent, func(), error) {
	r.mu.RLock()
	managed, ok := r.sessions[sessionID]
	r.mu.RUnlock()
	if !ok {
		return nil, nil, ErrNotFound
	}
	ch := make(chan TerminalSessionEvent, 8)
	managed.mu.Lock()
	managed.subscribers[ch] = struct{}{}
	managed.mu.Unlock()
	unsubscribe := func() {
		managed.mu.Lock()
		if _, ok := managed.subscribers[ch]; ok {
			delete(managed.subscribers, ch)
			close(ch)
		}
		managed.mu.Unlock()
	}
	return ch, unsubscribe, nil
}

func (r *terminalRegistry) Touch(sessionID string) {
	r.mu.RLock()
	managed, ok := r.sessions[sessionID]
	r.mu.RUnlock()
	if !ok {
		return
	}
	managed.mu.Lock()
	managed.lastActiveAt = time.Now().UTC()
	managed.warned = false
	managed.mu.Unlock()
}

func (r *terminalRegistry) HandleInput(sessionID string, data string) {
	r.Touch(sessionID)
	r.mu.RLock()
	managed, ok := r.sessions[sessionID]
	r.mu.RUnlock()
	if !ok || data == "" {
		return
	}
	managed.mu.Lock()
	defer managed.mu.Unlock()
	for _, ch := range data {
		switch ch {
		case '\r', '\n':
			cmd := strings.TrimSpace(managed.commandBuf)
			managed.commandBuf = ""
			if cmd == "" {
				continue
			}
			managed.lastCommand = truncateCommand(cmd)
			level := detectRiskLevel(cmd)
			if level != "none" {
				managed.riskCount++
				managed.riskLevel = level
				r.broadcastLocked(managed, TerminalSessionEvent{
					Type:      "risk-warning",
					Message:   "检测到高风险命令，请谨慎执行",
					SessionID: managed.session.SessionID,
					Command:   managed.lastCommand,
					RiskLevel: level,
				})
				if r.onRisk != nil {
					go r.onRisk(managed, managed.lastCommand, level)
				}
			}
		case '\b', 127:
			if managed.commandBuf != "" {
				managed.commandBuf = managed.commandBuf[:len(managed.commandBuf)-1]
			}
		default:
			if ch >= 32 && ch != 27 {
				managed.commandBuf += string(ch)
			}
		}
	}
}

func (r *terminalRegistry) SnapshotByUser(userID int64) []TerminalSessionSnapshot {
	r.mu.RLock()
	list := make([]*terminalManagedSession, 0, len(r.sessions))
	for _, managed := range r.sessions {
		if userID > 0 && managed.userID != userID {
			continue
		}
		list = append(list, managed)
	}
	r.mu.RUnlock()
	out := make([]TerminalSessionSnapshot, 0, len(list))
	for _, managed := range list {
		out = append(out, managed.snapshot())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ConnectedAt > out[j].ConnectedAt
	})
	return out
}

func (r *terminalRegistry) CloseSessionForUser(sessionID string, userID int64, reason string) error {
	r.mu.RLock()
	managed, ok := r.sessions[sessionID]
	r.mu.RUnlock()
	if !ok {
		return ErrNotFound
	}
	if userID > 0 && managed.userID != userID {
		return ErrNotFound
	}
	r.closeSession(sessionID, reason)
	return nil
}

func (r *terminalRegistry) closeSession(sessionID string, reason string) {
	r.mu.Lock()
	managed, ok := r.sessions[sessionID]
	if !ok {
		r.mu.Unlock()
		return
	}
	delete(r.sessions, sessionID)
	r.mu.Unlock()

	managed.mu.Lock()
	if managed.status == "closed" {
		managed.mu.Unlock()
		return
	}
	managed.status = "closed"
	managed.closeReason = strings.TrimSpace(reason)
	managed.lastActiveAt = time.Now().UTC()
	r.broadcastLocked(managed, TerminalSessionEvent{
		Type:      "closed",
		SessionID: managed.session.SessionID,
		Reason:    managed.closeReason,
		Message:   managed.closeReason,
	})
	subs := make([]chan TerminalSessionEvent, 0, len(managed.subscribers))
	for ch := range managed.subscribers {
		subs = append(subs, ch)
		delete(managed.subscribers, ch)
	}
	managed.mu.Unlock()

	managed.session.Close()
	for _, ch := range subs {
		close(ch)
	}
	if r.onClose != nil {
		r.onClose(managed)
	}
}

func (r *terminalRegistry) sweep() {
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
		close(r.stopped)
	}()
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			now := time.Now().UTC()
			var toClose []string
			r.mu.RLock()
			for id, managed := range r.sessions {
				managed.mu.Lock()
				idleFor := now.Sub(managed.lastActiveAt)
				age := now.Sub(managed.createdAt)
				deadline := managed.lastActiveAt.Add(managed.idleTimeout)
				remaining := time.Until(deadline)
				if managed.status == "connected" && !managed.warned && remaining > 0 && remaining <= r.warnBefore {
					managed.warned = true
					r.broadcastLocked(managed, TerminalSessionEvent{
						Type:       "idle-warning",
						SessionID:  managed.session.SessionID,
						Message:    "长时间无操作，会话即将自动断开",
						DeadlineAt: deadline.UTC().Format(time.RFC3339),
					})
				}
				shouldClose := idleFor >= managed.idleTimeout || age >= managed.absoluteTimeout
				if shouldClose {
					reason := "会话超时已自动断开"
					if age >= managed.absoluteTimeout {
						reason = "会话达到最长使用时长，已自动断开"
					}
					r.broadcastLocked(managed, TerminalSessionEvent{
						Type:      "timeout",
						SessionID: managed.session.SessionID,
						Message:   reason,
						Reason:    reason,
					})
					toClose = append(toClose, id)
				}
				managed.mu.Unlock()
			}
			r.mu.RUnlock()
			for _, id := range toClose {
				r.closeSession(id, "timeout")
			}
		}
	}
}

func (r *terminalRegistry) broadcastLocked(managed *terminalManagedSession, evt TerminalSessionEvent) {
	for ch := range managed.subscribers {
		select {
		case ch <- evt:
		default:
		}
	}
}

func truncateCommand(command string) string {
	trimmed := strings.TrimSpace(command)
	if len(trimmed) <= 255 {
		return trimmed
	}
	return trimmed[:252] + "..."
}

func detectRiskLevel(command string) string {
	lower := strings.ToLower(strings.TrimSpace(command))
	if lower == "" {
		return "none"
	}
	rules := []string{
		"rm -rf /", "mkfs", "fdisk", "dd if=", "shutdown", "reboot",
		"poweroff", "init 0", "kill -9 1", "chmod -r 777 /", "userdel",
	}
	for _, rule := range rules {
		if strings.Contains(lower, rule) {
			return "high"
		}
	}
	warnRules := []string{"iptables", "systemctl stop", "kubectl delete", "truncate -s 0", "rm -f"}
	for _, rule := range warnRules {
		if strings.Contains(lower, rule) {
			return "medium"
		}
	}
	return "none"
}

func ensureSessionLimit(limit, current int) error {
	if limit > 0 && current >= limit {
		return ErrWithMessage(ErrConflict, "终端会话数已达上限，请先关闭其他会话")
	}
	return nil
}
