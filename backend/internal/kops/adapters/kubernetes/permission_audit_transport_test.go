package kubernetes

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type staticKubeconfigProvider struct {
	value     string
	err       error
	clusterID uint64
}

func (p *staticKubeconfigProvider) Kubeconfig(_ context.Context, clusterID uint64) (string, error) {
	p.clusterID = clusterID
	return p.value, p.err
}

func TestPermissionAuditTransportBuildsManagedClientsFromNarrowProvider(t *testing.T) {
	provider := &staticKubeconfigProvider{value: validKubeconfig}
	transport := NewPermissionAuditTransportWithFactory(provider, NewClientFactory(45*time.Second, true))

	clients, err := transport.ManagedClients(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if provider.clusterID != 42 {
		t.Fatalf("provider cluster ID = %d, want 42", provider.clusterID)
	}
	if clients.Config == nil || clients.Config.Timeout != 45*time.Second {
		t.Fatalf("REST config timeout = %#v, want 45s", clients.Config)
	}
	if !clients.Config.TLSClientConfig.Insecure {
		t.Fatal("managed clients did not retain transport TLS policy")
	}
	if clients.Dynamic == nil || clients.Discovery == nil {
		t.Fatalf("clients = %#v, want dynamic and discovery clients", clients)
	}
}

func TestPermissionAuditTransportWrapsProviderFailures(t *testing.T) {
	want := errors.New("cluster credential is unavailable")
	transport := NewPermissionAuditTransport(&staticKubeconfigProvider{err: want}, false)

	_, err := transport.ManagedClients(context.Background(), 9)
	var lookup *KubeconfigLookupError
	if !errors.As(err, &lookup) {
		t.Fatalf("error = %v, want KubeconfigLookupError", err)
	}
	if !errors.Is(err, want) {
		t.Fatalf("wrapped error = %v, want %v", err, want)
	}
}

func TestPermissionAuditTransportProbesNormalizedAdhocKubeconfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"major":"1","minor":"30","gitVersion":"v1.30.0"}`))
	}))
	defer server.Close()

	kubeconfig := permissionAuditTestKubeconfig(server.URL)
	encoded := base64.StdEncoding.EncodeToString([]byte(kubeconfig))
	transport := NewPermissionAuditTransportWithFactory(nil, NewClientFactory(time.Minute, false))

	result, err := transport.ProbeAdhoc(context.Background(), encoded)
	if err != nil {
		t.Fatal(err)
	}
	if result.ClusterName != "demo" {
		t.Fatalf("cluster name = %q, want demo", result.ClusterName)
	}
	if result.Content != strings.TrimSpace(kubeconfig) {
		t.Fatalf("normalized kubeconfig = %q, want plaintext input", result.Content)
	}
	config, err := transport.factory.WithRequestTimeout(permissionAuditProbeTimeout).RESTConfig(result.Content)
	if err != nil {
		t.Fatal(err)
	}
	if config.Timeout != permissionAuditProbeTimeout {
		t.Fatalf("probe timeout = %s, want %s", config.Timeout, permissionAuditProbeTimeout)
	}
}

func TestPermissionAuditTransportRejectsCanceledAdhocProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	transport := NewPermissionAuditTransport(nil, false)

	_, err := transport.ProbeAdhoc(ctx, validKubeconfig)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("probe error = %v, want context cancellation", err)
	}
}

func permissionAuditTestKubeconfig(server string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: demo
  cluster:
    server: %s
contexts:
- name: demo
  context:
    cluster: demo
    user: demo
current-context: demo
users:
- name: demo
  user:
    token: token
`, server)
}
