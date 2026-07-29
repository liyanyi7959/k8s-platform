package service

import (
	"gorm.io/gorm"
	platformapp "k8s-platform-backend/internal/platform/application"
)

// Task centre ownership moved to platform/application. These aliases keep
// retained legacy executors source-compatible while their callers migrate.
type (
	TaskStore        = platformapp.TaskStore
	TaskStatus       = platformapp.TaskStatus
	TaskStepStatus   = platformapp.TaskStepStatus
	Task             = platformapp.Task
	TaskStep         = platformapp.TaskStep
	TaskSubStep      = platformapp.TaskSubStep
	TaskLogEntry     = platformapp.TaskLogEntry
	ListTasksRequest = platformapp.ListTasksRequest
)

const (
	TaskPending  = platformapp.TaskPending
	TaskRunning  = platformapp.TaskRunning
	TaskSuccess  = platformapp.TaskSuccess
	TaskFailed   = platformapp.TaskFailed
	TaskTimeout  = platformapp.TaskTimeout
	TaskCanceled = platformapp.TaskCanceled
	StepPending  = platformapp.StepPending
	StepRunning  = platformapp.StepRunning
	StepSuccess  = platformapp.StepSuccess
	StepFailed   = platformapp.StepFailed
)

func NewTaskStore(db *gorm.DB) *TaskStore { return platformapp.NewTaskStore(db) }
