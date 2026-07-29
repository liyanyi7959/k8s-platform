package ports

import (
	"context"

	"k8s-platform-backend/internal/workspace/domain"
)

type Repository interface {
	Create(context.Context, *domain.Project) error
	List(context.Context, int, int) ([]domain.Project, int64, error)
	Get(context.Context, uint64) (domain.Project, error)
	Update(context.Context, domain.Project) error
	Delete(context.Context, uint64) error
}

type ResourceCount struct {
	Key   string
	Count int
}

type NamespaceResourceReader interface {
	Summary(context.Context, uint64, string) ([]ResourceCount, int, error)
}
