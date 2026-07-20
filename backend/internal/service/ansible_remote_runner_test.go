package service

import (
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

func TestRunnerBootstrapScriptIncludesRequiredDependencies(t *testing.T) {
	withPassword := runnerBootstrapScript(true)
	for _, expected := range []string{"ansible-playbook", "apt-get", "dnf", "yum", "sshpass", "python3 -m pip install ansible-core"} {
		if !strings.Contains(withPassword, expected) {
			t.Fatalf("bootstrap script is missing %q", expected)
		}
	}
	withoutPassword := runnerBootstrapScript(false)
	if strings.Contains(withoutPassword, "install -y tar gzip sshpass") {
		t.Fatal("key-only runner must not require sshpass installation")
	}
}

func TestNormalizeMasterKubeconfig(t *testing.T) {
	raw := `apiVersion: v1
kind: Config
clusters:
- name: cluster
  cluster:
    server: https://127.0.0.1:6443
contexts: []
users: []
`
	normalized, err := normalizeMasterKubeconfig(raw, "192.0.2.10")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := clientcmd.Load([]byte(normalized))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Clusters["cluster"].Server; got != "https://192.0.2.10:6443" {
		t.Fatalf("unexpected API server: %s", got)
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote("a'b"); got != `'a'"'"'b'` {
		t.Fatalf("unexpected shell quote: %s", got)
	}
}
