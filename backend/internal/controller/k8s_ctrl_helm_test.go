package controller

import "testing"

func TestValidateHelmInstallRequest(t *testing.T) {
	valid := helmInstallRequest{
		ReleaseName: "redis-cache",
		Namespace:   "platform",
		Chart:       "bitnami/redis",
		RepoName:    "bitnami",
		RepoURL:     "https://charts.bitnami.com/bitnami",
	}
	if err := validateHelmInstallRequest(&valid); err != nil {
		t.Fatalf("valid Helm install request rejected: %v", err)
	}
	if valid.Namespace != "platform" {
		t.Fatalf("namespace changed unexpectedly: %q", valid.Namespace)
	}
}

func TestValidateHelmInstallRequestRejectsUnsafeSource(t *testing.T) {
	tests := []helmInstallRequest{
		{ReleaseName: "Redis", Namespace: "default", Chart: "bitnami/redis"},
		{ReleaseName: "redis", Namespace: "default", Chart: "../redis"},
		{ReleaseName: "redis", Namespace: "default", Chart: "bitnami/redis", RepoName: "bitnami", RepoURL: "http://charts.example.test"},
		{ReleaseName: "redis", Namespace: "default", Chart: "other/redis", RepoName: "bitnami", RepoURL: "https://charts.example.test"},
	}
	for _, req := range tests {
		if err := validateHelmInstallRequest(&req); err == nil {
			t.Fatalf("unsafe Helm install request accepted: %+v", req)
		}
	}
}
