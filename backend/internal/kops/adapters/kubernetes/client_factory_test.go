package kubernetes

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

const validKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: demo
  cluster:
    server: https://127.0.0.1:6443
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
`

func TestNormalizeKubeconfigContent(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(validKubeconfig))
	got, err := NormalizeKubeconfigContent(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got != validKubeconfig[:len(validKubeconfig)-1] {
		t.Fatalf("unexpected normalized kubeconfig: %q", got)
	}
}

func TestNormalizeKubeconfigContentRejectsInvalidInput(t *testing.T) {
	if _, err := NormalizeKubeconfigContent("not-a-kubeconfig"); !errors.Is(err, ErrInvalidKubeconfig) {
		t.Fatalf("expected invalid kubeconfig error, got %v", err)
	}
}

func TestClientFactoryAppliesTransportPolicy(t *testing.T) {
	factory := NewClientFactory(15*time.Second, true)
	config, err := factory.RESTConfig(validKubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Timeout != 15*time.Second {
		t.Fatalf("timeout = %s, want 15s", config.Timeout)
	}
	if !config.TLSClientConfig.Insecure || len(config.TLSClientConfig.CAData) != 0 || config.TLSClientConfig.CAFile != "" {
		t.Fatalf("unexpected TLS policy: %#v", config.TLSClientConfig)
	}
}

func TestClientFactoryWithRequestTimeoutPreservesTLSPolicy(t *testing.T) {
	factory := NewClientFactory(time.Minute, true).WithRequestTimeout(5 * time.Second)
	config, err := factory.RESTConfig(validKubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Timeout != 5*time.Second || !config.TLSClientConfig.Insecure {
		t.Fatalf("derived policy = timeout %s, insecure %t", config.Timeout, config.TLSClientConfig.Insecure)
	}
}
