package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"k8s-platform-backend/internal/fleet/domain"
	"k8s-platform-backend/internal/fleet/ports"
)

const (
	dashboardOverviewTTL  = 120 * time.Second
	dashboardCertTTL      = 10 * time.Minute
	dashboardCertRetryTTL = 45 * time.Second
)

// DashboardService owns the Fleet dashboard policy.  Kubernetes collection and
// cache I/O are supplied by ports so neither operational detail leaks into the
// application layer.
type DashboardService struct {
	registry *Registry
	runtime  ports.DashboardRuntime
	cache    ports.DashboardCache

	certMu        sync.Mutex
	certCache     map[uint64]cachedCertificateRisks
	overviewGroup singleflight.Group
	trendMu       sync.Mutex
	trendCache    map[string][]resourceTrendSample
}

type cachedOverview struct {
	Data      map[string]any `json:"data"`
	ExpiresAt time.Time      `json:"expires_at"`
}

type cachedCertificateRisks struct {
	Data      []map[string]any `json:"data"`
	ExpiresAt time.Time        `json:"expires_at"`
}

type resourceTrendSample struct {
	Timestamp time.Time `json:"timestamp"`
	CPU       float64   `json:"cpu"`
	Memory    float64   `json:"memory"`
}

type trendDataCache struct {
	Samples []resourceTrendSample `json:"samples"`
}

func NewDashboardService(registry *Registry, runtime ports.DashboardRuntime, cache ports.DashboardCache) *DashboardService {
	return &DashboardService{
		registry:   registry,
		runtime:    runtime,
		cache:      cache,
		certCache:  make(map[uint64]cachedCertificateRisks),
		trendCache: make(map[string][]resourceTrendSample),
	}
}

func (s *DashboardService) GetClusterOverview(ctx context.Context, clusterID uint64) (map[string]any, error) {
	if s == nil || s.registry == nil || s.runtime == nil {
		return nil, domain.ErrRuntime
	}
	if clusterID == 0 {
		return nil, domain.ErrValidation
	}
	value, err, _ := s.overviewGroup.Do(fmt.Sprintf("cluster:%d", clusterID), func() (any, error) {
		return s.getClusterOverview(ctx, clusterID)
	})
	if err != nil {
		return nil, err
	}
	overview, ok := value.(map[string]any)
	if !ok {
		return nil, domain.ErrRuntime
	}
	return overview, nil
}

