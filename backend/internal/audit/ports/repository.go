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
}

type Recorder interface {
	Record(context.Context, domain.Entry)
}
