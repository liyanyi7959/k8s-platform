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
