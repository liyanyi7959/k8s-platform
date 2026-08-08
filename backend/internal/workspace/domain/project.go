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
	return NewNamespaceAssignment(strings.Split(value, ",")).Values()
}

func JoinNamespaces(values []string) string {
	return NewNamespaceAssignment(values).String()
}

// NamespaceAssignment 是项目命名空间分配的值对象，负责规范化与去重不变式。
type NamespaceAssignment struct{ values []string }

// NewNamespaceAssignment 构造命名空间分配：去空、去重并保持原有顺序。
func NewNamespaceAssignment(values []string) NamespaceAssignment {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
	}
	return NamespaceAssignment{values: result}
}

// Values 返回规范化后的命名空间列表。
func (a NamespaceAssignment) Values() []string { return a.values }

// String 返回逗号分隔的持久化表示。
func (a NamespaceAssignment) String() string { return strings.Join(a.values, ",") }

// Contains 判断命名空间是否已分配。
func (a NamespaceAssignment) Contains(name string) bool {
	target := strings.TrimSpace(name)
	for _, value := range a.values {
		if value == target {
			return true
		}
	}
	return false
}

// AssignNamespaces 通过聚合方法更新命名空间分配（规范化去重）。
func (p *Project) AssignNamespaces(values []string) {
	p.Namespaces = NewNamespaceAssignment(values).String()
}
