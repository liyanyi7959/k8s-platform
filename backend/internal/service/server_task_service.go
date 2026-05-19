package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type ServerTaskStatus string

const (
	ServerTaskPending  ServerTaskStatus = "pending"
	ServerTaskRunning  ServerTaskStatus = "running"
	ServerTaskSuccess  ServerTaskStatus = "success"
	ServerTaskFailed   ServerTaskStatus = "failed"
	ServerTaskTimeout  ServerTaskStatus = "timeout"
	ServerTaskCanceled ServerTaskStatus = "canceled"
)

type ServerTaskTargetStatus string

const (
	ServerTaskTargetPending ServerTaskTargetStatus = "pending"
	ServerTaskTargetRunning ServerTaskTargetStatus = "running"
	ServerTaskTargetSuccess ServerTaskTargetStatus = "success"
	ServerTaskTargetFailed  ServerTaskTargetStatus = "failed"
	ServerTaskTargetTimeout ServerTaskTargetStatus = "timeout"
)

type ServerTaskTarget struct {
	ServerID   uint64                 `json:"server_id"`
	Status     ServerTaskTargetStatus `json:"status"`
	ExitCode   *int                   `json:"exit_code,omitempty"`
	DurationMs *int64                 `json:"duration_ms,omitempty"`

	mu   sync.Mutex
	logs []string
}

type ServerTask struct {
	ID         int64            `json:"id"`
	Type       string           `json:"type"`
	Status     ServerTaskStatus `json:"status"`
	CreatedAt  string           `json:"created_at"`
	CreatedBy  int64            `json:"created_by"`
	Command    string           `json:"command"`
	TimeoutSec int              `json:"timeout_sec"`
	Targets    []*ServerTaskTarget

	mu sync.Mutex
}

type ServerTaskStore struct {
	mu    sync.Mutex
	seq   int64
	tasks map[int64]*ServerTask
}

func NewServerTaskStore() *ServerTaskStore {
	return &ServerTaskStore{seq: 9100, tasks: map[int64]*ServerTask{}}
}

func (s *ServerTaskStore) NextID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

func (s *ServerTaskStore) Put(t *ServerTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
}

func (s *ServerTaskStore) Get(id int64) (*ServerTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	return t, ok
}

func (s *ServerTaskStore) List() []*ServerTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*ServerTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out
}

type ServerTaskService struct {
	db        *gorm.DB
	secretKey string
	store     *ServerTaskStore
}

func NewServerTaskService(db *gorm.DB, secretKey string) *ServerTaskService {
	return &ServerTaskService{db: db, secretKey: secretKey, store: NewServerTaskStore()}
}

type ServerBrief struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type CreateServerTaskRequest struct {
	ServerIDs  []uint64
	Command    string
	TimeoutSec int
	CreatedBy  int64
}

func (s *ServerTaskService) CreateTask(ctx context.Context, req CreateServerTaskRequest) (int64, error) {
	if s.db == nil {
		return 0, errors.New("db is required")
	}
	cmd := strings.TrimSpace(req.Command)
	if len(req.ServerIDs) == 0 || cmd == "" {
		return 0, ErrInvalidParams
	}
	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 300
	}
	now := time.Now().UTC()
	id := s.store.NextID()
	targets := make([]*ServerTaskTarget, 0, len(req.ServerIDs))
	for _, sid := range req.ServerIDs {
		if sid == 0 {
			continue
		}
		targets = append(targets, &ServerTaskTarget{ServerID: sid, Status: ServerTaskTargetPending})
	}
	if len(targets) == 0 {
		return 0, ErrInvalidParams
	}
	t := &ServerTask{
		ID:         id,
		Type:       "server_command",
		Status:     ServerTaskPending,
		CreatedAt:  now.Format(time.RFC3339),
		CreatedBy:  req.CreatedBy,
		Command:    cmd,
		TimeoutSec: timeout,
		Targets:    targets,
	}
	s.store.Put(t)
	go s.runTask(t)
	return id, nil
}

