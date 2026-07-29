package application

import "testing"

func TestBuildPrometheusDetectionResult(t *testing.T) {
	result, ok := BuildPrometheusDetectionResult(PrometheusServiceCandidate{Name: "prometheus", Namespace: "monitoring", ClusterIP: "10.96.12.2", Ports: []PrometheusServicePort{{Name: "web", Port: 8080}, {Name: "http", Port: 9090}}})
	if !ok || result.URL != "http://10.96.12.2:9090" {
		t.Fatalf("result = %#v, ok=%t", result, ok)
	}
	headless, ok := BuildPrometheusDetectionResult(PrometheusServiceCandidate{Name: "prometheus", Namespace: "monitoring", ClusterIP: "None", Ports: []PrometheusServicePort{{Port: 9090}}})
	if !ok || headless.URL != "http://prometheus.monitoring.svc.cluster.local:9090" {
		t.Fatalf("headless result = %#v, ok=%t", headless, ok)
	}
	if _, ok := BuildPrometheusDetectionResult(PrometheusServiceCandidate{Name: "prometheus", Namespace: "monitoring"}); ok {
		t.Fatal("candidate without ports should be rejected")
	}
}

func TestPrometheusDiscoverySelectorsReturnsCopy(t *testing.T) {
	first := PrometheusDiscoverySelectors()
	first[0] = "mutated"
	if PrometheusDiscoverySelectors()[0] == "mutated" {
		t.Fatal("selector list must not share mutable storage")
	}
}
