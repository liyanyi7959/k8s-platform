package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/resp"
)

// ListHelmReleases 列出集群中所有 Helm Release。
// 通过查询所有命名空间中 label owner=helm 的 Secret 来获取 release 信息：
// Helm v3 将 release 存储在 Secret 中，label 包含 name/status/version，
// annotation 包含 chart/modifiedAt。
// Secret 脱敏只修改 data，metadata.labels/annotations 保留。
func (kc *K8sController) ListHelmReleases(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if kubeconfigFile, err := kc.writeKubeconfigTmp(ctx, id); err == nil {
		defer os.Remove(kubeconfigFile)
		args := []string{"list", "--all", "--output", "json", "--kubeconfig", kubeconfigFile}
		if namespace == "" {
			args = append(args, "--all-namespaces")
		} else {
			args = append(args, "--namespace", namespace)
		}
		if output, runErr := runHelm(ctx, args...); runErr == nil {
			var releases []map[string]any
			if jsonErr := json.Unmarshal([]byte(output), &releases); jsonErr == nil {
				resp.OK(c, gin.H{"list": releases, "source": "helm"})
				return
			}
		}
	}

	// Helm CLI 不可用时降级到只查询 Helm Secret，避免扫描集群内全部 Secret。
	list, err := kc.svc.List(c.Request.Context(), id, gvrSecrets(), namespace, "", "", map[string]string{"labelSelector": "owner=helm"})
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	// 从 Secret 的 label/annotation 中提取 Helm release 信息
	releaseMap := map[string]map[string]any{}
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		meta, _ := raw["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		labels, _ := meta["labels"].(map[string]any)
		if labels == nil || labels["owner"] != "helm" {
			continue
		}
		name, _ := labels["name"].(string)
		if name == "" {
			continue
		}
		versionStr, _ := labels["version"].(string)
		version, _ := strconv.Atoi(versionStr)
		status, _ := labels["status"].(string)
		ns, _ := meta["namespace"].(string)
		annotations, _ := meta["annotations"].(map[string]any)
		chart, _ := annotations["helm.sh/chart"].(string)
		modified, _ := annotations["modifiedAt"].(string)
		// 转换时间戳
		updated := ""
		if ts, err := strconv.ParseInt(modified, 10, 64); err == nil {
			updated = time.Unix(ts, 0).UTC().Format(time.RFC3339)
		}
		key := ns + "/" + name
		existing, exists := releaseMap[key]
		if !exists || version > existing["revision"].(int) {
			releaseMap[key] = map[string]any{
				"name":      name,
				"namespace": ns,
				"revision":  version,
				"status":    status,
				"chart":     chart,
				"updated":   updated,
			}
		}
	}
	// 转为列表
	releases := make([]map[string]any, 0, len(releaseMap))
	for _, r := range releaseMap {
		releases = append(releases, r)
	}
	sort.Slice(releases, func(i, j int) bool {
		return fmt.Sprint(releases[i]["namespace"], "/", releases[i]["name"]) < fmt.Sprint(releases[j]["namespace"], "/", releases[j]["name"])
	})
	resp.OK(c, gin.H{"list": releases, "source": "kubernetes-secrets"})
}

