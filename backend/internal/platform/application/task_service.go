package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s-platform-backend/internal/platform/domain"
	"k8s-platform-backend/internal/platform/ports"
)

// TaskStore owns the platform-wide asynchronous task centre. Cluster, Kops
// and provisioning workflows share this store, so it belongs to platform
// rather than to a legacy business package.
type TaskStore struct {
	tasks    ports.TaskRepository
	logs     ports.TaskLogRepository
	cancelMu sync.Mutex
	cancels  map[int64]func()
}

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

var (
	ErrTaskNotFound     = errors.New("not found")
	ErrTaskCannotCancel = errors.New("cannot cancel")
)

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

	store *TaskStore
}

type TaskStep struct {
	Key        string         `json:"key"`
	Title      string         `json:"title"`
	Status     TaskStepStatus `json:"status"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Message    *string        `json:"message,omitempty"`
	SubSteps   []TaskSubStep  `json:"sub_steps,omitempty"`
}

type TaskSubStep struct {
	Key        string         `json:"key"`
	Title      string         `json:"title"`
	Status     TaskStepStatus `json:"status"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
}

type TaskLogEntry struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func NewTaskStore(tasks ports.TaskRepository, logs ports.TaskLogRepository) *TaskStore {
	return &TaskStore{tasks: tasks, logs: logs, cancels: map[int64]func(){}}
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

// NextID is retained for callers that allocate the record through Put; the
// database remains the source of task identifiers.
func (*TaskStore) NextID() int64 { return 0 }

func (s *TaskStore) Put(task *Task) error {
	if s == nil || s.tasks == nil {
		return errors.New("task repository is required")
	}
	if task == nil {
		return errors.New("task is required")
	}
	row := taskToDomain(task)
	if task.ID > 0 {
		if err := s.tasks.Update(context.Background(), row); err != nil {
			return err
		}
	} else {
		if err := s.tasks.Create(context.Background(), row); err != nil {
			return err
		}
		task.ID = int64(row.ID)
		task.CreatedAt = row.CreatedAt.UTC().Format(time.RFC3339)
	}
	task.store = s
	return nil
}

func (s *TaskStore) Get(id int64) (*Task, bool) {
	if s == nil || s.tasks == nil || id <= 0 {
		return nil, false
	}
	row, err := s.tasks.FindByID(context.Background(), uint64(id))
	if err != nil || row == nil {
		return nil, false
	}
	return s.toTask(row), true
}

func (s *TaskStore) List() []*Task {
	if s == nil || s.tasks == nil {
		return []*Task{}
	}
	rows, err := s.tasks.List(context.Background())
	if err != nil {
		return []*Task{}
	}
	items := make([]*Task, 0, len(rows))
	for index := range rows {
		items = append(items, s.toTask(&rows[index]))
	}
	return items
}

func taskToDomain(task *Task) *domain.Task {
	row := &domain.Task{Type: task.Type, Status: string(task.Status), Percent: 0, CreatedBy: uint64(task.CreatedBy)}
	if task.ID > 0 {
		row.ID = uint64(task.ID)
	}
	if task.Title != nil {
		row.Title = *task.Title
	}
	if task.Percent != nil {
		row.Percent = *task.Percent
	}
	if task.Message != nil {
		row.Message = *task.Message
	}
	if task.Meta != nil {
		row.Meta = domain.JSONMap(task.Meta)
	}
	if task.Steps != nil {
		row.Steps = toDomainTaskSteps(task.Steps)
	}
	return row
}

func toDomainTaskSteps(steps []TaskStep) domain.JSONSteps {
	items := make(domain.JSONSteps, len(steps))
	for index, step := range steps {
		items[index] = domain.TaskStep{Key: step.Key, Title: step.Title, Status: string(step.Status), StartedAt: step.StartedAt, FinishedAt: step.FinishedAt, SubSteps: toDomainSubSteps(step.SubSteps)}
		if step.Message != nil {
			items[index].Message = *step.Message
		}
	}
	return items
}

func toDomainSubSteps(steps []TaskSubStep) []domain.TaskSubStep {
	if steps == nil {
		return nil
	}
	items := make([]domain.TaskSubStep, len(steps))
	for index, step := range steps {
		items[index] = domain.TaskSubStep{Key: step.Key, Title: step.Title, Status: string(step.Status), StartedAt: step.StartedAt, FinishedAt: step.FinishedAt}
	}
	return items
}

func (s *TaskStore) toTask(row *domain.Task) *Task {
	task := &Task{ID: int64(row.ID), Type: row.Type, Status: TaskStatus(row.Status), CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), CreatedBy: int64(row.CreatedBy), store: s}
	if row.Title != "" {
		value := row.Title
		task.Title = &value
	}
	if row.Percent > 0 {
		value := row.Percent
		task.Percent = &value
	}
	if row.Message != "" {
		value := row.Message
		task.Message = &value
	}
	if row.Meta != nil {
		task.Meta = map[string]any(row.Meta)
	}
	if row.Steps != nil {
		task.Steps = fromDomainTaskSteps(row.Steps)
	}
	return task
}

