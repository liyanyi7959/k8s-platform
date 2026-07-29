package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	fleetmysql "k8s-platform-backend/internal/fleet/adapters/mysql"
	fleetapp "k8s-platform-backend/internal/fleet/application"
	"k8s-platform-backend/internal/fleet/domain"
)

type ClusterItem = fleetapp.ClusterItem
type ClusterHealth = fleetapp.ClusterHealth
type ClusterDetail = fleetapp.ClusterDetail
type ListClustersRequest = fleetapp.ListRequest
type PatchClusterRequest = fleetapp.PatchRequest

type ClusterRegistryService struct {
	application *fleetapp.Registry
	repository  *fleetmysql.Registry
}

func NewClusterRegistryService(db *gorm.DB, kubeconfigKey string) *ClusterRegistryService {
	repository := fleetmysql.NewRegistry(db, kubeconfigKey)
	return &ClusterRegistryService{application: fleetapp.NewRegistry(repository), repository: repository}
}

func (s *ClusterRegistryService) ListClusters(ctx context.Context, request ListClustersRequest) (PageResult[ClusterItem], error) {
	page, err := s.application.List(ctx, request)
	if err != nil {
		return PageResult[ClusterItem]{}, legacyClusterError(err)
	}
	return PageResult[ClusterItem]{List: page.List, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}
func (s *ClusterRegistryService) ImportCluster(ctx context.Context, name, kubeconfig, description string) (uint64, error) {
	id, err := s.application.Import(ctx, name, kubeconfig, description)
	return id, legacyClusterError(err)
}
func (s *ClusterRegistryService) GetCluster(ctx context.Context, id uint64) (ClusterDetail, error) {
	result, err := s.application.Get(ctx, id)
	return result, legacyClusterError(err)
}
func (s *ClusterRegistryService) GetKubeconfig(ctx context.Context, id uint64) (string, error) {
	result, err := s.application.Kubeconfig(ctx, id)
	return result, legacyClusterError(err)
}
func (s *ClusterRegistryService) UpdateClusterHealth(ctx context.Context, id uint64, apiOK bool, _, nodeTotal int, version string) error {
	return legacyClusterError(s.application.UpdateHealth(ctx, id, apiOK, nodeTotal, version))
}
func (s *ClusterRegistryService) UpdateClusterMonitorSource(ctx context.Context, id uint64, source MonitorSource, url string, status PrometheusStatus) error {
	return legacyClusterError(s.application.UpdateMonitorSource(ctx, id, string(source), url, string(status)))
}
func (s *ClusterRegistryService) GetClusterMonitorSource(ctx context.Context, id uint64) (*domain.Cluster, error) {
	row, err := s.repository.Get(ctx, id)
	if err != nil {
		return nil, legacyClusterError(err)
	}
	return &domain.Cluster{ID: row.ID, MonitorSource: row.MonitorSource, PrometheusURL: row.PrometheusURL, PrometheusStatus: row.PrometheusStatus, PrometheusDetectedAt: row.PrometheusDetectedAt}, nil
}
func (s *ClusterRegistryService) PatchCluster(ctx context.Context, id uint64, request PatchClusterRequest) error {
	return legacyClusterError(s.application.Patch(ctx, id, request))
}
func (s *ClusterRegistryService) DeleteCluster(ctx context.Context, id uint64) error {
	return legacyClusterError(s.application.Delete(ctx, id))
}

func legacyClusterError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, domain.ErrValidation):
		return ErrWithMessage(ErrInvalidParams, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, domain.ErrConflict):
		return ErrConflict
	case errors.Is(err, domain.ErrCrypto):
		return ErrCrypto
	default:
		return err
	}
}