// GetHelmReleaseDetail 获取 Helm release 详情。
// 通过 labelSelector owner=helm,name=<release> 查询指定 release 的所有版本 Secret，
// 返回最新版本的基本信息。
func (kc *K8sController) GetHelmReleaseDetail(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		resp.Fail(c, 4000, "name is required")
		return
	}
	// 查询指定命名空间的 Helm release Secret
	labelSelector := "owner=helm,name=" + name
	list, err := kc.svc.List(c.Request.Context(), id, gvrSecrets(), ns, "", "", map[string]string{"labelSelector": labelSelector})
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	// 找到最新版本的 Secret
	var latestVersion int
	var latestMeta map[string]any
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		meta, _ := raw["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		labels, _ := meta["labels"].(map[string]any)
		if labels == nil {
			continue
		}
		versionStr, _ := labels["version"].(string)
		version, _ := strconv.Atoi(versionStr)
		if version > latestVersion {
			latestVersion = version
			latestMeta = meta
		}
	}
	if latestMeta == nil {
		resp.Fail(c, 4040, "release not found")
		return
	}
	// 返回基本信息
	labels, _ := latestMeta["labels"].(map[string]any)
	annotations, _ := latestMeta["annotations"].(map[string]any)
	modified, _ := annotations["modifiedAt"].(string)
	updated := ""
	if ts, err := strconv.ParseInt(modified, 10, 64); err == nil {
		updated = time.Unix(ts, 0).UTC().Format(time.RFC3339)
	}
	detail := gin.H{
		"name":      labels["name"],
		"namespace": latestMeta["namespace"],
		"revision":  latestVersion,
		"status":    labels["status"],
		"chart":     annotations["helm.sh/chart"],
		"updated":   updated,
	}

	// 详情是用户按需触发，补充 values、渲染清单和 revision 历史。
	ctx, cancel := context.WithTimeout(c.Request.Context(), helmCmdTimeout)
	defer cancel()
	if kubeconfigFile, err := kc.writeKubeconfigTmp(ctx, id); err == nil {
		defer os.Remove(kubeconfigFile)
		common := []string{"--namespace", ns, "--kubeconfig", kubeconfigFile}
		if output, runErr := runHelm(ctx, append([]string{"get", "values", name, "--all", "--output", "yaml"}, common...)...); runErr == nil {
			detail["values_yaml"] = strings.TrimSpace(output)
		}
		if output, runErr := runHelm(ctx, append([]string{"get", "manifest", name}, common...)...); runErr == nil {
			detail["manifest"] = strings.TrimSpace(output)
		}
		if output, runErr := runHelm(ctx, append([]string{"history", name, "--output", "json"}, common...)...); runErr == nil {
			var history []any
			if json.Unmarshal([]byte(output), &history) == nil {
				detail["history"] = history
			}
		}
	}
	resp.OK(c, detail)
}

// ---------------------------------------------------------------------------
// Helm CLI 集成（安装 / 卸载 / 仓库管理 / 搜索）
// ---------------------------------------------------------------------------

// helmCmdTimeout Helm CLI 命令超时时间
const helmCmdTimeout = 60 * time.Second

// writeKubeconfigTmp 将集群 kubeconfig 写入临时文件，返回文件路径。
// 调用方需在使用完成后 defer os.Remove(path) 清理。
func (kc *K8sController) writeKubeconfigTmp(ctx context.Context, id uint64) (string, error) {
	kubeconfig, err := kc.svc.GetKubeconfig(ctx, id)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "kubeconfig-*")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	if _, err := f.WriteString(kubeconfig); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("写入 kubeconfig 失败: %w", err)
	}
	f.Close()
	return f.Name(), nil
}

// runHelm 执行 helm CLI 命令并返回合并输出。
// 如果 helm 未安装返回友好错误。
func runHelm(ctx context.Context, args ...string) (string, error) {
	if _, err := exec.LookPath("helm"); err != nil {
		return "", fmt.Errorf("helm CLI 未安装")
	}
	cmd := exec.CommandContext(ctx, "helm", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("helm 命令执行失败: %w", err)
	}
	return string(out), nil
}

