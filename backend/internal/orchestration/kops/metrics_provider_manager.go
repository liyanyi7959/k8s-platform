package kops

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fleetapp "k8s-platform-backend/internal/fleet/application"
	fleetdomain "k8s-platform-backend/internal/fleet/domain"
	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

// MetricsProviderManager owns metrics-provider discovery, selection and
// persistence. Kubernetes clients remain a narrow transport dependency.
type MetricsProviderManager struct {
	transport *service.K8sService
	clusters  *fleetapp.Registry
}

func NewMetricsProviderManager(transport *service.K8sService, clusters *fleetapp.Registry) *MetricsProviderManager {
	return &MetricsProviderManager{transport: transport, clusters: clusters}
}

func (m *MetricsProviderManager) GetPrometheusInfo(ctx context.Context, clusterID uint64) (*kopsruntime.PrometheusInfo, error) {
	cluster, err := m.monitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return &kopsruntime.PrometheusInfo{URL: cluster.PrometheusURL, Status: kopsapp.PrometheusStatus(cluster.PrometheusStatus)}, nil
}

func (m *MetricsProviderManager) DetectAndUpdatePrometheus(ctx context.Context, clusterID uint64) (*kopsruntime.PrometheusInfo, error) {
	info, err := m.detectPrometheus(ctx, clusterID)
	if err != nil {
		_ = m.updateMonitorSource(ctx, clusterID, kopsapp.MonitorSourceMetricsServer, "", kopsapp.PrometheusStatusUnhealthy)
		return nil, err
	}

	status := kopsapp.PrometheusStatusHealthy
	if err := kopsruntime.CheckPrometheusHealth(ctx, info.URL); err != nil {
		status = kopsapp.PrometheusStatusUnhealthy
	}
	if err := m.updateMonitorSource(ctx, clusterID, kopsapp.MonitorSourcePrometheus, info.URL, status); err != nil {
		return nil, err
	}
	return &kopsruntime.PrometheusInfo{URL: info.URL, Status: status}, nil
}

func (m *MetricsProviderManager) Provider(ctx context.Context, clusterID uint64) (kopsapp.MetricsProvider, error) {
	cluster, err := m.monitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	source := kopsapp.MonitorSource(cluster.MonitorSource)
	if source == kopsapp.MonitorSourceAuto || source == "" {
		if _, detectErr := m.DetectAndUpdatePrometheus(ctx, clusterID); detectErr == nil {
			source = kopsapp.MonitorSourcePrometheus
		} else {
			source = kopsapp.MonitorSourceMetricsServer
		}
	}
	if source == kopsapp.MonitorSourcePrometheus {
		return kopsruntime.NewPrometheusProvider(m), nil
	}
	return kopsruntime.NewMetricsServerProvider(m.transport), nil
}

func (m *MetricsProviderManager) SwitchProvider(ctx context.Context, clusterID uint64, source kopsapp.MonitorSource) error {
	if source != kopsapp.MonitorSourcePrometheus && source != kopsapp.MonitorSourceMetricsServer && source != kopsapp.MonitorSourceAuto {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "未知的数据源类型")
	}
	if m == nil || m.clusters == nil {
		return kopsapp.ErrConflict
	}
	switch source {
	case kopsapp.MonitorSourcePrometheus:
		info, err := m.GetPrometheusInfo(ctx, clusterID)
		if err != nil {
			return err
		}
		return m.updateMonitorSource(ctx, clusterID, source, info.URL, info.Status)
	case kopsapp.MonitorSourceMetricsServer:
		return m.updateMonitorSource(ctx, clusterID, source, "", kopsapp.PrometheusStatusUnknown)
	case kopsapp.MonitorSourceAuto:
		_, err := m.DetectAndUpdatePrometheus(ctx, clusterID)
		return err
	default:
		return kopsapp.ErrInvalidParams
	}
}

