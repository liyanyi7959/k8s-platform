package application

import (
	"context"
	"strings"
)

type NamespaceCreateInput struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type NamespaceRuntime interface {
	List(context.Context, uint64, string, string) (any, error)
	Create(context.Context, uint64, NamespaceCreateInput) error
	Delete(context.Context, uint64, string) error
	YAML(context.Context, uint64, string) (any, error)
	Summary(context.Context, uint64, string) (any, error)
	Inspection(context.Context, uint64, string) (any, error)
	WorkloadInventory(context.Context, uint64, string) (any, error)
}

type NamespaceService struct{ runtime NamespaceRuntime }

func NewNamespaceService(runtime NamespaceRuntime) *NamespaceService {
	return &NamespaceService{runtime: runtime}
}

func (s *NamespaceService) List(ctx context.Context, clusterID uint64, sortBy, order string) (any, error) {
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.List(ctx, clusterID, strings.TrimSpace(sortBy), strings.TrimSpace(order))
}
func (s *NamespaceService) Create(ctx context.Context, clusterID uint64, input NamespaceCreateInput) error {
	if clusterID == 0 {
		return ErrInvalidParams
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return ErrInvalidParams
	}
	labels := make(map[string]string, len(input.Labels))
	for key, value := range input.Labels {
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key != "" {
			labels[key] = value
		}
	}
	input.Labels = labels
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Create(ctx, clusterID, input)
}
func (s *NamespaceService) Delete(ctx context.Context, clusterID uint64, namespace string) error {
	if err := validateNamespace(clusterID, namespace); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, clusterID, strings.TrimSpace(namespace))
}
func (s *NamespaceService) YAML(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if err := validateNamespace(clusterID, namespace); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, clusterID, strings.TrimSpace(namespace))
}
func (s *NamespaceService) Summary(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if err := validateNamespace(clusterID, namespace); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Summary(ctx, clusterID, strings.TrimSpace(namespace))
}
func (s *NamespaceService) Inspection(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if err := validateNamespace(clusterID, namespace); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Inspection(ctx, clusterID, strings.TrimSpace(namespace))
}
func (s *NamespaceService) WorkloadInventory(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if err := validateNamespace(clusterID, namespace); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.WorkloadInventory(ctx, clusterID, strings.TrimSpace(namespace))
}
func validateNamespace(clusterID uint64, namespace string) error {
	if clusterID == 0 || strings.TrimSpace(namespace) == "" {
		return ErrInvalidParams
	}
	return nil
}
