package kops

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

func prometheusMockResponse(result any) map[string]any {
	return map[string]any{"status": "success", "data": result}
}

func TestPrometheusClientQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/query" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		_ = json.NewEncoder(writer).Encode(prometheusMockResponse(map[string]any{
			"resultType": "vector",
			"result":     []map[string]any{{"metric": map[string]string{"node": "node-1"}, "value": []any{1234567890.0, "12.5"}}},
		}))
	}))
	t.Cleanup(server.Close)

	result, err := newPrometheusClient(server.URL).query(context.Background(), "up")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(result.Result) != 1 || result.Result[0].Metric["node"] != "node-1" {
		t.Fatalf("unexpected result: %+v", result.Result)
	}
	if parsePrometheusValue(result.Result[0].Value[1]) != 12.5 {
		t.Fatal("unexpected query value")
	}
}

func TestPrometheusClientQueryRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/query_range" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		_ = json.NewEncoder(writer).Encode(prometheusMockResponse(map[string]any{
			"resultType": "matrix",
			"result":     []map[string]any{{"metric": map[string]string{"node": "node-1"}, "values": [][]any{{1234567890.0, "10.0"}, {1234567950.0, "20.0"}}}},
		}))
	}))
	t.Cleanup(server.Close)

	result, err := newPrometheusClient(server.URL).queryRange(context.Background(), "up", time.Now().Add(-5*time.Minute), time.Now(), 30*time.Second)
	if err != nil {
		t.Fatalf("query range failed: %v", err)
	}
	if len(result.Result) != 1 || len(result.Result[0].Values) != 2 {
		t.Fatalf("unexpected result: %+v", result.Result)
	}
}

func TestRangeResultToMetricPoints(t *testing.T) {
	result := prometheusQueryRangeResult{Result: []struct {
		Metric map[string]string `json:"metric"`
		Values [][]interface{}   `json:"values"`
	}{{
		Metric: map[string]string{"node": "node-1"},
		Values: [][]interface{}{{1234567890.0, "10.0"}, {1234567950.0, "20.0"}},
	}}}
	points := rangeResultToMetricPoints(result)
	if len(points) != 2 || points[0].Value != 10 || points[1].Value != 20 {
		t.Fatalf("unexpected points: %+v", points)
	}
}

func TestMetricsServerProviderTrendUnsupported(t *testing.T) {
	provider := &metricsServerProvider{}
	now := time.Now()
	if _, err := provider.GetNodeMetricTrend(context.Background(), 1, "node-1", "cpu", now.Add(-time.Hour), now, time.Minute); err == nil {
		t.Fatal("expected node trend error")
	}
	if _, err := provider.GetPodMetricTrend(context.Background(), 1, "default", "pod-1", "memory", now.Add(-time.Hour), now, time.Minute); err == nil {
		t.Fatal("expected pod trend error")
	}
}

func TestMetricsProviderManagerRejectsUnknownSource(t *testing.T) {
	manager := NewMetricsProviderManager(nil, nil)
	err := manager.SwitchProvider(context.Background(), 1, kopsapp.MonitorSource("invalid"))
	if !errors.Is(err, kopsapp.ErrInvalidParams) {
		t.Fatalf("SwitchProvider invalid error = %v, want invalid params", err)
	}
}
