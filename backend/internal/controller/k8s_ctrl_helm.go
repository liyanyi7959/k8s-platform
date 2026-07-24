package controller

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

	"k8s-platform-backend/pkg/resp"
)

const helmInstallTimeout = 10 * time.Minute

var helmKubernetesName = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

type helmInstallRequest struct {
	ReleaseName string `json:"release_name"`
	Namespace   string `json:"namespace"`
	Chart       string `json:"chart"`
	Version     string `json:"version"`
	RepoURL     string `json:"repo_url"`
	RepoName    string `json:"repo_name"`
	ValuesYAML  string `json:"values_yaml"`
}

// debugLog 用于抑制 Helm SDK 内部调试日志。
func debugLog(format string, v ...interface{}) {}

// helmRepoDir 返回当前集群独立的 Helm 仓库配置目录。
func (kc *K8sController) helmRepoDir(c *gin.Context) (string, error) {
	id, ok := parseClusterID(c)
	if !ok {
		return "", fmt.Errorf("invalid cluster id")
	}
	dir := filepath.Join("data", "helm-repos", fmt.Sprintf("%d", id))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建 Helm 仓库目录失败: %w", err)
	}
	return dir, nil
}

// newHelmConfig 使用集群 kubeconfig 初始化 Helm SDK action.Configuration。
// namespace 为空字符串表示跨所有命名空间（list --all-namespaces）。
// 返回的 settings 已配置集群独立的仓库路径，cleanup 负责清理临时 kubeconfig 文件。
func (kc *K8sController) newHelmConfig(c *gin.Context, namespace string) (*action.Configuration, *cli.EnvSettings, func(), error) {
	id, ok := parseClusterID(c)
	if !ok {
		return nil, nil, func() {}, fmt.Errorf("invalid cluster id")
	}
	kubeconfig, err := kc.svc.GetKubeconfig(c.Request.Context(), id)
	if err != nil {
		return nil, nil, func() {}, err
	}

	f, err := os.CreateTemp("", "helm-kubeconfig-*.yaml")
	if err != nil {
		return nil, nil, func() {}, fmt.Errorf("创建临时 kubeconfig 失败: %w", err)
	}
	cleanup := func() { os.Remove(f.Name()) }
	if _, err := f.WriteString(kubeconfig); err != nil {
		f.Close()
		cleanup()
		return nil, nil, func() {}, fmt.Errorf("写入 kubeconfig 失败: %w", err)
	}
	f.Close()

	repoDir, err := kc.helmRepoDir(c)
	if err != nil {
		cleanup()
		return nil, nil, func() {}, err
	}

	settings := cli.New()
	settings.KubeConfig = f.Name()
	settings.RepositoryConfig = filepath.Join(repoDir, "repositories.yaml")
	settings.RepositoryCache = filepath.Join(repoDir, "cache")

	cfg := new(action.Configuration)
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		cleanup()
		return nil, nil, func() {}, fmt.Errorf("初始化 Helm 配置失败: %w", err)
	}
	return cfg, settings, cleanup, nil
}

func releaseToMap(r *release.Release) map[string]any {
	chart := ""
	appVersion := ""
	if r.Chart != nil && r.Chart.Metadata != nil {
		chart = fmt.Sprintf("%s-%s", r.Chart.Metadata.Name, r.Chart.Metadata.Version)
		appVersion = r.Chart.Metadata.AppVersion
	}
	updated := ""
	if r.Info != nil && !r.Info.LastDeployed.IsZero() {
		updated = r.Info.LastDeployed.UTC().Format(time.RFC3339)
	}
	status := ""
	if r.Info != nil {
		status = r.Info.Status.String()
	}
	return map[string]any{
		"name":        r.Name,
		"namespace":   r.Namespace,
		"revision":    r.Version,
		"status":      status,
		"chart":       chart,
		"app_version": appVersion,
		"updated":     updated,
	}
}

func historyReleaseToMap(r *release.Release) map[string]any {
	m := releaseToMap(r)
	desc := ""
	if r.Info != nil {
		desc = r.Info.Description
	}
	m["description"] = desc
	return m
}

// addHelmRepo 添加 Helm 仓库并下载索引。
func addHelmRepo(settings *cli.EnvSettings, name, url string) error {
	repoFile := settings.RepositoryConfig
	if err := os.MkdirAll(filepath.Dir(repoFile), 0755); err != nil {
		return err
	}

	var f *repo.File
	if _, err := os.Stat(repoFile); err == nil {
		var loadErr error
		f, loadErr = repo.LoadFile(repoFile)
		if loadErr != nil {
			return loadErr
		}
	} else {
		f = repo.NewFile()
	}

	entry := &repo.Entry{Name: name, URL: url}
	f.Update(entry)

	r, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return err
	}
	if _, err := r.DownloadIndexFile(); err != nil {
		return err
	}
	return f.WriteFile(repoFile, 0644)
}

