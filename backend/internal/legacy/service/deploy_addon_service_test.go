package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeClusterAddons(t *testing.T) {
	addons, err := normalizeClusterAddons([]string{"metrics-server", "ingress-nginx", "metrics-server"})
	if err != nil {
		t.Fatalf("expected valid add-ons: %v", err)
	}
	if len(addons) != 2 || addons[0] != "metrics-server" || addons[1] != "ingress-nginx" {
		t.Fatalf("unexpected normalized add-ons: %#v", addons)
	}
	if _, err := normalizeClusterAddons([]string{"unsupported-addon"}); err == nil {
		t.Fatal("unsupported add-on should be rejected")
	}
}

func TestMergeClusterAddons(t *testing.T) {
	got := mergeClusterAddons([]string{"metrics-server"}, []string{"ingress-nginx", "metrics-server"})
	want := []string{"metrics-server", "ingress-nginx"}
	if len(got) != len(want) {
		t.Fatalf("merged add-on count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("merged add-ons = %#v, want %#v", got, want)
		}
	}
}

func TestSelectedInstalledAddons(t *testing.T) {
	got := selectedInstalledAddons(
		[]string{"metrics-server", "ingress-nginx"},
		true,
		[]string{"metrics-server", "local-storage", "helm"},
	)
	want := []string{"metrics-server", "helm"}
	if len(got) != len(want) {
		t.Fatalf("installed add-ons = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("installed add-ons = %#v, want %#v", got, want)
		}
	}
}

func TestTaskMetaStringSliceSupportsPersistedJSON(t *testing.T) {
	got := taskMetaStringSlice(map[string]any{"enabled_steps": []any{" install_addons ", 12}}, "enabled_steps")
	if len(got) != 1 || got[0] != "install_addons" {
		t.Fatalf("unexpected enabled steps: %#v", got)
	}
}

func TestAddonInstallStepsSeparatesHelmFromClusterAddons(t *testing.T) {
	selection := []string{"helm", "metrics-server"}
	steps := addonInstallSteps(selection)
	if len(steps) != 2 || steps[0] != "install_helm" || steps[1] != "install_addons" {
		t.Fatalf("unexpected selected playbook steps: %#v", steps)
	}
	addons := runtimeClusterAddons(selection)
	if len(addons) != 1 || addons[0] != "metrics-server" {
		t.Fatalf("unexpected persisted add-ons: %#v", addons)
	}
}

func TestNewAddonInstallTaskStepUsesHelmStepForHelmOnlySelection(t *testing.T) {
	startedAt := time.Now().UTC()
	helmStep := newAddonInstallTaskStep([]string{"helm"}, startedAt)
	if helmStep.Key != "install_helm" || helmStep.Title != "补充安装 Helm" {
		t.Fatalf("Helm-only task step = %#v", helmStep)
	}
	addonStep := newAddonInstallTaskStep([]string{"metrics-server"}, startedAt)
	if addonStep.Key != "install_addons" {
		t.Fatalf("component task step key = %q, want install_addons", addonStep.Key)
	}
}

func TestInstallHelmPlaybookEscapesHelmGoTemplate(t *testing.T) {
	playbookPath := filepath.Join("..", "..", "..", "ansible", "roles", "install_helm", "tasks", "main.yml")
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
