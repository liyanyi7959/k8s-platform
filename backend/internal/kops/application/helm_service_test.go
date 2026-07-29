package application

import (
	"context"
	"errors"
	"testing"
)

type helmRuntimeSpy struct{ install HelmInstallInput }

func (*helmRuntimeSpy) Preflight(context.Context, uint64) (string, any, error) {
	return "v1.30.0", nil, nil
}
func (spy *helmRuntimeSpy) Install(_ context.Context, input HelmInstallInput) (string, error) {
	spy.install = input
	return "done", nil
}
func (*helmRuntimeSpy) ListRepositories(context.Context, uint64) (any, error)     { return nil, nil }
func (*helmRuntimeSpy) AddRepository(context.Context, HelmRepositoryInput) error  { return nil }
func (*helmRuntimeSpy) DeleteRepository(context.Context, uint64, string) error    { return nil }
func (*helmRuntimeSpy) Search(context.Context, uint64, string) (any, error)       { return nil, nil }
func (*helmRuntimeSpy) ListReleases(context.Context, uint64, string) (any, error) { return nil, nil }
func (*helmRuntimeSpy) ReleaseDetail(context.Context, uint64, string, string) (any, error) {
	return nil, nil
}
func (*helmRuntimeSpy) Uninstall(context.Context, uint64, string, string) (string, error) {
	return "", nil
}
func (*helmRuntimeSpy) Upgrade(context.Context, HelmUpgradeInput) (string, error)   { return "", nil }
func (*helmRuntimeSpy) Rollback(context.Context, HelmRollbackInput) (string, error) { return "", nil }

func TestHelmServiceNormalizesAndRejectsUnsafeInstallSources(t *testing.T) {
	spy := &helmRuntimeSpy{}
	service := NewHelmService(spy)
	result, err := service.Install(context.Background(), HelmInstallInput{ClusterID: 2, ReleaseName: "redis", Chart: "bitnami/redis", RepoName: "bitnami", RepoURL: "https://charts.bitnami.com/bitnami", ValuesYAML: "replicaCount: 1"})
	if err != nil || result == nil || spy.install.Namespace != "default" {
		t.Fatalf("Install() result=%#v input=%#v err=%v", result, spy.install, err)
	}
	if _, err := service.Install(context.Background(), HelmInstallInput{ClusterID: 2, ReleaseName: "Redis", Namespace: "default", Chart: "bitnami/redis"}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid name error=%v", err)
	}
	if _, err := service.Install(context.Background(), HelmInstallInput{ClusterID: 2, ReleaseName: "redis", Namespace: "default", Chart: "../redis"}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("unsafe chart error=%v", err)
	}
}