func (s *DashboardService) getClusterOverview(ctx context.Context, clusterID uint64) (map[string]any, error) {
	if cached, ok := s.getCachedOverview(ctx, clusterID); ok {
		return cached, nil
	}
	cluster, err := s.registry.Get(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.runtime.CollectDashboard(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	nodeReady := 0
	for _, node := range snapshot.Nodes {
		if node.Ready {
			nodeReady++
		}
	}
	pods, namespacePods, failedPods, unscheduledPods := summarizePods(snapshot.Pods)
	usage, metricsAvailable := calculateClusterUsage(snapshot.Nodes, snapshot.Metrics)

	labels24h := make([]string, 0)
	timestamps24h := make([]string, 0)
	cpu24h := make([]float64, 0)
	memory24h := make([]float64, 0)
	nodeTrends := make([]map[string]any, 0, len(usage.Nodes))
	if metricsAvailable {
		sampledAt := time.Now()
		clusterTrend := s.recordResourceTrend(ctx, clusterID, "cluster", resourceTrendSample{Timestamp: sampledAt, CPU: usage.CPU, Memory: usage.Memory})
		labels24h, timestamps24h, cpu24h, memory24h = trendColumns(clusterTrend)
		for _, nodeUsage := range usage.Nodes {
			nodeTrend := s.recordResourceTrend(ctx, clusterID, "node:"+nodeUsage.Name, resourceTrendSample{Timestamp: sampledAt, CPU: nodeUsage.CPU, Memory: nodeUsage.Memory})
			labels, timestamps, cpu, memory := trendColumns(nodeTrend)
			nodeTrends = append(nodeTrends, map[string]any{
				"name": nodeUsage.Name, "ip": nodeUsage.IP,
				"labels": labels, "timestamps": timestamps, "cpu": cpu, "memory": memory, "sample_count": len(labels),
			})
		}
	}
	sort.Slice(nodeTrends, func(i, j int) bool { return fmt.Sprint(nodeTrends[i]["name"]) < fmt.Sprint(nodeTrends[j]["name"]) })

	out := map[string]any{
		"cluster": map[string]any{
			"id": cluster.ID, "name": cluster.Name, "status": cluster.Status,
			"api_ok": snapshot.APIOK, "k8s_version": snapshot.K8sVersion,
		},
		"stats": map[string]any{
			"nodes": map[string]any{"total": len(snapshot.Nodes), "ready": nodeReady},
			"pods": map[string]any{
				"total": pods.Total, "running": pods.Running, "pending": pods.Pending,
				"failed": pods.Failed, "succeeded": pods.Succeeded,
			},
			"workloads": workloadCounts(snapshot.Workloads),
			"cpu":       map[string]any{"used_percent": usage.CPU},
			"memory":    map[string]any{"used_percent": usage.Memory},
		},
		"charts": map[string]any{
			"cpu_memory_24h": map[string]any{
				"labels": labels24h, "timestamps": timestamps24h, "cpu": cpu24h, "memory": memory24h,
				"scope": "cluster", "sample_count": len(labels24h),
			},
			"node_cpu_memory_24h": nodeTrends,
			"pod_phase":           map[string]any{"running": pods.Running, "pending": pods.Pending, "failed": pods.Failed, "succeeded": pods.Succeeded},
			"namespace_pods_top":  namespaceTop(namespacePods),
			"node_ready":          map[string]any{"ready": nodeReady, "total": len(snapshot.Nodes)},
		},
		"anomalies": map[string]any{"failed_pods": failedPods, "unscheduled_pods": unscheduledPods},
		"events":    eventMaps(snapshot.RecentEvents),
		"meta": map[string]any{
			"source": "k8s-api", "updated_at": time.Now().Format(time.RFC3339), "cached": false,
			"metrics_available": metricsAvailable, "metrics_source": "metrics.k8s.io/v1beta1 nodes", "metrics_scope": "cluster",
			"metrics_basis": "sum(sampled node usage) / sum(sampled node allocatable)",
		},
	}
	if top := topWorkloads(snapshot.Workloads); len(top) > 0 {
		out["top_workloads"] = top
	}
	if risks, ok := s.getCachedCertificateRisks(ctx, clusterID); ok && len(risks) > 0 {
		out["risks"] = map[string]any{"certificates": risks}
	}
	s.setCachedOverview(ctx, clusterID, out)
	return out, nil
}

func (s *DashboardService) GetClusterCertificateRisks(ctx context.Context, clusterID uint64) ([]map[string]any, error) {
	if s == nil || s.registry == nil || s.runtime == nil {
		return nil, domain.ErrRuntime
	}
	if clusterID == 0 {
		return nil, domain.ErrValidation
	}
	if _, err := s.registry.Get(ctx, clusterID); err != nil {
		return nil, err
	}
	if cached, ok := s.getCachedCertificateRisks(ctx, clusterID); ok {
		return cached, nil
	}
	apiCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	apiOK := s.runtime.CheckDashboardAPI(apiCtx, clusterID)
	cancel()
	certCtx, certCancel := context.WithTimeout(ctx, 5*time.Second)
	defer certCancel()
	snapshot, cacheable := s.runtime.ProbeCertificates(certCtx, clusterID, apiOK)
	risks := certificateRiskMaps(snapshot)
	if cacheable {
		s.setCachedCertificateRisks(ctx, clusterID, risks, dashboardCertTTL)
	} else if len(risks) > 0 {
		s.setCachedCertificateRisks(ctx, clusterID, risks, dashboardCertRetryTTL)
	}
	return risks, nil
}

func (s *DashboardService) getCachedOverview(ctx context.Context, clusterID uint64) (map[string]any, bool) {
	if s.cache == nil || !s.cache.Enabled() {
		return nil, false
	}
	readCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	value, found, err := s.cache.Get(readCtx, fmt.Sprintf("dashboard:overview:cluster:%d", clusterID))
	cancel()
	if err != nil || !found || len(value) == 0 {
		return nil, false
	}
	var cached cachedOverview
	if json.Unmarshal(value, &cached) != nil || !time.Now().Before(cached.ExpiresAt) {
		return nil, false
	}
	return cached.Data, true
}

func (s *DashboardService) setCachedOverview(ctx context.Context, clusterID uint64, data map[string]any) {
	if s.cache == nil || !s.cache.Enabled() {
		return
	}
	value, err := json.Marshal(cachedOverview{Data: data, ExpiresAt: time.Now().Add(dashboardOverviewTTL)})
	if err != nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	_ = s.cache.Set(writeCtx, fmt.Sprintf("dashboard:overview:cluster:%d", clusterID), value, dashboardOverviewTTL)
	cancel()
}

func (s *DashboardService) getCachedCertificateRisks(ctx context.Context, clusterID uint64) ([]map[string]any, bool) {
	if s == nil || clusterID == 0 {
		return nil, false
	}
	s.certMu.Lock()
	entry, found := s.certCache[clusterID]
	if found && time.Now().After(entry.ExpiresAt) {
		delete(s.certCache, clusterID)
		found = false
	}
	s.certMu.Unlock()
	if found {
		return cloneMapSlice(entry.Data), true
	}
	if s.cache == nil || !s.cache.Enabled() {
		return nil, false
	}
	readCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	value, found, err := s.cache.Get(readCtx, certificateCacheKey(clusterID))
	cancel()
	if err != nil || !found || len(value) == 0 {
		return nil, false
	}
	var cached cachedCertificateRisks
	if json.Unmarshal(value, &cached) != nil || time.Now().After(cached.ExpiresAt) {
		return nil, false
	}
	s.certMu.Lock()
	s.certCache[clusterID] = cached
	s.certMu.Unlock()
	return cloneMapSlice(cached.Data), true
}

func (s *DashboardService) setCachedCertificateRisks(ctx context.Context, clusterID uint64, data []map[string]any, ttl time.Duration) {
	if s == nil || clusterID == 0 {
		return
	}
	entry := cachedCertificateRisks{Data: cloneMapSlice(data), ExpiresAt: time.Now().Add(ttl)}
	s.certMu.Lock()
	s.certCache[clusterID] = entry
	s.certMu.Unlock()
	if s.cache == nil || !s.cache.Enabled() {
		return
	}
	value, err := json.Marshal(entry)
	if err != nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	_ = s.cache.Set(writeCtx, certificateCacheKey(clusterID), value, ttl)
	cancel()
}

func certificateCacheKey(clusterID uint64) string {
	return fmt.Sprintf("dashboard:certificate-risks:v1:cluster:%d", clusterID)
}

func cloneMapSlice(items []map[string]any) []map[string]any {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			cloned = append(cloned, nil)
			continue
		}
		duplicate := make(map[string]any, len(item))
		for key, value := range item {
			duplicate[key] = value
		}
		cloned = append(cloned, duplicate)
	}
	return cloned
}

