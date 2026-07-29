package application

import (
	"errors"
	"strings"
	"testing"
)

func TestHelmMasterEnsureScriptUsesVerifiedOfficialArchive(t *testing.T) {
	script := HelmMasterEnsureScript()
	for _, expected := range []string{
		"https://get.helm.sh/",
		"sha256sum -c -",
		"HELM_VERSION=",
		"helm version --template",
		"HELM_INSTALLED_NOW=",
		"/usr/local/bin/helm",
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("Helm bootstrap script is missing %q", expected)
		}
	}
}

func TestHelmMasterInstallScriptRunsAtomicInstallOnMaster(t *testing.T) {
	script := HelmMasterInstallScript(HelmMasterInstallRequest{
		ReleaseName: "prometheus", Namespace: "monitoring", Chart: "prometheus-community/prometheus",
		Version: "27.0.0", RepoName: "prometheus-community", RepoURL: "https://prometheus-community.github.io/helm-charts",
		ValuesYAML: "server:\n  persistentVolume:\n    enabled: false\n",
	})
	for _, expected := range []string{
		"export KUBECONFIG=/etc/kubernetes/admin.conf",
		"for attempt in 1 2 3",
		"AIOPS_REPOSITORY_UNAVAILABLE",
		"helm upgrade --install 'prometheus' 'prometheus-community/prometheus'",
		"--atomic --wait --timeout 10m",
		"base64 -d > \"$values_file\"",
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("Helm installation script is missing %q", expected)
		}
	}
}

func TestHelmMasterOutputPoliciesValidateAndFilter(t *testing.T) {
	preflight, err := ParseHelmMasterPreflight(7, "master-1", "10.0.0.1", "noise\nHELM_VERSION=v3.16.4\nHELM_INSTALLED_NOW=true\n")
	if err != nil || preflight.HelmVersion != HelmMasterVersionToInstall || !preflight.InstalledNow {
		t.Fatalf("ParseHelmMasterPreflight() = %#v, %v", preflight, err)
	}
	if _, err := ParseHelmMasterPreflight(7, "master-1", "10.0.0.1", "HELM_VERSION=v3.15.0"); !errors.Is(err, ErrConflict) {
		t.Fatalf("unsupported Helm version error = %v", err)
	}

	items, err := ParseHelmMasterChartSearch(`[
		{"name":"bitnami/redis","version":"20.0.0","app_version":"7","description":"In-memory store"},
		{"name":"invalid","version":"1.0.0","description":"redis-like"}
	]`, "redis")
	if err != nil || len(items) != 1 || items[0].RepoName != "bitnami" || items[0].ChartName != "redis" {
		t.Fatalf("ParseHelmMasterChartSearch() = %#v, %v", items, err)
	}
	if _, err := ParseHelmMasterRepositories("not-json"); !errors.Is(err, ErrConflict) {
		t.Fatalf("invalid repository payload error = %v", err)
	}
}

func TestHelmMasterRepositoryScriptsQuoteNamedArguments(t *testing.T) {
	addScript := HelmMasterAddRepositoryScript("bitnami", "https://charts.bitnami.com/bitnami")
	if !strings.Contains(addScript, "helm repo add 'bitnami'") || !strings.Contains(addScript, "helm repo update 'bitnami'") {
		t.Fatal("repository synchronization must target one quoted repository name")
	}
	removeScript := HelmMasterDeleteRepositoryScript("bitnami")
	if !strings.Contains(removeScript, "helm repo remove 'bitnami'") {
		t.Fatal("repository removal must target one quoted repository name")
	}
}
