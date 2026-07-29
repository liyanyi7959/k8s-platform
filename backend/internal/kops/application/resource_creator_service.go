package application

import (
	"context"
	"strings"
)

type WorkloadContainer struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Command string `json:"command"`
}
type WorkloadCreateInput struct {
	Namespace  string              `json:"namespace"`
	Name       string              `json:"name"`
	Replicas   int32               `json:"replicas"`
	Containers []WorkloadContainer `json:"containers"`
	Labels     map[string]string   `json:"labels"`
}
type ServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"target_port"`
	Protocol   string `json:"protocol"`
}
type ServiceCreateInput struct {
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Selector  map[string]string `json:"selector"`
	Ports     []ServicePort     `json:"ports"`
}
type IngressPath struct {
	Path        string `json:"path"`
	PathType    string `json:"path_type"`
	ServiceName string `json:"service_name"`
	ServicePort int32  `json:"service_port"`
}
type IngressRule struct {
	Host  string        `json:"host"`
	Paths []IngressPath `json:"paths"`
}
type IngressCreateInput struct {
	Namespace     string            `json:"namespace"`
	Name          string            `json:"name"`
	IngressClass  string            `json:"ingress_class"`
	Rules         []IngressRule     `json:"rules"`
	TLSSecretName string            `json:"tls_secret_name"`
	Annotations   map[string]string `json:"annotations"`
}
type ResourceCreatorRuntime interface {
	CreateDeployment(context.Context, uint64, WorkloadCreateInput) error
	CreateStatefulSet(context.Context, uint64, WorkloadCreateInput) error
	CreateDaemonSet(context.Context, uint64, WorkloadCreateInput) error
	CreateService(context.Context, uint64, ServiceCreateInput) error
	CreateIngress(context.Context, uint64, IngressCreateInput) error
}
type ResourceCreatorService struct{ runtime ResourceCreatorRuntime }

func NewResourceCreatorService(runtime ResourceCreatorRuntime) *ResourceCreatorService {
	return &ResourceCreatorService{runtime: runtime}
}
func (s *ResourceCreatorService) CreateDeployment(ctx context.Context, id uint64, input WorkloadCreateInput) error {
	if err := validateWorkload(id, &input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.CreateDeployment(ctx, id, input)
}
func (s *ResourceCreatorService) CreateStatefulSet(ctx context.Context, id uint64, input WorkloadCreateInput) error {
	if err := validateWorkload(id, &input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.CreateStatefulSet(ctx, id, input)
}
func (s *ResourceCreatorService) CreateDaemonSet(ctx context.Context, id uint64, input WorkloadCreateInput) error {
	if err := validateWorkload(id, &input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.CreateDaemonSet(ctx, id, input)
}
func (s *ResourceCreatorService) CreateService(ctx context.Context, id uint64, input ServiceCreateInput) error {
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	if id == 0 || input.Namespace == "" || input.Name == "" || len(input.Ports) == 0 {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.CreateService(ctx, id, input)
}
func (s *ResourceCreatorService) CreateIngress(ctx context.Context, id uint64, input IngressCreateInput) error {
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	if id == 0 || input.Namespace == "" || input.Name == "" || len(input.Rules) == 0 {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.CreateIngress(ctx, id, input)
}
func validateWorkload(id uint64, input *WorkloadCreateInput) error {
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	if id == 0 || input.Namespace == "" || input.Name == "" || len(input.Containers) == 0 {
		return ErrInvalidParams
	}
	return nil
}
