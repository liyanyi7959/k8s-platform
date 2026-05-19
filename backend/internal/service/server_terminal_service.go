package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

const (
	defaultTerminalTicketTTL = 45 * time.Second
	defaultTerminalDialTTL   = 15 * time.Second
)

type TerminalTicket struct {
	Ticket    string `json:"ticket"`
	ExpiresAt string `json:"expires_at"`
}

type TerminalTicketClaims struct {
	ServerID  uint64
	UserID    int64
	ExpiresAt time.Time
}

type terminalTicketItem struct {
	claims TerminalTicketClaims
}

type TerminalTicketStore struct {
	mu  sync.Mutex
	m   map[string]terminalTicketItem
	ttl time.Duration
}

func NewTerminalTicketStore(ttl time.Duration) *TerminalTicketStore {
	if ttl <= 0 {
		ttl = defaultTerminalTicketTTL
	}
	return &TerminalTicketStore{
		m:   make(map[string]terminalTicketItem),
		ttl: ttl,
	}
}

func (s *TerminalTicketStore) IssueWithToken(serverID uint64, userID int64) (string, TerminalTicketClaims) {
	claims := TerminalTicketClaims{
		ServerID:  serverID,
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(s.ttl),
	}
	ticket := randomHex(16)
	s.mu.Lock()
	s.m[ticket] = terminalTicketItem{claims: claims}
	s.mu.Unlock()
	return ticket, claims
}

func (s *TerminalTicketStore) Take(ticket string, userID int64) (TerminalTicketClaims, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return TerminalTicketClaims{}, ErrWithMessage(ErrInvalidParams, "终端票据无效")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.m[ticket]
	if !ok {
		return TerminalTicketClaims{}, ErrWithMessage(ErrInvalidParams, "终端票据无效或已过期")
	}
	delete(s.m, ticket)
	if time.Now().UTC().After(item.claims.ExpiresAt) {
		return TerminalTicketClaims{}, ErrWithMessage(ErrInvalidParams, "终端票据无效或已过期")
	}
	if userID > 0 && item.claims.UserID > 0 && item.claims.UserID != userID {
		return TerminalTicketClaims{}, ErrWithMessage(ErrInvalidParams, "终端票据与当前用户不匹配")
	}
	return item.claims, nil
}

type ServerTerminalSession struct {
	SessionID   string
	ConnectedAt time.Time
	SSHClient   *ssh.Client
	Session     *ssh.Session
	Stdin       io.WriteCloser
	Stdout      io.Reader
	Stderr      io.Reader
}

func (s *ServerTerminalSession) Close() {
	if s == nil {
		return
	}
	if s.Stdin != nil {
		_ = s.Stdin.Close()
	}
	if s.Session != nil {
		_ = s.Session.Close()
	}
	if s.SSHClient != nil {
		_ = s.SSHClient.Close()
	}
}

type ServerTerminalService struct {
	serverSvc   *ServerService
	ticketStore *TerminalTicketStore
	registry    *terminalRegistry
}

type TerminalFavoriteItem struct {
	ServerItem
	FavoritedAt string `json:"favorited_at"`
}

type TerminalAuditItem struct {
	SessionID          string  `json:"session_id"`
	ServerID           uint64  `json:"server_id"`
	ServerName         string  `json:"server_name"`
	ServerIP           string  `json:"server_ip"`
	Status             string  `json:"status"`
	CloseReason        string  `json:"close_reason,omitempty"`
	RiskLevel          string  `json:"risk_level,omitempty"`
	RiskCount          int     `json:"risk_count"`
	LastCommand        string  `json:"last_command,omitempty"`
	StartedAt          string  `json:"started_at"`
	LastActiveAt       string  `json:"last_active_at"`
	EndedAt            *string `json:"ended_at,omitempty"`
	IdleTimeoutSec     int     `json:"idle_timeout_sec"`
	AbsoluteTimeoutSec int     `json:"absolute_timeout_sec"`
}

type TerminalWorkspaceData struct {
	Limits         TerminalLimits            `json:"limits"`
	Favorites      []TerminalFavoriteItem    `json:"favorites"`
	Recent         []ServerItem              `json:"recent"`
	ActiveSessions []TerminalSessionSnapshot `json:"active_sessions"`
}

func NewServerTerminalService(serverSvc *ServerService) *ServerTerminalService {
	svc := &ServerTerminalService{
		serverSvc:   serverSvc,
		ticketStore: NewTerminalTicketStore(defaultTerminalTicketTTL),
	}
	svc.registry = newTerminalRegistry(defaultTerminalIdleTimeout, defaultTerminalAbsoluteTimeout, defaultTerminalMaxSessions, svc.handleSessionClosed, svc.handleRiskCommand)
	return svc
}

