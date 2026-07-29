package application

import (
	"testing"
	"time"
)

func TestShouldRunMetricsHealthCheck(t *testing.T) {
	now := time.Date(2026, time.July, 30, 10, 0, 0, 0, time.UTC)

	if !ShouldRunMetricsHealthCheck(time.Time{}, now) {
		t.Fatal("first health check must be due")
	}
	if ShouldRunMetricsHealthCheck(now, now.Add(MetricsHealthCheckInterval-time.Nanosecond)) {
		t.Fatal("health check must remain throttled before the interval")
	}
	if ShouldRunMetricsHealthCheck(now, now.Add(MetricsHealthCheckInterval)) {
		t.Fatal("health check must remain throttled at the interval boundary")
	}
	if !ShouldRunMetricsHealthCheck(now, now.Add(MetricsHealthCheckInterval+time.Nanosecond)) {
		t.Fatal("health check must be due after the interval")
	}
	if ShouldRunMetricsHealthCheck(now, now.Add(-time.Second)) {
		t.Fatal("health check must remain throttled when the clock moves backwards")
	}
}
