package application

import (
	"context"
	"strings"
)

// NetworkResource identifies networking resources that need domain-specific patch semantics.
type NetworkResource string

const (
	NetworkKubernetesService NetworkResource = "service"
	NetworkIngress           NetworkResource = "ingress"
	NetworkIngressClass      NetworkResource = "ingressclass"
)

type NetworkListQuery struct {
	ClusterID uint64
	Namespace string
	SortBy    string
	Order     string
}

type NetworkReference struct {
	ClusterID uint64
	Namespace string
	Name      string
}

type ServiceEditInput struct {
	ClusterID   uint64             `json:"-"`
	Namespace   string             `json:"namespace"`
	Name        string             `json:"name"`
	Type        *string            `json:"type"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
	Selector    map[string]*string `json:"selector"`
}

type IngressEditInput struct {
	ClusterID        uint64             `json:"-"`
	Namespace        string             `json:"namespace"`
	Name             string             `json:"name"`
	IngressClassName *string            `json:"ingressClassName"`
	Labels           map[string]*string `json:"labels"`
	Annotations      map[string]*string `json:"annotations"`
}

type IngressClassEditInput struct {
	ClusterID   uint64             `json:"-"`
	Name        string             `json:"name"`
	Controller  *string            `json:"controller"`
	IsDefault   *bool              `json:"isDefault"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
}

type NetworkRuntime interface {
	List(context.Context, NetworkResource, NetworkListQuery) (any, error)
	YAML(context.Context, NetworkResource, NetworkReference) (any, error)
	Delete(context.Context, NetworkResource, NetworkReference) error
	EditService(context.Context, ServiceEditInput) error
	EditIngress(context.Context, IngressEditInput) error
	EditIngressClass(context.Context, IngressClassEditInput) error
}

type NetworkService struct{ runtime NetworkRuntime }

func NewNetworkService(runtime NetworkRuntime) *NetworkService {
	return &NetworkService{runtime: runtime}
}

func (s *NetworkService) List(ctx context.Context, resource NetworkResource, query NetworkListQuery) (any, error) {
	if !validNetworkResource(resource) || query.ClusterID == 0 {
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

func (s *NetworkService) YAML(ctx context.Context, resource NetworkResource, ref NetworkReference) (any, error) {
	if err := validateNetworkReference(resource, ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, resource, normalizeNetworkReference(ref))
}

func (s *NetworkService) Delete(ctx context.Context, resource NetworkResource, ref NetworkReference) error {
	if err := validateNetworkReference(resource, ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, resource, normalizeNetworkReference(ref))
}

func (s *NetworkService) EditService(ctx context.Context, input ServiceEditInput) error {
	if err := normalizeServiceEdit(&input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.EditService(ctx, input)
}

func (s *NetworkService) EditIngress(ctx context.Context, input IngressEditInput) error {
	if err := normalizeIngressEdit(&input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.EditIngress(ctx, input)
}

func (s *NetworkService) EditIngressClass(ctx context.Context, input IngressClassEditInput) error {
	if err := normalizeIngressClassEdit(&input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.EditIngressClass(ctx, input)
}

func validNetworkResource(resource NetworkResource) bool {
	return resource == NetworkKubernetesService || resource == NetworkIngress || resource == NetworkIngressClass
}

func networkResourceIsNamespaced(resource NetworkResource) bool {
	return resource == NetworkKubernetesService || resource == NetworkIngress
}

func validateNetworkReference(resource NetworkResource, ref NetworkReference) error {
	if !validNetworkResource(resource) || ref.ClusterID == 0 || strings.TrimSpace(ref.Name) == "" || (networkResourceIsNamespaced(resource) && strings.TrimSpace(ref.Namespace) == "") {
		return ErrInvalidParams
	}
	return nil
}

func normalizeNetworkReference(ref NetworkReference) NetworkReference {
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	ref.Name = strings.TrimSpace(ref.Name)
	return ref
}

func normalizeServiceEdit(input *ServiceEditInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	if input.Type != nil {
		value := strings.TrimSpace(*input.Type)
		input.Type = &value
	}
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizeIngressEdit(input *IngressEditInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	if input.IngressClassName != nil {
		value := strings.TrimSpace(*input.IngressClassName)
		input.IngressClassName = &value
	}
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizeIngressClassEdit(input *IngressClassEditInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Controller != nil {
		value := strings.TrimSpace(*input.Controller)
		if value == "" {
			return ErrInvalidParams
		}
		input.Controller = &value
	}
	if input.ClusterID == 0 || input.Name == "" {
		return ErrInvalidParams
	}
	return nil
}
