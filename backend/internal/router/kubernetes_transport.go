package router

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s-platform-backend/internal/fleet/domain"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

// kopsKubernetesTransport is composition-only glue between the retained
// client factory and the Kops adapter's narrow client contract.
type kopsKubernetesTransport struct{ k8s *kopsclient.K8sService }

func normalizeKubeconfigRegistryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrValidation):
		return kopsclient.ErrWithMessage(kopsclient.ErrInvalidParams, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return kopsclient.ErrNotFound
	case errors.Is(err, domain.ErrConflict):
		return kopsclient.ErrConflict
	case errors.Is(err, domain.ErrCrypto):
		return kopsclient.ErrCrypto
	default:
		return err
	}
}

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
	for _, candidate := range []struct{ transport, target error }{
		{kopsclient.ErrInvalidParams, kopsapp.ErrInvalidParams}, {kopsclient.ErrNotFound, kopsapp.ErrNotFound}, {kopsclient.ErrConflict, kopsapp.ErrConflict},
		{kopsclient.ErrK8sNetwork, kopsapp.ErrRuntimeNetwork}, {kopsclient.ErrK8sTimeout, kopsapp.ErrRuntimeTimeout},
		{kopsclient.ErrK8sUnauthorized, kopsapp.ErrRuntimeUnauthorized}, {kopsclient.ErrK8sForbidden, kopsapp.ErrRuntimeForbidden},
		{kopsclient.ErrK8sTLS, kopsapp.ErrRuntimeTLS}, {kopsclient.ErrK8s, kopsapp.ErrRuntime},
	} {
		if errors.Is(err, candidate.transport) {
			if message, ok := kopsclient.UserMessage(err); ok {
				return kopsapp.ErrWithMessage(candidate.target, message)
			}
			return candidate.target
		}
	}
	return err
}

// fleetKubernetesTransport is composition-only glue for Fleet's health
// adapter. It maps retained transport errors into Fleet's domain vocabulary.
type fleetKubernetesTransport struct{ k8s *kopsclient.K8sService }

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
	case errors.Is(err, kopsclient.ErrInvalidParams):
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	case errors.Is(err, kopsclient.ErrNotFound):
		return fmt.Errorf("%w: %v", domain.ErrNotFound, err)
	case errors.Is(err, kopsclient.ErrK8sUnauthorized):
		return domain.ErrRuntimeUnauthorized
	case errors.Is(err, kopsclient.ErrK8sForbidden):
		return domain.ErrRuntimeForbidden
	case errors.Is(err, kopsclient.ErrK8sNetwork):
		return domain.ErrRuntimeNetwork
	case errors.Is(err, kopsclient.ErrK8sTimeout):
		return domain.ErrRuntimeTimeout
	case errors.Is(err, kopsclient.ErrK8sTLS):
		return domain.ErrRuntimeTLS
	case errors.Is(err, kopsclient.ErrK8s):
		return domain.ErrRuntime
	default:
		return err
	}
}
