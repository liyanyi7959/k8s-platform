package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	provisionapp "k8s-platform-backend/internal/provisioning/application"

	"gopkg.in/yaml.v3"
)

func TestCIDRsOverlap(t *testing.T) {
	tests := []struct {
		name        string
		left, right string
		want        bool
	}{
		{name: "separate", left: "10.244.0.0/16", right: "10.96.0.0/12", want: false},
		{name: "nested", left: "10.0.0.0/8", right: "10.96.0.0/12", want: true},
		{name: "invalid", left: "bad", right: "10.96.0.0/12", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := provisionapp.CIDRsOverlap(tt.left, tt.right); got != tt.want {
				t.Fatalf("CIDRsOverlap(%q, %q) = %v, want %v", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestAnsibleYAMLFilesParse(t *testing.T) {
	root := filepath.Join("..", "..", "..", "ansible")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (filepath.Ext(path) != ".yml" && filepath.Ext(path) != ".yaml") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		decoder := yaml.NewDecoder(file)
		for {
			var document yaml.Node
			err = decoder.Decode(&document)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				t.Fatalf("invalid Ansible YAML %s: %v", path, err)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMarshalInventoryMasksSecretsAndKeepsWorkerEmpty(t *testing.T) {
	masters := []inventoryHost{{IP: "192.0.2.10", SSHPort: 22, User: "ops user", Password: "p@ss word:#value"}}
	content, err := marshalInventory(masters, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "p@ss") {
		t.Fatalf("masked inventory leaked password: %s", content)
	}
	if strings.Contains(content, "localhost") {
		t.Fatalf("empty worker group must not target localhost: %s", content)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("inventory is not valid YAML: %v", err)
	}
}

func TestSupportedLinux(t *testing.T) {
	for _, name := range []string{"Ubuntu", "Debian GNU/Linux", "Rocky Linux", "AlmaLinux", "CentOS Linux", "Red Hat Enterprise Linux", "Kylin Linux Advanced Server", "银河麒麟高级服务器操作系统"} {
		if !provisionapp.SupportedLinux(name) {
			t.Errorf("expected %q to be supported", name)
		}
	}
	if provisionapp.SupportedLinux("Windows Server") {
		t.Error("Windows must not be accepted as a deployment target")
	}
}