func validateHelmInstallRequest(req *helmInstallRequest) error {
	req.ReleaseName = strings.TrimSpace(req.ReleaseName)
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Chart = strings.TrimSpace(req.Chart)
	req.RepoURL = strings.TrimSpace(req.RepoURL)
	req.RepoName = strings.TrimSpace(req.RepoName)
	if req.Namespace == "" {
		req.Namespace = "default"
	}
	if req.ReleaseName == "" || req.Chart == "" {
		return fmt.Errorf("release_name 和 chart 不能为空")
	}
	if len(req.ReleaseName) > 53 || !helmKubernetesName.MatchString(req.ReleaseName) {
		return fmt.Errorf("release_name 必须是 1-53 位小写 DNS 名称")
	}
	if len(req.Namespace) > 63 || !helmKubernetesName.MatchString(req.Namespace) {
		return fmt.Errorf("namespace 必须是 1-63 位小写 DNS 名称")
	}
	if strings.Contains(req.Chart, "..") || strings.Contains(req.Chart, "://") || strings.HasPrefix(req.Chart, "/") {
		return fmt.Errorf("chart 只能是已配置仓库中的 chart 名称")
	}
	if len(req.ValuesYAML) > 1024*1024 {
		return fmt.Errorf("values.yaml 不能超过 1MB")
	}
	if req.RepoURL == "" && req.RepoName == "" {
		return nil
	}
	if req.RepoURL == "" || req.RepoName == "" {
		return fmt.Errorf("repo_url 和 repo_name 必须同时提供")
	}
	repoURL, err := url.ParseRequestURI(req.RepoURL)
	if err != nil || repoURL.Scheme != "https" || repoURL.Host == "" {
		return fmt.Errorf("Helm 仓库地址必须是有效的 HTTPS URL")
	}
	if len(req.RepoName) > 63 || !helmKubernetesName.MatchString(req.RepoName) {
		return fmt.Errorf("repo_name 必须是 1-63 位小写 DNS 名称")
	}
	if !strings.HasPrefix(req.Chart, req.RepoName+"/") {
		return fmt.Errorf("chart 必须以仓库名 %s/ 开头", req.RepoName)
	}
	return nil
}

func (kc *K8sController) helmPreflight(ctx context.Context, clusterID uint64) (string, any, error) {
	if kc.svc == nil || kc.deploySvc == nil {
		return "", nil, fmt.Errorf("Helm 部署前置服务未初始化")
	}
	apiOK, ready, total, version, err := kc.svc.CheckHealth(ctx, clusterID)
	if err != nil {
		return "", nil, err
	}
	if !apiOK || total == 0 || ready == 0 {
		return "", nil, fmt.Errorf("集群当前不可用于 Helm 部署：Ready 节点 %d/%d", ready, total)
	}
	master, err := kc.deploySvc.EnsureClusterMasterHelm(ctx, clusterID)
	if err != nil {
		return "", nil, err
	}
	return version, master, nil
}

// HelmPreflight validates the selected Kubernetes API, confirms the managed
// Master is reachable and installs Helm there when it is missing.
func (kc *K8sController) HelmPreflight(c *gin.Context) {
	clusterID, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "无效集群 ID")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	version, master, err := kc.helmPreflight(ctx, clusterID)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"cluster_version": version, "master": master})
}

// ListHelmReleases 列出集群中所有 Helm Release（使用 Helm SDK）。
func (kc *K8sController) ListHelmReleases(c *gin.Context) {
	namespace := strings.TrimSpace(c.Query("namespace"))
	cfg, _, cleanup, err := kc.newHelmConfig(c, namespace)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	defer cleanup()

	list := action.NewList(cfg)
	list.All = true
	list.AllNamespaces = namespace == ""

	releases, err := list.Run()
	if err != nil {
		resp.Fail(c, 5000, "获取 Helm Releases 失败: "+err.Error())
		return
	}

	out := make([]map[string]any, 0, len(releases))
	for _, r := range releases {
		out = append(out, releaseToMap(r))
	}
	resp.OK(c, gin.H{"list": out, "source": "helm-sdk"})
}

