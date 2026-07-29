package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Task and its supporting values are platform-wide asynchronous work records.
// They intentionally live outside a business context because cluster, Kops and
// provisioning workflows all persist to the same task centre.
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

type TaskLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	TaskID    uint64    `gorm:"index;not null"`
	StepKey   string    `gorm:"column:step_key;size:128;not null;default:'';index"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP(3)"`
}

type JSONMap map[string]any

func (j *JSONMap) Scan(value any) error {
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

type JSONSteps []TaskStep

type TaskStep struct {
	Key        string        `json:"key"`
	Title      string        `json:"title"`
	Status     string        `json:"status"`
	StartedAt  *time.Time    `json:"started_at,omitempty"`
	FinishedAt *time.Time    `json:"finished_at,omitempty"`
	Message    string        `json:"message,omitempty"`
	SubSteps   []TaskSubStep `json:"sub_steps,omitempty"`
}

type TaskSubStep struct {
	Key        string     `json:"key"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

func (j *JSONSteps) Scan(value any) error {
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
