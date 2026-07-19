// metrics_provider.go 定义监控数据源的统一接口与数据模型。
// 上层业务通过 MetricsProvider 接口统一获取节点/Pod 指标，无需关心底层是
// Prometheus 还是 metrics-server。
package service

import (
	"context"
	"time"
)

// MonitorSource 表示监控数据源类型。
type MonitorSource string

const (
	MonitorSourceAuto         MonitorSource = "auto"
	MonitorSourcePrometheus   MonitorSource = "prometheus"
	MonitorSourceMetricsServer MonitorSource = "metrics_server"
)

// PrometheusStatus 表示 Prometheus 健康状态。
type PrometheusStatus string

const (
	PrometheusStatusUnknown   PrometheusStatus = "unknown"
	PrometheusStatusHealthy   PrometheusStatus = "healthy"
	PrometheusStatusUnhealthy PrometheusStatus = "unhealthy"
)

// MetricPoint 为时序数据点。
type MetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// NodeMetrics 为节点资源使用率统一模型。
type NodeMetrics struct {
	Name           string  `json:"name"`
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	CPUCapacity    int64   `json:"cpu_capacity"`
	MemoryCapacity int64   `json:"memory_capacity"`
	CPUUsed        int64   `json:"cpu_used"`
	MemoryUsed     int64   `json:"memory_used"`
}

// PodMetrics 为 Pod 资源使用量统一模型。
type PodMetrics struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CPU       int64  `json:"cpu"`
	Memory    int64  `json:"memory"`
}

// MetricsProvider 是监控数据源的抽象接口。
type MetricsProvider interface {
	// Name 返回 Provider 名称，用于日志与调试。
	Name() string

	// GetNodeMetrics 获取所有节点实时资源使用率。
	GetNodeMetrics(ctx context.Context, clusterID uint64) ([]NodeMetrics, error)

	// GetPodMetrics 获取指定命名空间下所有 Pod 资源使用量。
	GetPodMetrics(ctx context.Context, clusterID uint64, namespace string) ([]PodMetrics, error)

	// GetNodeMetricTrend 获取节点某指标在指定时间范围内的趋势数据。
	GetNodeMetricTrend(ctx context.Context, clusterID uint64, nodeName, metric string, start, end time.Time, step time.Duration) ([]MetricPoint, error)

	// GetPodMetricTrend 获取 Pod 某指标在指定时间范围内的趋势数据。
	GetPodMetricTrend(ctx context.Context, clusterID uint64, namespace, podName, metric string, start, end time.Time, step time.Duration) ([]MetricPoint, error)

	// Health 校验当前数据源是否可用。
	Health(ctx context.Context, clusterID uint64) error
}
