package application

import (
	"context"
	"errors"
	"testing"
)

type networkRuntimeSpy struct{ service ServiceEditInput }

func (*networkRuntimeSpy) List(context.Context, NetworkResource, NetworkListQuery) (any, error) {
	return nil, nil
}
func (*networkRuntimeSpy) YAML(context.Context, NetworkResource, NetworkReference) (any, error) {
	return nil, nil
}
func (*networkRuntimeSpy) Delete(context.Context, NetworkResource, NetworkReference) error {
	return nil
}
func (spy *networkRuntimeSpy) EditService(_ context.Context, value ServiceEditInput) error {
	spy.service = value
	return nil
}
func (*networkRuntimeSpy) EditIngress(context.Context, IngressEditInput) error           { return nil }
func (*networkRuntimeSpy) EditIngressClass(context.Context, IngressClassEditInput) error { return nil }

func TestNetworkServiceNormalizesServiceEdits(t *testing.T) {
	spy := &networkRuntimeSpy{}
	service := NewNetworkService(spy)
	serviceType := " LoadBalancer "
	if err := service.EditService(context.Background(), ServiceEditInput{ClusterID: 2, Namespace: " ops ", Name: " api ", Type: &serviceType}); err != nil {
		t.Fatalf("EditService() error=%v", err)
	}
	if spy.service.Namespace != "ops" || spy.service.Name != "api" || spy.service.Type == nil || *spy.service.Type != "LoadBalancer" {
		t.Fatalf("service=%#v", spy.service)
	}
	if _, err := service.YAML(context.Background(), NetworkIngressClass, NetworkReference{ClusterID: 2, Name: " "}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("YAML invalid reference error=%v", err)
	}
	blank := " "
	if err := service.EditIngressClass(context.Background(), IngressClassEditInput{ClusterID: 2, Name: "public", Controller: &blank}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("EditIngressClass invalid controller error=%v", err)
	}
}
