package kubernetes

import (
	"context"
	"errors"
	"strings"
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const permissionAuditProbeTimeout = 15 * time.Second

// KubeconfigProvider is the only cluster-registry capability required by the
// permission-audit transport. Keeping the seam this small lets the transport
// stay independent from Fleet and Kops application packages.
type KubeconfigProvider interface {
	Kubeconfig(ctx context.Context, clusterID uint64) (string, error)
}

var ErrKubeconfigProviderUnavailable = errors.New("kubeconfig provider is required")

// KubeconfigLookupError keeps a failed managed-cluster lookup distinguishable
// from an invalid credential or a Kubernetes transport failure. Runtime
// adapters can map its wrapped domain error to their public error vocabulary.
type KubeconfigLookupError struct {
	Err error
}

func (e *KubeconfigLookupError) Error() string {
	if e == nil || e.Err == nil {
		return "kubeconfig lookup failed"
	}
	return e.Err.Error()
}

func (e *KubeconfigLookupError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// PermissionAuditTransport centralizes the Kubernetes client construction
// needed by the permission-audit scanner. It is a transport adapter only: it
// does not own persistence, audit policy, or cluster naming.
type PermissionAuditTransport struct {
	provider KubeconfigProvider
	factory  ClientFactory
}

// PermissionAuditClients groups the clients built from one credential. The
// scanner owns API discovery interpretation and resource selection; this
// transport owns credential parsing and client construction.
type PermissionAuditClients struct {
	Config    *rest.Config
	Dynamic   dynamic.Interface
	Discovery discovery.DiscoveryInterface
}

// AdhocKubeconfig is the normalized credential retained by an ad-hoc audit
// together with a best-effort display name derived from its current context.
type AdhocKubeconfig struct {
	Content     string
	ClusterName string
}

// NewPermissionAuditTransport creates the normal permission-audit transport
// policy. The default factory deadline is used for scans; interactive ad-hoc
// probes use their own short deadline.
func NewPermissionAuditTransport(provider KubeconfigProvider, insecureSkipTLS bool) *PermissionAuditTransport {
	return NewPermissionAuditTransportWithFactory(provider, NewClientFactory(0, insecureSkipTLS))
}

// NewPermissionAuditTransportWithFactory exists for explicit transport policy
// wiring and focused tests. The supplied factory is never mutated.
func NewPermissionAuditTransportWithFactory(provider KubeconfigProvider, factory ClientFactory) *PermissionAuditTransport {
	return &PermissionAuditTransport{provider: provider, factory: factory}
}

// ManagedClients builds clients for a credential supplied by the caller's
// cluster registry through the narrow KubeconfigProvider port.
func (t *PermissionAuditTransport) ManagedClients(ctx context.Context, clusterID uint64) (*PermissionAuditClients, error) {
	if t == nil || t.provider == nil {
		return nil, ErrKubeconfigProviderUnavailable
	}
	kubeconfig, err := t.provider.Kubeconfig(ctx, clusterID)
	if err != nil {
		return nil, &KubeconfigLookupError{Err: err}
	}
	return t.clientsForKubeconfig(kubeconfig)
}

// AdhocClients builds clients from an already normalized ad-hoc credential.
// Validation and normalization happen before the credential is persisted.
func (t *PermissionAuditTransport) AdhocClients(kubeconfig string) (*PermissionAuditClients, error) {
	if t == nil {
		return nil, ErrKubeconfigProviderUnavailable
	}
	return t.clientsForKubeconfig(kubeconfig)
}

func (t *PermissionAuditTransport) clientsForKubeconfig(kubeconfig string) (*PermissionAuditClients, error) {
	config, err := t.factory.RESTConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := t.factory.DynamicClient(rest.CopyConfig(config))
	if err != nil {
		return nil, err
	}
	discoveryClient, err := t.factory.DiscoveryClient(rest.CopyConfig(config))
	if err != nil {
		return nil, err
	}
	return &PermissionAuditClients{Config: config, Dynamic: dynamicClient, Discovery: discoveryClient}, nil
}

// ProbeAdhoc normalizes a pasted kubeconfig, derives its display name and
// verifies API reachability with a short client deadline before an audit row
// stores the credential.
func (t *PermissionAuditTransport) ProbeAdhoc(ctx context.Context, kubeconfig string) (AdhocKubeconfig, error) {
	if t == nil {
		return AdhocKubeconfig{}, ErrKubeconfigProviderUnavailable
	}
	if err := ctx.Err(); err != nil {
		return AdhocKubeconfig{}, err
	}
	normalized, err := NormalizeKubeconfigContent(kubeconfig)
	if err != nil {
		return AdhocKubeconfig{}, err
	}
	loaded, err := clientcmd.Load([]byte(normalized))
	if err != nil {
		return AdhocKubeconfig{}, ErrInvalidKubeconfig
	}
	result := AdhocKubeconfig{Content: normalized, ClusterName: kubeconfigClusterName(loaded)}
	config, err := t.factory.WithRequestTimeout(permissionAuditProbeTimeout).RESTConfig(normalized)
	if err != nil {
		return AdhocKubeconfig{}, err
	}
	discoveryClient, err := t.factory.WithRequestTimeout(permissionAuditProbeTimeout).DiscoveryClient(config)
	if err != nil {
		return AdhocKubeconfig{}, err
	}
	if _, err := discoveryClient.ServerVersion(); err != nil {
		return AdhocKubeconfig{}, err
	}
	return result, nil
}

func kubeconfigClusterName(loaded *clientcmdapi.Config) string {
	if loaded == nil {
		return ""
	}
	current := strings.TrimSpace(loaded.CurrentContext)
	if contextConfig, ok := loaded.Contexts[loaded.CurrentContext]; ok && contextConfig != nil {
		return firstNonEmpty(current, strings.TrimSpace(contextConfig.Cluster))
	}
	return current
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
