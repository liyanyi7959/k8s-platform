package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// TaskStatus 任务状态枚举
const (
	TaskPending  = "pending"
	TaskRunning  = "running"
	TaskSuccess  = "success"
	TaskFailed   = "failed"
	TaskTimeout  = "timeout"
	TaskCanceled = "canceled"
)

// Task 任务模型
type Task struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	Type      string    `gorm:"size:64;not null"`
	Status    string    `gorm:"size:32;not null;default:pending;index"`
	Title     string    `gorm:"size:255"`
	Percent   int       `gorm:"default:0"`
	Message   string    `gorm:"type:text"`
	Meta      JSONMap   `gorm:"type:json"`
	Steps     JSONSteps `gorm:"type:json"`
	CreatedBy uint64    `gorm:"index;default:0"`
	CreatedAt time.Time `gorm:"index;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP(3)"`
}

// TaskLog 任务日志模型
type TaskLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	TaskID    uint64    `gorm:"index;not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP(3)"`
}

// JSONMap 用于 GORM 的 JSON 字段映射
type JSONMap map[string]any

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// JSONSteps 用于 GORM 的 Steps 字段映射
type JSONSteps []TaskStep

type TaskStep struct {
	Key        string     `json:"key"`
	Title      string     `json:"title"`
	Status     string     `json:"status"` // pending/running/success/failed
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Message    string     `json:"message,omitempty"`
}

func (j *JSONSteps) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONSteps) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