func (s *DashboardService) recordResourceTrend(ctx context.Context, clusterID uint64, scope string, sample resourceTrendSample) []resourceTrendSample {
	s.trendMu.Lock()
	defer s.trendMu.Unlock()
	trendID := fmt.Sprintf("%d:%s", clusterID, scope)
	samples := append([]resourceTrendSample(nil), s.trendCache[trendID]...)
	cacheKey := fmt.Sprintf("dashboard:trend:v3:%d:%s", clusterID, scope)
	if len(samples) == 0 && s.cache != nil && s.cache.Enabled() {
		readCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		value, found, _ := s.cache.Get(readCtx, cacheKey)
		cancel()
		if found {
			var cached trendDataCache
			if json.Unmarshal(value, &cached) == nil {
				samples = cached.Samples
			}
		}
	}
	samples = appendResourceTrendSample(samples, sample)
	s.trendCache[trendID] = append([]resourceTrendSample(nil), samples...)
	if s.cache != nil && s.cache.Enabled() {
		if value, err := json.Marshal(trendDataCache{Samples: samples}); err == nil {
			writeCtx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			_ = s.cache.Set(writeCtx, cacheKey, value, 25*time.Hour)
			cancel()
		}
	}
	return samples
}

func appendResourceTrendSample(samples []resourceTrendSample, sample resourceTrendSample) []resourceTrendSample {
	cutoff := sample.Timestamp.Add(-24 * time.Hour)
	filtered := make([]resourceTrendSample, 0, len(samples)+1)
	for _, existing := range samples {
		if existing.Timestamp.Before(cutoff) || existing.Timestamp.After(sample.Timestamp.Add(5*time.Minute)) || existing.CPU < 0 || existing.CPU > 100 || existing.Memory < 0 || existing.Memory > 100 {
			continue
		}
		filtered = append(filtered, existing)
	}
	if len(filtered) > 0 && sample.Timestamp.Sub(filtered[len(filtered)-1].Timestamp) < time.Minute {
		filtered[len(filtered)-1] = sample
	} else {
		filtered = append(filtered, sample)
	}
	return filtered
}

