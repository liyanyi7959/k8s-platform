package http

import "testing"

func TestAppTemplateRequestNormalizesHelmCatalogEntry(t *testing.T) {
	req := appTemplateReq{
		Name:         "  redis  ",
		DeployType:   "HELM",
		Template:     " bitnami/redis ",
		HelmRepoName: " bitnami ",
		HelmRepoURL:  " https://charts.bitnami.com/bitnami ",
	}
	if err := req.normalizeAndValidate(); err != nil {
		t.Fatalf("valid Helm catalog entry rejected: %v", err)
	}
	if req.Name != "redis" || req.DeployType != "helm" || req.Template != "bitnami/redis" || req.HelmRepoName != "bitnami" {
		t.Fatalf("request fields were not normalized: %+v", req)
	}
}

func TestAppTemplateRequestRejectsIncompleteHelmRepository(t *testing.T) {
	for _, req := range []appTemplateReq{
		{DeployType: "helm", Template: "bitnami/redis", HelmRepoName: "bitnami"},
		{DeployType: "helm", Template: "bitnami/redis", HelmRepoURL: "https://charts.bitnami.com/bitnami"},
		{DeployType: "helm"},
		{DeployType: "binary", Template: "redis"},
		{DeployType: "helm", Template: "bitnami/redis", HelmRepoName: "bitnami", HelmRepoURL: "http://charts.bitnami.com/bitnami"},
		{DeployType: "helm", Template: "other/redis", HelmRepoName: "bitnami", HelmRepoURL: "https://charts.bitnami.com/bitnami"},
		{DeployType: "helm", Template: "bitnami/redis", HelmValuesYAML: "key: ["},
	} {
		if err := req.normalizeAndValidate(); err == nil {
			t.Fatalf("invalid application catalog request accepted: %+v", req)
		}
	}
}
