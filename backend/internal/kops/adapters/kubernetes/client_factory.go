// Package kubernetes adapts kubeconfig credentials to Kubernetes API clients.
// It intentionally contains no application or persistence dependencies so the
// same client construction rules can be shared by Kops resource operations,
// diagnostics and future transport adapters.
package kubernetes

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	maxKubeconfigContentSize = 1024 * 1024
	maxEncodedKubeconfigSize = maxKubeconfigContentSize*4/3 + 4096
	defaultRequestTimeout    = 60 * time.Second
)

var (
	ErrKubeconfigEmpty    = errors.New("kubeconfig cannot be empty")
	ErrKubeconfigTooLarge = errors.New("kubeconfig content exceeds 1MB")
	ErrInvalidKubeconfig  = errors.New("invalid kubeconfig")
)

// NormalizeKubeconfigContent accepts a regular kubeconfig or a kubeconfig
// whose whole content was Base64 encoded, returning canonical plaintext.
func NormalizeKubeconfigContent(kubeconfig string) (string, error) {
	kubeconfig = strings.TrimSpace(kubeconfig)
	if kubeconfig == "" {
		return "", ErrKubeconfigEmpty
	}
	if len(kubeconfig) <= maxKubeconfigContentSize {
		if _, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig)); err == nil {
			return kubeconfig, nil
		}
	}
	if len(kubeconfig) > maxEncodedKubeconfigSize {
		return "", ErrKubeconfigTooLarge
	}

	encoded := strings.Join(strings.Fields(kubeconfig), "")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil && len(decoded) <= maxKubeconfigContentSize {
		decodedKubeconfig := strings.TrimSpace(string(decoded))
		if _, parseErr := clientcmd.RESTConfigFromKubeConfig([]byte(decodedKubeconfig)); parseErr == nil {
			return decodedKubeconfig, nil
		}
	}
	if len(kubeconfig) > maxKubeconfigContentSize {
		return "", ErrKubeconfigTooLarge
	}
	return "", ErrInvalidKubeconfig
}

// ClientFactory centralizes Kubernetes transport construction and transport
// policy. Kops callers should not configure TLS or request deadlines ad hoc.
type ClientFactory struct {
	requestTimeout time.Duration
	insecureTLS    bool
}

func NewClientFactory(requestTimeout time.Duration, insecureSkipTLS bool) ClientFactory {
	if requestTimeout <= 0 {
		requestTimeout = defaultRequestTimeout
	}
	return ClientFactory{requestTimeout: requestTimeout, insecureTLS: insecureSkipTLS}
}

// WithRequestTimeout derives a factory for a bounded probe without changing
// its TLS policy. It is useful for interactive credential validation while
// background resource operations retain their normal deadline.
func (f ClientFactory) WithRequestTimeout(requestTimeout time.Duration) ClientFactory {
	return NewClientFactory(requestTimeout, f.insecureTLS)
}

// RESTConfig parses a persisted, already-normalized kubeconfig and applies
// the Kops transport policy.
func (f ClientFactory) RESTConfig(kubeconfig string) (*rest.Config, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(strings.TrimSpace(kubeconfig)))
	if err != nil {
		return nil, ErrInvalidKubeconfig
	}
	config.Timeout = f.timeout()
	if f.insecureTLS {
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAData = nil
		config.TLSClientConfig.CAFile = ""
	}
	return config, nil
}

func (f ClientFactory) DynamicClient(config *rest.Config) (*dynamic.DynamicClient, error) {
	return dynamic.NewForConfig(config)
}

func (f ClientFactory) DiscoveryClient(config *rest.Config) (*discovery.DiscoveryClient, error) {
	return discovery.NewDiscoveryClientForConfig(config)
}

func (f ClientFactory) TypedClient(config *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

// InformerClient returns a typed client with no request deadline because list
// and watch streams used by shared informers must remain long-lived.
func (f ClientFactory) InformerClient(config *rest.Config) (*kubernetes.Clientset, error) {
	watchConfig := rest.CopyConfig(config)
	watchConfig.Timeout = 0
	return kubernetes.NewForConfig(watchConfig)
}

// Validate reaches the Kubernetes discovery endpoint after parsing a user
// supplied kubeconfig. Context cancellation is checked before the request;
// the transport deadline is still enforced by the configured REST client.
func (f ClientFactory) Validate(ctx context.Context, kubeconfig string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := NormalizeKubeconfigContent(kubeconfig)
	if err != nil {
		return err
	}
	config, err := f.RESTConfig(normalized)
	if err != nil {
		return err
	}
	client, err := f.TypedClient(config)
	if err != nil {
		return err
	}
	_, err = client.Discovery().ServerVersion()
	return err
}

func (f ClientFactory) timeout() time.Duration {
	if f.requestTimeout <= 0 {
		return defaultRequestTimeout
	}
	return f.requestTimeout
}
