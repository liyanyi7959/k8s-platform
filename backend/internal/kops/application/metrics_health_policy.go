package application

import "time"

// MetricsHealthCheckInterval limits active metrics-provider probes for one
// cluster. The runtime owns the last-check cache and clock; this package only
// defines the deterministic eligibility rule.
const MetricsHealthCheckInterval = 30 * time.Second

// ShouldRunMetricsHealthCheck reports whether a health probe is due. A zero
// last-check timestamp represents a cluster that has not been checked yet.
// The interval boundary remains exclusive to preserve the existing cadence:
// probes resume only after a full interval has elapsed.
func ShouldRunMetricsHealthCheck(lastCheckedAt, now time.Time) bool {
	return lastCheckedAt.IsZero() || now.After(lastCheckedAt.Add(MetricsHealthCheckInterval))
}
