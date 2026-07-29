package application

import (
	"fmt"
	"strings"
)

// PrometheusServicePort is the minimal service-port information needed to
// construct an in-cluster Prometheus endpoint.
type PrometheusServicePort struct {
	Name string
	Port int32
}

// PrometheusServiceCandidate is a Kubernetes-independent service snapshot.
type PrometheusServiceCandidate struct {
	Name      string
	Namespace string
	ClusterIP string
	Ports     []PrometheusServicePort
}

// PrometheusDetectionResult is the stable monitoring endpoint selected by
// discovery before the runtime performs its health probe.
type PrometheusDetectionResult struct {
	ServiceName string
	Namespace   string
	ClusterIP   string
	URL         string
}

func PrometheusDiscoverySelectors() []string {
	return []string{
		"app.kubernetes.io/name=prometheus",
		"app=prometheus",
		"component=server",
	}
}

// BuildPrometheusDetectionResult returns false when no usable service port is
// present. ClusterIP is preferred; headless services use their cluster DNS.
func BuildPrometheusDetectionResult(candidate PrometheusServiceCandidate) (PrometheusDetectionResult, bool) {
	name := strings.TrimSpace(candidate.Name)
	namespace := strings.TrimSpace(candidate.Namespace)
	if name == "" || namespace == "" {
		return PrometheusDetectionResult{}, false
	}
	port := prometheusHTTPPort(candidate.Ports)
	if port == 0 {
		return PrometheusDetectionResult{}, false
	}
	host := strings.TrimSpace(candidate.ClusterIP)
	if host == "" || strings.EqualFold(host, "none") {
		host = fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace)
	}
	return PrometheusDetectionResult{
		ServiceName: name, Namespace: namespace, ClusterIP: strings.TrimSpace(candidate.ClusterIP),
		URL: fmt.Sprintf("http://%s:%d", host, port),
	}, true
}

func prometheusHTTPPort(ports []PrometheusServicePort) int32 {
	for _, port := range ports {
		if strings.Contains(strings.ToLower(strings.TrimSpace(port.Name)), "http") || port.Port == 9090 {
			return port.Port
		}
	}
	if len(ports) > 0 {
		return ports[0].Port
	}
	return 0
}
