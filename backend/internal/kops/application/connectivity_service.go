package application

import (
	"context"
	"strings"
)

type ConnectivityResource string

const (
	ConnectivityEndpoints      ConnectivityResource = "endpoints"
	ConnectivityEndpointSlices ConnectivityResource = "endpointslices"
	ConnectivityLeases         ConnectivityResource = "leases"
)

type ConnectivityListQuery struct {
	ClusterID uint64
	Namespace string
	SortBy    string
	Order     string
}
type ConnectivityReference struct {
	ClusterID uint64
	Namespace string
	Name      string
}
type ConnectivityEdit struct {
	ClusterID uint64
	Namespace string
	YAML      string
}
type ConnectivityRuntime interface {
	List(context.Context, ConnectivityResource, ConnectivityListQuery) (any, error)
	YAML(context.Context, ConnectivityResource, ConnectivityReference) (any, error)
	Apply(context.Context, ConnectivityResource, ConnectivityEdit) error
	Delete(context.Context, ConnectivityResource, ConnectivityReference) error
}
type ConnectivityService struct{ runtime ConnectivityRuntime }

func NewConnectivityService(runtime ConnectivityRuntime) *ConnectivityService {
	return &ConnectivityService{runtime: runtime}
}
func (s *ConnectivityService) List(ctx context.Context, resource ConnectivityResource, query ConnectivityListQuery) (any, error) {
	if !validConnectivityResource(resource) || query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.Namespace = strings.TrimSpace(query.Namespace)
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.Order = strings.TrimSpace(query.Order)
	return s.runtime.List(ctx, resource, query)
}
func (s *ConnectivityService) YAML(ctx context.Context, resource ConnectivityResource, ref ConnectivityReference) (any, error) {
	if err := validateConnectivityRef(resource, ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	ref.Name = strings.TrimSpace(ref.Name)
	return s.runtime.YAML(ctx, resource, ref)
}
func (s *ConnectivityService) Apply(ctx context.Context, resource ConnectivityResource, edit ConnectivityEdit) error {
	if !validConnectivityResource(resource) || edit.ClusterID == 0 || strings.TrimSpace(edit.Namespace) == "" || strings.TrimSpace(edit.YAML) == "" {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	edit.Namespace = strings.TrimSpace(edit.Namespace)
	edit.YAML = strings.TrimSpace(edit.YAML)
	return s.runtime.Apply(ctx, resource, edit)
}
func (s *ConnectivityService) Delete(ctx context.Context, resource ConnectivityResource, ref ConnectivityReference) error {
	if err := validateConnectivityRef(resource, ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	ref.Name = strings.TrimSpace(ref.Name)
	return s.runtime.Delete(ctx, resource, ref)
}
func validConnectivityResource(resource ConnectivityResource) bool {
	return resource == ConnectivityEndpoints || resource == ConnectivityEndpointSlices || resource == ConnectivityLeases
}
func validateConnectivityRef(resource ConnectivityResource, ref ConnectivityReference) error {
	if !validConnectivityResource(resource) || ref.ClusterID == 0 || strings.TrimSpace(ref.Namespace) == "" || strings.TrimSpace(ref.Name) == "" {
		return ErrInvalidParams
	}
	return nil
}
