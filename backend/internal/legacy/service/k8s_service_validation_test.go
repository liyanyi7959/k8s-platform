package service

import (
	"encoding/base64"
	"testing"
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
	if _, err := NormalizeKubeconfigContent("not-a-kubeconfig"); err == nil {
		t.Fatal("expected invalid kubeconfig error")
	}
}
