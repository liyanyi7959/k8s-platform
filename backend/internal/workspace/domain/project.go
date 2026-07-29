package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound   = errors.New("project not found")
	ErrConflict   = errors.New("project conflict")
	ErrValidation = errors.New("project validation failed")
)

type Project struct {
	ID          uint64
	Name        string
	Description string
	ClusterID   uint64
	Namespaces  string
	QuotaCPU    string
	QuotaMemory string
	QuotaPods   string
	CreatorID   uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

func (p *Project) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: 项目参数不能为空", ErrValidation)
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return fmt.Errorf("%w: 项目名称不能为空", ErrValidation)
	}
	p.Namespaces = JoinNamespaces(SplitNamespaces(p.Namespaces))
	return nil
}

func SplitNamespaces(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func JoinNamespaces(values []string) string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return strings.Join(result, ",")
}
