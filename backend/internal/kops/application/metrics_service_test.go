package application

import (
	"context"
	"errors"
	"testing"
	"time"
)

type metricsRuntimeSpy struct {
	source string
	trend  MetricsTrendQuery
}

func (spy *metricsRuntimeSpy) NodeMetrics(context.Context, uint64) (any, error) { return nil, nil }
func (spy *metricsRuntimeSpy) PodMetrics(context.Context, uint64, string) (any, error) {
	return nil, nil
}
func (spy *metricsRuntimeSpy) Source(context.Context, uint64) (any, error) { return nil, nil }
func (spy *metricsRuntimeSpy) Detect(context.Context, uint64) (any, error) { return nil, nil }
func (spy *metricsRuntimeSpy) Switch(_ context.Context, _ uint64, source string) error {
	spy.source = source
	return nil
}
func (spy *metricsRuntimeSpy) Trend(_ context.Context, query MetricsTrendQuery) (any, error) {
	spy.trend = query
	return nil, nil
}
func (spy *metricsRuntimeSpy) HealthCheck(context.Context, uint64) (any, error) { return nil, nil }
func TestMetricsServiceValidatesRuntimeCommands(t *testing.T) {
	spy := &metricsRuntimeSpy{}
	service := NewMetricsService(spy)
	if err := service.Switch(context.Background(), 2, " Prometheus "); err != nil || spy.source != "prometheus" {
		t.Fatalf("Switch() source=%q err=%v", spy.source, err)
	}
	start := time.Unix(100, 0)
	if _, err := service.Trend(context.Background(), MetricsTrendQuery{ClusterID: 2, Target: "pod", Name: "api", Namespace: "ops", Metric: "cpu", Start: start, End: start.Add(time.Minute), Step: time.Second}); err != nil {
		t.Fatalf("Trend() error=%v", err)
	}
	if spy.trend.Target != "pod" {
		t.Fatalf("trend=%#v", spy.trend)
	}
	if _, err := service.Trend(context.Background(), MetricsTrendQuery{ClusterID: 2, Target: "node", Name: "n", Metric: "disk", Start: start, End: start, Step: time.Second}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid trend error=%v", err)
	}
}