func trendColumns(samples []resourceTrendSample) ([]string, []string, []float64, []float64) {
	labels, timestamps := make([]string, 0, len(samples)), make([]string, 0, len(samples))
	cpu, memory := make([]float64, 0, len(samples)), make([]float64, 0, len(samples))
	for _, sample := range samples {
		labels = append(labels, sample.Timestamp.Format("01-02 15:04"))
		timestamps = append(timestamps, sample.Timestamp.Format(time.RFC3339))
		cpu, memory = append(cpu, sample.CPU), append(memory, sample.Memory)
	}
	return labels, timestamps, cpu, memory
}

type podCounts struct{ Total, Running, Pending, Failed, Succeeded int }

func summarizePods(items []ports.DashboardPod) (podCounts, map[string]int, []map[string]any, []map[string]any) {
	counts := podCounts{}
	namespaces := map[string]int{}
	failed, unscheduled := make([]map[string]any, 0, 10), make([]map[string]any, 0, 10)
	for _, pod := range items {
		counts.Total++
		switch pod.Phase {
		case "Running":
			counts.Running++
		case "Pending":
			counts.Pending++
			if pod.NodeName == "" && len(unscheduled) < 10 {
				unscheduled = append(unscheduled, map[string]any{"name": pod.Name, "namespace": pod.Namespace, "reason": "Unschedulable"})
			}
		case "Failed":
			counts.Failed++
			if len(failed) < 10 {
				failed = append(failed, map[string]any{"name": pod.Name, "namespace": pod.Namespace, "reason": failureReason(pod)})
			}
		case "Succeeded":
			counts.Succeeded++
		}
		if pod.Namespace != "" {
			namespaces[pod.Namespace]++
		}
		if pod.Phase != "Failed" && len(failed) < 10 {
			if reason := unhealthyPodReason(pod); reason != "" {
				failed = append(failed, map[string]any{"name": pod.Name, "namespace": pod.Namespace, "reason": reason})
			}
		}
	}
	return counts, namespaces, failed, unscheduled
}

func failureReason(pod ports.DashboardPod) string {
	if strings.TrimSpace(pod.WaitingReason) != "" {
		return pod.WaitingReason
	}
	if strings.TrimSpace(pod.TerminatedReason) != "" {
		return pod.TerminatedReason
	}
	return pod.Phase
}

func unhealthyPodReason(pod ports.DashboardPod) string {
	if pod.Deleting || pod.Phase == "Succeeded" {
		return ""
	}
	if reason := strings.TrimSpace(pod.WaitingReason); reason != "" && reason != "ContainerCreating" && reason != "PodInitializing" {
		return reason
	}
	if pod.TerminatedExitCode != 0 {
		if reason := strings.TrimSpace(pod.TerminatedReason); reason != "" {
			return reason
		}
		return "ExitError"
	}
	if pod.Phase == "Failed" {
		return "Failed"
	}
	return ""
}

