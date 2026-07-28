// k8s_ctrl_metrics.go 提供资源使用率相关 API。
// 底层通过 MetricsProvider 统一适配 Prometheus / metrics-server 两种数据源。
package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/pkg/resp"
)

// ListNodeMetrics 列出所有节点的 CPU/内存使用率。
func (kc *K8sController) ListNodeMetrics(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	provider, err := kc.svc.MetricsProvider(c.Request.Context(), id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	metrics, err := provider.GetNodeMetrics(c.Request.Context(), id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	list := make([]map[string]any, 0, len(metrics))
	for _, m := range metrics {
		list = append(list, map[string]any{
			"name":            m.Name,
			"cpuUsage":        m.CPUUsage,
			"memoryUsage":     m.MemoryUsage,
			"cpuCapacity":     m.CPUCapacity,
			"memoryCapacity":  m.MemoryCapacity,
			"cpuUsed":         m.CPUUsed,
			"memoryUsed":      m.MemoryUsed,
			"source":          provider.Name(),
		})
	}
	resp.OK(c, gin.H{"list": list, "source": provider.Name()})
}

// ListPodMetricsUsage 列出指定命名空间下所有 Pod 的 CPU/内存使用量。
func (kc *K8sController) ListPodMetricsUsage(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	namespace := c.Query("namespace")

	provider, err := kc.svc.MetricsProvider(c.Request.Context(), id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	metrics, err := provider.GetPodMetrics(c.Request.Context(), id, namespace)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	list := make([]map[string]any, 0, len(metrics))
	for _, m := range metrics {
		list = append(list, map[string]any{
			"name":      m.Name,
			"namespace": m.Namespace,
			"cpu":       m.CPU,
			"memory":    m.Memory,
			"source":    provider.Name(),
		})
	}
	resp.OK(c, gin.H{"list": list, "source": provider.Name()})
}

// GetMetricsSource 返回当前集群的监控数据源配置。
func (kc *K8sController) GetMetricsSource(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	info, err := kc.svc.GetPrometheusInfo(c.Request.Context(), id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	c2, err := kc.svc.GetClusterMonitorSource(c.Request.Context(), id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{
		"monitor_source":         c2.MonitorSource,
		"prometheus_url":         info.URL,
		"prometheus_status":      info.Status,
		"prometheus_detected_at": c2.PrometheusDetectedAt,
	})
}

// DetectMetricsSource 手动触发 Prometheus 检测。
func (kc *K8sController) DetectMetricsSource(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	info, err := kc.svc.DetectAndUpdatePrometheus(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, 5000, "Prometheus 检测失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{
		"prometheus_url":    info.URL,
		"prometheus_status": info.Status,
	})
}

// SwitchMetricsSource 手动切换监控数据源。
func (kc *K8sController) SwitchMetricsSource(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req struct {
		Source string `json:"source" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := kc.svc.SwitchProvider(c.Request.Context(), id, service.MonitorSource(req.Source)); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"source": req.Source})
}

// GetMetricsTrend 查询资源使用率趋势。
// query：target=(node|pod)、name=节点/Pod 名、namespace=Pod 命名空间、metric=(cpu|memory)、start/end=Unix 秒、step=秒。
func (kc *K8sController) GetMetricsTrend(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	target := strings.ToLower(c.Query("target"))
	name := c.Query("name")
	namespace := c.Query("namespace")
	metric := strings.ToLower(c.Query("metric"))
	startSec, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	endSec, _ := strconv.ParseInt(c.Query("end"), 10, 64)
	stepSec, _ := strconv.ParseInt(c.Query("step"), 10, 64)

	if target == "" || name == "" || metric == "" || startSec == 0 || endSec == 0 || stepSec == 0 {
		resp.Fail(c, 4000, "target/name/metric/start/end/step 不能为空")
		return
	}

	provider, err := kc.svc.MetricsProvider(c.Request.Context(), id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	start := time.Unix(startSec, 0)
	end := time.Unix(endSec, 0)
	step := time.Duration(stepSec) * time.Second

	var points []service.MetricPoint
	if target == "node" {
		points, err = provider.GetNodeMetricTrend(c.Request.Context(), id, name, metric, start, end, step)
	} else {
		points, err = provider.GetPodMetricTrend(c.Request.Context(), id, namespace, name, metric, start, end, step)
	}
	if err != nil {
		resp.Fail(c, 5000, err.Error())
		return
	}

	resp.OK(c, gin.H{"points": points, "source": provider.Name()})
}

// HealthCheckMetricsSource 对当前数据源做健康检查并返回应使用的 Provider 名称。
func (kc *K8sController) HealthCheckMetricsSource(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	provider, err := kc.svc.HealthCheckAndSwitch(c.Request.Context(), id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"source": provider.Name()})
}
