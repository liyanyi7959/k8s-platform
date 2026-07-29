package provisioning

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
)

func TestNormalizeClusterAddons(t *testing.T) {
	addons, err := normalizeClusterAddons([]string{"metrics-server", "ingress-nginx", "metrics-server"})
	if err != nil {
		t.Fatalf("expected valid add-ons: %v", err)
	}
	if got, want := strings.Join(addons, ","), "metrics-server,ingress-nginx"; got != want {
		t.Fatalf("normalized add-ons = %q, want %q", got, want)
	}
	if _, err := normalizeClusterAddons([]string{"unsupported-addon"}); !errors.Is(err, provisionapp.ErrInvalidParams) {
		t.Fatalf("unsupported add-on error = %v, want invalid params", err)
	}
}

func TestMergeClusterAddons(t *testing.T) {
	got := mergeClusterAddons([]string{"metrics-server"}, []string{"ingress-nginx", "metrics-server"})
	if want := []string{"metrics-server", "ingress-nginx"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("merged add-ons = %#v, want %#v", got, want)
	}
}

func TestSelectedInstalledAddons(t *testing.T) {
	got := selectedInstalledAddons([]string{"metrics-server", "ingress-nginx"}, true, []string{"metrics-server", "local-storage", "helm"})
	if want := []string{"metrics-server", "helm"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("installed add-ons = %#v, want %#v", got, want)
	}
}

func TestTaskMetaStringSliceSupportsPersistedJSON(t *testing.T) {
	got := taskMetaStringSlice(map[string]any{"enabled_steps": []any{" install_addons ", 12}}, "enabled_steps")
	if len(got) != 1 || got[0] != "install_addons" {
		t.Fatalf("unexpected enabled steps: %#v", got)
	}
}

func TestAddonInstallStepsSeparatesHelmFromClusterAddons(t *testing.T) {
	steps := addonInstallSteps([]string{"helm", "metrics-server"})
	if want := []string{"install_helm", "install_addons"}; strings.Join(steps, ",") != strings.Join(want, ",") {
		t.Fatalf("selected playbook steps = %#v, want %#v", steps, want)
	}
	addons := runtimeClusterAddons([]string{"helm", "metrics-server"})
	if len(addons) != 1 || addons[0] != "metrics-server" {
		t.Fatalf("persisted add-ons = %#v", addons)
	}
}

func TestNewAddonInstallTaskStepUsesHelmStepForHelmOnlySelection(t *testing.T) {
	startedAt := time.Now().UTC()
	if step := newAddonInstallTaskStep([]string{"helm"}, startedAt); step.Key != "install_helm" || step.Title != "补充安装 Helm" {
		t.Fatalf("Helm-only task step = %#v", step)
	}
	if step := newAddonInstallTaskStep([]string{"metrics-server"}, startedAt); step.Key != "install_addons" {
		t.Fatalf("component task step key = %q, want install_addons", step.Key)
	}
}

func TestInstallHelmPlaybookEscapesHelmGoTemplate(t *testing.T) {
	playbookPath := filepath.Join(integrationAnsibleRoot(t), "roles", "install_helm", "tasks", "main.yml")
	content, err := os.ReadFile(playbookPath)
	if err != nil {
		t.Fatalf("read Helm playbook: %v", err)
	}
	if strings.Contains(string(content), "'{{ .Version }}'") {
		t.Fatal("Helm Go template must be escaped before Ansible renders the playbook")
	}
	if count := strings.Count(string(content), "'{{ \"{{\" }} .Version {{ \"}}\" }}'"); count != 2 {
		t.Fatalf("escaped Helm version template count = %d, want 2", count)
	}
}
