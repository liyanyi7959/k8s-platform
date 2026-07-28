package ports

import (
	"context"

	"k8s-platform-backend/internal/incident/domain"
)

type ListFilter struct {
	Page     int
	PageSize int
	Status   domain.Status
}

type Repository interface {
	List(context.Context, ListFilter) ([]domain.Incident, int64, error)
	Get(context.Context, uint64) (domain.Incident, []domain.TimelineEntry, error)
	SaveTransition(context.Context, domain.Incident, uint64, domain.TimelineEntry) error
}

type ClusterReader interface {
	Names(context.Context, []uint64) (map[uint64]string, error)
}