// GetHelmReleaseDetail 获取 Helm release 详情（使用 Helm SDK）。
func (kc *K8sController) GetHelmReleaseDetail(c *gin.Context) {
	ns := strings.TrimSpace(c.Query("namespace"))
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		resp.Fail(c, 4000, "name is required")
		return
	}
	cfg, _, cleanup, err := kc.newHelmConfig(c, ns)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	defer cleanup()

	get := action.NewGet(cfg)
	rel, err := get.Run(name)
	if err != nil {
		resp.Fail(c, 5000, "获取 Helm Release 详情失败: "+err.Error())
		return
	}

	detail := releaseToMap(rel)

	valuesGet := action.NewGetValues(cfg)
	valuesGet.AllValues = true
	if values, err := valuesGet.Run(name); err == nil {
		if b, err := yaml.Marshal(values); err == nil {
			detail["values_yaml"] = string(b)
		}
	}

	if rel.Manifest != "" {
		detail["manifest"] = rel.Manifest
	}

	hist := action.NewHistory(cfg)
	hist.Max = 256
	if history, err := hist.Run(name); err == nil {
		historyItems := make([]map[string]any, 0, len(history))
		for _, h := range history {
			historyItems = append(historyItems, historyReleaseToMap(h))
		}
		detail["history"] = historyItems
	}

	resp.OK(c, detail)
}

// HelmInstall 通过 Helm SDK 安装 chart。
// POST /clusters/:id/helm/install
// body: { release_name, namespace, chart, repo_url, repo_name, values_yaml }
func (kc *K8sController) HelmInstall(c *gin.Context) {
	clusterID, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "无效集群 ID")
		return
	}
	var req helmInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := validateHelmInstallRequest(&req); err != nil {
		resp.Fail(c, 4000, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), helmInstallTimeout+2*time.Minute)
	defer cancel()
	clusterVersion, master, err := kc.helmPreflight(ctx, clusterID)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	cfg, settings, cleanup, err := kc.newHelmConfig(c, req.Namespace)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	defer cleanup()

	if req.RepoURL != "" && req.RepoName != "" {
		if err := addHelmRepo(settings, req.RepoName, req.RepoURL); err != nil {
			resp.Fail(c, 5001, "添加 Helm 仓库失败: "+err.Error())
			return
		}
	}

	client := action.NewInstall(cfg)
	client.ReleaseName = req.ReleaseName
	client.Namespace = req.Namespace
	client.CreateNamespace = true
	client.Wait = true
	client.Atomic = true
	client.Timeout = helmInstallTimeout
	client.ChartPathOptions.Version = strings.TrimSpace(req.Version)

	cp, err := client.ChartPathOptions.LocateChart(req.Chart, settings)
	if err != nil {
		resp.Fail(c, 5001, "定位 Chart 失败: "+err.Error())
		return
	}

	chart, err := loader.Load(cp)
	if err != nil {
		resp.Fail(c, 5001, "加载 Chart 失败: "+err.Error())
		return
	}

	values := map[string]any{}
	if strings.TrimSpace(req.ValuesYAML) != "" {
		if err := yaml.Unmarshal([]byte(req.ValuesYAML), &values); err != nil {
			resp.Fail(c, 4000, "解析 values.yaml 失败: "+err.Error())
			return
		}
	}

	rel, err := client.Run(chart, values)
	if err != nil {
		resp.Fail(c, 5001, "Helm 安装失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{
		"output":          fmt.Sprintf("Release %s/%s 已安装并通过就绪校验，revision %d", rel.Namespace, rel.Name, rel.Version),
		"cluster_version": clusterVersion,
		"master":          master,
	})
}

