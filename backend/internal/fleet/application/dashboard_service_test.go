package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"k8s-platform-backend/internal/fleet/ports"
)

type memoryDashboardCache struct{ values map[string][]byte }

func (c *memoryDashboardCache) Enabled() bool { return c != nil }
func (c *memoryDashboardCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	value, found := c.values[key]
	return append([]byte(nil), value...), found, nil
}
func (c *memoryDashboardCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if c.values == nil { c.values = map[string][]byte{} }
	c.values[key] = append([]byte(nil), value...)
	return nil
}

func TestDashboardCertificateRisksPersistAcrossInstances(t *testing.T) {
	cache := &memoryDashboardCache{}
	ctx := context.Background()
	written := []map[string]any{{"key": "apiserver", "name": "API Server certificate", "status": "warn", "days_left": 12}}
	first := NewDashboardService(nil, nil, cache)
	first.setCachedCertificateRisks(ctx, 16, written, 10*time.Minute)
	second := NewDashboardService(nil, nil, cache)
	loaded, ok := second.getCachedCertificateRisks(ctx, 16)
	if !ok || len(loaded) != 1 || loaded[0]["key"] != "apiserver" || loaded[0]["status"] != "warn" {
		t.Fatalf("persisted cache = (%v, %#v)", ok, loaded)
	}
	loaded[0]["status"] = "mutated"
	again, ok := second.getCachedCertificateRisks(ctx, 16)
	if !ok || again[0]["status"] != "warn" { t.Fatalf("cache values must be cloned: %#v", again) }
}

func TestDashboardCertificateRisksIgnoreExpiredPersistentEntry(t *testing.T) {
	cache := &memoryDashboardCache{}
	payload, err := json.Marshal(cachedCertificateRisks{Data: []map[string]any{{"key": "expired"}}, ExpiresAt: time.Now().Add(-time.Minute)})
	if err != nil { t.Fatal(err) }
	if err := cache.Set(context.Background(), certificateCacheKey(16), payload, time.Minute); err != nil { t.Fatal(err) }
	service := NewDashboardService(nil, nil, cache)
	if loaded, ok := service.getCachedCertificateRisks(context.Background(), 16); ok || len(loaded) > 0 { t.Fatalf("expired entry = %#v", loaded) }
}

func TestAppendResourceTrendSampleKeepsReal24HourWindow(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	samples := []resourceTrendSample{{Timestamp: now.Add(-25*time.Hour), CPU: 99, Memory: 99}, {Timestamp: now.Add(-time.Hour), CPU: 12.3, Memory: 45.6}}
	got := appendResourceTrendSample(samples, resourceTrendSample{Timestamp: now, CPU: 13.4, Memory: 46.7})
	if len(got) != 2 || got[0].CPU != 12.3 || got[1].Memory != 46.7 { t.Fatalf("trend = %#v", got) }
}

func TestAppendResourceTrendSampleReplacesSameMinute(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 30, 0, time.UTC)
	got := appendResourceTrendSample([]resourceTrendSample{{Timestamp: now.Add(-30*time.Second), CPU: 10, Memory: 20}}, resourceTrendSample{Timestamp: now, CPU: 11, Memory: 21})
	if len(got) != 1 || got[0].CPU != 11 || got[0].Memory != 21 { t.Fatalf("trend = %#v", got) }
}

func TestTopWorkloadsAndClusterUsageRemainProtocolCompatible(t *testing.T) {
	workloads := topWorkloads([]ports.DashboardWorkload{
		{Name: "api", Namespace: "prod", Kind: "Deployment", Replicas: 3, Ready: 2},
		{Name: "agent", Namespace: "ops", Kind: "DaemonSet", Replicas: 5, Ready: 5},
	})
	if len(workloads) != 2 || workloads[0]["name"] != "agent" || workloads[0]["replicas"] != int32(5) || workloads[1]["ready"] != int32(2) { t.Fatalf("workloads = %#v", workloads) }
	usage, available := calculateClusterUsage([]ports.DashboardNode{{Name: "node-a", IP: "10.0.0.11", CPUAllocatableMilli: 2000, MemoryAllocatable: 4 << 30}, {Name: "node-b", CPUAllocatableMilli: 2000, MemoryAllocatable: 4 << 30}}, []ports.DashboardNodeMetric{{NodeName: "node-a", CPUMilli: 1000, MemoryBytes: 2 << 30}})
	if !available || usage.CPU != 50 || usage.Memory != 50 || len(usage.Nodes) != 1 || usage.Nodes[0].IP != "10.0.0.11" { t.Fatalf("usage = (%+v, %v)", usage, available) }
}

func TestCertificateRiskPolicyPreservesRiskOrdering(t *testing.T) {
	now := time.Now().UTC()
	items := certificateRiskMaps(ports.CertificateSnapshot{
		APIServer: ports.CertificateObservation{Available: true, CommonName: "api", NotBefore: now.Add(-time.Hour), NotAfter: now.Add(5 * 24 * time.Hour)},
		ClusterCA: ports.CertificateObservation{Available: true, CommonName: "ca", NotBefore: now.Add(-time.Hour), NotAfter: now.Add(60 * 24 * time.Hour)},
	})
	if len(items) != 5 || items[0]["key"] != "apiserver" || items[0]["status"] != "critical" { t.Fatalf("certificate risks = %#v", items) }
}
