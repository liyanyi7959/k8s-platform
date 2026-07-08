package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	// 用 K8sService.List 查询所有 Secret（会走缓存的脱敏逻辑，但 metadata.labels/annotations 保留）
	list, err := kc.svc.List(c.Request.Context(), id, gvrSecrets(), "", "", "", nil)
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
	resp.OK(c, gin.H{"list": releases})
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
	resp.OK(c, gin.H{
		"name":      labels["name"],
		"namespace": latestMeta["namespace"],
		"revision":  latestVersion,
		"status":    labels["status"],
		"chart":     annotations["helm.sh/chart"],
		"updated":   updated,
	})
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
