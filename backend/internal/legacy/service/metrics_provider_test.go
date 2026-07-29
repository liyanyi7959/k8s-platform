// metrics_provider_test.go 验证 Prometheus / metrics-server 双数据源适配核心逻辑。
package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

// prometheusMockResponse 构造 Prometheus API 成功响应。
func prometheusMockResponse(result any) map[string]any {
	return map[string]any{
		"status": "success",
		"data":   result,
	}
}

// TestPrometheusClient_Query 验证瞬时查询解析。
func TestPrometheusClient_Query(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := prometheusMockResponse(map[string]any{
			"resultType": "vector",
			"result": []map[string]any{
				{
					"metric": map[string]string{"node": "node-1"},
					"value":  []any{1234567890.0, "12.5"},
				},
			},
		})
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	pc := newPrometheusClient(ts.URL)
	res, err := pc.query(context.Background(), "up")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(res.Result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res.Result))
	}
	if res.Result[0].Metric["node"] != "node-1" {
		t.Fatalf("unexpected node label: %s", res.Result[0].Metric["node"])
	}
	if parsePrometheusValue(res.Result[0].Value[1]) != 12.5 {
		t.Fatalf("unexpected value")
	}
}

// TestPrometheusClient_QueryRange 验证范围查询解析。
func TestPrometheusClient_QueryRange(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query_range" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := prometheusMockResponse(map[string]any{
			"resultType": "matrix",
			"result": []map[string]any{
				{
					"metric": map[string]string{"node": "node-1"},
					"values": [][]any{
						{1234567890.0, "10.0"},
						{1234567950.0, "20.0"},
					},
				},
			},
		})
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	pc := newPrometheusClient(ts.URL)
	res, err := pc.queryRange(context.Background(), "up", time.Now().Add(-5*time.Minute), time.Now(), 30*time.Second)
	if err != nil {
		t.Fatalf("query_range failed: %v", err)
	}
	if len(res.Result) != 1 || len(res.Result[0].Values) != 2 {
		t.Fatalf("unexpected result count")
	}
}

// TestPrometheusProvider_RangeResultToPoints 验证范围查询结果转点。
func TestPrometheusProvider_RangeResultToPoints(t *testing.T) {
	p := &prometheusProvider{}
	res := prometheusQueryRangeResult{
		Result: []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		}{
			{
				Metric: map[string]string{"node": "node-1"},
				Values: [][]interface{}{
					{1234567890.0, "10.0"},
					{1234567950.0, "20.0"},
				},
			},
		},
	}
	points := p.rangeResultToPoints(res)
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].Value != 10.0 || points[1].Value != 20.0 {
		t.Fatalf("unexpected point values")
	}
}

// TestMetricsServerProvider_TrendUnsupported 验证 metrics-server 不支持趋势查询。
func TestMetricsServerProvider_TrendUnsupported(t *testing.T) {
	p := &metricsServerProvider{}
	ctx := context.Background()
	now := time.Now()
	if _, err := p.GetNodeMetricTrend(ctx, 1, "node-1", "cpu", now.Add(-time.Hour), now, time.Minute); err == nil {
		t.Fatal("expected error for metrics-server trend")
	}
	if _, err := p.GetPodMetricTrend(ctx, 1, "default", "pod-1", "memory", now.Add(-time.Hour), now, time.Minute); err == nil {
		t.Fatal("expected error for metrics-server trend")
	}
}

// TestProviderManager_SwitchProviderValidation 验证非法数据源切换被拒绝。
func TestProviderManager_SwitchProviderValidation(t *testing.T) {
	// 无 db 的 K8sService 应返回错误。
	svc := NewK8sService(nil, nil, 0)
	err := svc.SwitchProvider(context.Background(), 1, kopsapp.MonitorSource("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
}