func (s *ServerTerminalService) IssueTicket(ctx context.Context, serverID uint64, userID int64) (TerminalTicket, error) {
	if s == nil || s.serverSvc == nil {
		return TerminalTicket{}, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	if serverID == 0 {
		return TerminalTicket{}, ErrInvalidParams
	}
	if _, err := s.serverSvc.GetServer(ctx, serverID); err != nil {
		return TerminalTicket{}, err
	}
	if err := ensureSessionLimit(s.registry.maxSessionsPerUser, s.registry.CountByUser(userID)); err != nil {
		return TerminalTicket{}, err
	}
	ticket, claims := s.ticketStore.IssueWithToken(serverID, userID)
	return TerminalTicket{
		Ticket:    ticket,
		ExpiresAt: claims.ExpiresAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *ServerTerminalService) GetWorkspace(ctx context.Context, userID int64) (TerminalWorkspaceData, error) {
	if s == nil || s.serverSvc == nil {
		return TerminalWorkspaceData{}, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	favorites, err := s.ListFavorites(ctx, userID)
	if err != nil {
		return TerminalWorkspaceData{}, err
	}
	recent, err := s.ListRecentServers(ctx, userID, 8)
	if err != nil {
		return TerminalWorkspaceData{}, err
	}
	return TerminalWorkspaceData{
		Limits:         s.registry.Limits(),
		Favorites:      favorites,
		Recent:         recent,
		ActiveSessions: s.registry.SnapshotByUser(userID),
	}, nil
}

func (s *ServerTerminalService) ListFavorites(ctx context.Context, userID int64) ([]TerminalFavoriteItem, error) {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil {
		return nil, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	if userID <= 0 {
		return []TerminalFavoriteItem{}, nil
	}
	var favorites []model.ServerTerminalFavorite
	err := s.serverSvc.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&favorites).Error
	if err != nil {
		return nil, err
	}
	out := make([]TerminalFavoriteItem, 0, len(favorites))
	for _, favorite := range favorites {
		detail, getErr := s.serverSvc.GetServer(ctx, favorite.ServerID)
		if getErr != nil {
			if errors.Is(getErr, ErrNotFound) {
				continue
			}
			return nil, getErr
		}
		out = append(out, TerminalFavoriteItem{
			ServerItem:  detail.ServerItem,
			FavoritedAt: favorite.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func (s *ServerTerminalService) AddFavorite(ctx context.Context, userID int64, serverID uint64) error {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil {
		return ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	if userID <= 0 || serverID == 0 {
		return ErrInvalidParams
	}
	if _, err := s.serverSvc.GetServer(ctx, serverID); err != nil {
		return err
	}
	item := model.ServerTerminalFavorite{UserID: uint64(userID), ServerID: serverID}
	return firstOrCreateIgnoreNotFound(s.serverSvc.db.WithContext(ctx).Where("user_id = ? AND server_id = ?", userID, serverID), &item)
}

func (s *ServerTerminalService) RemoveFavorite(ctx context.Context, userID int64, serverID uint64) error {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil {
		return ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	if userID <= 0 || serverID == 0 {
		return ErrInvalidParams
	}
	return s.serverSvc.db.WithContext(ctx).Where("user_id = ? AND server_id = ?", userID, serverID).Delete(&model.ServerTerminalFavorite{}).Error
}

func (s *ServerTerminalService) ListRecentServers(ctx context.Context, userID int64, limit int) ([]ServerItem, error) {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil {
		return nil, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	if userID <= 0 {
		return []ServerItem{}, nil
	}
	if limit <= 0 {
		limit = 8
	}
	var ids []uint64
	err := s.serverSvc.db.WithContext(ctx).
		Table("server_terminal_audits AS a").
		Select("a.server_id").
		Where("a.user_id = ?", userID).
		Group("a.server_id").
		Order("MAX(a.started_at) DESC").
		Limit(limit).
		Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	out := make([]ServerItem, 0, len(ids))
	for _, serverID := range ids {
		detail, getErr := s.serverSvc.GetServer(ctx, serverID)
		if getErr != nil {
			if errors.Is(getErr, ErrNotFound) {
				continue
			}
			return nil, getErr
		}
		out = append(out, detail.ServerItem)
	}
	return out, nil
}

func (s *ServerTerminalService) ListAudits(ctx context.Context, userID int64, limit int) ([]TerminalAuditItem, error) {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil {
		return nil, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	if userID <= 0 {
		return []TerminalAuditItem{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []model.ServerTerminalAudit
	err := s.serverSvc.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("last_command IS NOT NULL AND TRIM(last_command) <> ''").
		Order("started_at DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]TerminalAuditItem, 0, len(rows))
	for _, row := range rows {
		item := TerminalAuditItem{
			SessionID:          row.SessionID,
			ServerID:           row.ServerID,
			ServerName:         row.ServerName,
			ServerIP:           row.ServerIP,
			Status:             row.Status,
			RiskLevel:          row.RiskLevel,
			RiskCount:          row.RiskCount,
			StartedAt:          row.StartedAt.UTC().Format(time.RFC3339),
			LastActiveAt:       row.LastActiveAt.UTC().Format(time.RFC3339),
			IdleTimeoutSec:     row.IdleTimeoutSec,
			AbsoluteTimeoutSec: row.AbsoluteTimeoutSec,
		}
		if row.CloseReason != nil {
			item.CloseReason = *row.CloseReason
		}
		if row.LastCommand != nil {
			item.LastCommand = *row.LastCommand
		}
		if row.EndedAt != nil {
			endedAt := row.EndedAt.UTC().Format(time.RFC3339)
			item.EndedAt = &endedAt
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *ServerTerminalService) ConsumeTicket(ticket string, userID int64) (TerminalTicketClaims, error) {
	if s == nil || s.ticketStore == nil {
		return TerminalTicketClaims{}, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	return s.ticketStore.Take(ticket, userID)
}

func (s *ServerTerminalService) OpenSession(ctx context.Context, serverID uint64, cols int, rows int) (*ServerTerminalSession, error) {
	if s == nil || s.serverSvc == nil {
		return nil, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	if cols < 20 {
		cols = 120
	}
	if rows < 5 {
		rows = 36
	}
	info, err := s.serverSvc.GetServerSSHAuth(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(info.Username) == "" || strings.TrimSpace(info.Addr) == "" {
		return nil, ErrWithMessage(ErrInvalidParams, "服务器 SSH 配置不完整")
	}

	var authMethod ssh.AuthMethod
	switch info.AuthType {
	case "password":
		if strings.TrimSpace(info.Password) == "" {
			return nil, ErrInvalidParams
		}
		authMethod = ssh.Password(info.Password)
	case "key":
		if strings.TrimSpace(info.PrivateKey) == "" {
			return nil, ErrInvalidParams
		}
		signer, parseErr := ssh.ParsePrivateKey([]byte(info.PrivateKey))
		if parseErr != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "私钥格式错误")
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		return nil, ErrInvalidParams
	}

	sshCfg := &ssh.ClientConfig{
		User:            info.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         resolveDialTimeout(ctx, defaultTerminalDialTTL),
	}
	sshClient, err := ssh.Dial("tcp", info.Addr, sshCfg)
	if err != nil {
		return nil, normalizeSSHErr(err)
	}

	sess, err := sshClient.NewSession()
	if err != nil {
		_ = sshClient.Close()
		return nil, ErrWithMessage(ErrSSHNetwork, "创建 SSH 会话失败")
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		_ = sshClient.Close()
		return nil, ErrWithMessage(ErrSSHNetwork, "创建终端输入流失败")
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		_ = sess.Close()
		_ = sshClient.Close()
		return nil, ErrWithMessage(ErrSSHNetwork, "创建终端输出流失败")
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = sess.Close()
		_ = sshClient.Close()
		return nil, ErrWithMessage(ErrSSHNetwork, "创建终端错误流失败")
	}
	if err := sess.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{}); err != nil {
		_ = stdin.Close()
		_ = sess.Close()
		_ = sshClient.Close()
		return nil, ErrWithMessage(ErrSSHNetwork, "申请终端 PTY 失败")
	}
	if err := sess.Shell(); err != nil {
		_ = stdin.Close()
		_ = sess.Close()
		_ = sshClient.Close()
		return nil, ErrWithMessage(ErrSSHNetwork, "启动远程 Shell 失败")
	}

	return &ServerTerminalSession{
		SessionID:   randomHex(16),
		ConnectedAt: time.Now().UTC(),
		SSHClient:   sshClient,
		Session:     sess,
		Stdin:       stdin,
		Stdout:      stdout,
		Stderr:      stderr,
	}, nil
}

func (s *ServerTerminalService) RegisterSession(ctx context.Context, session *ServerTerminalSession, serverID uint64, userID int64) (uint64, error) {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil || session == nil {
		return 0, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	detail, err := s.serverSvc.GetServer(ctx, serverID)
	if err != nil {
		return 0, err
	}
	audit := model.ServerTerminalAudit{
		SessionID:          session.SessionID,
		UserID:             uint64(userID),
		ServerID:           serverID,
		ServerName:         detail.Name,
		ServerIP:           detail.IP,
		Status:             "connected",
		RiskLevel:          "none",
		StartedAt:          session.ConnectedAt.UTC(),
		LastActiveAt:       session.ConnectedAt.UTC(),
		IdleTimeoutSec:     s.registry.Limits().IdleTimeoutSec,
		AbsoluteTimeoutSec: s.registry.Limits().AbsoluteTimeoutSec,
	}
	if err := s.serverSvc.db.WithContext(ctx).Create(&audit).Error; err != nil {
		return 0, err
	}
	s.registry.Register(session, userID, serverID, detail.Name, detail.IP, audit.ID)
	return audit.ID, nil
}

func (s *ServerTerminalService) SubscribeSessionEvents(sessionID string) (<-chan TerminalSessionEvent, func(), error) {
	if s == nil || s.registry == nil {
		return nil, nil, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	return s.registry.Subscribe(sessionID)
}

func (s *ServerTerminalService) TouchSession(sessionID string) {
	if s == nil || s.registry == nil {
		return
	}
	s.registry.Touch(sessionID)
	s.updateActiveAudit(sessionID)
}

func (s *ServerTerminalService) HandleSessionInput(sessionID string, data string) {
	if s == nil || s.registry == nil {
		return
	}
	s.registry.HandleInput(sessionID, data)
	s.updateActiveAudit(sessionID)
}

func (s *ServerTerminalService) CloseSession(sessionID string, userID int64, reason string) error {
	if s == nil || s.registry == nil {
		return ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	return s.registry.CloseSessionForUser(sessionID, userID, reason)
}

func (s *ServerTerminalService) ActiveSessions(userID int64) []TerminalSessionSnapshot {
	if s == nil || s.registry == nil {
		return nil
	}
	return s.registry.SnapshotByUser(userID)
}

func (s *ServerTerminalService) handleSessionClosed(managed *terminalManagedSession) {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil || managed == nil {
		return
	}
	managed.mu.Lock()
	endedAt := managed.lastActiveAt.UTC()
	status := managed.status
	if status == "" {
		status = "closed"
	}
	closeReason := managed.closeReason
	lastCommand := managed.lastCommand
	riskLevel := managed.riskLevel
	riskCount := managed.riskCount
	auditID := managed.auditID
	managed.mu.Unlock()

	updates := map[string]any{
		"status":         status,
		"ended_at":       endedAt,
		"last_active_at": endedAt,
		"risk_level":     riskLevel,
		"risk_count":     riskCount,
	}
	if strings.TrimSpace(closeReason) != "" {
		updates["close_reason"] = closeReason
	}
	if strings.TrimSpace(lastCommand) != "" {
		updates["last_command"] = truncateCommand(lastCommand)
	}
	_ = s.serverSvc.db.Model(&model.ServerTerminalAudit{}).Where("id = ?", auditID).Updates(updates).Error
}

func (s *ServerTerminalService) handleRiskCommand(managed *terminalManagedSession, command string, level string) {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil || managed == nil {
		return
	}
	managed.mu.Lock()
	auditID := managed.auditID
	riskCount := managed.riskCount
	managed.mu.Unlock()
	_ = s.serverSvc.db.Model(&model.ServerTerminalAudit{}).Where("id = ?", auditID).Updates(map[string]any{
		"risk_level":   level,
		"risk_count":   riskCount,
		"last_command": truncateCommand(command),
	}).Error
}

func (s *ServerTerminalService) updateActiveAudit(sessionID string) {
	if s == nil || s.registry == nil || s.serverSvc == nil || s.serverSvc.db == nil {
		return
	}
	s.registry.mu.RLock()
	managed, ok := s.registry.sessions[sessionID]
	s.registry.mu.RUnlock()
	if !ok {
		return
	}
	managed.mu.Lock()
	auditID := managed.auditID
	lastActiveAt := managed.lastActiveAt.UTC()
	lastCommand := managed.lastCommand
	riskCount := managed.riskCount
	riskLevel := managed.riskLevel
	managed.mu.Unlock()
	updates := map[string]any{
		"last_active_at": lastActiveAt,
		"risk_count":     riskCount,
		"risk_level":     riskLevel,
	}
	if strings.TrimSpace(lastCommand) != "" {
		updates["last_command"] = truncateCommand(lastCommand)
	}
	_ = s.serverSvc.db.Model(&model.ServerTerminalAudit{}).Where("id = ?", auditID).Updates(updates).Error
}

func (s *ServerTerminalService) dbOrErr() (*gorm.DB, error) {
	if s == nil || s.serverSvc == nil || s.serverSvc.db == nil {
		return nil, ErrWithMessage(ErrInvalidParams, "终端服务未初始化")
	}
	return s.serverSvc.db, nil
}

func firstOrCreateIgnoreNotFound(tx *gorm.DB, dest any, conds ...any) error {
	err := tx.FirstOrCreate(dest, conds...).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func randomHex(size int) string {
	if size <= 0 {
		size = 16
	}
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
