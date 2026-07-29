package http

import (
	"github.com/gin-gonic/gin"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
	"strconv"
	"strings"
	"time"
)

type MetricsController struct{ service *kopsapp.MetricsService }

func NewMetricsController(service *kopsapp.MetricsService) *MetricsController {
	return &MetricsController{service: service}
}
func (ctl *MetricsController) NodeMetrics(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.NodeMetrics(c.Request.Context(), metricsClusterID(c))
	ctl.respond(c, value, err)
}
func (ctl *MetricsController) PodMetrics(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.PodMetrics(c.Request.Context(), metricsClusterID(c), c.Query("namespace"))
	ctl.respond(c, value, err)
}
func (ctl *MetricsController) Source(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.Source(c.Request.Context(), metricsClusterID(c))
	ctl.respond(c, value, err)
}
func (ctl *MetricsController) Detect(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.Detect(c.Request.Context(), metricsClusterID(c))
	ctl.respond(c, value, err)
}
func (ctl *MetricsController) Switch(c *gin.Context) {
	var request struct {
		Source string `json:"source" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if !ctl.ready(c) {
		return
	}
	err := ctl.service.Switch(c.Request.Context(), metricsClusterID(c), request.Source)
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{"source": request.Source})
}
func (ctl *MetricsController) Trend(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	start, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end"), 10, 64)
	step, _ := strconv.ParseInt(c.Query("step"), 10, 64)
	value, err := ctl.service.Trend(c.Request.Context(), kopsapp.MetricsTrendQuery{ClusterID: metricsClusterID(c), Target: c.Query("target"), Name: c.Query("name"), Namespace: c.Query("namespace"), Metric: c.Query("metric"), Start: time.Unix(start, 0), End: time.Unix(end, 0), Step: time.Duration(step) * time.Second})
	ctl.respond(c, value, err)
}
func (ctl *MetricsController) HealthCheck(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.HealthCheck(c.Request.Context(), metricsClusterID(c))
	ctl.respond(c, value, err)
}
func (ctl *MetricsController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}
func (ctl *MetricsController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}
func metricsClusterID(c *gin.Context) uint64 {
	value, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil {
		return 0
	}
	return value
}
