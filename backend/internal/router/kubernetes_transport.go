package router

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s-platform-backend/internal/fleet/domain"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

// kopsKubernetesTransport is composition-only glue between the retained
// client factory and the Kops adapter's narrow client contract.
type kopsKubernetesTransport struct{ k8s *service.K8sService }

func (t kopsKubernetesTransport) TypedClient(ctx context.Context, clusterID uint64) (*kubernetes.Clientset, error) {
	if t.k8s == nil {
		return nil, kopsapp.ErrRuntime
	}
	client, err := t.k8s.TypedClient(ctx, clusterID)
	return client, kopsTransportError(err)
}

func (t kopsKubernetesTransport) RESTConfig(ctx context.Context, clusterID uint64) (*rest.Config, error) {
	if t.k8s == nil {
		return nil, kopsapp.ErrRuntime
	}
	config, err := t.k8s.RESTConfig(ctx, clusterID)
	return config, kopsTransportError(err)
}

func (t kopsKubernetesTransport) DynamicClient(ctx context.Context, clusterID uint64) (*dynamic.DynamicClient, error) {
	if t.k8s == nil {
		return nil, kopsapp.ErrRuntime
	}
	client, err := t.k8s.DynamicClient(ctx, clusterID)
	return client, kopsTransportError(err)
}

func kopsTransportError(err error) error {
	if err == nil {
		return nil
	}
	for _, candidate := range []struct{ legacy, target error }{
		{service.ErrInvalidParams, kopsapp.ErrInvalidParams}, {service.ErrNotFound, kopsapp.ErrNotFound}, {service.ErrConflict, kopsapp.ErrConflict},
		{service.ErrK8sNetwork, kopsapp.ErrRuntimeNetwork}, {service.ErrK8sTimeout, kopsapp.ErrRuntimeTimeout},
		{service.ErrK8sUnauthorized, kopsapp.ErrRuntimeUnauthorized}, {service.ErrK8sForbidden, kopsapp.ErrRuntimeForbidden},
		{service.ErrK8sTLS, kopsapp.ErrRuntimeTLS}, {service.ErrK8s, kopsapp.ErrRuntime},
	} {
		if errors.Is(err, candidate.legacy) {
			if message, ok := service.UserMessage(err); ok {
				return kopsapp.ErrWithMessage(candidate.target, message)
			}
			return candidate.target
		}
	}
	return err
}

// fleetKubernetesTransport is composition-only glue for Fleet's health
// adapter. It maps retained transport errors into Fleet's domain vocabulary.
type fleetKubernetesTransport struct{ k8s *service.K8sService }

func (t fleetKubernetesTransport) ValidateKubeconfigFormat(ctx context.Context, value string) error {
	if t.k8s == nil {
		return domain.ErrRuntime
	}
	return fleetTransportError(t.k8s.ValidateKubeconfigFormat(ctx, value))
}

func (t fleetKubernetesTransport) TypedClient(ctx context.Context, clusterID uint64) (*kubernetes.Clientset, error) {
	if t.k8s == nil {
		return nil, domain.ErrRuntime
	}
	client, err := t.k8s.TypedClient(ctx, clusterID)
	return client, fleetTransportError(err)
}

func (t fleetKubernetesTransport) StopCaches(clusterID uint64) {
	if t.k8s != nil {
		t.k8s.StopClusterCaches(clusterID)
	}
}

func fleetTransportError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, service.ErrInvalidParams):
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	case errors.Is(err, service.ErrNotFound):
		return fmt.Errorf("%w: %v", domain.ErrNotFound, err)
	case errors.Is(err, service.ErrK8sUnauthorized):
		return domain.ErrRuntimeUnauthorized
	case errors.Is(err, service.ErrK8sForbidden):
		return domain.ErrRuntimeForbidden
	case errors.Is(err, service.ErrK8sNetwork):
		return domain.ErrRuntimeNetwork
	case errors.Is(err, service.ErrK8sTimeout):
		return domain.ErrRuntimeTimeout
	case errors.Is(err, service.ErrK8sTLS):
		return domain.ErrRuntimeTLS
	case errors.Is(err, service.ErrK8s):
		return domain.ErrRuntime
	default:
		return err
	}
}
