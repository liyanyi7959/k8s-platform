package application

import (
	"context"
	"testing"
	"time"

	"k8s-platform-backend/internal/platform/domain"
)

func TestFilterTasksAppliesScopeKeywordAndSort(t *testing.T) {
	firstTitle := "Deploy API"
	secondTitle := "Deploy Web"
	items := []*Task{
		{ID: 2, Type: "deploy", Status: TaskRunning, Title: &firstTitle, CreatedBy: 7, CreatedAt: "2026-07-28T10:00:00Z"},
		{ID: 4, Type: "deploy", Status: TaskPending, Title: &secondTitle, CreatedBy: 7, CreatedAt: "2026-07-29T10:00:00Z", Meta: map[string]any{"name": "frontend"}},
		{ID: 8, Type: "audit", Status: TaskRunning, CreatedBy: 8, CreatedAt: "2026-07-30T10:00:00Z"},
	}

	filtered := filterTasks(items, ListTasksRequest{Type: "deploy", CreatedBy: int64Ptr(7), Keyword: "front", SortBy: "created_at", Order: "asc"})
	if len(filtered) != 1 || filtered[0].ID != 4 {
		t.Fatalf("filtered tasks = %#v", filtered)
	}

	ordered := filterTasks(items, ListTasksRequest{Type: "deploy"})
	if len(ordered) != 2 || ordered[0].ID != 4 || ordered[1].ID != 2 {
		t.Fatalf("default order = %#v", ordered)
	}
}

func TestTaskCanCancelOnlyWhilePendingOrRunning(t *testing.T) {
	for _, test := range []struct {
		status TaskStatus
		want   bool
	}{
		{TaskPending, true}, {TaskRunning, true}, {TaskSuccess, false}, {TaskFailed, false}, {TaskCanceled, false},
	} {
		if got := (&Task{Status: test.status}).CanCancel(); got != test.want {
			t.Fatalf("CanCancel(%q) = %v, want %v", test.status, got, test.want)
		}
	}
}

func TestTaskStorePersistsTaskAndLogsThroughPorts(t *testing.T) {
	tasks := newMemoryTaskRepository()
	logs := &memoryTaskLogRepository{}
	store := NewTaskStore(tasks, logs)
	title := "Install cluster"
	task := &Task{Type: "install_cluster", Status: TaskPending, Title: &title, CreatedBy: 7}

	if err := store.Put(task); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if task.ID == 0 || task.CreatedAt == "" {
		t.Fatalf("Put() did not assign task identity: %#v", task)
	}
	task.AppendLog("preflight started", "preflight")
	task.Status = TaskRunning
	if err := task.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	loaded, found := store.Get(task.ID)
	if !found || loaded.Status != TaskRunning {
		t.Fatalf("Get() = %#v, %v; want running task", loaded, found)
	}
	entries := loaded.LogEntries(0, 10, "preflight")
	if len(entries) != 1 || entries[0].Content != "preflight started" {
		t.Fatalf("LogEntries() = %#v", entries)
	}
}

type memoryTaskRepository struct {
	nextID uint64
	tasks  map[uint64]domain.Task
}

func newMemoryTaskRepository() *memoryTaskRepository {
	return &memoryTaskRepository{tasks: map[uint64]domain.Task{}}
}

func (r *memoryTaskRepository) Create(_ context.Context, task *domain.Task) error {
	r.nextID++
	task.ID = r.nextID
	task.CreatedAt = time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	r.tasks[task.ID] = *task
	return nil
}

func (r *memoryTaskRepository) Update(_ context.Context, task *domain.Task) error {
	r.tasks[task.ID] = *task
	return nil
}

func (r *memoryTaskRepository) FindByID(_ context.Context, id uint64) (*domain.Task, error) {
	task, found := r.tasks[id]
	if !found {
		return nil, domainTaskNotFound{}
	}
	return &task, nil
}

func (r *memoryTaskRepository) List(_ context.Context) ([]domain.Task, error) {
	items := make([]domain.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		items = append(items, task)
	}
	return items, nil
}

type memoryTaskLogRepository struct{ entries []domain.TaskLog }

func (r *memoryTaskLogRepository) Append(_ context.Context, entry *domain.TaskLog) error {
	entry.CreatedAt = time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	r.entries = append(r.entries, *entry)
	return nil
}

func (r *memoryTaskLogRepository) List(_ context.Context, taskID uint64, offset, limit int, stepKey string) ([]domain.TaskLog, error) {
	items := make([]domain.TaskLog, 0, len(r.entries))
	for _, entry := range r.entries {
		if entry.TaskID == taskID && (stepKey == "" || entry.StepKey == stepKey) {
			items = append(items, entry)
		}
	}
	if limit <= 0 {
		return items, nil
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []domain.TaskLog{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
}

type domainTaskNotFound struct{}

func (domainTaskNotFound) Error() string { return "task not found" }

func int64Ptr(value int64) *int64 { return &value }
