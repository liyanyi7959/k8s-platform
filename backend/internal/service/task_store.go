// 任务中心（Task Center）的内存存储实现。
//
// 背景：
// - 平台存在一些异步任务（例如创建集群、批量执行脚本等）
// - 这类任务通常需要：任务列表、详情、进度、步骤、日志、取消
//
// 当前实现特点：
// - 采用纯内存存储（TaskStore），适合开发/演示/单机部署
// - 不落库：服务重启后任务会丢失（如需生产级持久化可迁移到 DB/Redis）
// - 线程安全：TaskStore/Task 内部通过 mutex 保护读写
//
// 数据结构说明：
// - Task：任务主体（状态、标题、元数据、步骤、日志）
// - TaskStep：任务步骤（用于在 UI 上展示“执行到哪一步”）
// - PageResult：通用分页返回结构（供 list 接口使用）
package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s-platform-backend/internal/model"

	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskPending  TaskStatus = "pending"
	TaskRunning  TaskStatus = "running"
	TaskSuccess  TaskStatus = "success"
	TaskFailed   TaskStatus = "failed"
	TaskTimeout  TaskStatus = "timeout"
	TaskCanceled TaskStatus = "canceled"
)

type TaskStepStatus string

const (
	StepPending TaskStepStatus = "pending"
	StepRunning TaskStepStatus = "running"
	StepSuccess TaskStepStatus = "success"
	StepFailed  TaskStepStatus = "failed"
)

// Task 是业务层使用的 DTO，与 model.Task 对应
type Task struct {
	ID        int64          `json:"id"`
	Type      string         `json:"type"`
	Status    TaskStatus     `json:"status"`
	Title     *string        `json:"title,omitempty"`
	CreatedAt string         `json:"created_at"`
	CreatedBy int64          `json:"created_by"`
	Percent   *int           `json:"percent,omitempty"`
	Message   *string        `json:"message,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
	Steps     []TaskStep     `json:"steps,omitempty"`

	// store 引用，用于日志写入等快捷操作
	store *TaskStore
}

type TaskStep struct {
	Key        string         `json:"key"`
	Title      string         `json:"title"`
	Status     TaskStepStatus `json:"status"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Message    *string        `json:"message,omitempty"`
}

type TaskStore struct {
	db       *gorm.DB
	cancelMu sync.Mutex
	cancels  map[int64]func()
}

func NewTaskStore(db *gorm.DB) *TaskStore {
	return &TaskStore{db: db, cancels: map[int64]func(){}}
}

func (s *TaskStore) RegisterCancel(id int64, cancel func()) {
	if id <= 0 || cancel == nil {
		return
	}
	s.cancelMu.Lock()
	s.cancels[id] = cancel
	s.cancelMu.Unlock()
}

func (s *TaskStore) UnregisterCancel(id int64) {
	if id <= 0 {
		return
	}
	s.cancelMu.Lock()
	delete(s.cancels, id)
	s.cancelMu.Unlock()
}

