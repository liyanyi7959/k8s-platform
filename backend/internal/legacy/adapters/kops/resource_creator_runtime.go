package kops

import (
	"context"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

type ResourceCreatorRuntime struct{ service *service.K8sService }

func NewResourceCreatorRuntime(service *service.K8sService) *ResourceCreatorRuntime {
	return &ResourceCreatorRuntime{service: service}
}
func (r *ResourceCreatorRuntime) CreateDeployment(ctx context.Context, id uint64, input kopsapp.WorkloadCreateInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.CreateDeployment(ctx, id, workloadInput(input)))
}
func (r *ResourceCreatorRuntime) CreateStatefulSet(ctx context.Context, id uint64, input kopsapp.WorkloadCreateInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.CreateStatefulSet(ctx, id, workloadInput(input)))
}
func (r *ResourceCreatorRuntime) CreateDaemonSet(ctx context.Context, id uint64, input kopsapp.WorkloadCreateInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.CreateDaemonSet(ctx, id, workloadInput(input)))
}
func (r *ResourceCreatorRuntime) CreateService(ctx context.Context, id uint64, input kopsapp.ServiceCreateInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	ports := make([]service.CreateServicePort, 0, len(input.Ports))
	for _, v := range input.Ports {
		ports = append(ports, service.CreateServicePort{Name: v.Name, Port: v.Port, TargetPort: v.TargetPort, Protocol: v.Protocol})
	}
	return translateKopsRuntimeError(r.service.CreateService(ctx, id, service.CreateServiceInput{Namespace: input.Namespace, Name: input.Name, Type: input.Type, Selector: input.Selector, Ports: ports}))
}
func (r *ResourceCreatorRuntime) CreateIngress(ctx context.Context, id uint64, input kopsapp.IngressCreateInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	rules := make([]service.CreateIngressRule, 0, len(input.Rules))
	for _, rule := range input.Rules {
		paths := make([]service.CreateIngressPath, 0, len(rule.Paths))
		for _, path := range rule.Paths {
			paths = append(paths, service.CreateIngressPath{Path: path.Path, PathType: path.PathType, ServiceName: path.ServiceName, ServicePort: path.ServicePort})
		}
		rules = append(rules, service.CreateIngressRule{Host: rule.Host, Paths: paths})
	}
	return translateKopsRuntimeError(r.service.CreateIngress(ctx, id, service.CreateIngressInput{Namespace: input.Namespace, Name: input.Name, IngressClass: input.IngressClass, Rules: rules, TLSSecretName: input.TLSSecretName, Annotations: input.Annotations}))
}
func workloadInput(input kopsapp.WorkloadCreateInput) service.CreateDeploymentInput {
	items := make([]service.CreateDeploymentContainer, 0, len(input.Containers))
	for _, v := range input.Containers {
		items = append(items, service.CreateDeploymentContainer{Name: v.Name, Image: v.Image, CPU: v.CPU, Memory: v.Memory, Command: v.Command})
	}
	return service.CreateDeploymentInput{Namespace: input.Namespace, Name: input.Name, Replicas: input.Replicas, Containers: items, Labels: input.Labels}
}
