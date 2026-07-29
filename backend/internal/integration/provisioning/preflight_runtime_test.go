package provisioning

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	provisionapp "k8s-platform-backend/internal/provisioning/application"

	"gopkg.in/yaml.v3"
)

func TestPreflightRuntimeRequiresPersistence(t *testing.T) {
	runtime := NewPreflightRuntime(nil, "")
	if _, err := runtime.Preflight(context.Background(), 1); !errors.Is(err, provisionapp.ErrConflict) {
		t.Fatalf("Preflight without database error = %v, want conflict", err)
	}
	if err := runtime.SetPreflightIgnore(context.Background(), 1, "node.1.disk", true); !errors.Is(err, provisionapp.ErrConflict) {
		t.Fatalf("SetPreflightIgnore without database error = %v, want conflict", err)
	}
}

func TestCIDRsOverlap(t *testing.T) {
	for _, test := range []struct {
		name, left, right string
		want              bool
	}{
		{name: "separate", left: "10.244.0.0/16", right: "10.96.0.0/12", want: false},
		{name: "nested", left: "10.0.0.0/8", right: "10.96.0.0/12", want: true},
		{name: "invalid", left: "bad", right: "10.96.0.0/12", want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := provisionapp.CIDRsOverlap(test.left, test.right); got != test.want {
				t.Fatalf("CIDRsOverlap(%q, %q) = %v, want %v", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestAnsibleYAMLFilesParse(t *testing.T) {
	root := integrationAnsibleRoot(t)
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

func integrationAnsibleRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration provisioning test directory")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "ansible"))
}

func TestMarshalInventoryMasksSecretsAndKeepsWorkerEmpty(t *testing.T) {
	masters := []provisionapp.InventoryHost{{IP: "192.0.2.10", SSHPort: 22, User: "ops user", Password: "p@ss word:#value"}}
	content, err := provisionapp.MarshalAnsibleInventory(masters, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "p@ss") || strings.Contains(content, "localhost") {
		t.Fatalf("masked inventory leaked secret or targeted localhost: %s", content)
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
