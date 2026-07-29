package ports

import (
	"context"

	"k8s-platform-backend/internal/fleet/domain"
)

type ListFilter struct {
	Page, PageSize                       int
	Keyword, Status, Type, SortBy, Order string
}
type Patch struct{ Name, Kubeconfig *string }

type Repository interface {
	List(context.Context, ListFilter) ([]domain.Cluster, int64, error)
	CreateImported(context.Context, domain.Cluster, string) (uint64, error)
	Get(context.Context, uint64) (domain.Cluster, error)
	Kubeconfig(context.Context, uint64) (string, error)
	UpdateHealth(context.Context, uint64, bool, int, string) error
	UpdateMonitorSource(context.Context, uint64, string, string, string) error
	Patch(context.Context, uint64, Patch) error
	Delete(context.Context, uint64) error
}

type ClusterRuntime interface {
	NormalizeAndValidate(context.Context, string) (string, error)
	CheckHealth(context.Context, uint64) (bool, int, int, string, error)
	StopCaches(uint64)
}

type DashboardReader interface {
	GetClusterOverview(context.Context, uint64) (map[string]any, error)
	GetClusterCertificateRisks(context.Context, uint64) ([]map[string]any, error)
}
