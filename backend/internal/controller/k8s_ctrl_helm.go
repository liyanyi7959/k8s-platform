package controller

import (
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