func (s *ServerTaskService) GetTask(id int64) (*ServerTask, error) {
	if id <= 0 {
		return nil, ErrInvalidParams
	}
	t, ok := s.store.Get(id)
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

type ListServerTasksRequest struct {
	Page      int
	PageSize  int
	Status    string
	Keyword   string
	CreatedBy *int64
	SortBy    string
	Order     string
}

func (s *ServerTaskService) ListTasks(req ListServerTasksRequest) PageResult[*ServerTask] {
	all := s.store.List()
	filtered := make([]*ServerTask, 0, len(all))
	status := strings.TrimSpace(req.Status)
	kw := strings.ToLower(strings.TrimSpace(req.Keyword))
	for _, t := range all {
		if status != "" && string(t.Status) != status {
			continue
		}
		if req.CreatedBy != nil && t.CreatedBy != *req.CreatedBy {
			continue
		}
		if kw != "" {
			if !strings.Contains(strings.ToLower(t.Command), kw) {
				continue
			}
		}
		filtered = append(filtered, t)
	}

	sortBy := strings.TrimSpace(req.SortBy)
	order := strings.ToLower(strings.TrimSpace(req.Order))
	desc := order != "asc"
	if sortBy == "created_at" {
		sort.SliceStable(filtered, func(i, j int) bool {
			if desc {
				return filtered[i].CreatedAt > filtered[j].CreatedAt
			}
			return filtered[i].CreatedAt < filtered[j].CreatedAt
		})
	} else {
		sort.SliceStable(filtered, func(i, j int) bool {
			if desc {
				return filtered[i].ID > filtered[j].ID
			}
			return filtered[i].ID < filtered[j].ID
		})
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	slice := []*ServerTask{}
	if start < len(filtered) {
		slice = filtered[start:end]
	}
	return PageResult[*ServerTask]{List: slice, Total: len(filtered), Page: page, PageSize: pageSize}
}

func (s *ServerTaskService) GetServerBriefs(ctx context.Context, ids []uint64) (map[uint64]ServerBrief, error) {
	if len(ids) == 0 {
		return map[uint64]ServerBrief{}, nil
	}
	if s.db == nil {
		return nil, errors.New("db is required")
	}
	uniq := make([]uint64, 0, len(ids))
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return map[uint64]ServerBrief{}, nil
	}
	var rows []ServerBrief
	if err := s.db.WithContext(ctx).
		Model(&model.Server{}).
		Select("id", "name", "ip", "port").
		Where("deleted_at IS NULL AND id IN ?", uniq).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint64]ServerBrief, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out, nil
}

func (s *ServerTaskService) GetTaskLogs(id int64, serverID uint64, offset, limit int) (uint64, []string, error) {
	if id <= 0 || serverID == 0 {
		return 0, nil, ErrInvalidParams
	}
	t, ok := s.store.Get(id)
	if !ok {
		return 0, nil, ErrNotFound
	}
	var target *ServerTaskTarget
	for _, tg := range t.Targets {
		if tg.ServerID == serverID {
			target = tg
			break
		}
	}
	if target == nil {
		return 0, nil, ErrNotFound
	}
	target.mu.Lock()
	defer target.mu.Unlock()
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 200
	}
	if offset >= len(target.logs) {
		return target.ServerID, []string{}, nil
	}
	end := offset + limit
	if end > len(target.logs) {
		end = len(target.logs)
	}
	lines := append([]string{}, target.logs[offset:end]...)
	return target.ServerID, lines, nil
}

