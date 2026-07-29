package application

import (
	"context"
	"strings"
	"time"

	"k8s-platform-backend/internal/fleet/domain"
	"k8s-platform-backend/internal/fleet/ports"
)

type Registry struct{ repository ports.Repository }

func NewRegistry(repository ports.Repository) *Registry { return &Registry{repository: repository} }

type ClusterItem struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	K8sVersion   string `json:"k8s_version,omitempty"`
	Description  string `json:"description,omitempty"`
	NodeCount    int    `json:"node_count"`
	CreatedAt    string `json:"created_at,omitempty"`
	LastHealthAt string `json:"last_health_at,omitempty"`
}
type ClusterHealth struct {
	APIOk     bool `json:"api_ok"`
	NodeReady int  `json:"node_ready"`
	NodeTotal int  `json:"node_total"`
}
type ClusterDetail struct {
	ClusterItem
	Health *ClusterHealth `json:"health,omitempty"`
}
type Page struct {
	List     []ClusterItem `json:"list"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}
type ListRequest struct {
	Page, PageSize                       int
	Keyword, Status, Type, SortBy, Order string
}
type PatchRequest struct{ Name, Kubeconfig *string }

func (s *Registry) List(ctx context.Context, request ListRequest) (Page, error) {
	request.Page, request.PageSize = normalizePage(request.Page, request.PageSize)
	rows, total, err := s.repository.List(ctx, ports.ListFilter{Page: request.Page, PageSize: request.PageSize, Keyword: request.Keyword, Status: request.Status, Type: request.Type, SortBy: request.SortBy, Order: request.Order})
	if err != nil {
		return Page{}, err
	}
	items := make([]ClusterItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toItem(row))
	}
	return Page{List: items, Total: int(total), Page: request.Page, PageSize: request.PageSize}, nil
}
func (s *Registry) Import(ctx context.Context, name, kubeconfig, description string) (uint64, error) {
	cluster, plain, err := domain.NewImported(name, kubeconfig, description)
	if err != nil {
		return 0, err
	}
	return s.repository.CreateImported(ctx, cluster, plain)
}
func (s *Registry) Get(ctx context.Context, id uint64) (ClusterDetail, error) {
	row, err := s.repository.Get(ctx, id)
	if err != nil {
		return ClusterDetail{}, err
	}
	return ClusterDetail{ClusterItem: toItem(row)}, nil
}
func (s *Registry) Kubeconfig(ctx context.Context, id uint64) (string, error) {
	return s.repository.Kubeconfig(ctx, id)
}
func (s *Registry) UpdateHealth(ctx context.Context, id uint64, apiOK bool, nodeTotal int, version string) error {
	return s.repository.UpdateHealth(ctx, id, apiOK, nodeTotal, version)
}
func (s *Registry) UpdateMonitorSource(ctx context.Context, id uint64, source, url, status string) error {
	return s.repository.UpdateMonitorSource(ctx, id, source, url, status)
}

// MonitorSource returns only the persisted monitoring-source fields required
// by Kops runtime adapters, without exposing registry persistence details.
func (s *Registry) MonitorSource(ctx context.Context, id uint64) (*domain.Cluster, error) {
	row, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.Cluster{
		ID:                   row.ID,
		MonitorSource:        row.MonitorSource,
		PrometheusURL:        row.PrometheusURL,
		PrometheusStatus:     row.PrometheusStatus,
		PrometheusDetectedAt: row.PrometheusDetectedAt,
	}, nil
}
func (s *Registry) Patch(ctx context.Context, id uint64, request PatchRequest) error {
	if request.Name != nil {
		value := strings.TrimSpace(*request.Name)
		if err := domain.ValidateName(value); err != nil {
			return err
		}
		request.Name = &value
	}
	if request.Kubeconfig != nil {
		value := strings.TrimSpace(*request.Kubeconfig)
		if err := domain.ValidateKubeconfig(value); err != nil {
			return err
		}
		request.Kubeconfig = &value
	}
	return s.repository.Patch(ctx, id, ports.Patch{Name: request.Name, Kubeconfig: request.Kubeconfig})
}
func (s *Registry) Delete(ctx context.Context, id uint64) error { return s.repository.Delete(ctx, id) }
func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func toItem(row domain.Cluster) ClusterItem {
	last := ""
	if row.LastHealthAt != nil {
		last = row.LastHealthAt.UTC().Format(time.RFC3339)
	}
	created := ""
	if !row.CreatedAt.IsZero() {
		created = row.CreatedAt.UTC().Format(time.RFC3339)
	}
	return ClusterItem{ID: row.ID, Name: row.Name, Type: row.Type, Status: row.Status, K8sVersion: row.K8sVersion, Description: row.Description, NodeCount: row.NodeCount, CreatedAt: created, LastHealthAt: last}
}