func (s *TaskStore) CancelExecution(id int64) {
	if id <= 0 {
		return
	}
	s.cancelMu.Lock()
	cancel := s.cancels[id]
	delete(s.cancels, id)
	s.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// NextID 生成 ID (仅占位，实际由 DB 自增)
func (s *TaskStore) NextID() int64 {
	return 0 // DB auto increment
}

// Put 保存任务到数据库
func (s *TaskStore) Put(t *Task) error {
	if s.db == nil {
		return errors.New("db not initialized")
	}

	m, updates := taskToModelAndUpdates(t)
	if t.ID > 0 {
		if err := s.db.Model(&model.Task{}).Where("id = ?", t.ID).Updates(updates).Error; err != nil {
			return err
		}
	} else {
		if err := s.db.Create(m).Error; err != nil {
			return err
		}
		t.ID = int64(m.ID)
		t.CreatedAt = m.CreatedAt.UTC().Format(time.RFC3339)
	}
	t.store = s
	return nil
}

func taskToModelAndUpdates(t *Task) (*model.Task, map[string]any) {
	m := &model.Task{
		Type:      t.Type,
		Status:    string(t.Status),
		Percent:   0,
		CreatedBy: uint64(t.CreatedBy),
		Title:     "",
		Message:   "",
	}
	if t.ID > 0 {
		m.ID = uint64(t.ID)
	}
	if t.Title != nil {
		m.Title = *t.Title
	}
	if t.Percent != nil {
		m.Percent = *t.Percent
	}
	if t.Message != nil {
		m.Message = *t.Message
	}
	if t.Meta != nil {
		m.Meta = model.JSONMap(t.Meta)
	}

	if t.Steps != nil {
		steps := make(model.JSONSteps, len(t.Steps))
		for i, st := range t.Steps {
			steps[i] = model.TaskStep{
				Key:        st.Key,
				Title:      st.Title,
				Status:     string(st.Status),
				StartedAt:  st.StartedAt,
				FinishedAt: st.FinishedAt,
			}
			if st.Message != nil {
				steps[i].Message = *st.Message
			}
		}
		m.Steps = steps
	}

	updates := map[string]any{
		"type":       m.Type,
		"status":     m.Status,
		"title":      m.Title,
		"percent":    m.Percent,
		"message":    m.Message,
		"created_by": m.CreatedBy,
		"meta":       m.Meta,
		"steps":      m.Steps,
	}
	return m, updates
}

// Get 获取任务
func (s *TaskStore) Get(id int64) (*Task, bool) {
	if s.db == nil {
		return nil, false
	}
	var m model.Task
	if err := s.db.First(&m, id).Error; err != nil {
		return nil, false
	}
	return s.toDTO(&m), true
}

// List 获取任务列表
func (s *TaskStore) List() []*Task {
	if s.db == nil {
		return []*Task{}
	}
	var ms []model.Task
	s.db.Find(&ms)
	out := make([]*Task, len(ms))
	for i, m := range ms {
		out[i] = s.toDTO(&m)
	}
	return out
}

func (s *TaskStore) toDTO(m *model.Task) *Task {
	t := &Task{
		ID:        int64(m.ID),
		Type:      m.Type,
		Status:    TaskStatus(m.Status),
		CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
		CreatedBy: int64(m.CreatedBy),
		store:     s,
	}
	if m.Title != "" {
		title := m.Title
		t.Title = &title
	}
	if m.Percent > 0 {
		p := m.Percent
		t.Percent = &p
	}
	if m.Message != "" {
		msg := m.Message
		t.Message = &msg
	}
	if m.Meta != nil {
		t.Meta = map[string]any(m.Meta)
	}
	if m.Steps != nil {
		steps := make([]TaskStep, len(m.Steps))
		for i, st := range m.Steps {
			steps[i] = TaskStep{
				Key:        st.Key,
				Title:      st.Title,
				Status:     TaskStepStatus(st.Status),
				StartedAt:  st.StartedAt,
				FinishedAt: st.FinishedAt,
			}
			if st.Message != "" {
				msg := st.Message
				steps[i].Message = &msg
			}
		}
		t.Steps = steps
	}
	return t
}

// AppendLog 写入日志到 DB
func (t *Task) AppendLog(line string) {
	if t.store == nil || t.store.db == nil {
		return
	}
	t.store.db.Create(&model.TaskLog{
		TaskID:  uint64(t.ID),
		Content: line,
	})
}

// Logs 分页获取日志
func (t *Task) Logs(offset, limit int) []string {
	if t.store == nil || t.store.db == nil {
		return []string{}
	}
	var logs []model.TaskLog
	t.store.db.Where("task_id = ?", t.ID).
		Order("id asc").
		Offset(offset).
		Limit(limit).
		Find(&logs)

	out := make([]string, len(logs))
	for i, l := range logs {
		out[i] = l.Content
	}
	return out
}

func (t *Task) CanCancel() bool {
	return t.Status == TaskPending || t.Status == TaskRunning
}

func (t *Task) Cancel() error {
	if !t.CanCancel() {
		return errors.New("cannot cancel")
	}
	t.Status = TaskCanceled
	msg := "已取消"
	t.Message = &msg
	// 同步到 DB
	return t.store.Put(t)
}

// UpdateStatus 更新任务状态的辅助方法
func (t *Task) Update() error {
	return t.store.Put(t)
}

// PageResult 已统一迁移至 errors.go。

type ListTasksRequest struct {
	Page      int
	PageSize  int
	Type      string
	Status    string
	Keyword   string
	CreatedBy *int64
	SortBy    string
	Order     string
}

func listAndFilterTasks(all []*Task, req ListTasksRequest) []*Task {
	// 临时保留内存过滤逻辑，实际应下沉到 SQL
	// ... (代码量较大，暂复用原逻辑，后续可优化为 SQL 查询)
	out := make([]*Task, 0, len(all))
	kw := strings.ToLower(strings.TrimSpace(req.Keyword))
	for _, t := range all {
		if req.Type != "" && t.Type != req.Type {
			continue
		}
		if req.Status != "" && string(t.Status) != req.Status {
			continue
		}
		if req.CreatedBy != nil && t.CreatedBy != *req.CreatedBy {
			continue
		}
		if kw != "" {
			title := ""
			if t.Title != nil {
				title = *t.Title
			}
			metaName := ""
			if t.Meta != nil && t.Meta["name"] != nil {
				metaName = fmt.Sprintf("%v", t.Meta["name"])
			}
			hay := strings.ToLower(strings.TrimSpace(title + " " + metaName))
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		out = append(out, t)
	}

	sort.Slice(out, func(i, j int) bool {
		if req.SortBy == "created_at" {
			if req.Order == "asc" {
				return out[i].CreatedAt < out[j].CreatedAt
			}
			return out[i].CreatedAt > out[j].CreatedAt
		}
		return out[i].ID > out[j].ID
	})
	return out
}
