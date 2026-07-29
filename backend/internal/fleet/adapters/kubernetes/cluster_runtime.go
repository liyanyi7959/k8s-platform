package kubernetes

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"k8s-platform-backend/internal/fleet/domain"
)

const (
	fleetMaxKubeconfigContentSize = 1024 * 1024
	fleetMaxEncodedKubeconfigSize = fleetMaxKubeconfigContentSize*4/3 + 4096
)

// ClusterTransport is the small infrastructure contract needed by Fleet's
// health runtime. It is implemented at the composition boundary so this
// adapter remains independent of retained legacy services.
type ClusterTransport interface {
	ValidateKubeconfigFormat(context.Context, string) error
	TypedClient(context.Context, uint64) (*kubernetes.Clientset, error)
	StopCaches(uint64)
}

// ClusterRuntime is Fleet's Kubernetes boundary for credential validation and
// health checks. Fleet owns policy and error vocabulary; client construction
// is supplied through the injected transport.
type ClusterRuntime struct{ transport ClusterTransport }

func NewClusterRuntime(transport ClusterTransport) *ClusterRuntime {
	return &ClusterRuntime{transport: transport}
}

func (r *ClusterRuntime) NormalizeAndValidate(ctx context.Context, value string) (string, error) {
	if r == nil || r.transport == nil {
		return "", domain.ErrRuntime
	}
	normalized, err := normalizeFleetKubeconfig(value)
	if err != nil {
		return "", err
	}
	if err := r.transport.ValidateKubeconfigFormat(ctx, normalized); err != nil {
		return "", fleetTransportError(err)
	}
	return normalized, nil
}

func (r *ClusterRuntime) CheckHealth(ctx context.Context, clusterID uint64) (bool, int, int, string, error) {
	if r == nil || r.transport == nil {
		return false, 0, 0, "", domain.ErrRuntime
	}
	client, err := r.transport.TypedClient(ctx, clusterID)
	if err != nil {
		return false, 0, 0, "", fleetTransportError(err)
	}
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return false, 0, 0, "", fleetTransportError(err)
	}
	kubernetesVersion := ""
	if version != nil {
		kubernetesVersion = version.GitVersion
	}
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return true, 0, 0, kubernetesVersion, fleetTransportError(err)
	}
	ready := 0
	for index := range nodes.Items {
		if fleetNodeReady(&nodes.Items[index]) {
			ready++
		}
	}
	return true, ready, len(nodes.Items), kubernetesVersion, nil
}

func (r *ClusterRuntime) StopCaches(clusterID uint64) {
	if r != nil && r.transport != nil {
		r.transport.StopCaches(clusterID)
	}
}

func normalizeFleetKubeconfig(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: kubeconfig cannot be empty", domain.ErrValidation)
	}
	if len(value) <= fleetMaxKubeconfigContentSize {
		if _, err := clientcmd.RESTConfigFromKubeConfig([]byte(value)); err == nil {
			return value, nil
		}
	}
	if len(value) > fleetMaxEncodedKubeconfigSize {
		return "", fmt.Errorf("%w: kubeconfig content exceeds 1MB", domain.ErrValidation)
	}
	encoded := strings.Join(strings.Fields(value), "")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil && len(decoded) <= fleetMaxKubeconfigContentSize {
		normalized := strings.TrimSpace(string(decoded))
		if _, parseErr := clientcmd.RESTConfigFromKubeConfig([]byte(normalized)); parseErr == nil {
			return normalized, nil
		}
	}
	if len(value) > fleetMaxKubeconfigContentSize {
		return "", fmt.Errorf("%w: kubeconfig content exceeds 1MB", domain.ErrValidation)
	}
	return "", fmt.Errorf("%w: invalid kubeconfig", domain.ErrValidation)
}

func fleetNodeReady(node *corev1.Node) bool {
	if node == nil {
		return false
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func fleetTransportError(err error) error {
	if err == nil {
		return nil
	}
	for _, known := range []error{
		domain.ErrValidation, domain.ErrNotFound, domain.ErrConflict,
		domain.ErrRuntimeUnauthorized, domain.ErrRuntimeForbidden, domain.ErrRuntimeNetwork,
		domain.ErrRuntimeTimeout, domain.ErrRuntimeTLS, domain.ErrRuntime,
	} {
		if errors.Is(err, known) {
			return err
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domain.ErrRuntimeTimeout
	}
	switch {
	case apierrors.IsNotFound(err):
		return domain.ErrNotFound
	case apierrors.IsInvalid(err), apierrors.IsBadRequest(err):
		return domain.ErrValidation
	case apierrors.IsUnauthorized(err):
		return domain.ErrRuntimeUnauthorized
	case apierrors.IsForbidden(err):
		return domain.ErrRuntimeForbidden
	case apierrors.IsTimeout(err):
		return domain.ErrRuntimeTimeout
	}
	var requestError *url.Error
	if errors.As(err, &requestError) && requestError != nil && requestError.Unwrap() != nil {
		err = requestError.Unwrap()
	}
	var unknownAuthority *x509.UnknownAuthorityError
	var hostnameError x509.HostnameError
	var invalidCertificate x509.CertificateInvalidError
	lower := strings.ToLower(err.Error())
	if errors.As(err, &unknownAuthority) || errors.As(err, &hostnameError) || errors.As(err, &invalidCertificate) || strings.Contains(lower, "x509:") {
		return domain.ErrRuntimeTLS
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		if networkError.Timeout() {
			return domain.ErrRuntimeTimeout
		}
		return domain.ErrRuntimeNetwork
	}
	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "no route to host") || strings.Contains(lower, "i/o timeout") || strings.Contains(lower, "timeout") {
		return domain.ErrRuntimeNetwork
	}
	return domain.ErrRuntime
}
