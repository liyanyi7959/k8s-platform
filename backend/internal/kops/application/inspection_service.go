package application

import (
	"context"
	"strings"
)

// InspectionRuntime supplies the read models used by Kubernetes resource
// inspections. The implementation remains outside the Kops application layer
// while the command validation and route-facing contract live here.
type InspectionRuntime interface {
	Namespace(context.Context, uint64, string) (any, error)
	NamespaceWorkloadInventory(context.Context, uint64, string) (any, error)
	Pod(context.Context, uint64, string, string) (any, error)
}

type InspectionService struct{ runtime InspectionRuntime }

func NewInspectionService(runtime InspectionRuntime) *InspectionService {
	return &InspectionService{runtime: runtime}
}

func (s *InspectionService) Namespace(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if err := validateNamespace(clusterID, namespace); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Namespace(ctx, clusterID, strings.TrimSpace(namespace))
}

func (s *InspectionService) NamespaceWorkloadInventory(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if err := validateNamespace(clusterID, namespace); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.NamespaceWorkloadInventory(ctx, clusterID, strings.TrimSpace(namespace))
}

func (s *InspectionService) Pod(ctx context.Context, clusterID uint64, namespace, name string) (any, error) {
	if err := validateNamespace(clusterID, namespace); err != nil || strings.TrimSpace(name) == "" {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Pod(ctx, clusterID, strings.TrimSpace(namespace), strings.TrimSpace(name))
}