func (s *ServerTaskService) CancelTask(id int64) error {
	if id <= 0 {
		return ErrInvalidParams
	}
	t, ok := s.store.Get(id)
	if !ok {
		return ErrNotFound
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Status != ServerTaskPending && t.Status != ServerTaskRunning {
		return ErrConflict
	}
	t.Status = ServerTaskCanceled
	now := time.Now().UTC().Format(time.RFC3339)
	for _, tg := range t.Targets {
		tg.mu.Lock()
		if tg.Status == ServerTaskTargetPending || tg.Status == ServerTaskTargetRunning {
			tg.Status = ServerTaskTargetFailed
			tg.logs = append(tg.logs, fmt.Sprintf("[%s] task canceled", now))
		}
		tg.mu.Unlock()
	}
	return nil
}

func (s *ServerTaskService) runTask(t *ServerTask) {
	t.mu.Lock()
	t.Status = ServerTaskRunning
	t.mu.Unlock()
	var wg sync.WaitGroup
	for _, tg := range t.Targets {
		wg.Add(1)
		go func(target *ServerTaskTarget) {
			defer wg.Done()
			s.runOnTarget(t, target)
		}(tg)
	}
	wg.Wait()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Status == ServerTaskCanceled {
		return
	}
	allSuccess := true
	allDone := true
	for _, tg := range t.Targets {
		targetStatus := tg.Status
		if targetStatus == ServerTaskTargetPending || targetStatus == ServerTaskTargetRunning {
			allDone = false
		}
		if targetStatus != ServerTaskTargetSuccess {
			allSuccess = false
		}
	}
	if !allDone {
		return
	}
	if allSuccess {
		t.Status = ServerTaskSuccess
	} else {
		t.Status = ServerTaskFailed
	}
}

func (s *ServerTaskService) runOnTarget(t *ServerTask, target *ServerTaskTarget) {
	t.mu.Lock()
	canceled := t.Status == ServerTaskCanceled
	t.mu.Unlock()
	if canceled {
		target.mu.Lock()
		target.Status = ServerTaskTargetFailed
		target.logs = append(target.logs, fmt.Sprintf("[%s] task canceled", time.Now().UTC().Format(time.RFC3339)))
		target.mu.Unlock()
		return
	}

	target.mu.Lock()
	target.Status = ServerTaskTargetRunning
	target.logs = append(target.logs, fmt.Sprintf("[%s] start", time.Now().UTC().Format(time.RFC3339)))
	target.mu.Unlock()

	start := time.Now().UTC()
	defer func() {
		dur := time.Since(start)
		ms := int64(dur / time.Millisecond)
		target.mu.Lock()
		target.DurationMs = &ms
		target.mu.Unlock()
	}()

	if s.db == nil {
		target.mu.Lock()
		target.Status = ServerTaskTargetFailed
		target.logs = append(target.logs, fmt.Sprintf("[%s] db is required", time.Now().UTC().Format(time.RFC3339)))
		target.mu.Unlock()
		return
	}

	var row model.Server
	if err := s.db.WithContext(context.Background()).
		Where("deleted_at IS NULL AND id = ?", target.ServerID).
		First(&row).Error; err != nil {
		target.mu.Lock()
		target.Status = ServerTaskTargetFailed
		if errors.Is(err, gorm.ErrRecordNotFound) {
			target.logs = append(target.logs, fmt.Sprintf("[%s] server not found", time.Now().UTC().Format(time.RFC3339)))
		} else {
			target.logs = append(target.logs, fmt.Sprintf("[%s] db error: %s", time.Now().UTC().Format(time.RFC3339), err.Error()))
		}
		target.mu.Unlock()
		return
	}

	addr := net.JoinHostPort(strings.TrimSpace(row.IP), strconv.Itoa(row.Port))
	username := strings.TrimSpace(row.Username)
	authType := strings.TrimSpace(row.AuthType)
	var auth ssh.AuthMethod
	switch authType {
	case "password":
		if row.PasswordEnc == nil || strings.TrimSpace(*row.PasswordEnc) == "" {
			target.mu.Lock()
			target.Status = ServerTaskTargetFailed
			target.logs = append(target.logs, fmt.Sprintf("[%s] password not set", time.Now().UTC().Format(time.RFC3339)))
			target.mu.Unlock()
			return
		}
		pass, err := decryptText(s.secretKey, *row.PasswordEnc)
		if err != nil {
			target.mu.Lock()
			target.Status = ServerTaskTargetFailed
			target.logs = append(target.logs, fmt.Sprintf("[%s] decrypt error: %s", time.Now().UTC().Format(time.RFC3339), err.Error()))
			target.mu.Unlock()
			return
		}
		auth = ssh.Password(pass)
	case "key":
		if row.PrivateKeyEnc == nil || strings.TrimSpace(*row.PrivateKeyEnc) == "" {
			target.mu.Lock()
			target.Status = ServerTaskTargetFailed
			target.logs = append(target.logs, fmt.Sprintf("[%s] private key not set", time.Now().UTC().Format(time.RFC3339)))
			target.mu.Unlock()
			return
		}
		key, err := decryptText(s.secretKey, *row.PrivateKeyEnc)
		if err != nil {
			target.mu.Lock()
			target.Status = ServerTaskTargetFailed
			target.logs = append(target.logs, fmt.Sprintf("[%s] decrypt error: %s", time.Now().UTC().Format(time.RFC3339), err.Error()))
			target.mu.Unlock()
			return
		}
		signer, err := ssh.ParsePrivateKey([]byte(key))
		if err != nil {
			target.mu.Lock()
			target.Status = ServerTaskTargetFailed
			target.logs = append(target.logs, fmt.Sprintf("[%s] private key parse error", time.Now().UTC().Format(time.RFC3339)))
			target.mu.Unlock()
			return
		}
		auth = ssh.PublicKeys(signer)
	default:
		target.mu.Lock()
		target.Status = ServerTaskTargetFailed
		target.logs = append(target.logs, fmt.Sprintf("[%s] invalid auth_type", time.Now().UTC().Format(time.RFC3339)))
		target.mu.Unlock()
		return
	}

	sshCfg := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(t.TimeoutSec) * time.Second,
	}
	client, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		target.mu.Lock()
		target.Status = ServerTaskTargetFailed
		target.logs = append(target.logs, fmt.Sprintf("[%s] ssh dial error: %s", time.Now().UTC().Format(time.RFC3339), err.Error()))
		target.mu.Unlock()
		return
	}
	defer func() { _ = client.Close() }()

	sess, err := client.NewSession()
	if err != nil {
		target.mu.Lock()
		target.Status = ServerTaskTargetFailed
		target.logs = append(target.logs, fmt.Sprintf("[%s] new session error: %s", time.Now().UTC().Format(time.RFC3339), err.Error()))
		target.mu.Unlock()
		return
	}
	defer func() { _ = sess.Close() }()

	t.mu.Lock()
	canceled = t.Status == ServerTaskCanceled
	t.mu.Unlock()
	if canceled {
		target.mu.Lock()
		target.Status = ServerTaskTargetFailed
		target.logs = append(target.logs, fmt.Sprintf("[%s] task canceled", time.Now().UTC().Format(time.RFC3339)))
		target.mu.Unlock()
		return
	}

	out, err := sess.CombinedOutput(t.Command)
	nowStr := time.Now().UTC().Format(time.RFC3339)
	target.mu.Lock()
	lines := strings.Split(string(out), "\n")
	for _, ln := range lines {
		trim := strings.TrimRight(ln, "\r\n")
		if trim == "" {
			continue
		}
		target.logs = append(target.logs, fmt.Sprintf("[%s] %s", nowStr, trim))
	}
	if err != nil {
		target.Status = ServerTaskTargetFailed
		target.logs = append(target.logs, fmt.Sprintf("[%s] command error: %s", time.Now().UTC().Format(time.RFC3339), err.Error()))
	} else {
		target.Status = ServerTaskTargetSuccess
	}
	target.mu.Unlock()
}
