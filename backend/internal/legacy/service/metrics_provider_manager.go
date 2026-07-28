// metrics_provider_manager.go 负责 MetricsProvider 的选择、检测与自动切换。
package service

import (
	"context"
	"sync"
	"time"

	"k8s-platform-backend/internal/legacy/model"
)

// PrometheusInfo 集群 Prometheus 配置快照。
type PrometheusInfo struct {
	URL    string
	Status PrometheusStatus
}

var (
	providerCache   = map[uint64]MetricsProvider{}
	providerCacheMu sync.RWMutex
)

// GetPrometheusInfo 返回集群当前记录的 Prometheus 信息。
func (s *K8sService) GetPrometheusInfo(ctx context.Context, clusterID uint64) (*PrometheusInfo, error) {
	c, err := s.clusterReg.GetClusterMonitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return &PrometheusInfo{
		URL:    c.PrometheusURL,
		Status: PrometheusStatus(c.PrometheusStatus),
	}, nil
}

// GetClusterMonitorSource 代理到 ClusterRegistryService，供控制器查询完整数据源配置。
func (s *K8sService) GetClusterMonitorSource(ctx context.Context, clusterID uint64) (*model.Cluster, error) {
	return s.clusterReg.GetClusterMonitorSource(ctx, clusterID)
}

// DetectAndUpdatePrometheus 检测集群内 Prometheus 并持久化结果。
func (s *K8sService) DetectAndUpdatePrometheus(ctx context.Context, clusterID uint64) (*PrometheusInfo, error) {
	info, err := s.DetectPrometheus(ctx, clusterID)
	if err != nil {
		_ = s.clusterReg.UpdateClusterMonitorSource(ctx, clusterID, MonitorSourceMetricsServer, "", PrometheusStatusUnhealthy)
		return nil, err
	}

	status := PrometheusStatusHealthy
	pc := newPrometheusClient(info.URL)
	if err := pc.health(ctx); err != nil {
		status = PrometheusStatusUnhealthy
	}

	if err := s.clusterReg.UpdateClusterMonitorSource(ctx, clusterID, MonitorSourcePrometheus, info.URL, status); err != nil {
		return nil, err
	}
	return &PrometheusInfo{URL: info.URL, Status: status}, nil
}

// MetricsProvider 返回当前集群应使用的 MetricsProvider。
// 首次调用或 source=auto 时会触发检测。
func (s *K8sService) MetricsProvider(ctx context.Context, clusterID uint64) (MetricsProvider, error) {
	c, err := s.clusterReg.GetClusterMonitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	source := MonitorSource(c.MonitorSource)
	if source == MonitorSourceAuto || source == "" {
		// 未检测过，尝试检测 Prometheus。
		if _, detectErr := s.DetectAndUpdatePrometheus(ctx, clusterID); detectErr == nil {
			source = MonitorSourcePrometheus
		} else {
			source = MonitorSourceMetricsServer
		}
	}

	var provider MetricsProvider
	switch source {
	case MonitorSourcePrometheus:
		provider = newPrometheusProvider(s)
	default:
		provider = newMetricsServerProvider(s)
	}

	// 缓存当前 Provider，便于后续健康检查快速返回。
	providerCacheMu.Lock()
	providerCache[clusterID] = provider
	providerCacheMu.Unlock()

	return provider, nil
}

// SwitchProvider 手动切换数据源。
func (s *K8sService) SwitchProvider(ctx context.Context, clusterID uint64, source MonitorSource) error {
	var url string
	var status PrometheusStatus
	switch source {
	case MonitorSourcePrometheus:
		info, err := s.clusterReg.GetClusterMonitorSource(ctx, clusterID)
		if err != nil {
			return err
		}
		url = info.PrometheusURL
		status = PrometheusStatus(info.PrometheusStatus)
	case MonitorSourceMetricsServer:
		url = ""
		status = PrometheusStatusUnknown
	case MonitorSourceAuto:
		_, err := s.DetectAndUpdatePrometheus(ctx, clusterID)
		return err
	default:
		return ErrWithMessage(ErrInvalidParams, "未知的数据源类型")
	}

	providerCacheMu.Lock()
	delete(providerCache, clusterID)
	providerCacheMu.Unlock()

	return s.clusterReg.UpdateClusterMonitorSource(ctx, clusterID, source, url, status)
}

// HealthCheckAndSwitch 对当前数据源做健康检查，异常时自动降级，恢复后自动切回。
func (s *K8sService) HealthCheckAndSwitch(ctx context.Context, clusterID uint64) (MetricsProvider, error) {
	c, err := s.clusterReg.GetClusterMonitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	source := MonitorSource(c.MonitorSource)
	prometheusURL := c.PrometheusURL

	// 1. 尝试 Prometheus。
	if source == MonitorSourcePrometheus || source == MonitorSourceAuto {
		if prometheusURL != "" {
			pc := newPrometheusClient(prometheusURL)
			if err := pc.health(ctx); err == nil {
				if source != MonitorSourcePrometheus || c.PrometheusStatus != string(PrometheusStatusHealthy) {
					_ = s.clusterReg.UpdateClusterMonitorSource(ctx, clusterID, MonitorSourcePrometheus, prometheusURL, PrometheusStatusHealthy)
				}
				provider := newPrometheusProvider(s)
				providerCacheMu.Lock()
				providerCache[clusterID] = provider
				providerCacheMu.Unlock()
				return provider, nil
			}
		}
		// Prometheus 不可用，降级到 metrics-server。
		_ = s.clusterReg.UpdateClusterMonitorSource(ctx, clusterID, MonitorSourceMetricsServer, prometheusURL, PrometheusStatusUnhealthy)
	}

	// 2. 使用 metrics-server。
	provider := newMetricsServerProvider(s)
	providerCacheMu.Lock()
	providerCache[clusterID] = provider
	providerCacheMu.Unlock()
	return provider, nil
}

// 以下用于控制健康检查频率，避免每次请求都检测。
var lastHealthCheck = map[uint64]time.Time{}
var lastHealthCheckMu sync.Mutex

const healthCheckInterval = 30 * time.Second

func shouldHealthCheck(clusterID uint64) bool {
	lastHealthCheckMu.Lock()
	defer lastHealthCheckMu.Unlock()
	last, ok := lastHealthCheck[clusterID]
	if !ok || time.Since(last) > healthCheckInterval {
		lastHealthCheck[clusterID] = time.Now()
		return true
	}
	return false
}