func workloadCounts(items []ports.DashboardWorkload) map[string]any {
	counts := map[string]any{"deployments": 0, "statefulsets": 0, "daemonsets": 0}
	for _, item := range items {
		switch item.Kind {
		case "Deployment":
			counts["deployments"] = counts["deployments"].(int) + 1
		case "StatefulSet":
			counts["statefulsets"] = counts["statefulsets"].(int) + 1
		case "DaemonSet":
			counts["daemonsets"] = counts["daemonsets"].(int) + 1
		}
	}
	return counts
}

func topWorkloads(items []ports.DashboardWorkload) []map[string]any {
	items = append([]ports.DashboardWorkload(nil), items...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Replicas > items[j].Replicas })
	if len(items) > 5 {
		items = items[:5]
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{"name": item.Name, "namespace": item.Namespace, "kind": item.Kind, "replicas": item.Replicas, "ready": item.Ready})
	}
	return result
}

func namespaceTop(counts map[string]int) []map[string]any {
	type pair struct {
		Namespace string
		Pods      int
	}
	pairs := make([]pair, 0, len(counts))
	for namespace, pods := range counts {
		pairs = append(pairs, pair{Namespace: namespace, Pods: pods})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Pods != pairs[j].Pods {
			return pairs[i].Pods > pairs[j].Pods
		}
		return pairs[i].Namespace < pairs[j].Namespace
	})
	if len(pairs) > 10 {
		pairs = pairs[:10]
	}
	result := make([]map[string]any, 0, len(pairs))
	for _, item := range pairs {
		result = append(result, map[string]any{"namespace": item.Namespace, "pods": item.Pods})
	}
	return result
}

func eventMaps(events []ports.DashboardEvent) []map[string]any {
	result := make([]map[string]any, 0, len(events))
	for _, event := range events {
		if event.LastTimestamp.IsZero() || time.Since(event.LastTimestamp) > time.Hour || len(result) >= 20 {
			continue
		}
		result = append(result, map[string]any{
			"type": event.Type, "reason": event.Reason, "message": event.Message, "namespace": event.Namespace,
			"lastTimestamp": event.LastTimestamp.Format(time.RFC3339), "count": event.Count,
			"involvedObject": fmt.Sprintf("%s/%s", event.InvolvedKind, event.InvolvedName),
		})
	}
	return result
}

type clusterUsage struct {
	CPU, Memory float64
	Nodes       []nodeUsage
}
type nodeUsage struct {
	Name, IP    string
	CPU, Memory float64
}

func calculateClusterUsage(nodes []ports.DashboardNode, metrics []ports.DashboardNodeMetric) (clusterUsage, bool) {
	byName := make(map[string]ports.DashboardNode, len(nodes))
	for _, node := range nodes {
		byName[node.Name] = node
	}
	var usedCPU, usedMemory, allocCPU, allocMemory int64
	usages := make([]nodeUsage, 0, len(metrics))
	for _, metric := range metrics {
		node, found := byName[metric.NodeName]
		if !found || node.CPUAllocatableMilli <= 0 || node.MemoryAllocatable <= 0 {
			continue
		}
		if metric.CPUMilli < 0 || metric.MemoryBytes < 0 {
			continue
		}
		usedCPU += metric.CPUMilli
		usedMemory += metric.MemoryBytes
		allocCPU += node.CPUAllocatableMilli
		allocMemory += node.MemoryAllocatable
		usages = append(usages, nodeUsage{Name: node.Name, IP: node.IP, CPU: usagePercent(metric.CPUMilli, node.CPUAllocatableMilli), Memory: usagePercent(metric.MemoryBytes, node.MemoryAllocatable)})
	}
	if allocCPU <= 0 || allocMemory <= 0 || usedCPU < 0 || usedMemory < 0 {
		return clusterUsage{}, false
	}
	sort.Slice(usages, func(i, j int) bool { return usages[i].Name < usages[j].Name })
	return clusterUsage{CPU: usagePercent(usedCPU, allocCPU), Memory: usagePercent(usedMemory, allocMemory), Nodes: usages}, true
}