func (m *MetricsProviderManager) HealthCheckAndSwitch(ctx context.Context, clusterID uint64) (kopsapp.MetricsProvider, error) {
	cluster, err := m.monitorSource(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	source := kopsapp.MonitorSource(cluster.MonitorSource)
	if source == kopsapp.MonitorSourcePrometheus || source == kopsapp.MonitorSourceAuto {
		if url := strings.TrimSpace(cluster.PrometheusURL); url != "" {
			if err := kopsruntime.CheckPrometheusHealth(ctx, url); err == nil {
				if source != kopsapp.MonitorSourcePrometheus || cluster.PrometheusStatus != string(kopsapp.PrometheusStatusHealthy) {
					_ = m.updateMonitorSource(ctx, clusterID, kopsapp.MonitorSourcePrometheus, url, kopsapp.PrometheusStatusHealthy)
				}
				return kopsruntime.NewPrometheusProvider(m), nil
			}
		}
		_ = m.updateMonitorSource(ctx, clusterID, kopsapp.MonitorSourceMetricsServer, cluster.PrometheusURL, kopsapp.PrometheusStatusUnhealthy)
	}
	return kopsruntime.NewMetricsServerProvider(m.transport), nil
}

func (m *MetricsProviderManager) detectPrometheus(ctx context.Context, clusterID uint64) (*kopsapp.PrometheusDetectionResult, error) {
	if m == nil || m.transport == nil {
		return nil, kopsapp.ErrConflict
	}
	client, err := m.transport.TypedClient(ctx, clusterID)
	if err != nil {
		return nil, kopsruntime.TranslateKopsRuntimeError(err)
	}
	for _, selector := range kopsapp.PrometheusDiscoverySelectors() {
		services, err := client.CoreV1().Services(metav1.NamespaceAll).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			continue
		}
		for index := range services.Items {
			candidate, ok := prometheusServiceCandidate(&services.Items[index])
			if !ok {
				continue
			}
			if err := kopsruntime.CheckPrometheusHealth(ctx, candidate.URL); err == nil {
				return &candidate, nil
			}
		}
	}
	return nil, fmt.Errorf("未检测到可用的 Prometheus 服务")
}

func prometheusServiceCandidate(serviceValue *corev1.Service) (kopsapp.PrometheusDetectionResult, bool) {
	if serviceValue == nil {
		return kopsapp.PrometheusDetectionResult{}, false
	}
	ports := make([]kopsapp.PrometheusServicePort, 0, len(serviceValue.Spec.Ports))
	for _, port := range serviceValue.Spec.Ports {
		ports = append(ports, kopsapp.PrometheusServicePort{Name: port.Name, Port: port.Port})
	}
	return kopsapp.BuildPrometheusDetectionResult(kopsapp.PrometheusServiceCandidate{
		Name: serviceValue.Name, Namespace: serviceValue.Namespace, ClusterIP: serviceValue.Spec.ClusterIP, Ports: ports,
	})
}

func (m *MetricsProviderManager) monitorSource(ctx context.Context, clusterID uint64) (*fleetdomain.Cluster, error) {
	if m == nil || m.clusters == nil {
		return nil, kopsapp.ErrConflict
	}
	cluster, err := m.clusters.MonitorSource(ctx, clusterID)
	if err != nil {
		return nil, metricsManagerError(err)
	}
	return cluster, nil
}

func (m *MetricsProviderManager) updateMonitorSource(ctx context.Context, clusterID uint64, source kopsapp.MonitorSource, url string, status kopsapp.PrometheusStatus) error {
	if m == nil || m.clusters == nil {
		return kopsapp.ErrConflict
	}
	return metricsManagerError(m.clusters.UpdateMonitorSource(ctx, clusterID, string(source), url, string(status)))
}

func metricsManagerError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fleetdomain.ErrValidation):
		return kopsapp.ErrInvalidParams
	case errors.Is(err, fleetdomain.ErrNotFound):
		return kopsapp.ErrNotFound
	case errors.Is(err, fleetdomain.ErrConflict):
		return kopsapp.ErrConflict
	default:
		return err
	}
}
