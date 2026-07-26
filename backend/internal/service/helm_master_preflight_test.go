package service

import (
	"strings"
	"testing"
)

func TestHelmMasterEnsureScriptUsesVerifiedOfficialArchive(t *testing.T) {
	script := helmMasterEnsureScript()
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
	script := helmMasterInstallScript(HelmMasterInstallRequest{
		ReleaseName: "prometheus",
		Namespace:   "monitoring",
		Chart:       "prometheus-community/prometheus",
		Version:     "27.0.0",
		RepoName:    "prometheus-community",
		RepoURL:     "https://prometheus-community.github.io/helm-charts",
		ValuesYAML:  "server:\n  persistentVolume:\n    enabled: false\n",
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

func TestHelmMasterRepositoryScriptsUseNamedArguments(t *testing.T) {
	addScript := helmMasterAddRepositoryScript("bitnami", "https://charts.bitnami.com/bitnami")
	if !strings.Contains(addScript, "helm repo add 'bitnami'") || !strings.Contains(addScript, "helm repo update 'bitnami'") {
		t.Fatal("repository synchronization must target one quoted repository name")
	}
	removeScript := helmMasterDeleteRepositoryScript("bitnami")
	if !strings.Contains(removeScript, "helm repo remove 'bitnami'") {
		t.Fatal("repository removal must target one quoted repository name")
	}
}
