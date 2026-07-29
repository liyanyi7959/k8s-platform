package kops

import (
	"context"

	fleetapp "k8s-platform-backend/internal/fleet/application"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

// MetricsRuntime formats Kops metrics responses while provider discovery and
// selection live in the dedicated adapter manager.
type MetricsRuntime struct {
	manager *MetricsProviderManager
}

func NewMetricsRuntime(transport *service.K8sService, clusters *fleetapp.Registry) *MetricsRuntime {
	return &MetricsRuntime{manager: NewMetricsProviderManager(transport, clusters)}
}

func (r *MetricsRuntime) NodeMetrics(ctx context.Context, clusterID uint64) (any, error) {
	provider, err := r.provider(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	values, err := provider.GetNodeMetrics(ctx, clusterID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	list := make([]map[string]any, 0, len(values))
	for _, value := range values {
		list = append(list, map[string]any{
			"name": value.Name, "cpuUsage": value.CPUUsage, "memoryUsage": value.MemoryUsage,
			"cpuCapacity": value.CPUCapacity, "memoryCapacity": value.MemoryCapacity,
			"cpuUsed": value.CPUUsed, "memoryUsed": value.MemoryUsed, "source": provider.Name(),
		})
	}
	return map[string]any{"list": list, "source": provider.Name()}, nil
}

func (r *MetricsRuntime) PodMetrics(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	provider, err := r.provider(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	values, err := provider.GetPodMetrics(ctx, clusterID, namespace)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	list := make([]map[string]any, 0, len(values))
	for _, value := range values {
		list = append(list, map[string]any{"name": value.Name, "namespace": value.Namespace, "cpu": value.CPU, "memory": value.Memory, "source": provider.Name()})
	}
	return map[string]any{"list": list, "source": provider.Name()}, nil
}

func (r *MetricsRuntime) Source(ctx context.Context, clusterID uint64) (any, error) {
	if r == nil || r.manager == nil {
		return nil, kopsapp.ErrConflict
	}
	info, err := r.manager.GetPrometheusInfo(ctx, clusterID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	source, err := r.manager.monitorSource(ctx, clusterID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{
		"monitor_source": source.MonitorSource, "prometheus_url": info.URL, "prometheus_status": info.Status,
		"prometheus_detected_at": source.PrometheusDetectedAt,
	}, nil
}

func (r *MetricsRuntime) Detect(ctx context.Context, clusterID uint64) (any, error) {
	if r == nil || r.manager == nil {
		return nil, kopsapp.ErrConflict
	}
	info, err := r.manager.DetectAndUpdatePrometheus(ctx, clusterID)
	if err != nil {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrRuntime, "Prometheus detection failed: "+err.Error())
	}
	return map[string]any{"prometheus_url": info.URL, "prometheus_status": info.Status}, nil
}

func (r *MetricsRuntime) Switch(ctx context.Context, clusterID uint64, source string) error {
	if r == nil || r.manager == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.manager.SwitchProvider(ctx, clusterID, kopsapp.MonitorSource(source)))
}

func (r *MetricsRuntime) Trend(ctx context.Context, query kopsapp.MetricsTrendQuery) (any, error) {
	provider, err := r.provider(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	var values []kopsapp.MetricPoint
	if query.Target == "node" {
		values, err = provider.GetNodeMetricTrend(ctx, query.ClusterID, query.Name, query.Metric, query.Start, query.End, query.Step)
	} else {
		values, err = provider.GetPodMetricTrend(ctx, query.ClusterID, query.Namespace, query.Name, query.Metric, query.Start, query.End, query.Step)
	}
	if err != nil {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrRuntime, err.Error())
	}
	return map[string]any{"points": values, "source": provider.Name()}, nil
}

func (r *MetricsRuntime) HealthCheck(ctx context.Context, clusterID uint64) (any, error) {
	if r == nil || r.manager == nil {
		return nil, kopsapp.ErrConflict
	}
	provider, err := r.manager.HealthCheckAndSwitch(ctx, clusterID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"source": provider.Name()}, nil
}

func (r *MetricsRuntime) provider(ctx context.Context, clusterID uint64) (kopsapp.MetricsProvider, error) {
	if r == nil || r.manager == nil {
		return nil, kopsapp.ErrConflict
	}
	provider, err := r.manager.Provider(ctx, clusterID)
	return provider, translateKopsRuntimeError(err)
}
