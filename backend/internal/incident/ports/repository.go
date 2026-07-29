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

type MonitoringRepository interface {
	ListAlertRules(context.Context, int, int) ([]domain.AlertRule, int64, error)
	CreateAlertRule(context.Context, *domain.AlertRule) error
	UpdateAlertRule(context.Context, uint64, domain.AlertRule) error
	ToggleAlertRule(context.Context, uint64, bool) error
	DeleteAlertRule(context.Context, uint64) error
	UpsertAlert(context.Context, domain.AlertmanagerAlert) (uint64, error)
	ListLegacyIncidents(context.Context, int, int, string) ([]domain.LegacyIncident, int64, error)
	GetLegacyIncident(context.Context, uint64) (domain.LegacyIncident, []domain.TimelineEntry, error)
	LinkAI(context.Context, uint64, string, uint64, domain.Actor) error
}
