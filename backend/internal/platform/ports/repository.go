package ports

import (
	"context"

	"k8s-platform-backend/internal/platform/domain"
)

type SettingsRepository interface {
	Load(context.Context) (map[string]string, error)
	Save(context.Context, map[string]string) error
}

// TaskRepository persists the platform-wide asynchronous task records.  The
// application layer owns the task workflow; adapters only store and retrieve
// domain records.
type TaskRepository interface {
	Create(context.Context, *domain.Task) error
	Update(context.Context, *domain.Task) error
	FindByID(context.Context, uint64) (*domain.Task, error)
	List(context.Context) ([]domain.Task, error)
}

// TaskLogRepository persists task output independently from task state so
// long-running workflows can append and page logs without coupling to a
// concrete database implementation.
type TaskLogRepository interface {
	Append(context.Context, *domain.TaskLog) error
	List(context.Context, uint64, int, int, string) ([]domain.TaskLog, error)
}
