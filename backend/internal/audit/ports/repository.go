package ports

import (
	"context"
	"time"

	"k8s-platform-backend/internal/audit/domain"
)

type ListQuery struct {
	Page      int
	PageSize  int
	Keyword   string
	Username  string
	Action    string
	Resource  string
	ClusterID uint64
	Status    string
	StartTime *time.Time
	EndTime   *time.Time
}

type Repository interface {
	Append(context.Context, domain.Entry) error
	List(context.Context, ListQuery) ([]domain.Entry, int64, error)
	// DeleteBefore 删除早于 before 的记录，返回受影响行数。用于保留策略清理。
	DeleteBefore(context.Context, time.Time) (int64, error)
}

type Recorder interface {
	Record(context.Context, domain.Entry)
}
