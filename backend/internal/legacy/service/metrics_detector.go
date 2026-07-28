// metrics_detector.go 负责自动检测集群内可用的 Prometheus 服务。
package service

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// prometheusDetectionResult 保存检测到的 Prometheus 服务元数据。
type prometheusDetectionResult struct {
	ServiceName string
	Namespace   string
	ClusterIP   string
	URL         string
}

// DetectPrometheus 在指定集群中自动发现可用的 Prometheus 服务。
// 依次尝试多种常见 label selector，返回第一个可连通的实例。
func (s *K8sService) DetectPrometheus(ctx context.Context, clusterID uint64) (*prometheusDetectionResult, error) {
	client, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 常见 Prometheus 部署的 label selector。
	selectors := []string{
		"app.kubernetes.io/name=prometheus",
		"app=prometheus",
		"component=server",
	}

	var candidate *prometheusDetectionResult
	for _, sel := range selectors {
		list, err := client.CoreV1().Services(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
			LabelSelector: sel,
		})
		if err != nil {
			continue
		}
		for _, svc := range list.Items {
			info := s.serviceToPrometheusInfo(&svc)
			if info == nil {
				continue
			}
			if err := s.checkPrometheusHealth(ctx, info.URL); err == nil {
				candidate = info
				break
			}
		}
		if candidate != nil {
			break
		}
	}

	if candidate == nil {
		return nil, fmt.Errorf("未检测到可用的 Prometheus 服务")
	}
	return candidate, nil
}

// serviceToPrometheusInfo 将 K8s Service 转换为 prometheusDetectionResult。
func (s *K8sService) serviceToPrometheusInfo(svc *corev1.Service) *prometheusDetectionResult {
	if svc == nil {
		return nil
	}
	port := s.findPrometheusPort(svc)
	if port == 0 {
		return nil
	}

	// 优先使用 ClusterIP，若未分配则使用服务 DNS 名称。
	host := svc.Spec.ClusterIP
	if host == "" || host == "None" {
		host = fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace)
	}

	return &prometheusDetectionResult{
		ServiceName: svc.Name,
		Namespace:   svc.Namespace,
		ClusterIP:   svc.Spec.ClusterIP,
		URL:         fmt.Sprintf("http://%s:%d", host, port),
	}
}

// findPrometheusPort 查找 Prometheus HTTP 端口。
func (s *K8sService) findPrometheusPort(svc *corev1.Service) int32 {
	for _, p := range svc.Spec.Ports {
		if strings.Contains(strings.ToLower(p.Name), "http") || p.Port == 9090 {
			return p.Port
		}
	}
	if len(svc.Spec.Ports) > 0 {
		return svc.Spec.Ports[0].Port
	}
	return 0
}

// checkPrometheusHealth 校验 Prometheus 是否可访问且能返回数据。
func (s *K8sService) checkPrometheusHealth(ctx context.Context, url string) error {
	pc := newPrometheusClient(url)
	return pc.health(ctx)
}