func usagePercent(used, allocatable int64) float64 {
	value := float64(used) * 100 / float64(allocatable)
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	return float64(int(value*10+0.5)) / 10
}

type certificateRisk struct {
	Key, Name, Component, Purpose, Status string
	NotBefore, NotAfter                   *time.Time
	DaysLeft                              *int
}

func certificateRiskMaps(snapshot ports.CertificateSnapshot) []map[string]any {
	now := time.Now().UTC()
	unknown := func(key, name, component, purpose string) certificateRisk {
		return certificateRisk{Key: key, Name: name, Component: component, Purpose: purpose, Status: "unknown"}
	}
	items := []certificateRisk{
		unknown("apiserver", "API Server 证书", "API Server", "集群控制面入口（https://kube-apiserver）"),
		unknown("cluster_ca", "集群 CA 证书", "Cluster CA", "API Server 信任链根证书（kubeconfig CA）"),
		unknown("etcd", "etcd 证书", "etcd", "控制平面数据存储（etcd:2379，取最早到期）"),
		unknown("control_plane", "控制平面组件证书", "Control Plane", "controller-manager/scheduler HTTPS（10257/10259，取最早到期）"),
		unknown("kubelet", "kubelet 证书（节点）", "kubelet", "节点 kubelet HTTPS（10250，取最早到期）"),
	}
	apply := func(index int, observation ports.CertificateObservation, namePrefix string) {
		if !observation.Available {
			return
		}
		notBefore, notAfter := observation.NotBefore.UTC(), observation.NotAfter.UTC()
		days := int(notAfter.Sub(now).Hours() / 24)
		items[index].NotBefore, items[index].NotAfter, items[index].DaysLeft = &notBefore, &notAfter, &days
		items[index].Status = certificateStatus(days, notAfter, now)
		if strings.TrimSpace(observation.CommonName) != "" {
			items[index].Name = fmt.Sprintf("%s（%s）", namePrefix, observation.CommonName)
		}
	}
	apply(0, snapshot.APIServer, "API Server 证书")
	apply(1, snapshot.ClusterCA, "集群 CA 证书")
	apply(2, snapshot.Etcd, "etcd 证书")
	apply(3, snapshot.ControlPlane, "控制平面组件证书")
	if snapshot.ControlPlane.Available && strings.TrimSpace(snapshot.ControlPlaneName) != "" {
		items[3].Name = fmt.Sprintf("控制平面组件证书（%s）", snapshot.ControlPlaneName)
	}
	apply(4, snapshot.Kubelet, "kubelet 证书（节点）")
	sort.SliceStable(items, func(i, j int) bool {
		if certificateScore(items[i].Status) != certificateScore(items[j].Status) {
			return certificateScore(items[i].Status) > certificateScore(items[j].Status)
		}
		if items[i].DaysLeft == nil {
			return items[j].DaysLeft == nil && items[i].Key < items[j].Key
		}
		if items[j].DaysLeft == nil {
			return true
		}
		if *items[i].DaysLeft != *items[j].DaysLeft {
			return *items[i].DaysLeft < *items[j].DaysLeft
		}
		return items[i].Key < items[j].Key
	})
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry := map[string]any{"key": item.Key, "name": item.Name, "component": item.Component, "purpose": item.Purpose, "status": item.Status}
		if item.NotBefore != nil {
			entry["not_before"] = item.NotBefore.UTC().Format(time.RFC3339)
		}
		if item.NotAfter != nil {
			entry["not_after"] = item.NotAfter.UTC().Format(time.RFC3339)
		}
		if item.DaysLeft != nil {
			entry["days_left"] = *item.DaysLeft
		}
		result = append(result, entry)
	}
	return result
}

func certificateStatus(days int, notAfter, now time.Time) string {
	if notAfter.Before(now) || days <= 7 {
		return "critical"
	}
	if days <= 30 {
		return "warn"
	}
	return "ok"
}
func certificateScore(status string) int {
	switch status {
	case "critical":
		return 3
	case "warn":
		return 2
	case "ok":
		return 1
	default:
		return 0
	}
}
