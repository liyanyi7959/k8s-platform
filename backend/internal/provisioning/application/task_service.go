package application

import (
	"context"
	"strings"
	"time"
)

// DeploymentTask is a stable read model for deployment task progress. It is
// intentionally independent from the legacy task-store DTO.
type DeploymentTask struct {
	ID        int64                `json:"id"`
	Type      string               `json:"type"`
	Status    string               `json:"status"`
	Title     *string              `json:"title,omitempty"`
	CreatedAt string               `json:"created_at"`
	CreatedBy int64                `json:"created_by"`
	Percent   *int                 `json:"percent,omitempty"`
	Message   *string              `json:"message,omitempty"`
	Meta      map[string]any       `json:"meta,omitempty"`
	Steps     []DeploymentTaskStep `json:"steps,omitempty"`
}

type DeploymentTaskStep struct {
	Key        string                  `json:"key"`
	Title      string                  `json:"title"`
	Status     string                  `json:"status"`
	StartedAt  *time.Time              `json:"started_at,omitempty"`
	FinishedAt *time.Time              `json:"finished_at,omitempty"`
	Message    *string                 `json:"message,omitempty"`
	SubSteps   []DeploymentTaskSubStep `json:"sub_steps,omitempty"`
}

type DeploymentTaskSubStep struct {
	Key        string     `json:"key"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type DeploymentTaskLog struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type DeploymentTaskReader interface {
	GetTask(context.Context, int64) (DeploymentTask, error)
	TaskLogs(context.Context, int64, int, int, string) ([]DeploymentTaskLog, error)
}

type TaskService struct{ reader DeploymentTaskReader }

func NewTaskService(reader DeploymentTaskReader) *TaskService { return &TaskService{reader: reader} }

func (s *TaskService) Get(ctx context.Context, taskID int64) (DeploymentTask, error) {
	if taskID <= 0 {
		return DeploymentTask{}, ErrInvalidParams
	}
	return s.reader.GetTask(ctx, taskID)
}

func (s *TaskService) Logs(ctx context.Context, taskID int64, offset, limit int, stepKey string) ([]DeploymentTaskLog, error) {
	if taskID <= 0 {
		return nil, ErrInvalidParams
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	return s.reader.TaskLogs(ctx, taskID, offset, limit, strings.TrimSpace(stepKey))
}

func IsTerminalDeploymentStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "success", "failed", "canceled", "timeout":
		return true
	default:
		return false
	}
}