// HelmInstall 通过 helm CLI 安装 chart。
// POST /clusters/:id/helm/install
// body: { release_name, namespace, chart, repo_url, repo_name, values_yaml }
func (kc *K8sController) HelmInstall(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req struct {
		ReleaseName string `json:"release_name"`
		Namespace   string `json:"namespace"`
		Chart       string `json:"chart"`
		RepoURL     string `json:"repo_url"`
		RepoName    string `json:"repo_name"`
		ValuesYAML  string `json:"values_yaml"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if req.ReleaseName == "" || req.Chart == "" {
		resp.Fail(c, 4000, "release_name 和 chart 不能为空")
		return
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), helmCmdTimeout)
	defer cancel()

	// 写入 kubeconfig 临时文件
	kubeconfigFile, err := kc.writeKubeconfigTmp(ctx, id)
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}
	defer os.Remove(kubeconfigFile)

	// 如果提供了仓库地址，先添加仓库
	if req.RepoURL != "" && req.RepoName != "" {
		if _, err := runHelm(ctx, "repo", "add", req.RepoName, req.RepoURL, "--kubeconfig", kubeconfigFile); err != nil {
			resp.Fail(c, 5001, "添加 Helm 仓库失败: "+err.Error())
			return
		}
	}
	// 更新仓库索引
	if req.RepoName != "" {
		if _, err := runHelm(ctx, "repo", "update", "--kubeconfig", kubeconfigFile); err != nil {
			resp.Fail(c, 5001, "更新 Helm 仓库失败: "+err.Error())
			return
		}
	}

	// 构建 install 命令参数
	args := []string{"install", req.ReleaseName, req.Chart, "-n", req.Namespace, "--kubeconfig", kubeconfigFile}
	// 写入 values.yaml 临时文件
	if strings.TrimSpace(req.ValuesYAML) != "" {
		valuesFile, err := os.CreateTemp("", "values-*.yaml")
		if err != nil {
			resp.Fail(c, 5000, "创建临时文件失败: "+err.Error())
			return
		}
		if _, err := valuesFile.WriteString(req.ValuesYAML); err != nil {
			valuesFile.Close()
			os.Remove(valuesFile.Name())
			resp.Fail(c, 5000, "写入 values 文件失败: "+err.Error())
			return
		}
		valuesFile.Close()
		defer os.Remove(valuesFile.Name())
		args = append(args, "-f", valuesFile.Name())
	}

	output, err := runHelm(ctx, args...)
	if err != nil {
		resp.Fail(c, 5001, "Helm 安装失败: "+output+err.Error())
		return
	}
	resp.OK(c, gin.H{"output": strings.TrimSpace(output)})
}

// HelmUninstall 卸载 Helm release。
// DELETE /clusters/:id/helm/releases/:ns/:name
func (kc *K8sController) HelmUninstall(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Param("ns"))
	name := strings.TrimSpace(c.Param("name"))
	if ns == "" || name == "" {
		resp.Fail(c, 4000, "namespace 和 name 不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), helmCmdTimeout)
	defer cancel()

	kubeconfigFile, err := kc.writeKubeconfigTmp(ctx, id)
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}
	defer os.Remove(kubeconfigFile)

	output, err := runHelm(ctx, "uninstall", name, "-n", ns, "--kubeconfig", kubeconfigFile)
	if err != nil {
		resp.Fail(c, 5001, "Helm 卸载失败: "+output+err.Error())
		return
	}
	resp.OK(c, gin.H{"output": strings.TrimSpace(output)})
}

// HelmUpgrade 升级 Release，支持 values、版本锁定以及原子回滚。
func (kc *K8sController) HelmUpgrade(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), helmCmdTimeout)
	defer cancel()
	kubeconfigFile, err := kc.writeKubeconfigTmp(ctx, id)
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}
	defer os.Remove(kubeconfigFile)
	args := []string{"upgrade", name, strings.TrimSpace(req.Chart), "--namespace", ns, "--kubeconfig", kubeconfigFile}
	if version := strings.TrimSpace(req.Version); version != "" {
		args = append(args, "--version", version)
	}
	if req.Atomic {
		args = append(args, "--atomic")
	}
	if req.Wait {
		args = append(args, "--wait")
	}
	if timeout := strings.TrimSpace(req.Timeout); timeout != "" {
		args = append(args, "--timeout", timeout)
	}
	if strings.TrimSpace(req.ValuesYAML) != "" {
		valuesFile, fileErr := os.CreateTemp("", "helm-values-*.yaml")
		if fileErr != nil {
			resp.Fail(c, 5000, fileErr.Error())
			return
		}
		if _, fileErr = valuesFile.WriteString(req.ValuesYAML); fileErr != nil {
			valuesFile.Close()
			os.Remove(valuesFile.Name())
			resp.Fail(c, 5000, fileErr.Error())
			return
		}
		valuesFile.Close()
		defer os.Remove(valuesFile.Name())
		args = append(args, "--values", valuesFile.Name())
	}
	output, runErr := runHelm(ctx, args...)
	if runErr != nil {
		resp.Fail(c, 5001, "Helm 升级失败: "+output+runErr.Error())
		return
	}
	resp.OK(c, gin.H{"output": strings.TrimSpace(output)})
}

// HelmRollback 将 Release 回滚到指定 revision。
func (kc *K8sController) HelmRollback(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns, name := strings.TrimSpace(c.Param("ns")), strings.TrimSpace(c.Param("name"))
	var req struct {
		Revision int  `json:"revision"`
		Wait     bool `json:"wait"`
	}
	if ns == "" || name == "" || c.ShouldBindJSON(&req) != nil || req.Revision <= 0 {
		resp.Fail(c, 4000, "namespace、name 和有效 revision 不能为空")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), helmCmdTimeout)
	defer cancel()
	kubeconfigFile, err := kc.writeKubeconfigTmp(ctx, id)
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}
	defer os.Remove(kubeconfigFile)
	args := []string{"rollback", name, strconv.Itoa(req.Revision), "--namespace", ns, "--kubeconfig", kubeconfigFile}
	if req.Wait {
		args = append(args, "--wait")
	}
	output, runErr := runHelm(ctx, args...)
	if runErr != nil {
		resp.Fail(c, 5001, "Helm 回滚失败: "+output+runErr.Error())
		return
	}
	resp.OK(c, gin.H{"output": strings.TrimSpace(output)})
}

// HelmRepoList 列出已添加的 Helm 仓库。
// GET /clusters/:id/helm/repos
func (kc *K8sController) HelmRepoList(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), helmCmdTimeout)
	defer cancel()

	kubeconfigFile, err := kc.writeKubeconfigTmp(ctx, id)
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}
	defer os.Remove(kubeconfigFile)

	output, err := runHelm(ctx, "repo", "list", "--kubeconfig", kubeconfigFile, "-o", "json")
	if err != nil {
		// helm repo list 在没有仓库时退出码非零，返回空列表
		if strings.Contains(output, "no repositories") {
			resp.OK(c, gin.H{"list": []any{}})
			return
		}
		resp.Fail(c, 5001, "获取 Helm 仓库列表失败: "+err.Error())
		return
	}
	var list []any
	if strings.TrimSpace(output) != "" {
		if err := json.Unmarshal([]byte(output), &list); err != nil {
			resp.Fail(c, 5000, "解析 Helm 仓库列表失败: "+err.Error())
			return
		}
	}
	resp.OK(c, gin.H{"list": list})
}

// HelmSearch 搜索 Helm chart。
// GET /clusters/:id/helm/search?keyword=xxx
func (kc *K8sController) HelmSearch(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		resp.Fail(c, 4000, "keyword 不能为空")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), helmCmdTimeout)
	defer cancel()

	kubeconfigFile, err := kc.writeKubeconfigTmp(ctx, id)
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}
	defer os.Remove(kubeconfigFile)

	output, err := runHelm(ctx, "search", "repo", keyword, "--kubeconfig", kubeconfigFile, "-o", "json")
	if err != nil {
		resp.Fail(c, 5001, "搜索 Helm chart 失败: "+err.Error())
		return
	}
	var list []any
	if strings.TrimSpace(output) != "" {
		if err := json.Unmarshal([]byte(output), &list); err != nil {
			resp.Fail(c, 5000, "解析搜索结果失败: "+err.Error())
			return
		}
	}
	resp.OK(c, gin.H{"list": list})
}
