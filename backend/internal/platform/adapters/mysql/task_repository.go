package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"k8s-platform-backend/internal/platform/domain"
)

// TaskRepository is the MySQL adapter for platform task state. It deliberately
// contains no workflow decisions; those remain in platform/application.
type TaskRepository struct{ db *gorm.DB }

func NewTaskRepository(db *gorm.DB) *TaskRepository { return &TaskRepository{db: db} }

func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	if r == nil || r.db == nil {
		return errors.New("db not initialized")
	}
	if task == nil {
		return errors.New("task is required")
	}
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	if r == nil || r.db == nil {
		return errors.New("db not initialized")
	}
	if task == nil {
		return errors.New("task is required")
	}
	return r.db.WithContext(ctx).Model(&domain.Task{}).Where("id = ?", task.ID).Updates(taskUpdates(task)).Error
}

func (r *TaskRepository) FindByID(ctx context.Context, id uint64) (*domain.Task, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("db not initialized")
	}
	var task domain.Task
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) List(ctx context.Context) ([]domain.Task, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("db not initialized")
	}
	var tasks []domain.Task
	if err := r.db.WithContext(ctx).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func taskUpdates(task *domain.Task) map[string]any {
	return map[string]any{
		"type":       task.Type,
		"status":     task.Status,
		"title":      task.Title,
		"percent":    task.Percent,
		"message":    task.Message,
		"created_by": task.CreatedBy,
		"meta":       task.Meta,
		"steps":      task.Steps,
	}
}

// TaskLogRepository is the MySQL adapter for append-only task logs.
type TaskLogRepository struct{ db *gorm.DB }

func NewTaskLogRepository(db *gorm.DB) *TaskLogRepository { return &TaskLogRepository{db: db} }

func (r *TaskLogRepository) Append(ctx context.Context, entry *domain.TaskLog) error {
	if r == nil || r.db == nil {
		return errors.New("db not initialized")
	}
	if entry == nil {
		return errors.New("task log is required")
	}
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *TaskLogRepository) List(ctx context.Context, taskID uint64, offset, limit int, stepKey string) ([]domain.TaskLog, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("db not initialized")
	}
	var entries []domain.TaskLog
	query := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("id asc")
	if stepKey != "" {
		query = query.Where("step_key = ?", stepKey)
	}
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}
