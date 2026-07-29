package kops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"

	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
)

type HelmRuntime struct {
	k8s    *service.K8sService
	nodes  *kopsruntime.NodeOperations
	master *MasterHelmRuntime
}

func NewHelmRuntime(k8s *service.K8sService, nodes *kopsruntime.NodeOperations, master *MasterHelmRuntime) *HelmRuntime {
	return &HelmRuntime{k8s: k8s, nodes: nodes, master: master}
}
func (r *HelmRuntime) Preflight(ctx context.Context, clusterID uint64) (string, any, error) {
	if r == nil || r.nodes == nil || r.master == nil {
		return "", nil, kopsapp.ErrConflict
	}
	apiOK, ready, total, version, err := r.nodes.CheckHealth(ctx, clusterID)
	if err != nil {
		return "", nil, translateKopsRuntimeError(err)
	}
	if !apiOK || total == 0 || ready == 0 {
		return "", nil, fmt.Errorf("%w: cluster unavailable", kopsapp.ErrConflict)
	}
	master, err := r.master.Ensure(ctx, clusterID)
	if err != nil {
		return "", nil, translateKopsRuntimeError(err)
	}
	return version, master, nil
}
func (r *HelmRuntime) Install(ctx context.Context, input kopsapp.HelmInstallInput) (string, error) {
	if r == nil || r.master == nil {
		return "", kopsapp.ErrConflict
	}
	value, err := r.master.Install(ctx, input.ClusterID, provisionapp.HelmMasterInstallRequest{ReleaseName: input.ReleaseName, Namespace: input.Namespace, Chart: input.Chart, Version: input.Version, RepoName: input.RepoName, RepoURL: input.RepoURL, ValuesYAML: input.ValuesYAML})
	if err != nil {
		return "", translateKopsRuntimeError(err)
	}
	return value.Output, nil
}
func (r *HelmRuntime) ListRepositories(ctx context.Context, clusterID uint64) (any, error) {
	if r == nil || r.master == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.master.ListRepositories(ctx, clusterID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return value, nil
}
func (r *HelmRuntime) AddRepository(ctx context.Context, input kopsapp.HelmRepositoryInput) error {
	if r == nil || r.master == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.master.AddRepository(ctx, input.ClusterID, input.Name, input.URL))
}
func (r *HelmRuntime) DeleteRepository(ctx context.Context, clusterID uint64, name string) error {
	if r == nil || r.master == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.master.DeleteRepository(ctx, clusterID, name))
}
func (r *HelmRuntime) Search(ctx context.Context, clusterID uint64, keyword string) (any, error) {
	if r == nil || r.master == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.master.SearchCharts(ctx, clusterID, keyword)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return value, nil
}
func (r *HelmRuntime) ListReleases(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	cfg, _, cleanup, err := r.config(ctx, clusterID, namespace)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	list := action.NewList(cfg)
	list.All = true
	list.AllNamespaces = namespace == ""
	values, err := list.Run()
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	out := make([]map[string]any, 0, len(values))
	for _, value := range values {
		out = append(out, helmReleaseMap(value))
	}
	return map[string]any{"list": out, "source": "helm-sdk"}, nil
}
func (r *HelmRuntime) ReleaseDetail(ctx context.Context, clusterID uint64, namespace, name string) (any, error) {
	cfg, _, cleanup, err := r.config(ctx, clusterID, namespace)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	value, err := action.NewGet(cfg).Run(name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	detail := helmReleaseMap(value)
	getValues := action.NewGetValues(cfg)
	getValues.AllValues = true
	if values, err := getValues.Run(name); err == nil {
		if encoded, err := yaml.Marshal(values); err == nil {
			detail["values_yaml"] = string(encoded)
		}
	}
	if value.Manifest != "" {
		detail["manifest"] = value.Manifest
	}
	history := action.NewHistory(cfg)
	history.Max = 256
	if values, err := history.Run(name); err == nil {
		items := make([]map[string]any, 0, len(values))
		for _, item := range values {
			entry := helmReleaseMap(item)
			if item.Info != nil {
				entry["description"] = item.Info.Description
			}
			items = append(items, entry)
		}
		detail["history"] = items
	}
	return detail, nil
}
func (r *HelmRuntime) Uninstall(ctx context.Context, clusterID uint64, namespace, name string) (string, error) {
	cfg, _, cleanup, err := r.config(ctx, clusterID, namespace)
	if err != nil {
		return "", err
	}
	defer cleanup()
	if _, err := action.NewUninstall(cfg).Run(name); err != nil {
		return "", translateKopsRuntimeError(err)
	}
	return fmt.Sprintf("Release %s/%s 已卸载", namespace, name), nil
}
func (r *HelmRuntime) config(ctx context.Context, clusterID uint64, namespace string) (*action.Configuration, *cli.EnvSettings, func(), error) {
	if r == nil || r.k8s == nil {
		return nil, nil, func() {}, kopsapp.ErrConflict
	}
	kubeconfig, err := r.k8s.GetKubeconfig(ctx, clusterID)
	if err != nil {
		return nil, nil, func() {}, translateKopsRuntimeError(err)
	}
	file, err := os.CreateTemp("", "helm-kubeconfig-*.yaml")
	if err != nil {
		return nil, nil, func() {}, err
	}
	cleanup := func() { _ = os.Remove(file.Name()) }
	if _, err := file.WriteString(kubeconfig); err != nil {
		_ = file.Close()
		cleanup()
		return nil, nil, func() {}, err
	}
	_ = file.Close()
	repoDir := filepath.Join("data", "helm-repos", fmt.Sprintf("%d", clusterID))
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		cleanup()
		return nil, nil, func() {}, err
	}
	settings := cli.New()
	settings.KubeConfig = file.Name()
	settings.RepositoryConfig = filepath.Join(repoDir, "repositories.yaml")
	settings.RepositoryCache = filepath.Join(repoDir, "cache")
	cfg := new(action.Configuration)
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(string, ...interface{}) {}); err != nil {
		cleanup()
		return nil, nil, func() {}, err
	}
	return cfg, settings, cleanup, nil
}
func helmReleaseMap(value *release.Release) map[string]any {
	chart, appVersion, updated, status := "", "", "", ""
	if value.Chart != nil && value.Chart.Metadata != nil {
		chart = fmt.Sprintf("%s-%s", value.Chart.Metadata.Name, value.Chart.Metadata.Version)
		appVersion = value.Chart.Metadata.AppVersion
	}
	if value.Info != nil {
		status = value.Info.Status.String()
		if !value.Info.LastDeployed.IsZero() {
			updated = value.Info.LastDeployed.UTC().Format(time.RFC3339)
		}
	}
	return map[string]any{"name": value.Name, "namespace": value.Namespace, "revision": value.Version, "status": status, "chart": chart, "app_version": appVersion, "updated": updated}
}
func (r *HelmRuntime) Upgrade(ctx context.Context, input kopsapp.HelmUpgradeInput) (string, error) {
	cfg, settings, cleanup, err := r.config(ctx, input.ClusterID, input.Namespace)
	if err != nil {
		return "", err
	}
	defer cleanup()
	client := action.NewUpgrade(cfg)
	client.Namespace = input.Namespace
	client.ChartPathOptions.Version = input.Version
	client.Atomic = input.Atomic
	client.Wait = input.Wait
	if value, err := time.ParseDuration(input.Timeout); err == nil && input.Timeout != "" {
		client.Timeout = value
	}
	path, err := client.ChartPathOptions.LocateChart(input.Chart, settings)
	if err != nil {
		return "", translateKopsRuntimeError(err)
	}
	chart, err := loader.Load(path)
	if err != nil {
		return "", translateKopsRuntimeError(err)
	}
	values := map[string]any{}
	if input.ValuesYAML != "" {
		if err := yaml.Unmarshal([]byte(input.ValuesYAML), &values); err != nil {
			return "", kopsapp.ErrInvalidParams
		}
	}
	result, err := client.Run(input.Name, chart, values)
	if err != nil {
		return "", translateKopsRuntimeError(err)
	}
	return fmt.Sprintf("Release %s/%s 已升级至 revision %d", result.Namespace, result.Name, result.Version), nil
}
func (r *HelmRuntime) Rollback(ctx context.Context, input kopsapp.HelmRollbackInput) (string, error) {
	cfg, _, cleanup, err := r.config(ctx, input.ClusterID, input.Namespace)
	if err != nil {
		return "", err
	}
	defer cleanup()
	client := action.NewRollback(cfg)
	client.Version = input.Revision
	client.Wait = input.Wait
	if err := client.Run(input.Name); err != nil {
		return "", translateKopsRuntimeError(err)
	}
	return fmt.Sprintf("Release %s/%s 已回滚到 revision %d", input.Namespace, input.Name, input.Revision), nil
}