// HelmUninstall 卸载 Helm release。
// DELETE /clusters/:id/helm/releases/:ns/:name
func (kc *K8sController) HelmUninstall(c *gin.Context) {
	ns := strings.TrimSpace(c.Param("ns"))
	name := strings.TrimSpace(c.Param("name"))
	if ns == "" || name == "" {
		resp.Fail(c, 4000, "namespace 和 name 不能为空")
		return
	}
	cfg, _, cleanup, err := kc.newHelmConfig(c, ns)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	defer cleanup()

	client := action.NewUninstall(cfg)
	if _, err := client.Run(name); err != nil {
		resp.Fail(c, 5001, "Helm 卸载失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"output": fmt.Sprintf("Release %s/%s 已卸载", ns, name)})
}

// HelmUpgrade 升级 Release，支持 values、版本锁定以及原子回滚。
func (kc *K8sController) HelmUpgrade(c *gin.Context) {
	ns, name := strings.TrimSpace(c.Param("ns")), strings.TrimSpace(c.Param("name"))
	var req struct {
		Chart      string `json:"chart"`
		Version    string `json:"version"`
		ValuesYAML string `json:"values_yaml"`
		Atomic     bool   `json:"atomic"`
		Wait       bool   `json:"wait"`
		Timeout    string `json:"timeout"`
	}
	if ns == "" || name == "" || c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Chart) == "" {
		resp.Fail(c, 4000, "namespace、name 和 chart 不能为空")
		return
	}
	cfg, settings, cleanup, err := kc.newHelmConfig(c, ns)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	defer cleanup()

	client := action.NewUpgrade(cfg)
	client.Namespace = ns
	client.ChartPathOptions.Version = req.Version
	client.Atomic = req.Atomic
	client.Wait = req.Wait
	if req.Timeout != "" {
		if d, err := time.ParseDuration(req.Timeout); err == nil {
			client.Timeout = d
		}
	}

	cp, err := client.ChartPathOptions.LocateChart(req.Chart, settings)
	if err != nil {
		resp.Fail(c, 5001, "定位 Chart 失败: "+err.Error())
		return
	}
	chart, err := loader.Load(cp)
	if err != nil {
		resp.Fail(c, 5001, "加载 Chart 失败: "+err.Error())
		return
	}

	values := map[string]any{}
	if strings.TrimSpace(req.ValuesYAML) != "" {
		if err := yaml.Unmarshal([]byte(req.ValuesYAML), &values); err != nil {
			resp.Fail(c, 4000, "解析 values.yaml 失败: "+err.Error())
			return
		}
	}

	rel, err := client.Run(name, chart, values)
	if err != nil {
		resp.Fail(c, 5001, "Helm 升级失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"output": fmt.Sprintf("Release %s/%s 已升级至 revision %d", rel.Namespace, rel.Name, rel.Version)})
}

// HelmRollback 将 Release 回滚到指定 revision。
func (kc *K8sController) HelmRollback(c *gin.Context) {
	ns, name := strings.TrimSpace(c.Param("ns")), strings.TrimSpace(c.Param("name"))
	var req struct {
		Revision int  `json:"revision"`
		Wait     bool `json:"wait"`
	}
	if ns == "" || name == "" || c.ShouldBindJSON(&req) != nil || req.Revision <= 0 {
		resp.Fail(c, 4000, "namespace、name 和有效 revision 不能为空")
		return
	}
	cfg, _, cleanup, err := kc.newHelmConfig(c, ns)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	defer cleanup()

	client := action.NewRollback(cfg)
	client.Version = req.Revision
	client.Wait = req.Wait
	if err := client.Run(name); err != nil {
		resp.Fail(c, 5001, "Helm 回滚失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"output": fmt.Sprintf("Release %s/%s 已回滚到 revision %d", ns, name, req.Revision)})
}

// HelmRepoList 列出当前集群已添加的 Helm 仓库。
// GET /clusters/:id/helm/repos
func (kc *K8sController) HelmRepoList(c *gin.Context) {
	repoDir, err := kc.helmRepoDir(c)
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}
	settings := cli.New()
	settings.RepositoryConfig = filepath.Join(repoDir, "repositories.yaml")
	settings.RepositoryCache = filepath.Join(repoDir, "cache")
	repoFile := settings.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if err != nil {
		if os.IsNotExist(err) {
			resp.OK(c, gin.H{"list": []any{}})
			return
		}
		resp.Fail(c, 5000, "读取 Helm 仓库配置失败: "+err.Error())
		return
	}
	list := make([]any, 0, len(f.Repositories))
	for _, e := range f.Repositories {
		list = append(list, map[string]any{"name": e.Name, "url": e.URL})
	}
	resp.OK(c, gin.H{"list": list})
}

// HelmSearch 搜索当前集群 Helm 仓库中的 chart。
// GET /clusters/:id/helm/search?keyword=xxx
func (kc *K8sController) HelmSearch(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		resp.Fail(c, 4000, "keyword 不能为空")
		return
	}
	repoDir, err := kc.helmRepoDir(c)
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}
	settings := cli.New()
	settings.RepositoryConfig = filepath.Join(repoDir, "repositories.yaml")
	settings.RepositoryCache = filepath.Join(repoDir, "cache")
	repoFile := settings.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if err != nil {
		if os.IsNotExist(err) {
			resp.OK(c, gin.H{"list": []any{}})
			return
		}
		resp.Fail(c, 5000, "读取 Helm 仓库配置失败: "+err.Error())
		return
	}

	results := make([]any, 0)
	for _, e := range f.Repositories {
		idxFile := filepath.Join(filepath.Dir(repoFile), e.Name+"-index.yaml")
		idx, err := repo.LoadIndexFile(idxFile)
		if err != nil {
			continue
		}
		for name, versions := range idx.Entries {
			for _, v := range versions {
				if strings.Contains(name, keyword) || strings.Contains(v.Description, keyword) ||
					(v.Name != "" && strings.Contains(v.Name, keyword)) {
					results = append(results, map[string]any{
						"name":        e.Name + "/" + name,
						"chart_name":  name,
						"repo_name":   e.Name,
						"repo_url":    e.URL,
						"version":     v.Version,
						"description": v.Description,
					})
				}
			}
		}
	}
	resp.OK(c, gin.H{"list": results})
}
