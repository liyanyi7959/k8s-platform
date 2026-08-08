package application

import (
	"context"

	"k8s-platform-backend/internal/workspace/domain"
	"k8s-platform-backend/internal/workspace/ports"
)

type Service struct {
	repository ports.Repository
	resources  ports.NamespaceResourceReader
}

func NewService(repository ports.Repository, resources ports.NamespaceResourceReader) *Service {
	return &Service{repository: repository, resources: resources}
}

type ProjectDTO struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ClusterID   uint64 `json:"cluster_id"`
	Namespaces  string `json:"namespaces"`
	QuotaCPU    string `json:"quota_cpu"`
	QuotaMemory string `json:"quota_memory"`
	QuotaPods   string `json:"quota_pods"`
	CreatorID   uint64 `json:"creator_id"`
}

type Page struct {
	List     []ProjectDTO `json:"list"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

type SaveRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ClusterID   uint64 `json:"cluster_id"`
	Namespaces  string `json:"namespaces"`
	QuotaCPU    string `json:"quota_cpu"`
	QuotaMemory string `json:"quota_memory"`
	QuotaPods   string `json:"quota_pods"`
}

func (s *Service) Create(ctx context.Context, request SaveRequest, creatorID uint64) (uint64, error) {
	project := domain.Project{Name: request.Name, Description: request.Description, ClusterID: request.ClusterID, Namespaces: request.Namespaces, QuotaCPU: request.QuotaCPU, QuotaMemory: request.QuotaMemory, QuotaPods: request.QuotaPods, CreatorID: creatorID}
	if err := project.Validate(); err != nil {
		return 0, err
	}
	if err := s.repository.Create(ctx, &project); err != nil {
		return 0, err
	}
	return project.ID, nil
}

func (s *Service) List(ctx context.Context, page, pageSize int) (Page, error) {
	page, pageSize = normalizePage(page, pageSize)
	projects, total, err := s.repository.List(ctx, page, pageSize)
	if err != nil {
		return Page{}, err
	}
	items := make([]ProjectDTO, 0, len(projects))
	for _, project := range projects {
		items = append(items, toDTO(project))
	}
	return Page{List: items, Total: int(total), Page: page, PageSize: pageSize}, nil
}

func (s *Service) Get(ctx context.Context, id uint64) (ProjectDTO, error) {
	project, err := s.repository.Get(ctx, id)
	if err != nil {
		return ProjectDTO{}, err
	}
	return toDTO(project), nil
}

func (s *Service) Update(ctx context.Context, id uint64, request SaveRequest) error {
	project, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	project.Name, project.Description, project.ClusterID = request.Name, request.Description, request.ClusterID
	project.Namespaces, project.QuotaCPU, project.QuotaMemory, project.QuotaPods = request.Namespaces, request.QuotaCPU, request.QuotaMemory, request.QuotaPods
	if err := project.Validate(); err != nil {
		return err
	}
	return s.repository.Update(ctx, project)
}

func (s *Service) AssignNamespaces(ctx context.Context, id uint64, namespaces []string) error {
	project, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	project.AssignNamespaces(namespaces)
	return s.repository.Update(ctx, project)
}

func (s *Service) Delete(ctx context.Context, id uint64) error { return s.repository.Delete(ctx, id) }

type NamespaceResources struct {
	Pods        int `json:"pods"`
	Deployments int `json:"deployments"`
	Services    int `json:"services"`
	ConfigMaps  int `json:"configmaps"`
	Total       int `json:"total"`
}

type Resources struct {
	ClusterID  uint64                        `json:"cluster_id"`
	Namespaces map[string]NamespaceResources `json:"namespaces"`
}

func (s *Service) Resources(ctx context.Context, id uint64) (Resources, error) {
	project, err := s.repository.Get(ctx, id)
	if err != nil {
		return Resources{}, err
	}
	result := Resources{ClusterID: project.ClusterID, Namespaces: map[string]NamespaceResources{}}
	if project.ClusterID == 0 || s.resources == nil {
		return result, nil
	}
	for _, namespace := range domain.SplitNamespaces(project.Namespaces) {
		items, total, summaryErr := s.resources.Summary(ctx, project.ClusterID, namespace)
		if summaryErr != nil {
			result.Namespaces[namespace] = NamespaceResources{}
			continue
		}
		summary := NamespaceResources{Total: total}
		for _, item := range items {
			switch item.Key {
			case "pods|pod":
				summary.Pods = item.Count
			case "deployments|deployment":
				summary.Deployments = item.Count
			case "services|service":
				summary.Services = item.Count
			case "configmaps|configmap":
				summary.ConfigMaps = item.Count
			}
		}
		result.Namespaces[namespace] = summary
	}
	return result, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func toDTO(project domain.Project) ProjectDTO {
	return ProjectDTO{ID: project.ID, Name: project.Name, Description: project.Description, ClusterID: project.ClusterID, Namespaces: project.Namespaces, QuotaCPU: project.QuotaCPU, QuotaMemory: project.QuotaMemory, QuotaPods: project.QuotaPods, CreatorID: project.CreatorID}
}
