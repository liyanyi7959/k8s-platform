package controller

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/resp"
)

// ListNodeMetrics 列出所有节点的 CPU/内存使用率。
// 返回每个节点的：name、cpuUsage(%)、memoryUsage(%)、cpuCapacity、memoryCapacity、cpuUsed、memoryUsed。
func (kc *K8sController) ListNodeMetrics(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	metrics, err := kc.svc.GetNodeMetrics(c.Request.Context(), id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": metrics})
}

// ListPodMetricsUsage 列出指定命名空间下所有 Pod 的 CPU/内存使用量。
// 注：方法名使用 ListPodMetricsUsage 以避免与 k8s_ctrl_pod.go 中已有的
// ListPodMetrics（路由 /clusters/:id/podmetrics，返回原始 PodMetrics 资源）冲突。
// query：namespace（可选，为空则查询所有命名空间）。
func (kc *K8sController) ListPodMetricsUsage(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	namespace := c.Query("namespace")
	metrics, err := kc.svc.GetPodMetrics(c.Request.Context(), id, namespace)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": metrics})
}
