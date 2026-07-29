// metrics_provider_manager.go 负责 MetricsProvider 的选择、检测与自动切换。
package service

import (
	"context"
	"sync"
	"time"

	fleetdomain "k8s-platform-backend/internal/fleet/domain"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

// PrometheusInfo 集群 Prometheus 配置快照。
type PrometheusInfo struct {
	URL    string
	Status kopsapp.PrometheusStatus
}

var (
	providerCache   = map[uint64]kopsapp.MetricsProvider{}
	providerCacheMu sync.RWMutex
)

// GetPrometheusInfo 返回集群当前记录的 Prometheus 信息。
func (s *K8sService) GetPrometheusInfo(ctx context.Context, clusterID uint64) (*PrometheusInfo, error) {
	c, err := s.GetClusterMonitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return &PrometheusInfo{
		URL:    c.PrometheusURL,
		Status: kopsapp.PrometheusStatus(c.PrometheusStatus),
	}, nil
}

// GetClusterMonitorSource retrieves persisted monitoring configuration through Fleet.
func (s *K8sService) GetClusterMonitorSource(ctx context.Context, clusterID uint64) (*fleetdomain.Cluster, error) {
	cluster, err := s.clusterReg.MonitorSource(ctx, clusterID)
	return cluster, legacyClusterError(err)
}

func (s *K8sService) updateClusterMonitorSource(ctx context.Context, clusterID uint64, source kopsapp.MonitorSource, url string, status kopsapp.PrometheusStatus) error {
	return legacyClusterError(s.clusterReg.UpdateMonitorSource(ctx, clusterID, string(source), url, string(status)))
}

// DetectAndUpdatePrometheus 检测集群内 Prometheus 并持久化结果。
func (s *K8sService) DetectAndUpdatePrometheus(ctx context.Context, clusterID uint64) (*PrometheusInfo, error) {
	info, err := s.DetectPrometheus(ctx, clusterID)
	if err != nil {
		_ = s.updateClusterMonitorSource(ctx, clusterID, kopsapp.MonitorSourceMetricsServer, "", kopsapp.PrometheusStatusUnhealthy)
		return nil, err
	}

	status := kopsapp.PrometheusStatusHealthy
	pc := newPrometheusClient(info.URL)
	if err := pc.health(ctx); err != nil {
		status = kopsapp.PrometheusStatusUnhealthy
	}

	if err := s.updateClusterMonitorSource(ctx, clusterID, kopsapp.MonitorSourcePrometheus, info.URL, status); err != nil {
		return nil, err
	}
	return &PrometheusInfo{URL: info.URL, Status: status}, nil
}

// MetricsProvider 返回当前集群应使用的 MetricsProvider。
// 首次调用或 source=auto 时会触发检测。
func (s *K8sService) MetricsProvider(ctx context.Context, clusterID uint64) (kopsapp.MetricsProvider, error) {
	c, err := s.GetClusterMonitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	source := kopsapp.MonitorSource(c.MonitorSource)
	if source == kopsapp.MonitorSourceAuto || source == "" {
		// 未检测过，尝试检测 Prometheus。
		if _, detectErr := s.DetectAndUpdatePrometheus(ctx, clusterID); detectErr == nil {
			source = kopsapp.MonitorSourcePrometheus
		} else {
			source = kopsapp.MonitorSourceMetricsServer
		}
	}

	var provider kopsapp.MetricsProvider
	switch source {
	case kopsapp.MonitorSourcePrometheus:
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
func (s *K8sService) SwitchProvider(ctx context.Context, clusterID uint64, source kopsapp.MonitorSource) error {
	var url string
	var status kopsapp.PrometheusStatus
	switch source {
	case kopsapp.MonitorSourcePrometheus:
		info, err := s.GetClusterMonitorSource(ctx, clusterID)
		if err != nil {
			return err
		}
		url = info.PrometheusURL
		status = kopsapp.PrometheusStatus(info.PrometheusStatus)
	case kopsapp.MonitorSourceMetricsServer:
		url = ""
		status = kopsapp.PrometheusStatusUnknown
	case kopsapp.MonitorSourceAuto:
		_, err := s.DetectAndUpdatePrometheus(ctx, clusterID)
		return err
	default:
		return ErrWithMessage(ErrInvalidParams, "未知的数据源类型")
	}

	providerCacheMu.Lock()
	delete(providerCache, clusterID)
	providerCacheMu.Unlock()

	return s.updateClusterMonitorSource(ctx, clusterID, source, url, status)
}

// HealthCheckAndSwitch 对当前数据源做健康检查，异常时自动降级，恢复后自动切回。
func (s *K8sService) HealthCheckAndSwitch(ctx context.Context, clusterID uint64) (kopsapp.MetricsProvider, error) {
	c, err := s.GetClusterMonitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	source := kopsapp.MonitorSource(c.MonitorSource)
	prometheusURL := c.PrometheusURL

	// 1. 尝试 Prometheus。
	if source == kopsapp.MonitorSourcePrometheus || source == kopsapp.MonitorSourceAuto {
		if prometheusURL != "" {
			pc := newPrometheusClient(prometheusURL)
			if err := pc.health(ctx); err == nil {
				if source != kopsapp.MonitorSourcePrometheus || c.PrometheusStatus != string(kopsapp.PrometheusStatusHealthy) {
					_ = s.updateClusterMonitorSource(ctx, clusterID, kopsapp.MonitorSourcePrometheus, prometheusURL, kopsapp.PrometheusStatusHealthy)
				}
				provider := newPrometheusProvider(s)
				providerCacheMu.Lock()
				providerCache[clusterID] = provider
				providerCacheMu.Unlock()
				return provider, nil
			}
		}
		// Prometheus 不可用，降级到 metrics-server。
		_ = s.updateClusterMonitorSource(ctx, clusterID, kopsapp.MonitorSourceMetricsServer, prometheusURL, kopsapp.PrometheusStatusUnhealthy)
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
