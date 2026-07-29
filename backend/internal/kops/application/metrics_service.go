package application

import (
	"context"
	"strings"
	"time"
)

type MetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// MonitorSource identifies the metrics backend selected for a cluster.
type MonitorSource string

const (
	MonitorSourceAuto          MonitorSource = "auto"
	MonitorSourcePrometheus    MonitorSource = "prometheus"
	MonitorSourceMetricsServer MonitorSource = "metrics_server"
)

// PrometheusStatus records the latest observed health of a Prometheus source.
type PrometheusStatus string

const (
	PrometheusStatusUnknown   PrometheusStatus = "unknown"
	PrometheusStatusHealthy   PrometheusStatus = "healthy"
	PrometheusStatusUnhealthy PrometheusStatus = "unhealthy"
)

type NodeMetrics struct {
	Name           string  `json:"name"`
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	CPUCapacity    int64   `json:"cpu_capacity"`
	MemoryCapacity int64   `json:"memory_capacity"`
	CPUUsed        int64   `json:"cpu_used"`
	MemoryUsed     int64   `json:"memory_used"`
}

type PodMetrics struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CPU       int64  `json:"cpu"`
	Memory    int64  `json:"memory"`
}

// MetricsProvider is implemented by infrastructure adapters such as
// Prometheus and Kubernetes metrics-server clients.
type MetricsProvider interface {
	Name() string
	GetNodeMetrics(context.Context, uint64) ([]NodeMetrics, error)
	GetPodMetrics(context.Context, uint64, string) ([]PodMetrics, error)
	GetNodeMetricTrend(context.Context, uint64, string, string, time.Time, time.Time, time.Duration) ([]MetricPoint, error)
	GetPodMetricTrend(context.Context, uint64, string, string, string, time.Time, time.Time, time.Duration) ([]MetricPoint, error)
	Health(context.Context, uint64) error
}
type MetricsTrendQuery struct {
	ClusterID uint64
	Target    string
	Name      string
	Namespace string
	Metric    string
	Start     time.Time
	End       time.Time
	Step      time.Duration
}
type MetricsRuntime interface {
	NodeMetrics(context.Context, uint64) (any, error)
	PodMetrics(context.Context, uint64, string) (any, error)
	Source(context.Context, uint64) (any, error)
	Detect(context.Context, uint64) (any, error)
	Switch(context.Context, uint64, string) error
	Trend(context.Context, MetricsTrendQuery) (any, error)
	HealthCheck(context.Context, uint64) (any, error)
}
type MetricsService struct{ runtime MetricsRuntime }

func NewMetricsService(runtime MetricsRuntime) *MetricsService {
	return &MetricsService{runtime: runtime}
}
func (s *MetricsService) NodeMetrics(ctx context.Context, clusterID uint64) (any, error) {
	if err := validateMetricsCluster(clusterID); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.NodeMetrics(ctx, clusterID)
}
func (s *MetricsService) PodMetrics(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if err := validateMetricsCluster(clusterID); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.PodMetrics(ctx, clusterID, strings.TrimSpace(namespace))
}
func (s *MetricsService) Source(ctx context.Context, clusterID uint64) (any, error) {
	if err := validateMetricsCluster(clusterID); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Source(ctx, clusterID)
}
func (s *MetricsService) Detect(ctx context.Context, clusterID uint64) (any, error) {
	if err := validateMetricsCluster(clusterID); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Detect(ctx, clusterID)
}
func (s *MetricsService) Switch(ctx context.Context, clusterID uint64, source string) error {
	if err := validateMetricsCluster(clusterID); err != nil {
		return err
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if source != "auto" && source != "prometheus" && source != "metrics_server" {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Switch(ctx, clusterID, source)
}
func (s *MetricsService) Trend(ctx context.Context, query MetricsTrendQuery) (any, error) {
	query.Target = strings.ToLower(strings.TrimSpace(query.Target))
	query.Name = strings.TrimSpace(query.Name)
	query.Namespace = strings.TrimSpace(query.Namespace)
	query.Metric = strings.ToLower(strings.TrimSpace(query.Metric))
	if query.ClusterID == 0 || query.Name == "" || query.Start.Unix() <= 0 || query.End.Unix() <= 0 || query.Step <= 0 || query.End.Before(query.Start) || (query.Target != "node" && query.Target != "pod") || (query.Metric != "cpu" && query.Metric != "memory") {
		return nil, ErrInvalidParams
	}
	if query.Target == "pod" && query.Namespace == "" {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Trend(ctx, query)
}
func (s *MetricsService) HealthCheck(ctx context.Context, clusterID uint64) (any, error) {
	if err := validateMetricsCluster(clusterID); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.HealthCheck(ctx, clusterID)
}
func validateMetricsCluster(clusterID uint64) error {
	if clusterID == 0 {
		return ErrInvalidParams
	}
	return nil
}
