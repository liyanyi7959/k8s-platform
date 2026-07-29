package application

import (
	"context"
	"strings"
)

type NodeListQuery struct {
	ClusterID uint64
	SortBy    string
	Order     string
}

type NodeReference struct {
	ClusterID uint64
	Name      string
}

type NodeDrainInput struct {
	NodeReference
	TimeoutSeconds   int
	Force            bool
	IgnoreDaemonSets bool
}

type NodeRuntime interface {
	List(context.Context, NodeListQuery) (any, error)
	Detail(context.Context, NodeReference) (any, error)
	YAML(context.Context, NodeReference) (any, error)
	Pods(context.Context, NodeReference, string, string) (any, error)
	Events(context.Context, NodeReference) (any, error)
	SetSchedulable(context.Context, NodeReference, bool) error
	Drain(context.Context, NodeDrainInput) error
	Delete(context.Context, NodeReference) error
}

type NodeService struct{ runtime NodeRuntime }

func NewNodeService(runtime NodeRuntime) *NodeService { return &NodeService{runtime: runtime} }

func (s *NodeService) List(ctx context.Context, query NodeListQuery) (any, error) {
	if query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.Order = strings.TrimSpace(query.Order)
	return s.runtime.List(ctx, query)
}

func (s *NodeService) Detail(ctx context.Context, ref NodeReference) (any, error) {
	if err := validateNodeReference(ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Detail(ctx, normalizedNodeReference(ref))
}

func (s *NodeService) YAML(ctx context.Context, ref NodeReference) (any, error) {
	if err := validateNodeReference(ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, normalizedNodeReference(ref))
}

func (s *NodeService) Pods(ctx context.Context, ref NodeReference, sortBy, order string) (any, error) {
	if err := validateNodeReference(ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Pods(ctx, normalizedNodeReference(ref), strings.TrimSpace(sortBy), strings.TrimSpace(order))
}

func (s *NodeService) Events(ctx context.Context, ref NodeReference) (any, error) {
	if err := validateNodeReference(ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Events(ctx, normalizedNodeReference(ref))
}

func (s *NodeService) SetSchedulable(ctx context.Context, ref NodeReference, unschedulable bool) error {
	if err := validateNodeReference(ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.SetSchedulable(ctx, normalizedNodeReference(ref), unschedulable)
}

func (s *NodeService) Drain(ctx context.Context, input NodeDrainInput) error {
	if err := validateNodeReference(input.NodeReference); err != nil || input.TimeoutSeconds < 0 {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	input.NodeReference = normalizedNodeReference(input.NodeReference)
	return s.runtime.Drain(ctx, input)
}

func (s *NodeService) Delete(ctx context.Context, ref NodeReference) error {
	if err := validateNodeReference(ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, normalizedNodeReference(ref))
}

func validateNodeReference(ref NodeReference) error {
	if ref.ClusterID == 0 || strings.TrimSpace(ref.Name) == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizedNodeReference(ref NodeReference) NodeReference {
	ref.Name = strings.TrimSpace(ref.Name)
	return ref
}