func fromDomainTaskSteps(steps domain.JSONSteps) []TaskStep {
	items := make([]TaskStep, len(steps))
	for index, step := range steps {
		items[index] = TaskStep{Key: step.Key, Title: step.Title, Status: TaskStepStatus(step.Status), StartedAt: step.StartedAt, FinishedAt: step.FinishedAt, SubSteps: fromDomainSubSteps(step.SubSteps)}
		if step.Message != "" {
			value := step.Message
			items[index].Message = &value
		}
	}
	return items
}

func fromDomainSubSteps(steps []domain.TaskSubStep) []TaskSubStep {
	if steps == nil {
		return nil
	}
	items := make([]TaskSubStep, len(steps))
	for index, step := range steps {
		items[index] = TaskSubStep{Key: step.Key, Title: step.Title, Status: TaskStepStatus(step.Status), StartedAt: step.StartedAt, FinishedAt: step.FinishedAt}
	}
	return items
}

func (task *Task) AppendLog(line string, stepKey ...string) {
	if task == nil || task.store == nil || task.store.logs == nil {
		return
	}
	key := ""
	if len(stepKey) > 0 {
		key = stepKey[0]
	}
	_ = task.store.logs.Append(context.Background(), &domain.TaskLog{TaskID: uint64(task.ID), StepKey: key, Content: line})
}

func (task *Task) LogEntries(offset, limit int, stepKey ...string) []TaskLogEntry {
	if task == nil || task.store == nil || task.store.logs == nil {
		return []TaskLogEntry{}
	}
	key := ""
	if len(stepKey) > 0 {
		key = stepKey[0]
	}
	rows, err := task.store.logs.List(context.Background(), uint64(task.ID), offset, limit, key)
	if err != nil {
		return []TaskLogEntry{}
	}
	entries := make([]TaskLogEntry, len(rows))
	for index, row := range rows {
		entries[index] = TaskLogEntry{Content: row.Content, CreatedAt: row.CreatedAt}
	}
	return entries
}

func (task *Task) Logs(offset, limit int, stepKey ...string) []string {
	entries := task.LogEntries(offset, limit, stepKey...)
	items := make([]string, len(entries))
	for index, entry := range entries {
		items[index] = entry.Content
	}
	return items
}

func (task *Task) CanCancel() bool {
	return task != nil && (task.Status == TaskPending || task.Status == TaskRunning)
}

func (task *Task) Cancel() error {
	if !task.CanCancel() {
		return ErrTaskCannotCancel
	}
	if task.store == nil {
		return errors.New("task store is required")
	}
	task.Status = TaskCanceled
	message := "已取消"
	task.Message = &message
	return task.store.Put(task)
}

func (task *Task) Update() error {
	if task == nil || task.store == nil {
		return errors.New("task store is required")
	}
	return task.store.Put(task)
}

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

type PageResult[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type TaskService struct{ store *TaskStore }

func NewTaskService(store *TaskStore) *TaskService { return &TaskService{store: store} }

func (s *TaskService) List(request ListTasksRequest) PageResult[*Task] {
	items := filterTasks(s.store.List(), request)
	page, pageSize := request.Page, request.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	pageItems := []*Task{}
	if start < len(items) {
		pageItems = items[start:end]
	}
	return PageResult[*Task]{List: pageItems, Total: len(items), Page: page, PageSize: pageSize}
}

func (s *TaskService) Get(id int64) (*Task, bool) { return s.store.Get(id) }

func (s *TaskService) Logs(id int64, offset, limit int) ([]string, bool) {
	task, ok := s.store.Get(id)
	if !ok {
		return nil, false
	}
	return task.Logs(offset, limit), true
}

func (s *TaskService) Cancel(id int64) error {
	task, ok := s.store.Get(id)
	if !ok {
		return ErrTaskNotFound
	}
	if !task.CanCancel() {
		return ErrTaskCannotCancel
	}
	if err := task.Cancel(); err != nil {
		return err
	}
	s.store.CancelExecution(id)
	return nil
}

func filterTasks(items []*Task, request ListTasksRequest) []*Task {
	filtered := make([]*Task, 0, len(items))
	keyword := strings.ToLower(strings.TrimSpace(request.Keyword))
	for _, task := range items {
		if request.Type != "" && task.Type != request.Type {
			continue
		}
		if request.Status != "" && string(task.Status) != request.Status {
			continue
		}
		if request.CreatedBy != nil && task.CreatedBy != *request.CreatedBy {
			continue
		}
		if keyword != "" {
			title := ""
			if task.Title != nil {
				title = *task.Title
			}
			metaName := ""
			if task.Meta != nil && task.Meta["name"] != nil {
				metaName = fmt.Sprint(task.Meta["name"])
			}
			if !strings.Contains(strings.ToLower(strings.TrimSpace(title+" "+metaName)), keyword) {
				continue
			}
		}
		filtered = append(filtered, task)
	}
	sort.Slice(filtered, func(left, right int) bool {
		if request.SortBy == "created_at" {
			if request.Order == "asc" {
				return filtered[left].CreatedAt < filtered[right].CreatedAt
			}
			return filtered[left].CreatedAt > filtered[right].CreatedAt
		}
		return filtered[left].ID > filtered[right].ID
	})
	return filtered
}
