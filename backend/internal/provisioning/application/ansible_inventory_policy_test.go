package application

import (
	"strings"
	"testing"
)

func TestInventoryNodeAlias(t *testing.T) {
	if got := InventoryNodeAlias("master", 1, "192.0.2.10"); got != "master01-192-0-2-10" {
		t.Fatalf("master alias=%q", got)
	}
	if got := InventoryNodeAlias("worker", 12, "192.0.2.11"); got != "worker12-192-0-2-11" {
		t.Fatalf("worker alias=%q", got)
	}
}

func TestKubernetesMinorVersion(t *testing.T) {
	for input, want := range map[string]string{"v1.31.4": "1.31", "1.30.0": "1.30", "v1": "1"} {
		if got := KubernetesMinorVersion(input); got != want {
			t.Fatalf("KubernetesMinorVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMarshalAnsibleInventoryMasksSecrets(t *testing.T) {
	content, err := MarshalAnsibleInventory(
		[]InventoryHost{{Alias: "master01", IP: "192.0.2.10", SSHPort: 22, User: "ops", Password: "super-secret"}},
		[]InventoryHost{{Alias: "worker01", IP: "192.0.2.11", SSHPort: 2222, User: "root", KeyFile: "C:/tmp/private.key"}},
		true,
	)
	if err != nil {
		t.Fatalf("marshal inventory: %v", err)
	}
	for _, expected := range []string{"master01:", "worker01:", "ansible_port: 2222", "ansible_password: '***'", "ansible_ssh_private_key_file: <temporary-private-key>"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("inventory missing %q:\n%s", expected, content)
		}
	}
	if strings.Contains(content, "super-secret") || strings.Contains(content, "C:/tmp/private.key") {
		t.Fatalf("inventory leaked a secret:\n%s", content)
	}
}
